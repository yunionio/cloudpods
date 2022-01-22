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

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DiskBackupSyncstatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DiskBackupSyncstatusTask{})
}

func (self *DiskBackupSyncstatusTask) taskFailed(ctx context.Context, backup *models.SDiskBackup, err jsonutils.JSONObject) {
	logclient.AddActionLogWithContext(ctx, backup, logclient.ACT_SYNC_STATUS, err, self.UserCred, false)
	self.SetStageFailed(ctx, err)
}

func (self *DiskBackupSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	backup := obj.(*models.SDiskBackup)

	self.SetStage("OnDiskBackupSyncStatus", nil)
	rd, err := backup.GetRegionDriver()
	if err != nil {
		self.taskFailed(ctx, backup, jsonutils.NewString(err.Error()))
		return
	}
	err = rd.RequestSyncDiskBackupStatus(ctx, self.GetUserCred(), backup, self)
	if err != nil {
		self.taskFailed(ctx, backup, jsonutils.NewString(err.Error()))
		return
	}
}

func (self *DiskBackupSyncstatusTask) OnDiskBackupSyncStatus(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *DiskBackupSyncstatusTask) OnDiskBackupSyncStatusFailed(ctx context.Context, backup *models.SDiskBackup, data jsonutils.JSONObject) {
	self.taskFailed(ctx, backup, data)
}
