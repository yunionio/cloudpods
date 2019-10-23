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
	"database/sql"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SecurityGroupCacheDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SecurityGroupCacheDeleteTask{})
}

func (self *SecurityGroupCacheDeleteTask) taskFailed(ctx context.Context, cache *models.SSecurityGroupCache, err error) {
	cache.SetStatus(self.UserCred, api.SECGROUP_CACHE_STATUS_DELETE_FAILED, err.Error())
	secgroup, _ := cache.GetSecgroup()
	if secgroup != nil {
		logclient.AddActionLogWithStartable(self, secgroup, logclient.ACT_DELETE, err, self.UserCred, false)
	}
	self.SetStageFailed(ctx, err.Error())
}

func (self *SecurityGroupCacheDeleteTask) taskComplete(ctx context.Context, cache *models.SSecurityGroupCache) {
	cache.RealDelete(ctx, self.UserCred)
	self.SetStageComplete(ctx, nil)
}

func (self *SecurityGroupCacheDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cache := obj.(*models.SSecurityGroupCache)

	if len(cache.ExternalId) == 0 {
		self.taskComplete(ctx, cache)
		return
	}

	_, err := models.CloudproviderManager.FetchById(cache.ManagerId)
	if err == sql.ErrNoRows {
		self.taskComplete(ctx, cache)
		return
	}

	iRegion, err := cache.GetIRegion()
	if err != nil {
		if err == cloudprovider.ErrNotFound {
			self.taskComplete(ctx, cache)
			return
		}
		self.taskFailed(ctx, cache, errors.Wrap(err, "cache.GetIRegion"))
		return
	}
	iSecgroup, err := iRegion.GetISecurityGroupById(cache.ExternalId)
	if err != nil {
		if err == cloudprovider.ErrNotFound {
			self.taskComplete(ctx, cache)
			return
		}
		self.taskFailed(ctx, cache, errors.Wrap(err, "iRegion.GetIStoragecacheById"))
		return
	}
	err = iSecgroup.Delete()
	if err != nil {
		self.taskFailed(ctx, cache, err)
		return
	}
	self.taskComplete(ctx, cache)
}
