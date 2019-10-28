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

type DBInstanceAccountGrantPrivilegeTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DBInstanceAccountGrantPrivilegeTask{})
}

func (self *DBInstanceAccountGrantPrivilegeTask) taskFailed(ctx context.Context, account *models.SDBInstanceAccount, err error) {
	account.SetStatus(self.UserCred, api.DBINSTANCE_USER_AVAILABLE, err.Error())
	db.OpsLog.LogEvent(account, db.ACT_GRANT_PRIVILEGE, err.Error(), self.GetUserCred())
	logclient.AddActionLogWithStartable(self, account, logclient.ACT_GRANT_PRIVILEGE, err.Error(), self.UserCred, false)
	self.SetStageFailed(ctx, err.Error())
}

func (self *DBInstanceAccountGrantPrivilegeTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	account := obj.(*models.SDBInstanceAccount)
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

	accounts, err := iRds.GetIDBInstanceAccounts()
	if err != nil {
		self.taskFailed(ctx, account, errors.Wrap(err, "iRds.GetIDBInstanceAccounts"))
		return
	}

	databaseStr, _ := self.GetParams().GetString("database")
	privilegeStr, _ := self.GetParams().GetString("privilege")
	database, err := instance.GetDBInstanceDatabase(databaseStr)
	if err != nil {
		self.taskFailed(ctx, account, errors.Wrapf(err, "instance.GetDBInstanceDatabase(%s)", databaseStr))
		return
	}

	var iAccount cloudprovider.ICloudDBInstanceAccount = nil
	for _, ac := range accounts {
		if ac.GetName() == account.Name {
			iAccount = ac
		}
	}
	if iAccount == nil {
		self.taskFailed(ctx, account, fmt.Errorf("failed to found iAccount by %s", account.Name))
		return
	}

	err = iAccount.GrantPrivilege(databaseStr, privilegeStr)
	if err != nil {
		self.taskFailed(ctx, account, errors.Wrap(err, "ac.GrantPrivilege"))
		return
	}

	privilege := models.SDBInstancePrivilege{
		Privilege:            privilegeStr,
		DBInstanceaccountId:  account.Id,
		DBInstancedatabaseId: database.Id,
	}

	models.DBInstancePrivilegeManager.TableSpec().Insert(&privilege)

	account.SetStatus(self.UserCred, api.DBINSTANCE_USER_AVAILABLE, "")
	logclient.AddActionLogWithStartable(self, account, logclient.ACT_GRANT_PRIVILEGE, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}
