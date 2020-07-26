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

type LoadbalancerCertificateDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerCertificateDeleteTask{})
}

func (self *LoadbalancerCertificateDeleteTask) taskFail(ctx context.Context, lbcert *models.SCachedLoadbalancerCertificate, reason jsonutils.JSONObject) {
	lbcert.SetStatus(self.GetUserCred(), api.LB_STATUS_DELETE_FAILED, reason.String())
	db.OpsLog.LogEvent(lbcert, db.ACT_DELOCATE_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, lbcert, logclient.ACT_DELOCATE, reason, self.UserCred, false)
	notifyclient.NotifySystemError(lbcert.Id, lbcert.Name, api.LB_STATUS_DELETE_FAILED, reason.String())
	self.SetStageFailed(ctx, reason)
}

func (self *LoadbalancerCertificateDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lbcert := obj.(*models.SCachedLoadbalancerCertificate)
	region := lbcert.GetRegion()
	if region == nil {
		self.taskFail(ctx, lbcert, jsonutils.NewString(fmt.Sprintf("failed to find region for lbcert %s", lbcert.Name)))
		return
	}
	self.SetStage("OnLoadbalancerCertificateDeleteComplete", nil)
	if err := region.GetDriver().RequestDeleteLoadbalancerCertificate(ctx, self.GetUserCred(), lbcert, self); err != nil {
		self.taskFail(ctx, lbcert, jsonutils.Marshal(err))
	}
}

func (self *LoadbalancerCertificateDeleteTask) OnLoadbalancerCertificateDeleteComplete(ctx context.Context, lbcert *models.SCachedLoadbalancerCertificate, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(lbcert, db.ACT_DELETE, lbcert.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, lbcert, logclient.ACT_DELOCATE, nil, self.UserCred, true)
	lbcert.DoPendingDelete(ctx, self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerCertificateDeleteTask) OnLoadbalancerCertificateDeleteCompleteFailed(ctx context.Context, lbcert *models.SCachedLoadbalancerCertificate, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lbcert, reason)
}
