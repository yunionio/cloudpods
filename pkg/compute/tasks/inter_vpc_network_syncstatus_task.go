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
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type InterVpcNetworkSyncstatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(InterVpcNetworkSyncstatusTask{})
}

func (self *InterVpcNetworkSyncstatusTask) taskFail(ctx context.Context, peer *models.SInterVpcNetwork, err error) {
	peer.SetStatus(ctx, self.UserCred, api.VPC_PEERING_CONNECTION_STATUS_UNKNOWN, err.Error())
	db.OpsLog.LogEvent(peer, db.ACT_SYNC_STATUS, err, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, peer, logclient.ACT_SYNC_STATUS, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *InterVpcNetworkSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	snetwork := obj.(*models.SInterVpcNetwork)
	inetwork, err := snetwork.GetICloudInterVpcNetwork(ctx)
	if err != nil {
		self.taskFail(ctx, snetwork, errors.Wrap(err, "GetICloudInterVpcNetwork()"))
		return
	}

	err = snetwork.SyncWithCloudInterVpcNetwork(ctx, self.UserCred, inetwork)
	if err != nil {
		self.taskFail(ctx, snetwork, errors.Wrap(err, "snetwork.SyncWithCloudInterVpcNetwork()"))
		return
	}

	result := snetwork.SyncInterVpcNetworkRouteSets(ctx, self.UserCred, inetwork, false)
	log.Infof("sync routes for %s result: %s", snetwork.GetName(), result.Result())

	logclient.AddActionLogWithStartable(self, snetwork, logclient.ACT_SYNC_STATUS, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}
