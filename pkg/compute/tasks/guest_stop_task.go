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
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
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
	taskman.RegisterTask(GuestStopAndFreezeTask{})
}

func (self *GuestStopTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	db.OpsLog.LogEvent(guest, db.ACT_STOPPING, nil, self.UserCred)
	self.stopGuest(ctx, guest)
}

func (self *GuestStopTask) stopGuest(ctx context.Context, guest *models.SGuest) {
	host, err := guest.GetHost()
	if err != nil {
		self.OnGuestStopTaskCompleteFailed(ctx, guest, jsonutils.NewString(errors.Wrapf(err, "GetHost").Error()))
		return
	}
	if !self.IsSubtask() {
		guest.SetStatus(ctx, self.GetUserCred(), api.VM_STOPPING, "")
	}
	self.SetStage("OnGuestStopTaskComplete", nil)
	err = guest.GetDriver().RequestStopOnHost(ctx, guest, host, self, !self.IsSubtask())
	if err != nil {
		self.OnGuestStopTaskCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
	}
}

func (self *GuestStopTask) OnGuestStopTaskComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(guest, db.ACT_STOP, guest.GetShortDesc(ctx), self.UserCred)
	models.HostManager.ClearSchedDescCache(guest.HostId)
	if guest.Status != api.VM_READY && !self.IsSubtask() { // for kvm
		guest.SetStatus(ctx, self.GetUserCred(), api.VM_READY, "")
		if guest.CpuNumaPin != nil {
			guest.SetCpuNumaPin(ctx, self.UserCred, nil, nil)
		}
	}
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_STOP, "success", self.UserCred, true)
	self.SetStageComplete(ctx, nil)
	if guest.DisableDelete.IsFalse() && guest.ShutdownBehavior == api.SHUTDOWN_TERMINATE {
		guest.StartAutoDeleteGuestTask(ctx, self.UserCred, "")
		return
	}
}

func (self *GuestStopTask) OnGuestStopTaskCompleteFailed(ctx context.Context, guest *models.SGuest, reason jsonutils.JSONObject) {
	if !self.IsSubtask() {
		guest.SetStatus(ctx, self.UserCred, api.VM_STOP_FAILED, reason.String())
	}
	db.OpsLog.LogEvent(guest, db.ACT_STOP_FAIL, reason.String(), self.UserCred)
	self.SetStageFailed(ctx, reason)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_STOP, reason.String(), self.UserCred, false)
}

type GuestStopAndFreezeTask struct {
	SGuestBaseTask
}

func (self *GuestStopAndFreezeTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.SetStage("OnStopGuest", nil)
	err := guest.StartGuestStopTask(ctx, self.UserCred, 60, false, false, self.GetTaskId())
	if err != nil {
		self.OnStopGuestFailed(ctx, guest, jsonutils.NewString(err.Error()))
	}
}

func (self *GuestStopAndFreezeTask) OnStopGuestFailed(ctx context.Context, guest *models.SGuest, reason jsonutils.JSONObject) {
	db.OpsLog.LogEvent(guest, db.ACT_FREEZE_FAIL, reason.String(), self.UserCred)
	self.SetStageFailed(ctx, reason)
}

func (self *GuestStopAndFreezeTask) OnStopGuest(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStage("OnSyncStatus", nil)
	guest.StartSyncstatus(ctx, self.UserCred, self.GetTaskId())
}

func (self *GuestStopAndFreezeTask) OnSyncStatus(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	_, err := guest.SVirtualResourceBase.PerformFreeze(ctx, self.UserCred, nil, apis.PerformFreezeInput{})
	if err != nil {
		self.OnStopGuestFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	self.SetStageComplete(ctx, nil)
}

func (self *GuestStopAndFreezeTask) OnSyncStatusFailed(ctx context.Context, guest *models.SGuest, reason jsonutils.JSONObject) {
	db.OpsLog.LogEvent(guest, db.ACT_FREEZE_FAIL, reason.String(), self.UserCred)
	self.SetStageFailed(ctx, reason)
}
