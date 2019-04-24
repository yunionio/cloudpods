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

type LoadbalancerAclSyncTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerAclSyncTask{})
}

func (self *LoadbalancerAclSyncTask) taskFail(ctx context.Context, lbacl *models.SLoadbalancerAcl, reason string) {
	lbacl.SetStatus(self.GetUserCred(), api.LB_SYNC_CONF_FAILED, reason)
	db.OpsLog.LogEvent(lbacl, db.ACT_SYNC_CONF, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, lbacl, logclient.ACT_SYNC_CONF, reason, self.UserCred, false)
	notifyclient.NotifySystemError(lbacl.Id, lbacl.Name, api.LB_SYNC_CONF_FAILED, reason)
	self.SetStageFailed(ctx, reason)
}

func (self *LoadbalancerAclSyncTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lbacl := obj.(*models.SLoadbalancerAcl)
	region := lbacl.GetRegion()
	if region == nil {
		self.taskFail(ctx, lbacl, fmt.Sprintf("failed to find region for lbacl %s", lbacl.Name))
		return
	}
	self.SetStage("OnLoadbalancerAclSyncComplete", nil)
	if err := region.GetDriver().RequestSyncLoadbalancerAcl(ctx, self.GetUserCred(), lbacl, self); err != nil {
		self.taskFail(ctx, lbacl, err.Error())
	}
}

func (self *LoadbalancerAclSyncTask) OnLoadbalancerAclSyncComplete(ctx context.Context, lbacl *models.SLoadbalancerAcl, data jsonutils.JSONObject) {
	lbacl.SetStatus(self.GetUserCred(), api.LB_STATUS_ENABLED, "")
	db.OpsLog.LogEvent(lbacl, db.ACT_SYNC_CONF, lbacl.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, lbacl, logclient.ACT_SYNC_CONF, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerAclSyncTask) OnLoadbalancerAclSyncCompleteFailed(ctx context.Context, lbacl *models.SLoadbalancerAcl, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lbacl, reason.String())
}
