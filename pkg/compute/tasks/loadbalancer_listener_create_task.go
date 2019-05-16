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

type LoadbalancerListenerCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerListenerCreateTask{})
}

func (self *LoadbalancerListenerCreateTask) taskFail(ctx context.Context, lblis *models.SLoadbalancerListener, reason string) {
	lblis.SetStatus(self.GetUserCred(), api.LB_CREATE_FAILED, reason)
	db.OpsLog.LogEvent(lblis, db.ACT_ALLOCATE_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, lblis, logclient.ACT_CREATE, reason, self.UserCred, false)
	notifyclient.NotifySystemError(lblis.Id, lblis.Name, api.LB_CREATE_FAILED, reason)
	self.SetStageFailed(ctx, reason)
}

func (self *LoadbalancerListenerCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lblis := obj.(*models.SLoadbalancerListener)
	region := lblis.GetRegion()
	if region == nil {
		self.taskFail(ctx, lblis, fmt.Sprintf("failed to find region for lblis %s", lblis.Name))
		return
	}
	self.SetStage("OnLoadbalancerListenerCreateComplete", nil)
	if err := region.GetDriver().RequestCreateLoadbalancerListener(ctx, self.GetUserCred(), lblis, self); err != nil {
		self.taskFail(ctx, lblis, err.Error())
	}
}

func (self *LoadbalancerListenerCreateTask) OnPrepareLoadbalancerBackendgroup(ctx context.Context, lblis *models.SLoadbalancerListener, data jsonutils.JSONObject) {
	lblis.SetStatus(self.GetUserCred(), api.LB_STATUS_ENABLED, "")
	db.OpsLog.LogEvent(lblis, db.ACT_ALLOCATE, lblis.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, lblis, logclient.ACT_CREATE, nil, self.UserCred, true)
	self.SetStage("OnLoadbalancerListenerStartComplete", nil)
	lblis.StartLoadBalancerListenerStartTask(ctx, self.GetUserCred(), self.GetTaskId())
}

func (self *LoadbalancerListenerCreateTask) OnPrepareLoadbalancerBackendgroupFailed(ctx context.Context, lblis *models.SLoadbalancerListener, reason jsonutils.JSONObject) {
	lblis.SetStatus(self.GetUserCred(), api.LB_STATUS_DISABLED, "")
	self.taskFail(ctx, lblis, reason.String())
}

func (self *LoadbalancerListenerCreateTask) OnLoadbalancerListenerCreateComplete(ctx context.Context, lblis *models.SLoadbalancerListener, data jsonutils.JSONObject) {
	lbbg := lblis.GetLoadbalancerBackendGroup()
	// 目前只有华为才需要在创建监听器时创建服务器组,其他云直接绕过此步骤
	if lblis.GetProviderName() != api.CLOUD_PROVIDER_HUAWEI {
		self.OnPrepareLoadbalancerBackendgroup(ctx, lblis, data)
		return
	}

	if lblis.GetProviderName() == api.CLOUD_PROVIDER_HUAWEI && lbbg == nil {
		self.taskFail(ctx, lblis, "huawei loadbalancer listener releated backend group not found")
		return
	}

	// 将必要的配置复制到服务器组，以方便创建或同步服务器组
	err := updateHuaweiLbbg(lblis, lbbg, true)
	if err != nil {
		self.taskFail(ctx, lblis, err.Error())
		return
	}

	group, err := lbbg.GetBackendGroupParams()
	if err != nil {
		self.taskFail(ctx, lblis, err.Error())
		return
	}

	self.SetStage("OnPrepareLoadbalancerBackendgroup", nil)
	if len(lbbg.GetExternalId()) > 0 {
		ilbbg, err := lbbg.GetICloudLoadbalancerBackendGroup()
		if err != nil {
			self.taskFail(ctx, lblis, err.Error())
			return
		}
		// 服务器组已经存在，直接同步即可
		if err := ilbbg.Sync(&group); err != nil {
			self.taskFail(ctx, lblis, err.Error())
			return
		}
	} else {
		// 服务器组不存在
		lbbg.StartHuaweiLoadBalancerBackendGroupCreateTask(ctx, self.GetUserCred(), nil, self.GetTaskId())
	}
}

func (self *LoadbalancerListenerCreateTask) OnLoadbalancerListenerCreateCompleteFailed(ctx context.Context, lblis *models.SLoadbalancerListener, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lblis, reason.String())
}

func (self *LoadbalancerListenerCreateTask) OnLoadbalancerListenerStartComplete(ctx context.Context, lblis *models.SLoadbalancerListener, data jsonutils.JSONObject) {
	lblis.SetStatus(self.GetUserCred(), api.LB_STATUS_ENABLED, "")
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerListenerCreateTask) OnLoadbalancerListenerStartCompleteFailed(ctx context.Context, lblis *models.SLoadbalancerListener, reason jsonutils.JSONObject) {
	lblis.SetStatus(self.GetUserCred(), api.LB_STATUS_DISABLED, reason.String())
	self.SetStageFailed(ctx, reason.String())
}

func updateHuaweiLbbg(lblis *models.SLoadbalancerListener, lbbg *models.SLoadbalancerBackendGroup, withExtParams bool) error {
	_, err := lbbg.GetModelManager().TableSpec().Update(lbbg, func() error {
		if withExtParams {
			lbbg.StickySession = lblis.StickySession
			lbbg.StickySessionCookie = lblis.StickySessionCookie
			lbbg.StickySessionType = lblis.StickySessionType
			lbbg.StickySessionCookieTimeout = lblis.StickySessionCookieTimeout

			lbbg.HealthCheckType = lblis.HealthCheckType
			lbbg.HealthCheckReq = lblis.HealthCheckReq
			lbbg.HealthCheckExp = lblis.HealthCheckExp
			lbbg.HealthCheck = lblis.HealthCheck
			lbbg.HealthCheckTimeout = lblis.HealthCheckTimeout
			lbbg.HealthCheckDomain = lblis.HealthCheckDomain
			lbbg.HealthCheckHttpCode = lblis.HealthCheckHttpCode
			lbbg.HealthCheckURI = lblis.HealthCheckURI
			lbbg.HealthCheckInterval = lblis.HealthCheckInterval
			lbbg.HealthCheckRise = lblis.HealthCheckRise
			lbbg.HealthCheckFall = lblis.HealthCheckFall
		}

		lbbg.Scheduler = lblis.Scheduler
		lbbg.ProtocolType = lblis.ListenerType
		return nil
	})

	return err
}
