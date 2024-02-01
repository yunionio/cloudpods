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
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DBInstanceBackupSyncstatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DBInstanceBackupSyncstatusTask{})
}

func (self *DBInstanceBackupSyncstatusTask) taskFailed(ctx context.Context, backup *models.SDBInstanceBackup, err error) {
	backup.SetStatus(ctx, self.GetUserCred(), api.DBINSTANCE_BACKUP_UNKNOWN, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
	db.OpsLog.LogEvent(backup, db.ACT_SYNC_STATUS, backup.GetShortDesc(ctx), self.GetUserCred())
	logclient.AddActionLogWithContext(ctx, backup, logclient.ACT_SYNC_STATUS, err, self.UserCred, false)
}

func (self *DBInstanceBackupSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	backup := obj.(*models.SDBInstanceBackup)

	region, err := backup.GetRegion()
	if err != nil {
		self.taskFailed(ctx, backup, errors.Wrapf(err, "GetRegion"))
		return
	}

	self.SetStage("OnDBInstanceBackupSyncStatusComplete", nil)
	err = region.GetDriver().RequestSyncDBInstanceBackupStatus(ctx, self.GetUserCred(), backup, self)
	if err != nil {
		self.taskFailed(ctx, backup, errors.Wrap(err, "RequestSyncDBInstanceBackupStatus"))
		return
	}
}

func (self *DBInstanceBackupSyncstatusTask) OnDBInstanceBackupSyncStatusComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *DBInstanceBackupSyncstatusTask) OnDBInstanceBackupSyncStatusCompleteFailed(ctx context.Context, backup *models.SDBInstanceBackup, data jsonutils.JSONObject) {
	self.taskFailed(ctx, backup, fmt.Errorf(data.String()))
}
