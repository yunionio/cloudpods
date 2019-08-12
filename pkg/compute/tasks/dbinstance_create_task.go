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
	db.OpsLog.LogEvent(dbinstance, db.ACT_CREATE, err.Error(), self.GetUserCred())
	logclient.AddActionLogWithStartable(self, dbinstance, logclient.ACT_CREATE, err.Error(), self.UserCred, false)
	self.SetStageFailed(ctx, err.Error())
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

	self.SetStage("OnSyncDBInstanceStatusComplete", nil)
	dbinstance.StartDBInstanceSyncStatusTask(ctx, self.UserCred, nil, self.GetTaskId())
}

func (self *DBInstanceCreateTask) OnCreateDBInstanceCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	dbinstance := obj.(*models.SDBInstance)
	self.taskFailed(ctx, dbinstance, fmt.Errorf("%s", data.String()))
}

func (self *DBInstanceCreateTask) OnSyncDBInstanceStatusComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *DBInstanceCreateTask) OnSyncDBInstanceStatusCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data.String())
}
