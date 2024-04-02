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

type ClouduserSetPoliciesTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ClouduserSetPoliciesTask{})
}

func (self *ClouduserSetPoliciesTask) taskFailed(ctx context.Context, user *models.SClouduser, err error) {
	user.SetStatus(ctx, self.GetUserCred(), apis.STATUS_UNKNOWN, err.Error())
	logclient.AddActionLogWithStartable(self, user, logclient.ACT_SYNC_CONF, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *ClouduserSetPoliciesTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	user := obj.(*models.SClouduser)

	iUser, err := user.GetIClouduser()
	if err != nil {
		self.taskFailed(ctx, user, errors.Wrap(err, "GetIClouduser"))
		return
	}

	input := struct {
		Add []api.SPolicy
		Del []api.SPolicy
	}{}
	err = self.GetParams().Unmarshal(&input)
	if err != nil {
		self.taskFailed(ctx, user, errors.Wrapf(err, "Unmarshal"))
		return
	}

	for _, policy := range input.Add {
		err = iUser.AttachPolicy(policy.ExternalId, cloudid.TPolicyType(policy.PolicyType))
		if err != nil {
			self.taskFailed(ctx, user, errors.Wrapf(err, "AttachPolicy %s", policy.Name))
			return
		}
	}

	for _, policy := range input.Del {
		err = iUser.DetachPolicy(policy.ExternalId, cloudid.TPolicyType(policy.PolicyType))
		if err != nil {
			self.taskFailed(ctx, user, errors.Wrapf(err, "DetachPolicy %s", policy.Name))
			return
		}
	}

	self.taskComplete(ctx, user, iUser)
}

func (self *ClouduserSetPoliciesTask) taskComplete(ctx context.Context, user *models.SClouduser, iUser cloudprovider.IClouduser) {
	user.SyncCloudpolicies(ctx, self.GetUserCred(), iUser)
	user.SetStatus(ctx, self.GetUserCred(), apis.STATUS_AVAILABLE, "")
	self.SetStageComplete(ctx, nil)
}
