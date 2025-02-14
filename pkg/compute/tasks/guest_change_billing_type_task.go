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

type GuestChangeBillingTypeTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestChangeBillingTypeTask{})
}

func (self *GuestChangeBillingTypeTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.SetStage("OnGuestChangeBillingTypeTaskComplete", nil)
	drv := guest.GetDriver()
	err := drv.RequestChangeBillingType(ctx, guest, self)
	if err != nil {
		self.OnGuestChangeBillingTypeTaskCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
	}
}

func (self *GuestChangeBillingTypeTask) OnGuestChangeBillingTypeTaskComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_CHANGE_BILLING_TYPE, guest.BillingType, self.UserCred, false)
	self.SetStageComplete(ctx, nil)
}

func (self *GuestChangeBillingTypeTask) OnGuestChangeBillingTypeTaskCompleteFailed(ctx context.Context, guest *models.SGuest, reason jsonutils.JSONObject) {
	guest.SetStatus(ctx, self.GetUserCred(), api.VM_CHANGE_BILLING_TYPE_FAILED, "")
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_CHANGE_BILLING_TYPE, reason.String(), self.UserCred, false)
	self.SetStageFailed(ctx, reason)
}
