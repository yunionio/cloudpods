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
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type InterVpcNetworkDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(InterVpcNetworkDeleteTask{})
}

func (self *InterVpcNetworkDeleteTask) taskFailed(ctx context.Context, network *models.SInterVpcNetwork, err error) {
	network.SetStatus(ctx, self.UserCred, api.INTER_VPC_NETWORK_STATUS_DELETE_FAILED, err.Error())
	db.OpsLog.LogEvent(network, db.ACT_DELETE, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, network, logclient.ACT_DELETE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *InterVpcNetworkDeleteTask) taskComplete(ctx context.Context, network *models.SInterVpcNetwork) {
	logclient.AddActionLogWithStartable(self, network, logclient.ACT_DELETE, nil, self.UserCred, true)
	network.RealDelete(ctx, self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}

func (self *InterVpcNetworkDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	interVpcNetwork := obj.(*models.SInterVpcNetwork)
	provider, err := interVpcNetwork.GetProvider(ctx)
	if err != nil {
		self.taskFailed(ctx, interVpcNetwork, errors.Wrapf(err, "GetProvider"))
		return
	}
	if len(interVpcNetwork.ExternalId) == 0 {
		self.taskComplete(ctx, interVpcNetwork)
		return
	}
	inetwork, err := provider.GetICloudInterVpcNetworkById(interVpcNetwork.ExternalId)
	if err != nil {
		self.taskFailed(ctx, interVpcNetwork, errors.Wrap(err, "provider.GetICloudInterVpcNetworkById()"))
		return
	}
	err = inetwork.Delete()
	if err != nil {
		self.taskFailed(ctx, interVpcNetwork, errors.Wrap(err, "inetwork.Delete"))
		return
	}
	self.taskComplete(ctx, interVpcNetwork)
}
