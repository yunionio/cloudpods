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
	"strings"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/onecloud/pkg/util/yunionmeta"
)

type SCloudregionManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
	SI18nResourceBaseManager
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
	SI18nResourceBase
	db.SExternalizedResourceBase

	cloudprovider.SGeographicInfo

	// 云环境
	// example: ChinaCloud
	Environment string `width:"32" charset:"ascii" list:"user"`
	// 云平台
	// example: Huawei
	Provider string `width:"64" charset:"ascii" list:"user" nullable:"false" default:"OneCloud"`
}

func (self *SCloudregion) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	idstr, _ := data.GetString("id")
	if len(idstr) > 0 {
		self.Id = idstr
	}
	return nil
}

func (self *SCloudregion) ValidateDeleteCondition(ctx context.Context, info *api.CloudregionDetails) error {
	if self.Id == api.DEFAULT_REGION_ID {
		return httperrors.NewProtectedResourceError("not allow to delete default cloud region")
	}
	if gotypes.IsNil(info) {
		info = &api.CloudregionDetails{}
		usage, err := CloudregionManager.TotalResourceCount([]string{self.Id})
		if err != nil {
			return err
		}
		info.SCloudregionUsage, _ = usage[self.Id]
	}
	if info.ZoneCount > 0 || info.VpcCount > 0 {
		return httperrors.NewNotEmptyError("not empty cloud region")
	}
	return self.SEnabledStatusStandaloneResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (self *SCloudregion) GetElasticIps(managerId, eipMode string) ([]SElasticip, error) {
	q := ElasticipManager.Query().Equals("cloudregion_id", self.Id)
	if len(managerId) > 0 {
		q = q.Equals("manager_id", managerId)
	}
	if len(eipMode) > 0 {
		q = q.Equals("mode", eipMode)
	}
	eips := []SElasticip{}
	err := db.FetchModelObjects(ElasticipManager, q, &eips)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return eips, nil
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

func (self *SCloudregion) GetZoneBySuffix(suffix string) (*SZone, error) {
	sq := ZoneManager.Query().SubQuery()
	q := sq.Query().Filter(
		sqlchemy.AND(
			sqlchemy.Equals(sq.Field("cloudregion_id"), self.Id),
			sqlchemy.Endswith(sq.Field("external_id"), suffix),
		),
	)
	count, err := q.CountWithError()
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, suffix)
	}
	if count > 1 {
		return nil, errors.Wrapf(cloudprovider.ErrDuplicateId, suffix)
	}
	zone := &SZone{}
	zone.SetModelManager(ZoneManager, zone)
	return zone, q.First(zone)
}

func (self *SCloudregion) GetGuestCount() (int, error) {
	return self.getGuestCountInternal(false)
}

func (self *SCloudregion) GetManagedGuestsQuery(managerId string) *sqlchemy.SQuery {
	q := GuestManager.Query().IsNotEmpty("external_id")
	hosts := HostManager.Query().Equals("manager_id", managerId).SubQuery()
	zones := ZoneManager.Query().Equals("cloudregion_id", self.Id).SubQuery()
	q = q.Join(hosts, sqlchemy.Equals(q.Field("host_id"), hosts.Field("id")))
	q = q.Join(zones, sqlchemy.Equals(hosts.Field("zone_id"), zones.Field("id")))
	return q
}

func (self *SCloudregion) GetManagedGuests(managerId string) ([]SGuest, error) {
	q := self.GetManagedGuestsQuery(managerId)
	ret := []SGuest{}
	err := db.FetchModelObjects(GuestManager, q, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SCloudregion) GetManagedGuestsCount(managerId string) (int, error) {
	return self.GetManagedGuestsQuery(managerId).CountWithError()
}

func (self *SCloudregion) GetManagedLoadbalancerQuery(managerId string) *sqlchemy.SQuery {
	return LoadbalancerManager.Query().
		IsNotEmpty("external_id").
		Equals("cloudregion_id", self.Id).
		Equals("manager_id", managerId)
}

func (self *SCloudregion) GetManagedLoadbalancers(managerId string) ([]SLoadbalancer, error) {
	q := self.GetManagedLoadbalancerQuery(managerId)
	ret := []SLoadbalancer{}
	err := db.FetchModelObjects(LoadbalancerManager, q, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SCloudregion) GetManagedLoadbalancerCount(managerId string) (int, error) {
	return self.GetManagedLoadbalancerQuery(managerId).CountWithError()
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

func (self *SCloudregion) GetVpcQuery() *sqlchemy.SQuery {
	vpcs := VpcManager.Query()
	if self.Id == api.DEFAULT_REGION_ID {
		return vpcs.Filter(sqlchemy.OR(sqlchemy.IsNull(vpcs.Field("cloudregion_id")),
			sqlchemy.IsEmpty(vpcs.Field("cloudregion_id")),
			sqlchemy.Equals(vpcs.Field("cloudregion_id"), self.Id)))
	}
	return vpcs.Equals("cloudregion_id", self.Id)
}

func (self *SCloudregion) GetVpcCount() (int, error) {
	return self.GetVpcQuery().CountWithError()
}

func (self *SCloudregion) GetVpcs() ([]SVpc, error) {
	vpcs := []SVpc{}
	q := self.GetVpcQuery()
	err := db.FetchModelObjects(VpcManager, q, &vpcs)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return vpcs, nil
}

func (self *SCloudregion) GetCloudproviderVpcs(managerId string) ([]SVpc, error) {
	vpcs := []SVpc{}
	q := self.GetVpcQuery().Equals("manager_id", managerId)
	err := db.FetchModelObjects(VpcManager, q, &vpcs)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return vpcs, nil
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

func (self *SCloudregion) getUsage(ctx context.Context) api.SCloudregionUsage {
	out := api.SCloudregionUsage{}
	out.VpcCount, _ = self.GetVpcCount()
	out.ZoneCount, _ = self.GetZoneCount()
	out.GuestCount, _ = self.GetGuestCount()
	out.NetworkCount, _ = self.GetNetworkCount(ctx)
	out.GuestIncrementCount, _ = self.GetGuestIncrementCount()
	return out
}

type SRegionUsageCount struct {
	Id string
	api.SCloudregionUsage
}

func (cm *SCloudregionManager) query(manager db.IModelManager, field string, regionIds []string, filter func(*sqlchemy.SQuery) *sqlchemy.SQuery) *sqlchemy.SSubQuery {
	q := manager.Query()

	if filter != nil {
		q = filter(q)
	}

	sq := q.SubQuery()

	return sq.Query(
		sq.Field("cloudregion_id"),
		sqlchemy.COUNT(field),
	).In("cloudregion_id", regionIds).GroupBy(sq.Field("cloudregion_id")).SubQuery()
}

func (manager *SCloudregionManager) TotalResourceCount(regionIds []string) (map[string]api.SCloudregionUsage, error) {
	vpcSQ := manager.query(VpcManager, "vpc_cnt", regionIds, nil)
	zoneSQ := manager.query(ZoneManager, "zone_cnt", regionIds, nil)
	guestSQ := manager.query(GuestManager, "guest_cnt", regionIds, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		hosts := HostManager.Query().SubQuery()
		zones := ZoneManager.Query().SubQuery()
		sq := q.SubQuery()
		return sq.Query(
			sq.Field("id"),
			zones.Field("cloudregion_id").Label("cloudregion_id"),
		).Join(hosts, sqlchemy.Equals(sq.Field("host_id"), hosts.Field("id"))).Join(zones, sqlchemy.Equals(zones.Field("id"), hosts.Field("zone_id")))
	})
	guestIncSQ := manager.query(GuestManager, "guest_increment_cnt", regionIds, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		hosts := HostManager.Query().SubQuery()
		zones := ZoneManager.Query().SubQuery()
		year, month, _ := time.Now().UTC().Date()
		startOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
		q = q.GE("created_at", startOfMonth)
		sq := q.SubQuery()
		return sq.Query(
			sq.Field("id"),
			zones.Field("cloudregion_id").Label("cloudregion_id"),
		).Join(hosts, sqlchemy.Equals(sq.Field("host_id"), hosts.Field("id"))).Join(zones, sqlchemy.Equals(zones.Field("id"), hosts.Field("zone_id")))
	})
	networkSQ := manager.query(NetworkManager, "network_cnt", regionIds, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		wires := WireManager.Query().SubQuery()
		vpcs := VpcManager.Query().SubQuery()
		sq := q.SubQuery()
		return sq.Query(
			sq.Field("id"),
			vpcs.Field("cloudregion_id").Label("cloudregion_id"),
		).Join(wires, sqlchemy.Equals(sq.Field("wire_id"), wires.Field("id"))).Join(vpcs, sqlchemy.Equals(vpcs.Field("id"), wires.Field("vpc_id")))
	})

	regions := manager.Query().SubQuery()
	regionQ := regions.Query(
		sqlchemy.SUM("vpc_count", vpcSQ.Field("vpc_cnt")),
		sqlchemy.SUM("zone_count", zoneSQ.Field("zone_cnt")),
		sqlchemy.SUM("guest_count", guestSQ.Field("guest_cnt")),
		sqlchemy.SUM("guest_increment_count", guestIncSQ.Field("guest_increment_cnt")),
		sqlchemy.SUM("network_count", networkSQ.Field("network_cnt")),
	)

	regionQ.AppendField(regionQ.Field("id"))

	regionQ = regionQ.LeftJoin(vpcSQ, sqlchemy.Equals(regionQ.Field("id"), vpcSQ.Field("cloudregion_id")))
	regionQ = regionQ.LeftJoin(zoneSQ, sqlchemy.Equals(regionQ.Field("id"), zoneSQ.Field("cloudregion_id")))
	regionQ = regionQ.LeftJoin(guestSQ, sqlchemy.Equals(regionQ.Field("id"), guestSQ.Field("cloudregion_id")))
	regionQ = regionQ.LeftJoin(guestIncSQ, sqlchemy.Equals(regionQ.Field("id"), guestIncSQ.Field("cloudregion_id")))
	regionQ = regionQ.LeftJoin(networkSQ, sqlchemy.Equals(regionQ.Field("id"), networkSQ.Field("cloudregion_id")))

	regionQ = regionQ.Filter(sqlchemy.In(regionQ.Field("id"), regionIds)).GroupBy(regionQ.Field("id"))

	regionCount := []SRegionUsageCount{}
	err := regionQ.All(&regionCount)
	if err != nil {
		return nil, errors.Wrapf(err, "regionQ.All")
	}

	result := map[string]api.SCloudregionUsage{}
	for i := range regionCount {
		result[regionCount[i].Id] = regionCount[i].SCloudregionUsage
	}

	return result, nil
}

func (manager *SCloudregionManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.CloudregionDetails {
	rows := make([]api.CloudregionDetails, len(objs))
	stdRows := manager.SEnabledStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regionIds := make([]string, len(objs))
	for i := range rows {
		region := objs[i].(*SCloudregion)
		rows[i] = api.CloudregionDetails{
			EnabledStatusStandaloneResourceDetails: stdRows[i],
			CloudEnv:                               region.GetCloudEnv(),
		}
		regionIds[i] = region.Id
	}
	count, err := manager.TotalResourceCount(regionIds)
	if err != nil {
		log.Errorf("TotalResourceCount")
		return rows
	}
	for i := range rows {
		rows[i].SCloudregionUsage, _ = count[regionIds[i]]
	}
	return rows
}

func (self *SCloudregion) GetServerSkus() ([]SServerSku, error) {
	skus := []SServerSku{}
	q := ServerSkuManager.Query().Equals("cloudregion_id", self.Id)
	err := db.FetchModelObjects(ServerSkuManager, q, &skus)
	if err != nil {
		return nil, err
	}
	return skus, nil
}

func (self *SCloudprovider) GetRegionByExternalIdPrefix(prefix string) ([]SCloudregion, error) {
	prefix = strings.TrimSuffix(prefix, "/")
	regions := make([]SCloudregion, 0)
	q := CloudregionManager.Query()
	q = q.Filter(
		sqlchemy.OR(
			sqlchemy.Startswith(q.Field("external_id"), prefix+"/"),
			sqlchemy.Equals(q.Field("external_id"), prefix),
		),
	)
	err := db.FetchModelObjects(CloudregionManager, q, &regions)
	if err != nil {
		return nil, err
	}
	return regions, nil
}

func (manager *SCloudregionManager) GetRegionByProvider(provider string) ([]SCloudregion, error) {
	regions := make([]SCloudregion, 0)
	q := manager.Query().Equals("provider", provider)
	err := db.FetchModelObjects(manager, q, &regions)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
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
	lockman.LockRawObject(ctx, manager.Keyword(), externalIdPrefix)
	defer lockman.ReleaseRawObject(ctx, manager.Keyword(), externalIdPrefix)

	syncResult := compare.SyncResult{}
	localRegions := make([]SCloudregion, 0)
	remoteRegions := make([]cloudprovider.ICloudRegion, 0)
	cloudProviderRegions := make([]SCloudproviderregion, 0)

	dbRegions, err := cloudProvider.GetRegionByExternalIdPrefix(externalIdPrefix)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, nil, syncResult
	}
	log.Debugf("Region with provider %s %d -> %d", externalIdPrefix, len(regions), len(dbRegions))

	removed := make([]SCloudregion, 0)
	commondb := make([]SCloudregion, 0)
	commonext := make([]cloudprovider.ICloudRegion, 0)
	added := make([]cloudprovider.ICloudRegion, 0)
	err = compare.CompareSets(dbRegions, regions, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(errors.Wrapf(err, "CompareSets"))
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
		err = commondb[i].syncWithCloudRegion(ctx, userCred, commonext[i])
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			cpr := CloudproviderRegionManager.FetchByIdsOrCreate(cloudProvider.Id, commondb[i].Id)
			cpr.setCapabilities(ctx, userCred, commonext[i].GetCapabilities())
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
			cpr := CloudproviderRegionManager.FetchByIdsOrCreate(cloudProvider.Id, new.Id)
			cpr.setCapabilities(ctx, userCred, added[i].GetCapabilities())
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

	return self.purgeAll(ctx, cloudProvider.Id)
}

func (self *SCloudregion) syncWithCloudRegion(ctx context.Context, userCred mcclient.TokenCredential, cloudRegion cloudprovider.ICloudRegion) error {
	err := CloudregionManager.SyncI18ns(ctx, userCred, self, cloudRegion.GetI18n())
	if err != nil {
		return errors.Wrap(err, "SyncI18ns")
	}

	diff, err := db.UpdateWithLock(ctx, self, func() error {
		if !utils.IsInStringArray(self.Provider, api.PRIVATE_CLOUD_PROVIDERS) {
			self.Name = cloudRegion.GetName()
		}
		self.Status = cloudRegion.GetStatus()
		geoInfo := cloudRegion.GetGeographicInfo()
		if !self.SGeographicInfo.IsEquals(geoInfo) {
			self.SGeographicInfo = geoInfo
		}
		self.Provider = cloudRegion.GetProvider()
		self.Environment = cloudRegion.GetCloudEnv()
		self.SetEnabled(true)

		self.IsEmulated = cloudRegion.IsEmulated()

		return nil
	})
	if err != nil && errors.Cause(err) != sqlchemy.ErrNoDataToUpdate {
		log.Errorf("syncWithCloudRegion %s", err)
		return err
	}
	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return nil
}

func (manager *SCloudregionManager) newFromCloudRegion(ctx context.Context, userCred mcclient.TokenCredential, cloudRegion cloudprovider.ICloudRegion, provider *SCloudprovider) (*SCloudregion, error) {
	region := SCloudregion{}
	region.SetModelManager(manager, &region)

	region.ExternalId = cloudRegion.GetGlobalId()
	region.SGeographicInfo = cloudRegion.GetGeographicInfo()
	region.Status = cloudRegion.GetStatus()
	region.SetEnabled(true)
	region.Provider = cloudRegion.GetProvider()
	region.Environment = cloudRegion.GetCloudEnv()

	region.IsEmulated = cloudRegion.IsEmulated()

	var err error
	err = func() error {
		lockman.LockRawObject(ctx, manager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, manager.Keyword(), "name")

		region.Name, err = db.GenerateName(ctx, manager, nil, cloudRegion.GetName())
		if err != nil {
			return errors.Wrapf(err, "db.GenerateName")
		}

		return manager.TableSpec().Insert(ctx, &region)
	}()
	if err != nil {
		return nil, err
	}

	err = manager.SyncI18ns(ctx, userCred, &region, cloudRegion.GetI18n())
	if err != nil {
		return nil, errors.Wrap(err, "SyncI18ns")
	}

	db.OpsLog.LogEvent(&region, db.ACT_CREATE, region.GetShortDesc(ctx), userCred)
	return &region, nil
}

func (self *SCloudregion) PerformDefaultVpc(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	vpcs, err := self.GetVpcs()
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
			defRegion.SetEnabled(true)
			defRegion.Description = "Default Region"
			defRegion.Status = api.CLOUD_REGION_STATUS_INSERVER
			defRegion.Provider = api.CLOUD_PROVIDER_ONECLOUD
			err := manager.TableSpec().Insert(context.TODO(), &defRegion)
			if err != nil {
				return errors.Wrap(err, "insert default region")
			}
		} else {
			return errors.Wrap(err, "fetch default region")
		}
	} else if region := obj.(*SCloudregion); region.Provider != api.CLOUD_PROVIDER_ONECLOUD {
		_, err := db.Update(region, func() error {
			region.Provider = api.CLOUD_PROVIDER_ONECLOUD
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "update default region provider")
		}
	}
	return nil
}

func getCloudRegionIdByDomainId(domainId string) *sqlchemy.SSubQuery {
	cloudproviderregions := CloudproviderRegionManager.Query().SubQuery()

	// not managed region
	q1 := CloudregionManager.Query("id").Equals("provider", api.CLOUD_PROVIDER_ONECLOUD)

	// managed region
	q2 := cloudproviderregions.Query(cloudproviderregions.Field("cloudregion_id", "id"))
	providerIds := CloudproviderManager.Query("id")
	providerIds = CloudproviderManager.filterByDomainId(providerIds, domainId)
	providersIdsQ := providerIds.Distinct().SubQuery()
	q2 = q2.Join(providersIdsQ, sqlchemy.Equals(providersIdsQ.Field("id"), cloudproviderregions.Field("cloudprovider_id")))

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

func (manager *SCloudregionManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	if field == "region" {
		q = q.AppendField(q.Field("name").Label("region")).Distinct()
		return q, nil
	}
	q, err := manager.SEnabledStatusStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SCloudregionManager) OrderByExtraFields(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.CloudregionListInput) (*sqlchemy.SQuery, error) {
	q, err := manager.SEnabledStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.EnabledStatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusStandaloneResourceBaseManager.OrderByExtraFields")
	}
	if db.NeedOrderQuery([]string{query.OrderByZoneCount}) {
		zQ := ZoneManager.Query()
		zQ = zQ.AppendField(zQ.Field("cloudregion_id"), sqlchemy.COUNT("zone_count", zQ.Field("cloudregion_id")))
		zQ = zQ.GroupBy(zQ.Field("cloudregion_id"))
		zSQ := zQ.SubQuery()
		q = q.LeftJoin(zSQ, sqlchemy.Equals(zSQ.Field("cloudregion_id"), q.Field("id")))
		q = q.AppendField(q.QueryFields()...)
		q = q.AppendField(zSQ.Field("zone_count"))
		q = db.OrderByFields(q, []string{query.OrderByZoneCount}, []sqlchemy.IQueryField{q.Field("zone_count")})
	}
	if db.NeedOrderQuery([]string{query.OrderByVpcCount}) {
		vQ := VpcManager.Query()
		vQ = vQ.AppendField(vQ.Field("cloudregion_id"), sqlchemy.COUNT("vpc_count", vQ.Field("cloudregion_id")))
		vQ = vQ.GroupBy(vQ.Field("cloudregion_id"))
		vSQ := vQ.SubQuery()
		q = q.LeftJoin(vSQ, sqlchemy.Equals(vSQ.Field("cloudregion_id"), q.Field("id")))
		q = q.AppendField(q.QueryFields()...)
		q = q.AppendField(vSQ.Field("vpc_count"))
		q = db.OrderByFields(q, []string{query.OrderByZoneCount}, []sqlchemy.IQueryField{q.Field("vpc_count")})
	}
	if db.NeedOrderQuery([]string{query.OrderByGuestCount}) {
		guestQ := GuestManager.Query()
		guestQ = guestQ.AppendField(guestQ.Field("host_id"), sqlchemy.COUNT("guest_count"))
		guestQ = guestQ.GroupBy(guestQ.Field("host_id"))
		guestSQ := guestQ.SubQuery()

		hostQ := HostManager.Query()
		hostQ = hostQ.LeftJoin(guestSQ, sqlchemy.Equals(guestSQ.Field("host_id"), hostQ.Field("id")))
		hostQ = hostQ.AppendField(hostQ.QueryFields()...)
		hostQ = hostQ.AppendField(hostQ.Field("zone_id"), sqlchemy.COUNT("guest_count", guestSQ.Field("guest_count")))
		hostQ = hostQ.GroupBy("zone_id")
		hostSQ := hostQ.SubQuery()

		zQ := ZoneManager.Query()
		zQ = zQ.AppendField(zQ.Field("cloudregion_id"), sqlchemy.COUNT("guest_count", hostSQ.Field("guest_count")))
		zQ = zQ.GroupBy(zQ.Field("cloudregion_id"))
		zQ = zQ.LeftJoin(hostSQ, sqlchemy.Equals(zQ.Field("id"), hostSQ.Field("zone_id")))
		zSQ := zQ.SubQuery()

		q = q.LeftJoin(zSQ, sqlchemy.Equals(zSQ.Field("cloudregion_id"), q.Field("id")))
		q = q.AppendField(q.QueryFields()...)
		q = q.AppendField(zSQ.Field("guest_count"))
		q = db.OrderByFields(q, []string{query.OrderByGuestCount}, []sqlchemy.IQueryField{q.Field("guest_count")})
	}
	return q, nil
}

// 云平台区域列表
func (manager *SCloudregionManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.CloudregionListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	providerStrs := query.Providers
	if len(providerStrs) > 0 {
		subq := queryCloudregionIdsByProviders("provider", providerStrs)
		q = q.In("id", subq.SubQuery())
	}

	brandStrs := query.Brands
	if len(brandStrs) > 0 {
		subq := queryCloudregionIdsByProviders("brand", brandStrs)
		q = q.In("id", subq.SubQuery())
	}

	q, err = manager.SEnabledStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledStatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusStandaloneResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	cloudEnvStr := query.CloudEnv
	if cloudEnvStr == api.CLOUD_ENV_PUBLIC_CLOUD {
		q = q.In("provider", cloudprovider.GetPublicProviders())
	}

	if cloudEnvStr == api.CLOUD_ENV_PRIVATE_CLOUD {
		q = q.In("provider", cloudprovider.GetPrivateProviders())
	}

	if cloudEnvStr == api.CLOUD_ENV_ON_PREMISE {
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

	if query.IsManaged != nil {
		if *query.IsManaged {
			q = q.IsNotEmpty("external_id")
		} else {
			q = q.IsNullOrEmpty("external_id")
		}
	}

	managerStr := query.CloudproviderId
	if len(managerStr) > 0 {
		subq := CloudproviderRegionManager.QueryRelatedRegionIds(nil, managerStr...)
		q = q.In("id", subq)
	}
	accountArr := query.CloudaccountId
	if len(accountArr) > 0 {
		subq := CloudproviderRegionManager.QueryRelatedRegionIds(accountArr)
		q = q.In("id", subq)
	}

	domainId, err := db.FetchQueryDomain(ctx, userCred, jsonutils.Marshal(query))
	if len(domainId) > 0 {
		q = q.In("id", getCloudRegionIdByDomainId(domainId))
	}

	usableNet := (query.Usable != nil && *query.Usable)
	usableVpc := (query.UsableVpc != nil && *query.UsableVpc)
	if usableNet || usableVpc {
		providers := usableCloudProviders().SubQuery()
		networks := NetworkManager.Query().SubQuery()
		wires := WireManager.Query().SubQuery()
		vpcs := VpcManager.Query().SubQuery()

		sq := vpcs.Query(sqlchemy.DISTINCT("cloudregion_id", vpcs.Field("cloudregion_id")))
		if usableNet {
			sq = sq.Join(wires, sqlchemy.Equals(vpcs.Field("id"), wires.Field("vpc_id")))
			sq = sq.Join(networks, sqlchemy.Equals(wires.Field("id"), networks.Field("wire_id")))
		}
		sq = sq.Join(providers, sqlchemy.Equals(vpcs.Field("manager_id"), providers.Field("id")))
		if usableNet {
			sq = sq.Filter(sqlchemy.Equals(networks.Field("status"), api.NETWORK_STATUS_AVAILABLE))
		}
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

		service := query.Service
		switch service {
		case DBInstanceManager.KeywordPlural():
			regionSQ := CloudproviderCapabilityManager.Query("cloudregion_id").Equals("capability", cloudprovider.CLOUD_CAPABILITY_RDS).SubQuery()
			q = q.In("id", regionSQ)
		case ElasticcacheManager.KeywordPlural():
			regionSQ := CloudproviderCapabilityManager.Query("cloudregion_id").Equals("capability", cloudprovider.CLOUD_CAPABILITY_CACHE).SubQuery()
			q = q.In("id", regionSQ)
		default:
			break
		}
	}

	cityStr := query.City
	if cityStr == "Other" {
		q = q.IsNullOrEmpty("city")
	} else if len(cityStr) > 0 {
		q = q.Equals("city", cityStr)
	}

	if len(query.Environment) > 0 {
		q = q.In("environment", query.Environment)
	}

	if len(query.Capability) > 0 {
		subq := CloudproviderCapabilityManager.Query("cloudregion_id").In("capability", query.Capability).Distinct().SubQuery()
		q = q.In("id", subq)
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

func (self *SCloudregion) getCloudaccounts() []SCloudaccount {
	providers := CloudproviderManager.Query().SubQuery()
	providerregions := CloudproviderRegionManager.Query().SubQuery()
	q := CloudaccountManager.Query()
	q = q.Join(providers, sqlchemy.Equals(q.Field("id"), providers.Field("cloudaccount_id")))
	q = q.Join(providerregions, sqlchemy.Equals(providers.Field("id"), providerregions.Field("cloudprovider_id")))
	q = q.Filter(sqlchemy.Equals(providerregions.Field("cloudregion_id"), self.Id))
	q = q.Distinct()

	accounts := make([]SCloudaccount, 0)
	err := db.FetchModelObjects(CloudaccountManager, q, &accounts)
	if err != nil {
		if errors.Cause(err) != sql.ErrNoRows {
			log.Errorf("get cloudregion's cloudaccounts fail: %s", err)
		}
		return nil
	}
	return accounts
}

func (self *SCloudregion) GetRegionCloudenvInfo() api.CloudenvResourceInfo {
	info := api.CloudenvResourceInfo{
		CloudEnv:    self.GetCloudEnv(),
		Provider:    self.Provider,
		Environment: self.Environment,
	}
	return info
}

func (self *SCloudregion) GetI18N(ctx context.Context) *jsonutils.JSONDict {
	return self.GetModelI18N(ctx, self)
}

func (self *SCloudregion) GetRegionInfo(ctx context.Context) api.CloudregionResourceInfo {
	name := self.Name
	if v, ok := self.GetModelKeyI18N(ctx, self, "name"); ok {
		name = v
	}

	return api.CloudregionResourceInfo{
		Region:           name,
		Cloudregion:      name,
		RegionId:         self.Id,
		RegionExtId:      fetchExternalId(self.ExternalId),
		RegionExternalId: self.ExternalId,
	}
}

func (self *SCloudregion) GetRegionExtId() string {
	return fetchExternalId(self.ExternalId)
}

func (self *SCloudregion) ValidateUpdateCondition(ctx context.Context) error {
	if len(self.ExternalId) > 0 {
		return httperrors.NewConflictError("Cannot update external resource")
	}
	return self.SEnabledStatusStandaloneResourceBase.ValidateUpdateCondition(ctx)
}

func (self *SCloudregion) SyncVpcs(ctx context.Context, userCred mcclient.TokenCredential, iregion cloudprovider.ICloudRegion, provider *SCloudprovider) error {
	syncResults, syncRange := SSyncResultSet{}, &SSyncRange{}
	syncRegionVPCs(ctx, userCred, syncResults, provider, self, iregion, syncRange)
	return nil
}

func (self *SCloudregion) GetDetailsCapability(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	capa, err := GetCapabilities(ctx, userCred, query, self, nil)
	if err != nil {
		return nil, err
	}
	return jsonutils.Marshal(&capa), nil
}

func (self *SCloudregion) GetDetailsDiskCapability(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	capa, err := GetDiskCapabilities(ctx, userCred, query, self, nil)
	if err != nil {
		return nil, err
	}
	return jsonutils.Marshal(&capa), nil
}

func (self *SCloudregion) GetNetworkCount(ctx context.Context) (int, error) {
	return getNetworkCount(ctx, nil, nil, rbacscope.ScopeSystem, self, nil)
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

func (self *SCloudregion) GetCloudEnv() string {
	return cloudprovider.GetProviderCloudEnv(self.Provider)
}

func (self *SCloudregion) GetSchedtags() []SSchedtag {
	return GetSchedtags(CloudregionschedtagManager, self.Id)
}

func (self *SCloudregion) GetDynamicConditionInput() *jsonutils.JSONDict {
	return jsonutils.Marshal(self).(*jsonutils.JSONDict)
}

func (self *SCloudregion) PerformSetSchedtag(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return PerformSetResourceSchedtag(self, ctx, userCred, query, data)
}

func (self *SCloudregion) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.CloudregionPurgeInput) (jsonutils.JSONObject, error) {
	if self.Id == api.DEFAULT_REGION_ID {
		return nil, httperrors.NewProtectedResourceError("not allow to delete default cloud region")
	}
	if len(input.ManagerId) == 0 {
		return nil, httperrors.NewMissingParameterError("manager_id")
	}
	return nil, self.purgeAll(ctx, input.ManagerId)
}

func (self *SCloudregion) GetSchedtagJointManager() ISchedtagJointManager {
	return CloudregionschedtagManager
}

func (self *SCloudregion) ClearSchedDescCache() error {
	zones, err := self.GetZones()
	if err != nil {
		return errors.Wrap(err, "get zones")
	}
	for i := range zones {
		if err := zones[i].ClearSchedDescCache(); err != nil {
			return errors.Wrapf(err, "clean zone %s sched cache", zones[i].GetName())
		}
	}
	return nil
}

func (self *SCloudregion) GetCloudimages() ([]SCloudimage, error) {
	q := CloudimageManager.Query().Equals("cloudregion_id", self.Id)
	images := []SCloudimage{}
	err := db.FetchModelObjects(CloudimageManager, q, &images)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return images, nil
}

func (self *SCloudregion) GetSystemImageCount() (int, error) {
	sq := CloudimageManager.Query("external_id").Equals("cloudregion_id", self.Id)
	q := CachedimageManager.Query().Equals("image_type", cloudprovider.ImageTypeSystem).In("external_id", sq.SubQuery())
	return q.CountWithError()
}

func (self *SCloudregion) SyncCloudImages(ctx context.Context, userCred mcclient.TokenCredential, refresh, xor bool) error {
	lockman.LockRawObject(ctx, CloudimageManager.Keyword(), self.Id)
	defer lockman.ReleaseRawObject(ctx, CloudimageManager.Keyword(), self.Id)

	systemImageCount, err := self.GetSystemImageCount()
	if err != nil {
		return errors.Wrapf(err, "GetSystemImageCount")
	}

	dbImages, err := self.GetCloudimages()
	if err != nil {
		return errors.Wrapf(err, "GetCloudimages")
	}
	if len(dbImages) > 0 && systemImageCount > 0 && !refresh {
		return nil
	}
	meta, err := yunionmeta.FetchYunionmeta(ctx)
	if err != nil {
		return errors.Wrapf(err, "FetchYunionmeta")
	}
	iImages := []SCachedimage{}
	err = meta.List(CloudimageManager.Keyword(), self.ExternalId, &iImages)
	if err != nil {
		return errors.Wrapf(err, "GetCloudimages")
	}

	removed := make([]SCloudimage, 0)
	commondb := make([]SCloudimage, 0)
	commonext := make([]SCachedimage, 0)
	added := make([]SCachedimage, 0)
	err = compare.CompareSets(dbImages, iImages, &removed, &commondb, &commonext, &added)
	if err != nil {
		return errors.Wrapf(err, "compare.CompareSets")
	}

	result := compare.SyncResult{}

	for i := 0; i < len(removed); i++ {
		err := removed[i].syncRemove(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}

	if !xor {
		for i := 0; i < len(commonext); i++ {
			err := commondb[i].syncWithImage(ctx, userCred, commonext[i], self)
			if err != nil {
				result.UpdateError(errors.Wrapf(err, "updateCachedImage"))
				continue
			}
			result.Update()
		}
	}

	for i := 0; i < len(added); i++ {
		err = self.newCloudimage(ctx, userCred, added[i])
		if err != nil {
			result.AddError(errors.Wrapf(err, "newCloudimage"))
			continue
		}
		result.Add()
	}
	log.Infof("Sync Cloudimages for region %s result: %s", self.Name, result.Result())
	return nil
}

func (self *SCloudregion) GetStoragecaches() ([]SStoragecache, error) {
	zones := ZoneManager.Query("id").Equals("cloudregion_id", self.Id).SubQuery()
	sq := StorageManager.Query("storagecache_id").In("zone_id", zones).SubQuery()
	q := StoragecacheManager.Query().In("id", sq)
	caches := []SStoragecache{}
	err := db.FetchModelObjects(StoragecacheManager, q, &caches)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return caches, nil
}

func (self *SCloudregion) newCloudimage(ctx context.Context, userCred mcclient.TokenCredential, iImage SCachedimage) error {
	externalId := iImage.GetGlobalId()
	lockman.LockRawObject(ctx, CachedimageManager.Keyword(), externalId)
	defer lockman.ReleaseRawObject(ctx, CachedimageManager.Keyword(), externalId)

	_, err := db.FetchByExternalId(CachedimageManager, externalId)
	if err != nil {
		if errors.Cause(err) != sql.ErrNoRows {
			return errors.Wrapf(err, "db.FetchModelObjects(%s)", externalId)
		}
		image := &iImage
		image.SetModelManager(CachedimageManager, image)
		meta, err := yunionmeta.FetchYunionmeta(ctx)
		if err != nil {
			return err
		}

		skuUrl := self.getMetaUrl(meta.ImageBase, externalId)
		err = meta.Get(skuUrl, image)
		if err != nil {
			return errors.Wrapf(err, "Get")
		}

		image.IsPublic = true
		image.ProjectId = "system"
		err = CachedimageManager.TableSpec().Insert(ctx, image)
		if err != nil {
			return errors.Wrapf(err, "Insert cachedimage")
		}
	}
	cloudimage := &SCloudimage{}
	cloudimage.SetModelManager(CloudimageManager, cloudimage)
	cloudimage.Name = iImage.Name
	cloudimage.CloudregionId = self.Id
	cloudimage.ExternalId = externalId
	err = CloudimageManager.TableSpec().Insert(ctx, cloudimage)
	if err != nil {
		return errors.Wrapf(err, "Insert cloudimage")
	}
	return nil
}

func (manager *SCloudregionManager) PerformSyncSkus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CloudregionSkuSyncInput) (jsonutils.JSONObject, error) {
	return PerformActionSyncSkus(ctx, userCred, input.Resource, input.SkuSyncInput)
}

func (manager *SCloudregionManager) GetPropertySyncTasks(ctx context.Context, userCred mcclient.TokenCredential, query api.SkuTaskQueryInput) (jsonutils.JSONObject, error) {
	return GetPropertySkusSyncTasks(ctx, userCred, query)
}

func (self *SCloudregion) PerformSyncImages(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.SyncImagesInput) (jsonutils.JSONObject, error) {
	return nil, self.StartSyncImagesTask(ctx, userCred, "")
}

func (self *SCloudregion) StartSyncImagesTask(ctx context.Context, userCred mcclient.TokenCredential, parentId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "CloudregionSyncImagesTask", self, userCred, nil, "", "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SCloudregion) StartSyncSkusTask(ctx context.Context, userCred mcclient.TokenCredential, res string) error {
	params := jsonutils.NewDict()
	params.Set("resource", jsonutils.NewString(res))
	task, err := taskman.TaskManager.NewTask(ctx, "CloudRegionSyncSkusTask", self, userCred, params, "", "", nil)
	if err != nil {
		return errors.Wrapf(err, "CloudRegionSyncSkusTask")
	}
	return task.ScheduleRun(nil)
}

func (self *SCloudregion) GetCloudproviders() ([]SCloudprovider, error) {
	sq := CloudproviderRegionManager.Query().Equals("cloudregion_id", self.Id).SubQuery()
	q := CloudproviderManager.Query()
	q = q.Join(sq, sqlchemy.Equals(sq.Field("cloudprovider_id"), q.Field("id")))
	ret := []SCloudprovider{}
	err := db.FetchModelObjects(CloudproviderManager, q, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
