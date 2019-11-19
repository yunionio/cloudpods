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

type ElasticcacheBackupCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ElasticcacheBackupCreateTask{})
}

func (self *ElasticcacheBackupCreateTask) taskFail(ctx context.Context, elasticcache *models.SElasticcacheBackup, reason string) {
	elasticcache.SetStatus(self.GetUserCred(), api.ELASTIC_CACHE_STATUS_CREATE_FAILED, reason)
	db.OpsLog.LogEvent(elasticcache, db.ACT_ALLOCATE_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, elasticcache, logclient.ACT_CREATE, reason, self.UserCred, false)
	notifyclient.NotifySystemError(elasticcache.Id, elasticcache.Name, api.ELASTIC_CACHE_STATUS_CREATE_FAILED, reason)
	self.SetStageFailed(ctx, reason)
}

func (self *ElasticcacheBackupCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	eb := obj.(*models.SElasticcacheBackup)
	region := eb.GetRegion()
	if region == nil {
		self.taskFail(ctx, eb, fmt.Sprintf("failed to find region for elastic cache backup %s", eb.GetName()))
		return
	}

	self.SetStage("OnElasticcacheBackupCreateComplete", nil)
	if err := region.GetDriver().RequestCreateElasticcacheBackup(ctx, self.GetUserCred(), eb, self); err != nil {
		self.taskFail(ctx, eb, err.Error())
		return
	}
}

func (self *ElasticcacheBackupCreateTask) OnElasticcacheBackupCreateComplete(ctx context.Context, eb *models.SElasticcacheBackup, data jsonutils.JSONObject) {
	logclient.AddActionLogWithStartable(self, eb, logclient.ACT_CREATE, "", self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *ElasticcacheBackupCreateTask) OnElasticcacheBackupCreateCompleteFailed(ctx context.Context, eb *models.SElasticcacheBackup, reason string) {
	self.taskFail(ctx, eb, reason)
}
