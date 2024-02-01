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

type LoadbalancerLoadbalancerBackendGroupCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerLoadbalancerBackendGroupCreateTask{})
}

func (self *LoadbalancerLoadbalancerBackendGroupCreateTask) taskFail(ctx context.Context, lbacl *models.SLoadbalancerBackendGroup, err error) {
	lbacl.SetStatus(ctx, self.GetUserCred(), api.LB_CREATE_FAILED, err.Error())
	db.OpsLog.LogEvent(lbacl, db.ACT_ALLOCATE_FAIL, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, lbacl, logclient.ACT_CREATE, err, self.UserCred, false)
	notifyclient.NotifySystemErrorWithCtx(ctx, lbacl.Id, lbacl.Name, api.LB_CREATE_FAILED, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *LoadbalancerLoadbalancerBackendGroupCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lbbg := obj.(*models.SLoadbalancerBackendGroup)
	region, err := lbbg.GetRegion()
	if err != nil {
		self.taskFail(ctx, lbbg, errors.Wrapf(err, "GetRegion"))
		return
	}
	self.SetStage("OnLoadbalancerBackendGroupCreateComplete", nil)
	err = region.GetDriver().RequestCreateLoadbalancerBackendGroup(ctx, self.GetUserCred(), lbbg, self)
	if err != nil {
		self.taskFail(ctx, lbbg, errors.Wrapf(err, "RequestCreateLoadbalancerBackendGroup"))
	}
}

func (self *LoadbalancerLoadbalancerBackendGroupCreateTask) OnLoadbalancerBackendGroupCreateComplete(ctx context.Context, lbbg *models.SLoadbalancerBackendGroup, data jsonutils.JSONObject) {
	lbbg.SetStatus(ctx, self.GetUserCred(), api.LB_STATUS_ENABLED, "")
	db.OpsLog.LogEvent(lbbg, db.ACT_ALLOCATE, lbbg.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, lbbg, logclient.ACT_CREATE, nil, self.UserCred, true)
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    lbbg,
		Action: notifyclient.ActionCreate,
	})
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerLoadbalancerBackendGroupCreateTask) OnLoadbalancerBackendGroupCreateCompleteFailed(ctx context.Context, lbbg *models.SLoadbalancerBackendGroup, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lbbg, errors.Errorf(reason.String()))
}
