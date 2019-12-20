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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SZoneManager struct {
	db.SStatusStandaloneResourceBaseManager
}

var ZoneManager *SZoneManager

func init() {
	ZoneManager = &SZoneManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SZone{},
			"zones_tbl",
			"zone",
			"zones",
		),
	}
	ZoneManager.NameRequireAscii = false
	ZoneManager.SetVirtualObject(ZoneManager)
}

type SZone struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase

	Location string `width:"256" charset:"utf8" get:"user" list:"user" update:"admin"` // = Column(VARCHAR(256, charset='utf8'))
	Contacts string `width:"256" charset:"utf8" get:"user" update:"admin"`             // = Column(VARCHAR(256, charset='utf8'))
	NameCn   string `width:"256" charset:"utf8"`                                       // = Column(VARCHAR(256, charset='utf8'))
	// status = Column(VARCHAR(36, charset='ascii'), nullable=False, default=ZONE_DISABLE)
	ManagerUri string `width:"256" charset:"ascii" list:"admin" update:"admin"` // = Column(VARCHAR(256, charset='ascii'), nullable=True)
	// admin_id = Column(VARCHAR(36, charset='ascii'), nullable=False)
	CloudregionId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"admin_required"`
}

func (manager *SZoneManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{CloudregionManager},
	}
}

func (self *SZoneManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, self)
}

func (self *SZone) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SZone) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SZone) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
}

func (manager *SZoneManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (zone *SZone) ValidateDeleteCondition(ctx context.Context) error {
	usage := zone.GeneralUsage()
	if !usage.isEmpty() {
		return httperrors.NewNotEmptyError("not empty zone")
	}
	return zone.SStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

func (manager *SZoneManager) Count() (int, error) {
	return manager.Query().CountWithError()
}

type ZoneGeneralUsage struct {
	Hosts             int
	HostsEnabled      int
	Baremetals        int
	BaremetalsEnabled int
	Wires             int
	Networks          int
	Storages          int
}

func (usage *ZoneGeneralUsage) isEmpty() bool {
	if usage.Hosts > 0 {
		return false
	}
	if usage.Wires > 0 {
		return false
	}
	if usage.Networks > 0 {
		return false
	}
	if usage.Storages > 0 {
		return false
	}
	return true
}

func (zone *SZone) GeneralUsage() ZoneGeneralUsage {
	usage := ZoneGeneralUsage{}
	usage.Hosts, _ = zone.HostCount("", "", tristate.None, "", tristate.None)
	usage.HostsEnabled, _ = zone.HostCount("", "", tristate.True, "", tristate.None)
	usage.Baremetals, _ = zone.HostCount("", "", tristate.None, "", tristate.True)
	usage.BaremetalsEnabled, _ = zone.HostCount("", "", tristate.True, "", tristate.True)
	usage.Wires, _ = zone.getWireCount()
	usage.Networks, _ = zone.getNetworkCount()
	usage.Storages, _ = zone.getStorageCount()
	return usage
}

func (zone *SZone) HostCount(status string, hostStatus string, enabled tristate.TriState, hostType string, isBaremetal tristate.TriState) (int, error) {
	q := HostManager.Query().Equals("zone_id", zone.Id)
	if len(status) > 0 {
		q = q.Equals("status", status)
	}
	if len(hostStatus) > 0 {
		q = q.Equals("host_status", hostStatus)
	}
	if enabled.IsTrue() {
		q = q.IsTrue("enabled")
	} else if enabled.IsFalse() {
		q = q.IsFalse("enabled")
	}
	if len(hostType) > 0 {
		q = q.Equals("host_type", hostType)
	}
	if isBaremetal.IsTrue() {
		q = q.IsTrue("is_baremetal")
	} else if isBaremetal.IsFalse() {
		q = q.IsFalse("is_baremetal")
	}
	return q.CountWithError()
}

func (zone *SZone) getWireCount() (int, error) {
	q := WireManager.Query().Equals("zone_id", zone.Id)
	return q.CountWithError()
}

func (zone *SZone) getStorageCount() (int, error) {
	q := StorageManager.Query().Equals("zone_id", zone.Id)
	return q.CountWithError()
}

func (zone *SZone) getNetworkCount() (int, error) {
	return getNetworkCount(nil, zone, "")
}

func zoneExtra(zone *SZone, extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	usage := zone.GeneralUsage()
	extra.Update(jsonutils.Marshal(usage))
	region := zone.GetRegion()
	if region != nil {
		extra.Add(jsonutils.NewString(region.Name), "cloudregion")
		extra.Add(jsonutils.NewString(region.Provider), "provider")
	}
	return extra
}

func (zone *SZone) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := zone.SStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return zoneExtra(zone, extra)
}

func (zone *SZone) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := zone.SStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return zoneExtra(zone, extra), nil
}

func (zone *SZone) GetCloudRegionId() string {
	if len(zone.CloudregionId) == 0 {
		return "default"
	} else {
		return zone.CloudregionId
	}
}

func (manager *SZoneManager) GetZonesByRegion(region *SCloudregion) ([]SZone, error) {
	zones := make([]SZone, 0)
	q := manager.Query().Equals("cloudregion_id", region.Id)
	err := db.FetchModelObjects(manager, q, &zones)
	if err != nil {
		return nil, err
	}
	return zones, nil
}

func (manager *SZoneManager) SyncZones(ctx context.Context, userCred mcclient.TokenCredential, region *SCloudregion, zones []cloudprovider.ICloudZone) ([]SZone, []cloudprovider.ICloudZone, compare.SyncResult) {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	localZones := make([]SZone, 0)
	remoteZones := make([]cloudprovider.ICloudZone, 0)
	syncResult := compare.SyncResult{}

	dbZones, err := manager.GetZonesByRegion(region)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	removed := make([]SZone, 0)
	commondb := make([]SZone, 0)
	commonext := make([]cloudprovider.ICloudZone, 0)
	added := make([]cloudprovider.ICloudZone, 0)

	err = compare.CompareSets(dbZones, zones, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].syncRemoveCloudZone(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}
	for i := 0; i < len(commondb); i += 1 {
		err = commondb[i].syncWithCloudZone(ctx, userCred, commonext[i], region)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncMetadata(ctx, userCred, &commondb[i], commonext[i])
			localZones = append(localZones, commondb[i])
			remoteZones = append(remoteZones, commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i += 1 {
		new, err := manager.newFromCloudZone(ctx, userCred, added[i], region)
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncMetadata(ctx, userCred, new, added[i])
			localZones = append(localZones, *new)
			remoteZones = append(remoteZones, added[i])
			syncResult.Add()
		}
	}

	return localZones, remoteZones, syncResult
}

func (self *SZone) syncRemoveCloudZone(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	err := self.ValidateDeleteCondition(ctx)
	if err != nil { // cannot delete
		err = self.SetStatus(userCred, api.ZONE_DISABLE, "sync to delete")
	} else {
		err = self.Delete(ctx, userCred)
	}
	return err
}

func (self *SZone) syncWithCloudZone(ctx context.Context, userCred mcclient.TokenCredential, extZone cloudprovider.ICloudZone, region *SCloudregion) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		self.Name = extZone.GetName()
		self.Status = extZone.GetStatus()

		self.IsEmulated = extZone.IsEmulated()

		self.CloudregionId = region.Id

		return nil
	})
	if err != nil {
		log.Errorf("syncWithCloudZone error %s", err)
		return err
	}
	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return nil
}

func (manager *SZoneManager) newFromCloudZone(ctx context.Context, userCred mcclient.TokenCredential, extZone cloudprovider.ICloudZone, region *SCloudregion) (*SZone, error) {
	zone := SZone{}
	zone.SetModelManager(manager, &zone)

	newName, err := db.GenerateName(manager, userCred, extZone.GetName())
	if err != nil {
		return nil, err
	}
	zone.Name = newName
	zone.Status = extZone.GetStatus()
	zone.ExternalId = extZone.GetGlobalId()

	zone.IsEmulated = extZone.IsEmulated()

	zone.CloudregionId = region.Id

	err = manager.TableSpec().Insert(&zone)
	if err != nil {
		log.Errorf("newFromCloudZone fail %s", err)
		return nil, err
	}
	db.OpsLog.LogEvent(&zone, db.ACT_CREATE, zone.GetShortDesc(ctx), userCred)
	return &zone, nil
}

func (manager *SZoneManager) FetchZoneById(zoneId string) *SZone {
	zoneObj, err := manager.FetchById(zoneId)
	if err != nil {
		log.Errorf("%s", err)
		return nil
	}
	return zoneObj.(*SZone)
}

func (zone *SZone) GetRegion() *SCloudregion {
	return CloudregionManager.FetchRegionById(zone.GetCloudRegionId())
}

func (manager *SZoneManager) InitializeData() error {
	// set cloudregion ID
	zones := make([]SZone, 0)
	q := manager.Query()
	err := db.FetchModelObjects(manager, q, &zones)
	if err != nil {
		return err
	}
	for _, z := range zones {
		if len(z.CloudregionId) == 0 {
			db.Update(&z, func() error {
				z.CloudregionId = api.DEFAULT_REGION_ID
				return nil
			})
		}
		if z.Status == api.ZONE_INIT || z.Status == api.ZONE_DISABLE {
			db.Update(&z, func() error {
				z.Status = api.ZONE_ENABLE
				return nil
			})
		}
	}
	return nil
}

/*
Query 1:
vpc.manager_id is not empty && wire.zone_id is not empty
*/
func usableZoneQ1(providers, vpcs, wires, networks *sqlchemy.SSubQuery, usableNet, usableVpc bool) *sqlchemy.SSubQuery {
	// join tables
	sq := wires.Query(sqlchemy.DISTINCT("zone_id", wires.Field("zone_id")))
	if usableNet {
		sq = sq.Join(networks, sqlchemy.Equals(wires.Field("id"), networks.Field("wire_id")))
	}
	sq = sq.Join(vpcs, sqlchemy.Equals(wires.Field("vpc_id"), vpcs.Field("id")))
	sq = sq.Join(providers, sqlchemy.Equals(vpcs.Field("manager_id"), providers.Field("id")))

	// add filters
	if usableNet {
		sq = sq.Filter(sqlchemy.Equals(networks.Field("status"), api.NETWORK_STATUS_AVAILABLE))
	}
	sq = sq.Filter(sqlchemy.IsNotEmpty(wires.Field("zone_id")))
	sq = sq.Filter(sqlchemy.IsTrue(providers.Field("enabled")))
	sq = sq.Filter(sqlchemy.In(providers.Field("status"), api.CLOUD_PROVIDER_VALID_STATUS))
	sq = sq.Filter(sqlchemy.In(providers.Field("health_status"), api.CLOUD_PROVIDER_VALID_HEALTH_STATUS))
	if usableVpc {
		sq = sq.Filter(sqlchemy.Equals(vpcs.Field("status"), api.VPC_STATUS_AVAILABLE))
	}

	return sq.SubQuery()
}

/*
Query 2:
vpc.manager_id is empty && wire.zone_id is not empty
*/
func usableZoneQ2(vpcs, wires, networks *sqlchemy.SSubQuery, usableNet, usableVpc bool) *sqlchemy.SSubQuery {
	// join tables
	sq := wires.Query(sqlchemy.DISTINCT("zone_id", wires.Field("zone_id")))
	if usableNet {
		sq = sq.Join(networks, sqlchemy.Equals(wires.Field("id"), networks.Field("wire_id")))
	}
	sq = sq.Join(vpcs, sqlchemy.Equals(wires.Field("vpc_id"), vpcs.Field("id")))

	// add filters
	if usableNet {
		sq = sq.Filter(sqlchemy.Equals(networks.Field("status"), api.NETWORK_STATUS_AVAILABLE))
	}
	sq = sq.Filter(sqlchemy.IsNotEmpty(wires.Field("zone_id")))
	sq = sq.Filter(sqlchemy.IsNullOrEmpty(vpcs.Field("manager_id")))
	if usableVpc {
		sq = sq.Filter(sqlchemy.Equals(vpcs.Field("status"), api.VPC_STATUS_AVAILABLE))
	}

	return sq.SubQuery()
}

/*
Query 3:
vpc.manager_id is not empty && wire.zone_id is empty

2019.01.17 目前华为云子网在整个region 可用。wire中zone_id留空。
*/
func usableZoneQ3(providers, vpcs, wires, networks, zones *sqlchemy.SSubQuery, usableNet, usableVpc bool) *sqlchemy.SSubQuery {
	// join tables
	sq := zones.Query(sqlchemy.DISTINCT("zone_id", zones.Field("id")))
	sq = sq.Join(vpcs, sqlchemy.Equals(zones.Field("cloudregion_id"), vpcs.Field("cloudregion_id")))
	sq = sq.Join(wires, sqlchemy.Equals(wires.Field("vpc_id"), vpcs.Field("id")))
	if usableNet {
		sq = sq.Join(networks, sqlchemy.Equals(wires.Field("id"), networks.Field("wire_id")))
	}
	sq = sq.Join(providers, sqlchemy.Equals(vpcs.Field("manager_id"), providers.Field("id")))

	// add filters
	if usableNet {
		sq = sq.Filter(sqlchemy.Equals(networks.Field("status"), api.NETWORK_STATUS_AVAILABLE))
	}
	sq = sq.Filter(sqlchemy.IsNullOrEmpty(wires.Field("zone_id")))
	sq = sq.Filter(sqlchemy.IsTrue(providers.Field("enabled")))
	sq = sq.Filter(sqlchemy.In(providers.Field("status"), api.CLOUD_PROVIDER_VALID_STATUS))
	sq = sq.Filter(sqlchemy.In(providers.Field("health_status"), api.CLOUD_PROVIDER_VALID_HEALTH_STATUS))
	if usableVpc {
		sq = sq.Filter(sqlchemy.Equals(vpcs.Field("status"), api.VPC_STATUS_AVAILABLE))
	}

	return sq.SubQuery()
}

/*
Query 4:
vpc.manager_id is empty && wire.zone_id is empty

2019.01.17 目前华为云子网在整个region 可用。wire中zone_id留空。
*/
func usableZoneQ4(vpcs, wires, networks, zones *sqlchemy.SSubQuery, usableNet, usableVpc bool) *sqlchemy.SSubQuery {
	// join tables
	sq := zones.Query(sqlchemy.DISTINCT("zone_id", zones.Field("id")))
	sq = sq.Join(vpcs, sqlchemy.Equals(zones.Field("cloudregion_id"), vpcs.Field("cloudregion_id")))
	sq = sq.Join(wires, sqlchemy.Equals(wires.Field("vpc_id"), vpcs.Field("id")))
	if usableNet {
		sq = sq.Join(networks, sqlchemy.Equals(wires.Field("id"), networks.Field("wire_id")))
	}

	// add filters
	if usableNet {
		sq = sq.Filter(sqlchemy.Equals(networks.Field("status"), api.NETWORK_STATUS_AVAILABLE))
	}
	sq = sq.Filter(sqlchemy.IsNullOrEmpty(wires.Field("zone_id")))
	sq = sq.Filter(sqlchemy.IsNullOrEmpty(vpcs.Field("manager_id")))
	if usableVpc {
		sq = sq.Filter(sqlchemy.Equals(vpcs.Field("status"), api.VPC_STATUS_AVAILABLE))
	}

	return sq.SubQuery()
}

func networkUsableZoneQueries(usableNet, usableVpc bool) []*sqlchemy.SSubQuery {
	queries := make([]*sqlchemy.SSubQuery, 4)
	queries[0] = usableZoneQ1(CloudproviderManager.Query().SubQuery(),
		VpcManager.Query().SubQuery(),
		WireManager.Query().SubQuery(),
		NetworkManager.Query().SubQuery(),
		usableNet, usableVpc)
	queries[1] = usableZoneQ2(VpcManager.Query().SubQuery(),
		WireManager.Query().SubQuery(),
		NetworkManager.Query().SubQuery(),
		usableNet, usableVpc)
	queries[2] = usableZoneQ3(CloudproviderManager.Query().SubQuery(),
		VpcManager.Query().SubQuery(),
		WireManager.Query().SubQuery(),
		NetworkManager.Query().SubQuery(),
		ZoneManager.Query().SubQuery(),
		usableNet, usableVpc)
	queries[3] = usableZoneQ4(VpcManager.Query().SubQuery(),
		WireManager.Query().SubQuery(),
		NetworkManager.Query().SubQuery(),
		ZoneManager.Query().SubQuery(),
		usableNet, usableVpc)
	return queries
}

func NetworkUsableZoneQueries(field sqlchemy.IQueryField, usableNet, usableVpc bool) []sqlchemy.ICondition {
	queries := networkUsableZoneQueries(usableNet, usableVpc)
	iconditions := make([]sqlchemy.ICondition, 0)
	for i := range queries {
		iconditions = append(iconditions, sqlchemy.In(field, queries[i]))
	}

	return iconditions
}

func (manager *SZoneManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}

	cloudEnvStr, _ := query.GetString("cloud_env")
	if cloudEnvStr == api.CLOUD_ENV_PRIVATE_CLOUD || jsonutils.QueryBoolean(query, "is_private", false) || jsonutils.QueryBoolean(query, "private", false) || jsonutils.QueryBoolean(query, "private_cloud", false) {
		regions := CloudregionManager.Query().SubQuery()
		subq := regions.Query(regions.Field("id"))
		subq = subq.Filter(sqlchemy.In(regions.Field("provider"), cloudprovider.GetPrivateProviders()))
		q = q.In("cloudregion_id", subq.SubQuery())
	}
	if cloudEnvStr == api.CLOUD_ENV_PUBLIC_CLOUD || jsonutils.QueryBoolean(query, "is_public", false) || jsonutils.QueryBoolean(query, "public", false) || jsonutils.QueryBoolean(query, "public_cloud", false) {
		regions := CloudregionManager.Query().SubQuery()
		subq := regions.Query(regions.Field("id"))
		subq = subq.Filter(sqlchemy.In(regions.Field("provider"), cloudprovider.GetPublicProviders()))
		q = q.In("cloudregion_id", subq.SubQuery())
	}
	if cloudEnvStr == api.CLOUD_ENV_ON_PREMISE || jsonutils.QueryBoolean(query, "is_on_premise", false) {
		regions := CloudregionManager.Query().SubQuery()
		subq := regions.Query(regions.Field("id"))
		subq = subq.Filter(sqlchemy.OR(
			sqlchemy.In(regions.Field("provider"), cloudprovider.GetOnPremiseProviders()),
			sqlchemy.Equals(regions.Field("provider"), api.CLOUD_PROVIDER_ONECLOUD),
		))
		q = q.In("cloudregion_id", subq.SubQuery())
	}
	if cloudEnvStr == api.CLOUD_ENV_PRIVATE_ON_PREMISE {
		regions := CloudregionManager.Query().SubQuery()
		subq := regions.Query(regions.Field("id"))
		subq = subq.Filter(sqlchemy.OR(
			sqlchemy.In(regions.Field("provider"), cloudprovider.GetPrivateProviders()),
			sqlchemy.In(regions.Field("provider"), cloudprovider.GetOnPremiseProviders()),
			sqlchemy.Equals(regions.Field("provider"), api.CLOUD_PROVIDER_ONECLOUD),
		))
		q = q.In("cloudregion_id", subq.SubQuery())
	}
	if jsonutils.QueryBoolean(query, "is_managed", false) {
		q = q.IsNotEmpty("external_id")
	}

	domainId, err := db.FetchQueryDomain(ctx, userCred, query)
	if len(domainId) > 0 {
		q = q.In("cloudregion_id", getCloudRegionIdByDomainId(domainId))
	}

	if jsonutils.QueryBoolean(query, "usable", false) || jsonutils.QueryBoolean(query, "usable_vpc", false) {
		usableNet := jsonutils.QueryBoolean(query, "usable", false)
		usableVpc := jsonutils.QueryBoolean(query, "usable_vpc",
			false)
		iconditions := NetworkUsableZoneQueries(q.Field("id"), usableNet, usableVpc)
		q = q.Filter(sqlchemy.OR(iconditions...))
		q = q.Equals("status", api.ZONE_ENABLE)

		service, _ := query.GetString("service")
		switch service {
		case ElasticcacheManager.KeywordPlural():
			q2 := ElasticcacheSkuManager.Query("zone_id").Distinct()
			statusFilter := sqlchemy.OR(sqlchemy.Equals(q2.Field("prepaid_status"), api.SkuStatusAvailable), sqlchemy.Equals(q2.Field("postpaid_status"), api.SkuStatusAvailable))
			skusSQ := q2.Filter(statusFilter).SubQuery()
			q = q.In("id", skusSQ)
		default:
			break
		}
	}

	managerStr, _ := query.GetString("manager")
	if len(managerStr) > 0 {
		subq := CloudproviderRegionManager.QueryRelatedRegionIds("", managerStr)
		q = q.In("cloudregion_id", subq)
	}
	accountStr, _ := query.GetString("account")
	if len(accountStr) > 0 {
		subq := CloudproviderRegionManager.QueryRelatedRegionIds(accountStr)
		q = q.In("cloudregion_id", subq)
	}

	providerStrs := jsonutils.GetQueryStringArray(query, "provider")
	if len(providerStrs) > 0 {
		query.(*jsonutils.JSONDict).Remove("provider")
		subq := queryCloudregionIdsByProviders("provider", providerStrs)
		q = q.In("cloudregion_id", subq.SubQuery())
	}

	brandStrs := jsonutils.GetQueryStringArray(query, "brand")
	if len(brandStrs) > 0 {
		query.(*jsonutils.JSONDict).Remove("brand")
		subq := queryCloudregionIdsByProviders("brand", brandStrs)
		q = q.In("cloudregion_id", subq.SubQuery())
	}

	city, _ := query.GetString("city")
	if len(city) > 0 {
		subq := CloudregionManager.Query("id").Equals("city", city).SubQuery()
		q = q.In("cloudregion_id", subq)
	}

	return q, nil
}

func (self *SZone) AllowGetDetailsCapability(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (self *SZone) GetDetailsCapability(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	capa, err := GetCapabilities(ctx, userCred, query, nil, self)
	if err != nil {
		return nil, err
	}
	return jsonutils.Marshal(&capa), nil
}

func (self *SZone) isManaged() bool {
	region := self.GetRegion()
	if region != nil && len(region.ExternalId) == 0 {
		return false
	} else {
		return true
	}
}

func (self *SZone) isSchedPolicySupported() bool {
	return !self.isManaged()
}

func (self *SZone) getMinNicCount() int {
	return options.Options.MinNicCount
}

func (self *SZone) getMaxNicCount() int {
	if self.isManaged() {
		return options.Options.MaxManagedNicCount
	} else {
		return options.Options.MaxNormalNicCount
	}
}

func (self *SZone) getMinDataDiskCount() int {
	return options.Options.MinDataDiskCount
}

func (self *SZone) getMaxDataDiskCount() int {
	return options.Options.MaxDataDiskCount
}

func (manager *SZoneManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	regionStr := jsonutils.GetAnyString(data, []string{"region", "region_id", "cloudregion", "cloudregion_id"})
	var regionId string
	if len(regionStr) > 0 {
		regionObj, err := CloudregionManager.FetchByIdOrName(nil, regionStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError("Region %s not found", regionStr)
			} else {
				return nil, httperrors.NewInternalServerError("Query region %s fail %s", regionStr, err)
			}
		}
		regionId = regionObj.GetId()
	} else {
		regionId = "default"
	}
	data.Add(jsonutils.NewString(regionId), "cloudregion_id")
	data.Set("status", jsonutils.NewString(api.ZONE_ENABLE))
	return manager.SStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data)
}
