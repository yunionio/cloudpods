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
	"strings"

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
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SDBInstanceSkuManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
	SCloudregionResourceBaseManager
}

var DBInstanceSkuManager *SDBInstanceSkuManager

func init() {
	DBInstanceSkuManager = &SDBInstanceSkuManager{
		SEnabledStatusStandaloneResourceBaseManager: db.NewEnabledStatusStandaloneResourceBaseManager(
			SDBInstanceSku{},
			"dbinstance_skus_tbl",
			"dbinstance_sku",
			"dbinstance_skus",
		),
	}
	DBInstanceSkuManager.SetVirtualObject(DBInstanceSkuManager)
	DBInstanceSkuManager.NameRequireAscii = false
}

type SDBInstanceSku struct {
	db.SEnabledStatusStandaloneResourceBase
	db.SExternalizedResourceBase
	SCloudregionResourceBase
	Provider string `width:"32" charset:"ascii" nullable:"false" list:"user" create:"admin_required" update:"admin"`

	StorageType   string `width:"32" index:"true" list:"user" create:"optional"`
	DiskSizeStep  int    `list:"user" default:"1" create:"optional"` //步长
	MaxDiskSizeGb int    `list:"user" create:"optional"`
	MinDiskSizeGb int    `list:"user" create:"optional"`

	IOPS           int `list:"user" create:"optional"`
	TPS            int `list:"user" create:"optional"`
	QPS            int `list:"user" create:"optional"`
	MaxConnections int `list:"user" create:"optional"`

	VcpuCount  int `nullable:"false" default:"1" list:"user" create:"optional"`
	VmemSizeMb int `nullable:"false" list:"user" create:"required"`

	Category      string `width:"32" index:"true" nullable:"false" list:"user" create:"optional"`
	Engine        string `width:"16" index:"true" charset:"ascii" nullable:"false" list:"user" create:"required"`
	EngineVersion string `width:"16" index:"true" charset:"ascii" nullable:"false" list:"user" create:"required"`

	Zone1  string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"admin_optional" update:"admin"`
	Zone2  string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"admin_optional" update:"admin"`
	Zone3  string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"admin_optional" update:"admin"`
	ZoneId string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"admin_optional" update:"admin"`
}

func (self *SDBInstanceSkuManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (self *SDBInstanceSkuManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (self *SDBInstanceSku) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (self *SDBInstanceSku) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return false
}

func (manager *SDBInstanceSkuManager) fetchDBInstanceSkus(provider string, region *SCloudregion) ([]SDBInstanceSku, error) {
	skus := []SDBInstanceSku{}
	q := manager.Query().Equals("provider", provider).Equals("cloudregion_id", region.Id)
	err := db.FetchModelObjects(manager, q, &skus)
	if err != nil {
		return nil, err
	}
	return skus, nil
}

// RDS套餐类型列表
func (manager *SDBInstanceSkuManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.DBInstanceSkuListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SEnabledStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledStatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusStandaloneResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}

	if domainStr := query.ProjectDomainId; len(domainStr) > 0 {
		domain, err := db.TenantCacheManager.FetchDomainByIdOrName(context.Background(), domainStr)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("domains", domainStr)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		query.ProjectDomainId = domain.GetId()
	}

	q = listItemDomainFilter(q, query.Providers, query.ProjectDomainId)

	q, err = managedResourceFilterByRegion(q, query.RegionalFilterListInput, "", nil)
	if err != nil {
		return nil, errors.Wrap(err, "managedResourceFilterByRegion")
	}

	// StorageType
	if len(query.StorageType) > 0 {
		q = q.In("storage_type", query.StorageType)
	}
	if len(query.VcpuCount) > 0 {
		q = q.In("vcpu_count", query.VcpuCount)
	}
	if len(query.VmemSizeMb) > 0 {
		q = q.In("vmem_size_mb", query.VmemSizeMb)
	}
	if len(query.Category) > 0 {
		q = q.In("category", query.Category)
	}
	if len(query.Engine) > 0 {
		q = q.In("engine", query.Engine)
	}
	if len(query.EngineVersion) > 0 {
		q = q.In("engine_version", query.EngineVersion)
	}
	if len(query.Providers) > 0 {
		q = q.In("provider", query.Providers)
	}

	for k, zoneIds := range map[string][]string{"zone1": query.Zone1, "zone2": query.Zone2, "zone3": query.Zone3} {
		ids := []string{}
		for _, zoneId := range zoneIds {
			zone, err := ZoneManager.FetchByIdOrName(userCred, zoneId)
			if err != nil {
				if errors.Cause(err) == sql.ErrNoRows {
					return nil, httperrors.NewResourceNotFoundError2("zone", zoneId)
				}
				return nil, httperrors.NewGeneralError(err)
			}
			ids = append(ids, zone.GetId())
		}
		if len(ids) > 0 {
			q = q.In(k, ids)
		}
	}

	return q, nil
}

func (self *SDBInstanceSku) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (api.DBInstanceSkuDetails, error) {
	return api.DBInstanceSkuDetails{}, nil
}

func (manager *SDBInstanceSkuManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.DBInstanceSkuDetails {
	rows := make([]api.DBInstanceSkuDetails, len(objs))
	enableRows := manager.SEnabledStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regionRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	zoneIds := map[string]string{}
	for i := range rows {
		rows[i] = api.DBInstanceSkuDetails{
			EnabledStatusStandaloneResourceDetails: enableRows[i],
			CloudregionResourceInfo:                regionRows[i],
		}

		rows[i].CloudEnv = strings.Split(regionRows[i].RegionExternalId, "/")[0]

		sku := objs[i].(*SDBInstanceSku)
		for _, zoneId := range []string{sku.Zone1, sku.Zone2, sku.Zone3} {
			if _, ok := zoneIds[zoneId]; !ok {
				zoneIds[zoneId] = ""
			}
		}
	}
	var err error
	zoneIds, err = db.FetchIdNameMap(ZoneManager, zoneIds)
	if err != nil {
		log.Errorf("db.FetchIdNameMap fail %s", err)
		return rows
	}

	for i := range rows {
		sku := objs[i].(*SDBInstanceSku)
		rows[i].Zone1Name, _ = zoneIds[sku.Zone1]
		rows[i].Zone2Name, _ = zoneIds[sku.Zone2]
		rows[i].Zone3Name, _ = zoneIds[sku.Zone3]
	}

	return rows
}

func (manager *SDBInstanceSkuManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.DBInstanceSkuListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SEnabledStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.EnabledStatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusStandaloneResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SDBInstanceSkuManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SEnabledStatusStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (manager *SDBInstanceSkuManager) GetDBStringArray(q *sqlchemy.SQuery) ([]string, error) {
	array := []string{}
	rows, err := q.Rows()
	if err != nil {
		return nil, errors.Wrap(err, "q.Rows")
	}
	defer rows.Close()
	for rows.Next() {
		data := ""
		err = rows.Scan(&data)
		if err != nil {
			return nil, errors.Wrap(err, "rows.Scan")
		}
		array = append(array, data)
	}
	return array, err

}

func (manager *SDBInstanceSkuManager) AllowGetPropertyInstanceSpecs(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (manager *SDBInstanceSkuManager) GetPropertyInstanceSpecs(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	listQuery := api.DBInstanceSkuListInput{}
	err := query.Unmarshal(&listQuery)
	if err != nil {
		return nil, errors.Wrap(err, "query.Unmarshal")
	}
	q, err := manager.ListItemFilter(ctx, manager.Query(), userCred, listQuery)
	if err != nil {
		return nil, errors.Wrap(err, "manager.ListItemFilter")
	}

	q2, err := manager.ListItemFilter(ctx, manager.Query(), userCred, listQuery)
	if err != nil {
		return nil, errors.Wrap(err, "manager.ListItemFilter")
	}

	input := &SDBInstanceSku{}
	query.Unmarshal(input)
	for k, v := range map[string]interface{}{
		"provider":       input.Provider,
		"storage_type":   input.StorageType,
		"category":       input.Category,
		"engine":         input.Engine,
		"engine_version": input.EngineVersion,
		"iops":           input.IOPS,
		"qps":            input.QPS,
		"tps":            input.TPS,
		"vcpu_count":     input.VcpuCount,
		"vmem_size_mb":   input.VmemSizeMb,
	} {
		switch v.(type) {
		case string:
			value := v.(string)
			if len(value) > 0 {
				q = q.Equals(k, v)
				q2 = q2.Equals(k, v)
			}
		case int, int64:
			value := fmt.Sprintf("%d", v)
			if value != "0" {
				q = q.Equals(k, v)
				q2 = q2.Equals(k, v)
			}
		}
	}

	sq := q2.SubQuery()
	q2 = sq.Query(sq.Field("zone1"), sq.Field("zone2"), sq.Field("zone3")).Distinct()

	skuZones := []struct {
		Zone1 string
		Zone2 string
		Zone3 string
	}{}
	err = q2.All(&skuZones)
	if err != nil {
		return nil, errors.Wrapf(err, "query sku zones")
	}

	skus := []SDBInstanceSku{}
	q = q.GroupBy(q.Field("vcpu_count"), q.Field("vmem_size_mb"))
	q = q.Asc(q.Field("vcpu_count"), q.Field("vmem_size_mb"))
	err = db.FetchModelObjects(manager, q, &skus)
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrap(err, "db.FetchModelObjects"))
	}
	result := struct {
		CpuMemsMb map[string][]int
		cpuMemsMb map[int]map[int]bool
		Cpus      []int
		MemsMb    []int
		memsMb    map[int]bool
		Zones     struct {
			Zones map[string]string
			Zone1 map[string]string
			Zone2 map[string]string
			Zone3 map[string]string
		}
		zones     []string
		zoneNames map[string]string
	}{
		CpuMemsMb: map[string][]int{},
		cpuMemsMb: map[int]map[int]bool{},
		Cpus:      []int{},
		MemsMb:    []int{},
		memsMb:    map[int]bool{},
		zones:     []string{},
		zoneNames: map[string]string{},
	}
	result.Zones.Zones = map[string]string{}
	result.Zones.Zone1 = map[string]string{}
	result.Zones.Zone2 = map[string]string{}
	result.Zones.Zone3 = map[string]string{}
	for _, sku := range skuZones {
		if _, ok := result.Zones.Zone1[sku.Zone1]; !ok && len(sku.Zone1) > 0 {
			result.Zones.Zone1[sku.Zone1] = sku.Zone1
		}

		if _, ok := result.Zones.Zone2[sku.Zone2]; !ok && len(sku.Zone2) > 0 {
			result.Zones.Zone2[sku.Zone2] = sku.Zone2
		}

		if _, ok := result.Zones.Zone3[sku.Zone3]; !ok && len(sku.Zone3) > 0 {
			result.Zones.Zone3[sku.Zone3] = sku.Zone3
		}

		zoneIds := []string{}
		for _, zone := range []string{sku.Zone1, sku.Zone2, sku.Zone3} {
			if len(zone) > 0 {
				zoneIds = append(zoneIds, zone)
				if !utils.IsInStringArray(zone, result.zones) {
					result.zones = append(result.zones, zone)
				}
			}
		}
		zoneId := strings.Join(zoneIds, "+")
		if _, ok := result.Zones.Zones[zoneId]; !ok && len(zoneId) > 0 {
			result.Zones.Zones[zoneId] = zoneId
		}
	}
	for _, sku := range skus {
		if _, ok := result.cpuMemsMb[sku.VcpuCount]; !ok {
			result.cpuMemsMb[sku.VcpuCount] = map[int]bool{}
			result.CpuMemsMb[fmt.Sprintf("%d", sku.VcpuCount)] = []int{}
			result.Cpus = append(result.Cpus, sku.VcpuCount)
		}

		if _, ok := result.cpuMemsMb[sku.VcpuCount][sku.VmemSizeMb]; !ok {
			result.cpuMemsMb[sku.VcpuCount][sku.VmemSizeMb] = true
			result.CpuMemsMb[fmt.Sprintf("%d", sku.VcpuCount)] = append(result.CpuMemsMb[fmt.Sprintf("%d", sku.VcpuCount)], sku.VmemSizeMb)
		}

		if _, ok := result.memsMb[sku.VmemSizeMb]; !ok {
			result.memsMb[sku.VmemSizeMb] = true
			result.MemsMb = append(result.MemsMb, sku.VmemSizeMb)
		}

	}

	zones := []struct {
		Id   string
		Name string
	}{}

	err = ZoneManager.Query("id", "name").In("id", result.zones).All(&zones)
	if err != nil {
		return nil, errors.Wrapf(err, "query zones")
	}
	for _, zone := range zones {
		result.zoneNames[zone.Id] = zone.Name
	}

	for _, zoneId := range result.Zones.Zone1 {
		result.Zones.Zone1[zoneId] = result.zoneNames[zoneId]
	}

	for _, zoneId := range result.Zones.Zone2 {
		result.Zones.Zone2[zoneId] = result.zoneNames[zoneId]
	}

	for _, zoneId := range result.Zones.Zone3 {
		result.Zones.Zone3[zoneId] = result.zoneNames[zoneId]
	}

	for _, zoneId := range result.Zones.Zones {
		zoneIds := strings.Split(zoneId, "+")
		zoneNames := []string{}
		for _, id := range zoneIds {
			zoneNames = append(zoneNames, result.zoneNames[id])
		}
		result.Zones.Zones[zoneId] = strings.Join(zoneNames, "+")
	}

	return jsonutils.Marshal(result), nil
}

func (manager *SDBInstanceSkuManager) GetEngines(provider, cloudregionId string) ([]string, error) {
	q := manager.Query("engine").Equals("provider", provider).Equals("cloudregion_id", cloudregionId).Distinct()
	return manager.GetDBStringArray(q)
}

func (manager *SDBInstanceSkuManager) GetEngineVersions(provider, cloudregionId, engine string) ([]string, error) {
	q := manager.Query("engine_version").Equals("provider", provider).Equals("cloudregion_id", cloudregionId).Equals("engine", engine).Distinct()
	return manager.GetDBStringArray(q)
}

func (manager *SDBInstanceSkuManager) GetCategories(provider, cloudregionId, engine, version string) ([]string, error) {
	q := manager.Query("category").Equals("provider", provider).Equals("cloudregion_id", cloudregionId).Equals("engine", engine).Equals("engine_version", version).Distinct()
	return manager.GetDBStringArray(q)
}

func (manager *SDBInstanceSkuManager) GetStorageTypes(provider, cloudregionId, engine, version, category string) ([]string, error) {
	q := manager.Query("storage_type").Equals("provider", provider).Equals("cloudregion_id", cloudregionId).Equals("engine", engine).Equals("engine_version", version).Distinct()
	q = q.Equals("category", category)
	return manager.GetDBStringArray(q)
}

func (manager *SDBInstanceSkuManager) GetInstanceTypes(provider, cloudregionId, engine, version, category, storageType string) ([]string, error) {
	q := manager.Query("name").Equals("provider", provider).Equals("cloudregion_id", cloudregionId).Equals("engine", engine).Equals("engine_version", version).Distinct()
	q = q.Equals("category", category).Equals("storage_type", storageType)
	return manager.GetDBStringArray(q)
}

func (manager *SDBInstanceSkuManager) GetDBInstanceSkus(provider, cloudregionId, engine, version, category, storageType string) ([]SDBInstanceSku, error) {
	skus := []SDBInstanceSku{}
	q := manager.Query("name").Equals("provider", provider).Equals("cloudregion_id", cloudregionId).Equals("engine", engine).Equals("engine_version", version).Distinct()
	q = q.Equals("category", category).Equals("storage_type", storageType)
	err := db.FetchModelObjects(manager, q, &skus)
	if err != nil {
		return nil, err
	}
	return skus, nil
}

func (manager *SDBInstanceSkuManager) SyncDBInstanceSkus(ctx context.Context, userCred mcclient.TokenCredential, region *SCloudregion, meta *SSkuResourcesMeta) compare.SyncResult {
	lockman.LockRawObject(ctx, "dbinstance-skus", region.Id)
	defer lockman.ReleaseRawObject(ctx, "dbinstance-skus", region.Id)

	syncResult := compare.SyncResult{}

	iskus, err := meta.GetDBInstanceSkusByRegionExternalId(region.ExternalId)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	dbSkus, err := manager.fetchDBInstanceSkus(region.Provider, region)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	removed := make([]SDBInstanceSku, 0)
	commondb := make([]SDBInstanceSku, 0)
	commonext := make([]SDBInstanceSku, 0)
	added := make([]SDBInstanceSku, 0)

	err = compare.CompareSets(dbSkus, iskus, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].Delete(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}
	for i := 0; i < len(commondb); i += 1 {
		err = commondb[i].syncWithCloudSku(ctx, userCred, commonext[i])
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i += 1 {
		err = manager.newFromCloudSku(ctx, userCred, added[i], region)
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncResult.Add()
		}
	}
	return syncResult
}

func (sku SDBInstanceSku) GetGlobalId() string {
	return sku.ExternalId
}

func (sku *SDBInstanceSku) syncWithCloudSku(ctx context.Context, userCred mcclient.TokenCredential, isku SDBInstanceSku) error {
	_, err := db.Update(sku, func() error {
		sku.Status = isku.Status
		sku.TPS = isku.TPS
		sku.QPS = isku.QPS
		sku.MaxConnections = isku.MaxConnections
		return nil
	})
	return err
}

func (manager *SDBInstanceSkuManager) newFromCloudSku(ctx context.Context, userCred mcclient.TokenCredential, isku SDBInstanceSku, region *SCloudregion) error {
	sku := &isku
	sku.SetModelManager(manager, sku)
	sku.Id = "" //避免使用yunion meta的id,导致出现duplicate entry问题
	sku.CloudregionId = region.Id
	return manager.TableSpec().Insert(ctx, sku)
}

func SyncRegionDBInstanceSkus(ctx context.Context, userCred mcclient.TokenCredential, regionId string, isStart bool) {
	if isStart {
		q := DBInstanceSkuManager.Query()
		if len(regionId) > 0 {
			q = q.Equals("cloudregion_id", regionId)
		}
		cnt, err := q.Limit(1).CountWithError()
		if err != nil && err != sql.ErrNoRows {
			log.Errorf("SyncDBInstanceSkus.QueryDBInstanceSku %s", err)
			return
		}
		if cnt > 0 {
			log.Debugf("SyncDBInstanceSkus synced skus, skip...")
			return
		}
	}

	q := CloudregionManager.Query()
	q = q.In("provider", CloudproviderManager.GetPublicProviderProvidersQuery())
	if len(regionId) > 0 {
		q = q.Equals("id", regionId)
	}
	cloudregions := []SCloudregion{}
	err := db.FetchModelObjects(CloudregionManager, q, &cloudregions)
	if err != nil {
		log.Errorf("failed to fetch cloudregions: %v", err)
		return
	}

	meta, err := FetchSkuResourcesMeta()
	if err != nil {
		log.Errorf("failed to fetch sku resource meta: %v", err)
		return
	}

	for _, region := range cloudregions {
		if !region.GetDriver().IsSupportedDBInstance() {
			log.Infof("region %s(%s) not support dbinstance, skip sync", region.Name, region.Id)
			continue
		}
		result := DBInstanceSkuManager.SyncDBInstanceSkus(ctx, userCred, &region, meta)
		msg := result.Result()
		notes := fmt.Sprintf("SyncDBInstanceSkus for region %s result: %s", region.Name, msg)
		log.Infof(notes)
	}

}

func SyncDBInstanceSkus(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	SyncRegionDBInstanceSkus(ctx, userCred, "", isStart)
}

func (manager *SDBInstanceSkuManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SEnabledStatusStandaloneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusStandaloneResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SCloudregionResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SCloudregionResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}

func (self *SDBInstanceSku) GetZoneInfo() (cloudprovider.SZoneInfo, error) {
	zoneInfo := cloudprovider.SZoneInfo{ZoneId: self.ZoneId}
	region := self.GetRegion()
	if region == nil {
		return zoneInfo, fmt.Errorf("empyt region for rds sku %s(%s)", self.Name, self.Id)
	}
	var cloudZoneId = func(id string) (string, error) {
		if len(id) == 0 {
			return "", nil
		}
		_zone, err := ZoneManager.FetchById(id)
		if err != nil {
			log.Errorf("ZoneManager.FetchById(%s) error: %v", id, err)
			return "", errors.Wrapf(err, "ZoneManager.FetchById(%s)", id)
		}
		zone := _zone.(*SZone)
		return strings.TrimPrefix(zone.ExternalId, region.ExternalId+"/"), nil
	}
	zoneInfo.Zone1, _ = cloudZoneId(self.Zone1)
	zoneInfo.Zone2, _ = cloudZoneId(self.Zone2)
	zoneInfo.Zone3, _ = cloudZoneId(self.Zone3)
	return zoneInfo, nil
}

func (manager *SDBInstanceSkuManager) AllowSyncSkus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, manager, "sync-skus")
}

func (manager *SDBInstanceSkuManager) PerformSyncSkus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.SkuSyncInput) (jsonutils.JSONObject, error) {
	return PerformActionSyncSkus(ctx, userCred, manager.Keyword(), input)
}

func (manager *SDBInstanceSkuManager) AllowGetPropertySyncTasks(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGetSpec(userCred, manager, "sync-tasks")
}

func (manager *SDBInstanceSkuManager) GetPropertySyncTasks(ctx context.Context, userCred mcclient.TokenCredential, query api.SkuTaskQueryInput) (jsonutils.JSONObject, error) {
	return GetPropertySkusSyncTasks(ctx, userCred, query)
}
