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
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type NetworkCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(NetworkCreateTask{})
}

func (self *NetworkCreateTask) taskFailed(ctx context.Context, network *models.SNetwork, err error) {
	network.SetStatus(ctx, self.UserCred, api.NETWORK_STATUS_FAILED, err.Error())
	db.OpsLog.LogEvent(network, db.ACT_ALLOCATE_FAIL, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, network, logclient.ACT_CREATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *NetworkCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	net := obj.(*models.SNetwork)

	net.SetStatus(ctx, self.UserCred, api.NETWORK_STATUS_PENDING, "")

	region, err := net.GetRegion()
	if err != nil {
		self.taskFailed(ctx, net, errors.Wrapf(err, "GetRegion"))
		return
	}

	driver := region.GetDriver()
	if driver == nil {
		self.taskFailed(ctx, net, errors.Wrapf(err, "GetRegionDriver for %s", region.Provider))
		return
	}

	err = driver.RequestCreateNetwork(ctx, self.GetUserCred(), net, self)
	if err != nil {
		self.taskFailed(ctx, net, errors.Wrapf(err, "RequestCreateNetwork"))
		return
	}

	net.ClearSchedDescCache()
	logclient.AddActionLogWithStartable(self, net, logclient.ACT_CREATE, "", self.UserCred, true)
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    net,
		Action: notifyclient.ActionCreate,
	})
	self.SetStageComplete(ctx, nil)
}
