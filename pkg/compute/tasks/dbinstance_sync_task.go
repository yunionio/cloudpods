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

type DBInstanceSyncTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DBInstanceSyncTask{})
}

func (self *DBInstanceSyncTask) taskFailed(ctx context.Context, rds *models.SDBInstance, err error) {
	rds.SetStatus(ctx, self.UserCred, api.DBINSTANCE_UNKNOWN, err.Error())
	db.OpsLog.LogEvent(rds, db.ACT_SYNC_STATUS, err, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, rds, logclient.ACT_SYNC_CONF, err, self.UserCred, false)
	notifyclient.EventNotify(ctx, self.GetUserCred(), notifyclient.SEventNotifyParam{
		Obj:    rds,
		Action: notifyclient.ActionSyncStatus,
		IsFail: true,
	})
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *DBInstanceSyncTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	rds := obj.(*models.SDBInstance)
	self.SyncDBInstance(ctx, rds)
}

func (self *DBInstanceSyncTask) SyncDBInstance(ctx context.Context, rds *models.SDBInstance) {
	irds, err := rds.GetIDBInstance(ctx)
	if err != nil {
		self.taskFailed(ctx, rds, errors.Wrapf(err, "rds.GetIDBInstance"))
		return
	}
	err = rds.SyncAllWithCloudDBInstance(ctx, self.UserCred, rds.GetCloudprovider(), irds)
	if err != nil {
		self.taskFailed(ctx, rds, errors.Wrapf(err, "rds.GetIDBInstance"))
		return
	}
	self.SetStageComplete(ctx, nil)
}
