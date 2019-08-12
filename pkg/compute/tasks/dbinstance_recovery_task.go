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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DBInstanceRecoveryTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DBInstanceRecoveryTask{})
}

func (self *DBInstanceRecoveryTask) taskFailed(ctx context.Context, instance *models.SDBInstance, err error) {
	instance.SetStatus(self.UserCred, api.DBINSTANCE_RESTORE_FAILED, err.Error())
	db.OpsLog.LogEvent(instance, db.ACT_RESTORE, err.Error(), self.GetUserCred())
	logclient.AddActionLogWithStartable(self, instance, logclient.ACT_RESTORE, err.Error(), self.UserCred, false)
	self.SetStageFailed(ctx, err.Error())
}

func (self *DBInstanceRecoveryTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	instance := obj.(*models.SDBInstance)

	iRds, err := instance.GetIDBInstance()
	if err != nil {
		self.taskFailed(ctx, instance, errors.Wrap(err, "instance.GetIDBInstance"))
		return
	}

	input := &api.SDBInstanceRecoveryConfigInput{}
	err = self.GetParams().Unmarshal(input)
	if err != nil {
		self.taskFailed(ctx, instance, errors.Wrap(err, "self.GetParams().Unmarshal(input)"))
		return
	}

	backup, err := instance.GetDBInstanceBackup(input.DBInstancebackupId)
	if err != nil {
		self.taskFailed(ctx, instance, errors.Wrapf(err, "instance.GetDBInstanceBackup(%s)", input.DBInstancebackupId))
		return
	}

	conf := &cloudprovider.SDBInstanceRecoveryConfig{
		BackupId:  backup.ExternalId,
		Databases: input.Databases,
	}

	err = iRds.RecoveryFromBackup(conf)
	if err != nil {
		self.taskFailed(ctx, instance, errors.Wrap(err, "iRds.RecoveryFromBackup"))
		return
	}

	err = cloudprovider.WaitStatus(iRds, api.DBINSTANCE_RUNNING, time.Second*10, time.Second*40)
	if err != nil {
		self.taskFailed(ctx, instance, errors.Wrap(err, "cloudprovider.WaitStatus(running)"))
		return
	}

	instance.SetStatus(self.UserCred, api.DBINSTANCE_RUNNING, "")
	self.SetStageComplete(ctx, nil)
}
