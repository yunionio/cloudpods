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

	api "yunion.io/x/onecloud/pkg/apis/cloudid"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudid/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type CloudroleDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(CloudroleDeleteTask{})
}

func (self *CloudroleDeleteTask) taskFailed(ctx context.Context, role *models.SCloudrole, err error) {
	role.SetStatus(self.GetUserCred(), api.CLOUD_ROLE_STATUS_DELETE_FAILED, err.Error())
	logclient.AddActionLogWithStartable(self, role, logclient.ACT_DELETE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *CloudroleDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	role := obj.(*models.SCloudrole)

	isPurge := jsonutils.QueryBoolean(self.Params, "purge", false)
	if len(role.ExternalId) == 0 || isPurge {
		role.RealDelete(ctx, self.GetUserCred())
		self.SetStageComplete(ctx, nil)
		return
	}

	account, err := role.GetCloudaccount()
	if err != nil {
		self.taskFailed(ctx, role, errors.Wrapf(err, "GetCloudaccount"))
		return
	}
	provider, err := account.GetProvider()
	if err != nil {
		self.taskFailed(ctx, role, errors.Wrapf(err, "GetProvider"))
		return
	}
	iRole, err := provider.GetICloudroleById(role.ExternalId)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			role.RealDelete(ctx, self.GetUserCred())
			self.SetStageComplete(ctx, nil)
			return
		}
		self.taskFailed(ctx, role, errors.Wrapf(err, "GetICloudroleById(%s)", role.ExternalId))
		return
	}
	err = iRole.Delete()
	if err != nil {
		self.taskFailed(ctx, role, errors.Wrapf(err, "iRole.Delete"))
		return
	}

	self.SetStageComplete(ctx, nil)
}
