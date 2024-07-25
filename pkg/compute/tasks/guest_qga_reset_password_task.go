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
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SGuestQgaBaseTask struct {
	SGuestBaseTask
}

func (self *SGuestQgaBaseTask) guestPing(ctx context.Context, guest *models.SGuest) error {
	host, err := guest.GetHost()
	if err != nil {
		return err
	}
	drv, err := guest.GetDriver()
	if err != nil {
		return err
	}
	return drv.QgaRequestGuestPing(ctx, self.GetTaskRequestHeader(), host, guest, true, nil)
}

func (self *SGuestQgaBaseTask) taskFailed(ctx context.Context, guest *models.SGuest, reason string) {
	guest.SetStatus(ctx, self.UserCred, api.VM_QGA_EXEC_COMMAND_FAILED, reason)
	db.OpsLog.LogEvent(guest, db.ACT_SET_USER_PASSWORD_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_SET_USER_PASSWORD, reason, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(reason))
}

type GuestQgaSetPasswordTask struct {
	SGuestQgaBaseTask
}

func init() {
	taskman.RegisterTask(GuestQgaSetPasswordTask{})
}

func (self *GuestQgaSetPasswordTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.SetStage("OnQgaGuestPing", nil)
	if err := self.guestPing(ctx, guest); err != nil {
		self.taskFailed(ctx, guest, err.Error())
	}
}

func (self *GuestQgaSetPasswordTask) OnQgaGuestPing(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	input := &api.ServerQgaSetPasswordInput{}
	self.GetParams().Unmarshal(input)
	self.SetStage("OnQgaSetUserPassword", nil)
	host, err := guest.GetHost()
	if err != nil {
		self.taskFailed(ctx, guest, err.Error())
		return
	}
	drv, err := guest.GetDriver()
	if err != nil {
		self.taskFailed(ctx, guest, err.Error())
		return
	}
	err = drv.QgaRequestSetUserPassword(ctx, self, host, guest, input)
	if err != nil {
		self.taskFailed(ctx, guest, err.Error())
	}
}

func (self *GuestQgaSetPasswordTask) OnQgaGuestPingFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.taskFailed(ctx, guest, data.String())
}

func (self *GuestQgaSetPasswordTask) OnQgaSetUserPassword(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	guest.SetStatus(ctx, self.UserCred, api.VM_RUNNING, "on qga set user password success")
	db.OpsLog.LogEvent(guest, db.ACT_SET_USER_PASSWORD, "", self.UserCred)

	input := &api.ServerQgaSetPasswordInput{}
	self.GetParams().Unmarshal(input)
	info := make(map[string]interface{})
	loginAccount := guest.GetMetadata(ctx, "login_account", self.UserCred)
	if loginAccount == input.Username || loginAccount == "" {
		secret, _ := utils.EncryptAESBase64(guest.Id, input.Password)
		info["login_key"] = secret
		info["login_key_timestamp"] = timeutils.UtcNow()
	}
	if loginAccount == "" {
		info["login_account"] = input.Username
	}
	guest.SetAllMetadata(ctx, info, self.UserCred)

	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_SET_USER_PASSWORD, "qga set password success", self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *GuestQgaSetPasswordTask) OnQgaSetUserPasswordFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.taskFailed(ctx, guest, data.String())
}
