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
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DBInstanceRenewTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DBInstanceRenewTask{})
}

func (self *DBInstanceRenewTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	instance := obj.(*models.SDBInstance)

	durationStr, _ := self.GetParams().GetString("duration")
	bc, _ := billing.ParseBillingCycle(durationStr)

	exp, err := instance.GetRegion().GetDriver().RequestRenewDBInstance(instance, bc)
	if err != nil {
		msg := fmt.Sprintf("RequestRenewDBInstance failed %s", err)
		log.Errorf(msg)
		db.OpsLog.LogEvent(instance, db.ACT_REW_FAIL, msg, self.UserCred)
		logclient.AddActionLogWithStartable(self, instance, logclient.ACT_RENEW, msg, self.UserCred, false)
		instance.SetStatus(self.GetUserCred(), api.DBINSTANCE_RENEW_FAILED, msg)
		self.SetStageFailed(ctx, msg)
		return
	}

	err = instance.SaveRenewInfo(ctx, self.UserCred, &bc, &exp)
	if err != nil {
		msg := fmt.Sprintf("SaveRenewInfo fail %s", err)
		log.Errorf(msg)
		self.SetStageFailed(ctx, msg)
		return
	}

	logclient.AddActionLogWithStartable(self, instance, logclient.ACT_RENEW, nil, self.UserCred, true)

	instance.StartDBInstanceSyncStatusTask(ctx, self.UserCred, nil, self.GetTaskId())
	self.SetStageComplete(ctx, nil)
}
