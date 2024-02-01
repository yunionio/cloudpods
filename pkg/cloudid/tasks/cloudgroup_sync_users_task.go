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

type CloudgroupSyncUsersTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(CloudgroupSyncUsersTask{})
}

func (self *CloudgroupSyncUsersTask) taskFailed(ctx context.Context, group *models.SCloudgroup, err error) {
	group.SetStatus(ctx, self.GetUserCred(), api.CLOUD_GROUP_STATUS_SYNC_USERS_FAILED, err.Error())
	logclient.AddActionLogWithStartable(self, group, logclient.ACT_SYNC_USERS, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *CloudgroupSyncUsersTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	group := obj.(*models.SCloudgroup)

	factory, err := group.GetProviderFactory()
	if err != nil {
		self.taskFailed(ctx, group, errors.Wrap(err, "GetProviderFactory"))
		return
	}

	users, err := group.GetCloudusers()
	if err != nil {
		self.taskFailed(ctx, group, errors.Wrap(err, "GetCloudusers"))
		return
	}

	if !factory.IsSupportCreateCloudgroup() && factory.IsSupportClouduserPolicy() {
		for i := range users {
			err := users[i].SyncSystemCloudpoliciesForCloud(ctx, self.GetUserCred())
			if err != nil {
				self.taskFailed(ctx, group, errors.Wrap(err, "SyncSystemCloudpoliciesForCloud"))
				return
			}
			err = users[i].SyncCustomCloudpoliciesForCloud(ctx, self.GetUserCred())
			if err != nil {
				self.taskFailed(ctx, group, errors.Wrap(err, "SyncCustomCloudpoliciesForCloud"))
				return
			}
		}
	} else {
		caches, err := group.GetCloudgroupcaches()
		if err != nil {
			self.taskFailed(ctx, group, errors.Wrap(err, "GetCloudgroupcaches"))
			return
		}

		accounts := map[string]string{}

		for i := range caches {
			account, err := caches[i].GetCloudaccount()
			if err != nil {
				self.taskFailed(ctx, group, errors.Wrapf(err, "GetCloudaccount"))
				return
			}
			_, err = caches[i].GetOrCreateICloudgroup(ctx, self.GetUserCred())
			if err != nil {
				self.taskFailed(ctx, group, errors.Wrapf(err, "GetOrCreateICloudgroup"))
				return
			}
			accounts[account.Id] = account.Name

			err = caches[i].SyncCloudusersForCloud(ctx, self.GetUserCred())
			if err != nil {
				self.taskFailed(ctx, group, errors.Wrapf(err, "SyncCloudusersForCloud"))
				return
			}
		}

		for i := range users {
			if _, ok := accounts[users[i].CloudaccountId]; !ok {
				account, err := users[i].GetCloudaccount()
				if err != nil {
					self.taskFailed(ctx, group, errors.Wrap(err, "GetCloudaccount"))
					return
				}

				cache, err := models.CloudgroupcacheManager.Register(group, account)
				if err != nil {
					self.taskFailed(ctx, group, errors.Wrap(err, "CloudgroupcacheManager.Register"))
					return
				}

				_, err = cache.GetOrCreateICloudgroup(ctx, self.GetUserCred())
				if err != nil {
					self.taskFailed(ctx, group, errors.Wrap(err, "GetOrCreateICloudgroup"))
					return
				}

				err = cache.SyncCloudusersForCloud(ctx, self.GetUserCred())
				if err != nil {
					self.taskFailed(ctx, group, errors.Wrapf(err, "SyncCloudusersForCloud"))
					return
				}
			}
		}
	}

	group.SetStatus(ctx, self.GetUserCred(), api.CLOUD_GROUP_STATUS_AVAILABLE, "")
	self.SetStageComplete(ctx, nil)
}
