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

type NetworkSyncstatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(NetworkSyncstatusTask{})
}

func (self *NetworkSyncstatusTask) taskFail(ctx context.Context, net *models.SNetwork, msg string) {
	net.SetStatus(self.UserCred, api.NETWORK_STATUS_UNKNOWN, msg)
	db.OpsLog.LogEvent(net, db.ACT_SYNC_STATUS, msg, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, net, logclient.ACT_SYNC_STATUS, msg, self.UserCred, false)
	self.SetStageFailed(ctx, msg)
}

func (self *NetworkSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	net := obj.(*models.SNetwork)

	net.SetStatus(self.UserCred, api.NETWORK_STATUS_SYNCING, "synchronize")

	extNet, err := net.GetINetwork()
	if err != nil {
		msg := fmt.Sprintf("fail to find ICloudNetwork for network %s", err)
		self.taskFail(ctx, net, msg)
		return
	}

	err = extNet.Refresh()
	if err != nil {
		msg := fmt.Sprintf("fail to refresh ICloudNetwork status %s", err)
		self.taskFail(ctx, net, msg)
		return
	}

	err = net.SyncWithCloudNetwork(ctx, self.UserCred, extNet, nil)
	if err != nil {
		msg := fmt.Sprintf("fail to sync network status %s", err)
		self.taskFail(ctx, net, msg)
		return
	}

	logclient.AddActionLogWithStartable(self, net, logclient.ACT_SYNC_STATUS, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}
