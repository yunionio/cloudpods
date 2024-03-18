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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/cloudid"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudid/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type ClouduserResetPasswordTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ClouduserResetPasswordTask{})
}

func (self *ClouduserResetPasswordTask) taskFailed(ctx context.Context, clouduser *models.SClouduser, err error) {
	clouduser.SetStatus(ctx, self.GetUserCred(), api.CLOUD_USER_STATUS_RESET_PASSWORD_FAILED, err.Error())
	logclient.AddActionLogWithStartable(self, clouduser, logclient.ACT_RESET_PASSWORD, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *ClouduserResetPasswordTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	clouduser := obj.(*models.SClouduser)
	password, _ := self.GetParams().GetString("password")

	iUser, err := clouduser.GetIClouduser()
	if err != nil {
		self.taskFailed(ctx, clouduser, errors.Wrap(err, "GetIClouduser"))
		return
	}

	err = iUser.ResetPassword(password)
	if err != nil {
		self.taskFailed(ctx, clouduser, errors.Wrap(err, "ResetPassword"))
		return
	}
	clouduser.SyncWithClouduser(ctx, self.GetUserCred(), iUser)

	clouduser.SavePassword(password)
	logclient.AddActionLogWithStartable(self, clouduser, logclient.ACT_RESET_PASSWORD, "", self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}
