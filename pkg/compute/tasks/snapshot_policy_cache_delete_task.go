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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SnapshotPolicyCacheDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SnapshotPolicyCacheDeleteTask{})
}

func (self *SnapshotPolicyCacheDeleteTask) taskFailed(ctx context.Context, cache *models.SSnapshotPolicyCache,
	err error) {

	cache.SetStatus(self.UserCred, api.SNAPSHOT_POLICY_CACHE_STATUS_DELETE_FAILED, err.Error())
	self.SetStageFailed(ctx, err.Error())
}

func (self *SnapshotPolicyCacheDeleteTask) taskComplete(ctx context.Context, cache *models.SSnapshotPolicyCache) {
	cache.RealDetele(ctx, self.UserCred)
	self.SetStageComplete(ctx, nil)
}

func (self *SnapshotPolicyCacheDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel,
	data jsonutils.JSONObject) {

	cache := obj.(*models.SSnapshotPolicyCache)

	if len(cache.ExternalId) == 0 {
		self.taskComplete(ctx, cache)
		return
	}

	err := cache.DeleteCloudSnapshotPolicy()
	if err != nil {
		self.taskFailed(ctx, cache, err)
		return
	}
	self.taskComplete(ctx, cache)
}
