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
	"yunion.io/x/pkg/util/billing"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type ElasticcacheRenewTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ElasticcacheRenewTask{})
}

func (self *ElasticcacheRenewTask) taskFail(ctx context.Context, cache *models.SElasticcache, err error) {
	db.OpsLog.LogEvent(cache, db.ACT_REW_FAIL, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, cache, logclient.ACT_RENEW, err, self.UserCred, false)
	cache.SetStatus(ctx, self.GetUserCred(), api.ELASTIC_CACHE_RENEW_FAILED, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *ElasticcacheRenewTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	instance := obj.(*models.SElasticcache)

	durationStr, _ := self.GetParams().GetString("duration")
	bc, _ := billing.ParseBillingCycle(durationStr)

	region, err := instance.GetRegion()
	if err != nil {
		self.taskFail(ctx, instance, err)
		return
	}
	err = region.GetDriver().RequestRenewElasticcache(ctx, self.UserCred, instance, bc)
	if err != nil {
		self.taskFail(ctx, instance, err)
		return
	}

	logclient.AddActionLogWithStartable(self, instance, logclient.ACT_RENEW, nil, self.UserCred, true)
	self.SetStage("OnElasticcacheSyncstatus", nil)
	models.StartResourceSyncStatusTask(ctx, self.UserCred, instance, "ElasticcacheSyncstatusTask", self.GetTaskId())
}

func (self *ElasticcacheRenewTask) OnElasticcacheSyncstatusComplete(ctx context.Context, ec *models.SElasticcache, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *ElasticcacheRenewTask) OnElasticcacheSyncstatusCompleteFailed(ctx context.Context, ec *models.SElasticcache, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}
