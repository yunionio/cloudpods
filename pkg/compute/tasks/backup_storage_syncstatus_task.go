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

type BackupStorageSyncstatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(BackupStorageSyncstatusTask{})
}

func (self *BackupStorageSyncstatusTask) taskFailed(ctx context.Context, bs *models.SBackupStorage, err jsonutils.JSONObject) {
	logclient.AddActionLogWithContext(ctx, bs, logclient.ACT_SYNC_STATUS, err, self.UserCred, false)
	bs.SetStatus(ctx, self.UserCred, api.BACKUPSTORAGE_STATUS_OFFLINE, err.String())
	self.SetStageFailed(ctx, err)
}

func (self *BackupStorageSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	bs := obj.(*models.SBackupStorage)

	self.SetStage("OnBackupStorageSyncStatus", nil)
	err := bs.GetRegionDriver().RequestSyncBackupStorageStatus(ctx, self.GetUserCred(), bs, self)
	if err != nil {
		self.taskFailed(ctx, bs, jsonutils.NewString(err.Error()))
		return
	}
}

func (self *BackupStorageSyncstatusTask) OnBackupStorageSyncStatus(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *BackupStorageSyncstatusTask) OnBackupStorageSyncStatusFailed(ctx context.Context, backup *models.SBackupStorage, data jsonutils.JSONObject) {
	self.taskFailed(ctx, backup, data)
}
