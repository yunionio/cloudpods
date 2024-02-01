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

type WafRuleDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(WafRuleDeleteTask{})
}

func (self *WafRuleDeleteTask) taskFailed(ctx context.Context, rule *models.SWafRule, err error) {
	rule.SetStatus(ctx, self.UserCred, api.WAF_RULE_STATUS_DELETE_FAILED, err.Error())
	logclient.AddActionLogWithStartable(self, rule, logclient.ACT_DELETE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *WafRuleDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	rule := obj.(*models.SWafRule)
	iRule, err := rule.GetICloudWafRule(ctx)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			self.taskComplete(ctx, rule)
			return
		}
		self.taskFailed(ctx, rule, errors.Wrapf(err, "GetICloudWafRule"))
		return
	}
	err = iRule.Delete()
	if err != nil {
		self.taskFailed(ctx, rule, errors.Wrapf(err, "iRule.Delete"))
		return
	}
	self.taskComplete(ctx, rule)
}

func (self *WafRuleDeleteTask) taskComplete(ctx context.Context, rule *models.SWafRule) {
	rule.RealDelete(ctx, self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}
