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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SCloudregionManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
}

var CloudregionManager *SCloudregionManager

func init() {
	CloudregionManager = &SCloudregionManager{
		SEnabledStatusStandaloneResourceBaseManager: db.NewEnabledStatusStandaloneResourceBaseManager(
			SCloudregion{},
			"cloudregions_tbl",
			"cloudregion",
			"cloudregions",
		),
	}
	CloudregionManager.SetVirtualObject(CloudregionManager)
}

type SCloudregion struct {
	db.SEnabledStatusStandaloneResourceBase
	SManagedResourceBase
	db.SExternalizedResourceBase

	cloudprovider.SGeographicInfo

	Environment string `width:"32" charset:"ascii" list:"user"`
	Provider    string `width:"64" charset:"ascii" list:"user" nullable:"false" default:"OneCloud"`
}

func (manager *SCloudregionManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (self *SCloudregionManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, self)
}

func (self *SCloudregion) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SCloudregion) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SCloudregion) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
}

func (self *SCloudregion) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	idstr, _ := data.GetString("id")
	if len(idstr) > 0 {
		self.Id = idstr
	}
	return nil
}

func (self *SCloudregion) ValidateDeleteCondition(ctx context.Context) error {
	zoneCnt, err := self.GetZoneCount()
	if err != nil {
		return httperrors.NewInternalServerError("GetZoneCount fail %s", err)
	}
	vpcCnt, err := self.GetVpcCount()
	if err != nil {
		return httperrors.NewInternalServerError("GetVpcCount fail %s", err)
	}
	if zoneCnt > 0 || vpcCnt > 0 {
		return httperrors.NewNotEmptyError("not empty cloud region")
	}
	if self.Id == api.DEFAULT_REGION_ID {
		return httperrors.NewProtectedResourceError("not allow to delete default cloud region")
	}
	return self.SEnabledStatusStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SCloudregion) GetZoneQuery() *sqlchemy.SQuery {
	zones := ZoneManager.Query()
	if self.Id == api.DEFAULT_REGION_ID {
		return zones.Filter(sqlchemy.OR(sqlchemy.IsNull(zones.Field("cloudregion_id")),
			sqlchemy.IsEmpty(zones.Field("cloudregion_id")),
			sqlchemy.Equals(zones.Field("cloudregion_id"), self.Id)))
	} else {
		return zones.Equals("cloudregion_id", self.Id)
	}
}

func (self *SCloudregion) GetZoneCount() (int, error) {
	return self.GetZoneQuery().CountWithError()
}

func (self *SCloudregion) GetZones() ([]SZone, error) {
	q := self.GetZoneQuery()
	zones := []SZone{}
	err := db.FetchModelObjects(ZoneManager, q, &zones)
	if err != nil {
		return nil, err
	}
	return zones, nil
}

func (self *SCloudregion) GetGuestCount() (int, error) {
	return self.getGuestCountInternal(false)
}

func (self *SCloudregion) GetGuestIncrementCount() (int, error) {
	return self.getGuestCountInternal(true)
}

func (self *SCloudregion) GetNetworkInterfaces() ([]SNetworkInterface, error) {
	interfaces := []SNetworkInterface{}
	q := NetworkInterfaceManager.Query().Equals("cloudregion_id", self.Id)
	err := db.FetchModelObjects(NetworkInterfaceManager, q, &interfaces)
	if err != nil {
		return nil, err
	}
	return interfaces, nil
}

func (self *SCloudregion) GetDBInstances(provider *SCloudprovider) ([]SDBInstance, error) {
	instances := []SDBInstance{}
	q := DBInstanceManager.Query().Equals("cloudregion_id", self.Id)
	if provider != nil {
		q = q.Equals("manager_id", provider.Id)
	}
	err := db.FetchModelObjects(DBInstanceManager, q, &instances)
	if err != nil {
		return nil, errors.Wrapf(err, "FetchModelObjects for region %s", self.Id)
	}
	return instances, nil
}

func (self *SCloudregion) GetDBInstanceBackups(provider *SCloudprovider, instance *SDBInstance) ([]SDBInstanceBackup, error) {
	backups := []SDBInstanceBackup{}
	q := DBInstanceBackupManager.Query().Equals("cloudregion_id", self.Id)
	if provider != nil {
		q = q.Equals("manager_id", provider.Id)
	}
	if instance != nil {
		q = q.Equals("dbinstance_id", instance.Id)
	}
	err := db.FetchModelObjects(DBInstanceBackupManager, q, &backups)
	if err != nil {
		return nil, errors.Wrapf(err, "FetchModelObjects for region %s", self.Id)
	}
	return backups, nil
}

func (self *SCloudregion) GetElasticcaches(provider *SCloudprovider) ([]SElasticcache, error) {
	instances := []SElasticcache{}
	// .IsFalse("pending_deleted")
	q := ElasticcacheManager.Query().Equals("cloudregion_id", self.Id)
	if provider != nil {
		q = q.Equals("manager_id", provider.Id)
	}
	err := db.FetchModelObjects(ElasticcacheManager, q, &instances)
	if err != nil {
		return nil, errors.Wrapf(err, "GetElasticcaches for region %s", self.Id)
	}

	return instances, nil
}

func (self *SCloudregion) getGuestCountInternal(increment bool) (int, error) {
	zoneTable := ZoneManager.Query("id")
	if self.Id == api.DEFAULT_REGION_ID {
		zoneTable = zoneTable.Filter(sqlchemy.OR(sqlchemy.IsNull(zoneTable.Field("cloudregion_id")),
			sqlchemy.IsEmpty(zoneTable.Field("cloudregion_id")),
			sqlchemy.Equals(zoneTable.Field("cloudregion_id"), self.Id)))
	} else {
		zoneTable = zoneTable.Equals("cloudregion_id", self.Id)
	}
	sq := HostManager.Query("id").In("zone_id", zoneTable)
	query := GuestManager.Query().In("host_id", sq)
	if increment {
		year, month, _ := time.Now().UTC().Date()
		startOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
		query.GE("created_at", startOfMonth)
	}
	return query.CountWithError()
}

func (self *SCloudregion) GetVpcCount() (int, error) {
	vpcs := VpcManager.Query()
	if self.Id == api.DEFAULT_REGION_ID {
		return vpcs.Filter(sqlchemy.OR(sqlchemy.IsNull(vpcs.Field("cloudregion_id")),
			sqlchemy.IsEmpty(vpcs.Field("cloudregion_id")),
			sqlchemy.Equals(vpcs.Field("cloudregion_id"), self.Id))).CountWithError()
	} else {
		return vpcs.Equals("cloudregion_id", self.Id).CountWithError()
	}
}

func (self *SCloudregion) GetDriver() IRegionDriver {
	provider := self.Provider
	if len(provider) == 0 {
		provider = api.CLOUD_PROVIDER_ONECLOUD
	}
	if !utils.IsInStringArray(provider, api.CLOUD_PROVIDERS) {
		log.Fatalf("Unsupported region provider %s", provider)
	}
	return GetRegionDriver(provider)
}

func (self *SCloudregion) getMoreDetails(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	return db.FetchModelExtraCountProperties(self, extra)
}

func (self *SCloudregion) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SEnabledStatusStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return self.getMoreDetails(extra)
}

func (self *SCloudregion) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SEnabledStatusStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return self.getMoreDetails(extra), nil
}

func (self *SCloudregion) GetSkus() ([]SServerSku, error) {
	skus := []SServerSku{}
	q := ServerSkuManager.Query().Equals("cloudregion_id", self.Id)
	err := db.FetchModelObjects(ServerSkuManager, q, &skus)
	if err != nil {
		return nil, err
	}
	return skus, nil
}

func (manager *SCloudregionManager) GetRegionByExternalIdPrefix(prefix string) ([]SCloudregion, error) {
	regions := make([]SCloudregion, 0)
	q := manager.Query().Startswith("external_id", prefix)
	err := db.FetchModelObjects(manager, q, &regions)
	if err != nil {
		log.Errorf("%s", err)
		return nil, err
	}
	return regions, nil
}

func (manager *SCloudregionManager) GetRegionByProvider(provider string) ([]SCloudregion, error) {
	regions := make([]SCloudregion, 0)
	q := manager.Query().Equals("provider", provider)
	err := db.FetchModelObjects(manager, q, &regions)
	if err != nil {
		log.Errorf("%s", err)
		return nil, err
	}
	return regions, nil
}

func (manager *SCloudregionManager) getCloudregionsByProviderId(providerId string) ([]SCloudregion, error) {
	regions := []SCloudregion{}
	err := fetchByManagerId(manager, providerId, &regions)
	if err != nil {
		return nil, err
	}
	return regions, nil
}

func (manager *SCloudregionManager) SyncRegions(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	cloudProvider *SCloudprovider,
	externalIdPrefix string,
	regions []cloudprovider.ICloudRegion,
) (
	[]SCloudregion,
	[]cloudprovider.ICloudRegion,
	[]SCloudproviderregion,
	compare.SyncResult,
) {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	syncResult := compare.SyncResult{}
	localRegions := make([]SCloudregion, 0)
	remoteRegions := make([]cloudprovider.ICloudRegion, 0)
	cloudProviderRegions := make([]SCloudproviderregion, 0)

	dbRegions, err := manager.GetRegionByExternalIdPrefix(externalIdPrefix)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, nil, syncResult
	}
	log.Debugf("Region with provider %s %d", externalIdPrefix, len(dbRegions))

	removed := make([]SCloudregion, 0)
	commondb := make([]SCloudregion, 0)
	commonext := make([]cloudprovider.ICloudRegion, 0)
	added := make([]cloudprovider.ICloudRegion, 0)
	err = compare.CompareSets(dbRegions, regions, &removed, &commondb, &commonext, &added)
	if err != nil {
		log.Errorf("compare regions fail %s", err)
		syncResult.Error(err)
		return nil, nil, nil, syncResult
	}
	for i := 0; i < len(removed); i += 1 {
		err = removed[i].syncRemoveCloudRegion(ctx, userCred, cloudProvider)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}
	for i := 0; i < len(commondb); i += 1 {
		// update
		err = commondb[i].syncWithCloudRegion(ctx, userCred, commonext[i], cloudProvider)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncMetadata(ctx, userCred, &commondb[i], commonext[i])
			cpr := CloudproviderRegionManager.FetchByIdsOrCreate(cloudProvider.Id, commondb[i].Id)
			cloudProviderRegions = append(cloudProviderRegions, *cpr)
			localRegions = append(localRegions, commondb[i])
			remoteRegions = append(remoteRegions, commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i += 1 {
		new, err := manager.newFromCloudRegion(ctx, userCred, added[i], cloudProvider)
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncMetadata(ctx, userCred, new, added[i])
			cpr := CloudproviderRegionManager.FetchByIdsOrCreate(cloudProvider.Id, new.Id)
			cloudProviderRegions = append(cloudProviderRegions, *cpr)
			localRegions = append(localRegions, *new)
			remoteRegions = append(remoteRegions, added[i])
			syncResult.Add()
		}
	}
	return localRegions, remoteRegions, cloudProviderRegions, syncResult
}

func (self *SCloudregion) syncRemoveCloudRegion(ctx context.Context, userCred mcclient.TokenCredential, cloudProvider *SCloudprovider) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	// err := self.ValidateDeleteCondition(ctx)
	// if err == nil {
	// 	err = self.Delete(ctx, userCred)
	// }

	err := self.SetStatus(userCred, api.CLOUD_REGION_STATUS_OUTOFSERVICE, "Out of sync")
	if err == nil {
		_, err = self.PerformDisable(ctx, userCred, nil, nil)
	}

	cpr := CloudproviderRegionManager.FetchByIds(cloudProvider.Id, self.Id)
	if cpr != nil {
		err = cpr.Detach(ctx, userCred)
	}

	return err
}

func (self *SCloudregion) syncWithCloudRegion(ctx context.Context, userCred mcclient.TokenCredential, cloudRegion cloudprovider.ICloudRegion, provider *SCloudprovider) error {
	factory, err := provider.GetProviderFactory()
	if err != nil {
		return err
	}

	diff, err := db.UpdateWithLock(ctx, self, func() error {
		self.Status = cloudRegion.GetStatus()
		self.SGeographicInfo = cloudRegion.GetGeographicInfo()
		self.Provider = cloudRegion.GetProvider()
		self.Environment = cloudRegion.GetCloudEnv()

		self.IsEmulated = cloudRegion.IsEmulated()

		if !factory.IsPublicCloud() && !factory.IsOnPremise() {
			self.ManagerId = provider.Id
		}

		return nil
	})
	if err != nil {
		log.Errorf("syncWithCloudRegion %s", err)
		return err
	}
	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return nil
}

func (manager *SCloudregionManager) newFromCloudRegion(ctx context.Context, userCred mcclient.TokenCredential, cloudRegion cloudprovider.ICloudRegion, provider *SCloudprovider) (*SCloudregion, error) {
	region := SCloudregion{}
	region.SetModelManager(manager, &region)

	newName, err := db.GenerateName(manager, nil, cloudRegion.GetName())
	if err != nil {
		return nil, err
	}
	region.ExternalId = cloudRegion.GetGlobalId()
	region.Name = newName
	region.SGeographicInfo = cloudRegion.GetGeographicInfo()
	region.Status = cloudRegion.GetStatus()
	region.Enabled = true
	region.Provider = cloudRegion.GetProvider()
	region.Environment = cloudRegion.GetCloudEnv()

	region.IsEmulated = cloudRegion.IsEmulated()

	factory, err := provider.GetProviderFactory()
	if err != nil {
		return nil, err
	}
	if !factory.IsOnPremise() && !factory.IsPublicCloud() {
		region.ManagerId = provider.Id
	}

	err = manager.TableSpec().Insert(&region)
	if err != nil {
		log.Errorf("newFromCloudRegion fail %s", err)
		return nil, err
	}
	db.OpsLog.LogEvent(&region, db.ACT_CREATE, region.GetShortDesc(ctx), userCred)
	return &region, nil
}

func (self *SCloudregion) AllowPerformDefaultVpc(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "default-vpc")
}

func (self *SCloudregion) PerformDefaultVpc(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	vpcs, err := VpcManager.getVpcsByRegion(self, nil)
	if err != nil {
		return nil, err
	}
	vpcStr, _ := data.GetString("vpc")
	if len(vpcStr) == 0 {
		return nil, httperrors.NewMissingParameterError("vpc")
	}
	findVpc := false
	for _, vpc := range vpcs {
		if vpc.Id == vpcStr || vpc.Name == vpcStr {
			findVpc = true
			break
		}
	}
	if !findVpc {
		return nil, httperrors.NewResourceNotFoundError("VPC %s not found", vpcStr)
	}
	for _, vpc := range vpcs {
		if vpc.Id == vpcStr || vpc.Name == vpcStr {
			err = vpc.setDefault(true)
		} else {
			err = vpc.setDefault(false)
		}
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (manager *SCloudregionManager) FetchRegionById(id string) *SCloudregion {
	obj, err := manager.FetchById(id)
	if err != nil {
		log.Errorf("region %s %s", id, err)
		return nil
	}
	return obj.(*SCloudregion)
}

func (manager *SCloudregionManager) InitializeData() error {
	// check if default region exists
	obj, err := manager.FetchById(api.DEFAULT_REGION_ID)
	if err != nil {
		if err == sql.ErrNoRows {
			defRegion := SCloudregion{}
			defRegion.Id = api.DEFAULT_REGION_ID
			defRegion.Name = "Default"
			defRegion.Enabled = true
			defRegion.Description = "Default Region"
			defRegion.Status = api.CLOUD_REGION_STATUS_INSERVER
			defRegion.Provider = api.CLOUD_PROVIDER_ONECLOUD
			err := manager.TableSpec().Insert(&defRegion)
			return err
		}
		return err
	}
	if region := obj.(*SCloudregion); region.Provider != api.CLOUD_PROVIDER_ONECLOUD {
		_, err := db.Update(region, func() error {
			region.Provider = api.CLOUD_PROVIDER_ONECLOUD
			return nil
		})
		return err
	}
	return nil
}

func getCloudRegionIdByDomainId(domainId string) *sqlchemy.SSubQuery {
	accounts := CloudaccountManager.Query().SubQuery()
	cloudproviderregions := CloudproviderRegionManager.Query().SubQuery()
	providers := CloudproviderManager.Query().SubQuery()

	// not managed region
	q1 := CloudregionManager.Query("id").Equals("provider", api.CLOUD_PROVIDER_ONECLOUD)

	// managed region
	q2 := cloudproviderregions.Query(cloudproviderregions.Field("cloudregion_id", "id"))
	q2 = q2.Join(providers, sqlchemy.Equals(providers.Field("id"), cloudproviderregions.Field("cloudprovider_id")))
	q2 = q2.Join(accounts, sqlchemy.Equals(providers.Field("cloudaccount_id"), accounts.Field("id")))
	q2 = q2.Filter(sqlchemy.OR(
		sqlchemy.AND(
			sqlchemy.Equals(providers.Field("domain_id"), domainId),
			sqlchemy.Equals(accounts.Field("share_mode"), api.CLOUD_ACCOUNT_SHARE_MODE_PROVIDER_DOMAIN),
		),
		sqlchemy.Equals(accounts.Field("share_mode"), api.CLOUD_ACCOUNT_SHARE_MODE_SYSTEM),
		sqlchemy.AND(
			sqlchemy.Equals(accounts.Field("domain_id"), domainId),
			sqlchemy.Equals(accounts.Field("share_mode"), api.CLOUD_ACCOUNT_SHARE_MODE_ACCOUNT_DOMAIN),
		),
	))

	return sqlchemy.Union(q1, q2).Query().SubQuery()
}

func queryCloudregionIdsByProviders(providerField string, providerStrs []string) *sqlchemy.SQuery {
	q := CloudregionManager.Query("id")
	oneCloud, providers := splitProviders(providerStrs)
	conds := make([]sqlchemy.ICondition, 0)
	if len(providers) > 0 {
		cloudproviders := CloudproviderManager.Query().SubQuery()
		cloudaccounts := CloudaccountManager.Query().SubQuery()
		subq := CloudproviderRegionManager.Query("cloudregion_id")
		subq = subq.Join(cloudproviders, sqlchemy.Equals(subq.Field("cloudprovider_id"), cloudproviders.Field("id")))
		subq = subq.Join(cloudaccounts, sqlchemy.Equals(cloudproviders.Field("cloudaccount_id"), cloudaccounts.Field("id")))
		subq = subq.Filter(sqlchemy.In(cloudaccounts.Field(providerField), providers))
		conds = append(conds, sqlchemy.In(q.Field("id"), subq.SubQuery()))
	}
	if oneCloud {
		conds = append(conds, sqlchemy.Equals(q.Field("provider"), api.CLOUD_PROVIDER_ONECLOUD))
	}
	if len(conds) == 1 {
		q = q.Filter(conds[0])
	} else if len(conds) == 2 {
		q = q.Filter(sqlchemy.OR(conds...))
	}
	return q
}

func (manager *SCloudregionManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	providerStrs := jsonutils.GetQueryStringArray(query, "provider")
	if len(providerStrs) > 0 {
		query.(*jsonutils.JSONDict).Remove("provider")
		subq := queryCloudregionIdsByProviders("provider", providerStrs)
		q = q.In("id", subq.SubQuery())
	}

	brandStrs := jsonutils.GetQueryStringArray(query, "brand")
	if len(brandStrs) > 0 {
		query.(*jsonutils.JSONDict).Remove("brand")
		subq := queryCloudregionIdsByProviders("brand", brandStrs)
		q = q.In("id", subq.SubQuery())
	}

	q, err := manager.SEnabledStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}

	cloudEnvStr, _ := query.GetString("cloud_env")
	if cloudEnvStr == api.CLOUD_ENV_PUBLIC_CLOUD || jsonutils.QueryBoolean(query, "is_public", false) || jsonutils.QueryBoolean(query, "public_cloud", false) {
		q = q.In("provider", cloudprovider.GetPublicProviders())
	}

	if cloudEnvStr == api.CLOUD_ENV_PRIVATE_CLOUD || jsonutils.QueryBoolean(query, "is_private", false) || jsonutils.QueryBoolean(query, "private_cloud", false) {
		q = q.In("provider", cloudprovider.GetPrivateProviders())
	}

	if cloudEnvStr == api.CLOUD_ENV_ON_PREMISE || jsonutils.QueryBoolean(query, "is_on_premise", false) {
		q = q.Filter(sqlchemy.OR(
			sqlchemy.In(q.Field("provider"), cloudprovider.GetOnPremiseProviders()),
			sqlchemy.Equals(q.Field("provider"), api.CLOUD_PROVIDER_ONECLOUD),
		))
	}

	if cloudEnvStr == api.CLOUD_ENV_PRIVATE_ON_PREMISE {
		q = q.Filter(sqlchemy.OR(
			sqlchemy.In(q.Field("provider"), cloudprovider.GetPrivateProviders()),
			sqlchemy.In(q.Field("provider"), cloudprovider.GetOnPremiseProviders()),
			sqlchemy.Equals(q.Field("provider"), api.CLOUD_PROVIDER_ONECLOUD),
		))
	}

	if jsonutils.QueryBoolean(query, "is_managed", false) {
		q = q.IsNotEmpty("external_id")
	}

	managerStr, _ := query.GetString("manager")
	if len(managerStr) > 0 {
		subq := CloudproviderRegionManager.QueryRelatedRegionIds("", managerStr)
		q = q.In("id", subq)
	}
	accountStr, _ := query.GetString("account")
	if len(accountStr) > 0 {
		subq := CloudproviderRegionManager.QueryRelatedRegionIds(accountStr)
		q = q.In("id", subq)
	}

	domainId, err := db.FetchQueryDomain(ctx, userCred, query)
	if len(domainId) > 0 {
		q = q.In("id", getCloudRegionIdByDomainId(domainId))
	}

	if jsonutils.QueryBoolean(query, "usable", false) || jsonutils.QueryBoolean(query, "usable_vpc", false) {
		providers := CloudproviderManager.Query().SubQuery()
		networks := NetworkManager.Query().SubQuery()
		wires := WireManager.Query().SubQuery()
		vpcs := VpcManager.Query().SubQuery()

		usableNet := jsonutils.QueryBoolean(query, "usable", false)
		usableVpc := jsonutils.QueryBoolean(query, "usable_vpc", false)

		sq := vpcs.Query(sqlchemy.DISTINCT("cloudregion_id", vpcs.Field("cloudregion_id")))
		if usableNet {
			sq = sq.Join(wires, sqlchemy.Equals(vpcs.Field("id"), wires.Field("vpc_id")))
			sq = sq.Join(networks, sqlchemy.Equals(wires.Field("id"), networks.Field("wire_id")))
		}
		sq = sq.Join(providers, sqlchemy.Equals(vpcs.Field("manager_id"), providers.Field("id")))
		if usableNet {
			sq = sq.Filter(sqlchemy.Equals(networks.Field("status"), api.NETWORK_STATUS_AVAILABLE))
		}
		sq = sq.Filter(sqlchemy.IsTrue(providers.Field("enabled")))
		sq = sq.Filter(sqlchemy.In(providers.Field("status"), api.CLOUD_PROVIDER_VALID_STATUS))
		sq = sq.Filter(sqlchemy.In(providers.Field("health_status"), api.CLOUD_PROVIDER_VALID_HEALTH_STATUS))
		if usableVpc {
			sq = sq.Filter(sqlchemy.Equals(vpcs.Field("status"), api.VPC_STATUS_AVAILABLE))
		}

		sq2 := vpcs.Query(sqlchemy.DISTINCT("cloudregion_id", vpcs.Field("cloudregion_id")))
		if usableNet {
			sq2 = sq2.Join(wires, sqlchemy.Equals(vpcs.Field("id"), wires.Field("vpc_id")))
			sq2 = sq2.Join(networks, sqlchemy.Equals(wires.Field("id"), networks.Field("wire_id")))
			sq2 = sq2.Filter(sqlchemy.Equals(networks.Field("status"), api.NETWORK_STATUS_AVAILABLE))
		}
		sq2 = sq2.Filter(sqlchemy.IsNullOrEmpty(vpcs.Field("manager_id")))
		if usableVpc {
			sq2 = sq2.Filter(sqlchemy.Equals(vpcs.Field("status"), api.VPC_STATUS_AVAILABLE))
		}

		q = q.Filter(sqlchemy.OR(
			sqlchemy.In(q.Field("id"), sq.SubQuery()),
			sqlchemy.In(q.Field("id"), sq2.SubQuery()),
		))

		service, _ := query.GetString("service")
		switch service {
		case DBInstanceManager.KeywordPlural():
			skusSQ := DBInstanceSkuManager.Query("cloudregion_id").Equals("status", api.DBINSTANCE_SKU_AVAILABLE).IsTrue("enabled").SubQuery()
			q = q.In("id", skusSQ)
		case ElasticcacheManager.KeywordPlural():
			q2 := ElasticcacheSkuManager.Query("cloudregion_id")
			statusFilter := sqlchemy.OR(sqlchemy.Equals(q2.Field("prepaid_status"), api.SkuStatusAvailable), sqlchemy.Equals(q2.Field("postpaid_status"), api.SkuStatusAvailable))
			skusSQ := q2.Filter(statusFilter).SubQuery()
			q = q.In("id", skusSQ)
		default:
			break
		}
	}

	cityStr, _ := query.GetString("city")
	if cityStr == "Other" {
		q = q.IsNullOrEmpty("city")
		query.(*jsonutils.JSONDict).Remove("city")
	}
	return q, nil
}

/*func (manager *SCloudregionManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return manager.SEnabledStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data)
}*/

func (self *SCloudregion) isManaged() bool {
	if len(self.ExternalId) > 0 {
		return true
	} else {
		return false
	}
}

func (self *SCloudregion) ValidateUpdateCondition(ctx context.Context) error {
	if len(self.ExternalId) > 0 && len(self.ManagerId) == 0 {
		return httperrors.NewConflictError("Cannot update external resource")
	}
	return self.SEnabledStatusStandaloneResourceBase.ValidateUpdateCondition(ctx)
}

func (self *SCloudregion) AllowGetDetailsCapability(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (self *SCloudregion) GetDetailsCapability(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	capa, err := GetCapabilities(ctx, userCred, query, self, nil)
	if err != nil {
		return nil, err
	}
	return jsonutils.Marshal(&capa), nil
}

func (self *SCloudregion) GetNetworkCount() (int, error) {
	return getNetworkCount(self, nil, "")
}

func (self *SCloudregion) getMinNicCount() int {
	return options.Options.MinNicCount
}

func (self *SCloudregion) getMaxNicCount() int {
	if self.isManaged() {
		return options.Options.MaxManagedNicCount
	}
	return options.Options.MaxNormalNicCount
}

func (self *SCloudregion) getMinDataDiskCount() int {
	return options.Options.MinDataDiskCount
}

func (self *SCloudregion) getMaxDataDiskCount() int {
	return options.Options.MaxDataDiskCount
}

func (manager *SCloudregionManager) FetchDefaultRegion() *SCloudregion {
	return manager.FetchRegionById(api.DEFAULT_REGION_ID)
}
