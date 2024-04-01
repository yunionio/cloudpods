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
)

type CloudgroupSyncstatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(CloudgroupSyncstatusTask{})
}

func (self *CloudgroupSyncstatusTask) taskFailed(ctx context.Context, group *models.SCloudgroup, err error) {
	group.SetStatus(ctx, self.GetUserCred(), apis.STATUS_UNKNOWN, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *CloudgroupSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	group := obj.(*models.SCloudgroup)

	iGroup, err := group.GetICloudgroup()
	if err != nil {
		self.taskFailed(ctx, group, errors.Wrapf(err, "GetICloudgroup"))
		return
	}

	err = group.SyncWithCloudgroup(ctx, self.GetUserCred(), iGroup)
	if err != nil {
		self.taskFailed(ctx, group, errors.Wrapf(err, "SyncWithCloudgroup"))
		return
	}

	group.SyncCloudusers(ctx, self.GetUserCred(), iGroup)
	group.SyncCloudpolicies(ctx, self.GetUserCred(), iGroup)

	self.SetStageComplete(ctx, nil)
}
