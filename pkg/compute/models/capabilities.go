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
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SCapabilities struct {
	Hypervisors                 []string `json:",allowempty"`
	Brands                      []string `json:",allowempty"`
	DisabledBrands              []string `json:",allowempty"`
	ComputeEngineBrands         []string `json:",allowempty"`
	DisabledComputeEngineBrands []string `json:",allowempty"`
	RdsEngineBrands             []string `json:",allowempty"`
	RedisEngineBrands           []string `json:",allowempty"`
	LoadbalancerEngineBrands    []string `json:",allowempty"`
	DisabledRdsEngineBrands     []string `json:",allowempty"`
	CloudIdBrands               []string `json:",allowempty"`
	DisabledCloudIdBrands       []string `json:",allowempty"`
	// 支持SAML 2.0
	SamlAuthBrands              []string `json:",allowempty"`
	DisabledSamlAuthBrands      []string `json:",allowempty"`
	NatBrands                   []string `json:",allowempty"`
	DisabledNatBrands           []string `json:",allowempty"`
	NasBrands                   []string `json:",allowempty"`
	DisabledNasBrands           []string `json:",allowempty"`
	PublicIpBrands              []string `json:",allowempty"`
	NetworkManageBrands         []string `json:",allowempty"`
	DisabledNetworkManageBrands []string `json:",allowempty"`
	ObjectStorageBrands         []string `json:",allowempty"`
	DisabledObjectStorageBrands []string `json:",allowempty"`
	ResourceTypes               []string `json:",allowempty"`
	StorageTypes                []string `json:",allowempty"` // going to remove on 2.14
	DataStorageTypes            []string `json:",allowempty"` // going to remove on 2.14
	GPUModels                   []string `json:",allowempty"`
	HostCpuArchs                []string `json:",allowempty"` // x86_64 aarch64
	MinNicCount                 int
	MaxNicCount                 int
	MinDataDiskCount            int
	MaxDataDiskCount            int
	SchedPolicySupport          bool
	Usable                      bool

	// Deprecated
	PublicNetworkCount int

	AutoAllocNetworkCount int
	DBInstance            map[string]map[string]map[string][]string //map[engine][engineVersion][category][]{storage_type}
	Specs                 jsonutils.JSONObject
	AvailableHostCount    int

	StorageTypes2     map[string][]string                      `json:",allowempty"`
	StorageTypes3     map[string]map[string]*SimpleStorageInfo `json:",allowempty"`
	DataStorageTypes2 map[string][]string                      `json:",allowempty"`
	DataStorageTypes3 map[string]map[string]*SimpleStorageInfo `json:",allowempty"`

	InstanceCapabilities []cloudprovider.SInstanceCapability
}

func GetDiskCapabilities(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, region *SCloudregion, zone *SZone) (SCapabilities, error) {
	capa := SCapabilities{}
	s1, d1, s2, s3, d2, d3 := getStorageTypes(region, zone, "")
	capa.StorageTypes, capa.DataStorageTypes = s1, d1
	capa.StorageTypes2, capa.StorageTypes3 = s2, s3
	capa.DataStorageTypes2, capa.DataStorageTypes3 = d2, d3
	capa.MinDataDiskCount = getMinDataDiskCount(region, zone)
	capa.MaxDataDiskCount = getMaxDataDiskCount(region, zone)
	return capa, nil
}

func GetCapabilities(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, region *SCloudregion, zone *SZone) (SCapabilities, error) {
	capa := SCapabilities{}
	var ownerId mcclient.IIdentityProvider
	scopeStr := jsonutils.GetAnyString(query, []string{"scope"})
	scope := rbacutils.String2Scope(scopeStr)
	var domainId string
	domainStr := jsonutils.GetAnyString(query, []string{"domain", "domain_id", "project_domain", "project_domain_id"})
	if len(domainStr) > 0 {
		domain, err := db.TenantCacheManager.FetchDomainByIdOrName(ctx, domainStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return capa, httperrors.NewResourceNotFoundError2("domains", domainStr)
			}
			return capa, httperrors.NewGeneralError(err)
		}
		domainId = domain.GetId()
		ownerId = &db.SOwnerId{DomainId: domainId}
		scope = rbacutils.ScopeDomain
	} else {
		domainId = userCred.GetProjectDomainId()
		ownerId = userCred
	}
	if scope == rbacutils.ScopeSystem {
		result := policy.PolicyManager.Allow(scope, userCred, consts.GetServiceType(), "capabilities", policy.PolicyActionList)
		if result != rbacutils.Allow {
			return capa, httperrors.NewForbiddenError("not allow to query system capability")
		}
		domainId = ""
	}
	capa.Hypervisors = getHypervisors(region, zone, domainId)
	capa.InstanceCapabilities = []cloudprovider.SInstanceCapability{}
	for _, hypervisor := range capa.Hypervisors {
		driver := GetDriver(hypervisor)
		if driver != nil {
			capa.InstanceCapabilities = append(capa.InstanceCapabilities, driver.GetInstanceCapability())
		}
	}
	getBrands(region, zone, domainId, &capa)
	// capa.Brands, capa.ComputeEngineBrands, capa.NetworkManageBrands, capa.ObjectStorageBrands = a, c, n, o
	capa.ResourceTypes = getResourceTypes(region, zone, domainId)
	s1, d1, s2, s3, d2, d3 := getStorageTypes(region, zone, domainId)
	capa.StorageTypes, capa.DataStorageTypes = s1, d1
	capa.StorageTypes2, capa.StorageTypes3 = s2, s3
	capa.DataStorageTypes2, capa.DataStorageTypes3 = d2, d3
	capa.GPUModels = getGPUs(region, zone, domainId)
	capa.SchedPolicySupport = isSchedPolicySupported(region, zone)
	capa.MinNicCount = getMinNicCount(region, zone)
	capa.MaxNicCount = getMaxNicCount(region, zone)
	capa.MinDataDiskCount = getMinDataDiskCount(region, zone)
	capa.MaxDataDiskCount = getMaxDataDiskCount(region, zone)
	capa.DBInstance = getDBInstanceInfo(region, zone)
	capa.Usable = isUsable(ownerId, scope, region, zone)
	capa.HostCpuArchs = getHostCpuArchs(region, zone, domainId)
	if query == nil {
		query = jsonutils.NewDict()
	}
	if region != nil {
		query.(*jsonutils.JSONDict).Add(jsonutils.NewString(region.GetId()), "region")
	}
	if zone != nil {
		query.(*jsonutils.JSONDict).Add(jsonutils.NewString(zone.GetId()), "zone")
	}
	if len(domainId) > 0 {
		query.(*jsonutils.JSONDict).Add(jsonutils.NewString(domainId), "domain_id")
	}
	var err error
	serverType := jsonutils.GetAnyString(query, []string{"host_type", "server_type"})
	autoAllocNetworkCount, _ := getAutoAllocNetworkCount(ownerId, scope, region, zone, serverType)
	capa.PublicNetworkCount = autoAllocNetworkCount
	capa.AutoAllocNetworkCount = autoAllocNetworkCount
	mans := []ISpecModelManager{HostManager, IsolatedDeviceManager}
	capa.Specs, err = GetModelsSpecs(ctx, userCred, query.(*jsonutils.JSONDict), mans...)
	if err != nil {
		return capa, err
	}
	capa.AvailableHostCount, err = GetAvailableHostCount(userCred, query.(*jsonutils.JSONDict))
	return capa, err
}

func GetAvailableHostCount(userCred mcclient.TokenCredential, query *jsonutils.JSONDict) (int, error) {
	zoneStr, _ := query.GetString("zone")
	izone, _ := ZoneManager.FetchByIdOrName(userCred, zoneStr)
	var zoneId string
	if izone != nil {
		zoneId = izone.GetId()
	}

	regionStr, _ := query.GetString("region")
	iregion, _ := CloudregionManager.FetchByIdOrName(userCred, regionStr)
	var regionId string
	if iregion != nil {
		regionId = iregion.GetId()
	}

	domainId, _ := query.GetString("domain_id")
	q := HostManager.Query().Equals("enabled", true).
		Equals("host_status", "online").Equals("host_type", api.HOST_TYPE_HYPERVISOR)
	if len(domainId) > 0 {
		ownerId := &db.SOwnerId{DomainId: domainId}
		q = HostManager.FilterByOwner(q, ownerId, rbacutils.ScopeDomain)
	}
	if len(zoneId) > 0 {
		q = q.Equals("zone_id", zoneId)
	}
	if len(regionId) > 0 {
		subq := ZoneManager.Query("id").Equals("cloudregion_id", regionId).SubQuery()
		q = q.Filter(sqlchemy.In(q.Field("zone_id"), subq))
	}
	return q.CountWithError()
}

func getRegionZoneSubq(region *SCloudregion) *sqlchemy.SSubQuery {
	return ZoneManager.Query("id").Equals("cloudregion_id", region.GetId()).SubQuery()
}

func domainManagerFieldFilter(domainId, field string) *sqlchemy.SSubQuery {
	accounts := CloudaccountManager.Query("id")
	accounts = CloudaccountManager.filterByDomainId(accounts, domainId)
	accounts = accounts.Equals("status", api.CLOUD_PROVIDER_CONNECTED)
	accounts = accounts.IsTrue("enabled")

	q := CloudproviderManager.Query(field).In("cloudaccount_id", accounts.SubQuery())
	/*q := providers.Query(providers.Field(field))
	q = q.Join(accounts, sqlchemy.Equals(accounts.Field("id"), providers.Field("cloudaccount_id")))
	q = q.Filter(sqlchemy.OR(
		sqlchemy.AND(
			sqlchemy.Equals(providers.Field("domain_id"), domainId),
			sqlchemy.Equals(accounts.Field("share_mode"), api.CLOUD_ACCOUNT_SHARE_MODE_PROVIDER_DOMAIN),
		),
		sqlchemy.Equals(accounts.Field("share_mode"), api.CLOUD_ACCOUNT_SHARE_MODE_SYSTEM),
		sqlchemy.AND(
			sqlchemy.Equals(accounts.Field("domain_id"), domainId),
			sqlchemy.Equals(accounts.Field("share_mode"), api.CLOUD_ACCOUNT_SHARE_MODE_ACCOUNT_DOMAIN),
		),
	))
	q = q.Filter(sqlchemy.Equals(accounts.Field("status"), api.CLOUD_PROVIDER_CONNECTED))
	q = q.Filter(sqlchemy.IsTrue(accounts.Field("enabled")))*/

	return q.SubQuery()
}

/*func getDomainManagerSubq(domainId string) *sqlchemy.SSubQuery {
	return domainManagerFieldFilter(domainId, "id")
}*/

func getDomainManagerProviderSubq(domainId string) *sqlchemy.SSubQuery {
	return domainManagerFieldFilter(domainId, "provider")
}

func getDBInstanceInfo(region *SCloudregion, zone *SZone) map[string]map[string]map[string][]string {
	if zone != nil {
		region = zone.GetRegion()
	}
	if region == nil {
		return nil
	}

	q := DBInstanceSkuManager.Query("engine", "engine_version", "category", "storage_type", "zone1", "zone2", "zone3").Equals("cloudregion_id", region.Id).IsTrue("enabled").Equals("status", api.DBINSTANCE_SKU_AVAILABLE).Distinct()
	if zone != nil {
		q = q.Filter(sqlchemy.OR(
			sqlchemy.Equals(q.Field("zone1"), zone.Id),
			sqlchemy.Equals(q.Field("zone2"), zone.Id),
			sqlchemy.Equals(q.Field("zone3"), zone.Id),
		))
	}
	rows, err := q.Rows()
	if err != nil {
		return nil
	}
	defer rows.Close()
	result := map[string]map[string]map[string][]string{}
	for rows.Next() {
		var engine, engineVersion, category, storageType, zone1, zone2, zone3 string
		rows.Scan(&engine, &engineVersion, &category, &storageType, &zone1, &zone2, &zone3)
		if _, ok := result[engine]; !ok {
			result[engine] = map[string]map[string][]string{}
		}
		if _, ok := result[engine][engineVersion]; !ok {
			result[engine][engineVersion] = map[string][]string{}
		}
		if _, ok := result[engine][engineVersion][category]; !ok {
			result[engine][engineVersion][category] = []string{}
		}
		if !utils.IsInStringArray(storageType, result[engine][engineVersion][category]) {
			result[engine][engineVersion][category] = append(result[engine][engineVersion][category], storageType)
		}
	}
	return result
}

// set all brands, compute engine brands, network manage brands, object storage brands
func getBrands(region *SCloudregion, zone *SZone, domainId string, capa *SCapabilities) {
	capa.Brands, _ = CloudaccountManager.getBrandsOfCapability(region, zone, domainId, tristate.True, "")
	capa.ComputeEngineBrands, _ = CloudaccountManager.getBrandsOfCapability(region, zone, domainId, tristate.True, cloudprovider.CLOUD_CAPABILITY_COMPUTE)
	capa.RdsEngineBrands, _ = CloudaccountManager.getBrandsOfCapability(region, zone, domainId, tristate.True, cloudprovider.CLOUD_CAPABILITY_RDS)
	capa.RedisEngineBrands, _ = CloudaccountManager.getBrandsOfCapability(region, zone, domainId, tristate.True, cloudprovider.CLOUD_CAPABILITY_CACHE)
	capa.NetworkManageBrands, _ = CloudaccountManager.getBrandsOfCapability(region, zone, domainId, tristate.True, cloudprovider.CLOUD_CAPABILITY_NETWORK)
	capa.ObjectStorageBrands, _ = CloudaccountManager.getBrandsOfCapability(region, zone, domainId, tristate.True, cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE)
	capa.CloudIdBrands, _ = CloudaccountManager.getBrandsOfCapability(region, zone, domainId, tristate.True, cloudprovider.CLOUD_CAPABILITY_CLOUDID)
	capa.PublicIpBrands, _ = CloudaccountManager.getBrandsOfCapability(region, zone, domainId, tristate.True, cloudprovider.CLOUD_CAPABILITY_PUBLIC_IP)
	capa.LoadbalancerEngineBrands, _ = CloudaccountManager.getBrandsOfCapability(region, zone, domainId, tristate.True, cloudprovider.CLOUD_CAPABILITY_LOADBALANCER)
	capa.SamlAuthBrands, _ = CloudaccountManager.getBrandsOfCapability(region, zone, domainId, tristate.True, cloudprovider.CLOUD_CAPABILITY_SAML_AUTH)
	capa.NatBrands, _ = CloudaccountManager.getBrandsOfCapability(region, zone, domainId, tristate.True, cloudprovider.CLOUD_CAPABILITY_NAT)
	capa.NasBrands, _ = CloudaccountManager.getBrandsOfCapability(region, zone, domainId, tristate.True, cloudprovider.CLOUD_CAPABILITY_NAS)

	if utils.IsInStringArray(api.HYPERVISOR_KVM, capa.Hypervisors) || utils.IsInStringArray(api.HYPERVISOR_BAREMETAL, capa.Hypervisors) {
		capa.Brands = append(capa.Brands, api.ONECLOUD_BRAND_ONECLOUD)
		capa.ComputeEngineBrands = append(capa.ComputeEngineBrands, api.ONECLOUD_BRAND_ONECLOUD)
	}

	if count, _ := LoadbalancerClusterManager.Query().Limit(1).CountWithError(); count > 0 {
		capa.LoadbalancerEngineBrands = append(capa.LoadbalancerEngineBrands, api.ONECLOUD_BRAND_ONECLOUD)
	}

	capa.NetworkManageBrands = append(capa.NetworkManageBrands, api.ONECLOUD_BRAND_ONECLOUD)

	capa.DisabledBrands, _ = CloudaccountManager.getBrandsOfCapability(region, zone, domainId, tristate.False, "")
	capa.DisabledComputeEngineBrands, _ = CloudaccountManager.getBrandsOfCapability(region, zone, domainId, tristate.False, cloudprovider.CLOUD_CAPABILITY_COMPUTE)
	capa.DisabledRdsEngineBrands, _ = CloudaccountManager.getBrandsOfCapability(region, zone, domainId, tristate.False, cloudprovider.CLOUD_CAPABILITY_RDS)
	capa.DisabledNetworkManageBrands, _ = CloudaccountManager.getBrandsOfCapability(region, zone, domainId, tristate.False, cloudprovider.CLOUD_CAPABILITY_NETWORK)
	capa.DisabledObjectStorageBrands, _ = CloudaccountManager.getBrandsOfCapability(region, zone, domainId, tristate.False, cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE)
	capa.DisabledCloudIdBrands, _ = CloudaccountManager.getBrandsOfCapability(region, zone, domainId, tristate.False, cloudprovider.CLOUD_CAPABILITY_CLOUDID)
	capa.DisabledSamlAuthBrands, _ = CloudaccountManager.getBrandsOfCapability(region, zone, domainId, tristate.False, cloudprovider.CLOUD_CAPABILITY_SAML_AUTH)
	capa.DisabledNatBrands, _ = CloudaccountManager.getBrandsOfCapability(region, zone, domainId, tristate.False, cloudprovider.CLOUD_CAPABILITY_NAT)
	capa.DisabledNasBrands, _ = CloudaccountManager.getBrandsOfCapability(region, zone, domainId, tristate.False, cloudprovider.CLOUD_CAPABILITY_NAS)

	return
}

func getHypervisors(region *SCloudregion, zone *SZone, domainId string) []string {
	q := HostManager.Query("host_type", "manager_id")
	if region != nil {
		subq := getRegionZoneSubq(region)
		q = q.Filter(sqlchemy.In(q.Field("zone_id"), subq))
	}
	if zone != nil {
		q = q.Equals("zone_id", zone.Id)
	}
	if len(domainId) > 0 {
		ownerId := &db.SOwnerId{DomainId: domainId}
		q = HostManager.FilterByOwner(q, ownerId, rbacutils.ScopeDomain)
		/*subq := getDomainManagerSubq(domainId)
		q = q.Filter(sqlchemy.OR(
			sqlchemy.In(q.Field("manager_id"), subq),
			sqlchemy.IsNullOrEmpty(q.Field("manager_id")),
		))*/
	}
	q = q.IsNotEmpty("host_type").IsNotNull("host_type")
	// q = q.Equals("host_status", HOST_ONLINE)
	q = q.IsTrue("enabled")
	q = q.Distinct()
	rows, err := q.Rows()
	if err != nil {
		return nil
	}
	defer rows.Close()
	hypervisors := make([]string, 0)
	for rows.Next() {
		var hostType string
		var managerId string
		rows.Scan(&hostType, &managerId)
		if len(hostType) > 0 && IsProviderAccountEnabled(managerId) {
			hypervisor := api.HOSTTYPE_HYPERVISOR[hostType]
			if !utils.IsInStringArray(hypervisor, hypervisors) {
				hypervisors = append(hypervisors, hypervisor)
			}
		}
	}
	return hypervisors
}

func getResourceTypes(region *SCloudregion, zone *SZone, domainId string) []string {
	q := HostManager.Query("resource_type", "manager_id")
	if region != nil {
		subq := getRegionZoneSubq(region)
		q = q.Filter(sqlchemy.In(q.Field("zone_id"), subq))
	}
	if zone != nil {
		q = q.Equals("zone_id", zone.Id)
	}
	if len(domainId) > 0 {
		ownerId := &db.SOwnerId{DomainId: domainId}
		q = HostManager.FilterByOwner(q, ownerId, rbacutils.ScopeDomain)
		/*subq := getDomainManagerSubq(domainId)
		q = q.Filter(sqlchemy.OR(
			sqlchemy.In(q.Field("manager_id"), subq),
			sqlchemy.IsNullOrEmpty(q.Field("manager_id")),
		))*/
	}
	q = q.IsNotEmpty("resource_type").IsNotNull("resource_type")
	q = q.IsTrue("enabled")
	q = q.Distinct()
	rows, err := q.Rows()
	if err != nil {
		return nil
	}
	defer rows.Close()
	resourceTypes := make([]string, 0)
	for rows.Next() {
		var resType string
		var managerId string
		rows.Scan(&resType, &managerId)
		if len(resType) > 0 && IsProviderAccountEnabled(managerId) {
			if !utils.IsInStringArray(resType, resourceTypes) {
				resourceTypes = append(resourceTypes, resType)
			}
		}
	}
	return resourceTypes
}

type StorageInfo struct {
	Id              string
	Name            string
	VirtualCapacity int64
	Capacity        int64
	Reserved        sql.NullInt64
	StorageType     string
	MediumType      string
	Cmtbound        sql.NullFloat64
	UsedCapacity    sql.NullInt64
	WasteCapacity   sql.NullInt64
	FreeCapacity    int64
	IsSysDiskStore  bool
	HostType        string
}

type sStorage struct {
	Id   string
	Name string
}

type SimpleStorageInfo struct {
	Storages []sStorage

	VirtualCapacity int64
	Capacity        int64
	Reserved        int64
	UsedCapacity    int64
	WasteCapacity   int64
	FreeCapacity    int64
	IsSysDiskStore  bool
}

func getStorageTypes(
	region *SCloudregion, zone *SZone, domainId string,
) (
	[]string, []string,
	map[string][]string, map[string]map[string]*SimpleStorageInfo,
	map[string][]string, map[string]map[string]*SimpleStorageInfo,
) {
	storages := StorageManager.Query().SubQuery()
	disks1 := DiskManager.Query().SubQuery()
	usedDisk := disks1.Query(
		disks1.Field("storage_id"),
		sqlchemy.SUM("used_capacity", disks1.Field("disk_size")),
	).Equals("status", api.DISK_READY).GroupBy("storage_id").SubQuery()
	disks2 := DiskManager.Query().SubQuery()
	failedDisk := disks2.Query(
		disks2.Field("storage_id"),
		sqlchemy.SUM("waste_capacity", disks2.Field("disk_size")),
	).NotEquals("status", api.DISK_READY).GroupBy("storage_id").SubQuery()

	hostStorages := HoststorageManager.Query().SubQuery()
	hostQuery := HostManager.Query()
	if len(domainId) > 0 {
		ownerId := &db.SOwnerId{DomainId: domainId}
		hostQuery = HostManager.FilterByOwner(hostQuery, ownerId, rbacutils.ScopeDomain)
	}
	hosts := hostQuery.SubQuery()

	q := storages.Query(
		storages.Field("id"),
		storages.Field("name"),
		storages.Field("capacity"),
		storages.Field("reserved"),
		storages.Field("storage_type"),
		storages.Field("medium_type"),
		storages.Field("cmtbound"),
		usedDisk.Field("used_capacity"),
		failedDisk.Field("waste_capacity"),
		storages.Field("is_sys_disk_store"),
		hosts.Field("host_type"),
	)
	q = q.LeftJoin(usedDisk, sqlchemy.Equals(usedDisk.Field("storage_id"), storages.Field("id")))
	q = q.LeftJoin(failedDisk, sqlchemy.Equals(failedDisk.Field("storage_id"), storages.Field("id")))

	q = q.Join(hostStorages, sqlchemy.Equals(
		hostStorages.Field("storage_id"),
		storages.Field("id"),
	))
	q = q.Join(hosts, sqlchemy.Equals(
		hosts.Field("id"),
		hostStorages.Field("host_id"),
	))
	if region != nil {
		subq := getRegionZoneSubq(region)
		q = q.Filter(sqlchemy.In(storages.Field("zone_id"), subq))
	}
	if zone != nil {
		q = q.Filter(sqlchemy.Equals(storages.Field("zone_id"), zone.Id))
	}
	if len(domainId) > 0 {
		ownerId := &db.SOwnerId{DomainId: domainId}
		q = StorageManager.FilterByOwner(q, ownerId, rbacutils.ScopeDomain)
	}
	q = q.Filter(sqlchemy.Equals(hosts.Field("resource_type"), api.HostResourceTypeShared))
	q = q.Filter(sqlchemy.IsNotEmpty(storages.Field("storage_type")))
	q = q.Filter(sqlchemy.IsNotNull(storages.Field("storage_type")))
	q = q.Filter(sqlchemy.IsNotEmpty(storages.Field("medium_type")))
	q = q.Filter(sqlchemy.IsNotNull(storages.Field("medium_type")))
	q = q.Filter(sqlchemy.In(storages.Field("status"), []string{api.STORAGE_ENABLED, api.STORAGE_ONLINE}))
	q = q.Filter(sqlchemy.IsTrue(storages.Field("enabled")))
	q = q.Filter(sqlchemy.NotEquals(hosts.Field("host_type"), api.HOST_TYPE_BAREMETAL))
	rows, err := q.Rows()
	if err != nil {
		log.Errorf("get storage types failed %s", err)
		return nil, nil, nil, nil, nil, nil
	}
	defer rows.Close()

	var (
		sysStorageTypes           = make([]string, 0)
		allStorageTypes           = make([]string, 0)
		storageInfos              = make(map[string]*SimpleStorageInfo)
		sysHypervisorStorageTypes = make(map[string][]string)
		allHypervisorStorageTypes = make(map[string][]string)
		sysHypervisorStorageInfos = make(map[string]map[string]*SimpleStorageInfo)
		allHypervisorStorageInfos = make(map[string]map[string]*SimpleStorageInfo)

		setStorageTypes = func(storageHypervisor, storageType string, hypervisorStorageTypes map[string][]string) {
			sts, ok := hypervisorStorageTypes[storageHypervisor]
			if !ok {
				sts = make([]string, 0)
			}

			if !utils.IsInStringArray(storageType, sts) {
				sts = append(sts, storageType)
			}

			hypervisorStorageTypes[storageHypervisor] = sts
		}
		addStorageInfo = func(storage *StorageInfo, simpleStorage *SimpleStorageInfo) {
			simpleStorage.VirtualCapacity += storage.VirtualCapacity
			simpleStorage.FreeCapacity += storage.FreeCapacity
			simpleStorage.Reserved += storage.Reserved.Int64
			simpleStorage.Capacity += storage.Capacity
			simpleStorage.WasteCapacity += storage.WasteCapacity.Int64
			simpleStorage.UsedCapacity += storage.UsedCapacity.Int64
			simpleStorage.Storages = append(simpleStorage.Storages, sStorage{Id: storage.Id, Name: storage.Name})
		}
		setStorageInfos = func(storageHypervisor, storageType string, storage *StorageInfo,
			hypervisorStorageInfos map[string]map[string]*SimpleStorageInfo) bool {
			var notFound bool
			sfs, ok := hypervisorStorageInfos[storageHypervisor]
			if !ok {
				sfs = make(map[string]*SimpleStorageInfo)
				notFound = true
			}
			simpleStorage, ok := sfs[storageType]
			if !ok {
				notFound = true
				simpleStorage = &SimpleStorageInfo{Storages: []sStorage{}}
			}
			if !utils.IsInStringArray(storageHypervisor, api.PUBLIC_CLOUD_HYPERVISORS) {
				addStorageInfo(storage, simpleStorage)
				sfs[storageType] = simpleStorage
				hypervisorStorageInfos[storageHypervisor] = sfs
			}
			return notFound
		}
	)

	for rows.Next() {
		var storage StorageInfo
		err := rows.Scan(
			&storage.Id, &storage.Name,
			&storage.Capacity, &storage.Reserved,
			&storage.StorageType, &storage.MediumType,
			&storage.Cmtbound, &storage.UsedCapacity,
			&storage.WasteCapacity, &storage.IsSysDiskStore,
			&storage.HostType,
		)
		if err != nil {
			log.Errorf("Scan storage rows %s", err)
			return nil, nil, nil, nil, nil, nil
		}
		storageHypervisor := api.HOSTTYPE_HYPERVISOR[storage.HostType]
		if len(storage.StorageType) > 0 && len(storage.MediumType) > 0 {
			storageType := fmt.Sprintf("%s/%s", storage.StorageType, storage.MediumType)
			simpleStorage, ok := storageInfos[storageType]
			if !ok {
				simpleStorage = &SimpleStorageInfo{Storages: []sStorage{}}
				if storage.IsSysDiskStore {
					sysStorageTypes = append(sysStorageTypes, storageType)
				}
				allStorageTypes = append(allStorageTypes, storageType)
			}
			if storage.Cmtbound.Float64 == 0 {
				storage.Cmtbound.Float64 = float64(options.Options.DefaultStorageOvercommitBound)
			}
			storage.VirtualCapacity = int64(float64(storage.Capacity-storage.Reserved.Int64) * storage.Cmtbound.Float64)
			storage.FreeCapacity = storage.VirtualCapacity - storage.UsedCapacity.Int64 - storage.WasteCapacity.Int64
			addStorageInfo(&storage, simpleStorage)
			storageInfos[storageType] = simpleStorage

			// set hypervisor storage types and infos
			if storage.IsSysDiskStore {
				if setStorageInfos(storageHypervisor, storageType, &storage, sysHypervisorStorageInfos) {
					setStorageTypes(storageHypervisor, storageType, sysHypervisorStorageTypes)
				}
			}
			if setStorageInfos(storageHypervisor, storageType, &storage, allHypervisorStorageInfos) {
				setStorageTypes(storageHypervisor, storageType, allHypervisorStorageTypes)
			}
		}
	}
	return sysStorageTypes, allStorageTypes,
		sysHypervisorStorageTypes, sysHypervisorStorageInfos,
		allHypervisorStorageTypes, allHypervisorStorageInfos
}

func getGPUs(region *SCloudregion, zone *SZone, domainId string) []string {
	devices := IsolatedDeviceManager.Query().SubQuery()
	hostQuery := HostManager.Query()
	if len(domainId) > 0 {
		ownerId := &db.SOwnerId{DomainId: domainId}
		hostQuery = StorageManager.FilterByOwner(hostQuery, ownerId, rbacutils.ScopeDomain)
	}
	hosts := hostQuery.SubQuery()

	q := devices.Query(devices.Field("model"))
	if region != nil {
		subq := getRegionZoneSubq(region)
		q = q.Join(hosts, sqlchemy.Equals(devices.Field("host_id"), hosts.Field("id")))
		q = q.Filter(sqlchemy.In(hosts.Field("zone_id"), subq))
	}
	if zone != nil {
		q = q.Join(hosts, sqlchemy.Equals(devices.Field("host_id"), hosts.Field("id")))
		q = q.Filter(sqlchemy.Equals(hosts.Field("zone_id"), zone.Id))
	}
	/*if len(domainId) > 0 {
		subq := getDomainManagerSubq(domainId)
		q = q.Filter(sqlchemy.OR(
			sqlchemy.In(hosts.Field("manager_id"), subq),
			sqlchemy.IsNullOrEmpty(hosts.Field("manager_id")),
		))
	}*/
	q = q.Distinct()

	rows, err := q.Rows()
	if err != nil {
		return nil
	}
	defer rows.Close()
	gpus := make([]string, 0)
	for rows.Next() {
		var model string
		rows.Scan(&model)
		if len(model) > 0 {
			gpus = append(gpus, model)
		}
	}
	return gpus
}

func getHostCpuArchs(region *SCloudregion, zone *SZone, domainId string) []string {
	q := HostManager.Query("cpu_architecture").Equals("enabled", true).
		Equals("host_status", "online").Equals("host_type", api.HOST_TYPE_HYPERVISOR)
	if len(domainId) > 0 {
		ownerId := &db.SOwnerId{DomainId: domainId}
		q = HostManager.FilterByOwner(q, ownerId, rbacutils.ScopeDomain)
	}
	if zone != nil {
		q = q.Equals("zone_id", zone.Id)
	}
	if region != nil {
		subq := ZoneManager.Query("id").Equals("cloudregion_id", region.Id).SubQuery()
		q = q.Filter(sqlchemy.In(q.Field("zone_id"), subq))
	}
	q = q.Distinct()
	type CpuArch struct {
		CpuArchitecture string
	}
	archs := make([]CpuArch, 0)
	if err := q.All(&archs); err != nil && err != sql.ErrNoRows {
		log.Errorf("failed fetch host cpu archs %s", err)
		return nil
	}
	if len(archs) == 0 {
		return nil
	}
	res := make([]string, len(archs))
	for i := 0; i < len(archs); i++ {
		res[i] = archs[i].CpuArchitecture
	}
	return res
}

func getNetworkCount(ownerId mcclient.IIdentityProvider, scope rbacutils.TRbacScope, region *SCloudregion, zone *SZone) (int, error) {
	return getNetworkCountByFilter(ownerId, scope, region, zone, tristate.None, "")
}

func getAutoAllocNetworkCount(ownerId mcclient.IIdentityProvider, scope rbacutils.TRbacScope, region *SCloudregion, zone *SZone, serverType string) (int, error) {
	return getNetworkCountByFilter(ownerId, scope, region, zone, tristate.True, serverType)
}

func getNetworkCountByFilter(ownerId mcclient.IIdentityProvider, scope rbacutils.TRbacScope, region *SCloudregion, zone *SZone, isAutoAlloc tristate.TriState, serverType string) (int, error) {
	if zone != nil && region == nil {
		region = zone.GetRegion()
	}

	networks := NetworkManager.Query().SubQuery()

	q := networks.Query()

	if zone != nil && !utils.IsInStringArray(region.Provider, api.REGIONAL_NETWORK_PROVIDERS) {
		wires := WireManager.Query("id").Equals("zone_id", zone.Id)
		q = q.In("wire_id", wires.SubQuery())
	} else if region != nil {
		if utils.IsInStringArray(region.Provider, api.REGIONAL_NETWORK_PROVIDERS) {
			wires := WireManager.Query().SubQuery()
			vpcs := VpcManager.Query().SubQuery()
			subq := wires.Query(wires.Field("id")).
				Join(vpcs, sqlchemy.Equals(wires.Field("vpc_id"), vpcs.Field("id"))).
				Filter(sqlchemy.Equals(vpcs.Field("cloudregion_id"), region.Id))
			q = q.Filter(sqlchemy.In(q.Field("wire_id"), subq))
		} else {
			subq := getRegionZoneSubq(region)
			wires := WireManager.Query("id").In("zone_id", subq)
			q = q.In("wire_id", wires.SubQuery())
		}
	}

	q = NetworkManager.FilterByOwner(q, ownerId, scope)
	if !isAutoAlloc.IsNone() {
		if isAutoAlloc.IsTrue() {
			q = q.IsTrue("is_auto_alloc")
		} else {
			q = q.IsFalse("is_auto_alloc")
		}
	}
	if len(serverType) > 0 {
		q = q.Filter(sqlchemy.Equals(networks.Field("server_type"), serverType))
	}
	q = q.Filter(sqlchemy.Equals(networks.Field("status"), api.NETWORK_STATUS_AVAILABLE))

	return q.CountWithError()
}

func isSchedPolicySupported(region *SCloudregion, zone *SZone) bool {
	return true
}

func getMinNicCount(region *SCloudregion, zone *SZone) int {
	if region != nil {
		return region.getMinNicCount()
	}
	if zone != nil {
		return zone.getMinNicCount()
	}
	return 0
}

func getMaxNicCount(region *SCloudregion, zone *SZone) int {
	if region != nil {
		return region.getMaxNicCount()
	}
	if zone != nil {
		return zone.getMaxNicCount()
	}
	return 0
}

func getMinDataDiskCount(region *SCloudregion, zone *SZone) int {
	if region != nil {
		return region.getMinDataDiskCount()
	}
	if zone != nil {
		return zone.getMinDataDiskCount()
	}
	return 0
}

func getMaxDataDiskCount(region *SCloudregion, zone *SZone) int {
	if region != nil {
		return region.getMaxDataDiskCount()
	}
	if zone != nil {
		return zone.getMaxDataDiskCount()
	}
	return 0
}

func isUsable(ownerId mcclient.IIdentityProvider, scope rbacutils.TRbacScope, region *SCloudregion, zone *SZone) bool {
	cnt, err := getNetworkCount(ownerId, scope, region, zone)
	if err != nil {
		return false
	}
	if cnt > 0 {
		return true
	} else {
		return false
	}
}
