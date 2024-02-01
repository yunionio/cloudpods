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
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DBInstanceCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DBInstanceCreateTask{})
}

func (self *DBInstanceCreateTask) taskFailed(ctx context.Context, rds *models.SDBInstance, err error) {
	rds.SetStatus(ctx, self.UserCred, api.DBINSTANCE_CREATE_FAILED, err.Error())
	db.OpsLog.LogEvent(rds, db.ACT_CREATE, err, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, rds, logclient.ACT_CREATE, err, self.UserCred, false)
	notifyclient.EventNotify(ctx, self.GetUserCred(), notifyclient.SEventNotifyParam{
		Obj:    rds,
		Action: notifyclient.ActionCreate,
		IsFail: true,
	})
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *DBInstanceCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	rds := obj.(*models.SDBInstance)
	self.CreateDBInstance(ctx, rds)
}

func (self *DBInstanceCreateTask) CreateDBInstance(ctx context.Context, rds *models.SDBInstance) {
	region, err := rds.GetRegion()
	if err != nil {
		self.taskFailed(ctx, rds, errors.Wrapf(err, "GetRegion"))
		return
	}
	self.SetStage("OnCreateDBInstanceComplete", nil)
	if len(rds.DBInstancebackupId) > 0 {
		err = region.GetDriver().RequestCreateDBInstanceFromBackup(ctx, self.UserCred, rds, self)
	} else {
		err = region.GetDriver().RequestCreateDBInstance(ctx, self.UserCred, rds, self)
	}
	if err != nil {
		self.taskFailed(ctx, rds, err)
		return
	}
}

func (self *DBInstanceCreateTask) OnCreateDBInstanceComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	rds := obj.(*models.SDBInstance)
	logclient.AddActionLogWithStartable(self, rds, logclient.ACT_CREATE, nil, self.UserCred, true)
	self.SetStage("OnSyncDBInstanceStatusComplete", nil)
	rds.StartDBInstanceSyncTask(ctx, self.UserCred, self.GetTaskId())
}

func (self *DBInstanceCreateTask) OnCreateDBInstanceCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	rds := obj.(*models.SDBInstance)
	self.taskFailed(ctx, rds, fmt.Errorf("%s", data.String()))
}

func (self *DBInstanceCreateTask) OnSyncDBInstanceStatusComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	rds := obj.(*models.SDBInstance)
	//notifyclient.NotifyWebhook(ctx, self.UserCred, rds, notifyclient.ActionCreate)
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    rds,
		Action: notifyclient.ActionCreate,
	})
	self.SetStageComplete(ctx, nil)
}

func (self *DBInstanceCreateTask) OnSyncDBInstanceStatusCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}
