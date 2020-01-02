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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type NetworkCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(NetworkCreateTask{})
}

func (self *NetworkCreateTask) taskFailed(ctx context.Context, network *models.SNetwork, event string, err error) {
	log.Errorf("network create task fail on %s: %s", event, err)
	network.SetStatus(self.UserCred, api.NETWORK_STATUS_FAILED, err.Error())
	db.OpsLog.LogEvent(network, db.ACT_ALLOCATE_FAIL, err.Error(), self.UserCred)
	logclient.AddActionLogWithStartable(self, network, logclient.ACT_CREATE, event, self.UserCred, false)
	self.SetStageFailed(ctx, err.Error())
}

func (self *NetworkCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	network := obj.(*models.SNetwork)

	network.SetStatus(self.UserCred, api.NETWORK_STATUS_PENDING, "")

	wire := network.GetWire()
	if wire == nil {
		self.taskFailed(ctx, network, "getwire", fmt.Errorf("no vpc"))
		return
	}

	iwire, err := wire.GetIWire()
	if err != nil {
		self.taskFailed(ctx, network, "getiwire", err)
		return
	}

	prefix, err := network.GetPrefix()
	if err != nil {
		self.taskFailed(ctx, network, "getprefix", err)
		return
	}

	inet, err := iwire.CreateINetwork(network.Name, prefix.String(), network.Description)
	if err != nil {
		self.taskFailed(ctx, network, "createinetwork", err)
		return
	}
	db.SetExternalId(network, self.UserCred, inet.GetGlobalId())

	err = cloudprovider.WaitStatus(inet, api.NETWORK_STATUS_AVAILABLE, 10*time.Second, 300*time.Second)
	if err != nil {
		self.taskFailed(ctx, network, "waitstatu", err)
		return
	}

	err = network.SyncWithCloudNetwork(ctx, self.UserCred, inet, nil)

	if err != nil {
		self.taskFailed(ctx, network, "SyncWithCloudNetwork", err)
		return
	}

	network.ClearSchedDescCache()
	logclient.AddActionLogWithStartable(self, network, logclient.ACT_CREATE, "", self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}
