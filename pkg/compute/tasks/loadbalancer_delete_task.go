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

type LoadbalancerDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerDeleteTask{})
}

func (self *LoadbalancerDeleteTask) taskFail(ctx context.Context, lb *models.SLoadbalancer, reason jsonutils.JSONObject) {
	lb.SetStatus(ctx, self.GetUserCred(), api.LB_STATUS_DELETE_FAILED, reason.String())
	db.OpsLog.LogEvent(lb, db.ACT_DELOCATE_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, lb, logclient.ACT_DELOCATE, reason, self.UserCred, false)
	notifyclient.EventNotify(ctx, self.GetUserCred(), notifyclient.SEventNotifyParam{
		Obj:    lb,
		Action: notifyclient.ActionDelete,
		IsFail: true,
	})
	self.SetStageFailed(ctx, reason)
}

func (self *LoadbalancerDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lb := obj.(*models.SLoadbalancer)
	region, err := lb.GetRegion()
	if err != nil {
		self.taskFail(ctx, lb, jsonutils.NewString(err.Error()))
		return
	}
	self.SetStage("OnLoadbalancerDeleteComplete", nil)
	if err := region.GetDriver().RequestDeleteLoadbalancer(ctx, self.GetUserCred(), lb, self); err != nil {
		self.taskFail(ctx, lb, jsonutils.NewString(err.Error()))
	}
}

func (self *LoadbalancerDeleteTask) OnLoadbalancerDeleteComplete(ctx context.Context, lb *models.SLoadbalancer, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(lb, db.ACT_DELETE, lb.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, lb, logclient.ACT_DELOCATE, nil, self.UserCred, true)
	// notifyclient.NotifyWebhook(ctx, self.UserCred, lb, notifyclient.ActionDelete)
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    lb,
		Action: notifyclient.ActionDelete,
	})
	deleteEip := jsonutils.QueryBoolean(self.Params, "delete_eip", false)
	lb.DeleteEip(ctx, self.UserCred, deleteEip)
	lb.RealDelete(ctx, self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerDeleteTask) OnLoadbalancerDeleteCompleteFailed(ctx context.Context, lb *models.SLoadbalancer, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lb, reason)
}
