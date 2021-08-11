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
	log.Errorf("network create task fail on %s: %s", event, err.Error())
	reason := jsonutils.NewDict()
	reason.Set("event", jsonutils.NewString(event))
	reason.Set("reason", jsonutils.NewString(err.Error()))
	network.SetStatus(self.UserCred, api.NETWORK_STATUS_FAILED, reason.String())
	db.OpsLog.LogEvent(network, db.ACT_ALLOCATE_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, network, logclient.ACT_CREATE, reason, self.UserCred, false)
	self.SetStageFailed(ctx, reason)
}

func (self *NetworkCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	network := obj.(*models.SNetwork)

	network.SetStatus(self.UserCred, api.NETWORK_STATUS_PENDING, "")

	wire, _ := network.GetWire()
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

	opts := cloudprovider.SNetworkCreateOptions{
		Name: network.Name,
		Cidr: prefix.String(),
		Desc: network.Description,
	}

	provider := wire.GetCloudprovider()
	opts.ProjectId, err = provider.SyncProject(ctx, self.GetUserCred(), network.ProjectId)
	if err != nil {
		log.Errorf("failed to sync project %s for create %s network %s error: %v", network.ProjectId, provider.Provider, network.Name, err)
	}

	inet, err := iwire.CreateINetwork(&opts)
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

	err = network.SyncWithCloudNetwork(ctx, self.UserCred, inet, nil, nil)

	if err != nil {
		self.taskFailed(ctx, network, "SyncWithCloudNetwork", err)
		return
	}

	network.ClearSchedDescCache()
	logclient.AddActionLogWithStartable(self, network, logclient.ACT_CREATE, "", self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}
