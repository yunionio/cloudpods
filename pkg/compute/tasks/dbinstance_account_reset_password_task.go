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
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DBInstanceAccountResetPasswordTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DBInstanceAccountResetPasswordTask{})
}

func (self *DBInstanceAccountResetPasswordTask) taskFailed(ctx context.Context, account *models.SDBInstanceAccount, err error) {
	account.SetStatus(ctx, self.UserCred, api.DBINSTANCE_USER_RESET_PASSWD_FAILED, err.Error())
	db.OpsLog.LogEvent(account, db.ACT_RESET_PASSWORD, err, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, account, logclient.ACT_RESET_PASSWORD, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *DBInstanceAccountResetPasswordTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	account := obj.(*models.SDBInstanceAccount)
	instance, err := account.GetDBInstance()
	if err != nil {
		self.taskFailed(ctx, account, errors.Wrap(err, "account.GetDBInstance"))
		return
	}
	iRds, err := instance.GetIDBInstance(ctx)
	if err != nil {
		self.taskFailed(ctx, account, errors.Wrap(err, "instance.GetIDBInstance"))
		return
	}

	accounts, err := iRds.GetIDBInstanceAccounts()
	if err != nil {
		self.taskFailed(ctx, account, errors.Wrap(err, "iRds.GetIDBInstanceAccounts"))
		return
	}

	password, _ := self.GetParams().GetString("password")
	for _, ac := range accounts {
		if ac.GetName() == account.Name {
			err = ac.ResetPassword(password)
			if err != nil {
				self.taskFailed(ctx, account, errors.Wrap(err, "ac.ResetPassword"))
				return
			}
			account.SetPassword(password)
		}
	}

	self.resetPasswdNotify(ctx, instance, account, password)
	account.SetStatus(ctx, self.UserCred, api.DBINSTANCE_USER_AVAILABLE, "")
	logclient.AddActionLogWithStartable(self, account, logclient.ACT_RESET_PASSWORD, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *DBInstanceAccountResetPasswordTask) resetPasswdNotify(ctx context.Context, rds *models.SDBInstance, account *models.SDBInstanceAccount, password string) {
	detailsDecro := func(ctx context.Context, details *jsonutils.JSONDict) {
		details.Set("account", jsonutils.NewString(account.GetName()))
		details.Set("password", jsonutils.NewString(password))
	}
	notifyclient.EventNotify(ctx, self.GetUserCred(), notifyclient.SEventNotifyParam{
		Obj:                 rds,
		Action:              notifyclient.ActionResetPassword,
		ObjDetailsDecorator: detailsDecro,
	})
}
