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
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/nopanic"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SCloudproviderregionManager struct {
	db.SJointResourceBaseManager
	SSyncableBaseResourceManager
	SCloudregionResourceBaseManager
	SManagedResourceBaseManager
}

var CloudproviderRegionManager *SCloudproviderregionManager

func init() {
	db.InitManager(func() {
		CloudproviderRegionManager = &SCloudproviderregionManager{
			SJointResourceBaseManager: db.NewJointResourceBaseManager(
				SCloudproviderregion{},
				"cloud_provider_regions_tbl",
				"cloudproviderregion",
				"cloudproviderregions",
				CloudproviderManager,
				CloudregionManager,
			),
			SManagedResourceBaseManager: SManagedResourceBaseManager{
				managerIdFieldName: "cloudprovider_id",
			},
		}
		CloudproviderRegionManager.SetVirtualObject(CloudproviderRegionManager)
	})
}

type SCloudproviderregion struct {
	db.SJointResourceBase

	SSyncableBaseResource

	SCloudregionResourceBase `width:"36" charset:"ascii" nullable:"false" list:"domain"`

	// 云订阅ID
	CloudproviderId string `width:"36" charset:"ascii" nullable:"false" list:"domain"`

	//CloudregionId   string `width:"36" charset:"ascii" nullable:"false" list:"domain"`

	Enabled bool `nullable:"false" list:"domain" update:"domain"`

	// SyncIntervalSeconds int `list:"domain"`
	SyncResults jsonutils.JSONObject `list:"domain"`

	LastDeepSyncAt time.Time `list:"domain"`
	LastAutoSyncAt time.Time `list:"domain"`
}

func (manager *SCloudproviderregionManager) GetMasterFieldName() string {
	return "cloudprovider_id"
}

func (manager *SCloudproviderregionManager) GetSlaveFieldName() string {
	return "cloudregion_id"
}

func (self *SCloudproviderregion) GetProvider() *SCloudprovider {
	providerObj, err := CloudproviderManager.FetchById(self.CloudproviderId)
	if err != nil {
		log.Errorf("CloudproviderManager.FetchById fail %s", err)
		return nil
	}
	return providerObj.(*SCloudprovider)
}

func (self *SCloudproviderregion) GetAccount() *SCloudaccount {
	provider := self.GetProvider()
	if provider != nil {
		return provider.GetCloudaccount()
	}
	return nil
}

func (self *SCloudproviderregion) GetRegion() *SCloudregion {
	regionObj, err := CloudregionManager.FetchById(self.CloudregionId)
	if err != nil {
		log.Errorf("CloudregionManager.FetchById(%s) fail %s", self.CloudregionId, err)
		return nil
	}
	return regionObj.(*SCloudregion)
}

func (self *SCloudproviderregion) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (api.CloudproviderregionDetails, error) {
	return api.CloudproviderregionDetails{}, nil
}

func (manager *SCloudproviderregionManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.CloudproviderregionDetails {
	rows := make([]api.CloudproviderregionDetails, len(objs))

	jointRows := manager.SJointResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regionRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	managerIds := make([]string, len(rows))

	for i := range rows {
		rows[i].JointResourceBaseDetails = jointRows[i]
		rows[i].CloudregionResourceInfo = regionRows[i]
		rows[i].Capabilities, _ = objs[i].(*SCloudproviderregion).getCapabilities()
		managerIds[i] = objs[i].(*SCloudproviderregion).CloudproviderId
	}

	managers := make(map[string]SCloudprovider)
	err := db.FetchStandaloneObjectsByIds(CloudproviderManager, managerIds, &managers)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail %s", err)
		return rows
	}

	for i := range rows {
		if manager, ok := managers[managerIds[i]]; ok {
			rows[i].Cloudprovider = manager.Name
			account := manager.GetCloudaccount()
			rows[i].EnableAutoSync = false
			if account != nil {
				rows[i].CloudaccountId = account.Id
				rows[i].Cloudaccount = account.Name
				rows[i].CloudaccountDomainId = account.DomainId
				if account.GetEnabled() && account.EnableAutoSync {
					rows[i].EnableAutoSync = true
				}
				rows[i].SyncIntervalSeconds = account.getSyncIntervalSeconds()
			}
		}
	}

	return rows
}

func (self *SCloudproviderregion) getSyncIntervalSeconds(account *SCloudaccount) int {
	if account == nil {
		account = self.GetAccount()
	}
	return account.getSyncIntervalSeconds()
}

func (manager *SCloudproviderregion) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewForbiddenError("not allow to create")
}

func (self *SCloudproviderregion) ValidateDeleteCondition(ctx context.Context) error {
	return nil
}

func (self *SCloudproviderregion) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SCloudproviderregion) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, self)
}

/*
过滤出指定cloudAccountId || providerIds || cloudAccountId+providerIds关联的region id
*/
func (manager *SCloudproviderregionManager) QueryRelatedRegionIds(cloudAccounts []string, providerIds ...string) *sqlchemy.SSubQuery {
	q := manager.Query("cloudregion_id")

	if len(providerIds) > 0 {
		q = q.Filter(sqlchemy.In(q.Field("cloudprovider_id"), providerIds))
	}

	if len(cloudAccounts) > 0 {
		cpq := CloudaccountManager.Query().SubQuery()
		subcpq := cpq.Query(cpq.Field("id")).Filter(sqlchemy.OR(
			sqlchemy.In(cpq.Field("id"), cloudAccounts),
			sqlchemy.In(cpq.Field("name"), cloudAccounts),
		)).SubQuery()
		providers := CloudproviderManager.Query().SubQuery()
		q = q.Join(providers, sqlchemy.Equals(providers.Field("id"), q.Field("cloudprovider_id")))
		q.Filter(sqlchemy.In(providers.Field("cloudaccount_id"), subcpq))
	}

	return q.Distinct().SubQuery()
}

func (manager *SCloudproviderregionManager) FetchByIds(providerId string, regionId string) *SCloudproviderregion {
	q := manager.Query().Equals("cloudprovider_id", providerId).Equals("cloudregion_id", regionId)
	obj, err := db.NewModelObject(manager)
	if err != nil {
		log.Errorf("db.NewModelObject fail %s", err)
		return nil
	}
	err = q.First(obj)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Errorf("q.First fail %s", err)
		}
		return nil
	}
	return obj.(*SCloudproviderregion)
}

func (manager *SCloudproviderregionManager) FetchByIdsOrCreate(providerId string, regionId string) *SCloudproviderregion {
	cpr := manager.FetchByIds(providerId, regionId)
	if cpr != nil {
		return cpr
	}
	cpr = &SCloudproviderregion{}
	cpr.SetModelManager(manager, cpr)

	cpr.CloudproviderId = providerId
	cpr.CloudregionId = regionId
	cpr.Enabled = true
	cpr.SyncStatus = api.CLOUD_PROVIDER_SYNC_STATUS_IDLE

	err := manager.TableSpec().Insert(context.Background(), cpr)
	if err != nil {
		log.Errorf("insert fail %s", err)
		return nil
	}
	return cpr
}

func (self *SCloudproviderregion) markStartingSync(userCred mcclient.TokenCredential, syncRange *SSyncRange) error {
	if !self.Enabled {
		return fmt.Errorf("Cloudprovider(%s)region(%s) disabled", self.CloudproviderId, self.CloudregionId)
	}
	regionIds := []string{}
	if syncRange != nil {
		regionIds, _ = syncRange.GetRegionIds()
	}
	if syncRange == nil || len(regionIds) == 0 || utils.IsInStringArray(self.CloudregionId, regionIds) {
		_, err := db.Update(self, func() error {
			self.SyncStatus = api.CLOUD_PROVIDER_SYNC_STATUS_QUEUING
			return nil
		})
		if err != nil {
			log.Errorf("Failed to markStartingSync error: %v", err)
			return err
		}
	}
	return nil
}

func (self *SCloudproviderregion) markStartSync(userCred mcclient.TokenCredential) error {
	if !self.Enabled {
		return fmt.Errorf("Cloudprovider(%s)region(%s) disabled", self.CloudproviderId, self.CloudregionId)
	}
	_, err := db.Update(self, func() error {
		self.SyncStatus = api.CLOUD_PROVIDER_SYNC_STATUS_QUEUED
		return nil
	})
	if err != nil {
		log.Errorf("Failed to markStartSync error: %v", err)
		return err
	}
	return nil
}

func (self *SCloudproviderregion) markSyncing(userCred mcclient.TokenCredential) error {
	if !self.Enabled {
		return fmt.Errorf("Cloudprovider(%s)region(%s) disabled", self.CloudproviderId, self.CloudregionId)
	}
	_, err := db.Update(self, func() error {
		self.SyncStatus = api.CLOUD_PROVIDER_SYNC_STATUS_SYNCING
		self.LastSync = timeutils.UtcNow()
		self.LastSyncEndAt = time.Time{}
		return nil
	})
	if err != nil {
		log.Errorf("Failed to markSyncing error: %v", err)
		return err
	}
	return nil
}

func (self *SCloudproviderregion) markEndSync(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, deepSync *bool) error {
	log.Debugf("markEndSync deepSync %v", *deepSync)
	err := self.markEndSyncInternal(userCred, syncResults, deepSync)
	if err != nil {
		return errors.Wrapf(err, "markEndSyncInternal")
	}
	err = self.GetProvider().markEndSyncWithLock(ctx, userCred)
	if err != nil {
		return errors.Wrapf(err, "markEndSyncWithLock")
	}
	return nil
}

func (self *SCloudproviderregion) markEndSyncInternal(userCred mcclient.TokenCredential, syncResults SSyncResultSet, deepSync *bool) error {
	_, err := db.Update(self, func() error {
		self.SyncStatus = api.CLOUD_PROVIDER_SYNC_STATUS_IDLE
		self.LastSyncEndAt = timeutils.UtcNow()
		self.SyncResults = jsonutils.Marshal(syncResults)
		if deepSync != nil && *deepSync {
			self.LastDeepSyncAt = timeutils.UtcNow()
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "db.Update")
	}
	return nil
}

func (self *SCloudproviderregion) cancelStartingSync(userCred mcclient.TokenCredential) error {
	if self.SyncStatus == api.CLOUD_PROVIDER_SYNC_STATUS_QUEUING {
		_, err := db.Update(self, func() error {
			self.SyncStatus = api.CLOUD_PROVIDER_SYNC_STATUS_IDLE
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "db.Update")
		}
	}
	return nil
}

type SSyncResultSet map[string]*compare.SyncResult

func (set SSyncResultSet) Add(manager db.IModelManager, result compare.SyncResult) {
	key := manager.KeywordPlural()
	if _, ok := set[key]; !ok {
		set[key] = &compare.SyncResult{}
	}
	res := set[key]
	res.AddCnt += result.AddCnt
	res.AddErrCnt += result.AddErrCnt
	res.UpdateCnt += result.UpdateCnt
	res.UpdateErrCnt += result.UpdateErrCnt
	res.DelCnt += result.DelCnt
	res.DelErrCnt += result.DelErrCnt
}

func (self *SCloudproviderregion) DoSync(ctx context.Context, userCred mcclient.TokenCredential, syncRange SSyncRange) error {
	syncResults := SSyncResultSet{}

	localRegion := self.GetRegion()
	provider := self.GetProvider()

	self.markSyncing(userCred)

	defer func() {
		err := self.markEndSync(ctx, userCred, syncResults, &syncRange.DeepSync)
		if err != nil {
			log.Errorf("markEndSync for %s(%s) : %v", localRegion.Name, provider.Name, err)
		}
	}()

	driver, err := provider.GetProvider()
	if err != nil {
		log.Errorf("Failed to get driver, connection problem?")
		return err
	}

	if !syncRange.DeepSync {
		log.Debugf("no need to do deep sync, check...")
		intval := self.getSyncIntervalSeconds(nil)
		if self.LastDeepSyncAt.IsZero() || time.Now().Sub(self.LastDeepSyncAt) > time.Hour*24 || (time.Now().Sub(self.LastDeepSyncAt) > time.Duration(intval)*time.Second*8 && rand.Float32() < 0.5) {
			syncRange.DeepSync = true
		}
	}
	log.Debugf("need to do deep sync? ... %v", syncRange.DeepSync)

	if localRegion.isManaged() {
		remoteRegion, err := driver.GetIRegionById(localRegion.ExternalId)
		if err != nil {
			return errors.Wrap(err, "GetIRegionById")
		}
		err = syncPublicCloudProviderInfo(ctx, userCred, syncResults, provider, driver, localRegion, remoteRegion, &syncRange)
	} else {
		err = syncOnPremiseCloudProviderInfo(ctx, userCred, syncResults, provider, driver, &syncRange)
	}

	if err != nil {
		log.Errorf("dosync fail %s", err)
	}

	log.Debugf("dosync result: %s", jsonutils.Marshal(syncResults))

	return err
}

func (self *SCloudproviderregion) getSyncTaskKey() string {
	region := self.GetRegion()
	if len(region.ExternalId) > 0 {
		return region.ExternalId
	} else {
		return self.CloudproviderId
	}
}

func (self *SCloudproviderregion) submitSyncTask(ctx context.Context, userCred mcclient.TokenCredential, syncRange SSyncRange, waitChan chan bool) {
	self.markStartSync(userCred)
	RunSyncCloudproviderRegionTask(ctx, self.getSyncTaskKey(), func() {
		nopanic.Run(func() {
			ctx = context.WithValue(ctx, "provider-region", fmt.Sprintf("%d", self.RowId))
			err := self.DoSync(ctx, userCred, syncRange)
			if err != nil {
				log.Errorf("DoSync faild %v", err)
			}
		})
		if waitChan != nil {
			waitChan <- true
		}
	})
}

func (cpr *SCloudproviderregion) resetAutoSync() {
	_, err := db.Update(cpr, func() error {
		cpr.LastAutoSyncAt = time.Time{}
		return nil
	})
	if err != nil {
		log.Errorf("reset LastAutoSyncAt fail %s", err)
	}
}

func (cpr *SCloudproviderregion) needAutoSync() bool {
	if cpr.needAutoSyncInternal() {
		_, err := db.Update(cpr, func() error {
			cpr.LastAutoSyncAt = time.Now()
			return nil
		})
		if err != nil {
			log.Errorf("set LastAutoSyncAt fail %s", err)
		}
		return true
	}
	return false
}

func (cpr *SCloudproviderregion) needAutoSyncInternal() bool {
	if cpr.LastAutoSyncAt.IsZero() {
		return true
	}
	account := cpr.GetAccount()
	intval := cpr.getSyncIntervalSeconds(account)
	isEmpty := false
	if account.IsOnPremise {
		isEmpty = cpr.isEmptyOnPremise()
	} else {
		isEmpty = cpr.isEmptyPublicCloud()
	}
	if isEmpty {
		intval = intval * 16  // no need to check empty region
		if intval > 24*3600 { // at least once everyday
			intval = 24 * 3600
		}
		region := cpr.GetRegion()
		log.Debugf("empty region %s! no need to check so frequently", region.GetName())
	}
	if time.Now().Sub(cpr.LastSync) > time.Duration(intval)*time.Second {
		return true
	}
	return false
}

func (cpr *SCloudproviderregion) isEmptyOnPremise() bool {
	return cpr.isEmpty(HostManager.KeywordPlural())
}

func (cpr *SCloudproviderregion) isEmptyPublicCloud() bool {
	return cpr.isEmpty(NetworkManager.KeywordPlural())
}

func (cpr *SCloudproviderregion) isEmpty(resKey string) bool {
	if cpr.SyncResults == nil {
		return false
	}
	syncResults := SSyncResultSet{}
	err := cpr.SyncResults.Unmarshal(&syncResults)
	if err != nil {
		return false
	}
	result := syncResults[resKey]
	if result != nil && (result.UpdateCnt > 0 || result.AddCnt > 0) {
		return false
	}
	return true
}

func (cprm *SCloudproviderregionManager) fetchRecordsByCloudproviderId(providerId string) ([]SCloudproviderregion, error) {
	q := cprm.Query().Equals("cloudprovider_id", providerId)
	recs := make([]SCloudproviderregion, 0)
	err := db.FetchModelObjects(cprm, q, &recs)
	if err != nil {
		return nil, err
	}
	return recs, nil
}

func (manager *SCloudproviderregionManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.CloudproviderregionListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SJointResourceBaseManager.ListItemFilter(ctx, q, userCred, query.JointResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SJointResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SSyncableBaseResourceManager.ListItemFilter(ctx, q, userCred, query.SyncableBaseResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSyncableBaseResourceManager.ListItemFilter")
	}
	q, err = manager.SCloudregionResourceBaseManager.ListItemFilter(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}

	if query.Enabled != nil {
		if *query.Enabled {
			q = q.IsTrue("enabled")
		} else {
			q = q.IsFalse("enabled")
		}
	}

	if len(query.Capability) > 0 {
		subq := CloudproviderCapabilityManager.Query().SubQuery()
		q = q.Join(subq, sqlchemy.AND(
			sqlchemy.Equals(q.Field("cloudprovider_id"), subq.Field("cloudprovider_id")),
			sqlchemy.Equals(q.Field("cloudregion_id"), subq.Field("cloudregion_id")),
		))
		q = q.Filter(sqlchemy.In(subq.Field("capability"), query.Capability))
	}

	return q, nil
}

func (manager *SCloudproviderregionManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.CloudproviderregionListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SJointResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.JointResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SJointResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SCloudregionResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SCloudproviderregionManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SJointResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SCloudregionResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (cpr *SCloudproviderregion) setCapabilities(ctx context.Context, userCred mcclient.TokenCredential, capa []string) error {
	return CloudproviderCapabilityManager.setRegionCapabilities(ctx, userCred, cpr.CloudproviderId, cpr.CloudregionId, capa)
}

func (cpr *SCloudproviderregion) removeCapabilities(ctx context.Context, userCred mcclient.TokenCredential) error {
	return CloudproviderCapabilityManager.removeRegionCapabilities(ctx, userCred, cpr.CloudproviderId, cpr.CloudregionId)
}

func (cpr *SCloudproviderregion) getCapabilities() ([]string, error) {
	return CloudproviderCapabilityManager.getRegionCapabilities(cpr.CloudproviderId, cpr.CloudregionId)
}

func (manager *SCloudproviderregionManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SJointResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SJointResourceBaseManager.ListItemExportKeys")
	}

	if keys.ContainsAny(manager.SManagedResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SManagedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemExportKeys")
		}
	}

	if keys.ContainsAny(manager.SCloudregionResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SCloudregionResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}
