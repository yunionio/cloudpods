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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudevent/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/s3gateway/session"
)

type SCloudproviderManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
	db.SProjectizedResourceBaseManager
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
	db.SProjectizedResourceBase

	SyncStatus     string
	LastSync       time.Time
	LastSyncEndAt  time.Time
	LastSyncTimeAt time.Time

	Provider string `width:"64" charset:"ascii" list:"domain"`
	Brand    string `width:"64" charset:"ascii" list:"domain"`
}

func (manager *SCloudproviderManager) GetRegionCloudproviders(ctx context.Context, userCred mcclient.TokenCredential) ([]SCloudprovider, error) {
	s := session.GetSession(ctx, userCred)

	data := []jsonutils.JSONObject{}
	offset := int64(0)
	params := jsonutils.NewDict()
	params.Set("scope", jsonutils.NewString("system"))
	params.Set("limit", jsonutils.NewInt(1024))
	for {
		params.Set("offset", jsonutils.NewInt(offset))
		result, err := modules.Cloudproviders.List(s, params)
		if err != nil {
			return nil, errors.Wrap(err, "modules.Cloudproviders.List")
		}
		data = append(data, result.Data...)
		if len(data) >= result.Total {
			break
		}
		offset += 1024
	}

	providers := []SCloudprovider{}
	err := jsonutils.Update(&providers, data)
	if err != nil {
		return nil, errors.Wrap(err, "jsonutils.Update")
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

func (manager *SCloudproviderManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SEnabledStatusStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (manager *SCloudproviderManager) syncCloudproviders(ctx context.Context, userCred mcclient.TokenCredential) compare.SyncResult {
	result := compare.SyncResult{}
	providers, err := manager.GetRegionCloudproviders(ctx, userCred)
	if err != nil {
		result.Error(errors.Wrap(err, "GetRegionCloudproviders"))
		return result
	}

	dbProviders, err := manager.GetLocalCloudproviders()
	if err != nil {
		result.Error(errors.Wrap(err, "GetLocalCloudproviders"))
		return result
	}

	removed := make([]SCloudprovider, 0)
	commondb := make([]SCloudprovider, 0)
	commonext := make([]SCloudprovider, 0)
	added := make([]SCloudprovider, 0)

	err = compare.CompareSets(dbProviders, providers, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(errors.Wrap(err, "CompareSets"))
		return result
	}

	for i := 0; i < len(removed); i++ {
		err = removed[i].Delete(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}

	for i := 0; i < len(commondb); i++ {
		err = commondb[i].syncWithRegionProvider(ctx, userCred, commonext[i])
		if err != nil {
			result.UpdateError(err)
			continue
		}
		result.Update()
	}

	for i := 0; i < len(added); i++ {
		err = manager.newFromRegionProvider(ctx, userCred, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}
	return result
}

func (manager *SCloudproviderManager) SyncCloudproviders(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	result := manager.syncCloudproviders(ctx, userCred)
	info := result.Result()
	log.Infof("sync cloudproviders result: %s", info)
}

func (provider *SCloudprovider) syncWithRegionProvider(ctx context.Context, userCred mcclient.TokenCredential, cloudprovider SCloudprovider) error {
	_, err := db.Update(provider, func() error {
		provider.Status = cloudprovider.Status
		provider.Enabled = cloudprovider.Enabled
		provider.Brand = cloudprovider.Brand
		provider.ProjectId = cloudprovider.ProjectId
		provider.DomainId = cloudprovider.DomainId
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
		return errors.Wrap(err, "db.Update")
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
		return errors.Wrap(err, "db.Update")
	}
	return nil
}

func (self *SCloudprovider) SetLastSyncTimeAt(userCred mcclient.TokenCredential, last time.Time) error {
	_, err := db.Update(self, func() error {
		self.LastSyncTimeAt = last
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "db.Update")
	}
	return nil
}

func (manager *SCloudproviderManager) newFromRegionProvider(ctx context.Context, userCred mcclient.TokenCredential, cloudprovider SCloudprovider) error {
	cloudprovider.SyncStatus = api.CLOUD_PROVIDER_SYNC_STATUS_IDLE
	return manager.TableSpec().Insert(ctx, &cloudprovider)
}

func (manager *SCloudproviderManager) syncCloudeventTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	cloudproviders := []SCloudprovider{}
	q := manager.Query().IsTrue("enabled").Equals("status", api.CLOUD_PROVIDER_CONNECTED).Equals("sync_status", api.CLOUD_PROVIDER_SYNC_STATUS_IDLE)
	err := db.FetchModelObjects(manager, q, &cloudproviders)
	if err != nil {
		return errors.Wrap(err, "db.FetchModelObjects")
	}
	for i := range cloudproviders {
		err = cloudproviders[i].StartCloudeventSyncTask(ctx, userCred)
		if err != nil {
			log.Errorf("Failed start cloudevent sync task for cloudprovider %s (%s) error: %v", cloudproviders[i].Name, cloudproviders[i].Id, err)
		}
	}
	return nil
}

func (manager *SCloudproviderManager) SyncCloudeventTask(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	if options.Options.DisableSyncCloudEvent {
		return
	}
	err := manager.syncCloudeventTask(ctx, userCred)
	if err != nil {
		log.Errorf("syncCloudeventTask error: %v", err)
	}
}

func (provider *SCloudprovider) StartCloudeventSyncTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	params := jsonutils.NewDict()
	task, err := taskman.TaskManager.NewTask(ctx, "CloudeventSyncTask", provider, userCred, params, "", "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	provider.MarkSyncing(userCred)
	task.ScheduleRun(nil)
	return nil
}

func (self *SCloudprovider) GetNextTimeRange() (time.Time, time.Time, error) {
	start, end := time.Time{}, time.Time{}
	factory, err := self.GetProviderFactory()
	if err != nil {
		return start, end, errors.Wrap(err, "self.GetProviderFactory")
	}
	if !self.LastSyncTimeAt.IsZero() {
		start = self.LastSyncTimeAt
	} else {
		start = time.Now().AddDate(0, 0, -1*factory.GetMaxCloudEventKeepDays())
	}

	// 避免cloudevent过长时间未运行，再次运行时记录的最后一条时间距离现在间隔太长
	if start.Before(time.Now().AddDate(0, 0, factory.GetMaxCloudEventKeepDays()*-1)) {
		start = time.Now().AddDate(0, 0, factory.GetMaxCloudEventKeepDays()*-1)
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

func (provider *SCloudprovider) GetProviderFactory() (cloudprovider.ICloudProviderFactory, error) {
	return cloudprovider.GetProviderFactory(provider.Provider)
}

func (provider SCloudprovider) GetGlobalId() string {
	return provider.Id
}

func (provider SCloudprovider) GetExternalId() string {
	return provider.Id
}

func (self *SCloudprovider) GetProvider() (cloudprovider.ICloudProvider, error) {
	ctx := context.Background()
	s := auth.GetAdminSession(ctx, options.Options.Region)
	return modules.Cloudproviders.GetProvider(ctx, s, self.Id)
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
