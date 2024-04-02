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

type ServerSkuCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ServerSkuCreateTask{})
}

func (self *ServerSkuCreateTask) taskFail(ctx context.Context, sku *models.SServerSku, err error) {
	sku.SetStatus(ctx, self.UserCred, api.SkuStatusCreatFailed, err.Error())
	db.OpsLog.LogEvent(sku, db.ACT_ALLOCATE, err, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, sku, logclient.ACT_ALLOCATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *ServerSkuCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	sku := obj.(*models.SServerSku)
	region, err := sku.GetRegion()
	if err != nil {
		self.taskFail(ctx, sku, errors.Wrapf(err, "GetRegion"))
		return
	}

	providers, err := region.GetCloudproviders()
	if err != nil {
		self.taskFail(ctx, sku, errors.Wrapf(err, "GetCloudproviders"))
		return
	}

	for i := range providers {
		provider := providers[i]
		driver, err := provider.GetProvider(ctx)
		if err != nil {
			self.taskFail(ctx, sku, errors.Wrapf(err, "GetDriver"))
			return
		}

		iRegion, err := driver.GetIRegionById(region.ExternalId)
		if err != nil {
			self.taskFail(ctx, sku, errors.Wrapf(err, "GetIRegionById"))
			return
		}

		opts := cloudprovider.SServerSkuCreateOption{
			Name:             sku.Name,
			CpuCount:         sku.CpuCoreCount,
			VmemSizeMb:       sku.MemorySizeMB,
			SysDiskMinSizeGb: sku.SysDiskMinSizeGB,
			SysDiskMaxSizeGb: sku.SysDiskMaxSizeGB,
		}

		iSku, err := iRegion.CreateISku(&opts)
		if err != nil {
			self.taskFail(ctx, sku, errors.Wrapf(err, "CreateISku"))
			return
		}

		sku.SyncWithPrivateCloudSku(ctx, self.GetUserCred(), iSku)
		self.SetStageComplete(ctx, nil)
		return
	}
	self.taskFail(ctx, sku, errors.Wrapf(cloudprovider.ErrNotFound, "region %s", region.Name))
}
