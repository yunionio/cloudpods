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

type InterVpcNetworkUpdateRoutesetTask struct {
	taskman.STask
}

func (self *InterVpcNetworkUpdateRoutesetTask) taskFailed(ctx context.Context, network *models.SInterVpcNetwork, err error) {
	network.SetStatus(ctx, self.GetUserCred(), api.INTER_VPC_NETWORK_STATUS_UPDATEROUTE_FAILED, err.Error())
	db.OpsLog.LogEvent(network, db.ACT_NETWORK_MODIFY_ROUTE, err, self.GetUserCred())
	logclient.AddActionLogWithContext(ctx, network, logclient.ACT_NETWORK_MODIFY_ROUTE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func init() {
	taskman.RegisterTask(InterVpcNetworkUpdateRoutesetTask{})
}

func (self *InterVpcNetworkUpdateRoutesetTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	network := obj.(*models.SInterVpcNetwork)
	action, err := self.Params.GetString("action")
	if err != nil {
		self.taskFailed(ctx, network, errors.Wrapf(err, "self.Params.GetString(action)"))
		return
	}

	routeSetId, err := self.Params.GetString("inter_vpc_network_route_set_id")
	if err != nil {
		self.taskFailed(ctx, network, errors.Wrapf(err, "self.Params.GetString(inter_vpc_network_route_set_id)"))
		return
	}

	_routeSet, err := models.InterVpcNetworkRouteSetManager.FetchById(routeSetId)
	if err != nil {
		self.taskFailed(ctx, network, errors.Wrapf(err, "InterVpcNetworkRouteSetManager.FetchById(routeSetId)"))
		return
	}
	routeSet := _routeSet.(*models.SInterVpcNetworkRouteSet)

	iNetwork, err := network.GetICloudInterVpcNetwork(ctx)
	if err != nil {
		self.taskFailed(ctx, network, errors.Wrap(err, "GetICloudInterVpcNetwork()"))
		return
	}

	switch action {
	case "enable":
		err := iNetwork.EnableRouteEntry(routeSet.ExternalId)
		if err != nil {
			self.taskFailed(ctx, network, errors.Wrapf(err, "iNetwork.EnableRouteEntry(%s)", routeSet.ExternalId))
			return
		}
	case "disable":
		err := iNetwork.DisableRouteEntry(routeSet.ExternalId)
		if err != nil {
			self.taskFailed(ctx, network, errors.Wrapf(err, "iNetwork.DisableRouteEntry(%s)", routeSet.ExternalId))
			return
		}
	default:
		self.taskFailed(ctx, network, errors.Wrapf(err, "invalid InterVpcNetworkRoute update action"))
		return
	}
	logclient.AddActionLogWithContext(ctx, network, logclient.ACT_NETWORK_MODIFY_ROUTE, err, self.UserCred, true)

	self.SetStage("OnSyncInterVpcNetworkComplete", nil)
	models.StartResourceSyncStatusTask(ctx, self.GetUserCred(), network, "InterVpcNetworkSyncstatusTask", self.GetTaskId())
}

func (self *InterVpcNetworkUpdateRoutesetTask) OnSyncInterVpcNetworkComplete(ctx context.Context, network *models.SInterVpcNetwork, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *InterVpcNetworkUpdateRoutesetTask) OnSyncInterVpcNetworkCompleteFailed(ctx context.Context, network *models.SInterVpcNetwork, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}
