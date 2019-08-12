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
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DBInstanceDatabaseDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DBInstanceDatabaseDeleteTask{})
}

func (self *DBInstanceDatabaseDeleteTask) taskFailed(ctx context.Context, database *models.SDBInstanceDatabase, err error) {
	database.SetStatus(self.UserCred, api.DBINSTANCE_DATABASE_DELETE_FAILED, err.Error())
	db.OpsLog.LogEvent(database, db.ACT_DELETE, err.Error(), self.GetUserCred())
	logclient.AddActionLogWithStartable(self, database, logclient.ACT_DELETE, err.Error(), self.UserCred, false)
	self.SetStageFailed(ctx, err.Error())
}

func (self *DBInstanceDatabaseDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	database := obj.(*models.SDBInstanceDatabase)
	self.DeleteDBInstanceDatabase(ctx, database)
}

func (self *DBInstanceDatabaseDeleteTask) DeleteDBInstanceDatabase(ctx context.Context, database *models.SDBInstanceDatabase) {
	instance, err := database.GetDBInstance()
	if err != nil {
		self.taskFailed(ctx, database, errors.Wrap(err, "database.GetDBInstance"))
		return
	}
	iRds, err := instance.GetIDBInstance()
	if err != nil {
		self.taskFailed(ctx, database, errors.Wrap(err, "instance.GetIDBInstance"))
		return
	}

	databases, err := iRds.GetIDBInstanceDatabases()
	if err != nil {
		self.taskFailed(ctx, database, errors.Wrap(err, "iRds.GetIDBInstanceDatabases"))
		return
	}

	for _, db := range databases {
		if db.GetName() == database.Name {
			err = db.Delete()
			if err != nil {
				self.taskFailed(ctx, database, errors.Wrap(err, "db.Delete()"))
				return
			}
		}
	}

	err = database.Purge(ctx, self.UserCred)
	if err != nil {
		self.taskFailed(ctx, database, errors.Wrap(err, "database.Purge"))
		return
	}

	self.SetStageComplete(ctx, nil)
}
