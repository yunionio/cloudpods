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

type RouteTableUpdateTask struct {
	taskman.STask
}

func (self *RouteTableUpdateTask) taskFailed(ctx context.Context, routeTable *models.SRouteTable, err error) {
	routeTable.SetStatus(ctx, self.GetUserCred(), api.ROUTE_TABLE_UPDATEFAILED, err.Error())
	db.OpsLog.LogEvent(routeTable, db.ACT_UPDATE, err, self.GetUserCred())
	logclient.AddActionLogWithContext(ctx, routeTable, logclient.ACT_UPDATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func init() {
	taskman.RegisterTask(RouteTableUpdateTask{})
}

func (self *RouteTableUpdateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	routeTable := obj.(*models.SRouteTable)
	action, err := self.Params.GetString("action")
	if err != nil {
		self.taskFailed(ctx, routeTable, errors.Wrapf(err, "self.Params.GetString(action)"))
		return
	}

	routeSetId, err := self.Params.GetString("route_table_route_set_id")
	if err != nil {
		self.taskFailed(ctx, routeTable, errors.Wrapf(err, "self.Params.GetString(route_table_route_set_id)"))
		return
	}

	_routeSet, err := models.RouteTableRouteSetManager.FetchById(routeSetId)
	if err != nil {
		self.taskFailed(ctx, routeTable, errors.Wrapf(err, "RouteTableRouteSetManager.FetchById(routeSetId)"))
		return
	}
	routeSet := _routeSet.(*models.SRouteTableRouteSet)

	iRouteTable, err := routeTable.GetICloudRouteTable(ctx)
	if err != nil {
		self.taskFailed(ctx, routeTable, errors.Wrapf(err, "routeTable.GetICloudRouteTable()"))
		return
	}
	cloudRouteSet := cloudprovider.RouteSet{
		RouteId:     routeSet.ExternalId,
		Destination: routeSet.Cidr,
		NextHopType: routeSet.NextHopType,
		NextHop:     routeSet.ExtNextHopId,
	}
	err = nil
	switch action {
	case "create":
		err = iRouteTable.CreateRoute(cloudRouteSet)
		if err != nil {
			self.taskFailed(ctx, routeTable, errors.Wrapf(err, "iRouteTable.CreateRoute(%s)", jsonutils.Marshal(cloudRouteSet).String()))
			return
		}
	case "update":
		err = iRouteTable.UpdateRoute(cloudRouteSet)
		if err != nil {
			self.taskFailed(ctx, routeTable, errors.Wrapf(err, "iRouteTable.UpdateRoute(%s)", jsonutils.Marshal(cloudRouteSet).String()))
			return
		}
	case "delete":
		err = iRouteTable.RemoveRoute(cloudRouteSet)
		if err != nil {
			self.taskFailed(ctx, routeTable, errors.Wrapf(err, "iRouteTable.RemoveRoute(%s)", jsonutils.Marshal(cloudRouteSet).String()))
			return
		}
	default:
		self.taskFailed(ctx, routeTable, errors.Wrapf(err, "invalid routetable update action"))
		return
	}
	logclient.AddActionLogWithContext(ctx, routeTable, logclient.ACT_UPDATE, nil, self.UserCred, true)

	self.SetStage("OnSyncRouteTableComplete", nil)
	models.StartResourceSyncStatusTask(ctx, self.GetUserCred(), routeTable, "RouteTableSyncStatusTask", self.GetTaskId())
}

func (self *RouteTableUpdateTask) OnSyncRouteTableComplete(ctx context.Context, routeTable *models.SRouteTable, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *RouteTableUpdateTask) OnSyncRouteTableCompleteFailed(ctx context.Context, routeTable *models.SRouteTable, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}
