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

type AppGatewaySyncStatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(AppGatewaySyncStatusTask{})
}

func (self *AppGatewaySyncStatusTask) taskFailed(ctx context.Context, app *models.SAppGateway, err error) {
	app.SetStatus(self.UserCred, api.APP_GATEWAY_STATUS_UNKNOWN, err.Error())
	db.OpsLog.LogEvent(app, db.ACT_SYNC_STATUS, err, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, app, logclient.ACT_SYNC_STATUS, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *AppGatewaySyncStatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	app := obj.(*models.SAppGateway)
	iApp, err := app.GetICloudAppGateway()
	if err != nil {
		self.taskFailed(ctx, app, errors.Wrapf(err, "app.GetIAppGateway"))
		return
	}
	app.SyncWithCloudAppGateway(ctx, self.UserCred, iApp)
	self.SetStageComplete(ctx, nil)
}
