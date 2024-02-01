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

type LoadbalancerListenerSyncstatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerListenerSyncstatusTask{})
}

func (self *LoadbalancerListenerSyncstatusTask) taskFail(ctx context.Context, lblis *models.SLoadbalancerListener, err error) {
	lblis.SetStatus(ctx, self.GetUserCred(), api.LB_STATUS_UNKNOWN, err.Error())
	db.OpsLog.LogEvent(lblis, db.ACT_SYNC_STATUS, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, lblis, logclient.ACT_SYNC_STATUS, err, self.UserCred, false)
	notifyclient.NotifySystemErrorWithCtx(ctx, lblis.Id, lblis.Name, api.LB_SYNC_CONF_FAILED, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *LoadbalancerListenerSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lblis := obj.(*models.SLoadbalancerListener)
	region, err := lblis.GetRegion()
	if err != nil {
		self.taskFail(ctx, lblis, errors.Wrapf(err, "GetRegion"))
		return
	}
	self.SetStage("OnLoadbalancerListenerSyncstatusComplete", nil)
	err = region.GetDriver().RequestSyncstatusLoadbalancerListener(ctx, self.GetUserCred(), lblis, self)
	if err != nil {
		self.taskFail(ctx, lblis, errors.Wrapf(err, "RequestSyncstatusLoadbalancerListener"))
	}
}

func (self *LoadbalancerListenerSyncstatusTask) OnLoadbalancerListenerSyncstatusComplete(ctx context.Context, lblis *models.SLoadbalancerListener, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(lblis, db.ACT_SYNC_STATUS, lblis.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, lblis, logclient.ACT_SYNC_STATUS, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerListenerSyncstatusTask) OnLoadbalancerListenerSyncstatusCompleteFailed(ctx context.Context, lblis *models.SLoadbalancerListener, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lblis, errors.Errorf(reason.String()))
}
