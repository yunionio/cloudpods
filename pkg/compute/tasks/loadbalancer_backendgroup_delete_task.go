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

type LoadbalancerBackendGroupDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerBackendGroupDeleteTask{})
}

func (self *LoadbalancerBackendGroupDeleteTask) taskFail(ctx context.Context, lbbg *models.SLoadbalancerBackendGroup, err error) {
	lbbg.SetStatus(ctx, self.GetUserCred(), api.LB_STATUS_DELETE_FAILED, err.Error())
	db.OpsLog.LogEvent(lbbg, db.ACT_DELOCATE_FAIL, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, lbbg, logclient.ACT_DELOCATE, err, self.UserCred, false)
	notifyclient.NotifySystemErrorWithCtx(ctx, lbbg.Id, lbbg.Name, api.LB_STATUS_DELETE_FAILED, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *LoadbalancerBackendGroupDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lbbg := obj.(*models.SLoadbalancerBackendGroup)
	region, err := lbbg.GetRegion()
	if err != nil {
		self.taskFail(ctx, lbbg, errors.Wrapf(err, "GetRegion"))
		return
	}
	self.SetStage("OnLoadbalancerBackendGroupDeleteComplete", nil)
	err = region.GetDriver().RequestDeleteLoadbalancerBackendGroup(ctx, self.GetUserCred(), lbbg, self)
	if err != nil {
		self.taskFail(ctx, lbbg, errors.Wrapf(err, "RequestDeleteLoadbalancerBackendGroup"))
	}
}

func (self *LoadbalancerBackendGroupDeleteTask) OnLoadbalancerBackendGroupDeleteComplete(ctx context.Context, lbbg *models.SLoadbalancerBackendGroup, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(lbbg, db.ACT_DELETE, lbbg.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, lbbg, logclient.ACT_DELOCATE, nil, self.UserCred, true)
	lbbg.RealDelete(ctx, self.GetUserCred())
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    lbbg,
		Action: notifyclient.ActionDelete,
	})
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerBackendGroupDeleteTask) OnLoadbalancerBackendGroupDeleteCompleteFailed(ctx context.Context, lbbg *models.SLoadbalancerBackendGroup, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lbbg, errors.Errorf(reason.String()))
}
