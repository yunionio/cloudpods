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
	"yunion.io/x/pkg/util/secrules"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SecurityGroupRuleUpdateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SecurityGroupRuleUpdateTask{})
}

func (self *SecurityGroupRuleUpdateTask) taskFailed(ctx context.Context, secgroup *models.SSecurityGroup, err error) {
	logclient.AddActionLogWithContext(ctx, secgroup, logclient.ACT_UPDATE, err, self.UserCred, false)
	rule, _ := self.getRule()
	if rule != nil {
		rule.SetStatus(ctx, self.UserCred, apis.STATUS_UNKNOWN, "")
		logclient.AddActionLogWithContext(ctx, rule, logclient.ACT_UPDATE, err, self.UserCred, false)
	}
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *SecurityGroupRuleUpdateTask) getRule() (*models.SSecurityGroupRule, error) {
	ruleId, err := self.GetParams().GetString("rule_id")
	if err != nil {
		return nil, errors.Wrapf(err, "get rule_id")
	}
	return models.SecurityGroupRuleManager.FetchRuleById(ruleId)
}

func (self *SecurityGroupRuleUpdateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	secgroup := obj.(*models.SSecurityGroup)
	if len(secgroup.ManagerId) == 0 {
		self.taskComplete(ctx, secgroup, nil)
		return
	}

	rule, err := self.getRule()
	if err != nil {
		self.taskFailed(ctx, secgroup, errors.Wrapf(err, "getRule"))
		return
	}

	if len(rule.ExternalId) == 0 {
		self.taskFailed(ctx, secgroup, errors.Wrapf(err, "empty external id"))
		return
	}

	iGroup, err := secgroup.GetISecurityGroup(ctx)
	if err != nil {
		self.taskFailed(ctx, secgroup, errors.Wrapf(err, "GetISecurityGroup"))
		return
	}

	rules, err := iGroup.GetRules()
	if err != nil {
		self.taskFailed(ctx, secgroup, errors.Wrapf(err, "GetRules"))
		return
	}

	for i := range rules {
		if rules[i].GetGlobalId() == rule.ExternalId {
			opts := &cloudprovider.SecurityGroupRuleUpdateOptions{
				CIDR:     rule.CIDR,
				Action:   secrules.TSecurityRuleAction(rule.Action),
				Desc:     rule.Description,
				Ports:    rule.Ports,
				Protocol: rule.Protocol,
				Priority: rule.Priority,
			}
			err = rules[i].Update(opts)
			if err != nil {
				self.taskFailed(ctx, secgroup, errors.Wrapf(err, "Update"))
				return
			}
			self.taskComplete(ctx, secgroup, iGroup)
			return
		}
	}

	self.taskComplete(ctx, secgroup, iGroup)
	return

}

func (self *SecurityGroupRuleUpdateTask) taskComplete(ctx context.Context, secgroup *models.SSecurityGroup, iGroup cloudprovider.ICloudSecurityGroup) {
	rule, _ := self.getRule()
	if rule != nil {
		rule.SetStatus(ctx, self.UserCred, apis.STATUS_AVAILABLE, "")
	}
	self.SetStageComplete(ctx, nil)
}
