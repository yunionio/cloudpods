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
)

type ClouduserSyncGroupsTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ClouduserSyncGroupsTask{})
}

func (self *ClouduserSyncGroupsTask) taskFailed(ctx context.Context, clouduser *models.SClouduser, err error) {
	clouduser.SetStatus(ctx, self.GetUserCred(), api.CLOUD_USER_STATUS_SYNC_GROUPS_FAILED, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *ClouduserSyncGroupsTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	user := obj.(*models.SClouduser)

	account, err := user.GetCloudaccount()
	if err != nil {
		self.taskFailed(ctx, user, errors.Wrap(err, "GetCloudaccount"))
		return
	}

	factory, err := account.GetProviderFactory()
	if err != nil {
		self.taskFailed(ctx, user, errors.Wrap(err, "GetCloudaccount"))
		return
	}

	if factory.IsSupportCreateCloudgroup() {
		groups, err := user.GetCloudgroups()
		if err != nil {
			self.taskFailed(ctx, user, errors.Wrap(err, "GetCloudgroups"))
			return
		}

		for i := range groups {
			cache, err := models.CloudgroupcacheManager.Register(&groups[i], account)
			if err != nil {
				self.taskFailed(ctx, user, errors.Wrap(err, "CloudgroupcacheManager.Register"))
				return
			}
			_, err = cache.GetOrCreateICloudgroup(ctx, self.GetUserCred())
			if err != nil {
				self.taskFailed(ctx, user, errors.Wrap(err, "GetOrCreateICloudgroup"))
				return
			}

			if factory.IsSupportClouduserPolicy() {
				err := user.SyncSystemCloudpoliciesForCloud(ctx, self.GetUserCred())
				if err != nil {
					self.taskFailed(ctx, user, errors.Wrap(err, "SyncCloudpoliciesForCloud"))
					return
				}
			}

			err = cache.SyncSystemCloudpoliciesForCloud(ctx, self.GetUserCred())
			if err != nil {
				self.taskFailed(ctx, user, errors.Wrap(err, "SyncSystemCloudpoliciesForCloud"))
				return
			}

			err = cache.SyncCustomCloudpoliciesForCloud(ctx, self.GetUserCred())
			if err != nil {
				self.taskFailed(ctx, user, errors.Wrap(err, "SyncCustomCloudpoliciesForCloud"))
				return
			}

			err = cache.SyncCloudusersForCloud(ctx, self.GetUserCred())
			if err != nil {
				self.taskFailed(ctx, user, errors.Wrap(err, "SyncCloudusersForCloud"))
				return
			}
		}
	} else if factory.IsSupportClouduserPolicy() {
		err := user.SyncSystemCloudpoliciesForCloud(ctx, self.GetUserCred())
		if err != nil {
			self.taskFailed(ctx, user, errors.Wrap(err, "SyncCloudpoliciesForCloud"))
			return
		}
	}

	if !self.IsSubtask() {
		user.SetStatus(ctx, self.GetUserCred(), api.CLOUD_USER_STATUS_AVAILABLE, "")
	}
	self.SetStageComplete(ctx, nil)
}
