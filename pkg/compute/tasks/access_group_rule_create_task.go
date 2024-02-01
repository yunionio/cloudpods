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

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type AccessGroupRuleCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(AccessGroupRuleCreateTask{})
}

func (self *AccessGroupRuleCreateTask) taskFailed(ctx context.Context, rule *models.SAccessGroupRule, err error) {
	rule.SetStatus(ctx, self.UserCred, apis.STATUS_CREATE_FAILED, err.Error())
	logclient.AddActionLogWithStartable(self, rule, logclient.ACT_ALLOCATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *AccessGroupRuleCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	rule := obj.(*models.SAccessGroupRule)

	ag, err := rule.GetAccessGroup()
	if err != nil {
		self.taskFailed(ctx, rule, errors.Wrapf(err, "GetAccessGroup"))
		return
	}

	iGroup, err := ag.GetICloudAccessGroup(ctx)
	if err != nil {
		self.taskFailed(ctx, rule, errors.Wrapf(err, "GetICloudAccessGroup"))
		return
	}

	opts := &cloudprovider.AccessGroupRule{
		Priority:       rule.Priority,
		RWAccessType:   cloudprovider.TRWAccessType(rule.RWAccessType),
		UserAccessType: cloudprovider.TUserAccessType(rule.UserAccessType),
		Source:         rule.Source,
	}

	iRule, err := iGroup.CreateRule(opts)
	if err != nil {
		self.taskFailed(ctx, rule, errors.Wrapf(err, "CreateRule"))
		return
	}

	err = rule.SyncWithAccessGroupRule(ctx, self.UserCred, iRule)
	if err != nil {
		self.taskFailed(ctx, rule, errors.Wrapf(err, "SyncWithAccessGroupRule"))
		return
	}

	self.SetStageComplete(ctx, nil)
}
