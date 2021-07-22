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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type ElasticcacheSetAutoRenewTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ElasticcacheSetAutoRenewTask{})
}

func (self *ElasticcacheSetAutoRenewTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	ec := obj.(*models.SElasticcache)

	autoRenew, _ := self.GetParams().Bool("auto_renew")
	err := ec.GetRegion().GetDriver().RequestElasticcacheSetAutoRenew(ctx, self.UserCred, ec, autoRenew, self)
	if err != nil {
		db.OpsLog.LogEvent(ec, db.ACT_SET_AUTO_RENEW_FAIL, err, self.UserCred)
		logclient.AddActionLogWithStartable(self, ec, logclient.ACT_SET_AUTO_RENEW, err, self.UserCred, false)
		ec.SetStatus(self.GetUserCred(), api.ELASTIC_CACHE_SET_AUTO_RENEW_FAILED, err.Error())
		self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
		return
	}

	logclient.AddActionLogWithStartable(self, ec, logclient.ACT_SET_AUTO_RENEW, nil, self.UserCred, true)
	self.SetStage("OnElasticcacheSyncstatus", nil)
	models.StartResourceSyncStatusTask(ctx, self.UserCred, ec, "ElasticcacheSyncstatusTask", self.GetTaskId())
}

func (self *ElasticcacheSetAutoRenewTask) OnElasticcacheSyncstatusComplete(ctx context.Context, ec *models.SElasticcache, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *ElasticcacheSetAutoRenewTask) OnElasticcacheSyncstatusCompleteFailed(ctx context.Context, ec *models.SElasticcache, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}
