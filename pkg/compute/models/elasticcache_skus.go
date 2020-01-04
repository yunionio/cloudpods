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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SElasticcacheSkuManager struct {
	db.SStatusStandaloneResourceBaseManager
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

	SCloudregionResourceBase        // 区域
	SZoneResourceBase               // 主可用区
	SlaveZoneId              string `width:"64" charset:"ascii" nullable:"false" list:"user" create:"admin_optional" update:"admin"` // 备可用区

	InstanceSpec  string `width:"96" charset:"ascii" nullable:"false" list:"user" create:"admin_optional" update:"admin"`
	EngineArch    string `width:"32" charset:"utf8" nullable:"false" list:"user" create:"admin_optional" update:"admin"`
	LocalCategory string `width:"32" charset:"utf8" nullable:"false" list:"user" create:"admin_optional" update:"admin" default:""`

	PrepaidStatus  string `width:"32" charset:"utf8" nullable:"false" list:"user" create:"admin_optional" update:"admin" default:"available"`
	PostpaidStatus string `width:"32" charset:"utf8" nullable:"false" list:"user" create:"admin_optional" update:"admin" default:"available"`

	Engine          string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"admin_optional" update:"admin"` // 引擎	redis|memcached
	EngineVersion   string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"admin_optional" update:"admin"` // 引擎版本	3.0
	CpuArch         string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"admin_optional" update:"admin"` // CPU 架构 x86|ARM
	StorageType     string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"admin_optional" update:"admin"` // 存储类型	DRAM|SCM
	PerformanceType string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"admin_optional" update:"admin"` // standrad|enhanced
	NodeType        string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"admin_optional" update:"admin"` // single（单副本） | double（双副本) | readone (单可读) | readthree （3可读） | readfive（5只读）

	MemorySizeMB   int `nullable:"false" list:"user" create:"admin_required" update:"admin"` // 内存容量
	DiskSizeGB     int `nullable:"false" list:"user" create:"admin_required" update:"admin"` // 套餐附带硬盘容量
	ShardNum       int `nullable:"false" list:"user" create:"admin_optional" update:"admin"` // 最小分片数量
	MaxShardNum    int `nullable:"false" list:"user" create:"admin_optional" update:"admin"` // 最大分片数量
	ReplicasNum    int `nullable:"false" list:"user" create:"admin_optional" update:"admin"` // 最小副本数量
	MaxReplicasNum int `nullable:"false" list:"user" create:"admin_optional" update:"admin"` // 最大副本数量

	MaxClients       int `nullable:"false" list:"user" create:"admin_optional" update:"admin"` // 最大客户端数
	MaxConnections   int `nullable:"false" list:"user" create:"admin_optional" update:"admin"` // 最大连接数
	MaxInBandwidthMb int `nullable:"false" list:"user" create:"admin_optional" update:"admin"` // 最大内网带宽
	MaxMemoryMB      int `nullable:"false" list:"user" create:"admin_optional" update:"admin"` // 实际可使用的最大内存
	QPS              int `nullable:"false" list:"user" create:"admin_optional" update:"admin"` // QPS参考值

	Provider string `width:"32" charset:"ascii" nullable:"false" list:"user" create:"admin_required" update:"admin"` // 公有云厂商	Aliyun/Azure/AWS/Qcloud/...
}

func (self SElasticcacheSku) GetGlobalId() string {
	return self.ExternalId
}

func (self *SElasticcacheSku) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	return self.SStatusStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
}

func (manager *SElasticcacheSkuManager) GetSkuCountByRegion(regionId string) (int, error) {
	q := manager.Query().Equals("cloudregion_id", regionId)

	return q.CountWithError()
}

func (manager *SElasticcacheSkuManager) FetchCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, objs []db.IModel, fields stringutils2.SSortedStrings) []*jsonutils.JSONDict {
	regions := map[string]string{}
	for i := range objs {
		cloudregionId := objs[i].(*SElasticcacheSku).CloudregionId
		if _, ok := regions[cloudregionId]; !ok {
			regions[cloudregionId] = cloudregionId
		}
	}

	regionIds := []string{}
	for k, _ := range regions {
		regionIds = append(regionIds, regions[k])
	}

	if len(regionIds) == 0 {
		return nil
	}

	regionObjs := []SCloudregion{}
	err := CloudregionManager.Query().In("id", regionIds).All(&regionObjs)
	if err != nil {
		log.Errorf("elasticcacheSkuManager.FetchCustomizeColumns %s", err)
		return nil
	}

	for i := range regionObjs {
		regionObj := regionObjs[i]
		regions[regionObj.Id] = regionObj.Name
	}

	ret := []*jsonutils.JSONDict{}
	for i := range objs {
		cloudregionId := objs[i].(*SElasticcacheSku).CloudregionId

		fileds := jsonutils.NewDict()
		fileds.Set("region", jsonutils.NewString(regions[cloudregionId]))
		if region, err := db.FetchById(CloudregionManager, cloudregionId); err == nil {
			fileds.Set("region_external_id", jsonutils.NewString(region.(*SCloudregion).ExternalId))
			segs := strings.Split(region.(*SCloudregion).ExternalId, "/")
			if len(segs) >= 2 {
				fileds.Set("region_ext_id", jsonutils.NewString(segs[1]))
			}
		}

		zoneId := objs[i].(*SElasticcacheSku).ZoneId
		if len(zoneId) > 0 {
			if zone, err := db.FetchById(ZoneManager, zoneId); err == nil {
				fileds.Set("zone_external_id", jsonutils.NewString(zone.(*SZone).ExternalId))
				segs := strings.Split(zone.(*SZone).ExternalId, "/")
				if len(segs) >= 3 {
					fileds.Set("zone_ext_id", jsonutils.NewString(segs[2]))
				}
			}
		}

		ret = append(ret, fileds)
	}

	return ret
}

func (manager *SElasticcacheSkuManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	data := query.(*jsonutils.JSONDict)
	q, err := manager.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, data)
	if err != nil {
		return nil, err
	}

	if usable, _ := query.Bool("usable"); usable {
		q = usableFilter(q, true)
		sq := sqlchemy.OR(sqlchemy.Equals(q.Field("prepaid_status"), api.SkuStatusAvailable), sqlchemy.Equals(q.Field("postpaid_status"), api.SkuStatusAvailable))
		q = q.Filter(sq)
	}

	if b, _ := query.GetString("billing_type"); len(b) > 0 {
		switch b {
		case billing.BILLING_TYPE_POSTPAID:
			q = q.Equals("postpaid_status", api.SkuStatusAvailable)
		case billing.BILLING_TYPE_PREPAID:
			q = q.Equals("prepaid_status", api.SkuStatusAvailable)
		}
	}

	q, err = validators.ApplyModelFilters(q, data, []*validators.ModelFilterOptions{
		{Key: "zone", ModelKeyword: "zone", OwnerId: userCred},
		{Key: "cloudregion", ModelKeyword: "cloudregion", OwnerId: userCred},
	})
	if err != nil {
		return nil, err
	}

	city, _ := query.GetString("city")
	if len(city) > 0 {
		regionTable := CloudregionManager.Query().SubQuery()
		q = q.Join(regionTable, sqlchemy.Equals(regionTable.Field("id"), q.Field("cloudregion_id"))).Filter(sqlchemy.Equals(regionTable.Field("city"), city))
	}

	// 按区间查询内存, 避免0.75G这样的套餐不好过滤
	memSizeMB, _ := query.Int("memory_size_mb")
	if memSizeMB > 0 {
		s, e := intervalMem(int(memSizeMB))
		q.GT("memory_size_mb", s)
		q.LE("memory_size_mb", e)
		queryDict := query.(*jsonutils.JSONDict)
		queryDict.Remove("memory_size_mb")
	}

	return q, err
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

func (manager *SElasticcacheSkuManager) syncElasticcacheSkus(ctx context.Context, userCred mcclient.TokenCredential, region *SCloudregion, extSkuMeta *SSkuResourcesMeta) compare.SyncResult {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	syncResult := compare.SyncResult{}

	extSkuMeta.SetRegionFilter(region)
	extSkus, err := extSkuMeta.GetElasticCacheSkus()
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
		err = removed[i].MarkAsSoldout(ctx)
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
		err = manager.newFromCloudSku(ctx, userCred, added[i])
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncResult.Add()
		}
	}
	return syncResult
}

func (self *SElasticcacheSku) MarkAsSoldout(ctx context.Context) error {
	_, err := db.UpdateWithLock(ctx, self, func() error {
		self.PrepaidStatus = api.SkuStatusSoldout
		self.PostpaidStatus = api.SkuStatusSoldout
		return nil
	})

	return errors.Wrap(err, "ElasticcacheSku.MarkAsSoldout")
}

func (self *SElasticcacheSku) syncWithCloudSku(ctx context.Context, userCred mcclient.TokenCredential, extSku SElasticcacheSku) error {
	_, err := db.Update(self, func() error {
		self.PrepaidStatus = extSku.PrepaidStatus
		self.PostpaidStatus = extSku.PostpaidStatus
		return nil
	})
	return err
}

func (manager *SElasticcacheSkuManager) newFromCloudSku(ctx context.Context, userCred mcclient.TokenCredential, extSku SElasticcacheSku) error {
	return manager.TableSpec().Insert(&extSku)
}

func (manager *SElasticcacheSkuManager) AllowGetPropertyInstanceSpecs(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (manager *SElasticcacheSkuManager) GetPropertyInstanceSpecs(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	q := manager.Query("memory_size_mb")
	q, err := db.ListItemQueryFilters(manager, ctx, q, userCred, query, policy.PolicyActionList)
	if err != nil {
		return nil, err
	}

	q, err = manager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}

	q = q.GroupBy(q.Field("memory_size_mb")).Asc(q.Field("memory_size_mb")).Distinct()
	rows, err := q.Rows()
	if err != nil {
		return nil, err
	}

	mems := map[int]bool{}
	mems_mb := jsonutils.NewArray()
	defer rows.Close()
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

func (manager *SElasticcacheSkuManager) AllowGetPropertyCapability(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (manager *SElasticcacheSkuManager) GetPropertyCapability(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	q := manager.Query("engine", "engine_version", "local_category", "node_type", "performance_type")
	q, err := db.ListItemQueryFilters(manager, ctx, q, userCred, query, policy.PolicyActionList)
	if err != nil {
		return nil, err
	}

	q, err = manager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
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
