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

type LoadbalancerListenerDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerListenerDeleteTask{})
}

func (self *LoadbalancerListenerDeleteTask) taskFail(ctx context.Context, lblis *models.SLoadbalancerListener, reason jsonutils.JSONObject) {
	lblis.SetStatus(self.GetUserCred(), api.LB_STATUS_DELETE_FAILED, reason.String())
	db.OpsLog.LogEvent(lblis, db.ACT_DELOCATE_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, lblis, logclient.ACT_DELOCATE, reason, self.UserCred, false)
	notifyclient.EventNotify(ctx, self.GetUserCred(), notifyclient.SEventNotifyParam{
		Obj:    lblis,
		Action: notifyclient.ActionDelete,
		IsFail: true,
	})
	self.SetStageFailed(ctx, reason)
}

func (self *LoadbalancerListenerDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lblis := obj.(*models.SLoadbalancerListener)
	region := lblis.GetRegion()
	if region == nil {
		self.taskFail(ctx, lblis, jsonutils.NewString(fmt.Sprintf("failed to find region for lblis %s", lblis.Name)))
		return
	}
	self.SetStage("OnLoadbalancerListenerDeleteComplete", nil)
	if err := region.GetDriver().RequestDeleteLoadbalancerListener(ctx, self.GetUserCred(), lblis, self); err != nil {
		self.taskFail(ctx, lblis, jsonutils.NewString(err.Error()))
	}
}

func (self *LoadbalancerListenerDeleteTask) OnLoadbalancerListenerDeleteComplete(ctx context.Context, lblis *models.SLoadbalancerListener, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(lblis, db.ACT_DELETE, lblis.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, lblis, logclient.ACT_DELOCATE, nil, self.UserCred, true)
	notifyclient.NotifyWebhook(ctx, self.UserCred, lblis, notifyclient.ActionDelete)
	lblis.LBPendingDelete(ctx, self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerListenerDeleteTask) OnLoadbalancerListenerDeleteCompleteFailed(ctx context.Context, lblis *models.SLoadbalancerListener, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lblis, reason)
}
