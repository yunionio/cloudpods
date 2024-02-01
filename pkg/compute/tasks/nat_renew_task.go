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
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type NatGatewayRenewTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(NatGatewayRenewTask{})
}

func (self *NatGatewayRenewTask) taskFailed(ctx context.Context, nat *models.SNatGateway, err error) {
	db.OpsLog.LogEvent(nat, db.ACT_REW_FAIL, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, nat, logclient.ACT_RENEW, err, self.UserCred, false)
	nat.SetStatus(ctx, self.GetUserCred(), api.NAT_STATUS_RENEW_FAILED, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *NatGatewayRenewTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	nat := obj.(*models.SNatGateway)

	duration, _ := self.GetParams().GetString("duration")
	bc, err := billing.ParseBillingCycle(duration)
	if err != nil {
		self.taskFailed(ctx, nat, errors.Wrapf(err, "ParseBillingCycle(%s)", duration))
		return
	}

	iNat, err := nat.GetINatGateway(ctx)
	if err != nil {
		self.taskFailed(ctx, nat, errors.Wrapf(err, "GetINatGateway"))
		return
	}
	oldExpired := iNat.GetExpiredAt()

	err = iNat.Renew(bc)
	if err != nil {
		self.taskFailed(ctx, nat, errors.Wrapf(err, "iNat.Renew"))
		return
	}

	err = cloudprovider.WaitCreated(15*time.Second, 5*time.Minute, func() bool {
		err := iNat.Refresh()
		if err != nil {
			log.Errorf("failed refresh nat %s error: %v", nat.Name, err)
		}
		newExipred := iNat.GetExpiredAt()
		if newExipred.After(oldExpired) {
			return true
		}
		return false
	})
	if err != nil {
		self.taskFailed(ctx, nat, errors.Wrapf(err, "wait expired time refresh"))
		return
	}

	logclient.AddActionLogWithStartable(self, nat, logclient.ACT_RENEW, map[string]string{"duration": duration}, self.UserCred, true)

	self.SetStage("OnSyncstatusComplete", nil)
	nat.StartSyncstatus(ctx, self.GetUserCred(), self.GetTaskId())
}

func (self *NatGatewayRenewTask) OnSyncstatusComplete(ctx context.Context, nat *models.SNatGateway, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *NatGatewayRenewTask) OnSyncstatusCompleteFailed(ctx context.Context, nat *models.SNatGateway, reason jsonutils.JSONObject) {
	self.SetStageFailed(ctx, reason)
}
