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

package elasticcache

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

type ElasticcacheUpdateAuthModeTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ElasticcacheUpdateAuthModeTask{})
}

func (self *ElasticcacheUpdateAuthModeTask) taskFail(ctx context.Context, elasticcache *models.SElasticcache, reason jsonutils.JSONObject) {
	elasticcache.SetStatus(ctx, self.GetUserCred(), api.ELASTIC_CACHE_STATUS_CHANGE_FAILED, reason.String())
	db.OpsLog.LogEvent(elasticcache, db.ACT_UPDATE, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, elasticcache, logclient.ACT_UPDATE, reason, self.UserCred, false)
	notifyclient.NotifySystemErrorWithCtx(ctx, elasticcache.Id, elasticcache.Name, api.ELASTIC_CACHE_STATUS_CHANGE_FAILED, reason.String())
	self.SetStageFailed(ctx, reason)
}

func (self *ElasticcacheUpdateAuthModeTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	elasticcache := obj.(*models.SElasticcache)
	region, _ := elasticcache.GetRegion()
	if region == nil {
		self.taskFail(ctx, elasticcache, jsonutils.Marshal(fmt.Sprintf("failed to find region for elastic cache %s", elasticcache.GetName())))
		return
	}

	self.SetStage("OnElasticcacheUpdateAuthModeComplete", nil)
	if err := region.GetDriver().RequestUpdateElasticcacheAuthMode(ctx, self.GetUserCred(), elasticcache, self); err != nil {
		self.OnElasticcacheUpdateAuthModeCompleteFailed(ctx, elasticcache, jsonutils.NewString(err.Error()))
		return
	}

	self.OnElasticcacheUpdateAuthModeComplete(ctx, elasticcache, data)
	return
}

func (self *ElasticcacheUpdateAuthModeTask) OnElasticcacheUpdateAuthModeComplete(ctx context.Context, elasticcache *models.SElasticcache, data jsonutils.JSONObject) {
	elasticcache.SetStatus(ctx, self.GetUserCred(), api.ELASTIC_CACHE_STATUS_RUNNING, "")
	logclient.AddActionLogWithStartable(self, elasticcache, logclient.ACT_UPDATE, "", self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *ElasticcacheUpdateAuthModeTask) OnElasticcacheUpdateAuthModeCompleteFailed(ctx context.Context, elasticcache *models.SElasticcache, reason jsonutils.JSONObject) {
	self.taskFail(ctx, elasticcache, reason)
}
