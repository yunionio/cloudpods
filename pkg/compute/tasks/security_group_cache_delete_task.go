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
)

type SecurityGroupCacheDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SecurityGroupCacheDeleteTask{})
}

func (self *SecurityGroupCacheDeleteTask) taskFailed(ctx context.Context, cache *models.SSecurityGroupCache, err error) {
	cache.SetStatus(self.UserCred, api.SECGROUP_CACHE_STATUS_DELETE_FAILED, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *SecurityGroupCacheDeleteTask) taskComplete(ctx context.Context, cache *models.SSecurityGroupCache) {
	cache.RealDelete(ctx, self.UserCred)
	self.SetStageComplete(ctx, nil)
}

func (self *SecurityGroupCacheDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cache := obj.(*models.SSecurityGroupCache)

	iSecgroup, err := cache.GetISecurityGroup(ctx)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			self.taskComplete(ctx, cache)
			return
		}
		self.taskFailed(ctx, cache, errors.Wrapf(err, "GetISecurityGroup"))
		return
	}
	err = iSecgroup.Delete()
	if err != nil {
		self.taskFailed(ctx, cache, errors.Wrapf(err, "iSecgroup.Delete"))
		return
	}
	self.taskComplete(ctx, cache)
}
