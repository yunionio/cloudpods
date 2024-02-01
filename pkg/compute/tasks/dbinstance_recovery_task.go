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
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DBInstanceRecoveryTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DBInstanceRecoveryTask{})
}

func (self *DBInstanceRecoveryTask) taskFailed(ctx context.Context, instance *models.SDBInstance, err error) {
	instance.SetStatus(ctx, self.UserCred, api.DBINSTANCE_RESTORE_FAILED, err.Error())
	db.OpsLog.LogEvent(instance, db.ACT_RESTORE, err, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, instance, logclient.ACT_RESTORE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *DBInstanceRecoveryTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	instance := obj.(*models.SDBInstance)

	iRds, err := instance.GetIDBInstance(ctx)
	if err != nil {
		self.taskFailed(ctx, instance, errors.Wrap(err, "instance.GetIDBInstance"))
		return
	}

	input := &api.SDBInstanceRecoveryConfigInput{}
	err = self.GetParams().Unmarshal(input)
	if err != nil {
		self.taskFailed(ctx, instance, errors.Wrap(err, "self.GetParams().Unmarshal(input)"))
		return
	}

	_backup, err := models.DBInstanceBackupManager.FetchById(input.DBInstancebackupId)
	if err != nil {
		self.taskFailed(ctx, instance, errors.Wrapf(err, "instance.GetDBInstanceBackup(%s)", input.DBInstancebackupId))
		return
	}

	backup := _backup.(*models.SDBInstanceBackup)
	origin, _ := backup.GetDBInstance()

	conf := &cloudprovider.SDBInstanceRecoveryConfig{
		BackupId:  backup.ExternalId,
		Databases: input.Databases,
	}
	if origin != nil {
		conf.OriginDBInstanceExternalId = origin.ExternalId
	}

	err = iRds.RecoveryFromBackup(conf)
	if err != nil {
		self.taskFailed(ctx, instance, errors.Wrap(err, "iRds.RecoveryFromBackup"))
		return
	}

	err = cloudprovider.WaitStatus(iRds, api.DBINSTANCE_RUNNING, time.Second*10, time.Minute*40)
	if err != nil {
		self.taskFailed(ctx, instance, errors.Wrap(err, "cloudprovider.WaitStatus(running)"))
		return
	}

	databases, err := iRds.GetIDBInstanceDatabases()
	if err != nil {
		log.Errorf("failed to get dbinstance %s database error: %v", instance.Name, err)
	} else {
		models.DBInstanceDatabaseManager.SyncDBInstanceDatabases(ctx, self.UserCred, instance, databases)
	}

	db.OpsLog.LogEvent(instance, db.ACT_RESTORE, nil, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, instance, logclient.ACT_RESTORE, backup, self.UserCred, true)
	instance.SetStatus(ctx, self.UserCred, api.DBINSTANCE_RUNNING, "")
	self.SetStageComplete(ctx, nil)
}
