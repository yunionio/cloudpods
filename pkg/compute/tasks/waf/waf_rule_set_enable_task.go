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

package waf

import (
	"context"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type WafRuleSetEnabledTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(WafRuleSetEnabledTask{})
}

func (self *WafRuleSetEnabledTask) taskFailed(ctx context.Context, record *models.SWafRule, err error) {
	record.SetStatus(ctx, self.UserCred, apis.STATUS_UNKNOWN, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *WafRuleSetEnabledTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	record := obj.(*models.SWafRule)

	iRule, err := record.GetICloudWafRule(ctx)
	if err != nil {
		self.taskFailed(ctx, record, errors.Wrapf(err, "GetICloudWafRule"))
		return
	}

	if record.Enabled.Bool() {
		err = iRule.Enable()
	} else {
		err = iRule.Disable()
	}

	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotSupported {
			self.taskComplete(ctx, record)
			return
		}
		self.taskFailed(ctx, record, errors.Wrapf(err, "SetEnabled"))
		return
	}

	self.taskComplete(ctx, record)
}

func (self *WafRuleSetEnabledTask) taskComplete(ctx context.Context, record *models.SWafRule) {
	record.SetStatus(ctx, self.UserCred, api.WAF_RULE_STATUS_AVAILABLE, "")
	self.SetStageComplete(ctx, nil)
}
