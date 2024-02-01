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
	"yunion.io/x/pkg/util/billing"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type NatGatewaySetAutoRenewTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(NatGatewaySetAutoRenewTask{})
}

func (self *NatGatewaySetAutoRenewTask) taskFailed(ctx context.Context, nat *models.SNatGateway, err error) {
	db.OpsLog.LogEvent(nat, db.ACT_SET_AUTO_RENEW_FAIL, err, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, nat, logclient.ACT_SET_AUTO_RENEW, err, self.GetUserCred(), false)
	nat.SetStatus(ctx, self.GetUserCred(), api.NAT_STATUS_SET_AUTO_RENEW_FAILED, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *NatGatewaySetAutoRenewTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	nat := obj.(*models.SNatGateway)
	autoRenew := jsonutils.QueryBoolean(self.GetParams(), "auto_renew", false)
	iNat, err := nat.GetINatGateway(ctx)
	if err != nil {
		self.taskFailed(ctx, nat, errors.Wrapf(err, "GetINatGateway"))
		return
	}
	bc := billing.SBillingCycle{}
	bc.AutoRenew = autoRenew
	err = iNat.SetAutoRenew(bc)
	if err != nil {
		self.taskFailed(ctx, nat, errors.Wrapf(err, "iNat.SetAutoRenew"))
		return
	}
	self.SetStage("OnNatGatewaySyncComplete", nil)
	nat.StartSyncstatus(ctx, self.GetUserCred(), self.GetTaskId())
}

func (self *NatGatewaySetAutoRenewTask) OnNatGatewaySyncComplete(ctx context.Context, nat *models.SNatGateway, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *NatGatewaySetAutoRenewTask) OnNatGatewaySyncCompleteFailed(ctx context.Context, nat *models.SNatGateway, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}
