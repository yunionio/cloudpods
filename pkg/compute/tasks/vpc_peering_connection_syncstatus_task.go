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

type VpcPeeringConnectionSyncstatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(VpcPeeringConnectionSyncstatusTask{})
}

func (self *VpcPeeringConnectionSyncstatusTask) taskFail(ctx context.Context, peer *models.SVpcPeeringConnection, err error) {
	peer.SetStatus(ctx, self.UserCred, api.VPC_PEERING_CONNECTION_STATUS_UNKNOWN, err.Error())
	db.OpsLog.LogEvent(peer, db.ACT_SYNC_STATUS, err, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, peer, logclient.ACT_SYNC_STATUS, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *VpcPeeringConnectionSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	peer := obj.(*models.SVpcPeeringConnection)

	svpc, err := peer.GetVpc()
	if err != nil {
		self.taskFail(ctx, peer, errors.Wrap(err, "peer.GetVpc()"))
		return
	}

	extVpc, err := svpc.GetIVpc(ctx)
	if err != nil {
		self.taskFail(ctx, peer, errors.Wrap(err, "svpc.GetIVpc()"))
		return
	}

	ipeer, err := extVpc.GetICloudVpcPeeringConnectionById(peer.GetExternalId())
	if err != nil {
		self.taskFail(ctx, peer, errors.Wrapf(err, "extVpc.GetICloudVpcPeeringConnectionById(%s)", peer.GetExternalId()))
		return
	}

	err = ipeer.Refresh()
	if err != nil {
		self.taskFail(ctx, peer, errors.Wrap(err, "ipeer.Refresh()"))
		return
	}

	err = peer.SyncWithCloudPeerConnection(ctx, self.UserCred, ipeer)
	if err != nil {
		self.taskFail(ctx, peer, errors.Wrapf(err, "SyncWithCloudPeerConnection(ctx, self.UserCred, %s, nil)", jsonutils.Marshal(ipeer).String()))
		return
	}

	self.SetStageComplete(ctx, nil)
}
