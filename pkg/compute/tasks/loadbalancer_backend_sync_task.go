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

type LoadbalancerBackendSyncTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerBackendSyncTask{})
}

func (self *LoadbalancerBackendSyncTask) taskFail(ctx context.Context, lbb *models.SLoadbalancerBackend, reason jsonutils.JSONObject) {
	lbb.SetStatus(ctx, self.GetUserCred(), api.LB_SYNC_CONF_FAILED, reason.String())
	db.OpsLog.LogEvent(lbb, db.ACT_SYNC_CONF, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, lbb, logclient.ACT_SYNC_CONF, reason, self.UserCred, false)
	notifyclient.NotifySystemErrorWithCtx(ctx, lbb.Id, lbb.Name, api.LB_SYNC_CONF_FAILED, reason.String())
	lbbg, _ := lbb.GetLoadbalancerBackendGroup()
	if lbbg != nil {
		logclient.AddActionLogWithStartable(self, lbbg, logclient.ACL_LB_SYNC_BACKEND_CONF, reason, self.UserCred, false)
	}
	self.SetStageFailed(ctx, reason)
}

func (self *LoadbalancerBackendSyncTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lbb := obj.(*models.SLoadbalancerBackend)
	region, err := lbb.GetRegion()
	if err != nil {
		self.taskFail(ctx, lbb, jsonutils.NewString(err.Error()))
		return
	}
	self.SetStage("OnLoadbalancerBackendCreateComplete", nil)
	if err := region.GetDriver().RequestSyncLoadbalancerBackend(ctx, self.GetUserCred(), lbb, self); err != nil {
		self.taskFail(ctx, lbb, jsonutils.NewString(err.Error()))
	}
}

func (self *LoadbalancerBackendSyncTask) OnLoadbalancerBackendCreateComplete(ctx context.Context, lbb *models.SLoadbalancerBackend, data jsonutils.JSONObject) {
	lbb.SetStatus(ctx, self.GetUserCred(), api.LB_STATUS_ENABLED, "")
	db.OpsLog.LogEvent(lbb, db.ACT_SYNC_CONF, lbb.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, lbb, logclient.ACT_SYNC_CONF, nil, self.UserCred, true)
	lbbg, _ := lbb.GetLoadbalancerBackendGroup()
	if lbbg != nil {
		logclient.AddActionLogWithStartable(self, lbbg, logclient.ACL_LB_SYNC_BACKEND_CONF, nil, self.UserCred, true)
	}
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerBackendSyncTask) OnLoadbalancerBackendCreateCompleteFailed(ctx context.Context, lbb *models.SLoadbalancerBackend, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lbb, reason)
}
