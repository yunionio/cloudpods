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

type DBInstanceDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DBInstanceDeleteTask{})
}

func (self *DBInstanceDeleteTask) taskFailed(ctx context.Context, dbinstance *models.SDBInstance, err error) {
	dbinstance.SetStatus(self.UserCred, api.DBINSTANCE_DELETE_FAILED, err.Error())
	db.OpsLog.LogEvent(dbinstance, db.ACT_DELETE, err.Error(), self.GetUserCred())
	logclient.AddActionLogWithStartable(self, dbinstance, logclient.ACT_DELETE, err.Error(), self.UserCred, false)
	self.SetStageFailed(ctx, err.Error())
}

func (self *DBInstanceDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	dbinstance := obj.(*models.SDBInstance)
	self.DeleteDBInstance(ctx, dbinstance)
}

func (self *DBInstanceDeleteTask) DeleteDBInstance(ctx context.Context, dbinstance *models.SDBInstance) {
	idbinstance, err := dbinstance.GetIDBInstance()
	if err != nil {
		if err == cloudprovider.ErrNotFound {
			self.DeleteDBInstanceComplete(ctx, dbinstance)
			return
		}
		self.taskFailed(ctx, dbinstance, err)
		return
	}

	if !jsonutils.QueryBoolean(self.Params, "purge", false) {
		err = idbinstance.Delete()
		if err != nil {
			self.taskFailed(ctx, dbinstance, err)
			return
		}
	}

	self.DeleteDBInstanceComplete(ctx, dbinstance)
}

func (self *DBInstanceDeleteTask) DeleteDBInstanceComplete(ctx context.Context, dbinstance *models.SDBInstance) {
	if !dbinstance.GetRegion().GetDriver().IsSupportKeepDBInstanceManualBackup() || jsonutils.QueryBoolean(self.Params, "purge", false) {
		err := dbinstance.PurgeBackups(ctx, self.UserCred, api.BACKUP_MODE_MANUAL)
		if err != nil {
			self.taskFailed(ctx, dbinstance, errors.Wrap(err, "dbinstance.PurgeManualBackups"))
			return
		}
		err = dbinstance.Purge(ctx, self.UserCred)
		if err != nil {
			self.taskFailed(ctx, dbinstance, errors.Wrap(err, "dbinstance.Purge"))
			return
		}
		self.SetStageComplete(ctx, nil)
		return
	}

	self.DeleteBackups(ctx, dbinstance, nil)
}

func (self *DBInstanceDeleteTask) DeleteBackups(ctx context.Context, instance *models.SDBInstance, data jsonutils.JSONObject) {
	if !jsonutils.QueryBoolean(self.Params, "keep_backup", false) {
		backups, _ := instance.GetDBInstanceBackupByMode(api.BACKUP_MODE_MANUAL)
		for i := range backups {
			backups[i].StartDBInstanceBackupDeleteTask(ctx, self.UserCred, "")
		}
	}
	err := instance.Purge(ctx, self.UserCred)
	if err != nil {
		self.taskFailed(ctx, instance, errors.Wrap(err, "instance.Purge"))
		return
	}
	self.SetStageComplete(ctx, nil)
}
