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

type VpcSyncstatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(VpcSyncstatusTask{})
}

func (self *VpcSyncstatusTask) taskFail(ctx context.Context, vpc *models.SVpc, msg string) {
	vpc.SetStatus(self.UserCred, api.VPC_STATUS_UNKNOWN, msg)
	db.OpsLog.LogEvent(vpc, db.ACT_SYNC_STATUS, msg, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, vpc, logclient.ACT_SYNC_STATUS, msg, self.UserCred, false)
	self.SetStageFailed(ctx, msg)
}

func (self *VpcSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	vpc := obj.(*models.SVpc)

	vpc.SetStatus(self.UserCred, api.VPC_STATUS_SYNCING, "synchronize")

	extVpc, err := vpc.GetIVpc()
	if err != nil {
		msg := fmt.Sprintf("fail to find ICloudVpc for vpc %s", err)
		self.taskFail(ctx, vpc, msg)
		return
	}

	err = extVpc.Refresh()
	if err != nil {
		msg := fmt.Sprintf("fail to refresh ICloudVpc status %s", err)
		self.taskFail(ctx, vpc, msg)
		return
	}

	err = vpc.SyncWithCloudVpc(ctx, self.UserCred, extVpc)
	if err != nil {
		msg := fmt.Sprintf("fail to sync vpc status %s", err)
		self.taskFail(ctx, vpc, msg)
		return
	}

	logclient.AddActionLogWithStartable(self, vpc, logclient.ACT_SYNC_STATUS, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}
