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
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DBInstanceSyncStatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DBInstanceSyncStatusTask{})
}

func (self *DBInstanceSyncStatusTask) taskFailed(ctx context.Context, dbinstance *models.SDBInstance, err error) {
	dbinstance.SetStatus(self.UserCred, api.DBINSTANCE_UNKNOWN, err.Error())
	db.OpsLog.LogEvent(dbinstance, db.ACT_SYNC_STATUS, err.Error(), self.GetUserCred())
	logclient.AddActionLogWithStartable(self, dbinstance, logclient.ACT_SYNC_STATUS, err.Error(), self.UserCred, false)
	self.SetStageFailed(ctx, err.Error())
}

func (self *DBInstanceSyncStatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	dbinstance := obj.(*models.SDBInstance)
	self.SyncStatusDBInstance(ctx, dbinstance)
}

func (self *DBInstanceSyncStatusTask) SyncStatusDBInstance(ctx context.Context, dbinstance *models.SDBInstance) {
	idbinstance, err := dbinstance.GetIDBInstance()
	if err != nil {
		self.taskFailed(ctx, dbinstance, errors.Wrapf(err, "dbinstance.GetIDBInstance"))
		return
	}
	status := idbinstance.GetStatus()
	dbinstance.SetStatus(self.UserCred, status, "")
	self.SetStageComplete(ctx, nil)
}
