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
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DBInstanceCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DBInstanceCreateTask{})
}

func (self *DBInstanceCreateTask) taskFailed(ctx context.Context, dbinstance *models.SDBInstance, err error) {
	dbinstance.SetStatus(self.UserCred, api.DBINSTANCE_CREATE_FAILED, err.Error())
	db.OpsLog.LogEvent(dbinstance, db.ACT_CREATE, err, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, dbinstance, logclient.ACT_CREATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.Marshal(err))
}

func (self *DBInstanceCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	dbinstance := obj.(*models.SDBInstance)
	self.CreateDBInstance(ctx, dbinstance)
}

func (self *DBInstanceCreateTask) CreateDBInstance(ctx context.Context, dbinstance *models.SDBInstance) {
	region := dbinstance.GetRegion()
	self.SetStage("OnCreateDBInstanceComplete", nil)
	err := region.GetDriver().RequestCreateDBInstance(ctx, self.UserCred, dbinstance, self)
	if err != nil {
		self.taskFailed(ctx, dbinstance, err)
		return
	}
}

func (self *DBInstanceCreateTask) OnCreateDBInstanceComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	dbinstance := obj.(*models.SDBInstance)
	logclient.AddActionLogWithStartable(self, dbinstance, logclient.ACT_CREATE, nil, self.UserCred, true)

	accounts, err := dbinstance.GetDBInstanceAccounts()
	if err != nil {
		log.Errorf("failed to get dbinstance %s account error: %v", dbinstance.Name, err)
	}

	if len(accounts) > 0 {
		iRds, err := dbinstance.GetIDBInstance()
		if err != nil {
			log.Errorf("failed to found dbinstance %s error: %v", dbinstance.Name, err)
		} else {
			iAccounts, err := iRds.GetIDBInstanceAccounts()
			if err != nil {
				log.Errorf("failed to get accounts from cloud dbinstance %s error: %v", dbinstance.Name, err)
			}
			externalIds := map[string]string{}
			for _, iAccount := range iAccounts {
				externalIds[iAccount.GetName()] = iAccount.GetGlobalId()
			}
			for i := range accounts {
				externalId, ok := externalIds[accounts[i].Name]
				if !ok {
					log.Errorf("failed to get dbinstance account %s from cloud dbinstance for set externalId", accounts[i].Name)
				} else {
					db.SetExternalId(&accounts[i], self.UserCred, externalId)
				}
			}
		}
	}

	self.SetStage("OnSyncDBInstanceStatusComplete", nil)
	models.StartResourceSyncStatusTask(ctx, self.UserCred, dbinstance, "DBInstanceSyncStatusTask", self.GetTaskId())
}

func (self *DBInstanceCreateTask) OnCreateDBInstanceCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	dbinstance := obj.(*models.SDBInstance)
	self.taskFailed(ctx, dbinstance, fmt.Errorf("%s", data.String()))
}

func (self *DBInstanceCreateTask) OnSyncDBInstanceStatusComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *DBInstanceCreateTask) OnSyncDBInstanceStatusCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}
