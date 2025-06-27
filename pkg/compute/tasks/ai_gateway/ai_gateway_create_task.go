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

type AiGatewayCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(AiGatewayCreateTask{})
}

func (self *AiGatewayCreateTask) taskFailed(ctx context.Context, gateway *models.SAiGateway, err error) {
	gateway.SetStatus(ctx, self.UserCred, apis.STATUS_CREATE_FAILED, err.Error())
	db.OpsLog.LogEvent(gateway, db.ACT_CREATE, err.Error(), self.UserCred)
	logclient.AddActionLogWithStartable(self, gateway, logclient.ACT_CREATE, err.Error(), self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *AiGatewayCreateTask) taskComplete(ctx context.Context, gateway *models.SAiGateway) {
	gateway.SetStatus(ctx, self.UserCred, apis.STATUS_AVAILABLE, "")
	self.SetStageComplete(ctx, nil)
}

func (self *AiGatewayCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	gateway := obj.(*models.SAiGateway)

	driver, err := gateway.GetDriver(ctx)
	if err != nil {
		self.taskFailed(ctx, gateway, errors.Wrapf(err, "GetDriver"))
		return
	}

	opts := cloudprovider.AiGatewayCreateOptions{
		Name:                    gateway.Name,
		Authentication:          gateway.Authentication,
		CacheInvalidateOnUpdate: gateway.CacheInvalidateOnUpdate,
		CacheTTL:                gateway.CacheTTL,
		CollectLogs:             gateway.CollectLogs,
		RateLimitingInterval:    gateway.RateLimitingInterval,
		RateLimitingLimit:       gateway.RateLimitingLimit,
		RateLimitingTechnique:   gateway.RateLimitingTechnique,
	}

	iGateway, err := driver.CreateIAiGateway(&opts)
	if err != nil {
		self.taskFailed(ctx, gateway, errors.Wrapf(err, "CreateIAiGateway"))
		return
	}

	err = gateway.SyncWithCloudAiGateway(ctx, self.UserCred, iGateway)
	if err != nil {
		self.taskFailed(ctx, gateway, errors.Wrapf(err, "SyncWithCloudAiGateway"))
		return
	}

	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    gateway,
		Action: notifyclient.ActionCreate,
	})

	self.taskComplete(ctx, gateway)
}
