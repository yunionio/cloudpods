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

type ElasticcacheSetMaintainTimeTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ElasticcacheSetMaintainTimeTask{})
}

func (self *ElasticcacheSetMaintainTimeTask) taskFail(ctx context.Context, elasticcache *models.SElasticcache, reason string) {
	elasticcache.SetStatus(self.GetUserCred(), api.ELASTIC_CACHE_STATUS_CHANGE_FAILED, reason)
	db.OpsLog.LogEvent(elasticcache, db.ACT_UPDATE, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, elasticcache, logclient.ACT_UPDATE, reason, self.UserCred, false)
	notifyclient.NotifySystemError(elasticcache.Id, elasticcache.Name, api.ELASTIC_CACHE_STATUS_CHANGE_FAILED, reason)
	self.SetStageFailed(ctx, reason)
}

func (self *ElasticcacheSetMaintainTimeTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	elasticcache := obj.(*models.SElasticcache)
	region := elasticcache.GetRegion()
	if region == nil {
		self.taskFail(ctx, elasticcache, fmt.Sprintf("failed to find region for elastic cache %s", elasticcache.GetName()))
		return
	}

	self.SetStage("OnElasticcacheSetMaintainTimeComplete", nil)
	if err := region.GetDriver().RequestElasticcacheSetMaintainTime(ctx, self.GetUserCred(), elasticcache, self); err != nil {
		self.OnElasticcacheSetMaintainTimeCompleteFailed(ctx, elasticcache, err.Error())
		return
	}

	self.OnElasticcacheSetMaintainTimeComplete(ctx, elasticcache, data)
	return
}

func (self *ElasticcacheSetMaintainTimeTask) OnElasticcacheSetMaintainTimeComplete(ctx context.Context, elasticcache *models.SElasticcache, data jsonutils.JSONObject) {
	elasticcache.SetStatus(self.GetUserCred(), api.ELASTIC_CACHE_STATUS_RUNNING, "")
	logclient.AddActionLogWithStartable(self, elasticcache, logclient.ACT_UPDATE, "", self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *ElasticcacheSetMaintainTimeTask) OnElasticcacheSetMaintainTimeCompleteFailed(ctx context.Context, elasticcache *models.SElasticcache, reason string) {
	self.taskFail(ctx, elasticcache, reason)
}
