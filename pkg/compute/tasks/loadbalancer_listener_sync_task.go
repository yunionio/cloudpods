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

type LoadbalancerListenerSyncTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerListenerSyncTask{})
}

func (self *LoadbalancerListenerSyncTask) taskFail(ctx context.Context, lblis *models.SLoadbalancerListener, err error) {
	lblis.SetStatus(ctx, self.GetUserCred(), api.LB_SYNC_CONF_FAILED, err.Error())
	db.OpsLog.LogEvent(lblis, db.ACT_SYNC_CONF, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, lblis, logclient.ACT_SYNC_CONF, err, self.UserCred, false)
	notifyclient.NotifySystemErrorWithCtx(ctx, lblis.Id, lblis.Name, api.LB_SYNC_CONF_FAILED, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *LoadbalancerListenerSyncTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lblis := obj.(*models.SLoadbalancerListener)
	region, err := lblis.GetRegion()
	if err != nil {
		self.taskFail(ctx, lblis, errors.Wrapf(err, "GetRegion"))
		return
	}

	input := &api.LoadbalancerListenerUpdateInput{}
	self.GetParams().Unmarshal(input)

	self.SetStage("OnLoadbalancerListenerSyncComplete", nil)
	err = region.GetDriver().RequestSyncLoadbalancerListener(ctx, self.GetUserCred(), lblis, input, self)
	if err != nil {
		self.taskFail(ctx, lblis, errors.Wrapf(err, "RequestSyncLoadbalancerListener"))
	}
}

func (self *LoadbalancerListenerSyncTask) OnLoadbalancerListenerSyncComplete(ctx context.Context, lblis *models.SLoadbalancerListener, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(lblis, db.ACT_SYNC_CONF, lblis.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, lblis, logclient.ACT_SYNC_CONF, nil, self.UserCred, true)
	self.SetStage("OnLoadbalancerListenerSyncStatusComplete", nil)
	lblis.StartLoadBalancerListenerSyncstatusTask(ctx, self.GetUserCred(), self.GetParams(), self.GetTaskId())
}

func (self *LoadbalancerListenerSyncTask) OnLoadbalancerListenerSyncCompleteFailed(ctx context.Context, lblis *models.SLoadbalancerListener, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lblis, errors.Errorf(reason.String()))
}

func (self *LoadbalancerListenerSyncTask) OnLoadbalancerListenerSyncStatusComplete(ctx context.Context, lblis *models.SLoadbalancerListener, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerListenerSyncTask) OnLoadbalancerListenerSyncStatusCompleteFailed(ctx context.Context, lblis *models.SLoadbalancerListener, reason jsonutils.JSONObject) {
	lblis.SetStatus(ctx, self.GetUserCred(), api.LB_STATUS_UNKNOWN, reason.String())
	self.SetStageFailed(ctx, reason)
}
