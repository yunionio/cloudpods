package models

import (
	"context"
	"database/sql"
	"encoding/json"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
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
}

// SServerSku 实际对应的是instance type清单. 这里的Sku实际指的是instance type。
type SServerSku struct {
	db.SStandaloneResourceBase

	// SkuId       string `width:"64" charset:"ascii" nullable:"false" list:"user" create:"admin_required"`                 // x2.large
	InstanceTypeFamily   string `width:"32" charset:"ascii" nullable:"false" list:"user" create:"admin_optional" update:"admin"` // x2
	InstanceTypeCategory string `width:"32" charset:"utf8" nullable:"false" list:"user" create:"admin_optional" update:"admin"`  // 通用型

	CpuCoreCount int `nullable:"false" list:"user" create:"admin_required" update:"admin"`
	MemorySizeMB int `nullable:"false" list:"user" create:"admin_required" update:"admin"`

	OsName string `width:"32" charset:"ascii" nullable:"false" list:"user" create:"admin_required" update:"admin" default:"Any"` // Windows|Linux|Any

	SysDiskResizable bool   `default:"true" nullable:"false" list:"user" create:"admin_optional" update:"admin"`
	SysDiskType      string `width:"32" charset:"ascii" nullable:"false" list:"user" create:"admin_required" update:"admin"`
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
	Provider      string `width:"64" charset:"ascii" nullable:"false" list:"user" create:"admin_optional" update:"admin"`
}

func inWhiteList(provider string) bool {
	// 只有为true的hypervisor才进行创建和更新操作
	switch provider {
	case HYPERVISOR_ESXI, HYPERVISOR_KVM:
		return true
	default:
		return false
	}
}

func (self *SServerSkuManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (self *SServerSku) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (self *SServerSkuManager) ValidateCreateData(ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerProjId string,
	query jsonutils.JSONObject,
	data *jsonutils.JSONDict,
) (*jsonutils.JSONDict, error) {
	provider, err := data.GetString("provider")
	if err != nil || len(provider) == 0 {
		return nil, httperrors.NewMissingParameterError("provider")
	}

	if !inWhiteList(provider) {
		return nil, httperrors.NewInputParameterError("can not create with provider %s", provider)
	}

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

func (self *SServerSkuManager) GetPropertyInstanceSpecs(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	q := self.Query()
	zone, err := query.GetString("zone")
	if err == nil && len(zone) > 0 {
		q = q.Equals("zone", zone)
	} else {
		return nil, httperrors.NewMissingParameterError("zone")
	}

	skus := make([]SServerSku, 0)
	q = q.GroupBy(q.Field("cpu_core_count"), q.Field("memory_size_mb"))
	q = q.Asc(q.Field("cpu_core_count"), q.Field("memory_size_mb"))
	err = q.All(&skus)
	if err != nil {
		log.Errorf("%s", err)
		return nil, httperrors.NewBadRequestError("instance specs list query error")
	}

	cpus := jsonutils.NewArray()
	mems_mb := jsonutils.NewArray()
	cpu_mems_mb := map[int][]int{}

	oc, om := 0, 0
	for i := range skus {
		nc := skus[i].CpuCoreCount
		nm := skus[i].MemorySizeMB

		if nc > oc {
			cpus.Add(jsonutils.NewInt(int64(nc)))
			oc = nc
		}

		if nm > om {
			mems_mb.Add(jsonutils.NewInt(int64(nm)))
			om = nm
		}

		if _, exists := cpu_mems_mb[nc]; !exists {
			cpu_mems_mb[nc] = []int{nm}
		} else {
			cpu_mems_mb[nc] = append(cpu_mems_mb[nc], nm)
		}
	}

	ret := jsonutils.NewDict()
	ret.Add(cpus, "cpus")
	ret.Add(mems_mb, "mems_mb")

	r, err := json.Marshal(&cpu_mems_mb)
	if err != nil {
		log.Errorf("%s", err)
		return nil, httperrors.NewInternalServerError("instance specs list marshal failed")
	}

	r_obj, err := jsonutils.Parse(r)
	if err != nil {
		log.Errorf("%s", err)
		return nil, httperrors.NewInternalServerError("instance specs list parse failed")
	}

	ret.Add(r_obj, "cpu_mems_mb")
	return ret, nil
}

func (self *SServerSku) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	if self.SStandaloneResourceBase.AllowUpdateItem(ctx, userCred) == false {
		return false
	}

	return inWhiteList(self.Provider)
}

func (self *SServerSku) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	provider, err := data.GetString("provider")
	if err == nil && !inWhiteList(provider) {
		return nil, httperrors.NewInputParameterError("can not create with provider %s", provider)
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
	return self.SStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (self *SServerSku) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	if !inWhiteList(self.Provider) {
		return false
	}

	count := GuestManager.Query().Equals("sku_id", self.Id).Count()
	if count == 0 {
		return self.SStandaloneResourceBase.AllowDeleteItem(ctx, userCred, query, data)
	} else {
		return false
	}
}

func (self *SServerSku) GetZoneExternalId() (string, error) {
	zoneObj, err := ZoneManager.FetchById(self.ZoneId)
	if err != nil {
		return "", err
	}

	zone := zoneObj.(*SZone)
	return zone.GetExternalId(), nil
}
