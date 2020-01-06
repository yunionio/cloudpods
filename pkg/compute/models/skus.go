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
	"crypto/md5"
	"database/sql"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/hashcache"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

var Cache *hashcache.Cache

type SServerSkuManager struct {
	db.SStatusStandaloneResourceBaseManager
}

var ServerSkuManager *SServerSkuManager

func init() {
	ServerSkuManager = &SServerSkuManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SServerSku{},
			"serverskus_tbl",
			"serversku",
			"serverskus",
		),
	}
	ServerSkuManager.NameRequireAscii = false
	ServerSkuManager.SetVirtualObject(ServerSkuManager)

	Cache = hashcache.NewCache(2048, time.Second*300)
}

// SServerSku 实际对应的是instance type清单. 这里的Sku实际指的是instance type。
type SServerSku struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase
	SCloudregionResourceBase
	SZoneResourceBase

	Enabled bool `nullable:"true" list:"user" create:"optional"`
	// SkuId       string `width:"64" charset:"ascii" nullable:"false" list:"user" create:"admin_required"`                 // x2.large
	InstanceTypeFamily   string `width:"32" charset:"ascii" nullable:"true" list:"user" create:"admin_optional" update:"admin"`           // x2
	InstanceTypeCategory string `width:"32" charset:"utf8" nullable:"true" list:"user" create:"admin_optional" update:"admin"`            // 通用型
	LocalCategory        string `width:"32" charset:"utf8" nullable:"true" list:"user" create:"admin_optional" update:"admin" default:""` // 记录本地分类

	PrepaidStatus  string `width:"32" charset:"utf8" nullable:"true" list:"user" create:"admin_optional" default:"available"` // 预付费资源状态   available|soldout
	PostpaidStatus string `width:"32" charset:"utf8" nullable:"true" list:"user" create:"admin_optional" default:"available"` // 按需付费资源状态  available|soldout

	CpuCoreCount int `nullable:"false" list:"user" create:"admin_required"`
	MemorySizeMB int `nullable:"false" list:"user" create:"admin_required"`

	OsName string `width:"32" charset:"ascii" nullable:"true" list:"user" create:"admin_optional" update:"admin" default:"Any"` // Windows|Linux|Any

	SysDiskResizable tristate.TriState `default:"true" nullable:"true" list:"user" create:"admin_optional" update:"admin"`
	SysDiskType      string            `width:"32" charset:"ascii" nullable:"true" list:"user" create:"admin_optional" update:"admin"`
	SysDiskMinSizeGB int               `nullable:"true" list:"user" create:"admin_optional" update:"admin"` // not required。 windows比较新的版本都是50G左右。
	SysDiskMaxSizeGB int               `nullable:"true" list:"user" create:"admin_optional" update:"admin"` // not required

	AttachedDiskType   string `nullable:"true" list:"user" create:"admin_optional" update:"admin"`
	AttachedDiskSizeGB int    `nullable:"true" list:"user" create:"admin_optional" update:"admin"`
	AttachedDiskCount  int    `nullable:"true" list:"user" create:"admin_optional" update:"admin"`

	DataDiskTypes    string `width:"128" charset:"ascii" nullable:"true" list:"user" create:"admin_optional" update:"admin"`
	DataDiskMaxCount int    `nullable:"true" list:"user" create:"admin_optional" update:"admin"`

	NicType     string `nullable:"true" list:"user" create:"admin_optional" update:"admin"`
	NicMaxCount int    `default:"1" nullable:"true" list:"user" create:"admin_optional" update:"admin"`

	GpuAttachable tristate.TriState `default:"true" nullable:"true" list:"user" create:"admin_optional" update:"admin"`
	GpuSpec       string            `width:"128" charset:"ascii" nullable:"true" list:"user" create:"admin_optional" update:"admin"`
	GpuCount      int               `nullable:"true" list:"user" create:"admin_optional" update:"admin"`
	GpuMaxCount   int               `nullable:"true" list:"user" create:"admin_optional" update:"admin"`

	Provider string `width:"64" charset:"ascii" nullable:"true" list:"user" default:"OneCloud" create:"admin_optional"`
}

type SInstanceSpecQueryParams struct {
	Provider       string
	PublicCloud    bool
	ZoneId         string
	PostpaidStatus string
	PrepaidStatus  string
	IngoreCache    bool
}

func (self *SInstanceSpecQueryParams) GetCacheKey() string {
	hashStr := fmt.Sprintf("%s:%t:%s:%s:%s", self.Provider, self.PublicCloud, self.ZoneId, self.PostpaidStatus, self.PrepaidStatus)
	_md5 := md5.Sum([]byte(hashStr))
	return "InstanceSpecs_" + fmt.Sprintf("%x", _md5)
}

func NewInstanceSpecQueryParams(query jsonutils.JSONObject) *SInstanceSpecQueryParams {
	zone := jsonutils.GetAnyString(query, []string{"zone", "zone_id"})
	postpaid, _ := query.GetString("postpaid_status")
	prepaid, _ := query.GetString("prepaid_status")
	ingore_cache, _ := query.Bool("ingore_cache")
	provider := normalizeProvider(jsonutils.GetAnyString(query, []string{"provider"}))
	public_cloud, _ := query.Bool("public_cloud")
	if utils.IsInStringArray(provider, cloudprovider.GetPublicProviders()) {
		public_cloud = true
	}

	params := &SInstanceSpecQueryParams{
		Provider:       provider,
		PublicCloud:    public_cloud,
		ZoneId:         zone,
		PostpaidStatus: postpaid,
		PrepaidStatus:  prepaid,
		IngoreCache:    ingore_cache,
	}

	return params
}

func sliceToJsonObject(items []int) jsonutils.JSONObject {
	sort.Slice(items, func(i, j int) bool {
		if items[i] < items[j] {
			return true
		}

		return false
	})

	ret := jsonutils.NewArray()
	for _, item := range items {
		ret.Add(jsonutils.NewInt(int64(item)))
	}

	return ret
}

func inWhiteList(provider string) bool {
	// 私有云套餐也允许更新删除
	return provider == api.CLOUD_PROVIDER_ONECLOUD || utils.IsInStringArray(provider, api.PRIVATE_CLOUD_PROVIDERS)
}

func genInstanceType(family string, cpu, mem_mb int64) (string, error) {
	if cpu <= 0 {
		return "", fmt.Errorf("cpu_core_count should great than zero")
	}

	if mem_mb <= 0 || mem_mb%1024 != 0 {
		return "", fmt.Errorf("memory_size_mb should great than zero. and should be integral multiple of 1024")
	}

	return fmt.Sprintf("ecs.%s.c%dm%d", family, cpu, mem_mb/1024), nil
}

func skuRelatedGuestCount(self *SServerSku) (int, error) {
	var q *sqlchemy.SQuery
	if len(self.ZoneId) > 0 {
		hostTable := HostManager.Query().SubQuery()
		guestTable := GuestManager.Query().SubQuery()
		q = guestTable.Query().Join(hostTable, sqlchemy.Equals(hostTable.Field("id"), guestTable.Field("host_id")))
		q = q.Filter(sqlchemy.Equals(hostTable.Field("zone_id"), self.ZoneId))
	} else {
		q = GuestManager.Query()
	}

	q = q.Equals("instance_type", self.GetName())
	return q.CountWithError()
}

func getNameAndExtId(resId string, manager db.IModelManager) (string, string, error) {
	nKey := resId + ".Name"
	eKey := resId + ".ExtId"
	name := Cache.Get(nKey)
	extId := Cache.Get(eKey)
	if name == nil || extId == nil {
		imodel, err := manager.FetchById(resId)
		if err != nil {
			return "", "", err
		}

		_name := imodel.GetName()
		segs := strings.Split(_name, " ")
		if len(segs) > 1 {
			name = strings.Join(segs[1:], " ")
		} else {
			name = _name
		}

		_extId := ""
		if region, ok := imodel.(*SCloudregion); ok {
			_extId = region.ExternalId
		} else if zone, ok := imodel.(*SZone); ok {
			_extId = zone.ExternalId
		} else {
			return "", "", fmt.Errorf("res %s not a region/zone resource", resId)
		}

		segs = strings.Split(_extId, "/")
		if len(segs) > 0 {
			extId = segs[len(segs)-1]
		}

		Cache.Set(nKey, name)
		Cache.Set(eKey, extId)
	}

	return name.(string), extId.(string), nil
}

func (self *SServerSkuManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (self SServerSku) GetGlobalId() string {
	return self.ExternalId
}

func (self *SServerSku) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (self *SServerSku) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SStatusStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	// count
	var count int
	countKey := self.GetId() + ".total_guest_count"
	v := Cache.Get(countKey)
	if v == nil {
		count, _ = skuRelatedGuestCount(self)
		Cache.Set(countKey, count)
	} else {
		count = v.(int)
	}

	extra.Add(jsonutils.NewInt(int64(count)), "total_guest_count")

	zoneInfo := self.SZoneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	if zoneInfo != nil {
		extra.Update(zoneInfo)
	}

	// region
	name, extId, err := getNameAndExtId(self.CloudregionId, CloudregionManager)
	if err == nil {
		extra.Add(jsonutils.NewString(name), "region")
		extra.Add(jsonutils.NewString(extId), "region_ext_id")
	} else {
		log.Debugf("GetCustomizeColumns %s", err)
	}

	return extra
}

func (self *SServerSku) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SStatusStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	count, _ := skuRelatedGuestCount(self)
	extra.Add(jsonutils.NewInt(int64(count)), "total_guest_count")
	return extra, nil
}

func (manager *SServerSkuManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, manager)
}

func (self *SServerSkuManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	enabledV := validators.NewBoolValidator("enabled")
	regionV := validators.NewModelIdOrNameValidator("cloudregion", "cloudregion", ownerId)
	zoneV := validators.NewModelIdOrNameValidator("zone", "zone", ownerId)
	cpuV := validators.NewRangeValidator("cpu_core_count", 1, 256)
	memV := validators.NewRangeValidator("memory_size_mb", 512, 1024*512)
	categoryV := validators.NewStringChoicesValidator("instance_type_category", api.SKU_FAMILIES)
	keyV := map[string]validators.IValidator{
		"enabled":                enabledV.Default(true),
		"region":                 regionV.Optional(true),
		"zone":                   zoneV.Optional(true),
		"cpu_core_count":         cpuV,
		"memory_size_mb":         memV,
		"instance_type_category": categoryV.Default(api.SkuCategoryGeneralPurpose),
	}

	for _, v := range keyV {
		err := v.Validate(data)
		if err != nil {
			return nil, err
		}
	}

	provider, _ := data.GetString("provider")
	brand, _ := data.GetString("brand")
	if (len(provider) == 0 && len(brand) == 0) || provider == api.CLOUD_PROVIDER_ONECLOUD || brand == api.ONECLOUD_BRAND_ONECLOUD {
		provider = api.CLOUD_PROVIDER_ONECLOUD
		brand = api.ONECLOUD_BRAND_ONECLOUD
	} else if len(brand) > 0 {
		q := CloudaccountManager.Query().Equals("brand", brand).IsFalse("is_public_cloud").IsFalse("is_on_premise")
		cloudaccounts := []SCloudaccount{}
		err := db.FetchModelObjects(CloudaccountManager, q, &cloudaccounts)
		if err != nil {
			return nil, httperrors.NewGeneralError(err)
		}
		if len(cloudaccounts) == 0 {
			return nil, httperrors.NewInputParameterError("Not support brand %s or not import %s cloudaccount", brand, brand)
		}
		if len(provider) > 0 && provider != cloudaccounts[0].Provider {
			return nil, httperrors.NewInputParameterError("")
		}
		provider = cloudaccounts[0].Provider
	} else if len(provider) > 0 {
		q := CloudaccountManager.Query().Equals("provider", provider).IsFalse("is_public_cloud").IsFalse("is_on_premise")
		cloudaccounts := []SCloudaccount{}
		err := db.FetchModelObjects(CloudaccountManager, q, &cloudaccounts)
		if err != nil {
			return nil, httperrors.NewGeneralError(err)
		}
		if len(cloudaccounts) == 0 {
			return nil, httperrors.NewInputParameterError("Not support provider %s or not import %s cloudaccount", provider, provider)
		}
		brand = provider
	}
	data.Set("provider", jsonutils.NewString(provider))
	data.Set("brand", jsonutils.NewString(brand))
	data.Set("status", jsonutils.NewString(api.SkuStatusReady))

	family := api.InstanceFamilies[categoryV.Value]
	data.Add(jsonutils.NewString(categoryV.Value), "local_category")
	data.Set("instance_type_family", jsonutils.NewString(family))

	if !data.Contains("name") {
		// 格式 ecs.g1.c1m1
		name, err := genInstanceType(family, cpuV.Value, memV.Value)
		if err != nil {
			return nil, httperrors.NewInputParameterError(err.Error())
		}
		q := self.Query().Equals("name", name)
		count, err := q.CountWithError()
		if err != nil {
			return nil, httperrors.NewGeneralError(fmt.Errorf("checkout server sku name duplicate error: %v", err))
		}
		if count > 0 {
			return nil, httperrors.NewDuplicateResourceError("Duplicate sku %s", name)
		}
		data.Set("name", jsonutils.NewString(name))
	}

	input := apis.StatusStandaloneResourceCreateInput{}
	err := data.Unmarshal(&input)
	if err != nil {
		return nil, httperrors.NewInternalServerError("unmarshal StatusStandaloneResourceCreateInput fail %s", err)
	}
	input, err = self.SStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input)
	if err != nil {
		return nil, err
	}
	data.Update(jsonutils.Marshal(input))
	return data, nil
}

func (self *SServerSku) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SStatusStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	if self.Provider != api.CLOUD_PROVIDER_ONECLOUD {
		_, err := db.Update(self, func() error {
			self.CloudregionId = ""
			self.ZoneId = ""
			return nil
		})
		if err != nil {
			log.Warningf("failed to update %s sku %s cloudregion and zone info: %v", self.Provider, self.Name, err)
		}
	}
}

func (self *SServerSku) GetPrivateCloudproviders() ([]SCloudprovider, error) {
	providers := []SCloudprovider{}
	q := CloudproviderManager.Query().In("provider", CloudproviderManager.GetPrivateProviderProvidersQuery())
	err := db.FetchModelObjects(CloudproviderManager, q, &providers)
	if err != nil {
		return nil, err
	}
	return providers, nil
}

func (self *SServerSku) GetRegion() (*SCloudregion, error) {
	regionObj, err := CloudregionManager.FetchById(self.CloudregionId)
	if err != nil {
		return nil, err
	}
	return regionObj.(*SCloudregion), nil
}

func (self *SServerSkuManager) ClearSchedDescCache(wait bool) error {
	s := auth.GetAdminSession(context.Background(), options.Options.Region, "")
	_, err := modules.SchedManager.SyncSku(s, true)
	if err != nil {
		return errors.Wrapf(err, "chedManager.SyncSku")
	}
	return nil
}

func (self *SServerSkuManager) FetchByZoneExtId(zoneExtId string, name string) (db.IModel, error) {
	zoneObj, err := db.FetchByExternalId(ZoneManager, zoneExtId)
	if err != nil {
		return nil, err
	}

	return self.FetchByZoneId(zoneObj.GetId(), name)
}

func (self *SServerSkuManager) FetchByZoneId(zoneId string, name string) (db.IModel, error) {
	q := self.Query().Equals("zone_id", zoneId).Equals("name", name)

	count, err := q.CountWithError()
	if err != nil {
		return nil, err
	}

	if count == 1 {
		obj, err := db.NewModelObject(self)
		if err != nil {
			return nil, err
		}
		err = q.First(obj)
		if err != nil {
			return nil, err
		} else {
			return obj.(db.IStandaloneModel), nil
		}
	} else if count > 1 {
		return nil, sqlchemy.ErrDuplicateEntry
	} else {
		return nil, sql.ErrNoRows
	}
}

func (self *SServerSkuManager) AllowGetPropertyInstanceSpecs(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

// 四舍五入
func round(n int, step int) int {
	q := float64(n) / float64(step)
	return int(math.Floor(q+0.5)) * step
}

// 内存按GB取整
func roundMem(n int) int {
	if n <= 512 {
		return 512
	}

	return round(n, 1024)
}

// step必须是偶数
func interval(n int, step int) (int, int) {
	r := round(n, step)
	start := r - step/2
	end := r + step/2
	return start, end
}

// 计算内存所在区间范围
func intervalMem(n int) (int, int) {
	if n <= 512 {
		return 0, 512
	}
	return interval(n, 1024)
}

func normalizeProvider(provider string) string {
	if len(provider) == 0 {
		return provider
	}

	for _, p := range api.CLOUD_PROVIDERS {
		if strings.ToLower(p) == strings.ToLower(provider) {
			return p
		}
	}

	return provider
}

func networkUsableRegionQueries(f sqlchemy.IQueryField) []sqlchemy.ICondition {
	iconditions := make([]sqlchemy.ICondition, 0)
	providers := CloudproviderManager.Query().SubQuery()
	networks := NetworkManager.Query().SubQuery()
	wires := WireManager.Query().SubQuery()
	vpcs := VpcManager.Query().SubQuery()

	sq := vpcs.Query(sqlchemy.DISTINCT("cloudregion_id", vpcs.Field("cloudregion_id")))
	sq = sq.Join(wires, sqlchemy.Equals(vpcs.Field("id"), wires.Field("vpc_id")))
	sq = sq.Join(networks, sqlchemy.Equals(wires.Field("id"), networks.Field("wire_id")))
	sq = sq.Join(providers, sqlchemy.Equals(vpcs.Field("manager_id"), providers.Field("id")))
	sq = sq.Filter(sqlchemy.Equals(networks.Field("status"), api.NETWORK_STATUS_AVAILABLE))
	sq = sq.Filter(sqlchemy.IsTrue(providers.Field("enabled")))
	sq = sq.Filter(sqlchemy.In(providers.Field("status"), api.CLOUD_PROVIDER_VALID_STATUS))
	sq = sq.Filter(sqlchemy.In(providers.Field("health_status"), api.CLOUD_PROVIDER_VALID_HEALTH_STATUS))
	sq = sq.Filter(sqlchemy.Equals(vpcs.Field("status"), api.VPC_STATUS_AVAILABLE))

	sq2 := vpcs.Query(sqlchemy.DISTINCT("cloudregion_id", vpcs.Field("cloudregion_id")))
	sq2 = sq2.Join(wires, sqlchemy.Equals(vpcs.Field("id"), wires.Field("vpc_id")))
	sq2 = sq2.Join(networks, sqlchemy.Equals(wires.Field("id"), networks.Field("wire_id")))
	sq2 = sq2.Filter(sqlchemy.Equals(networks.Field("status"), api.NETWORK_STATUS_AVAILABLE))
	sq2 = sq2.Filter(sqlchemy.IsNullOrEmpty(vpcs.Field("manager_id")))
	sq2 = sq2.Filter(sqlchemy.Equals(vpcs.Field("status"), api.VPC_STATUS_AVAILABLE))

	iconditions = append(iconditions, sqlchemy.In(f, sq.SubQuery()))
	iconditions = append(iconditions, sqlchemy.In(f, sq2.SubQuery()))
	return iconditions
}

func usableFilter(q *sqlchemy.SQuery, public_cloud bool) *sqlchemy.SQuery {
	// 过滤出公有云provider状态健康的sku
	if public_cloud {
		providerTable := CloudproviderManager.Query().SubQuery()
		providerRegionTable := CloudproviderRegionManager.Query().SubQuery()

		subq := providerRegionTable.Query(sqlchemy.DISTINCT("cloudregion_id", providerRegionTable.Field("cloudregion_id")))
		subq = subq.Join(providerTable, sqlchemy.Equals(providerRegionTable.Field("cloudprovider_id"), providerTable.Field("id")))
		subq = subq.Filter(sqlchemy.IsTrue(providerTable.Field("enabled")))
		subq = subq.Filter(sqlchemy.In(providerTable.Field("status"), api.CLOUD_PROVIDER_VALID_STATUS))
		subq = subq.Filter(sqlchemy.In(providerTable.Field("health_status"), api.CLOUD_PROVIDER_VALID_HEALTH_STATUS))
		q = q.Filter(sqlchemy.In(q.Field("cloudregion_id"), subq.SubQuery()))
	}

	// 过滤出network usable的sku
	if public_cloud {
		iconditions := NetworkUsableZoneQueries(q.Field("zone_id"), true, true)
		iconditions = append(iconditions, sqlchemy.IsNullOrEmpty(q.Field("zone_id"))) //Azure的zone_id可能为空
		q = q.Filter(sqlchemy.OR(iconditions...))
	} else {
		// 本地IDC sku 只定义到region层级, zone id 为空.因此只能按region查询
		iconditions := networkUsableRegionQueries(q.Field("cloudregion_id"))
		// 私有云 sku region及zone为空
		iconditions = append(iconditions, sqlchemy.IsNullOrEmpty(q.Field("cloudregion_id")))
		q = q.Filter(sqlchemy.OR(iconditions...))
	}

	return q
}

func (manager *SServerSkuManager) GetPropertyInstanceSpecs(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	q, err := manager.ListItemFilter(ctx, manager.Query(), userCred, query)
	if err != nil {
		return nil, err
	}

	skus := make([]SServerSku, 0)
	q = q.GroupBy(q.Field("cpu_core_count"), q.Field("memory_size_mb"))
	q = q.Asc(q.Field("cpu_core_count"), q.Field("memory_size_mb"))
	err = db.FetchModelObjects(manager, q, &skus)
	if err != nil {
		return nil, httperrors.NewBadRequestError("instance specs list query error")
	}

	cpus := jsonutils.NewArray()
	mems_mb := []int{}
	cpu_mems_mb := map[string][]int{}

	mems := map[int]bool{}
	oc := 0
	for i := range skus {
		nc := skus[i].CpuCoreCount
		nm := roundMem(skus[i].MemorySizeMB) // 内存按GB取整

		if nc > oc {
			cpus.Add(jsonutils.NewInt(int64(nc)))
			oc = nc
		}

		if _, exists := mems[nm]; !exists {
			mems_mb = append(mems_mb, nm)
			mems[nm] = true
		}

		k := strconv.Itoa(nc)
		if _, exists := cpu_mems_mb[k]; !exists {
			cpu_mems_mb[k] = []int{nm}
		} else {
			idx := len(cpu_mems_mb[k]) - 1
			if cpu_mems_mb[k][idx] != nm {
				cpu_mems_mb[k] = append(cpu_mems_mb[k], nm)
			}
		}
	}

	ret := jsonutils.NewDict()
	ret.Add(cpus, "cpus")
	ret.Add(sliceToJsonObject(mems_mb), "mems_mb")

	r_obj := jsonutils.Marshal(&cpu_mems_mb)
	ret.Add(r_obj, "cpu_mems_mb")
	return ret, nil
}

func (self *SServerSku) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return inWhiteList(self.Provider) && db.IsAdminAllowUpdate(userCred, self)
}

func (self *SServerSku) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	// 目前不允许改sku信息
	if !inWhiteList(self.Provider) {
		return nil, httperrors.NewForbiddenError("can not update instance_type for public cloud %s", self.Provider)
	}

	if data.Contains("name") {
		return nil, httperrors.NewUnsupportOperationError("Cannot change server sku name")
	}
	return self.SStatusStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (self *SServerSku) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return inWhiteList(self.Provider) && db.IsAdminAllowDelete(userCred, self)
}

func (self *SServerSku) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("SServerSku delete do nothing")
	return nil
}

func (self *SServerSku) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	ServerSkuManager.ClearSchedDescCache(true)
	return self.SStatusStandaloneResourceBase.Delete(ctx, userCred)
}

func (self *SServerSku) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartServerSkuDeleteTask(ctx, userCred, jsonutils.QueryBoolean(data, "purge", false), "")
}

func (self *SServerSku) StartServerSkuDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, purge bool, parentTaskId string) error {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewBool(purge), "purge")
	task, err := taskman.TaskManager.NewTask(ctx, "ServerSkuDeleteTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		log.Errorf("newTask ServerSkuDeleteTask fail %s", err)
		return err
	}
	self.SetStatus(userCred, api.SkuStatusDeleting, "start to delete")
	task.ScheduleRun(nil)
	return nil
}

func (self *SServerSku) ValidateDeleteCondition(ctx context.Context) error {
	serverCount, err := skuRelatedGuestCount(self)
	if err != nil {
		return httperrors.NewInternalServerError("check instance")
	}
	if serverCount > 0 {
		return httperrors.NewNotEmptyError("now allow to delete inuse instance_type.please remove related servers first: %s", self.Name)
	}

	if !inWhiteList(self.Provider) {
		return httperrors.NewForbiddenError("not allow to delete public cloud instance_type: %s", self.Name)
	}
	/*count := GuestManager.Query().Equals("instance_type", self.Id).Count()
	if count > 0 {
		return httperrors.NewNotEmptyError("instance_type used by servers")
	}*/
	return nil
}

func (self *SServerSku) GetZoneExternalId() (string, error) {
	zoneObj, err := ZoneManager.FetchById(self.ZoneId)
	if err != nil {
		return "", err
	}

	zone := zoneObj.(*SZone)
	return zone.GetExternalId(), nil
}

func (manager *SServerSkuManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	publicCloud := false

	cloudEnvStr, _ := query.GetString("cloud_env")
	if cloudEnvStr == api.CLOUD_ENV_PUBLIC_CLOUD || jsonutils.QueryBoolean(query, "public_cloud", false) || jsonutils.QueryBoolean(query, "is_public", false) {
		publicCloud = true
		q = q.Filter(sqlchemy.In(q.Field("provider"), CloudproviderManager.GetPublicProviderProvidersQuery()))
	}
	if cloudEnvStr == api.CLOUD_ENV_PRIVATE_CLOUD || jsonutils.QueryBoolean(query, "private_cloud", false) || jsonutils.QueryBoolean(query, "is_private", false) {
		q = q.Filter(sqlchemy.In(q.Field("provider"), CloudproviderManager.GetPrivateProviderProvidersQuery()))
	}
	if jsonutils.QueryBoolean(query, "is_on_premise", false) {
		q = q.Filter(
			sqlchemy.OR(
				sqlchemy.Equals(q.Field("provider"), api.CLOUD_PROVIDER_ONECLOUD),
				sqlchemy.In(q.Field("provider"), CloudproviderManager.GetOnPremiseProviderProvidersQuery()),
			),
		)
	}
	if cloudEnvStr == api.CLOUD_ENV_PRIVATE_ON_PREMISE {
		q = q.Filter(sqlchemy.OR(
			sqlchemy.In(q.Field("provider"), CloudproviderManager.GetPrivateProviderProvidersQuery()), //私有云
			sqlchemy.Equals(q.Field("provider"), api.CLOUD_PROVIDER_ONECLOUD),                         //本地IDC
		),
		)
	}

	data := query.(*jsonutils.JSONDict)
	providers := jsonutils.GetQueryStringArray(query, "provider")
	if len(providers) > 0 {
		q = q.Filter(sqlchemy.In(q.Field("provider"), providers))
		if len(providers) == 1 && utils.IsInStringArray(providers[0], cloudprovider.GetPublicProviders()) {
			publicCloud = true
		}
		data.Remove("provider")
	}

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
		q = usableFilter(q, publicCloud)
		q = q.IsTrue("enabled")
	}

	if data.Contains("zone") {
		zoneStr, _ := data.GetString("zone")
		_zone, err := ZoneManager.FetchByIdOrName(userCred, zoneStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("zone", zoneStr)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		zone := _zone.(*SZone)
		region := zone.GetRegion()
		if region == nil {
			return nil, httperrors.NewResourceNotFoundError("failed to find cloudregion for zone %s(%s)", zone.Name, zone.Id)
		}
		//OneCloud忽略zone参数
		if region.Provider == api.CLOUD_PROVIDER_ONECLOUD {
			q = q.Equals("cloudregion_id", region.Id)
		} else {
			q = q.Equals("zone_id", zone.Id)
		}
	} else {
		q, err = validators.ApplyModelFilters(q, data, []*validators.ModelFilterOptions{
			{Key: "cloudregion", ModelKeyword: "cloudregion", OwnerId: userCred},
		})
		if err != nil {
			return nil, err
		}
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

func (manager *SServerSkuManager) GetMatchedSku(regionId string, cpu int64, memMB int64) (*SServerSku, error) {
	ret := &SServerSku{}

	q := manager.Query()
	q = q.Equals("cpu_core_count", cpu).Equals("memory_size_mb", memMB).Equals("cloudregion_id", regionId).Equals("postpaid_status", api.SkuStatusAvailable)
	err := q.First(ret)
	if err != nil {
		return nil, errors.Wrap(err, "ServerSkuManager.GetMatchedSku")
	}

	return ret, nil
}

func (manager *SServerSkuManager) FetchSkuByNameAndProvider(name string, provider string, checkConsistency bool) (*SServerSku, error) {
	q := manager.Query()
	q = q.Equals("name", name)
	if utils.IsInStringArray(provider, []string{api.CLOUD_PROVIDER_ONECLOUD, api.CLOUD_PROVIDER_VMWARE}) {
		q = q.Filter(
			sqlchemy.Equals(q.Field("provider"), api.CLOUD_PROVIDER_ONECLOUD),
		)
	} else if utils.IsInStringArray(provider, api.PRIVATE_CLOUD_PROVIDERS) {
		q = q.Filter(sqlchemy.OR(
			sqlchemy.Equals(q.Field("provider"), api.CLOUD_PROVIDER_ONECLOUD),
			sqlchemy.Equals(q.Field("provider"), provider),
		))
	} else {
		q = q.Equals("provider", provider)
	}

	skus := make([]SServerSku, 0)
	err := db.FetchModelObjects(manager, q, &skus)
	if err != nil {
		log.Errorf("fetch sku fail %s", err)
		return nil, err
	}
	if len(skus) == 0 {
		log.Errorf("no sku found for %s %s", name, provider)
		return nil, httperrors.NewResourceNotFoundError2(manager.Keyword(), name)
	}
	if len(skus) == 1 {
		return &skus[0], nil
	}
	if checkConsistency {
		for i := 1; i < len(skus); i += 1 {
			if skus[i].CpuCoreCount != skus[0].CpuCoreCount || skus[i].MemorySizeMB != skus[0].MemorySizeMB {
				log.Errorf("inconsistent sku %s %s", jsonutils.Marshal(&skus[0]), jsonutils.Marshal(&skus[i]))
				return nil, httperrors.NewDuplicateResourceError("duplicate instanceType %s", name)
			}
		}
	}
	return &skus[0], nil
}

func (manager *SServerSkuManager) GetPublicCloudSkuCount() (int, error) {
	q := manager.Query()
	q = q.Filter(sqlchemy.In(q.Field("provider"), cloudprovider.GetPublicProviders()))
	return q.CountWithError()
}

func (manager *SServerSkuManager) GetSkuCountByRegion(regionId string) (int, error) {
	q := manager.Query()
	if len(regionId) == 0 {
		q = q.IsNotEmpty("cloudregion_id")
	} else {
		q = q.Equals("cloudregion_id", regionId)
	}

	return q.CountWithError()
}

func (manager *SServerSkuManager) GetSkuCountByZone(zoneId string) []SServerSku {
	skus := []SServerSku{}
	q := manager.Query().Equals("zone_id", zoneId)
	if err := db.FetchModelObjects(manager, q, &skus); err != nil {
		log.Errorf("failed to get skus by zoneId %s error: %v", zoneId, err)
	}
	return skus
}

func (manager *SServerSkuManager) GetOneCloudSkus() ([]string, error) {
	skus := []SServerSku{}
	q := manager.Query().Equals("provider", api.CLOUD_PROVIDER_ONECLOUD)
	err := db.FetchModelObjects(manager, q, &skus)
	if err != nil {
		return nil, err
	}
	result := []string{}
	for _, sku := range skus {
		result = append(result, fmt.Sprintf("%d/%d", sku.CpuCoreCount, sku.MemorySizeMB))
	}
	return result, nil
}

func (manager *SServerSkuManager) GetSkus(provider string, cpu, memMB int) ([]SServerSku, error) {
	skus := []SServerSku{}
	q := manager.Query()
	if provider == api.CLOUD_PROVIDER_ONECLOUD {
		providerFilter := sqlchemy.OR(sqlchemy.Equals(q.Field("provider"), provider), sqlchemy.IsNullOrEmpty(q.Field("provider")))
		q = q.Equals("cpu_core_count", cpu).Equals("memory_size_mb", memMB).Filter(providerFilter)
	} else {
		q = q.Equals("cpu_core_count", cpu).Equals("memory_size_mb", memMB).Equals("provider", provider)
	}

	if err := db.FetchModelObjects(manager, q, &skus); err != nil {
		log.Errorf("failed to get skus with provider %s cpu %d mem %d error: %v", provider, cpu, memMB, err)
		return nil, err
	}

	return skus, nil
}

// 删除表中zone not found的记录
func (manager *SServerSkuManager) PendingDeleteInvalidSku() error {
	sq := ZoneManager.Query("id").Distinct().SubQuery()
	skus := make([]SServerSku, 0)
	q := manager.Query()
	q = q.NotIn("zone_id", sq).IsNotEmpty("zone_id")
	err := db.FetchModelObjects(manager, q, &skus)
	if err != nil {
		log.Errorln(err)
		return httperrors.NewInternalServerError("query sku list failed.")
	}

	for i := range skus {
		sku := skus[i]
		_, err = db.Update(&sku, func() error {
			return sku.MarkDelete()
		})

		if err != nil {
			log.Errorln(err)
			return httperrors.NewInternalServerError("delete sku %s failed.", sku.Id)
		}
	}

	return nil
}

func (self *SServerSku) AllowPerformCacheSku(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "cache-sku")
}

func (self *SServerSku) PerformCacheSku(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	cloudregionV := validators.NewModelIdOrNameValidator("cloudregion", "cloudregion", nil)
	err := cloudregionV.Validate(data.(*jsonutils.JSONDict))
	if err != nil {
		return nil, err
	}
	cloudregion := cloudregionV.Model.(*SCloudregion)
	if len(cloudregion.ManagerId) == 0 {
		return nil, httperrors.NewInputParameterError("Only support cache sku for private cloud")
	}

	cloudprovider := cloudregion.GetCloudprovider()
	if cloudprovider == nil {
		return nil, httperrors.NewInputParameterError("failed to get cloudprovider for region %s(%s)", cloudregion.Name, cloudregion.Id)
	}

	if !cloudprovider.Enabled {
		return nil, httperrors.NewInputParameterError("cloudprovider %s(%s) disabled", cloudprovider.Name, cloudprovider.Id)
	}

	return nil, self.StartServerSkuCacheTask(ctx, userCred, cloudregion.Id, "")
}

func (self *SServerSku) StartServerSkuCacheTask(ctx context.Context, userCred mcclient.TokenCredential, cloudregionId string, parentTaskId string) error {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(cloudregionId), "cloudregion_id")
	task, err := taskman.TaskManager.NewTask(ctx, "ServerSkuCacheTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (manager *SServerSkuManager) SyncPrivateCloudSkus(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, skus []cloudprovider.ICloudSku) compare.SyncResult {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	syncResult := compare.SyncResult{}
	localskus, err := manager.GetOneCloudSkus()
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	for _, sku := range skus {
		if !utils.IsInStringArray(fmt.Sprintf("%d/%d", sku.GetCpuCoreCount(), sku.GetMemorySizeMB()), localskus) {
			err := manager.newFromCloudSku(ctx, userCred, sku)
			if err != nil {
				syncResult.AddError(err)
			} else {
				syncResult.Add()
			}
		}
	}

	return syncResult
}

func (self *SServerSku) constructSku(extSku cloudprovider.ICloudSku) {
	self.InstanceTypeFamily = extSku.GetInstanceTypeFamily()
	self.InstanceTypeCategory = extSku.GetInstanceTypeCategory()

	self.PrepaidStatus = extSku.GetPrepaidStatus()
	self.PostpaidStatus = extSku.GetPostpaidStatus()

	self.CpuCoreCount = extSku.GetCpuCoreCount()
	self.MemorySizeMB = extSku.GetMemorySizeMB()

	self.OsName = extSku.GetOsName()

	self.SysDiskResizable = tristate.NewFromBool(extSku.GetSysDiskResizable())
	self.SysDiskType = extSku.GetSysDiskType()
	self.SysDiskMinSizeGB = extSku.GetSysDiskMinSizeGB()
	self.SysDiskMaxSizeGB = extSku.GetSysDiskMaxSizeGB()

	self.AttachedDiskType = extSku.GetAttachedDiskType()
	self.AttachedDiskSizeGB = extSku.GetAttachedDiskSizeGB()
	self.AttachedDiskCount = extSku.GetAttachedDiskCount()

	self.DataDiskTypes = extSku.GetDataDiskTypes()
	self.DataDiskMaxCount = extSku.GetDataDiskMaxCount()

	self.NicType = extSku.GetNicType()
	self.NicMaxCount = extSku.GetNicMaxCount()

	self.GpuAttachable = tristate.NewFromBool(extSku.GetGpuAttachable())
	self.GpuSpec = extSku.GetGpuSpec()
	self.GpuCount = extSku.GetGpuCount()
	self.GpuMaxCount = extSku.GetGpuMaxCount()
	self.Name = extSku.GetName()
}

func (self *SServerSku) setPrepaidPostpaidStatus(userCred mcclient.TokenCredential, prepaidStatus, postpaidStatus string) error {
	if prepaidStatus != self.PrepaidStatus || postpaidStatus != self.PostpaidStatus {
		diff, err := db.Update(self, func() error {
			self.PrepaidStatus = prepaidStatus
			self.PostpaidStatus = postpaidStatus
			return nil
		})
		if err != nil {
			return err
		}
		db.OpsLog.LogEvent(self, db.ACT_UPDATE, diff, userCred)
	}
	return nil
}

func (manager *SServerSkuManager) newFromCloudSku(ctx context.Context, userCred mcclient.TokenCredential, extSku cloudprovider.ICloudSku) error {
	sku := &SServerSku{Provider: api.CLOUD_PROVIDER_ONECLOUD}
	sku.SetModelManager(manager, sku)
	// 第一次同步新建的套餐是启用状态
	sku.Enabled = true
	sku.Status = api.SkuStatusReady
	sku.constructSku(extSku)

	sku.Name = extSku.GetName()
	sku.CloudregionId = api.DEFAULT_REGION_ID
	sku.SetModelManager(manager, sku)
	err := manager.TableSpec().Insert(sku)
	if err != nil {
		return errors.Wrapf(err, "newFromCloudSku.Insert")
	}
	db.OpsLog.LogEvent(sku, db.ACT_CREATE, sku.GetShortDesc(ctx), userCred)

	return nil
}

func (manager *SServerSkuManager) newPublicCloudSku(ctx context.Context, userCred mcclient.TokenCredential, extSku SServerSku) error {
	extSku.Enabled = true
	extSku.Status = api.SkuStatusReady
	return manager.TableSpec().Insert(&extSku)
}

func (self *SServerSku) AllowPerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAllowPerform(rbacutils.ScopeSystem, userCred, self, "enable")
}

func (self *SServerSku) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !self.Enabled {
		_, err := db.Update(self, func() error {
			self.Enabled = true
			return nil
		})
		if err != nil {
			return nil, errors.Wrap(err, "PerformEnable.db.Update()")
		}
		db.OpsLog.LogEvent(self, db.ACT_ENABLE, "", userCred)
		logclient.AddSimpleActionLog(self, logclient.ACT_ENABLE, nil, userCred, true)
	}
	return nil, nil
}

func (self *SServerSku) AllowPerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAllowPerform(rbacutils.ScopeSystem, userCred, self, "disable")
}

func (self *SServerSku) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Enabled {
		_, err := db.Update(self, func() error {
			self.Enabled = false
			return nil
		})
		if err != nil {
			return nil, errors.Wrap(err, "PerformEnable.db.Update()")
		}
		db.OpsLog.LogEvent(self, db.ACT_DISABLE, "", userCred)
		logclient.AddSimpleActionLog(self, logclient.ACT_DISABLE, nil, userCred, true)
	}
	return nil, nil
}

func (self *SServerSku) syncWithCloudSku(ctx context.Context, userCred mcclient.TokenCredential, extSku SServerSku) error {
	_, err := db.Update(self, func() error {
		self.ZoneId = extSku.ZoneId
		self.PrepaidStatus = extSku.PrepaidStatus
		self.PostpaidStatus = extSku.PostpaidStatus
		return nil
	})
	return err
}

func (self *SServerSku) MarkAsSoldout(ctx context.Context) error {
	_, err := db.UpdateWithLock(ctx, self, func() error {
		self.PrepaidStatus = api.SkuStatusSoldout
		self.PostpaidStatus = api.SkuStatusSoldout
		return nil
	})

	return errors.Wrap(err, "SServerSku.MarkAsSoldout")
}

func (manager *SServerSkuManager) FetchSkusByRegion(regionID string) ([]SServerSku, error) {
	q := manager.Query()
	q = q.Equals("cloudregion_id", regionID)

	skus := make([]SServerSku, 0)
	err := db.FetchModelObjects(manager, q, &skus)
	if err != nil {
		return nil, errors.Wrap(err, "SServerSkuManager.FetchSkusByRegion")
	}

	return skus, nil
}

func (manager *SServerSkuManager) SyncServerSkus(ctx context.Context, userCred mcclient.TokenCredential, region *SCloudregion, extSkuMeta *SSkuResourcesMeta) compare.SyncResult {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	syncResult := compare.SyncResult{}

	extSkus, err := extSkuMeta.GetServerSkus(region)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	dbSkus, err := manager.FetchSkusByRegion(region.GetId())
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	removed := make([]SServerSku, 0)
	commondb := make([]SServerSku, 0)
	commonext := make([]SServerSku, 0)
	added := make([]SServerSku, 0)

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
		err = manager.newPublicCloudSku(ctx, userCred, added[i])
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncResult.Add()
		}
	}
	return syncResult
}

// sku标记为soldout状态。
func (manager *SServerSkuManager) MarkAsSoldout(id string) error {
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
func (manager *SServerSkuManager) MarkAllAsSoldout(ids []string) error {
	var err error
	for _, id := range ids {
		err = manager.MarkAsSoldout(id)
		if err != nil {
			return err
		}
	}

	return nil
}

// 获取同一个zone下所有Available状态的sku id
func (manager *SServerSkuManager) FetchAllAvailableSkuIdByZoneId(zoneId string) ([]string, error) {
	q := manager.Query()
	if len(zoneId) == 0 {
		return nil, fmt.Errorf("FetchAllAvailableSkuIdByZoneId zone id should not be emtpy")
	}

	skus := make([]SServerSku, 0)
	q = q.Equals("zone_id", zoneId)
	q = q.Filter(sqlchemy.OR(
		sqlchemy.Equals(q.Field("prepaid_status"), api.SkuStatusAvailable),
		sqlchemy.Equals(q.Field("postpaid_status"), api.SkuStatusAvailable)))

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

func (manager *SServerSkuManager) initializeSkuProvider() error {
	skus := []SServerSku{}
	q := manager.Query()
	q = q.Filter(sqlchemy.OR(
		sqlchemy.In(q.Field("provider"), CloudproviderManager.GetPrivateProviderProvidersQuery()),
		sqlchemy.IsNullOrEmpty(q.Field("provider")),
	))
	err := db.FetchModelObjects(manager, q, &skus)
	if err != nil {
		return errors.Wrapf(err, "initializeSkuProvider.FetchModelObjects")
	}
	for _, sku := range skus {
		_, err = db.Update(&sku, func() error {
			sku.Provider = api.CLOUD_PROVIDER_ONECLOUD
			sku.CloudregionId = api.DEFAULT_REGION_ID
			sku.ZoneId = ""
			return nil
		})
		if err != nil {
			return errors.Wrapf(err, "sku.Update")
		}
	}
	return nil
}

func (manager *SServerSkuManager) initializeSkuStatus() error {
	skus := []SServerSku{}
	q := manager.Query().NotEquals("status", api.SkuStatusReady)
	err := db.FetchModelObjects(manager, q, &skus)
	if err != nil {
		return errors.Wrapf(err, "initializeSkuStatus.FetchModelObjects")
	}
	for _, sku := range skus {
		_, err = db.Update(&sku, func() error {
			sku.Status = api.SkuStatusReady
			return nil
		})
		if err != nil {
			return errors.Wrapf(err, "sku.Update")
		}
	}
	return nil
}

func (manager *SServerSkuManager) initializeSkuEnableField() error {
	skus := []SServerSku{}
	q := manager.Query().IsNull("enabled")
	err := db.FetchModelObjects(manager, q, &skus)
	if err != nil {
		return errors.Wrapf(err, "FetchModelObjects")
	}
	for _, sku := range skus {
		_, err = db.Update(&sku, func() error {
			sku.Status = api.SkuStatusReady
			sku.Enabled = true
			return nil
		})
		if err != nil {
			return errors.Wrapf(err, "sku.Update")
		}
	}
	return nil
}

func (manager *SServerSkuManager) InitializeData() error {
	count, err := manager.Query().Equals("cloudregion_id", api.DEFAULT_REGION_ID).IsNullOrEmpty("zone_id").CountWithError()
	if err == nil {
		if count == 0 {
			type Item struct {
				CPU   int
				MemMB int
			}

			items := []Item{
				{1, 1 * 1024},
				{1, 2 * 1024},
				{1, 4 * 1024},
				{1, 8 * 1024},
				{2, 2 * 1024},
				{2, 4 * 1024},
				{2, 8 * 1024},
				{2, 12 * 1024},
				{2, 16 * 1024},
				{4, 4 * 1024},
				{4, 8 * 1024},
				{4, 12 * 1024},
				{4, 16 * 1024},
				{4, 24 * 1024},
				{4, 32 * 1024},
				{8, 8 * 1024},
				{8, 12 * 1024},
				{8, 16 * 1024},
				{8, 24 * 1024},
				{8, 32 * 1024},
				{8, 64 * 1024},
				{12, 12 * 1024},
				{12, 16 * 1024},
				{12, 24 * 1024},
				{12, 32 * 1024},
				{12, 64 * 1024},
				{16, 16 * 1024},
				{16, 24 * 1024},
				{16, 32 * 1024},
				{16, 48 * 1024},
				{16, 64 * 1024},
				{24, 24 * 1024},
				{24, 32 * 1024},
				{24, 48 * 1024},
				{24, 64 * 1024},
				{24, 128 * 1024},
				{32, 32 * 1024},
				{32, 48 * 1024},
				{32, 64 * 1024},
				{32, 128 * 1024},
			}

			for i := range items {
				item := items[i]
				sku := &SServerSku{}
				sku.CloudregionId = api.DEFAULT_REGION_ID
				sku.CpuCoreCount = item.CPU
				sku.MemorySizeMB = item.MemMB
				sku.IsEmulated = false
				sku.Enabled = true
				sku.InstanceTypeCategory = api.SkuCategoryGeneralPurpose
				sku.LocalCategory = api.SkuCategoryGeneralPurpose
				sku.InstanceTypeFamily = api.InstanceFamilies[api.SkuCategoryGeneralPurpose]
				name, _ := genInstanceType(sku.InstanceTypeFamily, int64(item.CPU), int64(item.MemMB))
				sku.Name = name
				sku.PrepaidStatus = api.SkuStatusAvailable
				sku.PostpaidStatus = api.SkuStatusAvailable
				err := manager.TableSpec().Insert(sku)
				if err != nil {
					log.Errorf("ServerSkuManager Initialize local sku %s", err)
				}
			}
		}
	} else {
		log.Errorf("ServerSkuManager InitializeData %s", err)
	}

	privateSkus := make([]SServerSku, 0)
	err = manager.Query().IsNullOrEmpty("local_category").IsNullOrEmpty("zone_id").All(&privateSkus)
	if err != nil {
		return err
	}

	for _, sku := range privateSkus {
		_, err = manager.TableSpec().Update(&sku, func() error {
			sku.LocalCategory = sku.InstanceTypeCategory
			return nil
		})
		if err != nil {
			return err
		}
	}

	err = manager.initializeSkuEnableField()
	if err != nil {
		return errors.Wrap(err, "initializeSkuEnableField")
	}

	err = manager.initializeSkuProvider()
	if err != nil {
		return errors.Wrap(err, "InitializeProvider")
	}

	return manager.initializeSkuStatus()
}
