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

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type WafIPSetDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(WafIPSetDeleteTask{})
}

func (self *WafIPSetDeleteTask) taskFailed(ctx context.Context, ipset *models.SWafIPSet, err error) {
	ipset.SetStatus(ctx, self.UserCred, apis.STATUS_DELETE_FAILED, err.Error())
	logclient.AddActionLogWithStartable(self, ipset, logclient.ACT_DELETE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *WafIPSetDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	ipset := obj.(*models.SWafIPSet)
	iIpSet, err := ipset.GetICloudWafIPSet(ctx)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			self.taskComplete(ctx, ipset)
			return
		}
		self.taskFailed(ctx, ipset, errors.Wrapf(err, "GetICloudWafIPSet"))
		return
	}
	err = iIpSet.Delete()
	if err != nil {
		self.taskFailed(ctx, ipset, errors.Wrapf(err, "iCache.Delete"))
		return
	}
	self.taskComplete(ctx, ipset)
}

func (self *WafIPSetDeleteTask) taskComplete(ctx context.Context, ipset *models.SWafIPSet) {
	ipset.RealDelete(ctx, self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}
