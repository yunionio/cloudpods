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
	"database/sql"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type AccessGroupRuleDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(AccessGroupRuleDeleteTask{})
}

func (self *AccessGroupRuleDeleteTask) taskFailed(ctx context.Context, rule *models.SAccessGroupRule, err error) {
	rule.SetStatus(ctx, self.UserCred, apis.STATUS_DELETE_FAILED, err.Error())
	logclient.AddActionLogWithStartable(self, rule, logclient.ACT_DELOCATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *AccessGroupRuleDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	rule := obj.(*models.SAccessGroupRule)

	ag, err := rule.GetAccessGroup()
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			self.taskComplete(ctx, rule)
			return
		}
		self.taskFailed(ctx, rule, errors.Wrapf(err, "GetAccessGroup"))
		return
	}

	iGroup, err := ag.GetICloudAccessGroup(ctx)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			self.taskComplete(ctx, rule)
			return
		}
		self.taskFailed(ctx, rule, errors.Wrapf(err, "GetICloudAccessGroup"))
		return
	}

	rules, err := iGroup.GetRules()
	if err != nil {
		self.taskFailed(ctx, rule, errors.Wrapf(err, "GetRule"))
		return
	}
	for i := range rules {
		if rules[i].GetGlobalId() == rule.ExternalId {
			err = rules[i].Delete()
			if err != nil {
				self.taskFailed(ctx, rule, errors.Wrapf(err, "Delete"))
				return
			}
			break
		}
	}
	self.taskComplete(ctx, rule)
}

func (self *AccessGroupRuleDeleteTask) taskComplete(ctx context.Context, rule *models.SAccessGroupRule) {
	rule.RealDelete(ctx, self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}
