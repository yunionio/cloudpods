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
	"fmt"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type LoadbalancerCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerCreateTask{})
}

func (self *LoadbalancerCreateTask) taskFail(ctx context.Context, lb *models.SLoadbalancer, reason jsonutils.JSONObject) {
	lb.SetStatus(self.GetUserCred(), api.LB_CREATE_FAILED, reason.String())
	db.OpsLog.LogEvent(lb, db.ACT_ALLOCATE_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, lb, logclient.ACT_CREATE, reason, self.UserCred, false)
	notifyclient.NotifySystemErrorWithCtx(ctx, lb.Id, lb.Name, api.LB_CREATE_FAILED, reason.String())
	self.SetStageFailed(ctx, reason)
}

func (self *LoadbalancerCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lb := obj.(*models.SLoadbalancer)
	region := lb.GetRegion()
	if region == nil {
		self.taskFail(ctx, lb, jsonutils.NewString(fmt.Sprintf("failed to find region for lb %s", lb.Name)))
		return
	}
	self.SetStage("OnLoadbalancerCreateComplete", nil)
	if err := region.GetDriver().RequestCreateLoadbalancer(ctx, self.GetUserCred(), lb, self); err != nil {
		self.taskFail(ctx, lb, jsonutils.NewString(err.Error()))
	}
}

func (self *LoadbalancerCreateTask) OnLoadbalancerCreateComplete(ctx context.Context, lb *models.SLoadbalancer, data jsonutils.JSONObject) {
	lb.SetStatus(self.GetUserCred(), api.LB_STATUS_ENABLED, "")
	db.OpsLog.LogEvent(lb, db.ACT_ALLOCATE, lb.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, lb, logclient.ACT_CREATE, nil, self.UserCred, true)
	self.SetStage("OnLoadbalancerStartComplete", nil)
	lb.StartLoadBalancerStartTask(ctx, self.GetUserCred(), self.GetTaskId())
}

func (self *LoadbalancerCreateTask) OnLoadbalancerCreateCompleteFailed(ctx context.Context, lb *models.SLoadbalancer, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lb, reason)
}

func (self *LoadbalancerCreateTask) OnLoadbalancerStartComplete(ctx context.Context, lb *models.SLoadbalancer, data jsonutils.JSONObject) {
	lb.SetStatus(self.GetUserCred(), api.LB_STATUS_ENABLED, "")
	notifyclient.NotifyWebhook(ctx, self.UserCred, lb, notifyclient.ActionCreate)
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerCreateTask) OnLoadbalancerStartCompleteFailed(ctx context.Context, lb *models.SLoadbalancer, reason jsonutils.JSONObject) {
	lb.SetStatus(self.GetUserCred(), api.LB_STATUS_DISABLED, reason.String())
	self.SetStageFailed(ctx, reason)
}
