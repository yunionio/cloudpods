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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type WafRuleCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(WafRuleCreateTask{})
}

func (self *WafRuleCreateTask) taskFailed(ctx context.Context, rule *models.SWafRule, err error) {
	rule.SetStatus(self.UserCred, api.WAF_RULE_STATUS_CREATE_FAILED, err.Error())
	logclient.AddActionLogWithStartable(self, rule, logclient.ACT_ALLOCATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *WafRuleCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	rule := obj.(*models.SWafRule)
	iWaf, err := rule.GetICloudWafInstance(ctx)
	if err != nil {
		self.taskFailed(ctx, rule, errors.Wrapf(err, "GetICloudWafInstance"))
		return
	}
	opts := cloudprovider.SWafRule{
		Name:       rule.Name,
		Desc:       rule.Description,
		Action:     rule.Action,
		Priority:   rule.Priority,
		Statements: []cloudprovider.SWafStatement{},
	}
	opts.StatementCondition = rule.StatementConditon
	statements, err := rule.GetRuleStatements()
	if err != nil {
		self.taskFailed(ctx, rule, errors.Wrapf(err, "GetRuleStatements"))
		return
	}
	for i := range statements {
		opts.Statements = append(opts.Statements, statements[i].SWafStatement)
	}
	iRule, err := iWaf.AddRule(&opts)
	if err != nil {
		self.taskFailed(ctx, rule, errors.Wrapf(err, "iWaf.AddRule"))
		return
	}
	rule.SyncWithCloudRule(ctx, self.GetUserCred(), iRule)
	self.taskComplete(ctx, rule)
}

func (self *WafRuleCreateTask) taskComplete(ctx context.Context, rule *models.SWafRule) {
	self.SetStageComplete(ctx, nil)
}
