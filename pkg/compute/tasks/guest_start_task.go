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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type GuestStartTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestStartTask{})
	taskman.RegisterTask(GuestSchedStartTask{})
}

func (self *GuestStartTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.checkTemplate(ctx, guest)
}

func (self *GuestStartTask) checkTemplate(ctx context.Context, guest *models.SGuest) {
	/*diskCat := guest.CategorizeDisks()
	if diskCat.Root != nil && len(diskCat.Root.GetTemplateId()) > 0 {
		if len(guest.BackupHostId) > 0 {
			self.SetStage("OnMasterHostTemplateReady", nil)
		} else {
			self.SetStage("OnStartTemplateReady", nil)
		}
		guest.GetDriver().CheckDiskTemplateOnStorage(ctx, self.UserCred, diskCat.Root.GetTemplateId(), diskCat.Root.DiskFormat, diskCat.Root.StorageId, self)
	} else {
		self.startStart(ctx, guest)
	}*/
	self.startStart(ctx, guest)
}

func (self *GuestStartTask) OnMasterHostTemplateReady(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStage("OnStartTemplateReady", nil)
	diskCat := guest.CategorizeDisks()
	err := guest.GetDriver().CheckDiskTemplateOnStorage(ctx, self.UserCred, diskCat.Root.GetTemplateId(), diskCat.Root.DiskFormat,
		diskCat.Root.BackupStorageId, self)
	if err != nil {
		self.SetStageFailed(ctx, err.Error())
	}
}

func (self *GuestStartTask) OnStartTemplateReady(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.startStart(ctx, guest)
}

func (self *GuestStartTask) startStart(ctx context.Context, guest *models.SGuest) {
	db.OpsLog.LogEvent(guest, db.ACT_STARTING, nil, self.UserCred)
	if len(guest.BackupHostId) > 0 {
		self.RequestStartBacking(ctx, guest)
	} else {
		self.RequestStart(ctx, guest)
	}
}

func (self *GuestStartTask) RequestStart(ctx context.Context, guest *models.SGuest) {
	self.SetStage("OnStartComplete", nil)
	host := guest.GetHost()
	guest.SetStatus(self.UserCred, api.VM_STARTING, "")
	result, err := guest.GetDriver().RequestStartOnHost(ctx, guest, host, self.UserCred, self)
	if err != nil {
		self.OnStartCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
	} else {
		if result != nil && jsonutils.QueryBoolean(result, "is_running", false) {
			// guest.SetStatus(self.UserCred, models.VM_RUNNING, "start")
			// self.taskComplete(ctx, guest)
			self.OnStartComplete(ctx, guest, nil)
		}
	}
}

func (self *GuestStartTask) RequestStartBacking(ctx context.Context, guest *models.SGuest) {
	self.SetStage("OnStartBackupGuestComplete", nil)
	host := models.HostManager.FetchHostById(guest.BackupHostId)
	guest.SetStatus(self.UserCred, api.VM_BACKUP_STARTING, "")
	result, err := guest.GetDriver().RequestStartOnHost(ctx, guest, host, self.UserCred, self)
	if err != nil {
		self.OnStartCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
	} else {
		if result != nil && jsonutils.QueryBoolean(result, "is_running", false) {
			self.OnStartBackupGuestComplete(ctx, guest, nil)
		}
	}
}

func (self *GuestStartTask) OnStartBackupGuestComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	if data != nil {
		nbdServerPort, err := data.Int("nbd_server_port")
		if err == nil {
			backupHost := models.HostManager.FetchHostById(guest.BackupHostId)
			nbdServerUri := fmt.Sprintf("nbd:%s:%d", backupHost.AccessIp, nbdServerPort)
			guest.SetMetadata(ctx, "backup_nbd_server_uri", nbdServerUri, self.UserCred)
		} else {
			self.OnStartCompleteFailed(ctx, guest, jsonutils.NewString("Start backup guest result missing nbd_server_port"))
			return
		}
	}
	self.RequestStart(ctx, guest)
}

func (self *GuestStartTask) OnStartBackupGuestCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.OnStartCompleteFailed(ctx, guest, data)
}

func (self *GuestStartTask) OnStartComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	db.OpsLog.LogEvent(guest, db.ACT_START, guest.GetShortDesc(ctx), self.UserCred)
	self.SetStage("OnGuestSyncstatusAfterStart", nil)
	if guest.Hypervisor != api.HYPERVISOR_KVM {
		guest.StartSyncstatus(ctx, self.UserCred, self.GetTaskId())
	} else {
		logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_START, "", self.UserCred, true)
	}
	// self.taskComplete(ctx, guest)
}

func (self *GuestStartTask) OnGuestSyncstatusAfterStart(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.taskComplete(ctx, guest)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_START, "", self.UserCred, true)
}

func (self *GuestStartTask) OnStartCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(self.UserCred, api.VM_START_FAILED, err.String())
	db.OpsLog.LogEvent(guest, db.ACT_START_FAIL, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_START, err, self.UserCred, false)
	self.SetStageFailed(ctx, err.String())
}

func (self *GuestStartTask) taskComplete(ctx context.Context, guest *models.SGuest) {
	models.HostManager.ClearSchedDescCache(guest.HostId)
	self.SetStageComplete(ctx, nil)
}

type GuestSchedStartTask struct {
	SGuestBaseTask
}

func (self *GuestSchedStartTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.StartScheduler(ctx, guest)
}

func (self *GuestSchedStartTask) StartScheduler(ctx context.Context, guest *models.SGuest) {
	host := guest.GetHost()
	if guestsMem := host.GetRunningGuestMemorySize(); guestsMem < 0 {
		self.TaskFailed(ctx, guest, "Guest Start Failed: Can't Get Host Guests Memory")
	} else {
		if float32(guestsMem+guest.VmemSize) > host.GetVirtualMemorySize() {
			self.ScheduleFailed(ctx, guest)
		} else {
			self.ScheduleSucc(ctx, guest)
		}
	}
}

func (self *GuestSchedStartTask) ScheduleFailed(ctx context.Context, guest *models.SGuest) {
	self.SetStage("OnGuestMigrate", nil)
	guest.StartMigrateTask(ctx, self.UserCred, false, false, guest.Status, "", self.GetId())
}

func (self *GuestSchedStartTask) OnGuestMigrate(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
	guest.GuestNonSchedStartTask(ctx, self.UserCred, nil, "")
}

func (self *GuestSchedStartTask) OnGuestMigrateFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.TaskFailed(ctx, guest, fmt.Sprintf("OnGuestMigrateFailed %s", data))
}

func (self *GuestSchedStartTask) ScheduleSucc(ctx context.Context, guest *models.SGuest) {
	self.SetStageComplete(ctx, nil)
	guest.GuestNonSchedStartTask(ctx, self.UserCred, nil, "")
}

func (self *GuestSchedStartTask) TaskFailed(ctx context.Context, guest *models.SGuest, reason string) {
	self.SetStageFailed(ctx, reason)
	guest.SetStatus(self.UserCred, api.VM_START_FAILED, reason)
	db.OpsLog.LogEvent(guest, db.ACT_START_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(
		self, guest, logclient.ACT_VM_START, reason, self.UserCred, false)
}
