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
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudid/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type CloudgroupDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(CloudgroupDeleteTask{})
}

func (self *CloudgroupDeleteTask) taskFailed(ctx context.Context, group *models.SCloudgroup, err error) {
	group.SetStatus(ctx, self.GetUserCred(), apis.STATUS_DELETE_FAILED, err.Error())
	logclient.AddActionLogWithStartable(self, group, logclient.ACT_DELETE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *CloudgroupDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	group := obj.(*models.SCloudgroup)

	iGroup, err := group.GetICloudgroup()
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			self.taskComplete(ctx, group)
			return
		}
		self.taskFailed(ctx, group, errors.Wrap(err, "GetICloudgroup"))
		return
	}
	users, err := iGroup.GetICloudusers()
	if err != nil {
		self.taskFailed(ctx, group, errors.Wrap(err, "GetICloudusers"))
		return
	}
	for i := range users {
		err = iGroup.RemoveUser(users[i].GetName())
		if err != nil {
			self.taskFailed(ctx, group, errors.Wrapf(err, "RemoveUser(%s)", users[i].GetName()))
			return
		}
	}
	policies, err := iGroup.GetICloudpolicies()
	if err != nil {
		self.taskFailed(ctx, group, errors.Wrapf(err, "GetICloudpolicies"))
		return
	}
	for i := range policies {
		err = iGroup.DetachPolicy(policies[i].GetGlobalId(), policies[i].GetPolicyType())
		if err != nil {
			self.taskFailed(ctx, group, errors.Wrapf(err, "DetachPolicy(%s)", policies[i].GetName()))
			return
		}
	}

	err = iGroup.Delete()
	if err != nil {
		self.taskFailed(ctx, group, errors.Wrap(err, "iGroup.Delete"))
		return
	}

	self.taskComplete(ctx, group)
}

func (self *CloudgroupDeleteTask) taskComplete(ctx context.Context, group *models.SCloudgroup) {
	logclient.AddActionLogWithStartable(self, group, logclient.ACT_DELETE, nil, self.UserCred, true)
	group.RealDelete(ctx, self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}
