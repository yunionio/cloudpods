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
	role.SetStatus(ctx, self.GetUserCred(), apis.STATUS_DELETE_FAILED, err.Error())
	logclient.AddActionLogWithStartable(self, role, logclient.ACT_DELETE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *CloudroleDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	role := obj.(*models.SCloudrole)

	iRole, err := role.GetICloudrole()
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			self.taskComplete(ctx, role)
			return
		}
		self.taskFailed(ctx, role, errors.Wrapf(err, "GetICloudrole(%s)", role.ExternalId))
		return
	}

	err = iRole.Delete()
	if err != nil {
		self.taskFailed(ctx, role, errors.Wrapf(err, "iRole.Delete"))
		return
	}

	self.taskComplete(ctx, role)
}

func (self *CloudroleDeleteTask) taskComplete(ctx context.Context, role *models.SCloudrole) {
	role.RealDelete(ctx, self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}
