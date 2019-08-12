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

type DBInstanceRebootTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DBInstanceRebootTask{})
}

func (self *DBInstanceRebootTask) taskFailed(ctx context.Context, dbinstance *models.SDBInstance, err error) {
	dbinstance.SetStatus(self.UserCred, api.DBINSTANCE_REBOOT_FAILED, err.Error())
	db.OpsLog.LogEvent(dbinstance, db.ACT_REBOOT, err.Error(), self.GetUserCred())
	logclient.AddActionLogWithStartable(self, dbinstance, logclient.ACT_REBOOT, err.Error(), self.UserCred, false)
	self.SetStageFailed(ctx, err.Error())
}

func (self *DBInstanceRebootTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	dbinstance := obj.(*models.SDBInstance)
	self.RebootDBInstance(ctx, dbinstance)
}

func (self *DBInstanceRebootTask) RebootDBInstance(ctx context.Context, dbinstance *models.SDBInstance) {
	idbinstance, err := dbinstance.GetIDBInstance()
	if err != nil {
		self.taskFailed(ctx, dbinstance, errors.Wrap(err, "dbinstance.GetIDBInstance"))
		return
	}
	err = idbinstance.Reboot()
	if err != nil {
		self.taskFailed(ctx, dbinstance, errors.Wrap(err, "idbinstance.Reboot"))
		return
	}
	err = cloudprovider.WaitStatus(idbinstance, api.DBINSTANCE_RUNNING, 10*time.Second, time.Minute*30)
	if err != nil {
		self.taskFailed(ctx, dbinstance, errors.Wrap(err, "cloudprovider.WaitStatus"))
		return
	}
	self.taskComplete(ctx, dbinstance)
}

func (self *DBInstanceRebootTask) taskComplete(ctx context.Context, dbinstance *models.SDBInstance) {
	dbinstance.SetStatus(self.UserCred, api.DBINSTANCE_RUNNING, "")
	logclient.AddActionLogWithStartable(self, dbinstance, logclient.ACT_REBOOT, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}
