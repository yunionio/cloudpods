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
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SecurityGroupSyncRulesTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SecurityGroupSyncRulesTask{})
}

func (self *SecurityGroupSyncRulesTask) taskFailed(ctx context.Context, secgroup *models.SSecurityGroup, err error) {
	secgroup.ClearRuleDirty()
	secgroup.SetStatus(self.UserCred, api.SECGROUP_STATUS_READY, "")
	logclient.AddActionLogWithContext(ctx, secgroup, logclient.ACT_SYNC_CONF, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *SecurityGroupSyncRulesTask) taskComplete(ctx context.Context, secgroup *models.SSecurityGroup) {
	secgroup.ClearRuleDirty()
	secgroup.SetStatus(self.UserCred, api.SECGROUP_STATUS_READY, "")
	self.SetStageComplete(ctx, nil)
}

func (self *SecurityGroupSyncRulesTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	secgroup := obj.(*models.SSecurityGroup)
	caches, err := secgroup.GetSecurityGroupCaches()
	if err != nil {
		self.taskFailed(ctx, secgroup, errors.Wrapf(err, "GetSecurityGroupCaches"))
		return
	}

	for i := range caches {
		iSecgroup, err := caches[i].GetISecurityGroup(ctx)
		if err != nil {
			self.taskFailed(ctx, secgroup, errors.Wrapf(err, "GetISecurityGroup"))
			return
		}
		oldRules, _, err := caches[i].GetOldSecuritRuleSet(ctx)
		if err != nil {
			self.taskFailed(ctx, secgroup, errors.Wrapf(err, "GetOldSecuritRuleSet"))
			return
		}
		rules, err := iSecgroup.GetRules()
		if err != nil {
			self.taskFailed(ctx, secgroup, errors.Wrapf(err, "GetRules"))
			return
		}

		src := cloudprovider.NewSecRuleInfo(models.GetRegionDriver(api.CLOUD_PROVIDER_ONECLOUD))
		src.Rules = oldRules

		region, err := caches[i].GetRegion()
		if err != nil {
			self.taskFailed(ctx, secgroup, errors.Wrapf(err, "GetRegion"))
			return
		}
		dest := cloudprovider.NewSecRuleInfo(models.GetRegionDriver(region.Provider))
		dest.Rules = rules
		_, inAdds, outAdds, inDels, outDels := cloudprovider.CompareRules(src, dest, false)
		if len(inDels)+len(outDels)+len(inAdds)+len(outAdds) > 0 {
			self.taskFailed(ctx, secgroup, fmt.Errorf("secgroup rule not equal with cache %s(%s)", caches[i].Name, region.Provider))
			return
		}
	}
	for i := range caches {
		err := caches[i].SyncRules(ctx, false)
		if err != nil {
			logclient.AddActionLogWithContext(ctx, secgroup, logclient.ACT_SYNC_CONF, errors.Wrapf(err, "SyncRules"), self.UserCred, false)
		}
	}
	self.taskComplete(ctx, secgroup)
}
