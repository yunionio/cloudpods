package models

import (
	"context"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/util/timeutils"
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
	//if self.SyncIntervalSeconds > 0 {
	//	return self.SyncIntervalSeconds
	//}
	if account == nil {
		account = self.GetAccount()
	}
	if account.SyncIntervalSeconds > 0 {
		return account.SyncIntervalSeconds
	}
	return options.Options.MinimalSyncIntervalSeconds
}

func (self *SCloudproviderregion) getExtraDetails(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	account := self.GetAccount()
	extra.Add(jsonutils.NewString(account.Id), "cloudaccount_id")
	extra.Add(jsonutils.NewString(account.Name), "cloudaccount")
	if account.EnableAutoSync {
		extra.Add(jsonutils.JSONTrue, "auto_sync")
	} else {
		extra.Add(jsonutils.JSONFalse, "auto_sync")
	}
	extra.Add(jsonutils.NewInt(int64(self.getSyncIntervalSeconds(account))), "sync_interval_seconds")
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
		log.Errorf("q.First fail %s", err)
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
	cpr.SyncStatus = CLOUD_PROVIDER_SYNC_STATUS_IDLE

	err := manager.TableSpec().Insert(cpr)
	if err != nil {
		log.Errorf("insert fail %s", err)
		return nil
	}
	return cpr
}

func (self *SCloudproviderregion) markStartSync(userCred mcclient.TokenCredential) error {
	_, err := db.Update(self, func() error {
		self.SyncStatus = CLOUD_PROVIDER_SYNC_STATUS_QUEUED
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
		self.SyncStatus = CLOUD_PROVIDER_SYNC_STATUS_SYNCING
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

func (self *SCloudproviderregion) markEndSync(userCred mcclient.TokenCredential) error {
	_, err := db.Update(self, func() error {
		self.SyncStatus = CLOUD_PROVIDER_SYNC_STATUS_IDLE
		self.LastSyncEndAt = timeutils.UtcNow()
		return nil
	})
	if err != nil {
		log.Errorf("Fail tp update last_sync %s", err)
		return err
	}
	return nil
}

func (self *SCloudproviderregion) DoSync(ctx context.Context, userCred mcclient.TokenCredential, syncRange *SSyncRange) error {
	err := self.markSyncing(userCred)
	if err != nil {
		log.Errorf("start sync sql fail?? %s", err)
		return err
	}

	localRegion := self.GetRegion()
	provider := self.GetProvider()
	driver, err := provider.GetProvider()
	if err != nil {
		log.Errorf("fail to get driver, connection problem?")
		return err
	}

	if localRegion.isManaged() {
		remoteRegion, err := driver.GetIRegionById(localRegion.ExternalId)
		if err == nil {
			err = syncPublicCloudProviderInfo(ctx, userCred, provider, driver, localRegion, remoteRegion, syncRange)
		}
	} else {
		err = syncOnPremiseCloudProviderInfo(ctx, userCred, provider, driver, syncRange)
	}
	err = self.markEndSync(userCred)
	if err != nil {
		log.Errorf("mark end sync failed...")
		return err
	}
	return nil
}

func (self *SCloudproviderregion) getSyncTaskKey() string {
	region := self.GetRegion()
	if len(region.ExternalId) > 0 {
		return region.ExternalId
	} else {
		return self.CloudproviderId
	}
}

func (self *SCloudproviderregion) submitSyncTask(userCred mcclient.TokenCredential, syncRange *SSyncRange, waitChan chan bool) {
	self.markStartSync(userCred)
	RunSyncCloudproviderRegionTask(self.getSyncTaskKey(), func() {
		self.DoSync(context.Background(), userCred, syncRange)
		if waitChan != nil {
			waitChan <- true
		}
	})
}
