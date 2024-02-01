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
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type LoadbalancerListenerRuleDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerListenerRuleDeleteTask{})
}

func (self *LoadbalancerListenerRuleDeleteTask) taskFail(ctx context.Context, lbr *models.SLoadbalancerListenerRule, err error) {
	lbr.SetStatus(ctx, self.GetUserCred(), api.LB_STATUS_DELETE_FAILED, err.Error())
	db.OpsLog.LogEvent(lbr, db.ACT_DELOCATE_FAIL, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, lbr, logclient.ACT_DELOCATE, err, self.UserCred, false)
	notifyclient.NotifySystemErrorWithCtx(ctx, lbr.Id, lbr.Name, api.LB_STATUS_DELETE_FAILED, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *LoadbalancerListenerRuleDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lbr := obj.(*models.SLoadbalancerListenerRule)
	region, err := lbr.GetRegion()
	if err != nil {
		self.taskFail(ctx, lbr, errors.Wrapf(err, "GetRegion"))
		return
	}
	self.SetStage("OnLoadbalancerListenerRuleDeleteComplete", nil)
	err = region.GetDriver().RequestDeleteLoadbalancerListenerRule(ctx, self.GetUserCred(), lbr, self)
	if err != nil {
		self.taskFail(ctx, lbr, errors.Wrapf(err, "RequestDeleteLoadbalancerListenerRule"))
	}
}

func (self *LoadbalancerListenerRuleDeleteTask) OnLoadbalancerListenerRuleDeleteComplete(ctx context.Context, lbr *models.SLoadbalancerListenerRule, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(lbr, db.ACT_DELETE, lbr.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, lbr, logclient.ACT_DELOCATE, nil, self.UserCred, true)
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    lbr,
		Action: notifyclient.ActionDelete,
	})
	lblis, _ := lbr.GetLoadbalancerListener()
	if lblis != nil {
		logclient.AddActionLogWithStartable(self, lblis, logclient.ACT_LB_REMOVE_LISTENER_RULE, nil, self.UserCred, true)
	}
	lbr.RealDelete(ctx, self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerListenerRuleDeleteTask) OnLoadbalancerListenerRuleDeleteCompleteFailed(ctx context.Context, lbr *models.SLoadbalancerListenerRule, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lbr, errors.Errorf(reason.String()))
}
