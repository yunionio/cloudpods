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
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type LoadbalancerSyncstatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerSyncstatusTask{})
}

func (self *LoadbalancerSyncstatusTask) taskFail(ctx context.Context, lb *models.SLoadbalancer, err error) {
	lb.SetStatus(ctx, self.GetUserCred(), api.LB_STATUS_UNKNOWN, err.Error())
	db.OpsLog.LogEvent(lb, db.ACT_SYNC_STATUS, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, lb, logclient.ACT_SYNC_STATUS, err, self.UserCred, false)
	notifyclient.NotifySystemErrorWithCtx(ctx, lb.Id, lb.Name, api.LB_SYNC_CONF_FAILED, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *LoadbalancerSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lb := obj.(*models.SLoadbalancer)
	region, err := lb.GetRegion()
	if err != nil {
		self.taskFail(ctx, lb, errors.Wrapf(err, "lb.GetRegion"))
		return
	}
	self.SetStage("OnLoadbalancerSyncstatusComplete", nil)
	err = region.GetDriver().RequestSyncstatusLoadbalancer(ctx, self.GetUserCred(), lb, self)
	if err != nil {
		self.taskFail(ctx, lb, errors.Wrapf(err, "RequestSyncstatusLoadbalancer"))
		return
	}
}

func (self *LoadbalancerSyncstatusTask) OnLoadbalancerSyncstatusComplete(ctx context.Context, lb *models.SLoadbalancer, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(lb, db.ACT_SYNC_STATUS, lb.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, lb, logclient.ACT_SYNC_STATUS, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerSyncstatusTask) OnLoadbalancerSyncstatusCompleteFailed(ctx context.Context, lb *models.SLoadbalancer, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lb, errors.Errorf(reason.String()))
}
