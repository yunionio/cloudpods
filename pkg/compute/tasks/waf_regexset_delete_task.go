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

type WafRegexSetDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(WafRegexSetDeleteTask{})
}

func (self *WafRegexSetDeleteTask) taskFailed(ctx context.Context, regexset *models.SWafRegexSet, err error) {
	regexset.SetStatus(self.UserCred, api.WAF_REGEX_SET_STATUS_DELETE_FAILED, err.Error())
	logclient.AddActionLogWithStartable(self, regexset, logclient.ACT_DELETE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *WafRegexSetDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	regexset := obj.(*models.SWafRegexSet)
	caches, err := regexset.GetCaches()
	if err != nil {
		self.taskFailed(ctx, regexset, errors.Wrapf(err, "GetCaches"))
		return
	}
	for i := range caches {
		iCache, err := caches[i].GetICloudWafRegexSet(ctx)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				caches[i].RealDelete(ctx, self.GetUserCred())
				continue
			}
			self.taskFailed(ctx, regexset, errors.Wrapf(err, "GetICloudWafRegexSet"))
			return
		}
		err = iCache.Delete()
		if err != nil {
			self.taskFailed(ctx, regexset, errors.Wrapf(err, "iCache.Delete"))
			return
		}
		caches[i].RealDelete(ctx, self.GetUserCred())
	}
	self.taskComplete(ctx, regexset)
}

func (self *WafRegexSetDeleteTask) taskComplete(ctx context.Context, regexset *models.SWafRegexSet) {
	regexset.RealDelete(ctx, self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}
