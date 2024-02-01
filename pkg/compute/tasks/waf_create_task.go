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
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type WafCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(WafCreateTask{})
}

func (self *WafCreateTask) taskFailed(ctx context.Context, waf *models.SWafInstance, err error) {
	waf.SetStatus(ctx, self.UserCred, api.WAF_STATUS_CREATE_FAILED, err.Error())
	db.OpsLog.LogEvent(waf, db.ACT_ALLOCATE_FAIL, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, waf, logclient.ACT_ALLOCATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *WafCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	waf := obj.(*models.SWafInstance)

	iRegion, err := waf.GetIRegion(ctx)
	if err != nil {
		self.taskFailed(ctx, waf, errors.Wrapf(err, "GetIRegion"))
		return
	}
	params := api.WafInstanceCreateInput{}
	self.GetParams().Unmarshal(&params)
	opts := &cloudprovider.WafCreateOptions{
		Name:           waf.Name,
		Desc:           waf.Description,
		Type:           waf.Type,
		DefaultAction:  waf.DefaultAction,
		CloudResources: params.CloudResources,
		SourceIps:      params.SourceIps,
	}
	iWaf, err := iRegion.CreateICloudWafInstance(opts)
	if err != nil {
		self.taskFailed(ctx, waf, errors.Wrapf(err, "CreateICloudWafInstance"))
		return
	}
	cloudprovider.WaitStatus(iWaf, api.WAF_STATUS_AVAILABLE, time.Second*5, time.Minute*5)
	waf.SyncWithCloudWafInstance(ctx, self.GetUserCred(), iWaf)
	rules, err := iWaf.GetRules()
	if err == nil {
		waf.SyncWafRules(ctx, self.GetUserCred(), rules)
	}
	self.SetStageComplete(ctx, nil)
}
