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

package ai_gateway

import (
	"context"
	"database/sql"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type AiGatewayDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(AiGatewayDeleteTask{})
}

func (self *AiGatewayDeleteTask) taskFailed(ctx context.Context, gateway *models.SAiGateway, err error) {
	gateway.SetStatus(ctx, self.UserCred, apis.STATUS_DELETE_FAILED, err.Error())
	db.OpsLog.LogEvent(gateway, db.ACT_DELETE, err.Error(), self.UserCred)
	logclient.AddActionLogWithStartable(self, gateway, logclient.ACT_DELETE, err.Error(), self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *AiGatewayDeleteTask) taskComplete(ctx context.Context, gateway *models.SAiGateway) {
	gateway.RealDelete(ctx, self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}

func (self *AiGatewayDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	gateway := obj.(*models.SAiGateway)

	iGateway, err := gateway.GetIAiGateway(ctx)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound || errors.Cause(err) == sql.ErrNoRows {
			self.taskComplete(ctx, gateway)
			return
		}
		self.taskFailed(ctx, gateway, errors.Wrapf(err, "GetIAiGateway"))
		return
	}

	err = iGateway.Delete()
	if err != nil {
		self.taskFailed(ctx, gateway, errors.Wrapf(err, "Delete"))
		return
	}

	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    gateway,
		Action: notifyclient.ActionDelete,
	})

	self.taskComplete(ctx, gateway)
}
