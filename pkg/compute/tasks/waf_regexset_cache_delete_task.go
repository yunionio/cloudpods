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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type WafRegexSetCacheDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(WafRegexSetCacheDeleteTask{})
}

func (self *WafRegexSetCacheDeleteTask) taskFailed(ctx context.Context, cache *models.SWafRegexSetCache, err error) {
	cache.SetStatus(self.UserCred, api.WAF_REGEX_SET_STATUS_DELETE_FAILED, err.Error())
	logclient.AddActionLogWithStartable(self, cache, logclient.ACT_DELETE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *WafRegexSetCacheDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	cache := obj.(*models.SWafRegexSetCache)
	iCache, err := cache.GetICloudWafRegexSet(ctx)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			self.taskComplete(ctx, cache)
			return
		}
		self.taskFailed(ctx, cache, errors.Wrapf(err, "GetICloudWafRegexSet"))
		return
	}
	err = iCache.Delete()
	if err != nil {
		self.taskFailed(ctx, cache, errors.Wrapf(err, "iCache.Delete"))
		return
	}
	self.taskComplete(ctx, cache)
}

func (self *WafRegexSetCacheDeleteTask) taskComplete(ctx context.Context, cache *models.SWafRegexSetCache) {
	cache.RealDelete(ctx, self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}
