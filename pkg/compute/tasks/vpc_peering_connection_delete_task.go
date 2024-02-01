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

type VpcPeeringConnectionDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(VpcPeeringConnectionDeleteTask{})
}

func (self *VpcPeeringConnectionDeleteTask) taskFailed(ctx context.Context, peer *models.SVpcPeeringConnection, err error) {
	peer.SetStatus(ctx, self.UserCred, api.VPC_PEERING_CONNECTION_STATUS_DELETE_FAILED, err.Error())
	db.OpsLog.LogEvent(peer, db.ACT_DELETE, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, peer, logclient.ACT_DELETE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *VpcPeeringConnectionDeleteTask) taskComplete(ctx context.Context, peer *models.SVpcPeeringConnection) {
	logclient.AddActionLogWithStartable(self, peer, logclient.ACT_DELETE, nil, self.UserCred, true)
	peer.RealDelete(ctx, self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}

func (self *VpcPeeringConnectionDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	peer := obj.(*models.SVpcPeeringConnection)

	vpc, err := peer.GetVpc()
	if err != nil {
		self.taskFailed(ctx, peer, errors.Wrapf(err, "GetVpc"))
		return
	}

	iVpc, err := vpc.GetIVpc(ctx)
	if err != nil {
		self.taskFailed(ctx, peer, errors.Wrapf(err, "GetIVpc"))
		return
	}

	if len(peer.ExternalId) == 0 {
		self.taskComplete(ctx, peer)
		return
	}

	iPeer, err := iVpc.GetICloudVpcPeeringConnectionById(peer.ExternalId)
	if err != nil {
		if errors.Cause(err) != cloudprovider.ErrNotFound {
			self.taskFailed(ctx, peer, errors.Wrapf(err, "GetICloudVpcPeeringConnectionById(%s)", peer.ExternalId))
			return
		}
		self.taskComplete(ctx, peer)
		return
	}

	err = iPeer.Delete()
	if err != nil {
		self.taskFailed(ctx, peer, errors.Wrapf(err, "Delete"))
		return
	}

	self.taskComplete(ctx, peer)
}
