package models

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"
)

type SDBInstanceSkuManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
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

	DBInstanceSkuManager.NameRequireAscii = true
	db.RegisterModelManager(DBInstanceSkuManager)
	DBInstanceSkuManager.SetVirtualObject(DBInstanceSkuManager)
	DBInstanceSkuManager.NameRequireAscii = false
}

type SDBInstanceSku struct {
	db.SEnabledStatusStandaloneResourceBase
	db.SExternalizedResourceBase
	SCloudregionResourceBase
	Provider string `width:"32" charset:"ascii" nullable:"false" list:"user" create:"admin_required" update:"admin"`

	StorageType   string `list:"user" create:"optional"`
	DiskSizeStep  int    `list:"user" default:"1" create:"optional"` //步长
	MaxDiskSizeGb int    `list:"user" create:"optional"`
	MinDiskSizeGb int    `list:"user" create:"optional"`

	IOPS int `list:"user" create:"optional"`

	VcpuCount  int `nullable:"false" default:"1" list:"user" create:"optional"`
	VmemSizeMb int `nullable:"false" list:"user" create:"required"`

	Category      string `nullable:"false" list:"user" create:"optional"`
	Engine        string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"required"`
	EngineVersion string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"required"`

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

func (manager *SDBInstanceSkuManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := manager.SEnabledStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	data := query.(*jsonutils.JSONDict)
	return validators.ApplyModelFilters(q, data, []*validators.ModelFilterOptions{
		{Key: "cloudregion", ModelKeyword: "cloudregion", OwnerId: userCred},
	})
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
	q, err := manager.ListItemFilter(ctx, manager.Query(), userCred, query)
	if err != nil {
		return nil, err
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
		"vcpu_count":     input.VcpuCount,
		"vmem_size_mb":   input.VmemSizeMb,
	} {
		switch v.(type) {
		case string:
			value := v.(string)
			if len(value) > 0 {
				q = q.Equals(k, v)
			}
		case int, int64:
			value := fmt.Sprintf("%d", v)
			if value != "0" {
				q = q.Equals(k, v)
			}
		}
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

	zq := ZoneManager.Query("id", "name").In("id", result.zones)
	rows, err := zq.Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		id, name := "", ""
		err = rows.Scan(&id, &name)
		if err != nil {
			return nil, errors.Wrap(err, "rows.Scan")
		}
		result.zoneNames[id] = name
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

func (manager *SDBInstanceSkuManager) syncDBInstanceSkus(ctx context.Context, userCred mcclient.TokenCredential, region *SCloudregion) compare.SyncResult {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	syncResult := compare.SyncResult{}

	regionInfo := strings.Split(region.ExternalId, "/")
	if len(regionInfo) == 0 {
		syncResult.Error(fmt.Errorf("region %s is not belong public cloud???", region.Name))
		return syncResult
	}
	regionId := regionInfo[len(regionInfo)-1]

	s := auth.GetAdminSession(ctx, options.Options.Region, "")
	uri, err := s.GetServiceURL("yunionmeta", "")
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}
	iskus := []SDBInstanceSku{}
	params := url.Values{}
	params.Add("region_id", regionId)
	params.Add("provider", region.Provider)
	params.Set("limit", "2048")
	for {
		params.Set("offset", fmt.Sprintf("%d", len(iskus)))
		_, resp, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, httputils.THttpMethod("GET"), fmt.Sprintf("%s/dbinstance_skus?%s", uri, params.Encode()), nil, nil, false)
		if err != nil {
			syncResult.Error(errors.Wrap(err, "request yunionmeta dbinstance_skus"))
			return syncResult
		}

		parts := []SDBInstanceSku{}
		err = resp.Unmarshal(&parts, "dbinstance_skus")
		if err != nil {
			syncResult.Error(errors.Wrapf(err, "skus.Unmarshal"))
			return syncResult
		}
		iskus = append(iskus, parts...)

		total, _ := resp.Int("total")
		if len(iskus) >= int(total) {
			break
		}
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
		return nil
	})
	return err
}

func (manager *SDBInstanceSkuManager) getZoneBySuffix(region *SCloudregion, suffix string) (*SZone, error) {
	q := ZoneManager.Query().Equals("cloudregion_id", region.Id).Endswith("external_id", suffix)
	count, err := q.CountWithError()
	if err != nil {
		return nil, err
	}
	if count > 1 {
		return nil, fmt.Errorf("duplicate zone with suffix %s in region %s", suffix, region.Name)
	}
	if count == 0 {
		return nil, fmt.Errorf("failed to found zone with suffix %s in region %s", suffix, region.Name)
	}
	zone := &SZone{}
	return zone, q.First(zone)
}

func (manager *SDBInstanceSkuManager) newFromCloudSku(ctx context.Context, userCred mcclient.TokenCredential, isku SDBInstanceSku, region *SCloudregion) error {
	sku := &isku
	sku.SetModelManager(manager, sku)
	sku.Id = "" //避免使用yunion meta的id,导致出现duplicate entry问题
	sku.CloudregionId = region.Id

	if len(isku.Zone1) > 0 {
		zone, err := manager.getZoneBySuffix(region, isku.Zone1)
		if err != nil {
			return errors.Wrapf(err, "failed to get zone1 info by %s", isku.Zone1)
		}
		sku.Zone1 = zone.Id
	}

	if len(isku.Zone2) > 0 {
		zone, err := manager.getZoneBySuffix(region, isku.Zone2)
		if err != nil {
			return errors.Wrapf(err, "failed to get zone1 info by %s", isku.Zone2)
		}
		sku.Zone2 = zone.Id
	}

	if len(isku.Zone3) > 0 {
		zone, err := manager.getZoneBySuffix(region, isku.Zone3)
		if err != nil {
			return errors.Wrapf(err, "failed to get zone1 info by %s", isku.Zone3)
		}
		sku.Zone3 = zone.Id
	}

	return manager.TableSpec().Insert(sku)
}

func SyncDBInstanceSkus(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	q := CloudregionManager.Query()
	q = q.In("provider", CloudproviderManager.GetPublicProviderProvidersQuery())
	cloudregions := []SCloudregion{}
	err := db.FetchModelObjects(CloudregionManager, q, &cloudregions)
	if err != nil {
		log.Errorf("failed to fetch cloudregions: %v", err)
		return
	}

	for _, region := range cloudregions {
		result := DBInstanceSkuManager.syncDBInstanceSkus(ctx, userCred, &region)
		msg := result.Result()
		notes := fmt.Sprintf("SyncDBInstanceSkus for region %s result: %s", region.Name, msg)
		log.Infof(notes)
	}
}
