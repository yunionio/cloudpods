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

type DBInstanceChangeConfigTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DBInstanceChangeConfigTask{})
}

func (self *DBInstanceChangeConfigTask) taskFailed(ctx context.Context, rds *models.SDBInstance, err error) {
	rds.SetStatus(ctx, self.UserCred, api.DBINSTANCE_CHANGE_CONFIG_FAILED, err.Error())
	db.OpsLog.LogEvent(rds, db.ACT_CHANGE_CONFIG, err, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, rds, logclient.ACT_CHANGE_CONFIG, err, self.UserCred, false)
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    rds,
		Action: notifyclient.ActionChangeConfig,
		IsFail: true,
	})
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *DBInstanceChangeConfigTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	rds := obj.(*models.SDBInstance)
	self.SetStage("OnDBInstanceChangeConfigComplete", nil)
	input := &api.SDBInstanceChangeConfigInput{}
	err := self.GetParams().Unmarshal(input)
	if err != nil {
		self.taskFailed(ctx, rds, errors.Wrapf(err, "GetParams().Unmarshal"))
		return
	}

	region, err := rds.GetRegion()
	if err != nil {
		self.taskFailed(ctx, rds, errors.Wrapf(err, "GetRegion"))
		return
	}

	err = region.GetDriver().RequestChangeDBInstanceConfig(ctx, self.UserCred, rds, input, self)
	if err != nil {
		self.taskFailed(ctx, rds, err)
		return
	}
}

func (self *DBInstanceChangeConfigTask) OnDBInstanceChangeConfigComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	rds := obj.(*models.SDBInstance)
	logclient.AddActionLogWithStartable(self, rds, logclient.ACT_CHANGE_CONFIG, nil, self.UserCred, true)
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    rds,
		Action: notifyclient.ActionChangeConfig,
	})
	self.SetStage("OnSyncDBInstanceStatusComplete", nil)
	rds.StartDBInstanceSyncTask(ctx, self.UserCred, self.GetTaskId())
}

func (self *DBInstanceChangeConfigTask) OnDBInstanceChangeConfigCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	instance := obj.(*models.SDBInstance)
	self.taskFailed(ctx, instance, fmt.Errorf("%s", data.String()))
}

func (self *DBInstanceChangeConfigTask) OnSyncDBInstanceStatusComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *DBInstanceChangeConfigTask) OnSyncDBInstanceStatusCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}
