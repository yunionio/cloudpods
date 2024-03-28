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

	api "yunion.io/x/onecloud/pkg/apis/cloudid"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudid/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type CloudgroupSetUsersTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(CloudgroupSetUsersTask{})
}

func (self *CloudgroupSetUsersTask) taskFailed(ctx context.Context, group *models.SCloudgroup, err error) {
	group.SetStatus(ctx, self.GetUserCred(), apis.STATUS_UNKNOWN, err.Error())
	logclient.AddActionLogWithStartable(self, group, logclient.ACT_SYNC_CONF, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *CloudgroupSetUsersTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	group := obj.(*models.SCloudgroup)

	iGroup, err := group.GetICloudgroup()
	if err != nil {
		self.taskFailed(ctx, group, errors.Wrap(err, "GetICloudgroup"))
		return
	}

	input := struct {
		Add []api.GroupUser
		Del []api.GroupUser
	}{}
	err = self.GetParams().Unmarshal(&input)
	if err != nil {
		self.taskFailed(ctx, group, errors.Wrapf(err, "Unmarshal"))
		return
	}

	for _, user := range input.Add {
		err = iGroup.AddUser(user.Name)
		if err != nil {
			self.taskFailed(ctx, group, errors.Wrapf(err, "AddUser %s", user.Name))
			return
		}
	}

	for _, user := range input.Del {
		err = iGroup.RemoveUser(user.Name)
		if err != nil {
			self.taskFailed(ctx, group, errors.Wrapf(err, "RemoveUser %s", user.Name))
			return
		}
	}

	self.taskComplete(ctx, group, iGroup)
}

func (self *CloudgroupSetUsersTask) taskComplete(ctx context.Context, group *models.SCloudgroup, iGroup cloudprovider.ICloudgroup) {
	group.SyncCloudusers(ctx, self.GetUserCred(), iGroup)
	group.SetStatus(ctx, self.GetUserCred(), apis.STATUS_AVAILABLE, "")
	self.SetStageComplete(ctx, nil)
}
