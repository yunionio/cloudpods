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
	"yunion.io/x/onecloud/pkg/cloudid/models"
)

type SamlUserCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SamlUserCreateTask{})
}

func (self *SamlUserCreateTask) taskFailed(ctx context.Context, user *models.SSamluser, err error) {
	user.SetStatus(ctx, self.GetUserCred(), apis.STATUS_UNKNOWN, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *SamlUserCreateTask) taskComplete(ctx context.Context, user *models.SSamluser) {
	user.SetStatus(ctx, self.GetUserCred(), apis.STATUS_AVAILABLE, "")
	self.SetStageComplete(ctx, nil)
}

func (self *SamlUserCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	user := obj.(*models.SSamluser)

	group, err := user.GetCloudgroup()
	if err != nil {
		self.taskFailed(ctx, user, errors.Wrapf(err, "GetCloudgroup"))
		return
	}

	account, err := group.GetCloudaccount()
	if err != nil {
		self.taskFailed(ctx, user, errors.Wrapf(err, "GetCloudaccount"))
		return
	}

	driver, err := account.GetDriver()
	if err != nil {
		self.taskFailed(ctx, user, errors.Wrapf(err, "GetDriver"))
		return
	}

	err = driver.RequestCreateRoleForSamlUser(ctx, self.GetUserCred(), account, group, user)
	if err != nil {
		self.taskFailed(ctx, user, errors.Wrapf(err, "RequestCreateRoleForSamlUser"))
		return
	}

	self.taskComplete(ctx, user)
}
