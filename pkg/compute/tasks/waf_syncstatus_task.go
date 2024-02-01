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
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type WafSyncstatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(WafSyncstatusTask{})
}

func (self *WafSyncstatusTask) taskFailed(ctx context.Context, waf *models.SWafInstance, err error) {
	waf.SetStatus(ctx, self.UserCred, api.WAF_STATUS_UNKNOWN, err.Error())
	logclient.AddActionLogWithStartable(self, waf, logclient.ACT_SYNC_STATUS, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *WafSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	waf := obj.(*models.SWafInstance)
	iWaf, err := waf.GetICloudWafInstance(ctx)
	if err != nil {
		self.taskFailed(ctx, waf, errors.Wrapf(err, "GetICloudWafInstance"))
		return
	}
	waf.SyncWithCloudWafInstance(ctx, self.GetUserCred(), iWaf)
	rules, err := iWaf.GetRules()
	if err == nil {
		result := waf.SyncWafRules(ctx, self.GetUserCred(), rules)
		log.Infof("Sync waf %s rules result: %s", waf.Name, result.Result())
	}
	self.SetStageComplete(ctx, nil)
}
