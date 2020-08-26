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

type LoadbalancerCertificateCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerCertificateCreateTask{})
}

func (self *LoadbalancerCertificateCreateTask) taskFail(ctx context.Context, lbcert *models.SCachedLoadbalancerCertificate, reason jsonutils.JSONObject) {
	lbcert.SetStatus(self.GetUserCred(), api.LB_CREATE_FAILED, reason.String())
	db.OpsLog.LogEvent(lbcert, db.ACT_ALLOCATE_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, lbcert, logclient.ACT_CREATE, reason, self.UserCred, false)
	notifyclient.NotifySystemError(lbcert.Id, lbcert.Name, api.LB_CREATE_FAILED, reason.String())
	self.SetStageFailed(ctx, reason)
}

func (self *LoadbalancerCertificateCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lbcert := obj.(*models.SCachedLoadbalancerCertificate)
	region := lbcert.GetRegion()
	if region == nil {
		self.taskFail(ctx, lbcert, jsonutils.NewString(fmt.Sprintf("failed to find region for lbcert %s", lbcert.Name)))
		return
	}
	self.SetStage("OnLoadbalancerCertificateCreateComplete", nil)
	if err := region.GetDriver().RequestCreateLoadbalancerCertificate(ctx, self.GetUserCred(), lbcert, self); err != nil {
		self.taskFail(ctx, lbcert, jsonutils.Marshal(err))
	}
}

func (self *LoadbalancerCertificateCreateTask) OnLoadbalancerCertificateCreateComplete(ctx context.Context, lbcert *models.SCachedLoadbalancerCertificate, data jsonutils.JSONObject) {
	lbcert.SetStatus(self.GetUserCred(), api.LB_STATUS_ENABLED, "")
	db.OpsLog.LogEvent(lbcert, db.ACT_ALLOCATE, lbcert.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, lbcert, logclient.ACT_CREATE, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerCertificateCreateTask) OnLoadbalancerCertificateCreateCompleteFailed(ctx context.Context, lbcert *models.SCachedLoadbalancerCertificate, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lbcert, reason)
}
