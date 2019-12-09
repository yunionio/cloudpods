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

type GuestResumeTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestResumeTask{})
}

func (self *GuestResumeTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	db.OpsLog.LogEvent(guest, db.ACT_RESUMING, "", self.UserCred)
	self.SetStage("OnResumeComplete", nil)
	err := guest.GetDriver().RqeuestResumeOnHost(ctx, guest, self)
	if err != nil {
		self.OnResumeGuestFail(guest, err.Error())
	}
}

func (self *GuestResumeTask) OnResumeComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(self.UserCred, api.VM_RUNNING, "")
	db.OpsLog.LogEvent(guest, db.ACT_RESUME, "", self.UserCred)
	self.SetStageComplete(ctx, nil)
}

func (self *GuestResumeTask) OnResumeCompleteFailed(ctx context.Context, obj db.IStandaloneModel,
	err jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(self.UserCred, api.VM_SUSPEND, "")
	db.OpsLog.LogEvent(guest, db.ACT_RESUME_FAIL, err.String(), self.UserCred)
	self.SetStageFailed(ctx, err.String())
}

func (self *GuestResumeTask) OnResumeGuestFail(guest *models.SGuest, reason string) {
	guest.SetStatus(self.UserCred, api.VM_RESUME_FAILED, reason)
}
