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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SCapabilities struct {
	Hypervisors    []string            `json:",allowempty"`
	HypervisorInfo map[string][]string `json:",allowempty"`

	Brands                           []string `json:",allowempty"`
	EnabledBrands                    []string `json:",allowempty"`
	DisabledBrands                   []string `json:",allowempty"`
	ComputeEngineBrands              []string `json:",allowempty"`
	DisabledComputeEngineBrands      []string `json:",allowempty"`
	RdsEngineBrands                  []string `json:",allowempty"`
	DisabledRdsEngineBrands          []string `json:",allowempty"`
	RedisEngineBrands                []string `json:",allowempty"`
	DisabledRedisEngineBrands        []string `json:",allowempty"`
	LoadbalancerEngineBrands         []string `json:",allowempty"`
	DisabledLoadbalancerEngineBrands []string `json:",allowempty"`
	CloudIdBrands                    []string `json:",allowempty"`
	DisabledCloudIdBrands            []string `json:",allowempty"`
	// 支持SAML 2.0
	SamlAuthBrands               []string `json:",allowempty"`
	DisabledSamlAuthBrands       []string `json:",allowempty"`
	NatBrands                    []string `json:",allowempty"`
	DisabledNatBrands            []string `json:",allowempty"`
	NasBrands                    []string `json:",allowempty"`
	DisabledNasBrands            []string `json:",allowempty"`
	WafBrands                    []string `json:",allowempty"`
	DisabledWafBrands            []string `json:",allowempty"`
	CdnBrands                    []string `json:",allowempty"`
	DisabledCdnBrands            []string `json:",allowempty"`
	PublicIpBrands               []string `json:",allowempty"`
	DisabledPublicIpBrands       []string `json:",allowempty"`
	NetworkManageBrands          []string `json:",allowempty"`
	DisabledNetworkManageBrands  []string `json:",allowempty"`
	ObjectStorageBrands          []string `json:",allowempty"`
	DisabledObjectStorageBrands  []string `json:",allowempty"`
	DisabledModelartsPoolsBrands []string `json:",allowempty"`
	ModelartsPoolsBrands         []string `json:",allowempty"`

	ContainerBrands         []string `json:",allowempty"`
	DisabledContainerBrands []string `json:",allowempty"`

	VpcPeerBrands         []string `json:",allowempty"`
	DisabledVpcPeerBrands []string `json:",allowempty"`

	SecurityGroupBrands         []string `json:",allowempty"`
	DisabledSecurityGroupBrands []string `json:",allowempty"`

	SnapshotPolicyBrands         []string `json:",allowempty"`
	DisabledSnapshotPolicyBrands []string `json:",allowempty"`

	ReadOnlyBrands                           []string `json:",allowempty"`
	ReadOnlyDisabledBrands                   []string `json:",allowempty"`
	ReadOnlyComputeEngineBrands              []string `json:",allowempty"`
	ReadOnlyDisabledComputeEngineBrands      []string `json:",allowempty"`
	ReadOnlyRdsEngineBrands                  []string `json:",allowempty"`
	ReadOnlyDisabledRdsEngineBrands          []string `json:",allowempty"`
	ReadOnlyRedisEngineBrands                []string `json:",allowempty"`
	ReadOnlyDisabledRedisEngineBrands        []string `json:",allowempty"`
	ReadOnlyLoadbalancerEngineBrands         []string `json:",allowempty"`
	ReadOnlyDisabledLoadbalancerEngineBrands []string `json:",allowempty"`
	ReadOnlyCloudIdBrands                    []string `json:",allowempty"`
	ReadOnlyDisabledCloudIdBrands            []string `json:",allowempty"`
	// 支持SAML 2.0
	ReadOnlySamlAuthBrands               []string `json:",allowempty"`
	ReadOnlyDisabledSamlAuthBrands       []string `json:",allowempty"`
	ReadOnlyNatBrands                    []string `json:",allowempty"`
	ReadOnlyDisabledNatBrands            []string `json:",allowempty"`
	ReadOnlyNasBrands                    []string `json:",allowempty"`
	ReadOnlyDisabledNasBrands            []string `json:",allowempty"`
	ReadOnlyWafBrands                    []string `json:",allowempty"`
	ReadOnlyDisabledWafBrands            []string `json:",allowempty"`
	ReadOnlyCdnBrands                    []string `json:",allowempty"`
	ReadOnlyDisabledCdnBrands            []string `json:",allowempty"`
	ReadOnlyPublicIpBrands               []string `json:",allowempty"`
	ReadOnlyDisabledPublicIpBrands       []string `json:",allowempty"`
	ReadOnlyNetworkManageBrands          []string `json:",allowempty"`
	ReadOnlyDisabledNetworkManageBrands  []string `json:",allowempty"`
	ReadOnlyObjectStorageBrands          []string `json:",allowempty"`
	ReadOnlyDisabledObjectStorageBrands  []string `json:",allowempty"`
	ReadOnlyModelartsPoolsBrands         []string `json:",allowempty"`
	ReadOnlyDisabledModelartsPoolsBrands []string `json:",allowempty"`

	ReadOnlyContainerBrands         []string `json:",allowempty"`
	ReadOnlyDisabledContainerBrands []string `json:",allowempty"`

	ReadOnlyVpcPeerBrands         []string `json:",allowempty"`
	ReadOnlyDisabledVpcPeerBrands []string `json:",allowempty"`

	ReadOnlySecurityGroupBrands         []string `json:",allowempty"`
	ReadOnlyDisabledSecurityGroupBrands []string `json:",allowempty"`

	ReadOnlySnapshotPolicyBrands         []string `json:",allowempty"`
	ReadOnlyDisabledSnapshotPolicyBrands []string `json:",allowempty"`

	ResourceTypes      []string           `json:",allowempty"`
	GPUModels          []string           `json:",allowempty"` // Deprecated by PCIModelTypes
	PCIModelTypes      []PCIDevModelTypes `json:",allowempty"`
	HostCpuArchs       []string           `json:",allowempty"` // x86_64 aarch64
	MinNicCount        int
	MaxNicCount        int
	MinDataDiskCount   int
	MaxDataDiskCount   int
	SchedPolicySupport bool
	Usable             bool

	// Deprecated
	PublicNetworkCount int

	AutoAllocNetworkCount int
	DBInstance            map[string]map[string]map[string][]string //map[engine][engineVersion][category][]{storage_type}
	Specs                 jsonutils.JSONObject
	AvailableHostCount    int

	*StorageInfos
	InstanceCapabilities []cloudprovider.SInstanceCapability
}

type StorageInfos struct {
	StorageTypes2     map[string][]string                      `json:",allowempty"`
	StorageTypes3     map[string]map[string]*SimpleStorageInfo `json:",allowempty"`
	DataStorageTypes2 map[string][]string                      `json:",allowempty"`
	DataStorageTypes3 map[string]map[string]*SimpleStorageInfo `json:",allowempty"`

	SystemStorageTypes map[string]map[string]map[string]*SimpleStorageInfo `json:",allowempty"`
	DataStorageTypes   map[string]map[string]map[string]*SimpleStorageInfo `json:",allowempty"`
}

func GetDiskCapabilities(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, region *SCloudregion, zone *SZone) (SCapabilities, error) {
	capa := SCapabilities{}
	var err error
	capa.StorageInfos, err = getStorageTypes(ctx, userCred, region, zone, "")
	if err != nil {
		return capa, errors.Wrapf(err, "getStorageTypes")
	}
	capa.MinDataDiskCount = getMinDataDiskCount(region, zone)
	capa.MaxDataDiskCount = getMaxDataDiskCount(region, zone)
	return capa, nil
}

func GetCapabilities(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, region *SCloudregion, zone *SZone) (SCapabilities, error) {
	capa := SCapabilities{}
	var ownerId mcclient.IIdentityProvider
	scopeStr := jsonutils.GetAnyString(query, []string{"scope"})
	scope := rbacscope.String2Scope(scopeStr)
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
		scope = rbacscope.ScopeDomain
	} else {
		domainId = userCred.GetProjectDomainId()
		ownerId = userCred
	}
	if scope == rbacscope.ScopeSystem {
		result := policy.PolicyManager.Allow(scope, userCred, consts.GetServiceType(), "capabilities", policy.PolicyActionList)
		if result.Result.IsDeny() {
			return capa, httperrors.NewForbiddenError("not allow to query system capability")
		}
		domainId = ""
	}
	var err error
	capa.HypervisorInfo, err = getHypervisors(ctx, userCred, region, zone, domainId)
	if err != nil {
		return capa, errors.Wrapf(err, "getHypervisors")
	}
	capa.Hypervisors = []string{}
	for _, hypervisors := range capa.HypervisorInfo {
		for _, h := range hypervisors {
			if !utils.IsInStringArray(h, capa.Hypervisors) {
				capa.Hypervisors = append(capa.Hypervisors, h)
			}
		}
	}
	capa.InstanceCapabilities = []cloudprovider.SInstanceCapability{}
	for provider, hypervisors := range capa.HypervisorInfo {
		for _, hypervisor := range hypervisors {
			driver, _ := GetDriver(hypervisor, provider)
			if driver != nil {
				capa.InstanceCapabilities = append(capa.InstanceCapabilities, driver.GetInstanceCapability())
			}
		}
	}
	if zone != nil {
		region, err = zone.GetRegion()
		if err != nil {
			return capa, errors.Wrapf(err, "GetRegion")
		}
	}
	getBrands(region, domainId, &capa)
	capa.ResourceTypes = getResourceTypes(ctx, userCred, region, zone, domainId)
	capa.StorageInfos, err = getStorageTypes(ctx, userCred, region, zone, domainId)
	if err != nil {
		return capa, errors.Wrapf(err, "getStorageTypes")
	}
	capa.GPUModels, capa.PCIModelTypes = getIsolatedDeviceInfo(ctx, userCred, region, zone, domainId)
	capa.SchedPolicySupport = isSchedPolicySupported(region, zone)
	capa.MinNicCount = getMinNicCount(region, zone)
	capa.MaxNicCount = getMaxNicCount(region, zone)
	capa.MinDataDiskCount = getMinDataDiskCount(region, zone)
	capa.MaxDataDiskCount = getMaxDataDiskCount(region, zone)
	capa.DBInstance = getDBInstanceInfo(region, zone)
	capa.Usable = isUsable(ctx, userCred, ownerId, scope, region, zone)
	capa.HostCpuArchs = getHostCpuArchs(ctx, userCred, region, zone, domainId)
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
	serverType := jsonutils.GetAnyString(query, []string{"host_type", "server_type"})
	autoAllocNetworkCount, _ := getAutoAllocNetworkCount(ctx, userCred, ownerId, scope, region, zone, serverType)
	capa.PublicNetworkCount = autoAllocNetworkCount
	capa.AutoAllocNetworkCount = autoAllocNetworkCount
	mans := []ISpecModelManager{HostManager, IsolatedDeviceManager}
	capa.Specs, err = GetModelsSpecs(ctx, userCred, query.(*jsonutils.JSONDict), mans...)
	if err != nil {
		return capa, err
	}
	capa.AvailableHostCount, err = GetAvailableHostCount(ctx, userCred, query.(*jsonutils.JSONDict))
	return capa, err
}

func GetAvailableHostCount(ctx context.Context, userCred mcclient.TokenCredential, query *jsonutils.JSONDict) (int, error) {
	zoneStr, _ := query.GetString("zone")
	izone, _ := ZoneManager.FetchByIdOrName(ctx, userCred, zoneStr)
	var zoneId string
	if izone != nil {
		zoneId = izone.GetId()
	}

	regionStr, _ := query.GetString("region")
	iregion, _ := CloudregionManager.FetchByIdOrName(ctx, userCred, regionStr)
	var regionId string
	if iregion != nil {
		regionId = iregion.GetId()
	}

	domainId, _ := query.GetString("domain_id")
	q := HostManager.Query().Equals("enabled", true).
		Equals("host_status", "online").Equals("host_type", api.HOST_TYPE_HYPERVISOR)
	if len(domainId) > 0 {
		ownerId := &db.SOwnerId{DomainId: domainId}
		q = HostManager.FilterByOwner(ctx, q, HostManager, userCred, ownerId, rbacscope.ScopeDomain)
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
		region, _ = zone.GetRegion()
	}
	if region == nil || !region.GetDriver().IsSupportedDBInstance() {
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
func getBrands(region *SCloudregion, domainId string, capa *SCapabilities) {
	brands, err := CloudaccountManager.getBrandsOfCapability(region, domainId)
	if err != nil {
		log.Errorf("getBrandsOfCapability: %v", err)
	}
	brandMaps := map[string]map[string]bool{}
	for _, brand := range brands {
		_, ok := brandMaps[brand.Brand]
		if !ok {
			brandMaps[brand.Brand] = map[string]bool{}
		}
		_, ok = brandMaps[brand.Brand][brand.Capability]
		if !ok {
			brandMaps[brand.Brand][brand.Capability] = brand.Enabled
		}
		if brand.Enabled {
			brandMaps[brand.Brand][brand.Capability] = true
		}
	}

	if region == nil || region.Provider == api.ONECLOUD_BRAND_ONECLOUD {
		if utils.IsInStringArray(api.HYPERVISOR_KVM, capa.Hypervisors) || utils.IsInStringArray(api.HYPERVISOR_BAREMETAL, capa.Hypervisors) {
			capa.Brands = append(capa.Brands, api.ONECLOUD_BRAND_ONECLOUD)
			capa.SecurityGroupBrands = append(capa.SecurityGroupBrands, api.ONECLOUD_BRAND_ONECLOUD)
			capa.ComputeEngineBrands = append(capa.ComputeEngineBrands, api.ONECLOUD_BRAND_ONECLOUD)
			capa.SnapshotPolicyBrands = append(capa.SnapshotPolicyBrands, api.ONECLOUD_BRAND_ONECLOUD)
		} else if utils.IsInStringArray(api.HYPERVISOR_POD, capa.Hypervisors) {
			capa.Brands = append(capa.Brands, api.ONECLOUD_BRAND_ONECLOUD)
			capa.ComputeEngineBrands = append(capa.ComputeEngineBrands, api.ONECLOUD_BRAND_ONECLOUD)
		}

		if count, _ := LoadbalancerClusterManager.Query().Limit(1).CountWithError(); count > 0 {
			capa.LoadbalancerEngineBrands = append(capa.LoadbalancerEngineBrands, api.ONECLOUD_BRAND_ONECLOUD)
		}

		capa.NetworkManageBrands = append(capa.NetworkManageBrands, api.ONECLOUD_BRAND_ONECLOUD)
	}

	capa.EnabledBrands = []string{}
	capa.DisabledBrands = []string{}
	var appendBrand = func(enabled *[]string, disabled *[]string, readOnlyEnabled *[]string, readOnlyDisabled *[]string, brand, capability string, isEnable, readOnly bool) {
		if !utils.IsInArray(brand, capa.Brands) {
			capa.Brands = append(capa.Brands, brand)
		}
		if readOnly {
			if isEnable {
				if !utils.IsInArray(brand, *readOnlyEnabled) {
					*readOnlyEnabled = append(*readOnlyEnabled, brand)
				}
				if !utils.IsInArray(brand, capa.ReadOnlyBrands) {
					capa.ReadOnlyBrands = append(capa.ReadOnlyBrands, brand)
				}
			} else {
				if !utils.IsInArray(brand, *readOnlyDisabled) {
					*readOnlyDisabled = append(*readOnlyDisabled, brand)
				}
				if !utils.IsInArray(brand, capa.ReadOnlyDisabledBrands) {
					capa.ReadOnlyDisabledBrands = append(capa.ReadOnlyDisabledBrands, brand)
				}
			}
		} else {
			if isEnable {
				if !utils.IsInArray(brand, *enabled) {
					*enabled = append(*enabled, brand)
				}
				if !utils.IsInArray(brand, capa.EnabledBrands) {
					capa.EnabledBrands = append(capa.EnabledBrands, brand)
				}
			} else {
				if !utils.IsInArray(brand, *disabled) {
					*disabled = append(*disabled, brand)
				}
				if !utils.IsInArray(brand, capa.DisabledBrands) {
					capa.DisabledBrands = append(capa.DisabledBrands, brand)
				}
			}
		}
	}

	for brand, info := range brandMaps {
		for capability, enabled := range info {
			readOnly := false
			if strings.HasSuffix(capability, cloudprovider.READ_ONLY_SUFFIX) {
				readOnly = true
				capability = strings.TrimSuffix(capability, cloudprovider.READ_ONLY_SUFFIX)
			}
			switch capability {
			case cloudprovider.CLOUD_CAPABILITY_COMPUTE:
				appendBrand(&capa.ComputeEngineBrands, &capa.DisabledComputeEngineBrands, &capa.ReadOnlyComputeEngineBrands, &capa.ReadOnlyDisabledComputeEngineBrands, brand, capability, enabled, readOnly)
			case cloudprovider.CLOUD_CAPABILITY_RDS:
				appendBrand(&capa.RdsEngineBrands, &capa.DisabledRdsEngineBrands, &capa.ReadOnlyRdsEngineBrands, &capa.ReadOnlyDisabledRdsEngineBrands, brand, capability, enabled, readOnly)
			case cloudprovider.CLOUD_CAPABILITY_CACHE:
				appendBrand(&capa.RedisEngineBrands, &capa.DisabledRedisEngineBrands, &capa.ReadOnlyRedisEngineBrands, &capa.ReadOnlyDisabledRedisEngineBrands, brand, capability, enabled, readOnly)
			case cloudprovider.CLOUD_CAPABILITY_NETWORK:
				appendBrand(&capa.NetworkManageBrands, &capa.DisabledNetworkManageBrands, &capa.ReadOnlyNetworkManageBrands, &capa.ReadOnlyDisabledNetworkManageBrands, brand, capability, enabled, readOnly)
			case cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE:
				appendBrand(&capa.ObjectStorageBrands, &capa.DisabledObjectStorageBrands, &capa.ReadOnlyObjectStorageBrands, &capa.ReadOnlyDisabledObjectStorageBrands, brand, capability, enabled, readOnly)
			case cloudprovider.CLOUD_CAPABILITY_CLOUDID:
				appendBrand(&capa.CloudIdBrands, &capa.DisabledCloudIdBrands, &capa.ReadOnlyCloudIdBrands, &capa.ReadOnlyDisabledCloudIdBrands, brand, capability, enabled, readOnly)
			case cloudprovider.CLOUD_CAPABILITY_PUBLIC_IP:
				appendBrand(&capa.PublicIpBrands, &capa.DisabledPublicIpBrands, &capa.ReadOnlyPublicIpBrands, &capa.ReadOnlyDisabledPublicIpBrands, brand, capability, enabled, readOnly)
			case cloudprovider.CLOUD_CAPABILITY_LOADBALANCER:
				appendBrand(&capa.LoadbalancerEngineBrands, &capa.DisabledLoadbalancerEngineBrands, &capa.ReadOnlyLoadbalancerEngineBrands, &capa.ReadOnlyDisabledLoadbalancerEngineBrands, brand, capability, enabled, readOnly)
			case cloudprovider.CLOUD_CAPABILITY_SAML_AUTH:
				appendBrand(&capa.SamlAuthBrands, &capa.DisabledSamlAuthBrands, &capa.ReadOnlySamlAuthBrands, &capa.ReadOnlyDisabledSamlAuthBrands, brand, capability, enabled, readOnly)
			case cloudprovider.CLOUD_CAPABILITY_NAT:
				appendBrand(&capa.NatBrands, &capa.DisabledNatBrands, &capa.ReadOnlyNatBrands, &capa.ReadOnlyDisabledNatBrands, brand, capability, enabled, readOnly)
			case cloudprovider.CLOUD_CAPABILITY_NAS:
				appendBrand(&capa.NasBrands, &capa.DisabledNasBrands, &capa.ReadOnlyNasBrands, &capa.ReadOnlyDisabledNasBrands, brand, capability, enabled, readOnly)
			case cloudprovider.CLOUD_CAPABILITY_WAF:
				appendBrand(&capa.WafBrands, &capa.DisabledWafBrands, &capa.ReadOnlyWafBrands, &capa.ReadOnlyDisabledWafBrands, brand, capability, enabled, readOnly)
			case cloudprovider.CLOUD_CAPABILITY_CDN:
				appendBrand(&capa.CdnBrands, &capa.DisabledCdnBrands, &capa.ReadOnlyCdnBrands, &capa.ReadOnlyDisabledCdnBrands, brand, capability, enabled, readOnly)
			case cloudprovider.CLOUD_CAPABILITY_CONTAINER:
				appendBrand(&capa.ContainerBrands, &capa.DisabledContainerBrands, &capa.ReadOnlyContainerBrands, &capa.ReadOnlyDisabledContainerBrands, brand, capability, enabled, readOnly)
			case cloudprovider.CLOUD_CAPABILITY_VPC_PEER:
				appendBrand(&capa.VpcPeerBrands, &capa.DisabledVpcPeerBrands, &capa.ReadOnlyVpcPeerBrands, &capa.ReadOnlyDisabledVpcPeerBrands, brand, capability, enabled, readOnly)
			case cloudprovider.CLOUD_CAPABILITY_SECURITY_GROUP:
				appendBrand(&capa.SecurityGroupBrands, &capa.DisabledSecurityGroupBrands, &capa.ReadOnlySecurityGroupBrands, &capa.ReadOnlyDisabledSecurityGroupBrands, brand, capability, enabled, readOnly)
			case cloudprovider.CLOUD_CAPABILITY_SNAPSHOT_POLICY:
				appendBrand(&capa.SnapshotPolicyBrands, &capa.DisabledSnapshotPolicyBrands, &capa.ReadOnlySnapshotPolicyBrands, &capa.ReadOnlyDisabledSnapshotPolicyBrands, brand, capability, enabled, readOnly)
			case cloudprovider.CLOUD_CAPABILITY_MODELARTES:
				appendBrand(&capa.ModelartsPoolsBrands, &capa.DisabledModelartsPoolsBrands, &capa.ReadOnlyModelartsPoolsBrands, &capa.ReadOnlyDisabledModelartsPoolsBrands, brand, capability, enabled, readOnly)
			default:
			}
		}
	}

	return
}

func getHypervisors(ctx context.Context, userCred mcclient.TokenCredential, region *SCloudregion, zone *SZone, domainId string) (map[string][]string, error) {
	q := HostManager.Query().IsNotEmpty("host_type").IsTrue("enabled")
	if zone != nil {
		q = q.Equals("zone_id", zone.Id)
	}
	zoneQ := ZoneManager.Query()
	if region != nil {
		zoneQ = zoneQ.Equals("cloudregion_id", region.Id)
	}
	if len(domainId) > 0 {
		ownerId := &db.SOwnerId{DomainId: domainId}
		q = HostManager.FilterByOwner(ctx, q, HostManager, userCred, ownerId, rbacscope.ScopeDomain)
	}

	zones := zoneQ.SubQuery()
	regions := CloudregionManager.Query().SubQuery()

	sq := q.SubQuery()
	hQ := sq.Query(
		sq.Field("host_type"),
		regions.Field("provider"),
	)

	hQ = hQ.Join(zones, sqlchemy.Equals(hQ.Field("zone_id"), zones.Field("id")))
	hQ = hQ.Join(regions, sqlchemy.Equals(zones.Field("cloudregion_id"), regions.Field("id")))

	result := []struct {
		HostType string
		Provider string
	}{}

	hQ = hQ.Distinct()

	err := hQ.All(&result)
	if err != nil {
		return nil, err
	}
	ret := map[string][]string{}
	for _, h := range result {
		drv, err := GetHostDriver(h.HostType, h.Provider)
		if err != nil {
			return nil, errors.Wrapf(err, "GetHostDriver")
		}
		_, ok := ret[h.Provider]
		if !ok {
			ret[h.Provider] = []string{}
		}
		if !utils.IsInStringArray(drv.GetHypervisor(), ret[h.Provider]) {
			ret[h.Provider] = append(ret[h.Provider], drv.GetHypervisor())
		}
	}
	return ret, nil
}

func getResourceTypes(ctx context.Context, userCred mcclient.TokenCredential, region *SCloudregion, zone *SZone, domainId string) []string {
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
		q = HostManager.FilterByOwner(ctx, q, HostManager, userCred, ownerId, rbacscope.ScopeDomain)
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
	Provider        string
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
	ctx context.Context,
	userCred mcclient.TokenCredential,
	region *SCloudregion, zone *SZone, domainId string,
) (*StorageInfos, error) {
	storageQ := StorageManager.Query()
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
		hostQuery = HostManager.FilterByOwner(ctx, hostQuery, HostManager, userCred, ownerId, rbacscope.ScopeDomain)
	}
	hosts := hostQuery.SubQuery()

	zoneQ := ZoneManager.Query()
	if zone != nil {
		zoneQ = zoneQ.Equals("id", zone.Id)
	}
	if region != nil {
		zoneQ = zoneQ.Equals("cloudregion_id", region.Id)
	}
	zones := zoneQ.SubQuery()
	regions := CloudregionManager.Query().SubQuery()

	storages := storageQ.SubQuery()

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
		regions.Field("provider"),
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

	q = q.Join(zones, sqlchemy.Equals(
		storages.Field("zone_id"),
		zones.Field("id"),
	))

	q = q.Join(regions, sqlchemy.Equals(
		zones.Field("cloudregion_id"),
		regions.Field("id"),
	))

	if len(domainId) > 0 {
		ownerId := &db.SOwnerId{DomainId: domainId}
		q = StorageManager.FilterByOwner(ctx, q, StorageManager, userCred, ownerId, rbacscope.ScopeDomain)
	}
	q = q.Filter(sqlchemy.Equals(hosts.Field("resource_type"), api.HostResourceTypeShared))
	q = q.Filter(sqlchemy.IsNotEmpty(storages.Field("storage_type")))
	q = q.Filter(sqlchemy.IsNotNull(storages.Field("storage_type")))
	q = q.Filter(sqlchemy.IsNotEmpty(storages.Field("medium_type")))
	q = q.Filter(sqlchemy.IsNotNull(storages.Field("medium_type")))
	q = q.Filter(sqlchemy.In(storages.Field("status"), []string{api.STORAGE_ENABLED, api.STORAGE_ONLINE}))
	q = q.Filter(sqlchemy.IsTrue(storages.Field("enabled")))
	q = q.Filter(sqlchemy.NotEquals(hosts.Field("host_type"), api.HOST_TYPE_BAREMETAL))

	ret := &StorageInfos{
		StorageTypes2:     map[string][]string{},
		StorageTypes3:     map[string]map[string]*SimpleStorageInfo{},
		DataStorageTypes2: map[string][]string{},
		DataStorageTypes3: map[string]map[string]*SimpleStorageInfo{},

		SystemStorageTypes: map[string]map[string]map[string]*SimpleStorageInfo{},
		DataStorageTypes:   map[string]map[string]map[string]*SimpleStorageInfo{},
	}
	var (
		addStorageInfo = func(storage *StorageInfo, simpleStorage *SimpleStorageInfo) {
			simpleStorage.VirtualCapacity += storage.VirtualCapacity
			simpleStorage.FreeCapacity += storage.FreeCapacity
			simpleStorage.Reserved += storage.Reserved.Int64
			simpleStorage.Capacity += storage.Capacity
			simpleStorage.WasteCapacity += storage.WasteCapacity.Int64
			simpleStorage.UsedCapacity += storage.UsedCapacity.Int64
			simpleStorage.Storages = append(simpleStorage.Storages, sStorage{Id: storage.Id, Name: storage.Name})
		}
		setStorageInfos = func(hostDriver IHostDriver, storageType string, storage *StorageInfo,
			storageInfos map[string]map[string]*SimpleStorageInfo) {
			sfs, ok := storageInfos[hostDriver.GetHypervisor()]
			if !ok {
				sfs = make(map[string]*SimpleStorageInfo)
			}
			simpleStorage, ok := sfs[storageType]
			if !ok {
				simpleStorage = &SimpleStorageInfo{Storages: []sStorage{}}
			}
			if !utils.IsInStringArray(hostDriver.GetProvider(), api.PUBLIC_CLOUD_PROVIDERS) {
				addStorageInfo(storage, simpleStorage)
			}
			sfs[storageType] = simpleStorage
			storageInfos[hostDriver.GetHypervisor()] = sfs
		}

		setStorageInfos2 = func(hostDriver IHostDriver, storageType string, storage *StorageInfo,
			storageInfos2 map[string]map[string]map[string]*SimpleStorageInfo) {
			_, ok := storageInfos2[hostDriver.GetProvider()]
			if !ok {
				storageInfos2[hostDriver.GetProvider()] = make(map[string]map[string]*SimpleStorageInfo)
			}
			setStorageInfos(hostDriver, storageType, storage, storageInfos2[hostDriver.GetProvider()])
		}
	)

	info := []StorageInfo{}

	err := q.All(&info)
	if err != nil {
		return nil, errors.Wrapf(err, "q.All")
	}

	for i := range info {
		storage := info[i]
		if len(storage.Provider) == 0 {
			storage.Provider = api.CLOUD_PROVIDER_ONECLOUD
		}

		hostDriver, err := GetHostDriver(storage.HostType, storage.Provider)
		if err != nil {
			return nil, errors.Wrapf(err, "GetHostDriver")
		}
		if len(storage.StorageType) > 0 && len(storage.MediumType) > 0 {
			storageType := fmt.Sprintf("%s/%s", storage.StorageType, storage.MediumType)
			if storage.IsSysDiskStore {
				_, ok := ret.StorageTypes2[hostDriver.GetHypervisor()]
				if !ok {
					ret.StorageTypes2[hostDriver.GetHypervisor()] = []string{}
				}
				if !utils.IsInStringArray(storageType, ret.StorageTypes2[hostDriver.GetHypervisor()]) {
					ret.StorageTypes2[hostDriver.GetHypervisor()] = append(ret.StorageTypes2[hostDriver.GetHypervisor()], storageType)
				}
			}
			_, ok := ret.DataStorageTypes2[hostDriver.GetHypervisor()]
			if !ok {
				ret.DataStorageTypes2[hostDriver.GetHypervisor()] = []string{}
			}
			if !utils.IsInStringArray(storageType, ret.DataStorageTypes2[hostDriver.GetHypervisor()]) {
				ret.DataStorageTypes2[hostDriver.GetHypervisor()] = append(ret.DataStorageTypes2[hostDriver.GetHypervisor()], storageType)
			}

			simpleStorage := &SimpleStorageInfo{Storages: []sStorage{}}
			if storage.Cmtbound.Float64 == 0 {
				storage.Cmtbound.Float64 = float64(options.Options.DefaultStorageOvercommitBound)
			}
			storage.VirtualCapacity = int64(float64(storage.Capacity-storage.Reserved.Int64) * storage.Cmtbound.Float64)
			storage.FreeCapacity = storage.VirtualCapacity - storage.UsedCapacity.Int64 - storage.WasteCapacity.Int64
			addStorageInfo(&storage, simpleStorage)

			// set hypervisor storage types and infos
			if storage.IsSysDiskStore {
				setStorageInfos(hostDriver, storageType, &storage, ret.StorageTypes3)
				setStorageInfos2(hostDriver, storageType, &storage, ret.SystemStorageTypes)
			}
			setStorageInfos(hostDriver, storageType, &storage, ret.DataStorageTypes3)
			setStorageInfos2(hostDriver, storageType, &storage, ret.DataStorageTypes)
		}

	}

	return ret, nil
}

type PCIDevModelTypes struct {
	Model   string
	DevType string
	SizeMB  int

	VirtualDev bool
	Hypervisor string
}

func getIsolatedDeviceInfo(ctx context.Context, userCred mcclient.TokenCredential, region *SCloudregion, zone *SZone, domainId string) ([]string, []PCIDevModelTypes) {
	devices := IsolatedDeviceManager.Query().SubQuery()
	hostQuery := HostManager.Query()
	if len(domainId) > 0 {
		ownerId := &db.SOwnerId{DomainId: domainId}
		hostQuery = StorageManager.FilterByOwner(ctx, hostQuery, StorageManager, userCred, ownerId, rbacscope.ScopeDomain)
	}
	hosts := hostQuery.SubQuery()

	q := devices.Query(hosts.Field("host_type"), devices.Field("model"), devices.Field("dev_type"), devices.Field("nvme_size_mb"))
	q = q.Filter(sqlchemy.NotIn(devices.Field("dev_type"), []string{api.USB_TYPE, api.NIC_TYPE}))
	if zone != nil {
		q = q.Join(hosts, sqlchemy.Equals(devices.Field("host_id"), hosts.Field("id")))
		q = q.Filter(sqlchemy.Equals(hosts.Field("zone_id"), zone.Id))
	} else if region != nil {
		subq := getRegionZoneSubq(region)
		q = q.Join(hosts, sqlchemy.Equals(devices.Field("host_id"), hosts.Field("id")))
		q = q.Filter(sqlchemy.In(hosts.Field("zone_id"), subq))
	} else {
		q = q.Join(hosts, sqlchemy.Equals(devices.Field("host_id"), hosts.Field("id")))
	}
	/*if len(domainId) > 0 {
		subq := getDomainManagerSubq(domainId)
		q = q.Filter(sqlchemy.OR(
			sqlchemy.In(hosts.Field("manager_id"), subq),
			sqlchemy.IsNullOrEmpty(hosts.Field("manager_id")),
		))
	}*/
	q = q.GroupBy(hosts.Field("host_type"), devices.Field("model"), devices.Field("dev_type"), devices.Field("nvme_size_mb"))

	rows, err := q.Rows()
	if err != nil {
		log.Errorf("failed get gpu caps: %s", err)
		return nil, nil
	}
	defer rows.Close()
	gpus := make([]PCIDevModelTypes, 0)
	gpuModels := make([]string, 0)
	for rows.Next() {
		var m, t string
		var sizeMB int
		var vdev bool
		var hypervisor string
		var hostType string
		rows.Scan(&hostType, &m, &t, &sizeMB)

		if m == "" {
			continue
		}
		if utils.IsInStringArray(t, api.VITRUAL_DEVICE_TYPES) {
			vdev = true
		}
		if utils.IsInStringArray(t, api.VALID_CONTAINER_DEVICE_TYPES) {
			hypervisor = api.HYPERVISOR_POD
		} else {
			hypervisor = api.HYPERVISOR_KVM
		}

		if hostType == api.HOST_TYPE_ZETTAKIT {
			hypervisor = api.HYPERVISOR_ZETTAKIT
		}

		gpus = append(gpus, PCIDevModelTypes{m, t, sizeMB, vdev, hypervisor})

		if !utils.IsInStringArray(m, gpuModels) {
			gpuModels = append(gpuModels, m)
		}
	}
	return gpuModels, gpus
}

func getHostCpuArchs(ctx context.Context, userCred mcclient.TokenCredential, region *SCloudregion, zone *SZone, domainId string) []string {
	q := HostManager.Query("cpu_architecture").Equals("enabled", true).
		Equals("host_status", "online").In("host_type", []string{api.HOST_TYPE_HYPERVISOR, api.HOST_TYPE_CONTAINER})
	if len(domainId) > 0 {
		ownerId := &db.SOwnerId{DomainId: domainId}
		q = HostManager.FilterByOwner(ctx, q, HostManager, userCred, ownerId, rbacscope.ScopeDomain)
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

func getNetworkCount(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, scope rbacscope.TRbacScope, region *SCloudregion, zone *SZone) (int, error) {
	return getNetworkCountByFilter(ctx, userCred, ownerId, scope, region, zone, tristate.None, "")
}

func getAutoAllocNetworkCount(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, scope rbacscope.TRbacScope, region *SCloudregion, zone *SZone, serverType string) (int, error) {
	return getNetworkCountByFilter(ctx, userCred, ownerId, scope, region, zone, tristate.True, serverType)
}

func getNetworkCountByFilter(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, scope rbacscope.TRbacScope, region *SCloudregion, zone *SZone, isAutoAlloc tristate.TriState, serverType string) (int, error) {
	if zone != nil && region == nil {
		region, _ = zone.GetRegion()
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

	q = NetworkManager.FilterByOwner(ctx, q, NetworkManager, userCred, ownerId, scope)
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

func isUsable(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, scope rbacscope.TRbacScope, region *SCloudregion, zone *SZone) bool {
	cnt, err := getNetworkCount(ctx, userCred, ownerId, scope, region, zone)
	if err != nil {
		return false
	}
	if cnt > 0 {
		return true
	} else {
		return false
	}
}
