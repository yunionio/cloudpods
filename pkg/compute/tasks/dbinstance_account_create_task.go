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

	account.SetStatus(self.UserCred, api.DBINSTANCE_USER_AVAILABLE, "")

	input := api.SDBInstanceAccountCreateInput{}
	self.GetParams().Unmarshal(&input)
	for _, _privilege := range input.Privileges {
		account.StartGrantPrivilegeTask(ctx, self.UserCred, _privilege.Database, _privilege.Privilege, "")
	}

	self.SetStageComplete(ctx, nil)
}
