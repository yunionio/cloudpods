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
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/onecloud/pkg/util/yunionmeta"
)

// +onecloud:swagger-gen-model-singular=elasticcachesku
// +onecloud:swagger-gen-model-plural=elasticcacheskus
type SElasticcacheSkuManager struct {
	db.SStatusStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
	SCloudregionResourceBaseManager
	SZoneResourceBaseManager
}

var ElasticcacheSkuManager *SElasticcacheSkuManager

func init() {
	ElasticcacheSkuManager = &SElasticcacheSkuManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SElasticcacheSku{},
			"elasticcacheskus_tbl",
			"elasticcachesku",
			"elasticcacheskus",
		),
	}
	ElasticcacheSkuManager.NameRequireAscii = false
	ElasticcacheSkuManager.SetVirtualObject(ElasticcacheSkuManager)
}

type SElasticcacheSku struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase

	SCloudregionResourceBase // 区域
	SZoneResourceBase        // 主可用区

	SlaveZoneId string `width:"64" charset:"ascii" nullable:"false" list:"user" create:"admin_optional" update:"admin"` // 备可用区

	InstanceSpec  string `width:"96" charset:"ascii" nullable:"false" list:"user" create:"admin_optional" update:"admin"`
	EngineArch    string `width:"32" charset:"utf8" nullable:"false" list:"user" create:"admin_optional" update:"admin"`
	LocalCategory string `width:"32" charset:"utf8" nullable:"false" list:"user" create:"admin_optional" update:"admin" default:""`

	PrepaidStatus  string `width:"32" charset:"utf8" nullable:"false" list:"user" create:"admin_optional" update:"admin" default:"available"`
	PostpaidStatus string `width:"32" charset:"utf8" nullable:"false" list:"user" create:"admin_optional" update:"admin" default:"available"`

	// 引擎	redis|memcached
	Engine string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"admin_optional" update:"admin"`
	// 引擎版本	3.0
	EngineVersion string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"admin_optional" update:"admin"`
	// CPU 架构 x86|ARM
	CpuArch string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"admin_optional" update:"admin"`
	// 存储类型	DRAM|SCM
	StorageType string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"admin_optional" update:"admin"`
	// standrad|enhanced
	PerformanceType string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"admin_optional" update:"admin"`
	// single（单副本） | double（双副本) | readone (单可读) | readthree （3可读） | readfive（5只读）
	NodeType string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"admin_optional" update:"admin"`

	// 内存容量
	MemorySizeMB int `nullable:"false" list:"user" create:"admin_required" update:"admin"`
	// 套餐附带硬盘容量
	DiskSizeGB int `nullable:"false" list:"user" create:"admin_required" update:"admin"`
	// 最小分片数量
	ShardNum int `nullable:"false" list:"user" create:"admin_optional" update:"admin"`
	// 最大分片数量
	MaxShardNum int `nullable:"false" list:"user" create:"admin_optional" update:"admin"`
	// 最小副本数量
	ReplicasNum int `nullable:"false" list:"user" create:"admin_optional" update:"admin"`
	// 最大副本数量
	MaxReplicasNum int `nullable:"false" list:"user" create:"admin_optional" update:"admin"`

	// 最大客户端数
	MaxClients int `nullable:"false" list:"user" create:"admin_optional" update:"admin"`
	// 最大连接数
	MaxConnections int `nullable:"false" list:"user" create:"admin_optional" update:"admin"`
	// 最大内网带宽
	MaxInBandwidthMb int `nullable:"false" list:"user" create:"admin_optional" update:"admin"`
	// 实际可使用的最大内存
	MaxMemoryMB int `nullable:"false" list:"user" create:"admin_optional" update:"admin"`
	// QPS参考值
	QPS int `nullable:"false" list:"user" create:"admin_optional" update:"admin"`

	// 公有云厂商	Aliyun/Azure/AWS/Qcloud/...
	Provider string `width:"32" charset:"ascii" nullable:"false" list:"user" create:"admin_required" update:"admin"`
}

func (self SElasticcacheSku) GetGlobalId() string {
	return self.ExternalId
}

func (manager *SElasticcacheSkuManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ElasticcacheSkuDetails {
	rows := make([]api.ElasticcacheSkuDetails, len(objs))

	stdRows := manager.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	zoneRows := manager.SZoneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	slavezoneRows := manager.FetchSlaveZoneResourceInfos(ctx, userCred, query, objs)

	for i := range rows {
		rows[i] = api.ElasticcacheSkuDetails{
			StatusStandaloneResourceDetails: stdRows[i],
			CloudregionResourceInfo:         regRows[i],
			ZoneResourceInfoBase:            zoneRows[i].ZoneResourceInfoBase,
			SlaveZoneResourceInfoBase:       slavezoneRows[i],
		}

		rows[i].CloudEnv = strings.Split(regRows[i].RegionExternalId, "/")[0]
	}

	return rows
}

func (self *SElasticcacheSkuManager) FetchSlaveZoneResourceInfos(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{}) []api.SlaveZoneResourceInfoBase {
	rows := make([]api.SlaveZoneResourceInfoBase, len(objs))
	zoneIds := []string{}
	for i := range objs {
		slavezone := objs[i].(*SElasticcacheSku).SlaveZoneId
		if len(slavezone) > 0 {
			zoneIds = append(zoneIds, slavezone)
		}
	}

	zones := make(map[string]SZone)
	err := db.FetchStandaloneObjectsByIds(ZoneManager, zoneIds, &zones)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail %s", err)
		return rows
	}

	for i := range objs {
		if zone, ok := zones[objs[i].(*SElasticcacheSku).SlaveZoneId]; ok {
			rows[i].SlaveZone = zone.GetName()
			rows[i].SlaveZoneExtId = fetchExternalId(zone.GetExternalId())
		}
	}

	return rows
}

func (manager *SElasticcacheSkuManager) GetSkuCountByRegion(regionId string) (int, error) {
	q := manager.Query().Equals("cloudregion_id", regionId)

	return q.CountWithError()
}

// 弹性缓存套餐规格列表
func (manager *SElasticcacheSkuManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ElasticcacheSkuListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SCloudregionResourceBaseManager.ListItemFilter(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemFilter")
	}

	zoneQuery := api.ZonalFilterListInput{
		ZonalFilterListBase: query.ZonalFilterListBase,
	}
	q, err = manager.SZoneResourceBaseManager.ListItemFilter(ctx, q, userCred, zoneQuery)
	if err != nil {
		return nil, errors.Wrap(err, "SZoneResourceBaseManager.ListItemFilter")
	}

	if query.Usable != nil && *query.Usable {
		q, err = usableFilter(q, true)
		if err != nil {
			return nil, err
		}
		sq := sqlchemy.OR(sqlchemy.Equals(q.Field("prepaid_status"), api.SkuStatusAvailable), sqlchemy.Equals(q.Field("postpaid_status"), api.SkuStatusAvailable))
		q = q.Filter(sq)
	}

	if b := query.BillingType; len(b) > 0 {
		switch b {
		case billing.BILLING_TYPE_POSTPAID:
			q = q.Equals("postpaid_status", api.SkuStatusAvailable)
		case billing.BILLING_TYPE_PREPAID:
			q = q.Equals("prepaid_status", api.SkuStatusAvailable)
		}
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

	// 按区间查询内存, 避免0.75G这样的套餐不好过滤
	memSizeMB := query.MemorySizeMb
	if memSizeMB > 0 {
		s, e := intervalMem(int(memSizeMB))
		q.GT("memory_size_mb", s)
		q.LE("memory_size_mb", e)
	}

	if len(query.InstanceSpec) > 0 {
		q = q.In("instance_spec", query.InstanceSpec)
	}
	if len(query.EngineArch) > 0 {
		q = q.In("engine_arch", query.EngineArch)
	}
	if len(query.LocalCategory) > 0 {
		q = q.In("local_category", query.LocalCategory)
	}
	if len(query.PrepaidStatus) > 0 {
		q = q.In("prepaid_status", query.PrepaidStatus)
	}
	if len(query.PostpaidStatus) > 0 {
		q = q.In("postpaid_sStatus", query.PostpaidStatus)
	}
	if len(query.Engine) > 0 {
		q = q.In("engine", query.Engine)
	}
	if len(query.EngineVersion) > 0 {
		q = q.In("engine_version", query.EngineVersion)
	}
	if len(query.CpuArch) > 0 {
		q = q.In("cpu_arch", query.CpuArch)
	}
	if len(query.StorageType) > 0 {
		q = q.In("storage_type", query.StorageType)
	}
	if len(query.PerformanceType) > 0 {
		q = q.In("performance_type", query.PerformanceType)
	}
	if len(query.NodeType) > 0 {
		q = q.In("node_type", query.NodeType)
	}

	if len(query.Providers) > 0 {
		q = q.In("provider", query.Providers)
	}

	return q, nil
}

func (manager *SElasticcacheSkuManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ElasticcacheSkuListInput,
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
	q, err = manager.SZoneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ZonalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SZoneResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SElasticcacheSkuManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SCloudregionResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SZoneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

// 获取region下所有Available状态的sku id
func (manager *SElasticcacheSkuManager) FetchSkusByRegion(regionID string) ([]SElasticcacheSku, error) {
	q := manager.Query()
	q = q.Equals("cloudregion_id", regionID)

	skus := make([]SElasticcacheSku, 0)
	err := db.FetchModelObjects(manager, q, &skus)
	if err != nil {
		return nil, errors.Wrap(err, "ElasticcacheSkuManager.FetchSkusByRegion")
	}

	return skus, nil
}

func (self *SElasticcacheSku) GetElasticcacheCount() (int, error) {
	q := ElasticcacheManager.Query().Equals("instance_type", self.Name).Equals("zone_id", self.ZoneId)
	return q.CountWithError()
}

func (manager *SElasticcacheSkuManager) SyncElasticcacheSkus(ctx context.Context, userCred mcclient.TokenCredential, region *SCloudregion, xor bool) compare.SyncResult {
	lockman.LockRawObject(ctx, manager.Keyword(), region.Id)
	defer lockman.ReleaseRawObject(ctx, manager.Keyword(), region.Id)

	syncResult := compare.SyncResult{}

	meta, err := yunionmeta.FetchYunionmeta(ctx)
	if err != nil {
		return syncResult
	}

	extSkus := []SElasticcacheSku{}
	err = meta.List(manager.Keyword(), region.ExternalId, &extSkus)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	dbSkus, err := manager.FetchSkusByRegion(region.GetId())
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	removed := make([]SElasticcacheSku, 0)
	commondb := make([]SElasticcacheSku, 0)
	commonext := make([]SElasticcacheSku, 0)
	added := make([]SElasticcacheSku, 0)

	err = compare.CompareSets(dbSkus, extSkus, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	for i := 0; i < len(removed); i += 1 {
		if cnt, _ := removed[i].GetElasticcacheCount(); cnt > 0 {
			err = removed[i].MarkAsSoldout(ctx)
		} else {
			err = db.RealDeleteModel(ctx, userCred, &removed[i])
		}
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}
	if !xor {
		for i := 0; i < len(commondb); i += 1 {
			err = commondb[i].syncWithCloudSku(ctx, userCred, commonext[i])
			if err != nil {
				syncResult.UpdateError(err)
			} else {
				syncResult.Update()
			}
		}
	}
	ch := make(chan struct{}, options.Options.SkuBatchSync)
	defer close(ch)
	var wg sync.WaitGroup
	for i := 0; i < len(added); i += 1 {
		ch <- struct{}{}
		wg.Add(1)
		go func(sku SElasticcacheSku) {
			defer func() {
				wg.Done()
				<-ch
			}()
			err = region.newFromPublicCloudSku(ctx, userCred, sku.GetExternalId())
			if err != nil {
				syncResult.AddError(err)
				return
			}
			syncResult.Add()
		}(added[i])
	}
	wg.Wait()
	return syncResult
}

func (self *SElasticcacheSku) MarkAsSoldout(ctx context.Context) error {
	_, err := db.UpdateWithLock(ctx, self, func() error {
		self.PrepaidStatus = api.SkuStatusSoldout
		self.PostpaidStatus = api.SkuStatusSoldout
		return nil
	})

	return errors.Wrap(err, "MarkAsSoldout")
}

func (self *SElasticcacheSku) syncWithCloudSku(ctx context.Context, userCred mcclient.TokenCredential, extSku SElasticcacheSku) error {
	_, err := db.Update(self, func() error {
		self.PrepaidStatus = extSku.PrepaidStatus
		self.PostpaidStatus = extSku.PostpaidStatus
		return nil
	})
	return err
}

func (self *SCloudregion) newFromPublicCloudSku(ctx context.Context, userCred mcclient.TokenCredential, externalId string) error {
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

	skuUrl := self.getMetaUrl(meta.ElasticCacheBase, externalId)
	sku := &SElasticcacheSku{}
	sku.SetModelManager(ElasticcacheSkuManager, sku)
	err = meta.Get(skuUrl, sku)
	if err != nil {
		return errors.Wrapf(err, "Get")
	}
	sku.Status = api.SkuStatusAvailable
	sku.CloudregionId = self.Id
	sku.Provider = self.Provider
	if len(sku.ZoneId) > 0 {
		zoneId := yunionmeta.GetZoneIdBySuffix(zoneMaps, sku.ZoneId)
		if len(zoneId) == 0 {
			return errors.Wrapf(err, "empty zoneId for %s", sku.ZoneId)
		}
		sku.ZoneId = zoneId
	}
	if len(sku.SlaveZoneId) > 0 {
		zoneId := yunionmeta.GetZoneIdBySuffix(zoneMaps, sku.SlaveZoneId)
		if len(zoneId) == 0 {
			return errors.Wrapf(err, "empty zoneId for %s", sku.SlaveZoneId)
		}
		sku.SlaveZoneId = zoneId
	}

	return ElasticcacheSkuManager.TableSpec().Insert(ctx, sku)
}

func (manager *SElasticcacheSkuManager) GetPropertyInstanceSpecs(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	q := manager.Query("memory_size_mb")
	q, err := db.ListItemQueryFilters(manager, ctx, q, userCred, query, policy.PolicyActionList)
	if err != nil {
		return nil, errors.Wrap(err, "db.ListItemQueryFilters")
	}

	listQuery := api.ElasticcacheSkuListInput{}
	err = query.Unmarshal(&listQuery)
	if err != nil {
		return nil, errors.Wrap(err, "query.Unmarshal")
	}
	q, err = manager.ListItemFilter(ctx, q, userCred, listQuery)
	if err != nil {
		return nil, errors.Wrap(err, "manager.ListItemFilter")
	}

	q = q.GroupBy(q.Field("memory_size_mb")).Asc(q.Field("memory_size_mb")).Distinct()
	rows, err := q.Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	mems := map[int]bool{}
	mems_mb := jsonutils.NewArray()
	for rows.Next() {
		var ms int
		err := rows.Scan(&ms)
		if err == nil {
			m := roundMem(ms)
			if _, exist := mems[m]; !exist {
				if ms > 0 {
					mems_mb.Add(jsonutils.NewInt(int64(m)))
				}
				mems[m] = true
			}
		} else {
			log.Debugf("SElasticcacheSkuManager.GetPropertyInstanceSpecs %s", err)
		}
	}

	ret := jsonutils.NewDict()
	ret.Add(mems_mb, "mems_mb")
	return ret, nil
}

func (manager *SElasticcacheSkuManager) GetPropertyCapability(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	q := manager.Query("engine", "engine_version", "local_category", "node_type", "performance_type")
	q, err := db.ListItemQueryFilters(manager, ctx, q, userCred, query, policy.PolicyActionList)
	if err != nil {
		return nil, errors.Wrap(err, "db.ListItemQueryFilters")
	}

	listQuery := api.ElasticcacheSkuListInput{}
	err = query.Unmarshal(&listQuery)
	if err != nil {
		return nil, errors.Wrap(err, "query.Unmarshal")
	}
	q, err = manager.ListItemFilter(ctx, q, userCred, listQuery)
	if err != nil {
		return nil, errors.Wrap(err, "manager.ListItemFilter")
	}

	f1 := q.Field("engine")
	f2 := q.Field("engine_version")
	f3 := q.Field("local_category")
	f4 := q.Field("node_type")
	f5 := q.Field("performance_type")
	q = q.GroupBy(f1, f2, f3, f4, f5).Asc(f1, f2, f3, f4, f5)
	rows, err := q.Rows()
	if err != nil {
		return nil, err
	}

	var addNode func(src *jsonutils.JSONDict, keys ...string)
	// keys至少2位，最后一位为叶子节点
	addNode = func(src *jsonutils.JSONDict, keys ...string) {
		length := len(keys)
		if length < 2 {
			return
		}

		if length == 2 {
			if t, err := src.Get(keys[0]); err != nil {
				n := jsonutils.NewArray()
				n.Add(jsonutils.NewString(keys[1]))
				src.Set(keys[0], n)
			} else {
				t.(*jsonutils.JSONArray).Add(jsonutils.NewString(keys[1]))
			}

			return
		}

		var temp *jsonutils.JSONDict
		if t, err := src.Get(keys[0]); err != nil {
			temp = jsonutils.NewDict()
			src.Set(keys[0], temp)
		} else {
			temp = t.(*jsonutils.JSONDict)
		}

		nkeys := keys[1:]
		addNode(temp, nkeys...)
	}

	result := jsonutils.NewDict()
	defer rows.Close()
	for rows.Next() {
		var engine, version, category, node, performance string
		err := rows.Scan(&engine, &version, &category, &node, &performance)
		if err != nil {
			log.Debugf("SElasticcacheSkuManager.GetPropertyCapability %s", err)
			continue
		}

		switch engine {
		case "redis":
			addNode(result, engine, version, category, node, performance)
		case "memcached":
			addNode(result, engine, category)
		}
	}

	return result, nil
}

func (manager *SElasticcacheSkuManager) PerformActionSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	data := query.(*jsonutils.JSONDict)
	cloudprovider := validators.NewModelIdOrNameValidator("cloudprovider", "cloudprovider", nil)
	cloudregion := validators.NewModelIdOrNameValidator("cloudregion", "cloudregion", nil)

	keyV := map[string]validators.IValidator{
		"provider":    cloudprovider.Optional(true),
		"cloudregion": cloudregion.Optional(true),
	}

	for _, v := range keyV {
		if err := v.Validate(ctx, data); err != nil {
			return nil, err
		}
	}

	regions := []SCloudregion{}
	if region, err := data.GetString("cloudregion"); err == nil && len(region) > 0 {
		regions = append(regions, *cloudregion.Model.(*SCloudregion))
	} else if provider, err := data.GetString("cloudprovider"); err == nil && len(provider) > 0 {
		regions, err = CloudregionManager.GetRegionByProvider(provider)
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (manager *SElasticcacheSkuManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusStandaloneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SCloudregionResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SCloudregionResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemExportKeys")
		}
	}
	if keys.Contains("zone") {
		q, err = manager.SZoneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SZoneResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}

func (manager *SElasticcacheSkuManager) PerformSyncSkus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.SkuSyncInput) (jsonutils.JSONObject, error) {
	return PerformActionSyncSkus(ctx, userCred, manager.Keyword(), input)
}

func (manager *SElasticcacheSkuManager) GetPropertySyncTasks(ctx context.Context, userCred mcclient.TokenCredential, query api.SkuTaskQueryInput) (jsonutils.JSONObject, error) {
	return GetPropertySkusSyncTasks(ctx, userCred, query)
}

func (self *SCloudregion) SyncPrivateCloudCacheSkus(ctx context.Context, userCred mcclient.TokenCredential, iskus []cloudprovider.ICloudElasticcacheSku) compare.SyncResult {
	lockman.LockRawObject(ctx, self.Id, ElasticcacheSkuManager.Keyword())
	defer lockman.ReleaseRawObject(ctx, self.Id, ElasticcacheSkuManager.Keyword())

	result := compare.SyncResult{}

	dbSkus, err := self.GetElasticcacheSkus()
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SElasticcacheSku, 0)
	commondb := make([]SElasticcacheSku, 0)
	commonext := make([]cloudprovider.ICloudElasticcacheSku, 0)
	added := make([]cloudprovider.ICloudElasticcacheSku, 0)

	err = compare.CompareSets(dbSkus, iskus, &removed, &commondb, &commonext, &added)
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
	for i := 0; i < len(added); i += 1 {
		err = self.newFromCloudElasticcacheSku(ctx, userCred, added[i])
		if err != nil {
			result.AddError(err)
		} else {
			result.Add()
		}
	}
	return result
}

func (self *SCloudregion) newFromCloudElasticcacheSku(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudElasticcacheSku) error {
	sku := &SElasticcacheSku{}
	sku.SetModelManager(ElasticcacheSkuManager, sku)
	sku.Name = ext.GetName()
	sku.InstanceSpec = ext.GetName()
	sku.Status = apis.SKU_STATUS_AVAILABLE
	sku.CloudregionId = self.Id
	sku.ExternalId = ext.GetGlobalId()
	sku.Provider = self.Provider
	sku.EngineArch = ext.GetEngineArch()
	sku.LocalCategory = ext.GetLocalCategory()
	sku.PrepaidStatus = ext.GetPrepaidStatus()
	sku.PostpaidStatus = ext.GetPostpaidStatus()
	sku.Engine = ext.GetEngine()
	sku.EngineVersion = ext.GetEngineVersion()
	sku.CpuArch = ext.GetCpuArch()
	sku.StorageType = ext.GetStorageType()
	sku.MemorySizeMB = ext.GetMemorySizeMb()
	sku.PerformanceType = ext.GetPerformanceType()
	sku.NodeType = ext.GetNodeType()
	sku.DiskSizeGB = ext.GetDiskSizeGb()
	sku.ShardNum = ext.GetShardNum()
	sku.MaxShardNum = ext.GetMaxShardNum()
	sku.ReplicasNum = ext.GetReplicasNum()
	sku.MaxReplicasNum = ext.GetMaxReplicasNum()
	sku.MaxClients = ext.GetMaxClients()
	sku.MaxConnections = ext.GetMaxConnections()
	sku.MaxInBandwidthMb = ext.GetMaxInBandwidthMb()
	sku.MaxMemoryMB = ext.GetMaxMemoryMb()
	sku.QPS = ext.GetQps()

	zones, err := self.GetZones()
	if err != nil {
		return errors.Wrapf(err, "GetZones")
	}
	zoneId := ext.GetZoneId()
	slaveZoneId := ext.GetSlaveZoneId()
	for i := range zones {
		if len(zoneId) > 0 && strings.HasSuffix(zones[i].ExternalId, zoneId) {
			sku.ZoneId = zones[i].Id
		}
		if len(slaveZoneId) > 0 && strings.HasSuffix(zones[i].ExternalId, slaveZoneId) {
			sku.SlaveZoneId = zones[i].Id
		}
	}

	return ElasticcacheSkuManager.TableSpec().Insert(ctx, sku)
}

// 全量同步elasticcache sku列表.
func SyncElasticCacheSkus(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	if isStart {
		cnt, err := CloudaccountManager.Query().IsTrue("is_public_cloud").CountWithError()
		if err != nil && err != sql.ErrNoRows {
			log.Debugf("SyncElasticCacheSkus %s.sync skipped...", err)
			return
		} else if cnt == 0 {
			log.Debugf("SyncElasticCacheSkus no public cloud.sync skipped...")
			return
		}

		cnt, err = ElasticcacheSkuManager.Query().Limit(1).CountWithError()
		if err != nil && err != sql.ErrNoRows {
			log.Errorf("SyncElasticCacheSkus.QueryElasticcacheSku %s", err)
			return
		} else if cnt > 0 {
			log.Debugf("SyncElasticCacheSkus synced skus, skip...")
			return
		}
	}
	cloudregions := fetchSkuSyncCloudregions()
	if len(cloudregions) == 0 {
		return
	}

	meta, err := yunionmeta.FetchYunionmeta(ctx)
	if err != nil {
		log.Errorf("FetchYunionmeta %v", err)
		return
	}

	index, err := meta.Index(ElasticcacheSkuManager.Keyword())
	if err != nil {
		log.Errorf("get cache sku index error: %v", err)
		return
	}

	for i := range cloudregions {
		region := &cloudregions[i]

		if !region.GetDriver().IsSupportedElasticcache() {
			continue
		}

		skuMeta := &SElasticcacheSku{}
		skuMeta.SetModelManager(ElasticcacheSkuManager, skuMeta)
		skuMeta.Id = region.ExternalId

		oldMd5 := db.Metadata.GetStringValue(ctx, skuMeta, db.SKU_METADAT_KEY, userCred)
		newMd5, ok := index[region.ExternalId]
		if !ok || newMd5 == yunionmeta.EMPTY_MD5 || len(oldMd5) > 0 && newMd5 == oldMd5 {
			continue
		}

		db.Metadata.SetValue(ctx, skuMeta, db.SKU_METADAT_KEY, newMd5, userCred)

		result := ElasticcacheSkuManager.SyncElasticcacheSkus(ctx, userCred, region, false)
		notes := fmt.Sprintf("SyncElasticCacheSkusByRegion %s result: %s", region.Name, result.Result())
		log.Debugf(notes)
	}
}

// 同步Region elasticcache sku列表.
func SyncElasticCacheSkusByRegion(ctx context.Context, userCred mcclient.TokenCredential, region *SCloudregion, xor bool) error {
	if !region.GetDriver().IsSupportedElasticcache() {
		notes := fmt.Sprintf("SyncElasticCacheSkusByRegion %s not support elasticcache", region.Name)
		log.Infof(notes)
		return nil
	}

	result := ElasticcacheSkuManager.SyncElasticcacheSkus(ctx, userCred, region, xor)
	notes := fmt.Sprintf("SyncElasticCacheSkusByRegion %s result: %s", region.Name, result.Result())
	log.Infof(notes)
	return nil
}
