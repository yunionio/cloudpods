package models

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/hashcache"
)

const (
	SkuCategoryGeneralPurpose      = "general_purpose"      // 通用型
	SkuCategoryBurstable           = "burstable"            // 突发性能型
	SkuCategoryComputeOptimized    = "compute_optimized"    // 计算优化型
	SkuCategoryMemoryOptimized     = "memory_optimized"     // 内存优化型
	SkuCategoryStorageIOOptimized  = "storage_optimized"    // 存储IO优化型
	SkuCategoryHardwareAccelerated = "hardware_accelerated" // 硬件加速型
	SkuCategoryHighStorage         = "high_storage"         // 高存储型
	SkuCategoryHighMemory          = "high_memory"          // 高内存型
)

const (
	SkuStatusAvailable = "available"
	SkuStatusSoldout   = "soldout"
)

var InstanceFamilies = map[string]string{
	SkuCategoryGeneralPurpose:      "g1",
	SkuCategoryBurstable:           "t1",
	SkuCategoryComputeOptimized:    "c1",
	SkuCategoryMemoryOptimized:     "r1",
	SkuCategoryStorageIOOptimized:  "i1",
	SkuCategoryHardwareAccelerated: "",
	SkuCategoryHighStorage:         "hc1",
	SkuCategoryHighMemory:          "hr1",
}

var Cache *hashcache.Cache

type SServerSkuManager struct {
	db.SStandaloneResourceBaseManager
}

var ServerSkuManager *SServerSkuManager

func init() {
	ServerSkuManager = &SServerSkuManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SServerSku{},
			"serverskus_tbl",
			"serversku",
			"serverskus",
		),
	}
	ServerSkuManager.NameRequireAscii = false

	Cache = hashcache.NewCache(2048, time.Second*300)
}

// SServerSku 实际对应的是instance type清单. 这里的Sku实际指的是instance type。
type SServerSku struct {
	db.SStandaloneResourceBase

	// SkuId       string `width:"64" charset:"ascii" nullable:"false" list:"user" create:"admin_required"`                 // x2.large
	InstanceTypeFamily   string `width:"32" charset:"ascii" nullable:"false" list:"user" create:"admin_optional" update:"admin"`           // x2
	InstanceTypeCategory string `width:"32" charset:"utf8" nullable:"false" list:"user" create:"admin_optional" update:"admin"`            // 通用型
	LocalCategory        string `width:"32" charset:"utf8" nullable:"false" list:"user" create:"admin_optional" update:"admin" default:""` // 记录本地分类

	PrepaidStatus  string `width:"32" charset:"utf8" nullable:"false" list:"user" create:"admin_optional" update:"admin" default:"available"` // 预付费资源状态   available|soldout
	PostpaidStatus string `width:"32" charset:"utf8" nullable:"false" list:"user" create:"admin_optional" update:"admin" default:"available"` // 按需付费资源状态  available|soldout

	CpuCoreCount int `nullable:"false" list:"user" create:"admin_required" update:"admin"`
	MemorySizeMB int `nullable:"false" list:"user" create:"admin_required" update:"admin"`

	OsName string `width:"32" charset:"ascii" nullable:"false" list:"user" create:"admin_optional" update:"admin" default:"Any"` // Windows|Linux|Any

	SysDiskResizable bool   `default:"true" nullable:"false" list:"user" create:"admin_optional" update:"admin"`
	SysDiskType      string `width:"32" charset:"ascii" nullable:"false" list:"user" create:"admin_optional" update:"admin"`
	SysDiskMinSizeGB int    `nullable:"false" list:"user" create:"admin_optional" update:"admin"` // not required。 windows比较新的版本都是50G左右。
	SysDiskMaxSizeGB int    `nullable:"false" list:"user" create:"admin_optional" update:"admin"` // not required

	AttachedDiskType   string `nullable:"false" list:"user" create:"admin_optional" update:"admin"`
	AttachedDiskSizeGB int    `nullable:"false" list:"user" create:"admin_optional" update:"admin"`
	AttachedDiskCount  int    `nullable:"false" list:"user" create:"admin_optional" update:"admin"`

	DataDiskTypes    string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"admin_optional" update:"admin"`
	DataDiskMaxCount int    `nullable:"false" list:"user" create:"admin_optional" update:"admin"`

	NicType     string `nullable:"false" list:"user" create:"admin_optional" update:"admin"`
	NicMaxCount int    `default:"1" nullable:"false" list:"user" create:"admin_optional" update:"admin"`

	GpuAttachable bool   `default:"true" nullable:"false" list:"user" create:"admin_optional" update:"admin"`
	GpuSpec       string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"admin_optional" update:"admin"`
	GpuCount      int    `nullable:"false" list:"user" create:"admin_optional" update:"admin"`
	GpuMaxCount   int    `nullable:"false" list:"user" create:"admin_optional" update:"admin"`

	CloudregionId string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"admin_required" update:"admin"`
	ZoneId        string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"admin_optional" update:"admin"`
	Provider      string `width:"64" charset:"ascii" nullable:"true" list:"user" create:"admin_optional" update:"admin"`
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
	// provider 字段为空时表示私有云套餐
	if len(provider) == 0 {
		return true
	} else {
		return false
	}
}

func excludeSkus(q *sqlchemy.SQuery) *sqlchemy.SQuery {
	// 排除掉华为云对镜像有特殊要求的sku
	return q.Filter(
		sqlchemy.OR(
			sqlchemy.IsNullOrEmpty(q.Field("provider")),
			sqlchemy.NotEquals(q.Field("provider"), api.CLOUD_PROVIDER_HUAWEI),
			sqlchemy.AND(
				sqlchemy.Equals(q.Field("provider"), api.CLOUD_PROVIDER_HUAWEI),
				sqlchemy.NotIn(q.Field("instance_type_family"), []string{"e1", "e2", "e3", "d1", "d2", "i3", "h2", "g1", "g3", "p2v", "p1", "pi1", "fp1", "fp1c"}),
			)))
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

func skuRelatedGuestCount(self *SServerSku) int {
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
	return q.Count()
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

func (self *SServerSku) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (self *SServerSku) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	// count
	var count int
	countKey := self.GetId() + ".total_guest_count"
	v := Cache.Get(countKey)
	if v == nil {
		count = skuRelatedGuestCount(self)
		Cache.Set(countKey, count)
	} else {
		count = v.(int)
	}

	extra.Add(jsonutils.NewInt(int64(count)), "total_guest_count")

	// zone
	if len(self.ZoneId) > 0 {
		name, extId, err := getNameAndExtId(self.ZoneId, ZoneManager)
		if err == nil {
			extra.Add(jsonutils.NewString(name), "zone")
			extra.Add(jsonutils.NewString(extId), "zone_ext_id")
		} else {
			log.Debugf("GetCustomizeColumns %s", err)
		}
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
	extra, err := self.SStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	count := skuRelatedGuestCount(self)
	extra.Add(jsonutils.NewInt(int64(count)), "total_guest_count")
	return extra, nil
}

func (manager *SServerSkuManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, manager)
}

func (self *SServerSkuManager) ValidateCreateData(ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerProjId string,
	query jsonutils.JSONObject,
	data *jsonutils.JSONDict,
) (*jsonutils.JSONDict, error) {

	provider, _ := data.GetString("provider")

	if !inWhiteList(provider) {
		return nil, httperrors.NewForbiddenError("can not create instance_type for public cloud %s", provider)
	}

	data.Remove("provider")

	regionStr := jsonutils.GetAnyString(data, []string{"region", "region_id", "cloudregion", "cloudregion_id"})
	if len(regionStr) > 0 {
		regionObj, err := CloudregionManager.FetchByIdOrName(userCred, regionStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError("region %s not found", regionStr)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		data.Add(jsonutils.NewString(regionObj.GetId()), "cloudregion_id")
	} else {
		data.Add(jsonutils.NewString(DEFAULT_REGION_ID), "cloudregion_id")
	}
	zoneStr := jsonutils.GetAnyString(data, []string{"zone", "zone_id"})
	if len(zoneStr) > 0 {
		zoneObj, err := ZoneManager.FetchByIdOrName(userCred, zoneStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError("zone %s not found", zoneStr)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		data.Add(jsonutils.NewString(zoneObj.GetId()), "zone_id")
	}

	// name 由服务器端生成
	cpu, err := data.Int("cpu_core_count")
	if err != nil {
		return nil, httperrors.NewInputParameterError("cpu_core_count should not be empty")
	} else {
		data.Set("cpu_core_count", jsonutils.NewInt(cpu))
	}

	mem, err := data.Int("memory_size_mb")
	if err != nil {
		return nil, httperrors.NewInputParameterError("memory_size_mb should not be empty")
	} else {
		data.Set("memory_size_mb", jsonutils.NewInt(mem))
	}

	category, _ := data.GetString("instance_type_category")
	family, exists := InstanceFamilies[category]
	if !exists {
		return nil, httperrors.NewInputParameterError("instance_type_category %s is invalid", category)
	}
	data.Add(jsonutils.NewString(category), "local_category")

	data.Set("instance_type_family", jsonutils.NewString(family))
	// 格式 ecs.g1.c1m1
	name, err := genInstanceType(family, cpu, mem)
	if err != nil {
		return nil, httperrors.NewInputParameterError(err.Error())
	}

	data.Set("name", jsonutils.NewString(name))

	q := self.Query()
	q = q.Equals("name", name).Filter(sqlchemy.OR(
		sqlchemy.IsNull(q.Field("provider")),
		sqlchemy.IsEmpty(q.Field("provider")),
	))

	if q.Count() > 0 {
		return nil, httperrors.NewDuplicateResourceError("Duplicate sku %s", name)
	}

	return self.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
}

func (self *SServerSkuManager) FetchByZoneExtId(zoneExtId string, name string) (db.IModel, error) {
	zoneObj, err := ZoneManager.FetchByExternalId(zoneExtId)
	if err != nil {
		return nil, err
	}

	return self.FetchByZoneId(zoneObj.GetId(), name)
}

func (self *SServerSkuManager) FetchByZoneId(zoneId string, name string) (db.IModel, error) {
	q := self.Query().Equals("zone_id", zoneId).Equals("name", name)
	count := q.Count()
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

func providerFilter(q *sqlchemy.SQuery, provider string, public_cloud bool) *sqlchemy.SQuery {
	if provider == "all" {
		// provider 参数为all时。表示查询所有instance type.
		return q
	} else if len(provider) > 0 {
		q = q.Equals("provider", provider)
	} else if public_cloud {
		q = q.IsNotEmpty("provider")
		q = q.NotIn("provider", []string{api.CLOUD_PROVIDER_OPENSTACK})
	} else {
		q = q.Filter(sqlchemy.OR(
			sqlchemy.IsNull(q.Field("provider")),
			sqlchemy.IsEmpty(q.Field("provider")),
		))
	}

	return q
}

func (self *SServerSkuManager) GetPropertyInstanceSpecs(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	q := self.Query()
	// 未明确指定provider或者public_cloud时，默认查询私有云
	provider, _ := query.GetString("provider")
	public_cloud, _ := query.Bool("public_cloud")
	q = providerFilter(q, provider, public_cloud)
	q = excludeSkus(q)

	// 如果是查询私有云需要忽略zone参数
	zone := jsonutils.GetAnyString(query, []string{"zone", "zone_id"})
	if public_cloud && len(zone) > 0 {
		zoneObj, err := ZoneManager.FetchByIdOrName(userCred, zone)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(ZoneManager.Keyword(), zone)
			}
			return nil, httperrors.NewGeneralError(err)
		}

		q = q.Equals("zone_id", zoneObj.GetId())
	}

	skus := make([]SServerSku, 0)
	postpaid, _ := query.GetString("postpaid_status")
	if len(postpaid) > 0 {
		q.Equals("postpaid_status", postpaid)
	}

	prepaid, _ := query.GetString("prepaid_status")
	if len(prepaid) > 0 {
		q.Equals("prepaid_status", prepaid)
	}
	q = q.GroupBy(q.Field("cpu_core_count"), q.Field("memory_size_mb"))
	q = q.Asc(q.Field("cpu_core_count"), q.Field("memory_size_mb"))
	err := q.All(&skus)
	if err != nil {
		log.Errorf("%s", err)
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

func (self *SServerSku) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {

	if !inWhiteList(self.Provider) {
		return nil, httperrors.NewForbiddenError("can not update instance_type for public cloud %s", self.Provider)
	}

	provider, err := data.GetString("provider")
	if err == nil && !inWhiteList(provider) {
		return nil, httperrors.NewForbiddenError("can not update instance_type for public cloud %s", provider)
	}
	data.Remove("provider")

	zoneStr := jsonutils.GetAnyString(data, []string{"zone", "zone_id"})
	if len(zoneStr) > 0 {
		zoneObj, err := ZoneManager.FetchByIdOrName(userCred, zoneStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError("zone %s not found", zoneStr)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		data.Add(jsonutils.NewString(zoneObj.GetId()), "zone_id")
	}

	// 可用资源状态
	if postpaid, err := data.GetString("postpaid_status"); err != nil {
		if postpaid == SkuStatusSoldout {
			data.Set("postpaid_status", jsonutils.NewString(SkuStatusSoldout))
		} else {
			data.Set("postpaid_status", jsonutils.NewString(SkuStatusAvailable))
		}
	}

	prepaid, _ := data.GetString("prepaid_status")
	if prepaid == SkuStatusSoldout {
		data.Set("prepaid_status", jsonutils.NewString(SkuStatusSoldout))
	} else {
		data.Set("prepaid_status", jsonutils.NewString(SkuStatusAvailable))
	}

	// name 由服务器端生成
	// cpu, err := data.Int("cpu_core_count")
	// if err != nil {
	// 	cpu = int64(self.CpuCoreCount)
	// }
	// data.Set("cpu_core_count", jsonutils.NewInt(cpu))
	//
	// mem, err := data.Int("memory_size_mb")
	// if err != nil {
	// 	mem = int64(self.MemorySizeMB)
	// }
	// data.Set("memory_size_mb", jsonutils.NewInt(mem))
	//
	// category, err := data.GetString("instance_type_category")
	// family := ""
	// if err != nil {
	// 	family = self.InstanceTypeFamily
	// } else {
	// 	f, exists := InstanceFamilies[category]
	// 	if !exists {
	// 		return nil, httperrors.NewInputParameterError("instance_type_category %s is invalid", category)
	// 	}
	//
	// 	family = f
	// }
	//
	// data.Set("instance_type_family", jsonutils.NewString(family))
	// // 格式 ecs.g1.c1m1
	// name, err := genInstanceType(family, cpu, mem)
	// if err != nil {
	// 	return nil, httperrors.NewInputParameterError(err.Error())
	// }
	//
	// data.Set("name", jsonutils.NewString(name))
	// 暂时不允许修改CPU、MEM值
	data.Remove("cpu_core_count")
	data.Remove("memory_size_mb")
	data.Remove("name")
	// 暂时不允许修改CPU、MEM值
	// q := self.GetModelManager().Query()
	// q = q.Equals("name", name).Filter(sqlchemy.OR(
	// 	sqlchemy.IsNull(q.Field("provider")),
	// 	sqlchemy.IsEmpty(q.Field("provider")),
	// ))
	//
	// if q.Count() > 0 {
	// 	return nil, httperrors.NewDuplicateResourceError("sku cpu %d mem %d(Mb) already exists", cpu, mem)
	// }

	return self.SStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (self *SServerSku) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return inWhiteList(self.Provider) && db.IsAdminAllowDelete(userCred, self)
}

func (self *SServerSku) ValidateDeleteCondition(ctx context.Context) error {
	serverCount := GuestManager.Query().Equals("instance_type", self.Id).Count()
	if serverCount > 0 {
		return httperrors.NewForbiddenError("now allow to delete inuse instance_type.please remove related servers first: %s", self.Name)
	}

	if !inWhiteList(self.Provider) {
		return httperrors.NewForbiddenError("not allow to delete public cloud instance_type: %s", self.Name)
	}
	count := GuestManager.Query().Equals("instance_type", self.Name).Count()
	if count > 0 {
		return httperrors.NewNotEmptyError("instance_type used by servers")
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

func (manager *SServerSkuManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	provider := normalizeProvider(jsonutils.GetAnyString(query, []string{"provider"}))
	public_cloud, _ := query.Bool("public_cloud")
	queryDict := query.(*jsonutils.JSONDict)
	// 手动处理provider查询
	queryDict.Remove("provider")
	q = providerFilter(q, provider, public_cloud)
	q, err := manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}

	q = excludeSkus(q)

	regionTable := CloudregionManager.Query().SubQuery()
	q = q.Join(regionTable, sqlchemy.Equals(regionTable.Field("id"), q.Field("cloudregion_id")))

	// region filter
	regionStr := jsonutils.GetAnyString(query, []string{"region", "cloudregion", "region_id", "cloudregion_id"})
	if len(regionStr) > 0 {
		regionObj, err := CloudregionManager.FetchByIdOrName(nil, regionStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(CloudregionManager.Keyword(), regionStr)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		q = q.Equals("cloudregion_id", regionObj.GetId())
	}

	if public_cloud || len(provider) > 0 {
		zoneTable := ZoneManager.Query().SubQuery()
		q = q.Join(zoneTable, sqlchemy.Equals(zoneTable.Field("id"), q.Field("zone_id")))
	}

	zoneStr := jsonutils.GetAnyString(query, []string{"zone", "zone_id"})
	var zoneObj db.IModel
	if (public_cloud || len(provider) > 0) && len(zoneStr) > 0 {
		zoneObj, err = ZoneManager.FetchByIdOrName(nil, zoneStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(ZoneManager.Keyword(), zoneStr)
			}
			return nil, httperrors.NewGeneralError(err)
		}

		// 当查询私有云时，需要忽略zone参数
		if len(zoneObj.(*SZone).ExternalId) > 0 {
			q = q.Equals("zone_id", zoneObj.GetId())
		}
	}

	queryDict.Remove("zone")
	queryDict.Remove("zone_id")

	// city filter
	city, _ := query.GetString("city")
	if len(city) > 0 {
		q = q.Filter(sqlchemy.Equals(regionTable.Field("city"), city))
	}

	// 可用资源状态
	postpaid, _ := query.GetString("postpaid_status")
	if len(postpaid) > 0 {
		q.Equals("postpaid_status", postpaid)
	}

	prepaid, _ := query.GetString("prepaid_status")
	if len(prepaid) > 0 {
		q.Equals("prepaid_status", prepaid)
	}

	// 按区间查询内存
	memSizeMB, _ := queryDict.Int("memory_size_mb")
	if memSizeMB > 0 {
		s, e := intervalMem(int(memSizeMB))
		q.GT("memory_size_mb", s)
		q.LE("memory_size_mb", e)
		queryDict.Remove("memory_size_mb")
	}

	q = q.Asc(q.Field("cpu_core_count"), q.Field("memory_size_mb"))
	return q, err
}

func (manager *SServerSkuManager) FetchSkuByNameAndHypervisor(name string, hypervisor string, checkConsistency bool) (*SServerSku, error) {
	q := manager.Query()
	q = q.Equals("name", name)
	if len(hypervisor) > 0 {
		switch hypervisor {
		case HYPERVISOR_BAREMETAL, HYPERVISOR_CONTAINER:
			return nil, httperrors.NewNotImplementedError("%s not supported", hypervisor)
		case HYPERVISOR_KVM, HYPERVISOR_ESXI, HYPERVISOR_XEN, HOST_TYPE_HYPERV, HYPERVISOR_OPENSTACK:
			q = q.Filter(sqlchemy.OR(
				sqlchemy.IsEmpty(q.Field("provider")),
				sqlchemy.IsNull(q.Field("provider")),
				sqlchemy.Equals(q.Field("provider"), hypervisor),
			))
		default:
			q = q.Equals("provider", hypervisor)
		}
	} else {
		q = q.IsEmpty("provider")
	}
	skus := make([]SServerSku, 0)
	err := db.FetchModelObjects(manager, q, &skus)
	if err != nil {
		log.Errorf("fetch sku fail %s", err)
		return nil, err
	}
	if len(skus) == 0 {
		log.Errorf("no sku found for %s %s", name, hypervisor)
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

func (manager *SServerSkuManager) GetSkuCountByProvider(provider string) int {
	q := manager.Query()
	if len(provider) == 0 {
		q = q.IsNotEmpty("provider")
	} else {
		q = q.Equals("provider", provider)
	}

	return q.Count()
}

func (manager *SServerSkuManager) GetSkuCountByRegion(regionId string) int {
	q := manager.Query()
	if len(regionId) == 0 {
		q = q.IsNotEmpty("cloudregion_id")
	} else {
		q = q.Equals("cloudregion_id", regionId)
	}

	return q.Count()
}

func (manager *SServerSkuManager) GetSkuCountByZone(zoneId string) []SServerSku {
	skus := []SServerSku{}
	q := manager.Query().Equals("zone_id", zoneId)
	if err := db.FetchModelObjects(manager, q, &skus); err != nil {
		log.Errorf("failed to get skus by zoneId %s error: %v", zoneId, err)
	}
	return skus
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

func (manager *SServerSkuManager) SyncCloudSkusByZone(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, zone *SZone, skus []cloudprovider.ICloudSku) compare.SyncResult {
	lockman.LockClass(ctx, manager, manager.GetOwnerId(userCred))
	defer lockman.ReleaseClass(ctx, manager, manager.GetOwnerId(userCred))

	syncResult := compare.SyncResult{}
	dbSkus := manager.GetSkuCountByZone(zone.Id)

	removed := []SServerSku{}
	commondb := []SServerSku{}
	commonext := []cloudprovider.ICloudSku{}
	added := []cloudprovider.ICloudSku{}

	if err := compare.CompareSets(dbSkus, skus, &removed, &commondb, &commonext, &added); err != nil {
		syncResult.Error(err)
		return syncResult
	}

	for i := 0; i < len(removed); i++ {
		err := removed[i].syncRemoveCloudSku(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}

	for i := 0; i < len(commondb); i++ {
		err := commondb[i].syncWithCloudSku(ctx, userCred, commonext[i], zone, provider)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncResult.Update()
		}
	}

	for i := 0; i < len(added); i++ {
		err := manager.newFromCloudSku(ctx, userCred, added[i], zone, provider)
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncResult.Add()
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

	self.SysDiskResizable = extSku.GetSysDiskResizable()
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

	self.GpuAttachable = extSku.GetGpuAttachable()
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

func (self *SServerSku) syncRemoveCloudSku(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	err := self.ValidateDeleteCondition(ctx)
	if err == nil {
		err = self.Delete(ctx, userCred)
	} else {
		err = self.setPrepaidPostpaidStatus(userCred, SkuStatusSoldout, SkuStatusSoldout)
	}
	return err
}

func (self *SServerSku) syncWithCloudSku(ctx context.Context, userCred mcclient.TokenCredential, extSku cloudprovider.ICloudSku, zone *SZone, provider *SCloudprovider) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		self.constructSku(extSku)
		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return nil
}

func (manager *SServerSkuManager) newFromCloudSku(ctx context.Context, userCred mcclient.TokenCredential, extSku cloudprovider.ICloudSku, zone *SZone, provider *SCloudprovider) error {
	region := zone.GetRegion()
	sku := &SServerSku{
		CloudregionId: region.Id,
		ZoneId:        zone.Id,
		Provider:      provider.Provider,
	}
	sku.constructSku(extSku)
	sku.Name = db.GenerateName(manager, manager.GetOwnerId(userCred), extSku.GetName())
	sku.ExternalId = extSku.GetGlobalId()
	sku.SetModelManager(manager)
	err := manager.TableSpec().Insert(sku)
	if err != nil {
		log.Errorf("insert fail %s", err)
		return err
	}

	db.OpsLog.LogEvent(sku, db.ACT_CREATE, sku.GetShortDesc(ctx), userCred)

	return nil
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
		sku.PrepaidStatus = SkuStatusSoldout
		sku.PostpaidStatus = SkuStatusSoldout
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
		sqlchemy.Equals(q.Field("prepaid_status"), SkuStatusAvailable),
		sqlchemy.Equals(q.Field("postpaid_status"), SkuStatusAvailable)))

	err := q.All(&skus)
	if err != nil {
		return nil, err
	}

	ids := make([]string, len(skus))
	for i := range skus {
		ids[i] = skus[i].GetId()
	}

	return ids, nil
}

func (manager *SServerSkuManager) InitializeData() error {
	privateSkus := make([]SServerSku, 0)
	err := manager.Query().IsNullOrEmpty("local_category").IsNullOrEmpty("zone_id").All(&privateSkus)
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

	return nil
}
