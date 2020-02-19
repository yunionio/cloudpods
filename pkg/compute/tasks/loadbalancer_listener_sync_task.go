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

type LoadbalancerListenerSyncTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerListenerSyncTask{})
}

func (self *LoadbalancerListenerSyncTask) taskFail(ctx context.Context, lblis *models.SLoadbalancerListener, reason string) {
	lblis.SetStatus(self.GetUserCred(), api.LB_SYNC_CONF_FAILED, reason)
	db.OpsLog.LogEvent(lblis, db.ACT_SYNC_CONF, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, lblis, logclient.ACT_SYNC_CONF, reason, self.UserCred, false)
	notifyclient.NotifySystemError(lblis.Id, lblis.Name, api.LB_SYNC_CONF_FAILED, reason)
	self.SetStageFailed(ctx, reason)
}

func (self *LoadbalancerListenerSyncTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lblis := obj.(*models.SLoadbalancerListener)
	region := lblis.GetRegion()
	if region == nil {
		self.taskFail(ctx, lblis, fmt.Sprintf("failed to find region for lblis %s", lblis.Name))
		return
	}

	self.SetStage("OnLoadbalancerBackendgroupSyncComplete", nil)
	driver := region.GetDriver()
	userCred := self.GetUserCred()
	err := driver.RequestSyncLoadbalancerBackendGroup(ctx, userCred, lblis, self)
	if err != nil {
		self.taskFail(ctx, lblis, err.Error())
	}
}

func (self *LoadbalancerListenerSyncTask) OnLoadbalancerBackendgroupSyncComplete(ctx context.Context, lblis *models.SLoadbalancerListener, data jsonutils.JSONObject) {
	region := lblis.GetRegion()
	if region == nil {
		self.taskFail(ctx, lblis, fmt.Sprintf("failed to find region for lblis %s", lblis.Name))
		return
	}
	self.SetStage("OnLoadbalancerListenerSyncComplete", nil)
	if err := region.GetDriver().RequestSyncLoadbalancerListener(ctx, self.GetUserCred(), lblis, self); err != nil {
		self.taskFail(ctx, lblis, err.Error())
	}
}

func (self *LoadbalancerListenerSyncTask) OnLoadbalancerBackendgroupSyncCompleteFailed(ctx context.Context, lblis *models.SLoadbalancerListener, reason jsonutils.JSONObject) {
	lblis.SetStatus(self.GetUserCred(), api.LB_SYNC_CONF_FAILED, reason.String())
	self.taskFail(ctx, lblis, reason.String())
}

func (self *LoadbalancerListenerSyncTask) OnLoadbalancerListenerSyncComplete(ctx context.Context, lblis *models.SLoadbalancerListener, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(lblis, db.ACT_SYNC_CONF, lblis.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, lblis, logclient.ACT_SYNC_CONF, nil, self.UserCred, true)
	self.SetStage("OnLoadbalancerListenerSyncStatusComplete", nil)
	lblis.StartLoadBalancerListenerSyncstatusTask(ctx, self.GetUserCred(), self.GetParams(), self.GetTaskId())
}

func (self *LoadbalancerListenerSyncTask) OnLoadbalancerListenerSyncCompleteFailed(ctx context.Context, lblis *models.SLoadbalancerListener, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lblis, reason.String())
}

func (self *LoadbalancerListenerSyncTask) OnLoadbalancerListenerSyncStatusComplete(ctx context.Context, lblis *models.SLoadbalancerListener, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerListenerSyncTask) OnLoadbalancerListenerSyncStatusCompleteFailed(ctx context.Context, lblis *models.SLoadbalancerListener, reason jsonutils.JSONObject) {
	lblis.SetStatus(self.GetUserCred(), api.LB_STATUS_UNKNOWN, reason.String())
	self.SetStageFailed(ctx, reason.String())
}
