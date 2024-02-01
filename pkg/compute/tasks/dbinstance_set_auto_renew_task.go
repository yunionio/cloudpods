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
	"yunion.io/x/pkg/util/billing"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DBInstanceSetAutoRenewTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DBInstanceSetAutoRenewTask{})
}

func (self *DBInstanceSetAutoRenewTask) taskFailed(ctx context.Context, rds *models.SDBInstance, err error) {
	db.OpsLog.LogEvent(rds, db.ACT_SET_AUTO_RENEW_FAIL, err, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, rds, logclient.ACT_SET_AUTO_RENEW, err, self.GetUserCred(), false)
	rds.SetStatus(ctx, self.GetUserCred(), api.DBINSTANCE_SET_AUTO_RENEW_FAILED, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *DBInstanceSetAutoRenewTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	rds := obj.(*models.SDBInstance)
	autoRenew, _ := self.GetParams().Bool("auto_renew")
	iRds, err := rds.GetIDBInstance(ctx)
	if err != nil {
		self.taskFailed(ctx, rds, errors.Wrapf(err, "GetIDBInstance"))
		return
	}
	bc := billing.SBillingCycle{}
	bc.AutoRenew = autoRenew
	err = iRds.SetAutoRenew(bc)
	if err != nil {
		self.taskFailed(ctx, rds, errors.Wrapf(err, "iRds.SetAutoRenew"))
		return
	}
	self.SetStage("OnDBInstanceSyncComplete", nil)
	rds.StartDBInstanceSyncTask(ctx, self.GetUserCred(), self.GetTaskId())
}

func (self *DBInstanceSetAutoRenewTask) OnDBInstanceSyncComplete(ctx context.Context, rds *models.SDBInstance, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *DBInstanceSetAutoRenewTask) OnDBInstanceSyncCompleteFailed(ctx context.Context, rds *models.SDBInstance, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}
