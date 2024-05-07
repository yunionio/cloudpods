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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type GuestSetAutoRenewTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestSetAutoRenewTask{})
}

func (self *GuestSetAutoRenewTask) taskFailed(ctx context.Context, guest *models.SGuest, err error) {
	db.OpsLog.LogEvent(guest, db.ACT_SET_AUTO_RENEW_FAIL, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_SET_AUTO_RENEW, err, self.UserCred, false)
	guest.SetStatus(ctx, self.GetUserCred(), api.VM_SET_AUTO_RENEW_FAILED, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))

}

func (self *GuestSetAutoRenewTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)

	self.SetStage("OnSetAutoRenewComplete", nil)
	input := api.GuestAutoRenewInput{}
	self.GetParams().Unmarshal(&input)
	drv, err := guest.GetDriver()
	if err != nil {
		self.taskFailed(ctx, guest, errors.Wrapf(err, "GetDriver"))
		return
	}
	err = drv.RequestSetAutoRenewInstance(ctx, self.UserCred, guest, input, self)
	if err != nil {
		self.taskFailed(ctx, guest, errors.Wrapf(err, "RequestSetAutoRenewInstance"))
		return
	}
}

func (self *GuestSetAutoRenewTask) OnSetAutoRenewComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_SET_AUTO_RENEW, nil, self.UserCred, true)
	self.SetStage("OnGuestSyncstatusComplete", nil)
	guest.StartSyncstatus(ctx, self.UserCred, "")
}

func (self *GuestSetAutoRenewTask) OnSetAutoRenewCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_SET_AUTO_RENEW, data, self.UserCred, false)
	guest.SetStatus(ctx, self.GetUserCred(), api.VM_SET_AUTO_RENEW_FAILED, data.String())
	self.SetStageFailed(ctx, data)
}

func (self *GuestSetAutoRenewTask) OnGuestSyncstatusComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *GuestSetAutoRenewTask) OnGuestSyncstatusCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}
