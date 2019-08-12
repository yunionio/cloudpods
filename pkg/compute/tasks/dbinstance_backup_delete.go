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
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DBInstanceBackupDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DBInstanceBackupDeleteTask{})
}

func (self *DBInstanceBackupDeleteTask) taskFailed(ctx context.Context, backup *models.SDBInstanceBackup, err error) {
	backup.SetStatus(self.UserCred, api.DBINSTANCE_BACKUP_DELETE_FAILED, err.Error())
	db.OpsLog.LogEvent(backup, db.ACT_DELETE, err.Error(), self.GetUserCred())
	logclient.AddActionLogWithStartable(self, backup, logclient.ACT_DELETE, err.Error(), self.UserCred, false)
	self.SetStageFailed(ctx, err.Error())
}

func (self *DBInstanceBackupDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	backup := obj.(*models.SDBInstanceBackup)
	self.DeleteDBInstanceBackup(ctx, backup)
}

func (self *DBInstanceBackupDeleteTask) DeleteDBInstanceBackup(ctx context.Context, backup *models.SDBInstanceBackup) {
	iRegion, err := backup.GetIRegion()
	if err != nil {
		self.taskFailed(ctx, backup, errors.Wrap(err, "backup.GetIRegion"))
		return
	}

	iBackup, err := iRegion.GetIDBInstanceBackupById(backup.ExternalId)
	if err != nil && err != cloudprovider.ErrNotFound {
		self.taskFailed(ctx, backup, errors.Wrap(err, "iRegion.GetIDBInstanceBackupById"))
		return
	}

	if iBackup != nil {
		err = iBackup.Delete()
		if err != nil {
			self.taskFailed(ctx, backup, errors.Wrap(err, "iBackup.Delete()"))
			return
		}
	}

	err = backup.RealDelete(ctx, self.UserCred)
	if err != nil {
		self.taskFailed(ctx, backup, errors.Wrap(err, "backup.Purge"))
		return
	}

	self.SetStageComplete(ctx, nil)
}
