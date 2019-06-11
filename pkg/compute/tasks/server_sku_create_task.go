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

	"yunion.io/x/onecloud/pkg/cloudprovider"

	"yunion.io/x/jsonutils"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type ServerSkuCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ServerSkuCreateTask{})
}

func (self *ServerSkuCreateTask) taskFail(ctx context.Context, sku *models.SServerSku, reason string) {
	sku.SetStatus(self.GetUserCred(), api.SkuStatusCreatFailed, reason)
	db.OpsLog.LogEvent(sku, db.ACT_ALLOCATE_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, sku, logclient.ACT_ALLOCATE, reason, self.UserCred, false)
	notifyclient.NotifySystemError(sku.Id, sku.Name, api.SkuStatusCreatFailed, reason)
	self.SetStageFailed(ctx, reason)
}

func (self *ServerSkuCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	sku := obj.(*models.SServerSku)
	iregion, err := sku.GetIRegion()
	if err != nil {
		self.taskFail(ctx, sku, err.Error())
		return
	}
	isku, err := iregion.CreateISku(&cloudprovider.SServerSku{Name: sku.Name, CpuCoreCount: sku.CpuCoreCount, MemorySizeMB: sku.MemorySizeMB})
	if err != nil {
		self.taskFail(ctx, sku, err.Error())
		return
	}

	err = sku.SyncWithCloudSku(ctx, self.GetUserCred(), isku, nil, nil)
	if err != nil {
		self.taskFail(ctx, sku, err.Error())
		return
	}

	sku.SetStatus(self.UserCred, api.SkuStatusReady, "")
	models.ServerSkuManager.ClearSchedDescCache(true)
	logclient.AddActionLogWithStartable(self, sku, logclient.ACT_ALLOCATE, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}
