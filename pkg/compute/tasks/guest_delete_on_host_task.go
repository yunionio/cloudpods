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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

func init() {
	taskman.RegisterTask(GuestDeleteOnHostTask{})
}

type GuestDeleteOnHostTask struct {
	SGuestBaseTask
}

func (self *GuestDeleteOnHostTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	hostId, err := self.Params.GetString("host_id")
	if err != nil {
		self.OnFail(ctx, guest, jsonutils.NewString("Missing param host_id"))
		return
	}
	host := models.HostManager.FetchHostById(hostId)
	if host == nil {
		self.OnFail(ctx, guest, jsonutils.NewString("Host is nil"))
		return
	}

	self.SetStage("OnStopGuest", nil)
	self.Params.Set("is_force", jsonutils.JSONTrue)
	drv, err := guest.GetDriver()
	if err != nil {
		self.OnFail(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	if err := drv.RequestStopOnHost(ctx, guest, host, self, true); err != nil {
		log.Errorf("RequestStopGuestForDelete fail %s", err)
		self.OnStopGuest(ctx, guest, nil)
	}
}

func (self *GuestDeleteOnHostTask) OnStopGuest(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	hostId, _ := self.Params.GetString("host_id")
	isPurge := jsonutils.QueryBoolean(self.Params, "purge", false)

	if !isPurge {
		self.SetStage("OnUnDeployGuest", nil)
		guest.StartUndeployGuestTask(ctx, self.GetUserCred(), self.GetTaskId(), hostId)
	} else {
		self.OnUnDeployGuest(ctx, guest, nil)
	}
}

func (self *GuestDeleteOnHostTask) OnUnDeployGuestFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.OnFail(ctx, guest, data)
}

func (self *GuestDeleteOnHostTask) OnUnDeployGuest(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	hostId, _ := self.Params.GetString("host_id")
	if guest.BackupHostId == hostId {
		_, err := db.Update(guest, func() error {
			guest.BackupHostId = ""
			guest.BackupGuestStatus = compute.VM_INIT
			return nil
		})
		if err != nil {
			self.OnFail(ctx, guest, jsonutils.NewString(err.Error()))
			return
		}
		disks, err := guest.GetDisks()
		if err != nil {
			self.OnFail(ctx, guest, jsonutils.NewString(err.Error()))
			return
		}
		for i := 0; i < len(disks); i++ {
			disk := &disks[i]
			_, err := db.Update(disk, func() error {
				disk.BackupStorageId = ""
				return nil
			})
			if err != nil {
				self.OnFail(ctx, guest, jsonutils.NewString(err.Error()))
				return
			}
		}
		logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_DELETE_BACKUP, "GuestDeleteOnHost", self.UserCred, true)
		db.OpsLog.LogEvent(guest, db.ACT_DELETE_BACKUP, guest.GetShortDesc(ctx), self.UserCred)
	}
	self.SetStage("OnSync", nil)
	guest.StartSyncTask(ctx, self.UserCred, false, self.GetTaskId())
}

func (self *GuestDeleteOnHostTask) OnSync(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *GuestDeleteOnHostTask) OnFail(ctx context.Context, guest *models.SGuest, reason jsonutils.JSONObject) {
	hostId, _ := self.Params.GetString("host_id")
	if guest.BackupHostId == hostId {
		logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_DELETE_BACKUP, "GuestDeleteOnHost", self.UserCred, false)
		db.OpsLog.LogEvent(guest, db.ACT_DELETE_BACKUP_FAILED, "GuestDeleteOnHost", self.UserCred)
	}
	failedStatus, _ := self.Params.GetString("failed_status")
	if len(failedStatus) > 0 {
		guest.SetStatus(ctx, self.UserCred, failedStatus, reason.String())
	}
	self.SetStageFailed(ctx, reason)
}
