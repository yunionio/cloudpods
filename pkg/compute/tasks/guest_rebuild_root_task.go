// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/osprofile"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

func init() {
	taskman.RegisterTask(GuestRebuildRootTask{})
	taskman.RegisterTask(KVMGuestRebuildRootTask{})
	taskman.RegisterTask(ManagedGuestRebuildRootTask{})
}

type GuestRebuildRootTask struct {
	SGuestBaseTask
}

func (self *GuestRebuildRootTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	if jsonutils.QueryBoolean(self.Params, "need_stop", false) {
		self.SetStage("OnStopServerComplete", nil)
		guest.StartGuestStopTask(ctx, self.UserCred, 60, false, false, self.GetTaskId())
	} else {
		self.StartRebuildRootDisk(ctx, guest)
	}
}

func (self *GuestRebuildRootTask) OnStopServerComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.StartRebuildRootDisk(ctx, guest)
}

func (self *GuestRebuildRootTask) markFailed(ctx context.Context, guest *models.SGuest, reason jsonutils.JSONObject) {
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_REBUILD, reason, self.UserCred, false)
	guest.SetStatus(ctx, self.GetUserCred(), api.VM_REBUILD_ROOT_FAIL, reason.String())
	self.SGuestBaseTask.SetStageFailed(ctx, reason)
	notifyclient.EventNotify(ctx, self.GetUserCred(), notifyclient.SEventNotifyParam{
		Obj:    guest,
		Action: notifyclient.ActionRebuildRoot,
		IsFail: true,
	})
	gds := guest.CategorizeDisks()
	imageId, _ := self.Params.GetString("origin_image_id")
	_, err := db.Update(gds.Root, func() error {
		gds.Root.TemplateId = imageId
		return nil
	})
	if err != nil {
		log.Errorf("recover root disk image id %s failed %v", imageId, err)
	}
}

func (self *GuestRebuildRootTask) StartRebuildRootDisk(ctx context.Context, guest *models.SGuest) {
	db.OpsLog.LogEvent(guest, db.ACT_REBUILDING_ROOT, nil, self.UserCred)
	gds := guest.CategorizeDisks()
	imageId, _ := self.Params.GetString("image_id")
	oldStatus := gds.Root.Status
	self.GetParams().Set("origin_image_id", jsonutils.NewString(gds.Root.TemplateId))
	_, err := db.Update(gds.Root, func() error {
		gds.Root.TemplateId = imageId
		gds.Root.Status = api.DISK_REBUILD
		return nil
	})
	if err != nil {
		self.markFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	} else {
		db.OpsLog.LogEvent(gds.Root, db.ACT_UPDATE_STATUS,
			fmt.Sprintf("%s=>%s", oldStatus, api.DISK_REBUILD), self.UserCred)
	}

	self.SetStage("OnRebuildRootDiskComplete", nil)
	guest.SetStatus(ctx, self.UserCred, api.VM_REBUILD_ROOT, "")

	// clear logininfo
	loginParams := make(map[string]interface{})
	loginParams["login_account"] = "none"
	loginParams["login_key"] = "none"
	loginParams["login_key_timestamp"] = "none"
	guest.SetAllMetadata(ctx, loginParams, self.UserCred)

	drv, err := guest.GetDriver()
	if err != nil {
		self.OnRebuildRootDiskCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	drv.RequestRebuildRootDisk(ctx, guest, self)
}

func (self *GuestRebuildRootTask) OnRebuildRootDiskComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	allDisks := jsonutils.QueryBoolean(self.Params, "all_disks", false)
	if allDisks {
		disks, _ := guest.GetDisks()
		for i := 1; i < len(disks); i += 1 {
			disks[i].SetStatus(ctx, self.UserCred, api.DISK_INIT, "rebuild data disks")
		}
		self.SetStage("OnRebuildingDataDisksComplete", nil)
		self.OnRebuildingDataDisksComplete(ctx, guest, data)
	} else {
		self.OnRebuildAllDisksComplete(ctx, guest, data)
	}
}

func (self *GuestRebuildRootTask) OnRebuildingDataDisksComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	diskReady := true
	disks, _ := guest.GetDisks()
	if len(disks) > 0 {
		guest.SetStatus(ctx, self.UserCred, api.VM_REBUILD_ROOT, "rebuild data disks")
	}
	for i := 1; i < len(disks); i += 1 {
		if disks[i].Status == api.DISK_INIT {
			diskReady = false
			disks[i].StartDiskCreateTask(ctx, self.UserCred, true, "", self.GetTaskId())
		}
	}
	if diskReady {
		self.OnRebuildAllDisksComplete(ctx, guest, data)
	}
}

func (self *GuestRebuildRootTask) OnRebuildingDataDisksCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	/*
		XXX ignore rebuild data disk errors
		db.OpsLog.LogEvent(guest, db.ACT_REBUILD_ROOT_FAIL, data, self.UserCred)
		guest.SetStatus(ctx,self.UserCred, models.VM_REBUILD_ROOT_FAIL, "OnRebuildingDataDisksCompleteFailed")
		logclient.AddActionLog(guest, logclient.ACT_VM_REBUILD, data, self.UserCred, false)
		self.SetStageFailed(ctx, data.String())
	*/
	self.OnRebuildAllDisksComplete(ctx, guest, data)
}

func (self *GuestRebuildRootTask) OnRebuildAllDisksComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	imgId, _ := self.Params.GetString("image_id")
	imginfo, err := models.CachedimageManager.GetImageById(ctx, self.UserCred, imgId, false)
	if err != nil {
		self.markFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	osprof, err := osprofile.GetOSProfileFromImageProperties(imginfo.Properties, guest.Hypervisor)
	if err != nil {
		self.markFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	var bios = "BIOS"
	isUefi, _ := imginfo.Properties["uefi_support"]
	if isUefi == "true" {
		bios = "UEFI"
	}
	log.Infof("guest rebuild root new bios %s", bios)

	err = guest.SetMetadata(ctx, "__os_profile__", osprof, self.UserCred)
	if err != nil {
		self.markFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	if guest.OsType != osprof.OSType || guest.Bios != bios {
		_, err := db.Update(guest, func() error {
			guest.OsType = osprof.OSType
			guest.Bios = bios
			return nil
		})
		if err != nil {
			self.markFailed(ctx, guest, jsonutils.NewString(err.Error()))
			return
		}
	}
	db.OpsLog.LogEvent(guest, db.ACT_REBUILD_ROOT, "", self.UserCred)
	guest.EventNotify(ctx, self.UserCred, notifyclient.ActionRebuildRoot)
	self.SetStage("OnSyncStatusComplete", nil)
	guest.StartSyncstatus(ctx, self.UserCred, self.GetTaskId())
}

func (self *GuestRebuildRootTask) OnRebuildRootDiskCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(guest, db.ACT_REBUILD_ROOT_FAIL, data, self.UserCred)
	guest.SetStatus(ctx, self.UserCred, api.VM_REBUILD_ROOT_FAIL, "OnRebuildRootDiskCompleteFailed")
	self.markFailed(ctx, guest, data)
}

func (self *GuestRebuildRootTask) OnSyncStatusComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	if guest.Status == api.VM_READY && jsonutils.QueryBoolean(self.Params, "auto_start", false) {
		self.SetStage("OnGuestStartComplete", nil)
		guest.StartGueststartTask(ctx, self.UserCred, nil, self.GetTaskId())
	} else {
		self.SetStageComplete(ctx, nil)
	}

	oldImageId, _ := self.GetParams().GetString("origin_image_id")
	imgId, _ := self.Params.GetString("image_id")
	notes := map[string]interface{}{}
	if oldImageId != imgId && len(oldImageId) > 0 {
		notes["old image id"] = oldImageId
	}
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_REBUILD, notes, self.UserCred, true)
}

func (self *GuestRebuildRootTask) OnGuestStartComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

/* -------------------------------------------------- */
/* ------------ KVMGuestRebuildRootTask ------------- */
/* -------------------------------------------------- */

type KVMGuestRebuildRootTask struct {
	SGuestBaseTask
}

func (self *KVMGuestRebuildRootTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	gds := guest.CategorizeDisks()
	self.SetStage("OnRebuildRootDiskComplete", nil)
	gds.Root.StartDiskCreateTask(ctx, self.UserCred, true, "", self.GetTaskId())
}

func (self *KVMGuestRebuildRootTask) OnRebuildRootDiskComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)

	self.SetStage("OnGuestDeployComplete", nil)
	guest.SetStatus(ctx, self.UserCred, api.VM_DEPLOYING, "")
	deployParams, _ := self.Params.Get("deploy_params")
	var params *jsonutils.JSONDict
	if deployParams != nil {
		params = deployParams.(*jsonutils.JSONDict)
	} else {
		params = jsonutils.NewDict()
	}
	// params := jsonutils.NewDict()
	// params.Set("reset_password", jsonutils.JSONTrue)
	guest.StartGuestDeployTask(ctx, self.UserCred, params, "rebuild", self.GetTaskId())
}

func (self *KVMGuestRebuildRootTask) OnRebuildRootDiskCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}

func (self *KVMGuestRebuildRootTask) OnGuestDeployComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	// guest := obj.(*models.SGuest)
	self.SetStageComplete(ctx, nil)
	// logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_REBUILD, nil, self.UserCred, true)
}

func (self *KVMGuestRebuildRootTask) OnGuestDeployCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}

type ManagedGuestRebuildRootTask struct {
	SGuestBaseTask
}

func (self *ManagedGuestRebuildRootTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)

	self.SetStage("OnHostCacheImageComplete", nil)
	drv, err := guest.GetDriver()
	if err != nil {
		self.OnHostCacheImageCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	drv.RequestGuestCreateAllDisks(ctx, guest, self)
}

func (self *ManagedGuestRebuildRootTask) OnHostCacheImageComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)

	self.SetStage("OnGuestDeployComplete", nil)
	guest.SetStatus(ctx, self.UserCred, api.VM_DEPLOYING, "rebuild deploy")
	deployParams, _ := self.Params.Get("deploy_params")
	var params *jsonutils.JSONDict
	if deployParams != nil {
		params = deployParams.(*jsonutils.JSONDict)
	} else {
		params = jsonutils.NewDict()
	}
	guest.StartGuestDeployTask(ctx, self.UserCred, params, "rebuild", self.GetTaskId())
}

func (self *ManagedGuestRebuildRootTask) OnHostCacheImageCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	// guest := obj.(*models.SGuest)

	self.SetStageFailed(ctx, data)
	// logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_REBUILD, data, self.UserCred, false)
}

func (self *ManagedGuestRebuildRootTask) OnGuestDeployComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *ManagedGuestRebuildRootTask) OnGuestDeployCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}
