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

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type LoadbalancerAclUpdateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerAclUpdateTask{})
}

func (self *LoadbalancerAclUpdateTask) taskFail(ctx context.Context, lbacl *models.SLoadbalancerAcl, err error) {
	lbacl.SetStatus(ctx, self.GetUserCred(), api.LB_SYNC_CONF_FAILED, err.Error())
	db.OpsLog.LogEvent(lbacl, db.ACT_SYNC_CONF, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, lbacl, logclient.ACT_SYNC_CONF, err, self.UserCred, false)
	notifyclient.NotifySystemErrorWithCtx(ctx, lbacl.Id, lbacl.Name, api.LB_SYNC_CONF_FAILED, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *LoadbalancerAclUpdateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lbacl := obj.(*models.SLoadbalancerAcl)
	region, err := lbacl.GetRegion()
	if err != nil {
		self.taskFail(ctx, lbacl, errors.Wrapf(err, "GetRegion"))
		return
	}
	self.SetStage("OnLoadbalancerAclUpdateComplete", nil)
	err = region.GetDriver().RequestUpdateLoadbalancerAcl(ctx, self.GetUserCred(), lbacl, self)
	if err != nil {
		self.taskFail(ctx, lbacl, errors.Wrapf(err, "RequestUpdateLoadbalancerAcl"))
	}
}

func (self *LoadbalancerAclUpdateTask) OnLoadbalancerAclUpdateComplete(ctx context.Context, lbacl *models.SLoadbalancerAcl, data jsonutils.JSONObject) {
	lbacl.SetStatus(ctx, self.GetUserCred(), apis.STATUS_AVAILABLE, "")
	db.OpsLog.LogEvent(lbacl, db.ACT_SYNC_CONF, lbacl.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, lbacl, logclient.ACT_SYNC_CONF, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerAclUpdateTask) OnLoadbalancerAclUpdateCompleteFailed(ctx context.Context, lbacl *models.SLoadbalancerAcl, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lbacl, errors.Errorf(reason.String()))
}
