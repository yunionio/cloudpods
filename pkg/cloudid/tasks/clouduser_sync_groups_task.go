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
	"yunion.io/x/log"
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
	clouduser.SetStatus(self.GetUserCred(), api.CLOUD_USER_STATUS_SYNC_GROUPS_FAILED, err.Error())
	self.SetStageFailed(ctx, err.Error())
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
			result, err := cache.SyncCloudusersForCloud(ctx)
			if err != nil {
				self.taskFailed(ctx, user, errors.Wrap(err, "SyncCloudusersForCloud"))
				return
			}
			log.Infof("sync cloudusers for cache %s(%s) result: %s", cache.Name, cache.Id, result.Result())

			if result.AddErrCnt+result.DelErrCnt > 0 {
				self.taskFailed(ctx, user, result.AllError())
				return
			}

			result, err = cache.SyncCloudpoliciesForCloud(ctx)
			if err != nil {
				self.taskFailed(ctx, user, errors.Wrap(err, "SyncCloudpoliciesForCloud"))
				return
			}

			if result.AddErrCnt+result.DelErrCnt > 0 {
				self.taskFailed(ctx, user, result.AllError())
				return
			}
			log.Infof("sync cloudpolicies for cache %s(%s) result: %s", cache.Name, cache.Id, result.Result())
		}
	} else if factory.IsSupportClouduserPolicy() {
		result, err := user.SyncCloudpoliciesForCloud(ctx)
		if err != nil {
			self.taskFailed(ctx, user, errors.Wrap(err, "SyncCloudpoliciesForCloud"))
			return
		}
		log.Infof("sync cloudpolicies for user %s(%s) result: %s", user.Name, user.Id, result.Result())

		if result.AddErrCnt+result.DelErrCnt > 0 {
			self.taskFailed(ctx, user, result.AllError())
			return
		}
	}

	if !self.IsSubtask() {
		user.SetStatus(self.GetUserCred(), api.CLOUD_USER_STATUS_AVAILABLE, "")
	}
	self.SetStageComplete(ctx, nil)
}
