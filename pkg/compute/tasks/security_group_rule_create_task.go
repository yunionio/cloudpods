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

type SecurityGroupRuleCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SecurityGroupRuleCreateTask{})
}

func (self *SecurityGroupRuleCreateTask) taskFailed(ctx context.Context, secgroup *models.SSecurityGroup, err error) {
	logclient.AddActionLogWithContext(ctx, secgroup, logclient.ACT_CREATE_SECURITY_GROUP_RULE, err, self.UserCred, false)
	rule, _ := self.getRule()
	if rule != nil {
		rule.SetStatus(ctx, self.UserCred, apis.STATUS_CREATE_FAILED, "")
		logclient.AddActionLogWithContext(ctx, rule, logclient.ACT_ALLOCATE, err, self.UserCred, false)
	}
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *SecurityGroupRuleCreateTask) getRule() (*models.SSecurityGroupRule, error) {
	ruleId, err := self.GetParams().GetString("rule_id")
	if err != nil {
		return nil, errors.Wrapf(err, "get rule_id")
	}
	return models.SecurityGroupRuleManager.FetchRuleById(ruleId)
}

func (self *SecurityGroupRuleCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	secgroup := obj.(*models.SSecurityGroup)
	if len(secgroup.ManagerId) == 0 {
		self.SetStageComplete(ctx, nil)
		return
	}

	iGroup, err := secgroup.GetISecurityGroup(ctx)
	if err != nil {
		self.taskFailed(ctx, secgroup, errors.Wrapf(err, "GetISecurityGroup"))
		return
	}

	rule, err := self.getRule()
	if err != nil {
		self.taskFailed(ctx, secgroup, errors.Wrapf(err, "getRule"))
		return
	}

	opts := &cloudprovider.SecurityGroupRuleCreateOptions{
		Desc:      rule.Description,
		Priority:  rule.Priority,
		Protocol:  rule.Protocol,
		Ports:     rule.Ports,
		Direction: secrules.TSecurityRuleDirection(rule.Direction),
		CIDR:      rule.CIDR,
		Action:    secrules.TSecurityRuleAction(rule.Action),
	}
	if len(opts.CIDR) == 0 {
		opts.CIDR = "0.0.0.0/0"
	}

	iRule, err := iGroup.CreateRule(opts)
	if err != nil {
		self.taskFailed(ctx, secgroup, errors.Wrapf(err, "CreateRule"))
		return
	}

	_, err = db.Update(rule, func() error {
		rule.ExternalId = iRule.GetGlobalId()
		rule.Status = apis.STATUS_AVAILABLE
		return nil
	})

	if err != nil {
		self.taskFailed(ctx, secgroup, errors.Wrapf(err, "db.Update"))
		return
	}

	iGroup.Refresh()

	rules, err := iGroup.GetRules()
	if err != nil {
		self.taskFailed(ctx, secgroup, errors.Wrapf(err, "db.Update"))
		return
	}

	secgroup.SyncRules(ctx, self.UserCred, rules)

	self.SetStageComplete(ctx, nil)
}
