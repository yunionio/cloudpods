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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DBInstanceRenewTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DBInstanceRenewTask{})
}

func (self *DBInstanceRenewTask) taskFailed(ctx context.Context, rds *models.SDBInstance, err error) {
	db.OpsLog.LogEvent(rds, db.ACT_REW_FAIL, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, rds, logclient.ACT_RENEW, err, self.UserCred, false)
	rds.SetStatus(ctx, self.GetUserCred(), api.DBINSTANCE_RENEW_FAILED, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *DBInstanceRenewTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	rds := obj.(*models.SDBInstance)

	duration, _ := self.GetParams().GetString("duration")
	bc, err := billing.ParseBillingCycle(duration)
	if err != nil {
		self.taskFailed(ctx, rds, errors.Wrapf(err, "ParseBillingCycle(%s)", duration))
		return
	}

	iRds, err := rds.GetIDBInstance(ctx)
	if err != nil {
		self.taskFailed(ctx, rds, errors.Wrapf(err, "GetIDBInstance"))
		return
	}
	oldExpired := iRds.GetExpiredAt()

	err = iRds.Renew(bc)
	if err != nil {
		self.taskFailed(ctx, rds, errors.Wrapf(err, "iRds.Renew"))
		return
	}

	err = cloudprovider.WaitCreated(15*time.Second, 5*time.Minute, func() bool {
		err := iRds.Refresh()
		if err != nil {
			log.Errorf("failed refresh rds %s error: %v", rds.Name, err)
		}
		newExipred := iRds.GetExpiredAt()
		if newExipred.After(oldExpired) {
			return true
		}
		return false
	})
	if err != nil {
		self.taskFailed(ctx, rds, errors.Wrapf(err, "wait expired time refresh"))
		return
	}

	logclient.AddActionLogWithStartable(self, rds, logclient.ACT_RENEW, map[string]string{"duration": duration}, self.UserCred, true)

	self.SetStage("OnSyncstatusComplete", nil)
	rds.StartDBInstanceSyncTask(ctx, self.UserCred, self.GetTaskId())
}

func (self *DBInstanceRenewTask) OnSyncstatusComplete(ctx context.Context, rds *models.SDBInstance, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *DBInstanceRenewTask) OnSyncstatusCompleteFailed(ctx context.Context, rds *models.SDBInstance, reason jsonutils.JSONObject) {
	self.SetStageFailed(ctx, reason)
}
