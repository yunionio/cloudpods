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

type ClouduserSetGroupsTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ClouduserSetGroupsTask{})
}

func (self *ClouduserSetGroupsTask) taskFailed(ctx context.Context, user *models.SClouduser, err error) {
	user.SetStatus(ctx, self.GetUserCred(), apis.STATUS_UNKNOWN, err.Error())
	logclient.AddActionLogWithStartable(self, user, logclient.ACT_SYNC_CONF, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *ClouduserSetGroupsTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	user := obj.(*models.SClouduser)

	iUser, err := user.GetIClouduser()
	if err != nil {
		self.taskFailed(ctx, user, errors.Wrap(err, "GetIClouduser"))
		return
	}

	input := struct {
		Add []api.SGroup
		Del []api.SGroup
	}{}
	err = self.GetParams().Unmarshal(&input)
	if err != nil {
		self.taskFailed(ctx, user, errors.Wrapf(err, "Unmarshal"))
		return
	}

	for _, item := range input.Add {
		groupObj, err := models.CloudgroupManager.FetchById(item.Id)
		if err != nil {
			self.taskFailed(ctx, user, errors.Wrapf(err, "GetCloudgroup"))
		}
		group := groupObj.(*models.SCloudgroup)
		iGroup, err := group.GetICloudgroup()
		if err != nil {
			self.taskFailed(ctx, user, errors.Wrapf(err, "GetICloudgroup"))
			return
		}
		err = iGroup.AddUser(user.Name)
		if err != nil {
			self.taskFailed(ctx, user, errors.Wrapf(err, "add user for group %s", group.Name))
			return
		}
	}

	for _, item := range input.Del {
		group, err := user.GetCloudgroup(item.Id)
		if err != nil {
			self.taskFailed(ctx, user, errors.Wrapf(err, "GetCloudgroup"))
			return
		}
		iGroup, err := group.GetICloudgroup()
		if err != nil {
			self.taskFailed(ctx, user, errors.Wrapf(err, "GetICloudgroup"))
			return
		}
		err = iGroup.RemoveUser(user.Name)
		if err != nil {
			self.taskFailed(ctx, user, errors.Wrapf(err, "remove user from group %s", group.Name))
			return
		}
	}

	self.taskComplete(ctx, user, iUser)
}

func (self *ClouduserSetGroupsTask) taskComplete(ctx context.Context, user *models.SClouduser, iUser cloudprovider.IClouduser) {
	user.SyncCloudgroups(ctx, self.GetUserCred(), iUser)
	user.SetStatus(ctx, self.GetUserCred(), apis.STATUS_AVAILABLE, "")
	self.SetStageComplete(ctx, nil)
}
