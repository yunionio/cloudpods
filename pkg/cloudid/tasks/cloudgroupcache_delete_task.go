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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/cloudid"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudid/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type CloudgroupcacheDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(CloudgroupcacheDeleteTask{})
}

func (self *CloudgroupcacheDeleteTask) taskFailed(ctx context.Context, cache *models.SCloudgroupcache, err error) {
	cache.SetStatus(self.GetUserCred(), api.CLOUD_GROUP_STATUS_DELETE_FAILED, err.Error())
	logclient.AddActionLogWithStartable(self, cache, logclient.ACT_DELETE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *CloudgroupcacheDeleteTask) taskComplete(ctx context.Context, cache *models.SCloudgroupcache) {
	cache.RealDelete(ctx, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, cache, logclient.ACT_DELETE, "", self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *CloudgroupcacheDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	cache := obj.(*models.SCloudgroupcache)
	if len(cache.ExternalId) == 0 {
		self.taskComplete(ctx, cache)
		return
	}
	iGroup, err := cache.GetICloudgroup()
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			self.taskComplete(ctx, cache)
			return
		}
		self.taskFailed(ctx, cache, errors.Wrap(err, "GetICloudgroup"))
		return
	}
	err = iGroup.Delete()
	if err != nil {
		self.taskFailed(ctx, cache, errors.Wrap(err, "iGroup.Delete"))
		return
	}
	self.taskComplete(ctx, cache)
}
