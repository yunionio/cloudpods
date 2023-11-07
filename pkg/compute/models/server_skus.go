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
	"sync"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
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
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/scheduler"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/onecloud/pkg/util/yunionmeta"
)

type SServerSkuManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
	SCloudregionResourceBaseManager
	SZoneResourceBaseManager
}

var ServerSkuManager *SServerSkuManager

func init() {
	ServerSkuManager = &SServerSkuManager{
		SEnabledStatusStandaloneResourceBaseManager: db.NewEnabledStatusStandaloneResourceBaseManager(
			SServerSku{},
			"serverskus_tbl",
			"serversku",
			"serverskus",
		),
	}
	ServerSkuManager.NameRequireAscii = false
	ServerSkuManager.SetVirtualObject(ServerSkuManager)
	// CREATE INDEX sku_index  ON serverskus_tbl (`deleted`, `is_emulated`, `provider`, `cloudregion_id`, `postpaid_status`, `prepaid_status`)
	ServerSkuManager.TableSpec().AddIndex(false, "deleted", "is_emulated", "provider", "cloudregion_id", "postpaid_status", "prepaid_status")
}

// SServerSku 实际对应的是instance type清单. 这里的Sku实际指的是instance type。
type SServerSku struct {
	db.SEnabledStatusStandaloneResourceBase
	db.SExternalizedResourceBase
	SCloudregionResourceBase
	SZoneResourceBase

	// SkuId       string `width:"64" charset:"ascii" nullable:"false" list:"user" create:"admin_required"`                 // x2.large
	InstanceTypeFamily   string `width:"32" charset:"ascii" nullable:"true" list:"user" create:"admin_optional" update:"admin"`           // x2
	InstanceTypeCategory string `width:"32" charset:"utf8" nullable:"true" list:"user" create:"admin_optional" update:"admin"`            // 通用型
	LocalCategory        string `width:"32" charset:"utf8" nullable:"true" list:"user" create:"admin_optional" update:"admin" default:""` // 记录本地分类

	PrepaidStatus  string `width:"32" charset:"utf8" nullable:"true" list:"user" update:"admin" create:"admin_optional" default:"available"` // 预付费资源状态   available|soldout
	PostpaidStatus string `width:"32" charset:"utf8" nullable:"true" list:"user" update:"admin" create:"admin_optional" default:"available"` // 按需付费资源状态  available|soldout

	CpuArch      string `width:"16" charset:"ascii" nullable:"true" list:"user" create:"admin_optional" update:"admin"` // CPU 架构 x86|xarm
	CpuCoreCount int    `nullable:"false" list:"user" create:"admin_required"`
	MemorySizeMB int    `nullable:"false" list:"user" create:"admin_required"`

	OsName string `width:"32" charset:"ascii" nullable:"true" list:"user" create:"admin_optional" update:"admin" default:"Any"` // Windows|Linux|Any

	SysDiskResizable tristate.TriState `default:"true" list:"user" create:"admin_optional" update:"admin"`
	SysDiskType      string            `width:"128" charset:"ascii" nullable:"true" list:"user" create:"admin_optional" update:"admin"`
	SysDiskMinSizeGB int               `nullable:"true" list:"user" create:"admin_optional" update:"admin"` // not required。 windows比较新的版本都是50G左右。
	SysDiskMaxSizeGB int               `nullable:"true" list:"user" create:"admin_optional" update:"admin"` // not required

	AttachedDiskType   string `nullable:"true" list:"user" create:"admin_optional" update:"admin"`
	AttachedDiskSizeGB int    `nullable:"true" list:"user" create:"admin_optional" update:"admin"`
	AttachedDiskCount  int    `nullable:"true" list:"user" create:"admin_optional" update:"admin"`

	DataDiskTypes    string `width:"128" charset:"ascii" nullable:"true" list:"user" create:"admin_optional" update:"admin"`
	DataDiskMaxCount int    `nullable:"true" list:"user" create:"admin_optional" update:"admin"`

	NicType     string `nullable:"true" list:"user" create:"admin_optional" update:"admin"`
	NicMaxCount int    `default:"1" nullable:"true" list:"user" create:"admin_optional" update:"admin"`

	GpuAttachable tristate.TriState `default:"true" list:"user" create:"admin_optional" update:"admin"`
	GpuSpec       string            `width:"128" charset:"ascii" nullable:"true" list:"user" create:"admin_optional" update:"admin"`
	GpuCount      string            `nullable:"true" list:"user" create:"admin_optional" update:"admin"`
	GpuMaxCount   int               `nullable:"true" list:"user" create:"admin_optional" update:"admin"`

	Provider string `width:"64" charset:"ascii" nullable:"true" list:"user" default:"OneCloud" create:"admin_optional"`

	Md5 string `width:"32" charset:"utf8" nullable:"true"`
}

func (manager *SServerSkuManager) FetchUniqValues(ctx context.Context, data jsonutils.JSONObject) jsonutils.JSONObject {
	regionId, _ := data.GetString("cloudregion_id")
	return jsonutils.Marshal(map[string]string{"cloudregion_id": regionId})
}

func (manager *SServerSkuManager) FilterByUniqValues(q *sqlchemy.SQuery, values jsonutils.JSONObject) *sqlchemy.SQuery {
	regionId, _ := values.GetString("cloudregion_id")
	if len(regionId) > 0 {
		q = q.Equals("cloudregion_id", regionId)
	}
	return q
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

func genInstanceType(family string, cpu, memMb int64) (string, error) {
	if cpu <= 0 {
		return "", fmt.Errorf("cpu_core_count should great than zero")
	}

	if memMb <= 0 {
		return "", fmt.Errorf("memory_size_mb should great than zero")
	}

	if memMb%1024 != 0 && memMb != 512 {
		return "", fmt.Errorf("memory_size_mb should be 512 or integral multiple of 1024")
	}

	switch memMb {
	case 512:
		return fmt.Sprintf("ecs.%s.c%dm1.nano", family, cpu), nil
	default:
		return fmt.Sprintf("ecs.%s.c%dm%d", family, cpu, memMb/1024), nil
	}
}

func (self SServerSku) GetGlobalId() string {
	return self.ExternalId
}

func (manager *SServerSkuManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ServerSkuDetails {
	rows := make([]api.ServerSkuDetails, len(objs))

	stdRows := manager.SEnabledStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	zoneRows := manager.SZoneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	instanceTypes := []string{}
	for i := range rows {
		rows[i] = api.ServerSkuDetails{
			EnabledStatusStandaloneResourceDetails: stdRows[i],
			ZoneResourceInfoBase:                   zoneRows[i].ZoneResourceInfoBase,
			CloudregionResourceInfo:                regRows[i],
		}
		sku := objs[i].(*SServerSku)
		if !utils.IsInStringArray(sku.Name, instanceTypes) {
			instanceTypes = append(instanceTypes, sku.Name)
		}

		rows[i].CloudEnv = strings.Split(zoneRows[i].RegionExternalId, "/")[0]
	}

	ret := []struct {
		InstanceType  string
		CloudregionId string
		ZoneId        string
	}{}

	guestsQ := GuestManager.Query().SubQuery()
	hostsQ := HostManager.Query().SubQuery()
	zonesQ := ZoneManager.Query().SubQuery()
	q := guestsQ.Query(
		guestsQ.Field("instance_type"),
		hostsQ.Field("zone_id"),
		zonesQ.Field("cloudregion_id"),
	).
		Join(hostsQ, sqlchemy.Equals(guestsQ.Field("host_id"), hostsQ.Field("id"))).
		Join(zonesQ, sqlchemy.Equals(hostsQ.Field("zone_id"), zonesQ.Field("id"))).
		Filter(sqlchemy.In(guestsQ.Field("instance_type"), instanceTypes))
	err := q.All(&ret)
	if err != nil {
		log.Errorf("query instance cnt error: %v", err)
		return rows
	}
	skuMap := map[string]map[string]int{}
	for _, sku := range ret {
		_, ok := skuMap[sku.InstanceType]
		if !ok {
			skuMap[sku.InstanceType] = map[string]int{}
		}
		_, ok = skuMap[sku.InstanceType][sku.ZoneId]
		if !ok {
			skuMap[sku.InstanceType][sku.ZoneId] = 0
		}
		skuMap[sku.InstanceType][sku.ZoneId] += 1
		_, ok = skuMap[sku.InstanceType][sku.CloudregionId]
		if !ok {
			skuMap[sku.InstanceType][sku.CloudregionId] = 0
		}
		skuMap[sku.InstanceType][sku.CloudregionId] += 1
	}
	for i := range rows {
		sku := objs[i].(*SServerSku)
		if len(sku.ZoneId) > 0 {
			rows[i].TotalGuestCount = skuMap[sku.Name][sku.ZoneId]
		} else {
			rows[i].TotalGuestCount = skuMap[sku.Name][sku.CloudregionId]
		}
	}

	return rows
}

func (self *SServerSkuManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.ServerSkuCreateInput) (api.ServerSkuCreateInput, error) {
	var region *SCloudregion
	if len(input.CloudregionId) > 0 {
		_region, err := validators.ValidateModel(userCred, CloudregionManager, &input.CloudregionId)
		if err != nil {
			return input, err
		}
		region = _region.(*SCloudregion)
	}

	if len(input.ZoneId) > 0 {
		_zone, err := validators.ValidateModel(userCred, ZoneManager, &input.ZoneId)
		if err != nil {
			return input, err
		}
		zone := _zone.(*SZone)
		if len(input.CloudregionId) == 0 {
			input.CloudregionId = zone.CloudregionId
		}
		if input.CloudregionId != zone.CloudregionId {
			return input, httperrors.NewConflictError("zone %s not in cloudregion %s", zone.Name, input.CloudregionId)
		}
		region, _ = zone.GetRegion()
	}

	if input.CpuCoreCount < 1 || input.CpuCoreCount > options.Options.SkuMaxCpuCount {
		return input, httperrors.NewOutOfRangeError("cpu_core_count should be range of 1~%d", options.Options.SkuMaxCpuCount)
	}

	if input.MemorySizeMB < 512 || input.MemorySizeMB > 1024*options.Options.SkuMaxMemSize {
		return input, httperrors.NewOutOfRangeError("memory_size_mb, shoud be range of 512~%d", 1024*options.Options.SkuMaxMemSize)
	}

	if len(input.InstanceTypeCategory) == 0 {
		input.InstanceTypeCategory = api.SkuCategoryGeneralPurpose
	}

	if !utils.IsInStringArray(input.InstanceTypeCategory, api.SKU_FAMILIES) {
		return input, httperrors.NewInputParameterError("instance_type_category shoud be one of %s", api.SKU_FAMILIES)
	}

	if input.Enabled == nil {
		enabled := true
		input.Enabled = &enabled
	}

	input.Provider = api.CLOUD_PROVIDER_ONECLOUD
	input.Status = api.SkuStatusReady
	if region != nil {
		input.Provider = region.Provider
	}
	if input.Provider == api.CLOUD_PROVIDER_ONECLOUD {
	} else if utils.IsInStringArray(input.Provider, api.PRIVATE_CLOUD_PROVIDERS) {
		input.Status = api.SkuStatusCreating
	} else {
		return input, httperrors.NewUnsupportOperationError("Not support create public cloud sku")
	}

	input.LocalCategory = input.InstanceTypeCategory
	input.InstanceTypeFamily = api.InstanceFamilies[input.InstanceTypeCategory]

	var err error
	if len(input.Name) == 0 {
		// 格式 ecs.g1.c1m1
		input.Name, err = genInstanceType(input.InstanceTypeFamily, input.CpuCoreCount, input.MemorySizeMB)
		if err != nil {
			return input, httperrors.NewInputParameterError("%v", err)
		}
		q := self.Query().Equals("name", input.Name)
		if len(input.CloudregionId) > 0 {
			q = q.Equals("cloudregion_id", input.CloudregionId)
		}
		count, err := q.CountWithError()
		if err != nil {
			return input, httperrors.NewInternalServerError("checkout server sku name duplicate error: %v", err)
		}
		if count > 0 {
			return input, httperrors.NewDuplicateResourceError("Duplicate sku %s", input.Name)
		}
	}

	input.EnabledStatusStandaloneResourceCreateInput, err = self.SEnabledStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.EnabledStatusStandaloneResourceCreateInput)
	if err != nil {
		return input, err
	}
	return input, nil
}

func (self *SServerSku) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SEnabledStatusStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	if self.Provider != api.CLOUD_PROVIDER_ONECLOUD {
		self.StartSkuCreateTask(ctx, userCred)
	}
}

func (self *SServerSku) StartSkuCreateTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ServerSkuCreateTask", self, userCred, nil, "", "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	task.ScheduleRun(nil)
	return nil
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

func (self *SServerSkuManager) ClearSchedDescCache(wait bool) error {
	s := auth.GetAdminSession(context.Background(), options.Options.Region)
	_, err := scheduler.SchedManager.SyncSku(s, true)
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
	providers := usableCloudProviders()
	networks := NetworkManager.Query("wire_id").Equals("status", api.NETWORK_STATUS_AVAILABLE)
	wires := WireManager.Query("vpc_id").In("id", networks)
	_vpcs := VpcManager.Query("cloudregion_id").
		Equals("status", api.VPC_STATUS_AVAILABLE).
		In("id", wires)
	filters := sqlchemy.OR(sqlchemy.In(_vpcs.Field("manager_id"), providers), sqlchemy.IsNullOrEmpty(_vpcs.Field("manager_id")))
	vpcs := _vpcs.Filter(filters).SubQuery()
	sq := vpcs.Query(sqlchemy.DISTINCT("cloudregion_id", vpcs.Field("cloudregion_id")))
	return []sqlchemy.ICondition{sqlchemy.In(f, sq.SubQuery())}
}

func usableFilter(q *sqlchemy.SQuery, public_cloud bool) (*sqlchemy.SQuery, error) {
	// 过滤出公有云provider状态健康的sku
	if public_cloud {
		providerTable := usableCloudProviders().SubQuery()
		providerRegionTable := CloudproviderRegionManager.Query().SubQuery()

		_subq := providerRegionTable.Query(sqlchemy.DISTINCT("cloudregion_id", providerRegionTable.Field("cloudregion_id")))
		subq := _subq.Join(providerTable, sqlchemy.Equals(providerRegionTable.Field("cloudprovider_id"), providerTable.Field("id"))).SubQuery()
		q.Join(subq, sqlchemy.Equals(q.Field("cloudregion_id"), subq.Field("cloudregion_id")))
	}

	// 过滤出network usable的sku
	if public_cloud {
		zoneIds, err := NetworkUsableZoneIds(true, true, nil)
		if err != nil {
			return nil, errors.Wrap(err, "NetworkUsableZoneIds")
		}
		q = q.Filter(sqlchemy.OR(sqlchemy.In(q.Field("zone_id"), zoneIds), sqlchemy.IsNullOrEmpty(q.Field("zone_id")))) //Azure的zone_id可能为空
	} else {
		// 本地IDC sku 只定义到region层级, zone id 为空.因此只能按region查询
		iconditions := networkUsableRegionQueries(q.Field("cloudregion_id"))
		// 私有云 sku region及zone为空
		iconditions = append(iconditions, sqlchemy.IsNullOrEmpty(q.Field("cloudregion_id")))
		q = q.Filter(sqlchemy.OR(iconditions...))
	}

	return q, nil
}

func (manager *SServerSkuManager) GetPropertyInstanceSpecs(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	listQuery := api.ServerSkuListInput{}
	err := query.Unmarshal(&listQuery)
	if err != nil {
		return nil, errors.Wrap(err, "query.Unmarshal")
	}
	q, err := manager.ListItemFilter(ctx, manager.Query(), userCred, listQuery)
	if err != nil {
		return nil, errors.Wrap(err, "manager.ListItemFilter")
	}

	skus := make([]SServerSku, 0)
	q = q.GroupBy(q.Field("cpu_core_count"), q.Field("memory_size_mb"))
	q = q.Asc(q.Field("cpu_core_count"), q.Field("memory_size_mb"))
	err = db.FetchModelObjects(manager, q, &skus)
	if err != nil {
		log.Infof("FetchModelObjects %s", err)
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

func (self *SServerSku) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ServerSkuUpdateInput) (api.ServerSkuUpdateInput, error) {
	if len(input.Name) > 0 {
		return input, httperrors.NewUnsupportOperationError("Cannot change server sku name")
	}

	var err error
	input.EnabledStatusStandaloneResourceBaseUpdateInput, err = self.SEnabledStatusStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, input.EnabledStatusStandaloneResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SEnabledStatusStandaloneResourceBase.ValidateUpdateData")
	}

	return input, nil
}

func (self *SServerSku) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	ServerSkuManager.ClearSchedDescCache(true)
	self.SEnabledStatusStandaloneResourceBase.PostUpdate(ctx, userCred, query, data)
}

func (self *SServerSku) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("SServerSku delete do nothing")
	return nil
}

func (self *SServerSku) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	ServerSkuManager.ClearSchedDescCache(true)
	return db.RealDeleteModel(ctx, userCred, self)
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

func (self *SServerSku) GetGuestCount() (int, error) {
	guestsQ := GuestManager.Query().SubQuery()
	hostsQ := HostManager.Query().SubQuery()
	zonesQ := ZoneManager.Query().SubQuery()
	q := guestsQ.Query().
		Join(hostsQ, sqlchemy.Equals(guestsQ.Field("host_id"), hostsQ.Field("id"))).
		Join(zonesQ, sqlchemy.Equals(hostsQ.Field("zone_id"), zonesQ.Field("id"))).
		Filter(sqlchemy.Equals(guestsQ.Field("instance_type"), self.Name))
	if len(self.ZoneId) > 0 {
		q = q.Filter(sqlchemy.Equals(hostsQ.Field("zone_id"), self.ZoneId))
	} else {
		q = q.Filter(sqlchemy.Equals(zonesQ.Field("cloudregion_id"), self.CloudregionId))
	}
	return q.CountWithError()
}

func (self *SServerSku) ValidateDeleteCondition(ctx context.Context, info *api.ServerSkuDetails) error {
	totalGuestCnt := 0
	if info != nil {
		totalGuestCnt = info.TotalGuestCount
	} else {
		totalGuestCnt, _ = self.GetGuestCount()
	}
	if totalGuestCnt > 0 {
		return httperrors.NewNotEmptyError("now allow to delete inuse instance_type.please remove related servers first: %s", self.Name)
	}

	if !inWhiteList(self.Provider) {
		return httperrors.NewForbiddenError("not allow to delete public cloud instance_type: %s", self.Name)
	}
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

func listItemDomainFilter(q *sqlchemy.SQuery, providers []string, domainId string) *sqlchemy.SQuery {
	// CLOUD_PROVIDER_ONECLOUD 没有对应的cloudaccount
	if len(domainId) > 0 {
		if len(providers) >= 1 && !utils.IsInStringArray(api.CLOUD_PROVIDER_ONECLOUD, providers) {
			// 明确指定只查询公有云provider的情况，只查询公有云skus
			q = q.In("provider", getDomainManagerProviderSubq(domainId))
		} else if len(providers) == 1 && utils.IsInStringArray(api.CLOUD_PROVIDER_ONECLOUD, providers) {
			// 明确指定只查询私有云provider的情况
		} else {
			// 公有云skus & 私有云skus 混合查询
			publicSkusQ := sqlchemy.In(q.Field("provider"), getDomainManagerProviderSubq(domainId))
			privateSkusQ := sqlchemy.Equals(q.Field("provider"), api.CLOUD_PROVIDER_ONECLOUD)
			q = q.Filter(sqlchemy.OR(publicSkusQ, privateSkusQ))
		}
	}
	return q
}

// 主机套餐规格列表
func (manager *SServerSkuManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ServerSkuListInput,
) (*sqlchemy.SQuery, error) {
	publicCloud := false

	cloudEnvStr := query.CloudEnv
	if cloudEnvStr == api.CLOUD_ENV_PUBLIC_CLOUD {
		publicCloud = true
		pq := CloudproviderManager.GetPublicProviderProvidersQuery()
		q = q.Join(pq, sqlchemy.Equals(q.Field("provider"), pq.Field("provider")))
	}
	if cloudEnvStr == api.CLOUD_ENV_PRIVATE_CLOUD {
		pq := CloudproviderManager.GetPrivateProviderProvidersQuery()
		q = q.Join(pq, sqlchemy.Equals(q.Field("provider"), pq.Field("provider")))
	}
	if cloudEnvStr == api.CLOUD_ENV_ON_PREMISE {
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

	providers := query.Providers
	if len(providers) > 0 {
		q = q.Filter(sqlchemy.In(q.Field("provider"), providers))
		if len(providers) == 1 && utils.IsInStringArray(providers[0], cloudprovider.GetPublicProviders()) {
			publicCloud = true
		}
	}

	conditions := []sqlchemy.ICondition{}
	for _, arch := range query.CpuArch {
		if len(arch) == 0 {
			continue
		}
		if arch == apis.OS_ARCH_X86 {
			conditions = append(conditions, sqlchemy.OR(
				sqlchemy.Startswith(q.Field("cpu_arch"), arch),
				sqlchemy.Equals(q.Field("cpu_arch"), apis.OS_ARCH_I386),
				sqlchemy.IsNullOrEmpty(q.Field("cpu_arch")),
			))
		} else if arch == apis.OS_ARCH_ARM {
			conditions = append(conditions, sqlchemy.OR(
				sqlchemy.Startswith(q.Field("cpu_arch"), arch),
				sqlchemy.Equals(q.Field("cpu_arch"), apis.OS_ARCH_AARCH32),
				sqlchemy.Equals(q.Field("cpu_arch"), apis.OS_ARCH_AARCH64),
				sqlchemy.IsNullOrEmpty(q.Field("cpu_arch")),
			))
		} else {
			conditions = append(conditions, sqlchemy.Startswith(q.Field("cpu_arch"), arch))
		}
	}
	if len(conditions) > 0 {
		q = q.Filter(sqlchemy.OR(conditions...))
	}

	if query.Distinct {
		q = q.GroupBy(q.Field("name"))
	}

	brands := query.Brands
	if len(brands) > 0 {
		q = q.Filter(sqlchemy.In(q.Field("brand"), brands))
	}

	q, err := manager.SEnabledStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledStatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusStandaloneResourceBaseManager.ListItemFilter")
	}

	if query.Usable != nil && *query.Usable {
		q, err := usableFilter(q, publicCloud)
		if err != nil {
			return nil, err
		}
		q = q.IsTrue("enabled")
	}

	zoneStr := query.ZoneId
	if len(zoneStr) > 0 {
		_zone, err := ZoneManager.FetchByIdOrName(userCred, zoneStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("zone", zoneStr)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		zone := _zone.(*SZone)
		region, _ := zone.GetRegion()
		if region == nil {
			return nil, httperrors.NewResourceNotFoundError("failed to find cloudregion for zone %s(%s)", zone.Name, zone.Id)
		}
		//OneCloud忽略zone参数
		if region.Provider == api.CLOUD_PROVIDER_ONECLOUD {
			q = q.Equals("cloudregion_id", region.Id)
		} else {
			q = q.Equals("zone_id", zone.Id)
		}
	}

	q, err = managedResourceFilterByRegion(q, query.RegionalFilterListInput, "", nil)
	if err != nil {
		return nil, errors.Wrap(err, "managedResourceFilterByRegion")
	}

	if len(query.PostpaidStatus) > 0 {
		q = q.Equals("postpaid_status", query.PostpaidStatus)
	}
	if len(query.PrepaidStatus) > 0 {
		q = q.Equals("prepaid_status", query.PrepaidStatus)
	}

	conditions = []sqlchemy.ICondition{}
	for _, sizeMb := range query.MemorySizeMb {
		// 按区间查询内存, 避免0.75G这样的套餐不好过滤
		if sizeMb > 0 {
			s, e := intervalMem(sizeMb)
			conditions = append(
				conditions,
				sqlchemy.AND(
					sqlchemy.GT(q.Field("memory_size_mb"), s),
					sqlchemy.LE(q.Field("memory_size_mb"), e),
				),
			)
		}
	}
	if len(conditions) > 0 {
		q = q.Filter(sqlchemy.OR(conditions...))
	}
	if len(query.CpuCoreCount) > 0 {
		q = q.In("cpu_core_count", query.CpuCoreCount)
	}

	return q, err
}

func (manager *SServerSkuManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ServerSkuListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SEnabledStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.EnabledStatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusStandaloneResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SCloudregionResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.OrderByExtraFields")
	}

	if db.NeedOrderQuery([]string{query.OrderByTotalGuestCount}) {
		guestQ := GuestManager.Query()
		guestQ = guestQ.AppendField(guestQ.Field("instance_type"), sqlchemy.COUNT("total_guest_count"))
		guestQ = guestQ.GroupBy(guestQ.Field("instance_type"))
		guestSQ := guestQ.SubQuery()
		q = q.Join(guestSQ, sqlchemy.Equals(guestSQ.Field("instance_type"), q.Field("name")))
		q = q.AppendField(q.QueryFields()...)
		q = q.AppendField(guestSQ.Field("total_guest_count"))
		q = db.OrderByFields(q, []string{query.OrderByTotalGuestCount}, []sqlchemy.IQueryField{q.Field("total_guest_count")})
	}
	return q, nil
}

func (manager *SServerSkuManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SEnabledStatusStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SCloudregionResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SServerSkuManager) GetMatchedSku(regionId string, cpu int64, memMB int64) (*SServerSku, error) {
	ret := &SServerSku{}

	_region, err := CloudregionManager.FetchById(regionId)
	if err != nil {
		return nil, errors.Wrapf(err, "CloudregionManager.FetchById(%s)", regionId)
	}
	region := _region.(*SCloudregion)
	if utils.IsInStringArray(region.Provider, api.PRIVATE_CLOUD_PROVIDERS) {
		regionId = api.DEFAULT_REGION_ID
	}

	q := manager.Query()
	q = q.Equals("cpu_core_count", cpu).Equals("memory_size_mb", memMB).Equals("cloudregion_id", regionId).Equals("postpaid_status", api.SkuStatusAvailable)
	err = q.First(ret)
	if err != nil {
		return nil, errors.Wrap(err, "ServerSkuManager.GetMatchedSku")
	}

	return ret, nil
}

func (manager *SServerSkuManager) FetchSkuByNameAndProvider(name string, provider string, checkConsistency bool) (*SServerSku, error) {
	q := manager.Query().IsTrue("enabled")
	q = q.Equals("name", name)
	if utils.IsInStringArray(provider, []string{api.CLOUD_PROVIDER_ONECLOUD, api.CLOUD_PROVIDER_VMWARE, api.CLOUD_PROVIDER_NUTANIX}) {
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
func (manager *SServerSkuManager) DeleteInvalidSkus() error {
	_, err := sqlchemy.GetDB().Exec(
		fmt.Sprintf(
			"delete from %s where length(zone_id) > 0 and zone_id not in (select id from zones_tbl where deleted=0)",
			manager.TableSpec().Name(),
		),
	)
	return err
}

func (manager *SServerSkuManager) SyncPrivateCloudSkus(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	region *SCloudregion,
	skus []cloudprovider.ICloudSku,
	xor bool,
) compare.SyncResult {
	lockman.LockRawObject(ctx, manager.Keyword(), region.Id)
	defer lockman.ReleaseRawObject(ctx, manager.Keyword(), region.Id)

	result := compare.SyncResult{}

	dbSkus, err := region.GetServerSkus()
	if err != nil {
		result.Error(errors.Wrapf(err, "GetServerSkus"))
		return result
	}

	removed := make([]SServerSku, 0)
	commondb := make([]SServerSku, 0)
	commonext := make([]cloudprovider.ICloudSku, 0)
	added := make([]cloudprovider.ICloudSku, 0)

	err = compare.CompareSets(dbSkus, skus, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(errors.Wrapf(err, "CompareSets"))
		return result
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].RealDelete(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}

	if !xor {
		for i := 0; i < len(commondb); i += 1 {
			err = commondb[i].SyncWithPrivateCloudSku(ctx, userCred, commonext[i])
			if err != nil {
				result.UpdateError(err)
			}
			result.Update()
		}
	}

	for i := 0; i < len(added); i += 1 {
		err := manager.newPrivateCloudSku(ctx, userCred, region, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}

	return result
}

func (self *SServerSku) SyncWithPrivateCloudSku(ctx context.Context, userCred mcclient.TokenCredential, sku cloudprovider.ICloudSku) error {
	_, err := db.Update(self, func() error {
		self.Status = api.SkuStatusAvailable
		self.constructSku(sku)
		return nil
	})
	return err
}

func (self *SServerSku) constructSku(extSku cloudprovider.ICloudSku) {
	self.ExternalId = extSku.GetGlobalId()
	self.InstanceTypeFamily = extSku.GetInstanceTypeFamily()
	self.InstanceTypeCategory = extSku.GetInstanceTypeCategory()

	self.PrepaidStatus = extSku.GetPrepaidStatus()
	self.PostpaidStatus = extSku.GetPostpaidStatus()

	self.CpuArch = extSku.GetCpuArch()
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

func (region *SCloudregion) newPublicCloudSku(ctx context.Context, userCred mcclient.TokenCredential, extSku SServerSku) error {
	meta, err := yunionmeta.FetchYunionmeta(ctx)
	if err != nil {
		return err
	}
	zones, err := region.GetZones()
	if err != nil {
		return errors.Wrap(err, "GetZones")
	}
	zoneMaps := map[string]string{}
	for _, zone := range zones {
		zoneMaps[zone.ExternalId] = zone.Id
	}

	sku := &SServerSku{}
	sku.SetModelManager(ServerSkuManager, sku)

	skuUrl := fmt.Sprintf("%s/%s/%s.json", meta.ServerBase, region.ExternalId, extSku.ExternalId)
	err = meta.Get(skuUrl, sku)
	if err != nil {
		return errors.Wrapf(err, "Get")
	}

	if len(sku.ZoneId) > 0 {
		zoneId := sku.ZoneId
		sku.ZoneId = yunionmeta.GetZoneIdBySuffix(zoneMaps, zoneId)
		if len(sku.ZoneId) == 0 {
			return errors.Wrapf(cloudprovider.ErrNotFound, zoneId)
		}
	}

	// 第一次同步新建的套餐是启用状态
	sku.Enabled = tristate.True
	sku.Status = api.SkuStatusReady
	sku.CloudregionId = region.Id
	sku.Provider = region.Provider
	return ServerSkuManager.TableSpec().Insert(ctx, sku)
}

func (manager *SServerSkuManager) newPrivateCloudSku(ctx context.Context, userCred mcclient.TokenCredential, region *SCloudregion, extSku cloudprovider.ICloudSku) error {
	sku := &SServerSku{Provider: region.Provider}

	sku.SetModelManager(manager, sku)
	// 第一次同步新建的套餐是启用状态
	sku.Enabled = tristate.True
	sku.Status = api.SkuStatusReady
	sku.constructSku(extSku)

	sku.CloudregionId = region.Id
	sku.Name = extSku.GetName()

	return manager.TableSpec().Insert(ctx, sku)
}

func (self *SServerSku) syncWithCloudSku(ctx context.Context, userCred mcclient.TokenCredential, region *SCloudregion, extSku SServerSku) error {
	if self.Md5 == extSku.Md5 {
		return nil
	}

	meta, err := yunionmeta.FetchYunionmeta(ctx)
	if err != nil {
		return err
	}

	sku := &SServerSku{}
	skuUrl := fmt.Sprintf("%s/%s/%s.json", meta.ServerBase, region.ExternalId, extSku.ExternalId)
	err = meta.Get(skuUrl, sku)
	if err != nil {
		return errors.Wrapf(err, "Get")
	}

	_, err = db.Update(self, func() error {
		self.PrepaidStatus = sku.PrepaidStatus
		self.PostpaidStatus = sku.PostpaidStatus
		self.SysDiskType = sku.SysDiskType
		self.DataDiskTypes = sku.DataDiskTypes
		self.CpuArch = sku.CpuArch
		self.InstanceTypeFamily = sku.InstanceTypeFamily
		self.InstanceTypeCategory = sku.InstanceTypeCategory
		self.LocalCategory = sku.LocalCategory
		self.NicType = sku.NicType
		self.GpuAttachable = sku.GpuAttachable
		self.GpuSpec = sku.GpuSpec
		self.GpuCount = sku.GpuCount
		self.Md5 = sku.Md5
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

func (manager *SServerSkuManager) SyncServerSkus(ctx context.Context, userCred mcclient.TokenCredential, region *SCloudregion, xor bool) compare.SyncResult {
	lockman.LockRawObject(ctx, manager.Keyword(), region.Id)
	defer lockman.ReleaseRawObject(ctx, manager.Keyword(), region.Id)

	result := compare.SyncResult{}

	meta, err := yunionmeta.FetchYunionmeta(ctx)
	if err != nil {
		result.Error(errors.Wrapf(err, "FetchYunionmeta"))
		return result
	}

	extSkus := []SServerSku{}
	err = meta.List(manager.Keyword(), region.ExternalId, &extSkus)
	if err != nil {
		result.Error(errors.Wrapf(err, "List"))
		return result
	}

	dbSkus, err := manager.FetchSkusByRegion(region.GetId())
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SServerSku, 0)
	commondb := make([]SServerSku, 0)
	commonext := make([]SServerSku, 0)
	added := make([]SServerSku, 0)

	err = compare.CompareSets(dbSkus, extSkus, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(err)
		return result
	}

	for i := 0; i < len(removed); i += 1 {
		cnt, err := removed[i].GetGuestCount()
		if err != nil || cnt > 0 {
			err = removed[i].MarkAsSoldout(ctx)
		} else {
			err = removed[i].RealDelete(ctx, userCred)
		}
		if err != nil {
			result.DeleteError(err)
		} else {
			result.Delete()
		}
	}
	if !xor {
		for i := 0; i < len(commondb); i += 1 {
			err = commondb[i].syncWithCloudSku(ctx, userCred, region, commonext[i])
			if err != nil {
				result.UpdateError(err)
			} else {
				result.Update()
			}
		}
	}

	ch := make(chan struct{}, options.Options.SkuBatchSync)
	defer close(ch)
	var wg sync.WaitGroup
	for i := 0; i < len(added); i += 1 {
		ch <- struct{}{}
		wg.Add(1)
		go func(sku SServerSku) {
			defer func() {
				wg.Done()
				<-ch
			}()
			err = region.newPublicCloudSku(ctx, userCred, sku)
			if err != nil {
				result.AddError(err)
			} else {
				result.Add()
			}
		}(added[i])
	}
	wg.Wait()

	// notfiy sched manager
	_, err = scheduler.SchedManager.SyncSku(auth.GetAdminSession(ctx, options.Options.Region), false)
	if err != nil {
		log.Errorf("SchedManager SyncSku %s", err)
	}

	return result
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

func (manager *SServerSkuManager) fixAliyunSkus() error {
	q := manager.Query().Equals("provider", api.CLOUD_PROVIDER_ALIYUN)
	q = q.Filter(sqlchemy.OR(
		sqlchemy.AND(
			sqlchemy.Contains(q.Field("sys_disk_type"), api.STORAGE_CLOUD_ESSD),
			sqlchemy.NOT(sqlchemy.Contains(q.Field("sys_disk_type"), api.STORAGE_CLOUD_ESSD_PL0)),
		),
		sqlchemy.AND(
			sqlchemy.Contains(q.Field("data_disk_types"), api.STORAGE_CLOUD_ESSD),
			sqlchemy.NOT(sqlchemy.Contains(q.Field("data_disk_types"), api.STORAGE_CLOUD_ESSD_PL0)),
		),
	))
	skus := []SServerSku{}
	err := db.FetchModelObjects(manager, q, &skus)
	if err != nil {
		return errors.Wrapf(err, "db.FetchModelObjects")
	}
	storages := []string{api.STORAGE_CLOUD_ESSD_PL0, api.STORAGE_CLOUD_ESSD_PL2, api.STORAGE_CLOUD_ESSD_PL3}
	for i := range skus {
		_, err := db.Update(&skus[i], func() error {
			sys := strings.Split(skus[i].SysDiskType, ",")
			if utils.IsInStringArray(api.STORAGE_CLOUD_ESSD, sys) {
				for _, storage := range storages {
					if !utils.IsInStringArray(storage, sys) {
						sys = append(sys, storage)
					}
				}
				skus[i].SysDiskType = strings.Join(sys, ",")
			}
			data := strings.Split(skus[i].DataDiskTypes, ",")
			if utils.IsInStringArray(api.STORAGE_CLOUD_ESSD, data) {
				for _, storage := range storages {
					if !utils.IsInStringArray(storage, data) {
						data = append(data, storage)
					}
				}
				skus[i].DataDiskTypes = strings.Join(data, ",")
			}
			return nil
		})
		if err != nil {
			return errors.Wrapf(err, "db.Update")
		}
	}
	return nil
}

func (manager *SServerSkuManager) InitializeData() error {
	count, err := manager.Query().Equals("cloudregion_id", api.DEFAULT_REGION_ID).IsNullOrEmpty("zone_id").CountWithError()
	if err != nil {
		return errors.Wrapf(err, "fetch default region skus")
	}
	if count == 0 {
		skus := []struct {
			cpu   int
			memGb int
		}{
			{1, 1}, {1, 2}, {1, 4}, {1, 8},
			{2, 2}, {2, 4}, {2, 8}, {2, 12}, {2, 16},
			{4, 4}, {4, 12}, {4, 16}, {4, 24}, {4, 32},
			{8, 8}, {8, 16}, {8, 24}, {8, 32}, {8, 64},
			{12, 12}, {12, 16}, {12, 24}, {12, 32}, {12, 64},
			{16, 16}, {16, 24}, {16, 32}, {16, 48}, {16, 64},
			{24, 24}, {24, 32}, {24, 48}, {24, 64}, {24, 128},
			{32, 32}, {32, 48}, {32, 64}, {32, 128},
		}

		for _, item := range skus {
			sku := &SServerSku{}
			sku.CloudregionId = api.DEFAULT_REGION_ID
			sku.CpuCoreCount = item.cpu
			sku.MemorySizeMB = item.memGb * 1024
			sku.IsEmulated = false
			sku.Enabled = tristate.True
			sku.InstanceTypeCategory = api.SkuCategoryGeneralPurpose
			sku.LocalCategory = api.SkuCategoryGeneralPurpose
			sku.InstanceTypeFamily = api.InstanceFamilies[api.SkuCategoryGeneralPurpose]
			sku.Name, _ = genInstanceType(sku.InstanceTypeFamily, int64(item.cpu), int64(item.memGb*1024))
			sku.PrepaidStatus = api.SkuStatusAvailable
			sku.PostpaidStatus = api.SkuStatusAvailable
			err := manager.TableSpec().Insert(context.TODO(), sku)
			if err != nil {
				log.Errorf("ServerSkuManager Initialize local sku %s", err)
			}
		}
	}

	privateSkus := make([]SServerSku, 0)
	q := manager.Query().IsNullOrEmpty("local_category").IsNotNull("instance_type_category").IsNullOrEmpty("zone_id")
	if err != nil {
		return err
	}

	err = db.FetchModelObjects(manager, q, &privateSkus)
	if err != nil {
		return err
	}

	for i := range privateSkus {
		_, err = db.Update(&privateSkus[i], func() error {
			privateSkus[i].LocalCategory = privateSkus[i].InstanceTypeCategory
			return nil
		})
		if err != nil {
			return err
		}
	}

	err = manager.fixAliyunSkus()
	if err != nil {
		return errors.Wrapf(err, "fixAliyunSkus")
	}

	return manager.initializeSkuStatus()
}

func (manager *SServerSkuManager) ListItemExportKeys(ctx context.Context,
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
	if keys.Contains("zone") {
		q, err = manager.SZoneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, stringutils2.NewSortedStrings([]string{"zone"}))
		if err != nil {
			return nil, errors.Wrap(err, "SZoneResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}

func (manager *SServerSkuManager) PerformSyncSkus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.SkuSyncInput) (jsonutils.JSONObject, error) {
	return PerformActionSyncSkus(ctx, userCred, manager.Keyword(), input)
}

func (manager *SServerSkuManager) GetPropertySyncTasks(ctx context.Context, userCred mcclient.TokenCredential, query api.SkuTaskQueryInput) (jsonutils.JSONObject, error) {
	return GetPropertySkusSyncTasks(ctx, userCred, query)
}

func (self *SServerSku) GetICloudSku(ctx context.Context) (cloudprovider.ICloudSku, error) {
	region, err := self.GetRegion()
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, errors.Wrapf(cloudprovider.ErrNotFound, "GetRegion")
		}
		return nil, errors.Wrapf(err, "GetRegion")
	}
	provider, err := region.GetCloudprovider()
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, errors.Wrapf(cloudprovider.ErrNotFound, "GetCloudprovider")
		}
		return nil, errors.Wrapf(err, "GetCloudprovider")
	}
	driver, err := provider.GetProvider(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "GetDriver()")
	}
	iRegion, err := driver.GetIRegionById(region.ExternalId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetIRegionById(%s)", region.ExternalId)
	}
	skus, err := iRegion.GetISkus()
	if err != nil {
		return nil, errors.Wrapf(err, "GetICloudSku")
	}
	for i := range skus {
		if skus[i].GetGlobalId() == self.ExternalId {
			return skus[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, self.ExternalId)
}

func fetchSkuSyncCloudregions() []SCloudregion {
	cloudregions := []SCloudregion{}
	q := CloudregionManager.Query()
	q = q.In("provider", CloudproviderManager.GetPublicProviderProvidersQuery())
	err := db.FetchModelObjects(CloudregionManager, q, &cloudregions)
	if err != nil {
		log.Errorf("fetchSkuSyncCloudregions.FetchCloudregions failed: %v", err)
		return nil
	}

	return cloudregions
}

// 全量同步sku列表.
func SyncServerSkus(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	// 清理无效的sku
	log.Debugf("DeleteInvalidSkus in processing...")
	err := ServerSkuManager.DeleteInvalidSkus()

	if isStart {
		cnt, err := ServerSkuManager.GetPublicCloudSkuCount()
		if err != nil {
			log.Errorf("GetPublicCloudSkuCount fail %s", err)
			return
		}
		if cnt > 0 {
			log.Debugf("GetPublicCloudSkuCount synced skus, skip...")
			return
		}
	}

	meta, err := yunionmeta.FetchYunionmeta(ctx)
	if err != nil {
		log.Errorf("FetchYunionmeta %v", err)
		return
	}

	index, err := meta.Index(ServerSkuManager.Keyword())
	if err != nil {
		log.Errorf("getServerSkuIndex error: %v", err)
		return
	}

	cloudregions := fetchSkuSyncCloudregions()
	for i := range cloudregions {
		region := &cloudregions[i]

		skuMeta := &SServerSku{}
		skuMeta.SetModelManager(ServerSkuManager, skuMeta)
		skuMeta.Id = region.ExternalId

		oldMd5 := db.Metadata.GetStringValue(ctx, skuMeta, db.SKU_METADAT_KEY, userCred)
		newMd5, ok := index[region.ExternalId]
		if !ok || newMd5 == yunionmeta.EMPTY_MD5 || len(oldMd5) > 0 && newMd5 == oldMd5 {
			continue
		}

		db.Metadata.SetValue(ctx, skuMeta, db.SKU_METADAT_KEY, newMd5, userCred)

		result := ServerSkuManager.SyncServerSkus(ctx, userCred, region, false)
		notes := fmt.Sprintf("SyncServerSkusByRegion %s result: %s", region.Name, result.Result())
		log.Debugf(notes)
	}

}

// 同步指定region sku列表
func SyncServerSkusByRegion(ctx context.Context, userCred mcclient.TokenCredential, region *SCloudregion, xor bool) compare.SyncResult {
	result := compare.SyncResult{}
	result = ServerSkuManager.SyncServerSkus(ctx, userCred, region, xor)
	notes := fmt.Sprintf("SyncServerSkusByRegion %s result: %s", region.Name, result.Result())
	log.Infof(notes)
	return result
}
