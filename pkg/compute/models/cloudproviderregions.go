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
	"math/rand"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/timeutils"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SCloudproviderregionManager struct {
	db.SJointResourceBaseManager
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
		}
	})
}

type SCloudproviderregion struct {
	db.SJointResourceBase

	SSyncableBaseResource

	CloudproviderId string `width:"36" charset:"ascii" nullable:"false" list:"admin"` // Column(VARCHAR(36, charset='ascii'), nullable=False)
	CloudregionId   string `width:"36" charset:"ascii" nullable:"false" list:"admin"`

	Enabled bool `nullable:"false" list:"admin" update:"admin"`

	// SyncIntervalSeconds int `list:"admin"`
	SyncResults jsonutils.JSONObject `list:"admin"`

	LastDeepSyncAt time.Time `list:"admin"`
}

func (joint *SCloudproviderregion) Master() db.IStandaloneModel {
	return db.JointMaster(joint)
}

func (joint *SCloudproviderregion) Slave() db.IStandaloneModel {
	return db.JointSlave(joint)
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
		log.Errorf("CloudproviderManager.FetchById fail %s", err)
		return nil
	}
	return regionObj.(*SCloudregion)
}

func (self *SCloudproviderregion) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SJointResourceBase.GetCustomizeColumns(ctx, userCred, query)
	extra = db.JointModelExtra(self, extra)
	return self.getExtraDetails(extra)
}

func (self *SCloudproviderregion) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SJointResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	extra = db.JointModelExtra(self, extra)
	return self.getExtraDetails(extra), nil
}

func (self *SCloudproviderregion) getSyncIntervalSeconds(account *SCloudaccount) int {
	if account == nil {
		account = self.GetAccount()
	}
	return account.getSyncIntervalSeconds()
}

func (self *SCloudproviderregion) getExtraDetails(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	account := self.GetAccount()
	if account != nil {
		extra.Add(jsonutils.NewString(account.Id), "cloudaccount_id")
		extra.Add(jsonutils.NewString(account.Name), "cloudaccount")
		if account.EnableAutoSync {
			extra.Add(jsonutils.JSONTrue, "enable_auto_sync")
		} else {
			extra.Add(jsonutils.JSONFalse, "enable_auto_sync")
		}
		extra.Add(jsonutils.NewInt(int64(self.getSyncIntervalSeconds(account))), "sync_interval_seconds")
	}
	return extra
}

func (manager *SCloudproviderregion) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
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
	cpr.SetModelManager(manager)

	cpr.CloudproviderId = providerId
	cpr.CloudregionId = regionId
	cpr.Enabled = true
	cpr.SyncStatus = compute.CLOUD_PROVIDER_SYNC_STATUS_IDLE

	err := manager.TableSpec().Insert(cpr)
	if err != nil {
		log.Errorf("insert fail %s", err)
		return nil
	}
	return cpr
}

func (self *SCloudproviderregion) markStartSync(userCred mcclient.TokenCredential) error {
	_, err := db.Update(self, func() error {
		self.SyncStatus = compute.CLOUD_PROVIDER_SYNC_STATUS_QUEUED
		return nil
	})
	if err != nil {
		log.Errorf("Fail tp update last_sync %s", err)
		return err
	}
	return nil
}

func (self *SCloudproviderregion) markSyncing(userCred mcclient.TokenCredential) error {
	_, err := db.Update(self, func() error {
		self.SyncStatus = compute.CLOUD_PROVIDER_SYNC_STATUS_SYNCING
		self.LastSync = timeutils.UtcNow()
		self.LastSyncEndAt = time.Time{}
		return nil
	})
	if err != nil {
		log.Errorf("Fail tp update last_sync %s", err)
		return err
	}
	return nil
}

func (self *SCloudproviderregion) markEndSync(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, deepSync *bool) error {
	log.Debugf("markEndSync deepSync %v", *deepSync)
	err := self.markEndSyncInternal(userCred, syncResults, deepSync)
	if err != nil {
		return err
	}
	err = self.GetProvider().markEndSyncWithLock(ctx, userCred)
	if err != nil {
		return err
	}
	return nil
}

func (self *SCloudproviderregion) markEndSyncInternal(userCred mcclient.TokenCredential, syncResults SSyncResultSet, deepSync *bool) error {
	_, err := db.Update(self, func() error {
		self.SyncStatus = compute.CLOUD_PROVIDER_SYNC_STATUS_IDLE
		self.LastSyncEndAt = timeutils.UtcNow()
		self.SyncResults = jsonutils.Marshal(syncResults)
		if *deepSync {
			self.LastDeepSyncAt = timeutils.UtcNow()
		}
		return nil
	})
	if err != nil {
		log.Errorf("Fail tp update last_sync %s", err)
		return err
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

	self.markSyncing(userCred)
	defer self.markEndSync(ctx, userCred, syncResults, &syncRange.DeepSync)

	localRegion := self.GetRegion()
	provider := self.GetProvider()
	driver, err := provider.GetProvider()
	if err != nil {
		log.Errorf("fail to get driver, connection problem?")
		return err
	}

	log.Debugf("need to do deep sync ... %v", syncRange.DeepSync)
	if !syncRange.DeepSync {
		intval := self.getSyncIntervalSeconds(nil)
		if self.LastDeepSyncAt.IsZero() || time.Now().Sub(self.LastDeepSyncAt) > time.Hour*24 || (time.Now().Sub(self.LastDeepSyncAt) > time.Duration(intval)*time.Second*8 && rand.Float32() < 0.5) {
			syncRange.DeepSync = true
		}
	}
	log.Debugf("no need to do deep sync ... %v", syncRange.DeepSync)

	if localRegion.isManaged() {
		remoteRegion, err := driver.GetIRegionById(localRegion.ExternalId)
		if err == nil {
			err = syncPublicCloudProviderInfo(ctx, userCred, syncResults, provider, driver, localRegion, remoteRegion, &syncRange)
		}
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

func (self *SCloudproviderregion) submitSyncTask(userCred mcclient.TokenCredential, syncRange SSyncRange, waitChan chan bool) {
	self.markStartSync(userCred)
	RunSyncCloudproviderRegionTask(self.getSyncTaskKey(), func() {
		self.DoSync(context.Background(), userCred, syncRange)
		if waitChan != nil {
			waitChan <- true
		}
	})
}

func (cpr *SCloudproviderregion) needAutoSync() bool {
	if cpr.LastSyncEndAt.IsZero() {
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
	if time.Now().Sub(cpr.LastSyncEndAt) > time.Duration(intval)*time.Second && rand.Float32() < 0.6 {
		// add randomness
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
