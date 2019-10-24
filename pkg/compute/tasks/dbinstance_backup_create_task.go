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

type DBInstanceBackupCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DBInstanceBackupCreateTask{})
}

func (self *DBInstanceBackupCreateTask) taskFailed(ctx context.Context, backup *models.SDBInstanceBackup, err error) {
	backup.SetStatus(self.UserCred, api.DBINSTANCE_BACKUP_CREATE_FAILED, err.Error())
	db.OpsLog.LogEvent(backup, db.ACT_CREATE, err.Error(), self.GetUserCred())
	logclient.AddActionLogWithStartable(self, backup, logclient.ACT_CREATE, err.Error(), self.UserCred, false)
	self.SetStageFailed(ctx, err.Error())
}

func (self *DBInstanceBackupCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	backup := obj.(*models.SDBInstanceBackup)
	self.CreateDBInstanceBackup(ctx, backup)
}

func (self *DBInstanceBackupCreateTask) CreateDBInstanceBackup(ctx context.Context, backup *models.SDBInstanceBackup) {
	instance, err := backup.GetDBInstance()
	if err != nil {
		self.taskFailed(ctx, backup, errors.Wrap(err, "backup.GetDBInstance"))
		return
	}

	self.SetStage("OnCreateDBInstanceBackupComplete", nil)
	instance.GetRegion().GetDriver().RequestCreateDBInstanceBackup(ctx, self.UserCred, instance, backup, self)
}

func (self *DBInstanceBackupCreateTask) OnCreateDBInstanceBackupComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	backup := obj.(*models.SDBInstanceBackup)
	logclient.AddActionLogWithStartable(self, backup, logclient.ACT_CREATE, nil, self.UserCred, true)

	backup.SetStatus(self.UserCred, api.DBINSTANCE_BACKUP_READY, "")
	self.SetStageComplete(ctx, nil)
}

func (self *DBInstanceBackupCreateTask) OnCreateDBInstanceBackupCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	backup := obj.(*models.SDBInstanceBackup)
	self.taskFailed(ctx, backup, fmt.Errorf("%s", data.String()))
}
