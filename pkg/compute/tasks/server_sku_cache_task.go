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
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type ServerSkuCacheTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ServerSkuCacheTask{})
}

func (self *ServerSkuCacheTask) taskFailed(ctx context.Context, sku *models.SServerSku, err error) {
	self.SetStageFailed(ctx, err.Error())
}

func (self *ServerSkuCacheTask) getCloudregion() (*models.SCloudregion, error) {
	cloudregionId, _ := self.GetParams().GetString("cloudregion_id")
	if len(cloudregionId) == 0 {
		return nil, fmt.Errorf("Missing cloudregion_id params")
	}
	region, err := models.CloudregionManager.FetchById(cloudregionId)
	if err != nil {
		return nil, errors.Wrap(err, "CloudregionManager.FetchById")
	}
	return region.(*models.SCloudregion), nil
}

func (self *ServerSkuCacheTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	sku := obj.(*models.SServerSku)

	cloudregion, err := self.getCloudregion()
	if err != nil {
		self.taskFailed(ctx, sku, errors.Wrap(err, "self.getCloudregion()"))
		return
	}

	cloudprovider := cloudregion.GetCloudprovider()
	if cloudprovider == nil {
		self.taskFailed(ctx, sku, fmt.Errorf("failed to found cloudprovider for cloudregion %s(%s)", cloudregion.Name, cloudregion.Id))
		return
	}

	provider, err := cloudprovider.GetProvider()
	if err != nil {
		self.taskFailed(ctx, sku, errors.Wrap(err, "cloudprovider.GetProvider"))
		return
	}
	iRegion, err := provider.GetIRegionById(cloudregion.ExternalId)
	if err != nil {
		self.taskFailed(ctx, sku, errors.Wrap(err, "provider.GetIRegionById"))
		return
	}

	iskus, err := iRegion.GetISkus()
	if err != nil {
		self.taskFailed(ctx, sku, errors.Wrap(err, "provider.GetIRegionById"))
		return
	}

	for i := 0; i < len(iskus); i++ {
		if iskus[i].GetName() == sku.Name && iskus[i].GetMemorySizeMB() == sku.MemorySizeMB && iskus[i].GetCpuCoreCount() == sku.CpuCoreCount {
			self.SetStageComplete(ctx, nil)
			return
		}
	}

	err = iRegion.CreateISku(sku.Name, sku.CpuCoreCount, sku.MemorySizeMB)
	if err != nil {
		self.taskFailed(ctx, sku, errors.Wrap(err, "provider.GetIRegionById"))
		return
	}
	self.SetStageComplete(ctx, nil)
	return
}
