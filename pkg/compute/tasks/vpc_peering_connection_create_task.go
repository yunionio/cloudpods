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

type VpcPeeringConnectionCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(VpcPeeringConnectionCreateTask{})
}

func (self *VpcPeeringConnectionCreateTask) taskFailed(ctx context.Context, peer *models.SVpcPeeringConnection, err error) {
	peer.SetStatus(ctx, self.UserCred, api.VPC_PEERING_CONNECTION_STATUS_CREATE_FAILED, err.Error())
	db.OpsLog.LogEvent(peer, db.ACT_CREATE, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, peer, logclient.ACT_CREATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *VpcPeeringConnectionCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	peer := obj.(*models.SVpcPeeringConnection)

	vpc, err := peer.GetVpc()
	if err != nil {
		self.taskFailed(ctx, peer, errors.Wrapf(err, "GetVpc"))
		return
	}

	peerVpc, err := peer.GetPeerVpc()
	if err != nil {
		self.taskFailed(ctx, peer, errors.Wrapf(err, "GetPeerVpc"))
		return
	}

	iVpc, err := vpc.GetIVpc(ctx)
	if err != nil {
		self.taskFailed(ctx, peer, errors.Wrapf(err, "GetIVpc"))
		return
	}

	iPeerVpc, err := peerVpc.GetIVpc(ctx)
	if err != nil {
		self.taskFailed(ctx, peer, errors.Wrapf(err, "GetIVpc"))
		return
	}

	opts := &cloudprovider.VpcPeeringConnectionCreateOptions{
		Name:          peer.Name,
		Desc:          peer.Description,
		Bandwidth:     peer.Bandwidth,
		PeerVpcId:     iPeerVpc.GetId(),
		PeerRegionId:  iPeerVpc.GetRegion().GetId(),
		PeerAccountId: iPeerVpc.GetAuthorityOwnerId(),
	}
	iPeerConnection, err := iVpc.CreateICloudVpcPeeringConnection(opts)
	if err != nil {
		self.taskFailed(ctx, peer, errors.Wrapf(err, "CreateICloudVpcPeeringConnection"))
		return
	}
	err = iPeerVpc.AcceptICloudVpcPeeringConnection(iPeerConnection.GetGlobalId())
	if err != nil {
		self.taskFailed(ctx, peer, errors.Wrapf(err, "AcceptICloudVpcPeeringConnection"))
		return
	}

	iPeerConnection.Refresh()
	err = peer.SyncWithCloudPeerConnection(ctx, self.GetUserCred(), iPeerConnection)
	if err != nil {
		self.taskFailed(ctx, peer, errors.Wrapf(err, "SyncWithCloudPeerConnection"))
		return
	}
	self.taskComplete(ctx, peer)
}

func (self *VpcPeeringConnectionCreateTask) taskComplete(ctx context.Context, peer *models.SVpcPeeringConnection) {
	logclient.AddActionLogWithStartable(self, peer, logclient.ACT_CREATE, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}
