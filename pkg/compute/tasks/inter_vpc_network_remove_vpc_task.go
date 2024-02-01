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

type InterVpcNetworkRemoveVpcTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(InterVpcNetworkRemoveVpcTask{})
}

func (self *InterVpcNetworkRemoveVpcTask) taskFailed(ctx context.Context, network *models.SInterVpcNetwork, err error) {
	network.SetStatus(ctx, self.UserCred, api.INTER_VPC_NETWORK_STATUS_REMOVEVPC_FAILED, err.Error())
	db.OpsLog.LogEvent(network, db.ACT_NETWORK_REMOVE_VPC, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, network, logclient.ACT_NETWORK_REMOVE_VPC, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *InterVpcNetworkRemoveVpcTask) taskComplete(ctx context.Context, network *models.SInterVpcNetwork) {
	network.SetStatus(ctx, self.GetUserCred(), api.INTER_VPC_NETWORK_STATUS_AVAILABLE, "")
	logclient.AddActionLogWithStartable(self, network, logclient.ACT_NETWORK_REMOVE_VPC, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *InterVpcNetworkRemoveVpcTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	interVpcNetwork := obj.(*models.SInterVpcNetwork)
	vpcId, err := self.Params.GetString("vpc_id")
	if err != nil {
		self.taskFailed(ctx, interVpcNetwork, errors.Wrap(err, `self.Params.GetString("vpc_id")`))
		return
	}

	_vpc, err := models.VpcManager.FetchById(vpcId)
	if err != nil {
		self.taskFailed(ctx, interVpcNetwork, errors.Wrap(err, `models.VpcManager.FetchById(vpcId)`))
		return
	}
	vpc := _vpc.(*models.SVpc)
	iVpc, err := vpc.GetIVpc(ctx)
	if err != nil {
		self.taskFailed(ctx, interVpcNetwork, errors.Wrap(err, ` vpc.GetIVpc()`))
		return
	}

	iVpcNetwork, err := interVpcNetwork.GetICloudInterVpcNetwork(ctx)
	if err != nil {
		self.taskFailed(ctx, interVpcNetwork, errors.Wrap(err, "GetICloudInterVpcNetwork()"))
		return
	}

	removeVpcOpts := cloudprovider.SInterVpcNetworkDetachVpcOption{
		VpcId:               iVpc.GetId(),
		VpcRegionId:         iVpc.GetRegion().GetId(),
		VpcAuthorityOwnerId: iVpc.GetAuthorityOwnerId(),
	}

	err = iVpcNetwork.DetachVpc(&removeVpcOpts)
	if err != nil {
		self.taskFailed(ctx, interVpcNetwork, errors.Wrapf(err, "iVpcNetwork.RemoveVpc(%s)", jsonutils.Marshal(removeVpcOpts).String()))
		return
	}

	self.SetStage("OnSyncInterVpcNetworkComplete", nil)
	models.StartResourceSyncStatusTask(ctx, self.GetUserCred(), interVpcNetwork, "InterVpcNetworkSyncstatusTask", self.GetTaskId())
}

func (self *InterVpcNetworkRemoveVpcTask) OnSyncInterVpcNetworkComplete(ctx context.Context, network *models.SInterVpcNetwork, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *InterVpcNetworkRemoveVpcTask) OnSyncInterVpcNetworkCompleteFailed(ctx context.Context, network *models.SInterVpcNetwork, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}
