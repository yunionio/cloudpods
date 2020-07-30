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
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type CloudgroupDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(CloudgroupDeleteTask{})
}

func (self *CloudgroupDeleteTask) taskFailed(ctx context.Context, group *models.SCloudgroup, err jsonutils.JSONObject) {
	group.SetStatus(self.GetUserCred(), api.CLOUD_GROUP_STATUS_DELETE_FAILED, err.String())
	logclient.AddActionLogWithStartable(self, group, logclient.ACT_DELETE, err, self.UserCred, false)
	self.SetStageFailed(ctx, err)
}

func (self *CloudgroupDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	group := obj.(*models.SCloudgroup)

	caches, err := group.GetCloudgroupcaches()
	if err != nil {
		self.taskFailed(ctx, group, jsonutils.NewString(errors.Wrap(err, "GetCloudgroupcaches").Error()))
		return
	}

	for i := range caches {
		iGroup, err := caches[i].GetICloudgroup()
		if err != nil && errors.Cause(err) != cloudprovider.ErrNotFound {
			self.taskFailed(ctx, group, jsonutils.NewString(errors.Wrap(err, "caches[i].GetICloudgroup").Error()))
			return
		}
		if err == nil {
			err = iGroup.Delete()
			if err != nil {
				self.taskFailed(ctx, group, jsonutils.NewString(errors.Wrap(err, "iGroup.Delete").Error()))
				return
			}
		}
		caches[i].RealDelete(ctx, self.GetUserCred())
	}

	cnt, err := group.GetCloudgroupcacheCount()
	if err != nil {
		self.taskFailed(ctx, group, jsonutils.NewString(errors.Wrap(err, "GetCloudgroupcacheCount").Error()))
		return
	}
	if cnt == 0 {
		group.RealDelete(ctx, self.GetUserCred())
		logclient.AddActionLogWithStartable(self, group, logclient.ACT_DELETE, nil, self.UserCred, true)
		self.SetStageComplete(ctx, nil)
		return
	}
	self.SetStageComplete(ctx, nil)
}
