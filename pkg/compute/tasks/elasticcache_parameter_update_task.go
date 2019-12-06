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

type ElasticcacheParameterUpdateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ElasticcacheParameterUpdateTask{})
}

func (self *ElasticcacheParameterUpdateTask) taskFail(ctx context.Context, ep *models.SElasticcacheParameter, reason string) {
	ep.SetStatus(self.GetUserCred(), api.ELASTIC_CACHE_PARAMETER_STATUS_UPDATE_FAILED, reason)
	db.OpsLog.LogEvent(ep, db.ACT_UPDATE, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, ep, logclient.ACT_UPDATE, reason, self.UserCred, false)
	notifyclient.NotifySystemError(ep.Id, ep.Name, api.ELASTIC_CACHE_PARAMETER_STATUS_UPDATE_FAILED, reason)
	self.SetStageFailed(ctx, reason)
}

func (self *ElasticcacheParameterUpdateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	ep := obj.(*models.SElasticcacheParameter)
	region := ep.GetRegion()
	if region == nil {
		self.taskFail(ctx, ep, fmt.Sprintf("failed to find region for elastic cache parameter %s", ep.GetName()))
		return
	}

	iec, err := db.FetchById(models.ElasticcacheManager, ep.ElasticcacheId)
	if err != nil {
		self.taskFail(ctx, ep, fmt.Sprintf("failed to find elastic instance for  parameter %s", ep.GetName()))
		return
	}

	self.SetStage("OnElasticcacheParameterUpdateComplete", nil)
	if err := region.GetDriver().RequestElasticcacheUpdateInstanceParameters(ctx, self.GetUserCred(), iec.(*models.SElasticcache), self); err != nil {
		self.OnElasticcacheParameterUpdateCompleteFailed(ctx, ep, err.Error())
		return
	}

	self.OnElasticcacheParameterUpdateComplete(ctx, ep, data)
}

func (self *ElasticcacheParameterUpdateTask) OnElasticcacheParameterUpdateComplete(ctx context.Context, ep *models.SElasticcacheParameter, data jsonutils.JSONObject) {
	logclient.AddActionLogWithStartable(self, ep, logclient.ACT_UPDATE, "", self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *ElasticcacheParameterUpdateTask) OnElasticcacheParameterUpdateCompleteFailed(ctx context.Context, ep *models.SElasticcacheParameter, reason string) {
	self.taskFail(ctx, ep, reason)
}
