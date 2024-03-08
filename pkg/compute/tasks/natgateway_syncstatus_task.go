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

type NatGatewaySyncstatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(NatGatewaySyncstatusTask{})
}

func (self *NatGatewaySyncstatusTask) taskFailed(ctx context.Context, natgateway *models.SNatGateway, err error) {
	natgateway.SetStatus(ctx, self.GetUserCred(), api.NAT_STATUS_UNKNOWN, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
	db.OpsLog.LogEvent(natgateway, db.ACT_SYNC_STATUS, natgateway.GetShortDesc(ctx), self.GetUserCred())
	logclient.AddActionLogWithContext(ctx, natgateway, logclient.ACT_SYNC_STATUS, err, self.UserCred, false)
}

func (self *NatGatewaySyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	natgateway := obj.(*models.SNatGateway)

	region, err := natgateway.GetRegion()
	if err != nil {
		self.taskFailed(ctx, natgateway, errors.Wrapf(err, "GetRegion"))
		return
	}

	self.SetStage("OnNatGatewaySyncStatusComplete", nil)
	err = region.GetDriver().RequestSyncNatGatewayStatus(ctx, self.GetUserCred(), natgateway, self)
	if err != nil {
		self.taskFailed(ctx, natgateway, errors.Wrapf(err, "RequestSyncNatGatewayStatus"))
		return
	}
}

func (self *NatGatewaySyncstatusTask) OnNatGatewaySyncStatusComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *NatGatewaySyncstatusTask) OnNatGatewaySyncStatusCompleteFailed(ctx context.Context, natgateway *models.SNatGateway, data jsonutils.JSONObject) {
	self.taskFailed(ctx, natgateway, errors.Errorf(data.String()))
}
