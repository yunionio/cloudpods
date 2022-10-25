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
	case api.CLOUD_PROVIDER_HUAWEI, api.CLOUD_PROVIDER_HCSO, api.CLOUD_PROVIDER_HCS:
		return onHuaiweiPrepareLoadbalancerBackendgroup
	case api.CLOUD_PROVIDER_AWS:
		return onAwsPrepareLoadbalancerBackendgroup
	case api.CLOUD_PROVIDER_OPENSTACK:
		return onOpenstackPrepareLoadbalancerBackendgroup
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
		self.taskFail(ctx, lbr, jsonutils.NewString("huawei loadbalancer listener rule releated backend group not found"))
		return
	}

	lblis, err := lbr.GetLoadbalancerListener()
	if err != nil {
		self.taskFail(ctx, lbr, jsonutils.NewString(err.Error()))
		return
	}

	params := jsonutils.NewDict()
	params.Set("ruleId", jsonutils.NewString(lbr.GetId()))
	group, _ := models.HuaweiCachedLbbgManager.GetCachedBackendGroupByAssociateId(lbr.GetId())
	if group != nil {
		ilbbg, err := group.GetICloudLoadbalancerBackendGroup(ctx)
		if err != nil {
			self.taskFail(ctx, lbr, jsonutils.NewString(err.Error()))
			return
		}

		groupParams, err := lbbg.GetHuaweiBackendGroupParams(lblis, lbr)
		if err != nil {
			self.taskFail(ctx, lbr, jsonutils.NewString(err.Error()))
			return
		}
		groupParams.ListenerID = ""
		// 服务器组已经存在，直接同步即可
		if err := ilbbg.Sync(ctx, groupParams); err != nil {
			self.taskFail(ctx, lbr, jsonutils.NewString(err.Error()))
			return
		} else {
			group.SetModelManager(models.HuaweiCachedLbbgManager, group)
			if _, err := db.UpdateWithLock(ctx, group, func() error {
				group.AssociatedId = lbr.GetId()
				group.AssociatedType = api.LB_ASSOCIATE_TYPE_RULE
				return nil
			}); err != nil {
				self.taskFail(ctx, lbr, jsonutils.NewString(err.Error()))
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
		self.taskFail(ctx, lbr, jsonutils.NewString("aws loadbalancer listener rule releated backend group not found"))
		return
	}

	lblis, err := lbr.GetLoadbalancerListener()
	if err != nil {
		self.taskFail(ctx, lbr, jsonutils.NewString(err.Error()))
		return
	}

	params, err := lbbg.GetAwsBackendGroupParams(lblis, lbr)
	if err != nil {
		self.taskFail(ctx, lbr, jsonutils.NewString(err.Error()))
		return
	}

	group, _ := models.AwsCachedLbbgManager.GetUsableCachedBackendGroup(lblis.LoadbalancerId, lbr.BackendGroupId, lblis.ListenerType, lblis.HealthCheckType, lblis.HealthCheckInterval)
	if group != nil {
		ilbbg, err := group.GetICloudLoadbalancerBackendGroup(ctx)
		if err != nil {
			self.taskFail(ctx, lbr, jsonutils.NewString(err.Error()))
			return
		}

		// 服务器组已经存在，直接同步即可
		if err := ilbbg.Sync(ctx, params); err != nil {
			self.taskFail(ctx, lbr, jsonutils.NewString(err.Error()))
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

func onOpenstackPrepareLoadbalancerBackendgroup(ctx context.Context, region *models.SCloudregion, lbr *models.SLoadbalancerListenerRule, data jsonutils.JSONObject, self *LoadbalancerListenerRuleCreateTask) {
	lbbg := lbr.GetLoadbalancerBackendGroup()
	if lbbg == nil {
		self.taskFail(ctx, lbr, jsonutils.NewString("openstack loadbalancer listener rule releated backend group not found"))
		return
	}

	lblis, err := lbr.GetLoadbalancerListener()
	if err != nil {
		self.taskFail(ctx, lbr, jsonutils.NewString(err.Error()))
		return
	}

	params := jsonutils.NewDict()
	params.Set("ruleId", jsonutils.NewString(lbr.GetId()))
	group, _ := models.OpenstackCachedLbbgManager.GetCachedBackendGroupByAssociateId(lbr.GetId())
	if group != nil {
		ilbbg, err := group.GetICloudLoadbalancerBackendGroup(ctx)
		if err != nil {
			self.taskFail(ctx, lbr, jsonutils.NewString(err.Error()))
			return
		}

		groupParams, err := lbbg.GetOpenstackBackendGroupParams(lblis, lbr)
		if err != nil {
			self.taskFail(ctx, lbr, jsonutils.NewString(err.Error()))
			return
		}
		groupParams.ListenerID = ""
		// 服务器组已经存在，直接同步即可
		if err := ilbbg.Sync(ctx, groupParams); err != nil {
			self.taskFail(ctx, lbr, jsonutils.NewString(err.Error()))
			return
		} else {
			group.SetModelManager(models.OpenstackCachedLbbgManager, group)
			if _, err := db.UpdateWithLock(ctx, group, func() error {
				group.AssociatedId = lbr.GetId()
				group.AssociatedType = api.LB_ASSOCIATE_TYPE_RULE
				return nil
			}); err != nil {
				self.taskFail(ctx, lbr, jsonutils.NewString(err.Error()))
				return
			}

			self.OnCreateLoadbalancerListenerRule(ctx, lbr, data)
		}
	} else {
		// 服务器组不存在
		self.SetStage("OnCreateLoadbalancerListenerRule", nil)
		lbbg.StartOpenstackLoadBalancerBackendGroupCreateTask(ctx, self.GetUserCred(), params, self.GetTaskId())
	}
}

func (self *LoadbalancerListenerRuleCreateTask) taskFail(ctx context.Context, lbr *models.SLoadbalancerListenerRule, reason jsonutils.JSONObject) {
	lbr.SetStatus(self.GetUserCred(), api.LB_CREATE_FAILED, reason.String())
	db.OpsLog.LogEvent(lbr, db.ACT_ALLOCATE_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, lbr, logclient.ACT_CREATE, reason, self.UserCred, false)
	notifyclient.NotifySystemErrorWithCtx(ctx, lbr.Id, lbr.Name, api.LB_CREATE_FAILED, reason.String())
	lblis, _ := lbr.GetLoadbalancerListener()
	if lblis != nil {
		logclient.AddActionLogWithStartable(self, lblis, logclient.ACT_LB_ADD_LISTENER_RULE, reason, self.UserCred, false)
	}
	self.SetStageFailed(ctx, reason)
}

func (self *LoadbalancerListenerRuleCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lbr := obj.(*models.SLoadbalancerListenerRule)
	region, err := lbr.GetRegion()
	if err != nil {
		self.taskFail(ctx, lbr, jsonutils.NewString(err.Error()))
		return
	}

	self.OnPrepareLoadbalancerBackendgroup(ctx, region, lbr, data)
}

func (self *LoadbalancerListenerRuleCreateTask) OnPrepareLoadbalancerBackendgroup(ctx context.Context, region *models.SCloudregion, lbr *models.SLoadbalancerListenerRule, data jsonutils.JSONObject) {
	call := getOnPrepareLoadbalancerBackendgroupFunc(lbr.GetProviderName())
	call(ctx, region, lbr, data, self)
}

func (self *LoadbalancerListenerRuleCreateTask) OnCreateLoadbalancerListenerRule(ctx context.Context, lbr *models.SLoadbalancerListenerRule, data jsonutils.JSONObject) {
	region, err := lbr.GetRegion()
	if err != nil {
		self.taskFail(ctx, lbr, jsonutils.NewString(err.Error()))
		return
	}
	self.SetStage("OnLoadbalancerListenerRuleCreateComplete", nil)
	if err := region.GetDriver().RequestCreateLoadbalancerListenerRule(ctx, self.GetUserCred(), lbr, self); err != nil {
		self.taskFail(ctx, lbr, jsonutils.NewString(err.Error()))
	}
}

func (self *LoadbalancerListenerRuleCreateTask) OnCreateLoadbalancerListenerRuleFailed(ctx context.Context, lbr *models.SLoadbalancerListenerRule, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lbr, reason)
}

func (self *LoadbalancerListenerRuleCreateTask) OnLoadbalancerListenerRuleCreateComplete(ctx context.Context, lbr *models.SLoadbalancerListenerRule, data jsonutils.JSONObject) {
	lbr.SetStatus(self.GetUserCred(), api.LB_STATUS_ENABLED, "")
	db.OpsLog.LogEvent(lbr, db.ACT_ALLOCATE, lbr.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, lbr, logclient.ACT_CREATE, nil, self.UserCred, true)
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    lbr,
		Action: notifyclient.ActionCreate,
	})
	lblis, _ := lbr.GetLoadbalancerListener()
	if lblis != nil {
		logclient.AddActionLogWithStartable(self, lblis, logclient.ACT_LB_ADD_LISTENER_RULE, nil, self.UserCred, true)
	}
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerListenerRuleCreateTask) OnLoadbalancerListenerRuleCreateCompleteFailed(ctx context.Context, lbr *models.SLoadbalancerListenerRule, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lbr, reason)
}
