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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type GuestSuspendTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestSuspendTask{})
}

func (self *GuestSuspendTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	db.OpsLog.LogEvent(guest, db.ACT_STOPPING, "", self.UserCred)
	guest.SetStatus(self.UserCred, api.VM_SUSPENDING, "GuestSusPendTask")
	self.SetStage("on_suspend_complete", nil)
	err := guest.GetDriver().RqeuestSuspendOnHost(ctx, guest, self)
	if err != nil {
		self.OnSuspendGuestFail(guest, err.Error())
	}
}

func (self *GuestSuspendTask) OnSuspendComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(self.UserCred, api.VM_SUSPEND, "")
	db.OpsLog.LogEvent(guest, db.ACT_STOP, "", self.UserCred)
	self.SetStageComplete(ctx, nil)
}

func (self *GuestSuspendTask) OnSuspendCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(self.UserCred, api.VM_RUNNING, "")
	db.OpsLog.LogEvent(guest, db.ACT_STOP_FAIL, err.String(), self.UserCred)
	self.SetStageFailed(ctx, err.String())
}

func (self *GuestSuspendTask) OnSuspendGuestFail(guest *models.SGuest, reason string) {
	guest.SetStatus(self.UserCred, api.VM_SUSPEND_FAILED, reason)
}
