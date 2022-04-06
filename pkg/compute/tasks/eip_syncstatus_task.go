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

type EipSyncstatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(EipSyncstatusTask{})
}

func (self *EipSyncstatusTask) taskFail(ctx context.Context, eip *models.SElasticip, msg jsonutils.JSONObject) {
	eip.SetStatus(self.UserCred, api.EIP_STATUS_UNKNOWN, msg.String())
	db.OpsLog.LogEvent(eip, db.ACT_SYNC_STATUS, msg, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, eip, logclient.ACT_SYNC_STATUS, msg, self.UserCred, false)
	self.SetStageFailed(ctx, msg)
}

func (self *EipSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	eip := obj.(*models.SElasticip)

	extEip, err := eip.GetIEip(ctx)
	if err != nil {
		msg := fmt.Sprintf("fail to find ieip for eip %s", err)
		self.taskFail(ctx, eip, jsonutils.NewString(msg))
		return
	}

	err = extEip.Refresh()
	if err != nil {
		msg := fmt.Sprintf("fail to refresh eip status %s", err)
		self.taskFail(ctx, eip, jsonutils.NewString(msg))
		return
	}

	err = eip.SyncWithCloudEip(ctx, self.UserCred, eip.GetCloudprovider(), extEip, nil)
	if err != nil {
		msg := fmt.Sprintf("fail to sync eip status %s", err)
		self.taskFail(ctx, eip, jsonutils.NewString(msg))
		return
	}

	err = eip.SyncInstanceWithCloudEip(ctx, self.UserCred, extEip)
	if err != nil {
		msg := fmt.Sprintf("fail to sync eip status %s", err)
		self.taskFail(ctx, eip, jsonutils.NewString(msg))
		return
	}

	logclient.AddActionLogWithStartable(self, eip, logclient.ACT_SYNC_STATUS, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}
