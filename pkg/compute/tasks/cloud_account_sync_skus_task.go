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
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

var (
	SkuSyncWorkerManager *appsrv.SWorkerManager
)

type CloudAccountSyncSkusTask struct {
	taskman.STask
}

func init() {
	SkuSyncWorkerManager = appsrv.NewWorkerManager("SkuSyncWorkerManager", 8, 1024, false)
	taskman.RegisterTaskAndWorker(CloudAccountSyncSkusTask{}, SkuSyncWorkerManager)
}

func (self *CloudAccountSyncSkusTask) taskFailed(ctx context.Context, account *models.SCloudaccount, err error) {
	account.SetStatus(ctx, self.UserCred, api.CLOUD_PROVIDER_SYNC_STATUS_ERROR, err.Error())
	db.OpsLog.LogEvent(account, db.ACT_SYNC_CLOUD_SKUS, err.Error(), self.GetUserCred())
	logclient.AddActionLogWithStartable(self, account, logclient.ACT_CLOUD_SYNC, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *CloudAccountSyncSkusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	account := obj.(*models.SCloudaccount)

	regions := []models.SCloudregion{}
	if regionId, _ := self.GetParams().GetString("cloudregion_id"); len(regionId) > 0 {
		_region, err := db.FetchById(models.CloudregionManager, regionId)
		if err != nil {
			self.taskFailed(ctx, account, err)
			return
		}

		region := _region.(*models.SCloudregion)
		regions = append(regions, *region)
	} else if providerId, _ := self.GetParams().GetString("cloudprovider_id"); len(providerId) > 0 {
		provider, err := db.FetchById(models.CloudproviderManager, providerId)
		if err != nil {
			self.taskFailed(ctx, account, err)
			return
		}

		_regions := provider.(*models.SCloudprovider).GetCloudproviderRegions()
		for i := range _regions {
			region, _ := _regions[i].GetRegion()
			regions = append(regions, *region)
		}
	} else {
		providers := account.GetEnabledCloudproviders()
		for _, provider := range providers {
			ids := []string{}
			_regions := provider.GetCloudproviderRegions()
			for i := range _regions {
				region, _ := _regions[i].GetRegion()
				if region != nil && !utils.IsInStringArray(region.GetId(), ids) {
					regions = append(regions, *region)
					ids = append(ids, region.GetId())
				}
			}
		}
	}

	res, _ := self.GetParams().GetString("resource")

	type SyncFunc func(ctx context.Context, userCred mcclient.TokenCredential, region *models.SCloudregion, xor bool) compare.SyncResult
	var syncFunc SyncFunc
	for _, region := range regions {
		switch res {
		case models.ServerSkuManager.Keyword():
			syncFunc = models.ServerSkuManager.SyncServerSkus
		case models.ElasticcacheSkuManager.Keyword():
			syncFunc = models.ElasticcacheSkuManager.SyncElasticcacheSkus
		case models.DBInstanceSkuManager.Keyword():
			syncFunc = models.DBInstanceSkuManager.SyncDBInstanceSkus
		case models.NatSkuManager.Keyword():
			result := region.SyncNatSkus(ctx, self.GetUserCred(), false)
			log.Infof("Sync %s %s skus for region %s result: %s", region.Provider, res, region.Name, result.Result())
		case models.NasSkuManager.Keyword():
			result := region.SyncNasSkus(ctx, self.GetUserCred(), false)
			log.Infof("Sync %s %s skus for region %s result: %s", region.Provider, res, region.Name, result.Result())
		}

		if syncFunc != nil {
			result := syncFunc(ctx, self.GetUserCred(), &region, false)
			log.Infof("Sync %s %s skus for region %s result: %s", region.Provider, res, region.Name, result.Result())
		}
	}

	self.SetStageComplete(ctx, nil)
}
