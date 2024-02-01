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
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type LoadbalancerStopTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerStopTask{})
}

func (self *LoadbalancerStopTask) taskFail(ctx context.Context, lb *models.SLoadbalancer, reason jsonutils.JSONObject) {
	lb.SetStatus(ctx, self.GetUserCred(), api.LB_STATUS_ENABLED, reason.String())
	db.OpsLog.LogEvent(lb, db.ACT_DISABLE, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, lb, logclient.ACT_DISABLE, reason, self.UserCred, false)
	notifyclient.NotifySystemErrorWithCtx(ctx, lb.Id, lb.Name, api.LB_STATUS_ENABLED, reason.String())
	self.SetStageFailed(ctx, reason)
}

func (self *LoadbalancerStopTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lb := obj.(*models.SLoadbalancer)
	region, err := lb.GetRegion()
	if err != nil {
		self.taskFail(ctx, lb, jsonutils.NewString(err.Error()))
		return
	}
	self.SetStage("OnLoadbalancerStopComplete", nil)
	if err := region.GetDriver().RequestStopLoadbalancer(ctx, self.GetUserCred(), lb, self); err != nil {
		self.taskFail(ctx, lb, jsonutils.NewString(err.Error()))
	}
}

func (self *LoadbalancerStopTask) OnLoadbalancerStopComplete(ctx context.Context, lb *models.SLoadbalancer, data jsonutils.JSONObject) {
	lb.SetStatus(ctx, self.GetUserCred(), api.LB_STATUS_DISABLED, "")
	db.OpsLog.LogEvent(lb, db.ACT_DISABLE, lb.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, lb, logclient.ACT_DISABLE, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerStopTask) OnLoadbalancerStopCompleteFailed(ctx context.Context, lb *models.SLoadbalancer, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lb, reason)
}
