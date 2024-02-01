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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type InterVpcNetworkCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(InterVpcNetworkCreateTask{})
}

func (self *InterVpcNetworkCreateTask) taskFailed(ctx context.Context, network *models.SInterVpcNetwork, err error) {
	network.SetStatus(ctx, self.UserCred, api.INTER_VPC_NETWORK_STATUS_CREATE_FAILED, err.Error())
	db.OpsLog.LogEvent(network, db.ACT_CREATE, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, network, logclient.ACT_CREATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *InterVpcNetworkCreateTask) taskComplete(ctx context.Context, network *models.SInterVpcNetwork) {
	network.SetStatus(ctx, self.GetUserCred(), api.INTER_VPC_NETWORK_STATUS_AVAILABLE, "")
	logclient.AddActionLogWithStartable(self, network, logclient.ACT_CREATE, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *InterVpcNetworkCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	interVpcNetwork := obj.(*models.SInterVpcNetwork)
	provider, err := interVpcNetwork.GetProvider(ctx)
	if err != nil {
		self.taskFailed(ctx, interVpcNetwork, errors.Wrapf(err, "GetProvider"))
		return
	}
	opts := &cloudprovider.SInterVpcNetworkCreateOptions{
		Name: interVpcNetwork.Name,
		Desc: interVpcNetwork.Description,
	}

	inetwork, err := provider.CreateICloudInterVpcNetwork(opts)
	if err != nil {
		self.taskFailed(ctx, interVpcNetwork, errors.Wrap(err, "provider.CreateICloudInterVpcNetwork()"))
		return
	}
	err = interVpcNetwork.SyncWithCloudInterVpcNetwork(ctx, self.UserCred, inetwork)
	if err != nil {
		self.taskFailed(ctx, interVpcNetwork, errors.Wrap(err, "snetwork.SyncWithCloudInterVpcNetwork()"))
		return
	}
	self.taskComplete(ctx, interVpcNetwork)
}
