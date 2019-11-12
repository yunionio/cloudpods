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

type DBInstanceAccountCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DBInstanceAccountCreateTask{})
}

func (self *DBInstanceAccountCreateTask) taskFailed(ctx context.Context, account *models.SDBInstanceAccount, err error) {
	account.SetStatus(self.UserCred, api.DBINSTANCE_USER_CREATE_FAILED, err.Error())
	db.OpsLog.LogEvent(account, db.ACT_CREATE, err.Error(), self.GetUserCred())
	logclient.AddActionLogWithStartable(self, account, logclient.ACT_CREATE, err.Error(), self.UserCred, false)
	self.SetStageFailed(ctx, err.Error())
}

func (self *DBInstanceAccountCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	account := obj.(*models.SDBInstanceAccount)
	self.CreateDBInstanceAccount(ctx, account)
}

func (self *DBInstanceAccountCreateTask) CreateDBInstanceAccount(ctx context.Context, account *models.SDBInstanceAccount) {
	instance, err := account.GetDBInstance()
	if err != nil {
		self.taskFailed(ctx, account, errors.Wrap(err, "account.GetDBInstance"))
		return
	}

	iRds, err := instance.GetIDBInstance()
	if err != nil {
		self.taskFailed(ctx, account, errors.Wrap(err, "instance.GetIDBInstance"))
		return
	}

	desc := &cloudprovider.SDBInstanceAccountCreateConfig{
		Name: account.Name,
	}
	desc.Password, _ = account.GetPassword()

	err = iRds.CreateAccount(desc)
	if err != nil {
		self.taskFailed(ctx, account, errors.Wrap(err, "iRds.CreateAccount"))
		return
	}

	input := api.SDBInstanceAccountCreateInput{}
	self.GetParams().Unmarshal(&input)
	if len(input.Privileges) == 0 {
		account.SetStatus(self.UserCred, api.DBINSTANCE_USER_AVAILABLE, "")
		self.SetStageComplete(ctx, nil)
		return
	}

	iAccounts, err := iRds.GetIDBInstanceAccounts()
	if err != nil {
		msg := fmt.Sprintf("failed to found accounts from cloud dbinstance error: %v", err)
		db.OpsLog.LogEvent(account, db.ACT_GRANT_PRIVILEGE, msg, self.GetUserCred())
		logclient.AddActionLogWithStartable(self, account, logclient.ACT_GRANT_PRIVILEGE, msg, self.UserCred, false)
		account.SetStatus(self.UserCred, api.DBINSTANCE_USER_AVAILABLE, "")
		self.SetStageComplete(ctx, nil)
		return
	}

	var iAccount cloudprovider.ICloudDBInstanceAccount = nil
	for i := range iAccounts {
		if iAccounts[i].GetName() == account.Name {
			iAccount = iAccounts[i]
			break
		}
	}

	if iAccount == nil {
		msg := fmt.Sprintf("failed to found account %s from cloud dbinstance", account.Name)
		db.OpsLog.LogEvent(account, db.ACT_GRANT_PRIVILEGE, msg, self.GetUserCred())
		logclient.AddActionLogWithStartable(self, account, logclient.ACT_GRANT_PRIVILEGE, msg, self.UserCred, false)
		account.SetStatus(self.UserCred, api.DBINSTANCE_USER_AVAILABLE, "")
		self.SetStageComplete(ctx, nil)
		return
	}

	account.SetStatus(self.UserCred, api.DBINSTANCE_USER_GRANT_PRIVILEGE, "")
	for _, privilege := range input.Privileges {
		err = iAccount.GrantPrivilege(privilege.Database, privilege.Privilege)
		if err != nil {
			db.OpsLog.LogEvent(account, db.ACT_GRANT_PRIVILEGE, err, self.GetUserCred())
			logclient.AddActionLogWithStartable(self, account, logclient.ACT_GRANT_PRIVILEGE, err, self.UserCred, false)
			continue
		}
		_privilege := models.SDBInstancePrivilege{
			Privilege:            privilege.Privilege,
			DBInstanceaccountId:  account.Id,
			DBInstancedatabaseId: privilege.DBInstancedatabaseId,
		}
		models.DBInstancePrivilegeManager.TableSpec().Insert(&_privilege)
		logclient.AddActionLogWithStartable(self, account, logclient.ACT_GRANT_PRIVILEGE, privilege, self.UserCred, true)
	}

	account.SetStatus(self.UserCred, api.DBINSTANCE_USER_AVAILABLE, "")
	self.SetStageComplete(ctx, nil)
}
