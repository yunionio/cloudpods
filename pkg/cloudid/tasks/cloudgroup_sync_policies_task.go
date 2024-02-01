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

type CloudgroupSyncPoliciesTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(CloudgroupSyncPoliciesTask{})
}

func (self *CloudgroupSyncPoliciesTask) taskFailed(ctx context.Context, group *models.SCloudgroup, err error) {
	group.SetStatus(ctx, self.GetUserCred(), api.CLOUD_GROUP_STATUS_SYNC_POLICIES_FAILED, err.Error())
	logclient.AddActionLogWithStartable(self, group, logclient.ACT_SYNC_POLICIES, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *CloudgroupSyncPoliciesTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	group := obj.(*models.SCloudgroup)

	factory, err := group.GetProviderFactory()
	if err != nil {
		self.taskFailed(ctx, group, errors.Wrap(err, "group.GetProviderFactory"))
		return
	}

	if factory.IsSupportCreateCloudgroup() {
		caches, err := group.GetCloudgroupcaches()
		if err != nil {
			self.taskFailed(ctx, group, errors.Wrap(err, "GetCloudgroupcaches"))
			return
		}

		for i := range caches {
			err := caches[i].SyncSystemCloudpoliciesForCloud(ctx, self.GetUserCred())
			if err != nil {
				self.taskFailed(ctx, group, errors.Wrapf(err, "SyncSystemCloudpoliciesForCloud"))
				return
			}
			err = caches[i].SyncCustomCloudpoliciesForCloud(ctx, self.GetUserCred())
			if err != nil {
				self.taskFailed(ctx, group, errors.Wrapf(err, "SyncCustomCloudpoliciesForCloud"))
				return
			}
		}
	}

	if !factory.IsSupportCreateCloudgroup() {
		users, err := group.GetCloudusers()
		if err != nil {
			self.taskFailed(ctx, group, errors.Wrap(err, "group.GetCloudusers"))
			return
		}
		for i := range users {
			err := users[i].SyncSystemCloudpoliciesForCloud(ctx, self.GetUserCred())
			if err != nil {
				self.taskFailed(ctx, group, errors.Wrap(err, "SyncSystemCloudpoliciesForCloud"))
				return
			}
			err = users[i].SyncCustomCloudpoliciesForCloud(ctx, self.GetUserCred())
			if err != nil {
				self.taskFailed(ctx, group, errors.Wrap(err, "SyncSystemCloudpoliciesForCloud"))
				return
			}
		}
	}

	group.SetStatus(ctx, self.GetUserCred(), api.CLOUD_GROUP_STATUS_AVAILABLE, "")
	self.SetStageComplete(ctx, nil)
}
