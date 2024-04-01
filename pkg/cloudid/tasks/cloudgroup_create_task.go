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
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudid/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type CloudgroupCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(CloudgroupCreateTask{})
}

func (self *CloudgroupCreateTask) taskFailed(ctx context.Context, group *models.SCloudgroup, err error) {
	group.SetStatus(ctx, self.GetUserCred(), apis.STATUS_CREATE_FAILED, err.Error())
	logclient.AddActionLogWithStartable(self, group, logclient.ACT_ALLOCATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *CloudgroupCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	group := obj.(*models.SCloudgroup)

	provider, err := group.GetCloudprovider()
	if err != nil {
		self.taskFailed(ctx, group, errors.Wrapf(err, "GetCloudprovider"))
		return
	}

	driver, err := provider.GetDriver()
	if err != nil {
		self.taskFailed(ctx, group, errors.Wrapf(err, "GetDriver"))
		return
	}

	err = driver.RequestCreateCloudgroup(ctx, self.GetUserCred(), provider, group)
	if err != nil {
		self.taskFailed(ctx, group, errors.Wrapf(err, "RequestCreateCloudgroup"))
		return
	}

	self.SetStageComplete(ctx, nil)
}
