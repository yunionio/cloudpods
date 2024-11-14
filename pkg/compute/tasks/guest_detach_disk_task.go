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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type GuestDetachDiskTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestDetachDiskTask{})
}

func (self *GuestDetachDiskTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	diskId, _ := self.Params.GetString("disk_id")
	objDisk, err := models.DiskManager.FetchById(diskId)
	if err != nil {
		self.OnTaskFail(ctx, guest, nil, jsonutils.NewString(err.Error()))
		return
	}
	disk := objDisk.(*models.SDisk)
	guestdisks := disk.GetGuestdisks()
	if len(guestdisks) > 0 {
		guestdisk := guestdisks[0]
		self.Params.Add(jsonutils.NewString(guestdisk.Driver), "driver")
		self.Params.Add(jsonutils.NewString(guestdisk.CacheMode), "cache")
		self.Params.Add(jsonutils.NewString(guestdisk.Mountpoint), "mountpoint")
		if guestdisk.BootIndex >= 0 {
			self.Params.Add(jsonutils.NewInt(int64(guestdisk.BootIndex)), "boot_index")
		}
	}

	guest.DetachDisk(ctx, disk, self.UserCred)
	host, _ := guest.GetHost()
	if host != nil && !host.GetEnabled() && jsonutils.QueryBoolean(self.Params, "purge", false) {
		self.OnDetachDiskComplete(ctx, guest, nil)
		return
	}

	err = disk.RecordLastAttachedHost(ctx, self.UserCred, host.Id)
	if err != nil {
		self.OnTaskFail(ctx, guest, nil, jsonutils.NewString(err.Error()))
		return
	}

	if !host.GetEnabled() {
		self.OnDetachDiskCompleteFailed(ctx, guest, jsonutils.Marshal(map[string]string{"error": fmt.Sprintf("host %s(%s) is disabled", host.Name, host.Id)}))
		return
	}

	drv, err := guest.GetDriver()
	if err != nil {
		self.OnDetachDiskCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}

	self.SetStage("OnDetachDiskComplete", nil)
	err = drv.RequestDetachDisk(ctx, guest, disk, self)
	if err != nil {
		self.OnDetachDiskCompleteFailed(ctx, guest, jsonutils.Marshal(map[string]string{"error": err.Error()}))
	}
}

func (self *GuestDetachDiskTask) OnGetGuestStatus(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	statusStr, _ := data.GetString("status")
	if statusStr == "notfound" {
		self.OnDetachDiskComplete(ctx, guest, nil)
		return
	}

	self.SetStage("OnDetachDiskComplete", nil)
	if err := guest.StartSyncTaskWithoutSyncstatus(
		ctx, self.GetUserCred(), jsonutils.QueryBoolean(self.GetParams(), "sync_desc_only", false), self.GetTaskId(),
	); err != nil {
		self.OnDetachDiskCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
	}
}

func (self *GuestDetachDiskTask) OnGetGuestStatusFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.OnDetachDiskCompleteFailed(ctx, guest, data)
}

func (self *GuestDetachDiskTask) OnDetachDiskComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	diskId, _ := self.Params.GetString("disk_id")
	objDisk, err := models.DiskManager.FetchById(diskId)
	if err != nil {
		self.OnTaskFail(ctx, guest, nil, jsonutils.NewString(err.Error()))
		return
	}
	disk := objDisk.(*models.SDisk)
	disk.SetStatus(ctx, self.UserCred, api.DISK_READY, "on detach disk complete")
	keepDisk := jsonutils.QueryBoolean(self.Params, "keep_disk", true)
	host, _ := guest.GetHost()
	purge := false
	if host != nil && !host.GetEnabled() && jsonutils.QueryBoolean(self.Params, "purge", false) {
		purge = true
	}
	if !keepDisk && disk.AutoDelete {
		cnt, _ := disk.GetGuestDiskCount()
		if cnt == 0 {
			self.SetStage("OnDiskDeleteComplete", nil)
			db.OpsLog.LogEvent(disk, db.ACT_DELETE, "", self.UserCred)
			drv, err := guest.GetDriver()
			if err != nil {
				self.OnTaskFail(ctx, guest, disk, jsonutils.NewString(err.Error()))
				return
			}
			err = drv.RequestDeleteDetachedDisk(ctx, disk, self, purge)
			if err != nil {
				self.OnTaskFail(ctx, guest, disk, jsonutils.NewString(err.Error()))
			}
			return
		}
	}
	self.OnDiskDeleteComplete(ctx, guest, nil)
}

func (self *GuestDetachDiskTask) OnDetachDiskCompleteFailed(ctx context.Context, obj db.IStandaloneModel, reason jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	driver, _ := self.Params.GetString("driver")
	cache, _ := self.Params.GetString("cache")
	mountpoint, _ := self.Params.GetString("mountpoint")
	diskId, _ := self.Params.GetString("disk_id")
	var bootIndex *int8
	if self.Params.Contains("boot_index") {
		bd, _ := self.Params.Int("boot_index")
		bd8 := int8(bd)
		bootIndex = &bd8
	}
	objDisk, err := models.DiskManager.FetchById(diskId)
	if err != nil {
		log.Warningf("failed to fetch disk by id %s error: %v", diskId, err)
		self.OnTaskFail(ctx, guest, nil, reason)
		return
	}
	disk := objDisk.(*models.SDisk)
	db.OpsLog.LogEvent(disk, db.ACT_DETACH, reason.String(), self.UserCred)
	models.StartResourceSyncStatusTask(ctx, self.GetUserCred(), disk, "DiskSyncstatusTask", "")
	err = guest.AttachDisk(ctx, disk, self.UserCred, driver, cache, mountpoint, bootIndex)
	if err != nil {
		log.Warningf("recover attach disk %s(%s) for guest %s(%s) error: %v", disk.Name, disk.Id, guest.Name, guest.Id, err)
	}
	self.OnTaskFail(ctx, guest, nil, reason)
}

func (self *GuestDetachDiskTask) OnTaskFail(ctx context.Context, guest *models.SGuest, disk *models.SDisk, err jsonutils.JSONObject) {
	if disk != nil {
		disk.SetStatus(ctx, self.UserCred, api.DISK_READY, "")
	}
	guest.SetStatus(ctx, self.UserCred, api.VM_DETACH_DISK_FAILED, err.String())
	self.SetStageFailed(ctx, err)
	log.Errorf("Guest %s GuestDetachDiskTask failed %s", guest.Id, err.String())
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_DETACH_DISK, err, self.UserCred, false)
}

func (self *GuestDetachDiskTask) OnDiskDeleteComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.SetStage("OnSyncstatusComplete", nil)
	guest.StartSyncstatus(ctx, self.UserCred, self.GetTaskId())
}

func (self *GuestDetachDiskTask) OnSyncstatusComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
	logclient.AddActionLogWithStartable(self, obj, logclient.ACT_VM_DETACH_DISK, nil, self.UserCred, true)
}

func (self *GuestDetachDiskTask) OnSyncstatusCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.OnSyncstatusComplete(ctx, obj, data)
}
