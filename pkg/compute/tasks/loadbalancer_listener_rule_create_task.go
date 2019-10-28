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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type LoadbalancerListenerRuleCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerListenerRuleCreateTask{})
}

func getOnPrepareLoadbalancerBackendgroupFunc(provider string) func(ctx context.Context, region *models.SCloudregion, lbr *models.SLoadbalancerListenerRule, data jsonutils.JSONObject, self *LoadbalancerListenerRuleCreateTask) {
	switch provider {
	case api.CLOUD_PROVIDER_HUAWEI:
		return onHuaiweiPrepareLoadbalancerBackendgroup
	case api.CLOUD_PROVIDER_AWS:
		return onAwsPrepareLoadbalancerBackendgroup
	default:
		return onPrepareLoadbalancerBackendgroup
	}
}

func onPrepareLoadbalancerBackendgroup(ctx context.Context, region *models.SCloudregion, lbr *models.SLoadbalancerListenerRule, data jsonutils.JSONObject, self *LoadbalancerListenerRuleCreateTask) {
	self.OnCreateLoadbalancerListenerRule(ctx, lbr, data)
	return
}

func onHuaiweiPrepareLoadbalancerBackendgroup(ctx context.Context, region *models.SCloudregion, lbr *models.SLoadbalancerListenerRule, data jsonutils.JSONObject, self *LoadbalancerListenerRuleCreateTask) {
	lbbg := lbr.GetLoadbalancerBackendGroup()
	if lbbg == nil {
		self.taskFail(ctx, lbr, "huawei loadbalancer listener rule releated backend group not found")
		return
	}

	lblis := lbr.GetLoadbalancerListener()
	if lblis == nil {
		self.taskFail(ctx, lbr, "huawei loadbalancer listener rule releated listener not found")
		return
	}

	params := jsonutils.NewDict()
	params.Set("ruleId", jsonutils.NewString(lbr.GetId()))
	group, _ := models.HuaweiCachedLbbgManager.GetUsableCachedBackendGroup(lbbg.GetId(), lblis.ListenerType)
	if group != nil {
		ilbbg, err := group.GetICloudLoadbalancerBackendGroup()
		if err != nil {
			self.taskFail(ctx, lbr, err.Error())
			return
		}

		groupParams, err := lbbg.GetHuaweiBackendGroupParams(lblis, lbr)
		if err != nil {
			self.taskFail(ctx, lbr, err.Error())
			return
		}
		groupParams.ListenerID = ""
		// 服务器组已经存在，直接同步即可
		if err := ilbbg.Sync(groupParams); err != nil {
			self.taskFail(ctx, lbr, err.Error())
			return
		} else {
			group.SetModelManager(models.HuaweiCachedLbbgManager, group)
			if _, err := db.UpdateWithLock(ctx, group, func() error {
				group.AssociatedId = lbr.GetId()
				group.AssociatedType = api.LB_ASSOCIATE_TYPE_RULE
				return nil
			}); err != nil {
				self.taskFail(ctx, lbr, err.Error())
				return
			}

			self.OnCreateLoadbalancerListenerRule(ctx, lbr, data)
		}
	} else {
		// 服务器组不存在
		self.SetStage("OnCreateLoadbalancerListenerRule", nil)
		lbbg.StartHuaweiLoadBalancerBackendGroupCreateTask(ctx, self.GetUserCred(), params, self.GetTaskId())
	}
}

func onAwsPrepareLoadbalancerBackendgroup(ctx context.Context, region *models.SCloudregion, lbr *models.SLoadbalancerListenerRule, data jsonutils.JSONObject, self *LoadbalancerListenerRuleCreateTask) {
	lbbg := lbr.GetLoadbalancerBackendGroup()
	if lbbg == nil {
		self.taskFail(ctx, lbr, "aws loadbalancer listener rule releated backend group not found")
		return
	}

	lblis := lbr.GetLoadbalancerListener()
	if lblis == nil {
		self.taskFail(ctx, lbr, "aws loadbalancer listener rule releated listener not found")
		return
	}

	params, err := lbbg.GetAwsBackendGroupParams(lblis, lbr)
	if err != nil {
		self.taskFail(ctx, lbr, err.Error())
		return
	}

	group, _ := models.AwsCachedLbbgManager.GetUsableCachedBackendGroup(lblis.LoadbalancerId, lblis.BackendGroupId, lblis.ListenerType, lblis.HealthCheckType, lblis.HealthCheckInterval)
	if group != nil {
		ilbbg, err := group.GetICloudLoadbalancerBackendGroup()
		if err != nil {
			self.taskFail(ctx, lbr, err.Error())
			return
		}

		// 服务器组已经存在，直接同步即可
		if err := ilbbg.Sync(params); err != nil {
			self.taskFail(ctx, lbr, err.Error())
			return
		}

		self.OnCreateLoadbalancerListenerRule(ctx, lbr, data)
	} else {
		paramsObj := jsonutils.Marshal(params).(*jsonutils.JSONDict)
		// 服务器组不存在
		self.SetStage("OnCreateLoadbalancerListenerRule", nil)
		lbbg.StartAwsLoadBalancerBackendGroupCreateTask(ctx, self.GetUserCred(), paramsObj, self.GetTaskId())
	}
}

func (self *LoadbalancerListenerRuleCreateTask) taskFail(ctx context.Context, lbr *models.SLoadbalancerListenerRule, reason string) {
	lbr.SetStatus(self.GetUserCred(), api.LB_CREATE_FAILED, reason)
	db.OpsLog.LogEvent(lbr, db.ACT_ALLOCATE_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, lbr, logclient.ACT_CREATE, reason, self.UserCred, false)
	notifyclient.NotifySystemError(lbr.Id, lbr.Name, api.LB_CREATE_FAILED, reason)
	lblis := lbr.GetLoadbalancerListener()
	if lblis != nil {
		logclient.AddActionLogWithStartable(self, lblis, logclient.ACT_LB_ADD_LISTENER_RULE, reason, self.UserCred, false)
	}
	self.SetStageFailed(ctx, reason)
}

func (self *LoadbalancerListenerRuleCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lbr := obj.(*models.SLoadbalancerListenerRule)
	region := lbr.GetRegion()
	if region == nil {
		self.taskFail(ctx, lbr, fmt.Sprintf("failed to find region for lbr %s", lbr.Name))
		return
	}

	self.OnPrepareLoadbalancerBackendgroup(ctx, region, lbr, data)
}

func (self *LoadbalancerListenerRuleCreateTask) OnPrepareLoadbalancerBackendgroup(ctx context.Context, region *models.SCloudregion, lbr *models.SLoadbalancerListenerRule, data jsonutils.JSONObject) {
	call := getOnPrepareLoadbalancerBackendgroupFunc(lbr.GetProviderName())
	call(ctx, region, lbr, data, self)
}

func (self *LoadbalancerListenerRuleCreateTask) OnCreateLoadbalancerListenerRule(ctx context.Context, lbr *models.SLoadbalancerListenerRule, data jsonutils.JSONObject) {
	region := lbr.GetRegion()
	if region == nil {
		self.taskFail(ctx, lbr, fmt.Sprintf("failed to find region for lbr %s", lbr.Name))
		return
	}
	self.SetStage("OnLoadbalancerListenerRuleCreateComplete", nil)
	if err := region.GetDriver().RequestCreateLoadbalancerListenerRule(ctx, self.GetUserCred(), lbr, self); err != nil {
		self.taskFail(ctx, lbr, err.Error())
	}
}

func (self *LoadbalancerListenerRuleCreateTask) OnCreateLoadbalancerListenerRuleFailed(ctx context.Context, lbr *models.SLoadbalancerListenerRule, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lbr, reason.String())
}

func (self *LoadbalancerListenerRuleCreateTask) OnLoadbalancerListenerRuleCreateComplete(ctx context.Context, lbr *models.SLoadbalancerListenerRule, data jsonutils.JSONObject) {
	lbr.SetStatus(self.GetUserCred(), api.LB_STATUS_ENABLED, "")
	db.OpsLog.LogEvent(lbr, db.ACT_ALLOCATE, lbr.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, lbr, logclient.ACT_CREATE, nil, self.UserCred, true)
	lblis := lbr.GetLoadbalancerListener()
	if lblis != nil {
		logclient.AddActionLogWithStartable(self, lblis, logclient.ACT_LB_ADD_LISTENER_RULE, nil, self.UserCred, true)
	}
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerListenerRuleCreateTask) OnLoadbalancerListenerRuleCreateCompleteFailed(ctx context.Context, lbr *models.SLoadbalancerListenerRule, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lbr, reason.String())
}
