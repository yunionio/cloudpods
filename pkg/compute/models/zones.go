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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SZoneManager struct {
	db.SStatusStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
	SI18nResourceBaseManager
	SCloudregionResourceBaseManager
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
	SI18nResourceBase
	SCloudregionResourceBase `width:"36" charset:"ascii" nullable:"false" list:"user" create:"admin_required"`

	Location   string `width:"256" charset:"utf8" get:"user" list:"user" update:"admin"`
	Contacts   string `width:"256" charset:"utf8" get:"user" update:"admin"`
	NameCn     string `width:"256" charset:"utf8"`
	ManagerUri string `width:"256" charset:"ascii" list:"admin" update:"admin"`
}

func (manager *SZoneManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{CloudregionManager},
	}
}

func (zone *SZone) ValidateDeleteCondition(ctx context.Context, info *api.ZoneDetails) error {
	var usage api.ZoneGeneralUsage
	if info != nil {
		usage = info.ZoneGeneralUsage
	} else {
		usage = zone.GeneralUsage(ctx)
	}
	if !usage.IsEmpty() {
		return httperrors.NewNotEmptyError("not empty zone: %s", zone.Id)
	}
	return zone.SStandaloneResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (manager *SZoneManager) Count() (int, error) {
	return manager.Query().CountWithError()
}

func (zone *SZone) GeneralUsage(ctx context.Context) api.ZoneGeneralUsage {
	usage := api.ZoneGeneralUsage{}
	usage.Hosts, _ = zone.HostCount("", "", tristate.None, "", tristate.None)
	usage.HostsEnabled, _ = zone.HostCount("", "", tristate.True, "", tristate.None)
	usage.Baremetals, _ = zone.HostCount("", "", tristate.None, "", tristate.True)
	usage.BaremetalsEnabled, _ = zone.HostCount("", "", tristate.True, "", tristate.True)
	usage.Wires, _ = zone.getWireCount()
	usage.Networks, _ = zone.getNetworkCount(ctx)
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

func (zone *SZone) getNetworkCount(ctx context.Context) (int, error) {
	return getNetworkCount(ctx, nil, nil, rbacscope.ScopeSystem, nil, zone)
}

func (manager *SZoneManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ZoneDetails {
	rows := make([]api.ZoneDetails, len(objs))

	stdRows := manager.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	zoneIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.ZoneDetails{
			StatusStandaloneResourceDetails: stdRows[i],
			CloudregionResourceInfo:         regRows[i],
		}
		zone := objs[i].(*SZone)
		zoneIds[i] = zone.Id
	}
	regions := []SCloudregion{}
	zoneQ := ZoneManager.Query().SubQuery()
	q := CloudregionManager.Query()
	q = q.Join(zoneQ, sqlchemy.Equals(zoneQ.Field("cloudregion_id"), q.Field("id"))).Filter(sqlchemy.In(zoneQ.Field("id"), zoneIds))
	err := db.FetchModelObjects(CloudregionManager, q, &regions)
	if err != nil {
		return rows
	}
	regionMap := map[string]SCloudregion{}
	for i := range regions {
		regionMap[regions[i].Id] = regions[i]
	}
	zoneUsages := map[string]*api.ZoneGeneralUsage{}
	q = HostManager.Query().In("zone_id", zoneIds)
	hosts := []SHost{}
	err = db.FetchModelObjects(HostManager, q, &hosts)
	if err != nil {
		return rows
	}
	for i := range hosts {
		_, ok := zoneUsages[hosts[i].ZoneId]
		if !ok {
			zoneUsages[hosts[i].ZoneId] = &api.ZoneGeneralUsage{}
		}
		zoneUsages[hosts[i].ZoneId].Hosts += 1
		if hosts[i].GetEnabled() {
			zoneUsages[hosts[i].ZoneId].HostsEnabled += 1
		}
		if hosts[i].IsBaremetal {
			zoneUsages[hosts[i].ZoneId].Baremetals += 1
			if hosts[i].GetEnabled() {
				zoneUsages[hosts[i].ZoneId].BaremetalsEnabled += 1
			}
		}
	}
	wires := []SWire{}
	q = WireManager.Query().In("zone_id", zoneIds)
	err = db.FetchModelObjects(WireManager, q, &wires)
	if err != nil {
		return rows
	}
	wireMap := map[string]string{}
	for i := range wires {
		_, ok := zoneUsages[wires[i].ZoneId]
		if !ok {
			zoneUsages[wires[i].ZoneId] = &api.ZoneGeneralUsage{}
		}
		zoneUsages[wires[i].ZoneId].Wires += 1
		wireMap[wires[i].Id] = wires[i].ZoneId
	}
	networks := []SNetwork{}
	wireQ := WireManager.Query().SubQuery()
	q = NetworkManager.Query()
	q = q.Join(wireQ, sqlchemy.Equals(q.Field("wire_id"), wireQ.Field("id"))).Filter(sqlchemy.In(wireQ.Field("zone_id"), zoneIds))
	err = db.FetchModelObjects(NetworkManager, q, &networks)
	if err != nil {
		return rows
	}
	for i := range networks {
		zoneId, ok := wireMap[networks[i].WireId]
		if !ok {
			continue
		}
		_, ok = zoneUsages[zoneId]
		if !ok {
			zoneUsages[zoneId] = &api.ZoneGeneralUsage{}
		}
		zoneUsages[zoneId].Networks += 1
	}

	storages := []SStorage{}
	q = StorageManager.Query().In("zone_id", zoneIds)
	err = db.FetchModelObjects(StorageManager, q, &storages)
	if err != nil {
		return rows
	}
	for i := range storages {
		_, ok := zoneUsages[storages[i].ZoneId]
		if !ok {
			zoneUsages[storages[i].ZoneId] = &api.ZoneGeneralUsage{}
		}
		zoneUsages[storages[i].ZoneId].Storages += 1
	}
	for i := range rows {
		usage, ok := zoneUsages[zoneIds[i]]
		if ok {
			rows[i].ZoneGeneralUsage = *usage
		}
		zone := objs[i].(*SZone)
		region, ok := regionMap[zone.CloudregionId]
		if ok {
			rows[i].CloudenvResourceInfo = region.GetRegionCloudenvInfo()
		}
	}
	return rows
}

func (zone *SZone) GetCloudproviderId() string {
	return ""
}

func (zone *SZone) GetCloudRegionId() string {
	if len(zone.CloudregionId) == 0 {
		return "default"
	} else {
		return zone.CloudregionId
	}
}

func (zone *SZone) GetI18N(ctx context.Context) *jsonutils.JSONDict {
	return zone.GetModelI18N(ctx, zone)
}

func (manager *SZoneManager) SyncZones(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	region *SCloudregion,
	zones []cloudprovider.ICloudZone,
	provider *SCloudprovider,
	xor bool,
) ([]SZone, []cloudprovider.ICloudZone, compare.SyncResult) {
	lockman.LockRawObject(ctx, "zones", region.Id)
	defer lockman.ReleaseRawObject(ctx, "zones", region.Id)

	localZones := make([]SZone, 0)
	remoteZones := make([]cloudprovider.ICloudZone, 0)
	syncResult := compare.SyncResult{}

	dbZones, err := region.GetZones()
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
		err = removed[i].syncRemoveCloudZone(ctx, userCred, provider)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}
	for i := 0; i < len(commondb); i += 1 {
		if !xor {
			err = commondb[i].syncWithCloudZone(ctx, userCred, commonext[i], region)
			if err != nil {
				syncResult.UpdateError(err)
			}
		}
		localZones = append(localZones, commondb[i])
		remoteZones = append(remoteZones, commonext[i])
		syncResult.Update()
	}
	for i := 0; i < len(added); i += 1 {
		zone, err := manager.newFromCloudZone(ctx, userCred, added[i], region)
		if err != nil {
			syncResult.AddError(err)
			continue
		}
		localZones = append(localZones, *zone)
		remoteZones = append(remoteZones, added[i])
		syncResult.Add()
	}

	return localZones, remoteZones, syncResult
}

func (self *SZone) syncRemoveCloudZone(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	cnt, err := self.getNetworkCount(ctx)
	if err != nil {
		return errors.Wrapf(err, "getNetworkCount")
	}
	if cnt > 0 {
		return httperrors.NewNotEmptyError("contains %d networks", cnt)
	}
	return self.purgeAll(ctx, provider.Id)
}

func (self *SZone) syncWithCloudZone(ctx context.Context, userCred mcclient.TokenCredential, extZone cloudprovider.ICloudZone, region *SCloudregion) error {
	err := ZoneManager.SyncI18ns(ctx, userCred, self, extZone.GetI18n())
	if err != nil {
		return errors.Wrap(err, "SyncI18ns")
	}

	diff, err := db.UpdateWithLock(ctx, self, func() error {
		if options.Options.EnableSyncName {
			newName, _ := db.GenerateAlterName(self, extZone.GetName())
			if len(newName) > 0 && newName != self.Name {
				self.Name = newName
			}
		}
		self.Status = extZone.GetStatus()

		self.IsEmulated = extZone.IsEmulated()
		self.CloudregionId = region.Id

		return nil
	})
	if err != nil {
		log.Errorf("syncWithCloudZone error %s", err)
		return err
	}
	syncMetadata(ctx, userCred, self, extZone, false)
	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return nil
}

func (manager *SZoneManager) newFromCloudZone(ctx context.Context, userCred mcclient.TokenCredential, extZone cloudprovider.ICloudZone, region *SCloudregion) (*SZone, error) {
	zone := SZone{}
	zone.SetModelManager(manager, &zone)

	zone.Status = extZone.GetStatus()
	zone.ExternalId = extZone.GetGlobalId()

	zone.IsEmulated = extZone.IsEmulated()

	zone.CloudregionId = region.Id

	var err = func() error {
		lockman.LockRawObject(ctx, manager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, manager.Keyword(), "name")

		newName, err := db.GenerateName(ctx, manager, userCred, extZone.GetName())
		if err != nil {
			return err
		}
		zone.Name = newName

		return manager.TableSpec().Insert(ctx, &zone)
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}

	err = manager.SyncI18ns(ctx, userCred, &zone, extZone.GetI18n())
	if err != nil {
		return nil, errors.Wrap(err, "SyncI18ns")
	}
	syncMetadata(ctx, userCred, &zone, extZone, false)

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

func (zone *SZone) GetRegion() (*SCloudregion, error) {
	region, err := CloudregionManager.FetchById(zone.GetCloudRegionId())
	if err != nil {
		return nil, err
	}
	return region.(*SCloudregion), nil
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
wire.zone_id is not empty
*/
func usableZoneQ1(vpcs map[string]map[string]string, usableNet bool) ([]string, error) {
	// conditions
	vpcIds := make([]string, 0)
	for k, _ := range vpcs {
		vpcIds = append(vpcIds, k)
	}

	q := zoneWireFilter(usableNet, false, vpcIds)
	results, _, err := zoneWireFilterResult(q)
	if err != nil {
		return nil, errors.Wrap(err, "zoneWireFilterResult")
	}

	return results, nil
}

/*
Query 2:
wire.zone_id is empty

2019.01.17 目前华为云子网在整个region 可用。wire中zone_id留空。
*/
func usableZoneQ2(vpcs map[string]map[string]string, usableNet bool) ([]string, error) {
	// conditions
	vpcIds := make([]string, 0)
	for k, _ := range vpcs {
		vpcIds = append(vpcIds, k)
	}

	q := zoneWireFilter(usableNet, true, vpcIds)
	_, vpcIds, err := zoneWireFilterResult(q)
	if err != nil {
		return nil, errors.Wrap(err, "zoneWireFilterResult")
	}

	cloudregionIds := make([]string, 0)
	for i := range vpcIds {
		if vpc, ok := vpcs[vpcIds[i]]; ok {
			if c, ok := vpc["cloudregion_id"]; ok && len(c) > 0 {
				cloudregionIds = append(cloudregionIds, c)
			}
		}
	}

	return zoneRegionFilter(cloudregionIds)
}

func zoneVpcFilter(usableVpc bool, query *api.ZoneListInput) *sqlchemy.SQuery {
	// conditions
	q := VpcManager.Query("id", "cloudregion_id")
	if usableVpc {
		q = q.Equals("status", api.VPC_STATUS_AVAILABLE)
	}

	if query != nil && len(query.CloudregionId) > 0 {
		q.Equals("cloudregion_id", query.CloudregionId)
	}

	return q
}

// vpc.manager_id is empty
func usableVpc1(usableVpc bool, query *api.ZoneListInput) *sqlchemy.SQuery {
	// conditions
	return zoneVpcFilter(usableVpc, query).IsNullOrEmpty("manager_id")
}

// vpc.manager_id is not empty
func usableVpc2(usableVpc bool, query *api.ZoneListInput) (*sqlchemy.SQuery, error) {
	// conditions
	q := zoneVpcFilter(usableVpc, query).IsNotEmpty("manager_id")
	if query != nil && len(query.CloudproviderId) > 0 {
		q.Equals("manager_id", query.CloudproviderId)
	} else {
		providerIds, err := zoneUsableProviderQuery(query)
		if err != nil {
			return nil, errors.Wrap(err, "zoneUsableProviderQuery")
		}

		q.In("manager_id", providerIds)
	}

	return q, nil
}

func zoneUsableVpc(usableVpc bool, query *api.ZoneListInput) (map[string]map[string]string, error) {
	q1 := usableVpc1(usableVpc, query)
	q2, err := usableVpc2(usableVpc, query)
	if err != nil {
		return nil, errors.Wrap(err, "usableVpc2")
	}
	results := make(map[string]map[string]string, 0)
	for _, q := range []*sqlchemy.SQuery{q1, q2} {
		rows, err := q.Rows()
		if err != nil && err != sql.ErrNoRows {
			return nil, errors.Wrap(err, "zoneUsableVpc.rows")
		}

		for rows.Next() {
			var id string
			var cloudregion_id string
			err := rows.Scan(&id, &cloudregion_id)
			if err != nil {
				return nil, errors.Wrap(err, "rows.Scan")
			}

			results[id] = map[string]string{"id": id, "cloudregion_id": cloudregion_id}
		}
		rows.Close()
	}

	return results, nil
}

func zoneWireFilter(usableNet bool, zoneIdIsEmpty bool, vpcIds []string) *sqlchemy.SQuery {
	wireq := WireManager.Query("vpc_id", "zone_id").In("vpc_id", vpcIds)
	if zoneIdIsEmpty {
		wireq.IsNullOrEmpty("zone_id")
	} else {
		wireq.IsNotEmpty("zone_id")
	}

	if usableNet {
		netq := NetworkManager.Query().Equals("status", api.NETWORK_STATUS_AVAILABLE).SubQuery()
		wireq = wireq.Join(netq, sqlchemy.Equals(netq.Field("wire_id"), wireq.Field("id")))
	}

	return wireq
}

func zoneWireFilterResult(q *sqlchemy.SQuery) ([]string, []string, error) {
	rows, err := q.Rows()
	if err != nil && err != sql.ErrNoRows {
		return nil, nil, errors.Wrap(err, "zoneWireFilterResult.rows")
	}
	defer rows.Close()
	zoneIds := make([]string, 0)
	vpcIds := make([]string, 0)
	for rows.Next() {
		var vpcId string
		var zoneId string
		err := rows.Scan(&vpcId, &zoneId)
		if err != nil {
			return nil, nil, errors.Wrap(err, "rows.Scan")
		}
		zoneIds = append(zoneIds, zoneId)
		vpcIds = append(vpcIds, vpcId)
	}

	return zoneIds, vpcIds, nil
}

// avaliable providers
func zoneUsableProviderQuery(query *api.ZoneListInput) ([]string, error) {
	q := usableCloudProviders()
	if query != nil {
		if len(query.CloudproviderId) > 0 {
			q = q.Filter(sqlchemy.OR(sqlchemy.Equals(q.Field("id"), query.CloudproviderId), sqlchemy.Equals(q.Field("name"), query.CloudproviderId)))
		}
	}

	return filterResult(q)
}

func zoneRegionFilter(cloudregionIds []string) ([]string, error) {
	q := ZoneManager.Query("id").In("cloudregion_id", cloudregionIds)
	return filterResult(q)
}

func filterResult(q *sqlchemy.SQuery) ([]string, error) {
	rows, err := q.Rows()
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "filterResult.rows")
	}
	defer rows.Close()
	results := make([]string, 0)
	for rows.Next() {
		var id string
		err := rows.Scan(&id)
		if err != nil {
			return nil, errors.Wrap(err, "filterResult.Scan")
		}
		results = append(results, id)
	}

	return results, nil
}

func NetworkUsableZoneIds(usableNet, usableVpc bool, query *api.ZoneListInput) ([]string, error) {
	vpcs, err := zoneUsableVpc(usableVpc, query)
	if err != nil {
		return nil, errors.Wrap(err, "zoneUsableVpc")
	}
	r1, err := usableZoneQ1(vpcs, usableNet)
	if err != nil {
		return nil, errors.Wrap(err, "usableZoneQ1")
	}
	r2, err := usableZoneQ2(vpcs, usableNet)
	if err != nil {
		return nil, errors.Wrap(err, "usableZoneQ2")
	}

	r1 = append(r1, r2...)
	// remove dupliates
	ret := make([]string, 0)
	rm := make(map[string]bool)
	for _, zone := range r1 {
		if _, ok := rm[zone]; !ok {
			rm[zone] = true
			ret = append(ret, zone)
		}
	}
	return ret, nil
}

// 可用区列表
func (manager *SZoneManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ZoneListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}

	cloudEnvStr := query.CloudEnv
	if cloudEnvStr == api.CLOUD_ENV_PRIVATE_CLOUD {
		regions := CloudregionManager.Query().SubQuery()
		subq := regions.Query(regions.Field("id"))
		subq = subq.Filter(sqlchemy.In(regions.Field("provider"), cloudprovider.GetPrivateProviders()))
		q = q.In("cloudregion_id", subq.SubQuery())
	}
	if cloudEnvStr == api.CLOUD_ENV_PUBLIC_CLOUD {
		regions := CloudregionManager.Query().SubQuery()
		subq := regions.Query(regions.Field("id"))
		subq = subq.Filter(sqlchemy.In(regions.Field("provider"), cloudprovider.GetPublicProviders()))
		q = q.In("cloudregion_id", subq.SubQuery())
	}
	if cloudEnvStr == api.CLOUD_ENV_ON_PREMISE {
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
	if query.IsManaged != nil {
		if *query.IsManaged {
			q = q.IsNotEmpty("external_id")
		} else {
			q = q.IsNullOrEmpty("external_id")
		}
	}

	data := jsonutils.Marshal(query.DomainizedResourceListInput)
	domainId, err := db.FetchQueryDomain(ctx, userCred, data)
	if len(domainId) > 0 {
		q = q.In("cloudregion_id", getCloudRegionIdByDomainId(domainId))
	}

	usableNet := (query.Usable != nil && *query.Usable)
	usableVpc := (query.UsableVpc != nil && *query.UsableVpc)
	if usableNet || usableVpc {
		zoneIds, err := NetworkUsableZoneIds(usableNet, usableVpc, &query)
		if err != nil {
			return nil, err
		}
		q = q.In("id", zoneIds)
		q = q.Equals("status", api.ZONE_ENABLE)

		service := query.Service
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

	managerStr := query.CloudproviderId
	if len(managerStr) > 0 {
		subq := CloudproviderRegionManager.QueryRelatedRegionIds(nil, managerStr...)
		q = q.In("cloudregion_id", subq)
	}
	accountArr := query.CloudaccountId
	if len(accountArr) > 0 {
		subq := CloudproviderRegionManager.QueryRelatedRegionIds(accountArr)
		q = q.In("cloudregion_id", subq)
	}

	providerStrs := query.Providers
	if len(providerStrs) > 0 {
		subq := queryCloudregionIdsByProviders("provider", providerStrs)
		q = q.In("cloudregion_id", subq.SubQuery())
	}

	brandStrs := query.Brands
	if len(brandStrs) > 0 {
		subq := queryCloudregionIdsByProviders("brand", brandStrs)
		q = q.In("cloudregion_id", subq.SubQuery())
	}

	q, err = managedResourceFilterByRegion(ctx, q, query.RegionalFilterListInput, "", nil)

	if len(query.Location) > 0 {
		q = q.In("location", query.Location)
	}
	if len(query.Contacts) > 0 {
		q = q.In("contacts", query.Contacts)
	}

	return q, nil
}

func (manager *SZoneManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ZoneListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SCloudregionResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.OrderByExtraFields")
	}

	if db.NeedOrderQuery([]string{query.OrderByWires}) {
		wireQ := WireManager.Query()
		wireQ = wireQ.AppendField(wireQ.Field("zone_id"), sqlchemy.COUNT("wire_count", wireQ.Field("zone_id")))
		wireQ = wireQ.GroupBy(wireQ.Field("zone_id"))
		wireSQ := wireQ.SubQuery()
		q = q.LeftJoin(wireSQ, sqlchemy.Equals(wireSQ.Field("zone_id"), q.Field("id")))
		q = q.AppendField(q.QueryFields()...)
		q = q.AppendField(wireSQ.Field("wire_count"))
		q = db.OrderByFields(q, []string{query.OrderByWires}, []sqlchemy.IQueryField{q.Field("wire_count")})
	}
	if db.NeedOrderQuery([]string{query.OrderByHosts}) {
		hostQ := HostManager.Query()
		hostQ = hostQ.AppendField(hostQ.Field("zone_id"), sqlchemy.COUNT("host_count", hostQ.Field("zone_id")))
		hostQ = hostQ.GroupBy(hostQ.Field("zone_id"))
		hostSQ := hostQ.SubQuery()
		q = q.LeftJoin(hostSQ, sqlchemy.Equals(hostSQ.Field("zone_id"), q.Field("id")))
		q = q.AppendField(q.QueryFields()...)
		q = q.AppendField(hostSQ.Field("host_count"))
		q = db.OrderByFields(q, []string{query.OrderByHosts}, []sqlchemy.IQueryField{q.Field("host_count")})
	}
	if db.NeedOrderQuery([]string{query.OrderByHostsEnabled}) {
		hostQ := HostManager.Query()
		hostQ = hostQ.Filter(sqlchemy.Equals(hostQ.Field("enabled"), true))
		hostQ = hostQ.AppendField(hostQ.Field("zone_id"), sqlchemy.COUNT("host_count", hostQ.Field("zone_id")))
		hostQ = hostQ.GroupBy(hostQ.Field("zone_id"))
		hostSQ := hostQ.SubQuery()
		q = q.LeftJoin(hostSQ, sqlchemy.Equals(hostSQ.Field("zone_id"), q.Field("id")))
		q = q.AppendField(q.QueryFields()...)
		q = q.AppendField(hostSQ.Field("host_count"))
		q = db.OrderByFields(q, []string{query.OrderByHostsEnabled}, []sqlchemy.IQueryField{q.Field("host_count")})
	}
	if db.NeedOrderQuery([]string{query.OrderByBaremetals}) {
		hostQ := HostManager.Query()
		hostQ = hostQ.Filter(sqlchemy.Equals(hostQ.Field("is_baremetal"), true))
		hostQ = hostQ.AppendField(hostQ.Field("zone_id"), sqlchemy.COUNT("host_count", hostQ.Field("zone_id")))
		hostQ = hostQ.GroupBy(hostQ.Field("zone_id"))
		hostSQ := hostQ.SubQuery()
		q = q.LeftJoin(hostSQ, sqlchemy.Equals(hostSQ.Field("zone_id"), q.Field("id")))
		q = q.AppendField(q.QueryFields()...)
		q = q.AppendField(hostSQ.Field("host_count"))
		q = db.OrderByFields(q, []string{query.OrderByBaremetals}, []sqlchemy.IQueryField{q.Field("host_count")})
	}
	if db.NeedOrderQuery([]string{query.OrderByBaremetalsEnabled}) {
		hostQ := HostManager.Query()
		hostQ = hostQ.Filter(sqlchemy.Equals(hostQ.Field("is_baremetal"), true))
		hostQ = hostQ.Filter(sqlchemy.Equals(hostQ.Field("enabled"), true))
		hostQ = hostQ.AppendField(hostQ.Field("zone_id"), sqlchemy.COUNT("host_count", hostQ.Field("zone_id")))
		hostQ = hostQ.GroupBy(hostQ.Field("zone_id"))
		hostSQ := hostQ.SubQuery()
		q = q.LeftJoin(hostSQ, sqlchemy.Equals(hostSQ.Field("zone_id"), q.Field("id")))
		q = q.AppendField(q.QueryFields()...)
		q = q.AppendField(hostSQ.Field("host_count"))
		q = db.OrderByFields(q, []string{query.OrderByBaremetalsEnabled}, []sqlchemy.IQueryField{q.Field("host_count")})
	}
	return q, nil
}

func (manager *SZoneManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	if field == "zone" {
		q = q.AppendField(q.Field("name").Label("zone"))
		q = q.GroupBy(q.Field("name"))
		return q, nil
	}
	q, err = manager.SStatusStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SCloudregionResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (self *SZone) GetDetailsCapability(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	capa, err := GetCapabilities(ctx, userCred, query, nil, self)
	if err != nil {
		return nil, err
	}
	return jsonutils.Marshal(&capa), nil
}

func (self *SZone) GetDetailsDiskCapability(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	capa, err := GetDiskCapabilities(ctx, userCred, query, nil, self)
	if err != nil {
		return nil, err
	}
	return jsonutils.Marshal(&capa), nil
}

func (self *SZone) isManaged() bool {
	region, _ := self.GetRegion()
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

func (manager *SZoneManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.ZoneCreateInput) (*jsonutils.JSONDict, error) {
	for _, cloudregion := range []string{input.Cloudregion, input.Region, input.RegionId, input.CloudregionId, "default"} {
		if len(cloudregion) > 0 {
			input.Cloudregion = cloudregion
			break
		}
	}
	_region, err := CloudregionManager.FetchByIdOrName(ctx, nil, input.Cloudregion)
	if err != nil {
		if err != sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError("failed to found cloudregion %s", input.Cloudregion)
		}
		return nil, httperrors.NewGeneralError(err)
	}
	region := _region.(*SCloudregion)
	input.CloudregionId = region.Id
	input.Status = api.ZONE_ENABLE
	if region.Provider != api.CLOUD_PROVIDER_ONECLOUD {
		return nil, httperrors.NewNotSupportedError("not support create %s zone", region.Provider)
	}

	input.StatusStandaloneResourceCreateInput, err = manager.SStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StatusStandaloneResourceCreateInput)
	if err != nil {
		return nil, err
	}

	return input.JSON(input), nil
}

func (self *SZone) GetSchedtags() []SSchedtag {
	return GetSchedtags(ZoneschedtagManager, self.Id)
}

func (self *SZone) GetDynamicConditionInput() *jsonutils.JSONDict {
	return jsonutils.Marshal(self).(*jsonutils.JSONDict)
}

func (self *SZone) PerformSetSchedtag(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return PerformSetResourceSchedtag(self, ctx, userCred, query, data)
}

func (self *SZone) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.ZonePurgeInput) (jsonutils.JSONObject, error) {
	if len(input.ManagerId) == 0 {
		return nil, httperrors.NewMissingParameterError("manager_id")
	}
	return nil, self.purgeAll(ctx, input.ManagerId)
}

func (self *SZone) GetSchedtagJointManager() ISchedtagJointManager {
	return ZoneschedtagManager
}

func (self *SZone) ClearSchedDescCache() error {
	hosts := make([]SHost, 0)
	q := HostManager.Query().Equals("zone_id", self.Id)
	err := db.FetchModelObjects(HostManager, q, &hosts)
	if err != nil {
		return errors.Wrapf(err, "fetch hosts by zone_id %s", self.Id)
	}
	for i := range hosts {
		err := hosts[i].ClearSchedDescCache()
		if err != nil {
			return errors.Wrapf(err, "clean host %s sched cache", hosts[i].GetName())
		}
	}
	return nil
}

func (manager *SZoneManager) getOnpremiseZoneIds() ([]string, error) {
	return zoneRegionFilter([]string{api.DEFAULT_REGION_ID})
}
