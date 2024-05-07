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
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type GuestPublicipToEipTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(GuestPublicipToEipTask{})
}

func (self *GuestPublicipToEipTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)

	self.SetStage("OnEipConvertComplete", nil)
	drv, err := guest.GetDriver()
	if err != nil {
		db.OpsLog.LogEvent(guest, db.ACT_EIP_CONVERT_FAIL, err, self.UserCred)
		logclient.AddActionLogWithStartable(self, guest, logclient.ACT_EIP_CONVERT, err, self.UserCred, false)
		guest.SetStatus(ctx, self.GetUserCred(), api.VM_EIP_CONVERT_FAILED, err.Error())
		self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
		return
	}
	err = drv.RequestConvertPublicipToEip(ctx, self.GetUserCred(), guest, self)
	if err != nil {
		db.OpsLog.LogEvent(guest, db.ACT_EIP_CONVERT_FAIL, err, self.UserCred)
		logclient.AddActionLogWithStartable(self, guest, logclient.ACT_EIP_CONVERT, err, self.UserCred, false)
		guest.SetStatus(ctx, self.GetUserCred(), api.VM_EIP_CONVERT_FAILED, err.Error())
		self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
		return
	}
}

func (self *GuestPublicipToEipTask) OnEipConvertComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_EIP_CONVERT, nil, self.UserCred, true)
	self.SetStage("OnGuestSyncstatusComplete", nil)
	guest.StartSyncstatus(ctx, self.UserCred, self.GetTaskId())
}

func (self *GuestPublicipToEipTask) OnEipConvertCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_EIP_CONVERT, data, self.UserCred, false)
	guest.SetStatus(ctx, self.UserCred, api.VM_EIP_CONVERT_FAILED, data.String())
	self.SetStageFailed(ctx, data)
}

func (self *GuestPublicipToEipTask) OnGuestSyncstatusComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	if guest.Status == api.VM_READY && jsonutils.QueryBoolean(self.Params, "auto_start", false) {
		self.SetStage("OnGuestStartSucc", nil)
		guest.StartGueststartTask(ctx, self.UserCred, nil, self.GetId())
		return
	}
	self.SetStageComplete(ctx, nil)
}

func (self *GuestPublicipToEipTask) OnGuestSyncstatusCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}

func (self *GuestPublicipToEipTask) OnGuestStartSucc(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *GuestPublicipToEipTask) OnGuestStartSuccFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}
