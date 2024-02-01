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

type WafRuleUpdateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(WafRuleUpdateTask{})
}

func (self *WafRuleUpdateTask) taskFailed(ctx context.Context, rule *models.SWafRule, err error) {
	rule.SetStatus(ctx, self.UserCred, api.WAF_RULE_STATUS_UPDATE_FAILED, err.Error())
	logclient.AddActionLogWithStartable(self, rule, logclient.ACT_UPDATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *WafRuleUpdateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	rule := obj.(*models.SWafRule)

	iRule, err := rule.GetICloudWafRule(ctx)
	if err != nil {
		self.taskFailed(ctx, rule, errors.Wrapf(err, "GetICloudWafRule"))
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

	err = iRule.Update(&opts)
	if err != nil {
		self.taskFailed(ctx, rule, errors.Wrapf(err, "iRule.Update"))
		return
	}

	self.taskComplete(ctx, rule)
}

func (self *WafRuleUpdateTask) taskComplete(ctx context.Context, rule *models.SWafRule) {
	rule.SetStatus(ctx, self.UserCred, api.WAF_RULE_STATUS_AVAILABLE, "")
	self.SetStageComplete(ctx, nil)
}
