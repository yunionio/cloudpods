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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
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

func (self *SElasticcacheSku) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	return self.SStatusStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
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

		ret = append(ret, fileds)
	}

	return ret
}

func (manager *SElasticcacheSkuManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	data := query.(*jsonutils.JSONDict)
	brands := jsonutils.GetQueryStringArray(query, "brand")
	if len(brands) > 0 {
		q = q.Filter(sqlchemy.In(q.Field("brand"), brands))
		data.Remove("brand")
	}

	q, err := manager.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, data)
	if err != nil {
		return nil, err
	}

	if usable, _ := query.Bool("usable"); usable {
		q = usableFilter(q, true)
		q = q.Equals("postpaid_status", "available")
		q = q.Equals("prepaid_status", "available")
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

	return q, err
}

// 获取所有Available状态的sku id
func (manager *SElasticcacheSkuManager) FetchAllAvailableSkuId() ([]string, error) {
	q := manager.Query()
	q = q.Filter(sqlchemy.OR(
		sqlchemy.Equals(q.Field("prepaid_status"), api.SkuStatusAvailable),
		sqlchemy.Equals(q.Field("postpaid_status"), api.SkuStatusAvailable)))

	skus := make([]SElasticcacheSku, 0)
	err := db.FetchModelObjects(manager, q, &skus)
	if err != nil {
		return nil, err
	}

	ids := make([]string, len(skus))
	for i := range skus {
		ids[i] = skus[i].GetId()
	}

	return ids, nil
}

// 获取region下所有Available状态的sku id
func (manager *SElasticcacheSkuManager) FetchAllAvailableSkuIdByRegion(regionID string) ([]string, error) {
	q := manager.Query()
	q = q.Filter(sqlchemy.OR(
		sqlchemy.Equals(q.Field("prepaid_status"), api.SkuStatusAvailable),
		sqlchemy.Equals(q.Field("postpaid_status"), api.SkuStatusAvailable)))
	q = q.Equals("cloudregion_id", regionID)

	skus := make([]SElasticcacheSku, 0)
	err := db.FetchModelObjects(manager, q, &skus)
	if err != nil {
		return nil, err
	}

	ids := make([]string, len(skus))
	for i := range skus {
		ids[i] = skus[i].GetId()
	}

	return ids, nil
}

func (manager *SElasticcacheSkuManager) InitializeData() error {
	count, err := manager.Query().Limit(1).CountWithError()
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if count > 0 {
		return nil
	}

	count, err = CloudaccountManager.Query().IsTrue("is_public_cloud").CountWithError()
	if count > 0 {
		SyncElasticCacheSkus(nil, nil, true)
	}

	return nil
}

// sku标记为soldout状态。
func (manager *SElasticcacheSkuManager) MarkAsSoldout(id string) error {
	if len(id) == 0 {
		log.Debugf("MarkAsSoldout sku id should not be emtpy")
		return nil
	}

	isku, err := manager.FetchById(id)
	if err != nil {
		return err
	}

	sku, ok := isku.(*SServerSku)
	if !ok {
		return fmt.Errorf("%s is not a sku object", id)
	}

	_, err = manager.TableSpec().Update(sku, func() error {
		sku.PrepaidStatus = api.SkuStatusSoldout
		sku.PostpaidStatus = api.SkuStatusSoldout
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

// sku标记为soldout状态。
func (manager *SElasticcacheSkuManager) MarkAllAsSoldout(ids []string) error {
	var err error
	for _, id := range ids {
		err = manager.MarkAsSoldout(id)
		if err != nil {
			return err
		}
	}

	return nil
}
