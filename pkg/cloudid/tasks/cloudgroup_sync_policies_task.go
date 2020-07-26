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
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type CloudgroupSyncPoliciesTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(CloudgroupSyncPoliciesTask{})
}

func (self *CloudgroupSyncPoliciesTask) taskFailed(ctx context.Context, group *models.SCloudgroup, err jsonutils.JSONObject) {
	group.SetStatus(self.GetUserCred(), api.CLOUD_GROUP_STATUS_SYNC_POLICIES, err.String())
	logclient.AddActionLogWithStartable(self, group, logclient.ACT_SYNC_POLICIES, err, self.UserCred, false)
	self.SetStageFailed(ctx, err)
}

func (self *CloudgroupSyncPoliciesTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	group := obj.(*models.SCloudgroup)

	factory, err := group.GetProviderFactory()
	if err != nil {
		self.taskFailed(ctx, group, jsonutils.NewString(errors.Wrap(err, "group.GetProviderFactory").Error()))
		return
	}

	if !factory.IsSupportCreateCloudgroup() {
		if factory.IsSupportClouduserPolicy() {
			users, err := group.GetCloudusers()
			if err != nil {
				self.taskFailed(ctx, group, jsonutils.NewString(errors.Wrap(err, "group.GetCloudusers").Error()))
				return
			}
			for i := range users {
				result, err := users[i].SyncCloudpoliciesForCloud(ctx)
				if err != nil {
					logclient.AddActionLogWithStartable(self, &users[i], logclient.ACT_SYNC_POLICIES, err, self.UserCred, false)
					continue
				}
				log.Infof("Sync cloudpolicies for user %s(%s) result: %s", users[i].Name, users[i].Id, result.Result())

				if result.AddErrCnt+result.DelErrCnt > 0 {
					self.taskFailed(ctx, group, jsonutils.NewString(result.AllError().Error()))
					return
				}
			}
		}
	} else {
		caches, err := group.GetCloudgroupcaches()
		if err != nil {
			self.taskFailed(ctx, group, jsonutils.NewString(errors.Wrap(err, "GetCloudgroupcaches").Error()))
			return
		}

		for i := range caches {
			result, err := caches[i].SyncCloudpoliciesForCloud(ctx)
			if err != nil {
				logclient.AddActionLogWithStartable(self, &caches[i], logclient.ACT_SYNC_POLICIES, err, self.UserCred, false)
				continue
			}
			log.Infof("Sync cloudpolicies for group cache %s(%s) result: %s", caches[i].Name, caches[i].Id, result.Result())

			if result.AddErrCnt+result.DelErrCnt > 0 {
				self.taskFailed(ctx, group, jsonutils.NewString(result.AllError().Error()))
				return
			}
		}
	}

	group.SetStatus(self.GetUserCred(), api.CLOUD_GROUP_STATUS_AVAILABLE, "")
	self.SetStageComplete(ctx, nil)
}
