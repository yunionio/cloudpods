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
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DBInstanceDatabaseCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DBInstanceDatabaseCreateTask{})
}

func (self *DBInstanceDatabaseCreateTask) taskFailed(ctx context.Context, database *models.SDBInstanceDatabase, err error) {
	database.SetStatus(self.UserCred, api.DBINSTANCE_DATABASE_CREATE_FAILE, err.Error())
	db.OpsLog.LogEvent(database, db.ACT_CREATE, err.Error(), self.GetUserCred())
	logclient.AddActionLogWithStartable(self, database, logclient.ACT_CREATE, err.Error(), self.UserCred, false)
	self.SetStageFailed(ctx, err.Error())
}

func (self *DBInstanceDatabaseCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	database := obj.(*models.SDBInstanceDatabase)
	self.CreateDBInstanceDatabase(ctx, database)
}

func (self *DBInstanceDatabaseCreateTask) CreateDBInstanceDatabase(ctx context.Context, database *models.SDBInstanceDatabase) {
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

	desc := &cloudprovider.SDBInstanceDatabaseCreateConfig{
		CharacterSet: database.CharacterSet,
		Name:         database.Name,
		Description:  database.Description,
	}

	err = iRds.CreateDatabase(desc)
	if err != nil {
		self.taskFailed(ctx, database, errors.Wrap(err, "iRds.CreateDatabase"))
		return
	}

	input := api.SDBInstanceDatabaseCreateInput{}
	self.GetParams().Unmarshal(&input)
	if len(input.Accounts) == 0 {
		database.SetStatus(self.UserCred, api.DBINSTANCE_DATABASE_RUNNING, "")
		self.SetStageComplete(ctx, nil)
		return
	}

	_iAccounts, err := iRds.GetIDBInstanceAccounts()
	if err != nil {
		msg := fmt.Sprintf("failed to found accounts from cloud dbinstance error: %v", err)
		db.OpsLog.LogEvent(database, db.ACT_GRANT_PRIVILEGE, msg, self.GetUserCred())
		logclient.AddActionLogWithStartable(self, database, logclient.ACT_GRANT_PRIVILEGE, msg, self.UserCred, false)
		database.SetStatus(self.UserCred, api.DBINSTANCE_DATABASE_RUNNING, "")
		self.SetStageComplete(ctx, nil)
		return
	}

	database.SetStatus(self.UserCred, api.DBINSTANCE_DATABASE_GRANT_PRIVILEGE, "")
	iAccounts := map[string]cloudprovider.ICloudDBInstanceAccount{}
	for _, iAccount := range _iAccounts {
		iAccounts[iAccount.GetName()] = iAccount
	}

	for _, account := range input.Accounts {
		iAccount, ok := iAccounts[account.Account]
		if !ok {
			msg := fmt.Sprintf("failed to found account %s from dbinstance", account.Account)
			db.OpsLog.LogEvent(database, db.ACT_GRANT_PRIVILEGE, msg, self.GetUserCred())
			logclient.AddActionLogWithStartable(self, database, logclient.ACT_GRANT_PRIVILEGE, msg, self.UserCred, false)
			continue
		}
		err = iAccount.GrantPrivilege(database.Name, account.Privilege)
		if err != nil {
			db.OpsLog.LogEvent(database, db.ACT_GRANT_PRIVILEGE, err, self.GetUserCred())
			logclient.AddActionLogWithStartable(self, database, logclient.ACT_GRANT_PRIVILEGE, err, self.UserCred, false)
			continue
		}
		privilege := models.SDBInstancePrivilege{
			Privilege:            account.Privilege,
			DBInstanceaccountId:  account.DBInstanceaccountId,
			DBInstancedatabaseId: database.Id,
		}
		models.DBInstancePrivilegeManager.TableSpec().Insert(&privilege)
		logclient.AddActionLogWithStartable(self, database, logclient.ACT_GRANT_PRIVILEGE, account, self.UserCred, true)
	}
	database.SetStatus(self.UserCred, api.DBINSTANCE_DATABASE_RUNNING, "")
	self.SetStageComplete(ctx, nil)
}
