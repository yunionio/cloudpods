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
	"sync"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/onecloud/pkg/util/yunionmeta"
)

// +onecloud:swagger-gen-model-singular=dbinstance_sku
// +onecloud:swagger-gen-model-plural=dbinstance_skus
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
	DBInstanceSkuManager.TableSpec().AddIndex(false, "cloudregion_id", "deleted", "provider")
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
	EngineVersion string `width:"64" index:"true" charset:"ascii" nullable:"false" list:"user" create:"required"`

	Zone1   string            `width:"128" charset:"ascii" nullable:"false" list:"user" create:"admin_optional" update:"admin"`
	Zone2   string            `width:"128" charset:"ascii" nullable:"false" list:"user" create:"admin_optional" update:"admin"`
	Zone3   string            `width:"128" charset:"ascii" nullable:"false" list:"user" create:"admin_optional" update:"admin"`
	ZoneId  string            `width:"128" charset:"ascii" nullable:"false" list:"user" create:"admin_optional" update:"admin"`
	MultiAZ tristate.TriState `default:"false" list:"user" create:"optional"`
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
		domain, err := db.TenantCacheManager.FetchDomainByIdOrName(ctx, domainStr)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("domains", domainStr)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		query.ProjectDomainId = domain.GetId()
	}

	q = listItemDomainFilter(q, query.Providers, query.ProjectDomainId)

	q, err = managedResourceFilterByRegion(ctx, q, query.RegionalFilterListInput, "", nil)
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
			zone, err := ZoneManager.FetchByIdOrName(ctx, userCred, zoneId)
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

func (manager *SDBInstanceSkuManager) GetSkuCountByRegion(regionId string) (int, error) {
	q := manager.Query().Equals("cloudregion_id", regionId)
	return q.CountWithError()
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

func (manager *SDBInstanceSkuManager) SyncDBInstanceSkus(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	region *SCloudregion,
	xor bool,
) compare.SyncResult {
	lockman.LockRawObject(ctx, manager.Keyword(), region.Id)
	defer lockman.ReleaseRawObject(ctx, manager.Keyword(), region.Id)

	result := compare.SyncResult{}

	meta, err := yunionmeta.FetchYunionmeta(ctx)
	if err != nil {
		result.Error(errors.Wrapf(err, "FetchYunionmeta"))
		return result
	}

	iskus := []SDBInstanceSku{}
	err = meta.List(manager.Keyword(), region.ExternalId, &iskus)
	if err != nil {
		result.Error(err)
		return result
	}

	dbSkus, err := manager.fetchDBInstanceSkus(region.Provider, region)
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SDBInstanceSku, 0)
	commondb := make([]SDBInstanceSku, 0)
	commonext := make([]SDBInstanceSku, 0)
	added := make([]SDBInstanceSku, 0)

	err = compare.CompareSets(dbSkus, iskus, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(err)
		return result
	}

	for i := 0; i < len(removed); i += 1 {
		err = db.RealDeleteModel(ctx, userCred, &removed[i])
		if err != nil {
			result.DeleteError(err)
		} else {
			result.Delete()
		}
	}
	if !xor {
		for i := 0; i < len(commondb); i += 1 {
			err = commondb[i].syncWithCloudSku(ctx, userCred, commonext[i])
			if err != nil {
				result.UpdateError(err)
			} else {
				result.Update()
			}
		}
	}
	for i := 0; i < len(added); i += 1 {
		err = region.newDBInstanceSkuFromCloudSku(ctx, userCred, added[i].GetGlobalId())
		if err != nil {
			result.AddError(err)
		} else {
			result.Add()
		}
	}
	return result
}

func (sku SDBInstanceSku) GetGlobalId() string {
	return sku.ExternalId
}

func (sku *SDBInstanceSku) syncWithCloudSku(ctx context.Context, userCred mcclient.TokenCredential, isku SDBInstanceSku) error {
	_, err := db.Update(sku, func() error {
		sku.Status = isku.Status
		return nil
	})
	return err
}

func (self *SCloudregion) newDBInstanceSkuFromCloudSku(ctx context.Context, userCred mcclient.TokenCredential, externalId string) error {
	meta, err := yunionmeta.FetchYunionmeta(ctx)
	if err != nil {
		return err
	}
	zones, err := self.GetZones()
	if err != nil {
		return errors.Wrap(err, "GetZones")
	}
	zoneMaps := map[string]string{}
	for _, zone := range zones {
		zoneMaps[zone.ExternalId] = zone.Id
	}

	sku := &SDBInstanceSku{}
	sku.SetModelManager(DBInstanceSkuManager, sku)

	skuUrl := self.getMetaUrl(meta.DBInstanceBase, externalId)
	err = meta.Get(skuUrl, sku)
	if err != nil {
		return errors.Wrapf(err, "Get")
	}

	if len(sku.Zone1) > 0 {
		zoneId := yunionmeta.GetZoneIdBySuffix(zoneMaps, sku.Zone1)
		if len(zoneId) == 0 {
			return errors.Wrapf(err, "GetZoneIdBySuffix(%s)", sku.Zone1)
		}
		sku.Zone1 = zoneId
	}

	if len(sku.Zone2) > 0 {
		zoneId := yunionmeta.GetZoneIdBySuffix(zoneMaps, sku.Zone2)
		if len(zoneId) == 0 {
			return errors.Wrapf(err, "GetZoneIdBySuffix(%s)", sku.Zone2)
		}
		sku.Zone2 = zoneId
	}

	if len(sku.Zone3) > 0 {
		zoneId := yunionmeta.GetZoneIdBySuffix(zoneMaps, sku.Zone3)
		if len(zoneId) == 0 {
			return errors.Wrapf(err, "GetZoneIdBySuffix(%s)", sku.Zone3)
		}
		sku.Zone3 = zoneId
	}

	sku.CloudregionId = self.Id
	sku.Provider = self.Provider
	sku.Enabled = tristate.True
	return DBInstanceSkuManager.TableSpec().Insert(ctx, sku)
}

func SyncRegionDBInstanceSkus(ctx context.Context, userCred mcclient.TokenCredential, regionId string, isStart, xor bool) {
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
			log.Debugf("sync rds sku for %s synced skus, skip...", regionId)
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
	if len(cloudregions) == 0 {
		return
	}

	meta, err := yunionmeta.FetchYunionmeta(ctx)
	if err != nil {
		log.Errorf("FetchYunionmeta: %v", err)
		return
	}

	index, err := meta.Index(DBInstanceSkuManager.Keyword())
	if err != nil {
		log.Errorf("get rds sku index error: %v", err)
		return
	}

	for _, region := range cloudregions {
		if !region.GetDriver().IsSupportedDBInstance() {
			log.Debugf("region %s(%s) not support dbinstance, skip sync", region.Name, region.Id)
			continue
		}
		skuMeta := &SDBInstanceSku{}
		skuMeta.SetModelManager(DBInstanceSkuManager, skuMeta)
		skuMeta.Id = region.ExternalId

		oldMd5 := db.Metadata.GetStringValue(ctx, skuMeta, db.SKU_METADAT_KEY, userCred)
		newMd5, ok := index[region.ExternalId]
		if !ok || newMd5 == yunionmeta.EMPTY_MD5 || len(oldMd5) > 0 && newMd5 == oldMd5 {
			continue
		}

		db.Metadata.SetValue(ctx, skuMeta, db.SKU_METADAT_KEY, newMd5, userCred)

		result := DBInstanceSkuManager.SyncDBInstanceSkus(ctx, userCred, &region, xor)
		msg := result.Result()
		notes := fmt.Sprintf("sync rds sku for region %s result: %s", region.Name, msg)
		log.Debugf(notes)
	}

}

func SyncDBInstanceSkus(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	SyncRegionDBInstanceSkus(ctx, userCred, "", isStart, false)
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
	region, err := self.GetRegion()
	if err != nil {
		return zoneInfo, nil
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

func (manager *SDBInstanceSkuManager) PerformSyncSkus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.SkuSyncInput) (jsonutils.JSONObject, error) {
	return PerformActionSyncSkus(ctx, userCred, manager.Keyword(), input)
}

func (manager *SDBInstanceSkuManager) GetPropertySyncTasks(ctx context.Context, userCred mcclient.TokenCredential, query api.SkuTaskQueryInput) (jsonutils.JSONObject, error) {
	return GetPropertySkusSyncTasks(ctx, userCred, query)
}

func (self *SCloudregion) GetDBInstanceSkus() ([]SDBInstanceSku, error) {
	q := DBInstanceSkuManager.Query().Equals("cloudregion_id", self.Id)
	skus := []SDBInstanceSku{}
	err := db.FetchModelObjects(DBInstanceSkuManager, q, &skus)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return skus, nil
}

func (self *SCloudregion) SyncDBInstanceSkus(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	provider *SCloudprovider,
	exts []cloudprovider.ICloudDBInstanceSku,
) compare.SyncResult {
	lockman.LockRawObject(ctx, DBInstanceSkuManager.Keyword(), self.Id)
	defer lockman.ReleaseRawObject(ctx, DBInstanceSkuManager.Keyword(), self.Id)

	result := compare.SyncResult{}
	dbSkus, err := self.GetDBInstanceSkus()
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SDBInstanceSku, 0)
	commondb := make([]SDBInstanceSku, 0)
	commonext := make([]cloudprovider.ICloudDBInstanceSku, 0)
	added := make([]cloudprovider.ICloudDBInstanceSku, 0)

	err = compare.CompareSets(dbSkus, exts, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(err)
		return result
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].Delete(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}
	ch := make(chan struct{}, options.Options.SkuBatchSync)
	defer close(ch)
	var wg sync.WaitGroup
	for i := 0; i < len(added); i += 1 {
		ch <- struct{}{}
		wg.Add(1)
		go func(sku cloudprovider.ICloudDBInstanceSku) {
			defer func() {
				wg.Done()
				<-ch
			}()
			err = self.newFromCloudSku(ctx, userCred, sku)
			if err != nil {
				result.AddError(err)
				return
			}
			result.Add()
		}(added[i])
	}
	wg.Wait()
	return result
}

func (self *SCloudregion) newFromCloudSku(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudDBInstanceSku) error {
	sku := &SDBInstanceSku{}
	sku.SetModelManager(DBInstanceSkuManager, sku)
	sku.Name = ext.GetName()
	sku.Status = ext.GetStatus()
	sku.CloudregionId = self.Id
	sku.Enabled = tristate.True
	sku.ExternalId = ext.GetGlobalId()
	sku.Engine = ext.GetEngine()
	sku.EngineVersion = ext.GetEngineVersion()
	sku.StorageType = ext.GetStorageType()
	sku.DiskSizeStep = ext.GetDiskSizeStep()
	sku.MaxDiskSizeGb = ext.GetMaxDiskSizeGb()
	sku.MinDiskSizeGb = ext.GetMinDiskSizeGb()
	sku.IOPS = ext.GetIOPS()
	sku.TPS = ext.GetTPS()
	sku.QPS = ext.GetQPS()
	sku.MaxConnections = ext.GetMaxConnections()
	sku.VcpuCount = ext.GetVcpuCount()
	sku.VmemSizeMb = ext.GetVmemSizeMb()
	sku.Category = ext.GetCategory()
	sku.ZoneId = ext.GetZoneId()
	sku.Provider = self.Provider
	zones, _ := self.GetZones()
	if zone1 := ext.GetZone1Id(); len(zone1) > 0 {
		for _, zone := range zones {
			if strings.HasSuffix(zone.ExternalId, zone1) {
				sku.Zone1 = zone.Id
				break
			}
		}
	}

	if zone2 := ext.GetZone2Id(); len(zone2) > 0 {
		for _, zone := range zones {
			if strings.HasSuffix(zone.ExternalId, zone2) {
				sku.Zone2 = zone.Id
				break
			}
		}
	}

	if zone3 := ext.GetZone3Id(); len(zone3) > 0 {
		for _, zone := range zones {
			if strings.HasSuffix(zone.ExternalId, zone3) {
				sku.Zone3 = zone.Id
				break
			}
		}
	}

	return DBInstanceSkuManager.TableSpec().Insert(ctx, sku)
}
