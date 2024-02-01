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

type LoadbalancerListenerRuleCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerListenerRuleCreateTask{})
}

func (self *LoadbalancerListenerRuleCreateTask) taskFail(ctx context.Context, lbr *models.SLoadbalancerListenerRule, err error) {
	lbr.SetStatus(ctx, self.GetUserCred(), api.LB_CREATE_FAILED, err.Error())
	db.OpsLog.LogEvent(lbr, db.ACT_ALLOCATE_FAIL, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, lbr, logclient.ACT_CREATE, err, self.UserCred, false)
	notifyclient.NotifySystemErrorWithCtx(ctx, lbr.Id, lbr.Name, api.LB_CREATE_FAILED, err.Error())
	lblis, _ := lbr.GetLoadbalancerListener()
	if lblis != nil {
		logclient.AddActionLogWithStartable(self, lblis, logclient.ACT_LB_ADD_LISTENER_RULE, err, self.UserCred, false)
	}
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *LoadbalancerListenerRuleCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lbr := obj.(*models.SLoadbalancerListenerRule)
	region, err := lbr.GetRegion()
	if err != nil {
		self.taskFail(ctx, lbr, errors.Wrapf(err, "GetRegion"))
		return
	}

	self.SetStage("OnLoadbalancerListenerRuleCreateComplete", nil)
	err = region.GetDriver().RequestCreateLoadbalancerListenerRule(ctx, self.GetUserCred(), lbr, self)
	if err != nil {
		self.taskFail(ctx, lbr, errors.Wrapf(err, "RequestCreateLoadbalancerListenerRule"))
	}
}

func (self *LoadbalancerListenerRuleCreateTask) OnCreateLoadbalancerListenerRuleFailed(ctx context.Context, lbr *models.SLoadbalancerListenerRule, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lbr, errors.Errorf(reason.String()))
}

func (self *LoadbalancerListenerRuleCreateTask) OnLoadbalancerListenerRuleCreateComplete(ctx context.Context, lbr *models.SLoadbalancerListenerRule, data jsonutils.JSONObject) {
	lbr.SetStatus(ctx, self.GetUserCred(), api.LB_STATUS_ENABLED, "")
	db.OpsLog.LogEvent(lbr, db.ACT_ALLOCATE, lbr.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, lbr, logclient.ACT_CREATE, nil, self.UserCred, true)
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    lbr,
		Action: notifyclient.ActionCreate,
	})
	lblis, _ := lbr.GetLoadbalancerListener()
	if lblis != nil {
		logclient.AddActionLogWithStartable(self, lblis, logclient.ACT_LB_ADD_LISTENER_RULE, nil, self.UserCred, true)
	}
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerListenerRuleCreateTask) OnLoadbalancerListenerRuleCreateCompleteFailed(ctx context.Context, lbr *models.SLoadbalancerListenerRule, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lbr, errors.Errorf(reason.String()))
}
