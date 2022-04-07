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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type AccessGroupDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(AccessGroupDeleteTask{})
}

func (self *AccessGroupDeleteTask) taskFailed(ctx context.Context, ag *models.SAccessGroup, cache *models.SAccessGroupCache, err error) {
	ag.SetStatus(self.UserCred, api.ACCESS_GROUP_STATUS_DELETE_FAILED, err.Error())
	if cache != nil {
		cache.SetStatus(self.UserCred, api.ACCESS_GROUP_STATUS_DELETE_FAILED, err.Error())
	}
	logclient.AddActionLogWithStartable(self, ag, logclient.ACT_DELOCATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *AccessGroupDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	ag := obj.(*models.SAccessGroup)

	caches, err := ag.GetAccessGroupCaches()
	if err != nil {
		self.taskFailed(ctx, ag, nil, errors.Wrapf(err, "GetAccessGroupCaches"))
		return
	}
	for i := range caches {
		caches[i].SetStatus(self.GetUserCred(), api.ACCESS_GROUP_STATUS_DELETING, "")
		iAccessGroup, err := caches[i].GetICloudAccessGroup(ctx)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				caches[i].RealDelete(ctx, self.GetUserCred())
				continue
			}
			self.taskFailed(ctx, ag, &caches[i], errors.Wrapf(err, "cache.GetICloudAccessGroup"))
			return
		}
		err = iAccessGroup.Delete()
		if err != nil {
			self.taskFailed(ctx, ag, &caches[i], errors.Wrapf(err, "iAccessGroup.Delete"))
			return
		}
		caches[i].RealDelete(ctx, self.GetUserCred())
	}
	self.taskComplete(ctx, ag)
}

func (self *AccessGroupDeleteTask) taskComplete(ctx context.Context, ag *models.SAccessGroup) {
	ag.RealDelete(ctx, self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}
