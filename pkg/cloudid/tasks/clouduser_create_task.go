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

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/cloudid/models"
	npk "yunion.io/x/onecloud/pkg/mcclient/modules/notify"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type ClouduserCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ClouduserCreateTask{})
}

func (self *ClouduserCreateTask) taskFailed(ctx context.Context, clouduser *models.SClouduser, err error) {
	clouduser.SetStatus(ctx, self.GetUserCred(), apis.STATUS_CREATE_FAILED, err.Error())
	logclient.AddActionLogWithStartable(self, clouduser, logclient.ACT_ALLOCATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *ClouduserCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	user := obj.(*models.SClouduser)

	provider, err := user.GetCloudprovider()
	if err != nil {
		self.taskFailed(ctx, user, errors.Wrapf(err, "GetCloudprovider"))
		return
	}

	account, err := provider.GetCloudaccount()
	if err != nil {
		self.taskFailed(ctx, user, errors.Wrapf(err, "GetCloudaccount"))
		return
	}

	driver, err := provider.GetDriver()
	if err != nil {
		self.taskFailed(ctx, user, errors.Wrapf(err, "GetDriver"))
		return
	}

	err = driver.RequestCreateClouduser(ctx, self.GetUserCred(), provider, user)
	if err != nil {
		self.taskFailed(ctx, user, errors.Wrapf(err, "RequestCreateClouduser"))
		return
	}

	user.SetStatus(ctx, self.GetUserCred(), apis.STATUS_AVAILABLE, "")

	if jsonutils.QueryBoolean(self.GetParams(), "notify", false) && len(user.Email) > 0 {
		msg := struct {
			Account     string
			Name        string
			Password    string
			IamLoginUrl string
			Id          string
		}{
			Id:          self.Id,
			IamLoginUrl: account.IamLoginUrl,
			Account:     account.AccountId,
		}
		msg.Password, _ = user.GetPassword()

		notifyclient.NotifyWithContact(ctx, []string{user.Email}, npk.NotifyByEmail, npk.NotifyPriorityNormal, "CLOUD_USER_CREATED", jsonutils.Marshal(msg))
	}

	self.SetStageComplete(ctx, nil)
}
