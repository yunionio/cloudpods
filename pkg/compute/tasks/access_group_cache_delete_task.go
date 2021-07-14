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

// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or cachereed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific langucachee governing permissions and
// limitations under the License.

package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type AccessGroupCacheDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(AccessGroupCacheDeleteTask{})
}

func (self *AccessGroupCacheDeleteTask) taskFailed(ctx context.Context, cache *models.SAccessGroupCache, err error) {
	cache.SetStatus(self.UserCred, api.ACCESS_GROUP_STATUS_DELETE_FAILED, err.Error())
	logclient.AddActionLogWithStartable(self, cache, logclient.ACT_DELOCATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *AccessGroupCacheDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	cache := obj.(*models.SAccessGroupCache)

	iAccessGroup, err := cache.GetICloudAccessGroup()
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			self.taskComplete(ctx, cache)
			return
		}
		self.taskFailed(ctx, cache, errors.Wrapf(err, "GetICloudAccessGroup"))
		return
	}
	err = iAccessGroup.Delete()
	if err != nil {
		self.taskFailed(ctx, cache, errors.Wrapf(err, "iAccessGroup.Delete"))
		return
	}
	self.taskComplete(ctx, cache)
}

func (self *AccessGroupCacheDeleteTask) taskComplete(ctx context.Context, cache *models.SAccessGroupCache) {
	cache.RealDelete(ctx, self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}
