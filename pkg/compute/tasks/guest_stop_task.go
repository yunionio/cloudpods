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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type GuestStopTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestStopTask{})
}

func (self *GuestStopTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	db.OpsLog.LogEvent(guest, db.ACT_STOPPING, nil, self.UserCred)
	self.stopGuest(ctx, guest)
}

func (self *GuestStopTask) stopGuest(ctx context.Context, guest *models.SGuest) {
	host := guest.GetHost()
	if host == nil {
		self.OnGuestStopTaskCompleteFailed(ctx, guest, jsonutils.NewString("no associated host"))
		return
	}
	if !self.IsSubtask() {
		guest.SetStatus(self.UserCred, api.VM_STOPPING, "")
	}
	self.SetStage("OnMasterStopTaskComplete", nil)
	err := guest.GetDriver().RequestStopOnHost(ctx, guest, host, self)
	if err != nil {
		log.Errorf("RequestStopOnHost fail %s", err)
		self.OnGuestStopTaskCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
	}
}

func (self *GuestStopTask) OnMasterStopTaskComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	if len(guest.BackupHostId) > 0 {
		host := models.HostManager.FetchHostById(guest.BackupHostId)
		self.SetStage("OnGuestStopTaskComplete", nil)
		err := guest.GetDriver().RequestStopOnHost(ctx, guest, host, self)
		if err != nil {
			log.Errorf("RequestStopOnHost fail %s", err)
			self.OnGuestStopTaskCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
		}
	} else {
		self.OnGuestStopTaskComplete(ctx, guest, data)
	}
}

func (self *GuestStopTask) OnMasterStopTaskCompleteFailed(ctx context.Context, obj db.IStandaloneModel, reason jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.OnGuestStopTaskCompleteFailed(ctx, guest, reason)
}

func (self *GuestStopTask) OnGuestStopTaskComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	if !self.IsSubtask() {
		guest.StartSyncstatus(ctx, self.UserCred, "")
		// guest.SetStatus(self.UserCred, api.VM_READY, "")
	}
	db.OpsLog.LogEvent(guest, db.ACT_STOP, guest.GetShortDesc(ctx), self.UserCred)
	models.HostManager.ClearSchedDescCache(guest.HostId)
	self.SetStageComplete(ctx, nil)
	if guest.Status == api.VM_READY && guest.DisableDelete.IsFalse() && guest.ShutdownBehavior == api.SHUTDOWN_TERMINATE {
		guest.StartAutoDeleteGuestTask(ctx, self.UserCred, "")
	}
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_STOP, "", self.UserCred, true)
}

func (self *GuestStopTask) OnGuestStopTaskCompleteFailed(ctx context.Context, guest *models.SGuest, reason jsonutils.JSONObject) {
	if !self.IsSubtask() {
		guest.SetStatus(self.UserCred, api.VM_STOP_FAILED, reason.String())
	}
	db.OpsLog.LogEvent(guest, db.ACT_STOP_FAIL, reason.String(), self.UserCred)
	self.SetStageFailed(ctx, reason.String())
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_STOP, reason.String(), self.UserCred, false)
}
