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

type LoadbalancerBackendCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerBackendCreateTask{})
}

func (self *LoadbalancerBackendCreateTask) taskFail(ctx context.Context, lbb *models.SLoadbalancerBackend, err error) {
	lbb.SetStatus(ctx, self.GetUserCred(), api.LB_CREATE_FAILED, err.Error())
	db.OpsLog.LogEvent(lbb, db.ACT_ALLOCATE_FAIL, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, lbb, logclient.ACT_CREATE, err, self.UserCred, false)
	notifyclient.NotifySystemErrorWithCtx(ctx, lbb.Id, lbb.Name, api.LB_CREATE_FAILED, err.Error())
	lbbg, _ := lbb.GetLoadbalancerBackendGroup()
	if lbbg != nil {
		logclient.AddActionLogWithStartable(self, lbbg, logclient.ACT_LB_ADD_BACKEND, err, self.UserCred, false)
	}
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *LoadbalancerBackendCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lbb := obj.(*models.SLoadbalancerBackend)
	region, err := lbb.GetRegion()
	if err != nil {
		self.taskFail(ctx, lbb, errors.Wrapf(err, "lbb.GetRegion"))
		return
	}

	self.SetStage("OnLoadbalancerBackendCreateComplete", nil)
	if err := region.GetDriver().RequestCreateLoadbalancerBackend(ctx, self.GetUserCred(), lbb, self); err != nil {
		self.taskFail(ctx, lbb, errors.Wrapf(err, "RequestCreateLoadbalancerBackend"))
	}
}

func (self *LoadbalancerBackendCreateTask) OnLoadbalancerBackendCreateComplete(ctx context.Context, lbb *models.SLoadbalancerBackend, data jsonutils.JSONObject) {
	lbb.SetStatus(ctx, self.GetUserCred(), api.LB_STATUS_ENABLED, "")
	db.OpsLog.LogEvent(lbb, db.ACT_ALLOCATE, lbb.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, lbb, logclient.ACT_CREATE, nil, self.UserCred, true)
	lbbg, _ := lbb.GetLoadbalancerBackendGroup()
	if lbbg != nil {
		logclient.AddActionLogWithStartable(self, lbbg, logclient.ACT_LB_ADD_BACKEND, nil, self.UserCred, true)
	}
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    lbb,
		Action: notifyclient.ActionCreate,
	})
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerBackendCreateTask) OnLoadbalancerBackendCreateCompleteFailed(ctx context.Context, lbb *models.SLoadbalancerBackend, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lbb, errors.Errorf(reason.String()))
}
