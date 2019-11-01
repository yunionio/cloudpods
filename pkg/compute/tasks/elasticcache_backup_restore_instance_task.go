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

type ElasticcacheBackupRestoreInstanceTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ElasticcacheBackupRestoreInstanceTask{})
}

func (self *ElasticcacheBackupRestoreInstanceTask) taskFail(ctx context.Context, eb *models.SElasticcacheBackup, reason string) {
	eb.SetStatus(self.GetUserCred(), api.ELASTIC_CACHE_STATUS_CREATE_FAILED, reason)
	db.OpsLog.LogEvent(eb, db.ACT_CONVERT_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, eb, logclient.ACT_RESTORE, reason, self.UserCred, false)
	notifyclient.NotifySystemError(eb.Id, eb.Name, api.ELASTIC_CACHE_STATUS_CREATE_FAILED, reason)
	self.SetStageFailed(ctx, reason)
}

func (self *ElasticcacheBackupRestoreInstanceTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	eb := obj.(*models.SElasticcacheBackup)
	region := eb.GetRegion()
	if region == nil {
		self.taskFail(ctx, eb, fmt.Sprintf("failed to find region for elastic cache backup %s", eb.GetName()))
		return
	}

	self.SetStage("OnElasticcacheBackupRestoreInstanceComplete", nil)
	if err := region.GetDriver().RequestElasticcacheBackupRestoreInstance(ctx, self.GetUserCred(), eb, self); err != nil {
		self.OnElasticcacheBackupRestoreInstanceCompleteFailed(ctx, eb, err.Error())
		return
	}

	self.OnElasticcacheBackupRestoreInstanceComplete(ctx, eb, data)
	return
}

func (self *ElasticcacheBackupRestoreInstanceTask) OnElasticcacheBackupRestoreInstanceComplete(ctx context.Context, eb *models.SElasticcacheBackup, data jsonutils.JSONObject) {
	eb.SetStatus(self.GetUserCred(), api.ELASTIC_CACHE_STATUS_RUNNING, "")
	self.SetStageComplete(ctx, nil)
}

func (self *ElasticcacheBackupRestoreInstanceTask) OnElasticcacheBackupRestoreInstanceCompleteFailed(ctx context.Context, eb *models.SElasticcacheBackup, reason string) {
	self.taskFail(ctx, eb, reason)
}
