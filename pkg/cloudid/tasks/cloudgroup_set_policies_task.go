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

	"yunion.io/x/cloudmux/pkg/apis"
	"yunion.io/x/cloudmux/pkg/apis/cloudid"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/cloudid"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudid/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type CloudgroupSetPoliciesTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(CloudgroupSetPoliciesTask{})
}

func (self *CloudgroupSetPoliciesTask) taskFailed(ctx context.Context, group *models.SCloudgroup, err error) {
	group.SetStatus(ctx, self.GetUserCred(), apis.STATUS_UNKNOWN, err.Error())
	logclient.AddActionLogWithStartable(self, group, logclient.ACT_SYNC_CONF, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *CloudgroupSetPoliciesTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	group := obj.(*models.SCloudgroup)

	roles, err := group.GetCloudroles()
	if err != nil {
		self.taskFailed(ctx, group, errors.Wrapf(err, "GetCloudroles"))
		return
	}

	iGroup, err := group.GetICloudgroup()
	if err != nil {
		self.taskFailed(ctx, group, errors.Wrap(err, "GetICloudgroup"))
		return
	}

	input := struct {
		Add []api.SPolicy
		Del []api.SPolicy
	}{}
	err = self.GetParams().Unmarshal(&input)
	if err != nil {
		self.taskFailed(ctx, group, errors.Wrapf(err, "Unmarshal"))
		return
	}

	iRoleMap := map[string]cloudprovider.ICloudrole{}
	roleMap := map[string]*models.SCloudrole{}
	for i := range roles {
		roleMap[roles[i].Id] = &roles[i]
		iRole, err := roles[i].GetICloudrole()
		if err == nil {
			iRoleMap[roles[i].Id] = iRole
		}
	}

	for _, policy := range input.Add {
		for id, role := range iRoleMap {
			err := role.AttachPolicy(policy.ExternalId, cloudid.TPolicyType(policy.PolicyType))
			if err != nil {
				logclient.AddSimpleActionLog(roleMap[id], logclient.ACT_ATTACH_POLICY, err, self.GetUserCred(), false)
			}
		}
		err = iGroup.AttachPolicy(policy.ExternalId, cloudid.TPolicyType(policy.PolicyType))
		if err != nil {
			self.taskFailed(ctx, group, errors.Wrapf(err, "AttachPolicy %s", policy.Name))
			return
		}
	}

	for _, policy := range input.Del {
		for id, role := range iRoleMap {
			err := role.DetachPolicy(policy.ExternalId, cloudid.TPolicyType(policy.PolicyType))
			if err != nil {
				logclient.AddSimpleActionLog(roleMap[id], logclient.ACT_DETACH_POLICY, err, self.GetUserCred(), false)
			}
		}
		err = iGroup.DetachPolicy(policy.ExternalId, cloudid.TPolicyType(policy.PolicyType))
		if err != nil {
			self.taskFailed(ctx, group, errors.Wrapf(err, "DetachPolicy %s", policy.Name))
			return
		}
	}

	self.taskComplete(ctx, group, iGroup)
}

func (self *CloudgroupSetPoliciesTask) taskComplete(ctx context.Context, group *models.SCloudgroup, iGroup cloudprovider.ICloudgroup) {
	group.SyncCloudpolicies(ctx, self.GetUserCred(), iGroup)
	group.SetStatus(ctx, self.GetUserCred(), apis.STATUS_AVAILABLE, "")
	self.SetStageComplete(ctx, nil)
}
