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

	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
)

type SManagedResourceBase struct {
	ManagerId string `width:"128" charset:"ascii" nullable:"true" list:"admin" create:"optional"` // Column(VARCHAR(ID_LENGTH, charset='ascii'), nullable=True)
}

func (self *SManagedResourceBase) GetCloudprovider() *SCloudprovider {
	if len(self.ManagerId) > 0 {
		return CloudproviderManager.FetchCloudproviderById(self.ManagerId)
	}
	return nil
}

func (self *SManagedResourceBase) GetCloudaccount() *SCloudaccount {
	cp := self.GetCloudprovider()
	if cp == nil {
		return nil
	}
	return cp.GetCloudaccount()
}

func (self *SManagedResourceBase) GetRegionDriver() (IRegionDriver, error) {
	cloudprovider := self.GetCloudprovider()
	provider := api.CLOUD_PROVIDER_ONECLOUD
	if cloudprovider != nil {
		provider = cloudprovider.Provider
	}
	driver := GetRegionDriver(provider)
	if driver == nil {
		return nil, fmt.Errorf("failed to get %s region drivder", provider)
	}
	return driver, nil
}

func (self *SManagedResourceBase) GetProviderFactory() (cloudprovider.ICloudProviderFactory, error) {
	provider := self.GetCloudprovider()
	if provider == nil {
		if len(self.ManagerId) > 0 {
			return nil, cloudprovider.ErrInvalidProvider
		}
		return nil, fmt.Errorf("Resource is self managed")
	}
	return provider.GetProviderFactory()
}

func (self *SManagedResourceBase) GetDriver() (cloudprovider.ICloudProvider, error) {
	provider := self.GetCloudprovider()
	if provider == nil {
		if len(self.ManagerId) > 0 {
			return nil, cloudprovider.ErrInvalidProvider
		}
		return nil, fmt.Errorf("Resource is self managed")
	}
	return provider.GetProvider()
}

func (self *SManagedResourceBase) GetProviderName() string {
	account := self.GetCloudaccount()
	if account != nil {
		return account.Provider
	}
	return api.CLOUD_PROVIDER_ONECLOUD
}

func (self *SManagedResourceBase) GetBrand() string {
	account := self.GetCloudaccount()
	if account != nil {
		return account.Brand
	}
	return api.CLOUD_PROVIDER_ONECLOUD
}

func (self *SManagedResourceBase) IsManaged() bool {
	return len(self.ManagerId) > 0
}

func managedResourceFilterByDomain(q *sqlchemy.SQuery, query apis.DomainizedResourceListInput, filterField string, subqFunc func() *sqlchemy.SQuery) (*sqlchemy.SQuery, error) {
	domainStr := query.ProjectDomain
	if len(domainStr) > 0 {
		domain, err := db.TenantCacheManager.FetchDomainByIdOrName(context.Background(), domainStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("domains", domainStr)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		accounts := CloudaccountManager.Query().SubQuery()
		providers := CloudproviderManager.Query().SubQuery()
		subq := providers.Query(providers.Field("id"))
		subq = subq.Join(accounts, sqlchemy.Equals(providers.Field("cloudaccount_id"), accounts.Field("id")))
		subq = subq.Filter(sqlchemy.OR(
			sqlchemy.AND(
				sqlchemy.Equals(providers.Field("domain_id"), domain.GetId()),
				sqlchemy.Equals(accounts.Field("share_mode"), api.CLOUD_ACCOUNT_SHARE_MODE_PROVIDER_DOMAIN),
			),
			sqlchemy.Equals(accounts.Field("share_mode"), api.CLOUD_ACCOUNT_SHARE_MODE_SYSTEM),
			sqlchemy.AND(
				sqlchemy.Equals(accounts.Field("domain_id"), domain.GetId()),
				sqlchemy.Equals(accounts.Field("share_mode"), api.CLOUD_ACCOUNT_SHARE_MODE_ACCOUNT_DOMAIN),
			),
		))
		if len(filterField) == 0 {
			q = q.Filter(sqlchemy.OR(
				sqlchemy.IsNullOrEmpty(q.Field("manager_id")),
				sqlchemy.In(q.Field("manager_id"), subq.SubQuery()),
			))
		} else {
			sq := subqFunc()
			sq = sq.Filter(sqlchemy.OR(sqlchemy.In(sq.Field("manager_id"), subq.SubQuery()), sqlchemy.IsNullOrEmpty(sq.Field("manager_id"))))
			q = q.Filter(sqlchemy.In(q.Field(filterField), sq.SubQuery()))
		}
	}
	return q, nil
}

func splitProviders(providers []string) (bool, []string) {
	oneCloud := false
	others := make([]string, 0)
	for _, pro := range providers {
		if pro == api.CLOUD_PROVIDER_ONECLOUD {
			oneCloud = true
		} else {
			others = append(others, pro)
		}
	}
	return oneCloud, others
}

func filterByProviderStrs(q *sqlchemy.SQuery, filterField string, subqFunc func() *sqlchemy.SQuery, fieldName string, providerStrs []string) *sqlchemy.SQuery {
	oneCloud, providers := splitProviders(providerStrs)
	sq := q
	if len(filterField) > 0 {
		sq = subqFunc()
	}
	filters := make([]sqlchemy.ICondition, 0)
	if len(providers) > 0 {
		account := CloudaccountManager.Query().SubQuery()
		providers := CloudproviderManager.Query().SubQuery()
		subq := providers.Query(providers.Field("id"))
		subq = subq.Join(account, sqlchemy.Equals(
			account.Field("id"), providers.Field("cloudaccount_id"),
		))
		subq = subq.Filter(sqlchemy.In(account.Field(fieldName), providerStrs))
		filters = append(filters, sqlchemy.In(sq.Field("manager_id"), subq.SubQuery()))
	}
	if oneCloud {
		filters = append(filters, sqlchemy.IsNullOrEmpty(sq.Field("manager_id")))
	}
	if len(filters) == 1 {
		sq = sq.Filter(filters[0])
	} else if len(filters) > 1 {
		sq = sq.Filter(sqlchemy.OR(filters...))
	}
	if len(filterField) == 0 {
		q = sq
	} else {
		q = q.Filter(sqlchemy.In(q.Field(filterField), sq.SubQuery()))
	}
	return q
}

func managedResourceFilterByAccount(q *sqlchemy.SQuery, input api.ManagedResourceListInput, filterField string, subqFunc func() *sqlchemy.SQuery) (*sqlchemy.SQuery, error) {
	cloudproviderStr := input.Cloudprovider
	if len(cloudproviderStr) > 0 {
		provider, err := CloudproviderManager.FetchByIdOrName(nil, cloudproviderStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(CloudproviderManager.Keyword(), cloudproviderStr)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		if len(filterField) == 0 {
			q = q.Filter(sqlchemy.Equals(q.Field("manager_id"), provider.GetId()))
		} else {
			sq := subqFunc()
			sq = sq.Filter(sqlchemy.Equals(sq.Field("manager_id"), provider.GetId()))
			q = q.Filter(sqlchemy.In(q.Field(filterField), sq.SubQuery()))
		}
	}

	cloudaccountStr := input.Cloudaccount
	if len(cloudaccountStr) > 0 {
		account, err := CloudaccountManager.FetchByIdOrName(nil, cloudaccountStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(CloudaccountManager.Keyword(), cloudaccountStr)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		subq := CloudproviderManager.Query("id").Equals("cloudaccount_id", account.GetId()).SubQuery()
		if len(filterField) == 0 {
			q = q.Filter(sqlchemy.In(q.Field("manager_id"), subq))
		} else {
			sq := subqFunc()
			sq = sq.Filter(sqlchemy.In(sq.Field("manager_id"), subq))
			q = q.Filter(sqlchemy.In(q.Field(filterField), sq.SubQuery()))
		}
	}

	if len(input.Providers) > 0 {
		q = filterByProviderStrs(q, filterField, subqFunc, "provider", input.Providers)
	}

	if len(input.Brands) > 0 {
		q = filterByProviderStrs(q, filterField, subqFunc, "brand", input.Brands)
	}

	q = filterByCloudType(q, input, filterField, subqFunc)

	return q, nil
}

func managedResourceFilterByZone(q *sqlchemy.SQuery, query api.ZonalFilterListInput, filterField string, subqFunc func() *sqlchemy.SQuery) (*sqlchemy.SQuery, error) {
	zoneStr := query.Zone
	if len(zoneStr) > 0 {
		zoneObj, err := ZoneManager.FetchByIdOrName(nil, zoneStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(ZoneManager.Keyword(), zoneStr)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		if len(filterField) == 0 {
			q = q.Filter(sqlchemy.Equals(q.Field("zone_id"), zoneObj.GetId()))
		} else {
			sq := subqFunc()
			sq = sq.Filter(sqlchemy.Equals(sq.Field("zone_id"), zoneObj.GetId()))
			q = q.Filter(sqlchemy.In(q.Field(filterField), sq.SubQuery()))
		}
	}

	return q, nil
}

func managedResourceFilterByRegion(q *sqlchemy.SQuery, query api.RegionalFilterListInput, filterField string, subqFunc func() *sqlchemy.SQuery) (*sqlchemy.SQuery, error) {
	regionStr := query.Cloudregion
	if len(regionStr) > 0 {
		regionObj, err := CloudregionManager.FetchByIdOrName(nil, regionStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(CloudregionManager.Keyword(), regionStr)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		if len(filterField) == 0 {
			q = q.Filter(sqlchemy.Equals(q.Field("cloudregion_id"), regionObj.GetId()))
		} else {
			sq := subqFunc()
			sq = sq.Filter(sqlchemy.Equals(sq.Field("cloudregion_id"), regionObj.GetId()))
			q = q.Filter(sqlchemy.In(q.Field(filterField), sq.SubQuery()))
		}
	}
	if len(query.City) > 0 {
		subq := CloudregionManager.Query("id").Equals("city", query.City).SubQuery()
		if len(filterField) == 0 {
			q = q.Filter(sqlchemy.In(q.Field("cloudregion_id"), subq))
		} else {
			sq := subqFunc()
			sq = sq.Filter(sqlchemy.In(sq.Field("cloudregion_id"), subq))
			q = q.Filter(sqlchemy.In(q.Field(filterField), sq.SubQuery()))
		}
	}
	return q, nil
}

func filterByCloudType(q *sqlchemy.SQuery, input api.ManagedResourceListInput, filterField string, subqFunc func() *sqlchemy.SQuery) *sqlchemy.SQuery {
	cloudEnvStr := input.CloudEnv

	switch cloudEnvStr {
	case api.CLOUD_ENV_PUBLIC_CLOUD:
		if len(filterField) == 0 {
			q = q.Filter(sqlchemy.In(q.Field("manager_id"), CloudproviderManager.GetPublicProviderIdsQuery()))
		} else {
			sq := subqFunc()
			sq = sq.Filter(sqlchemy.In(sq.Field("manager_id"), CloudproviderManager.GetPublicProviderIdsQuery()))
			q = q.Filter(sqlchemy.In(q.Field(filterField), sq.SubQuery()))
		}
	case api.CLOUD_ENV_PRIVATE_CLOUD:
		if len(filterField) == 0 {
			q = q.Filter(sqlchemy.In(q.Field("manager_id"), CloudproviderManager.GetPrivateProviderIdsQuery()))
		} else {
			sq := subqFunc()
			sq = sq.Filter(sqlchemy.In(sq.Field("manager_id"), CloudproviderManager.GetPrivateProviderIdsQuery()))
			q = q.Filter(sqlchemy.In(q.Field(filterField), sq.SubQuery()))
		}
	case api.CLOUD_ENV_ON_PREMISE:
		if len(filterField) == 0 {
			q = q.Filter(
				sqlchemy.OR(
					sqlchemy.In(q.Field("manager_id"), CloudproviderManager.GetOnPremiseProviderIdsQuery()),
					sqlchemy.IsNullOrEmpty(q.Field("manager_id")),
				),
			)
		} else {
			sq := subqFunc()
			sq = sq.Filter(
				sqlchemy.OR(
					sqlchemy.In(sq.Field("manager_id"), CloudproviderManager.GetOnPremiseProviderIdsQuery()),
					sqlchemy.IsNullOrEmpty(sq.Field("manager_id")),
				),
			)
			q = q.Filter(sqlchemy.In(q.Field(filterField), sq.SubQuery()))
		}
	case api.CLOUD_ENV_PRIVATE_ON_PREMISE:
		if len(filterField) == 0 {
			q = q.Filter(
				sqlchemy.OR(
					sqlchemy.In(q.Field("manager_id"), CloudproviderManager.GetPrivateOrOnPremiseProviderIdsQuery()),
					sqlchemy.IsNullOrEmpty(q.Field("manager_id")),
				),
			)
		} else {
			sq := subqFunc()
			sq = sq.Filter(
				sqlchemy.OR(
					sqlchemy.In(sq.Field("manager_id"), CloudproviderManager.GetPrivateOrOnPremiseProviderIdsQuery()),
					sqlchemy.IsNullOrEmpty(sq.Field("manager_id")),
				),
			)
			q = q.Filter(sqlchemy.In(q.Field(filterField), sq.SubQuery()))
		}
	}

	if input.IsManaged {
		if len(filterField) == 0 {
			q = q.Filter(sqlchemy.IsNotEmpty(q.Field("manager_id")))
		} else {
			sq := subqFunc()
			sq = sq.Filter(sqlchemy.IsNotEmpty(sq.Field("manager_id")))
			q = q.Filter(sqlchemy.In(q.Field(filterField), sq.SubQuery()))
		}
	}

	return q
}

type SCloudProviderInfo struct {
	Provider         string `json:",omitempty"`
	Brand            string `json:",omitempty"`
	Account          string `json:",omitempty"`
	AccountId        string `json:",omitempty"`
	Manager          string `json:",omitempty"`
	ManagerId        string `json:",omitempty"`
	ManagerProject   string `json:",omitempty"`
	ManagerProjectId string `json:",omitempty"`
	ManagerDomain    string `json:",omitempty"`
	ManagerDomainId  string `json:",omitempty"`
	Region           string `json:",omitempty"`
	RegionId         string `json:",omitempty"`
	CloudregionId    string `json:",omitempty"`
	RegionExternalId string `json:",omitempty"`
	RegionExtId      string `json:",omitempty"`
	Zone             string `json:",omitempty"`
	ZoneId           string `json:",omitempty"`
	ZoneExtId        string `json:",omitempty"`
	CloudEnv         string `json:",omitempty"`
}

var (
	providerInfoFields = []string{
		"provider",
		"brand",
		"account",
		"account_id",
		"manager",
		"manager_id",
		"manager_project",
		"manager_project_id",
		"region",
		"region_id",
		"region_external_id",
		"region_ext_id",
		"zone",
		"zone_id",
		"zone_ext_id",
		"cloud_env",
	}
)

func fetchExternalId(extId string) string {
	pos := strings.LastIndexByte(extId, '/')
	if pos > 0 {
		return extId[pos+1:]
	} else {
		return extId
	}
}

func MakeCloudProviderInfo(region *SCloudregion, zone *SZone, provider *SCloudprovider) api.CloudproviderInfo {
	info := api.CloudproviderInfo{}

	if zone != nil {
		info.Zone = zone.GetName()
		info.ZoneId = zone.GetId()
	}

	if region != nil {
		info.Region = region.GetName()
		info.RegionId = region.GetId()
		info.CloudregionId = region.GetId()
	}

	if provider != nil {
		info.Manager = provider.GetName()
		info.ManagerId = provider.GetId()
		info.Provider = provider.Provider

		if len(provider.ProjectId) > 0 {
			info.ManagerProjectId = provider.ProjectId
			tc, err := db.TenantCacheManager.FetchTenantById(appctx.Background, provider.ProjectId)
			if err == nil {
				info.ManagerProject = tc.GetName()
				info.ManagerDomain = tc.Domain
				info.ManagerDomainId = tc.DomainId
			}
		}

		account := provider.GetCloudaccount()
		if account != nil {
			info.Account = account.GetName()
			info.AccountId = account.GetId()
			info.Brand = account.Brand
			info.CloudEnv = account.GetCloudEnv()
		}

		if region != nil {
			info.RegionExternalId = region.ExternalId
			info.RegionExtId = fetchExternalId(region.ExternalId)
			if zone != nil {
				info.ZoneExtId = fetchExternalId(zone.ExternalId)
			}
		}
	} else {
		info.CloudEnv = api.CLOUD_ENV_ON_PREMISE
		info.Provider = api.CLOUD_PROVIDER_ONECLOUD
		info.Brand = api.CLOUD_PROVIDER_ONECLOUD
	}

	return info
}

func fetchByManagerId(manager db.IModelManager, providerId string, receiver interface{}) error {
	q := manager.Query().Equals("manager_id", providerId)
	return db.FetchModelObjects(manager, q, receiver)
}
