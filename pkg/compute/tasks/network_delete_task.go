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
	"database/sql"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type NetworkDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(NetworkDeleteTask{})
}

func (self *NetworkDeleteTask) taskFailed(ctx context.Context, network *models.SNetwork, err error) {
	network.SetStatus(ctx, self.UserCred, api.NETWORK_STATUS_DELETE_FAILED, err.Error())
	db.OpsLog.LogEvent(network, db.ACT_ALLOCATE_FAIL, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, network, logclient.ACT_DELETE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *NetworkDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	network := obj.(*models.SNetwork)

	network.SetStatus(ctx, self.UserCred, api.NETWORK_STATUS_DELETING, "")
	db.OpsLog.LogEvent(network, db.ACT_DELOCATING, network.GetShortDesc(ctx), self.UserCred)

	inet, err := network.GetINetwork(ctx)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound || errors.Cause(err) == sql.ErrNoRows {
			self.taskComplete(ctx, network)
			return
		}
		self.taskFailed(ctx, network, errors.Wrapf(err, "GetINetwork"))
		return
	}
	err = inet.Delete()
	if err != nil {
		self.taskFailed(ctx, network, errors.Wrapf(err, "inet.Delete"))
		return
	}
	self.taskComplete(ctx, network)
}

func (self *NetworkDeleteTask) taskComplete(ctx context.Context, network *models.SNetwork) {
	network.RealDelete(ctx, self.UserCred)
	logclient.AddActionLogWithStartable(self, network, logclient.ACT_DELETE, "", self.UserCred, true)
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    network,
		Action: notifyclient.ActionDelete,
	})
	self.SetStageComplete(ctx, nil)
}
