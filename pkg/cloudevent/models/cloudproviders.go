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

package models

import (
	"context"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudevent/options"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/s3gateway/session"
)

type SCloudproviderManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
}

var CloudproviderManager *SCloudproviderManager

func init() {
	CloudproviderManager = &SCloudproviderManager{
		SEnabledStatusStandaloneResourceBaseManager: db.NewEnabledStatusStandaloneResourceBaseManager(
			SCloudprovider{},
			"cloudproviders_tbl",
			"cloudprovider",
			"cloudproviders",
		),
	}
	CloudproviderManager.SetVirtualObject(CloudproviderManager)
}

type SCloudprovider struct {
	db.SEnabledStatusStandaloneResourceBase

	HealthStatus  string
	SyncStatus    string
	LastSync      time.Time
	LastSyncEndAt time.Time

	AccessUrl string
	Account   string
	Secret    string

	Provider string
}

func (manager *SCloudproviderManager) GetRegionCloudproviders(ctx context.Context, userCred mcclient.TokenCredential) ([]SCloudprovider, error) {
	s := session.GetSession(ctx, userCred)
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString("public"), "cloud_env")
	result, err := modules.Cloudproviders.List(s, params)
	if err != nil {
		return nil, errors.Wrap(err, "modules.Cloudproviders.List")
	}
	providers := []SCloudprovider{}
	for _, _provider := range result.Data {
		provider := SCloudprovider{}
		provider.SetModelManager(manager, &provider)
		err = _provider.Unmarshal(&provider)
		if err != nil {
			return nil, errors.Wrap(err, "_provider.Unmarshal")
		}
		providers = append(providers, provider)
	}
	return providers, nil
}

func (manager *SCloudproviderManager) GetLocalCloudproviders() ([]SCloudprovider, error) {
	dbProviders := []SCloudprovider{}
	q := manager.Query()
	err := db.FetchModelObjects(manager, q, &dbProviders)
	if err != nil {
		return nil, err
	}
	return dbProviders, nil
}

func (manager *SCloudproviderManager) SyncCloudproviders(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	providers, err := manager.GetRegionCloudproviders(ctx, userCred)
	if err != nil {
		log.Errorf("failed to get region cloudproviders: %v", err)
		return
	}

	dbProviders, err := manager.GetLocalCloudproviders()
	if err != nil {
		log.Errorf("failed to get local cloudproviders: %v", err)
		return
	}

	removed := make([]SCloudprovider, 0)
	commondb := make([]SCloudprovider, 0)
	commonext := make([]SCloudprovider, 0)
	added := make([]SCloudprovider, 0)

	err = compare.CompareSets(dbProviders, providers, &removed, &commondb, &commonext, &added)
	if err != nil {
		log.Errorf("compare.CompareSets: %v", err)
		return
	}

	for i := 0; i < len(removed); i++ {
		err = removed[i].Delete(ctx, userCred)
		if err != nil {
			log.Errorf("failed to remove cloudprovider %s(%s) error: %v", removed[i].Name, removed[i].Id, err)
		}
	}

	for i := 0; i < len(commondb); i++ {
		err = commondb[i].syncWithRegionProvider(ctx, userCred, commonext[i])
		if err != nil {
			log.Errorf("failed to sync cloudprovider %s(%s) error: %v", commondb[i].Name, commondb[i].Id, err)
		}
	}

	for i := 0; i < len(added); i++ {
		err = manager.newFromRegionProvider(ctx, userCred, added[i])
		if err != nil {
			log.Errorf("failed to add cloudprovider %s(%s) error: %v", added[i].Name, added[i].Id, err)
		}
	}
}

func (provider *SCloudprovider) syncWithRegionProvider(ctx context.Context, userCred mcclient.TokenCredential, cloudprovider SCloudprovider) error {
	_, err := db.Update(provider, func() error {
		provider.Status = cloudprovider.Status
		provider.Secret = cloudprovider.Secret
		provider.Enabled = cloudprovider.Enabled
		return nil
	})
	return err
}

func (self *SCloudprovider) MarkSyncing(userCred mcclient.TokenCredential) error {
	_, err := db.Update(self, func() error {
		self.SyncStatus = api.CLOUD_PROVIDER_SYNC_STATUS_SYNCING
		self.LastSync = timeutils.UtcNow()
		self.LastSyncEndAt = time.Time{}
		return nil
	})
	if err != nil {
		log.Errorf("Failed to MarkSyncing error: %v", err)
		return err
	}
	return nil
}

func (self *SCloudprovider) MarkEndSync(userCred mcclient.TokenCredential) error {
	_, err := db.Update(self, func() error {
		self.SyncStatus = api.CLOUD_PROVIDER_SYNC_STATUS_IDLE
		self.LastSyncEndAt = timeutils.UtcNow()
		return nil
	})
	if err != nil {
		log.Errorf("Failed to markEndSync error: %v", err)
		return err
	}
	return nil
}

func (manager *SCloudproviderManager) newFromRegionProvider(ctx context.Context, userCred mcclient.TokenCredential, cloudprovider SCloudprovider) error {
	cloudprovider.SyncStatus = api.CLOUD_PROVIDER_SYNC_STATUS_IDLE
	return manager.TableSpec().Insert(&cloudprovider)
}

func (manager *SCloudproviderManager) SyncCloudeventTask(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	cloudproviders := []SCloudprovider{}
	q := manager.Query().IsTrue("enabled").Equals("status", api.CLOUD_PROVIDER_CONNECTED).Equals("sync_status", api.CLOUD_PROVIDER_SYNC_STATUS_IDLE)
	err := db.FetchModelObjects(manager, q, &cloudproviders)
	if err != nil {
		log.Errorf("failed to fetch cloudproviders")
		return
	}
	for _, provider := range cloudproviders {
		err = provider.StartCloudeventSyncTask(ctx, userCred)
		if err != nil {
			log.Errorf("Failed start cloudevent sync task error: %v", err)
		}
	}
	return
}

func (provider *SCloudprovider) StartCloudeventSyncTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	params := jsonutils.NewDict()
	provider.MarkSyncing(userCred)
	task, err := taskman.TaskManager.NewTask(ctx, "CloudeventSyncTask", provider, userCred, params, "", "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SCloudprovider) GetNextTimeRange() (time.Time, time.Time, error) {
	start, end := time.Time{}, time.Time{}
	factory, err := self.GetProviderFactory()
	if err != nil {
		return start, end, errors.Wrap(err, "self.GetProviderFactory")
	}
	q := CloudeventManager.Query().Equals("cloudprovider_id", self.Id).Desc("created_at")
	count, err := q.CountWithError()
	if err != nil {
		return start, end, errors.Wrap(err, "q.CountWithError")
	}
	if count == 0 {
		start = time.Now().AddDate(0, 0, -1*factory.GetMaxCloudEventKeepDays())
	} else {
		provider := SCloudprovider{}
		err = q.First(&provider)
		if err != nil {
			return start, end, errors.Wrap(err, "q.First")
		}
		start = provider.CreatedAt
		if start.Before(time.Now().AddDate(0, 0, factory.GetMaxCloudEventKeepDays()*-1)) {
			start = time.Now().AddDate(0, 0, factory.GetMaxCloudEventKeepDays()*-1)
		}
	}
	if options.Options.OneSyncForHours > factory.GetMaxCloudEventSyncDays()*24 {
		end = start.Add(time.Duration(factory.GetMaxCloudEventSyncDays()*24) * time.Hour)
	} else {
		end = start.Add(time.Duration(options.Options.OneSyncForHours) * time.Hour)
	}
	start = start.Add(time.Second)
	if end.After(time.Now()) {
		end = time.Now()
	}
	return start, end, nil
}

func (provider *SCloudprovider) getPassword() (string, error) {
	return utils.DescryptAESBase64(provider.Id, provider.Secret)
}

func (provider *SCloudprovider) getAccessUrl() string {
	return provider.AccessUrl
}

func (provider *SCloudprovider) GetProviderFactory() (cloudprovider.ICloudProviderFactory, error) {
	return cloudprovider.GetProviderFactory(provider.Provider)
}

func (provider SCloudprovider) GetGlobalId() string {
	return provider.Id
}

func (provider SCloudprovider) GetExternalId() string {
	return provider.Id
}

func (provider *SCloudprovider) GetProvider() (cloudprovider.ICloudProvider, error) {
	if !provider.Enabled {
		return nil, errors.Error("Cloud provider is not enabled")
	}
	accessUrl := provider.getAccessUrl()
	passwd, err := provider.getPassword()
	if err != nil {
		return nil, err
	}
	return cloudprovider.GetProvider(provider.Id, provider.Name, accessUrl, provider.Account, passwd, provider.Provider)
}

func (manager *SCloudproviderManager) InitializeData() error {
	providers := []SCloudprovider{}
	q := manager.Query().NotEquals("sync_status", api.CLOUD_PROVIDER_SYNC_STATUS_IDLE)
	err := db.FetchModelObjects(manager, q, &providers)
	if err != nil {
		return err
	}
	for i := range providers {
		_, err = db.Update(&providers[i], func() error {
			providers[i].SyncStatus = api.CLOUD_PROVIDER_SYNC_STATUS_IDLE
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}
