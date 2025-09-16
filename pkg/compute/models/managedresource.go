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
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/appctx"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SManagedResourceBase struct {
	// 云订阅ID
	ManagerId string `width:"128" charset:"ascii" nullable:"true" list:"user" create:"optional"`
}

type SManagedResourceBaseManager struct {
	managerIdFieldName string
}

func (self *SManagedResourceBase) GetCloudproviderId() string {
	return self.ManagerId
}

func ValidateCloudproviderResourceInput(ctx context.Context, userCred mcclient.TokenCredential, query api.CloudproviderResourceInput) (*SCloudprovider, api.CloudproviderResourceInput, error) {
	managerObj, err := CloudproviderManager.FetchByIdOrName(ctx, userCred, query.CloudproviderId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, query, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s", CloudproviderManager.Keyword(), query.CloudproviderId)
		} else {
			return nil, query, errors.Wrap(err, "CloudproviderManager.FetchByIdOrName")
		}
	}
	query.CloudproviderId = managerObj.GetId()
	return managerObj.(*SCloudprovider), query, nil
}

func (manager *SManagedResourceBaseManager) getManagerIdFileName() string {
	if len(manager.managerIdFieldName) > 0 {
		return manager.managerIdFieldName
	}
	return "manager_id"
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
	account, _ := cp.GetCloudaccount()
	return account
}

func (self *SManagedResourceBase) GetRegionDriver() (IRegionDriver, error) {
	cloudprovider := self.GetCloudprovider()
	provider := api.CLOUD_PROVIDER_ONECLOUD
	if cloudprovider != nil {
		provider = cloudprovider.Provider
	}
	driver := GetRegionDriver(provider)
	if driver == nil {
		return nil, errors.Wrapf(httperrors.ErrInvalidStatus, "failed to get %s region drivder", provider)
	}
	return driver, nil
}

func (self *SManagedResourceBase) GetProviderFactory() (cloudprovider.ICloudProviderFactory, error) {
	provider := self.GetCloudprovider()
	if provider == nil {
		if len(self.ManagerId) > 0 {
			return nil, cloudprovider.ErrInvalidProvider
		}
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "Resource is self managed")
	}
	return provider.GetProviderFactory()
}

func (self *SManagedResourceBase) GetDriver(ctx context.Context) (cloudprovider.ICloudProvider, error) {
	provider := self.GetCloudprovider()
	if provider == nil {
		if len(self.ManagerId) > 0 {
			return nil, cloudprovider.ErrInvalidProvider
		}
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "Resource is self managed")
	}
	return provider.GetProvider(ctx)
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

func (self *SManagedResourceBase) CanShareToDomain(domainId string) bool {
	provider := self.GetCloudprovider()
	if provider == nil {
		return true
	}
	account, _ := provider.GetCloudaccount()
	if account == nil {
		// no cloud account, can share to any domain
		return true
	}
	switch account.ShareMode {
	case api.CLOUD_ACCOUNT_SHARE_MODE_ACCOUNT_DOMAIN:
		if domainId == account.DomainId {
			return true
		} else {
			return false
		}
	case api.CLOUD_ACCOUNT_SHARE_MODE_PROVIDER_DOMAIN:
		if domainId == provider.DomainId {
			return true
		} else {
			return false
		}
	case api.CLOUD_ACCOUNT_SHARE_MODE_SYSTEM:
		if account.PublicScope == string(rbacscope.ScopeSystem) {
			return true
		} else {
			// public_scope = domain
			if domainId == account.DomainId {
				return true
			}
			if utils.IsInStringArray(domainId, account.GetSharedDomains()) {
				return true
			}
			return false
		}
	default:
		return true
	}
}

func (manager *SManagedResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ManagedResourceInfo {
	rows := make([]api.ManagedResourceInfo, len(objs))
	managerIds := make([]string, len(objs))
	managerCnt := 0
	for i := range objs {
		rows[i] = api.ManagedResourceInfo{}
		rows[i].CloudEnv = api.CLOUD_ENV_ON_PREMISE
		rows[i].Provider = api.CLOUD_PROVIDER_ONECLOUD
		rows[i].Brand = api.CLOUD_PROVIDER_ONECLOUD
		var base *SManagedResourceBase
		err := reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if err != nil {
			log.Errorf("Cannot find SCloudregionResourceBase in object %s", objs[i])
			continue
		}
		if base != nil && len(base.ManagerId) > 0 {
			managerIds[i] = base.ManagerId
			managerCnt += 1
		}
	}

	if managerCnt == 0 {
		return rows
	}

	managers := make(map[string]SCloudprovider)
	err := db.FetchStandaloneObjectsByIds(CloudproviderManager, managerIds, &managers)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail %s", err)
		return rows
	}

	accountIds := make([]string, len(objs))
	projectIds := make([]string, len(objs))
	for i := range rows {
		if _, ok := managers[managerIds[i]]; ok {
			manager := managers[managerIds[i]]
			rows[i].Manager = manager.Name
			rows[i].ManagerDomainId = manager.DomainId
			rows[i].ManagerProjectId = manager.ProjectId
			rows[i].AccountId = manager.CloudaccountId

			projectIds[i] = manager.ProjectId
			accountIds[i] = manager.CloudaccountId
		}
	}

	accounts := make(map[string]SCloudaccount)
	err = db.FetchStandaloneObjectsByIds(CloudaccountManager, accountIds, &accounts)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds for accounts fail %s", err)
		return nil
	}

	projects := db.DefaultProjectsFetcher(ctx, projectIds, false)

	for i := range rows {
		if account, ok := accounts[rows[i].AccountId]; ok {
			rows[i].Account = account.Name
			rows[i].AccountStatus = account.Status
			rows[i].AccountHealthStatus = account.HealthStatus
			rows[i].AccountReadOnly = account.ReadOnly
			rows[i].Brand = account.Brand
			rows[i].Provider = account.Provider
			rows[i].CloudEnv = account.GetCloudEnv()
			rows[i].Environment = account.GetEnvironment()
		}
		if project, ok := projects[rows[i].ManagerProjectId]; ok {
			rows[i].ManagerProject = project.Name
			rows[i].ManagerDomain = project.Domain
			rows[i].ManagerDomainId = project.DomainId
		}
	}

	return rows
}

func (manager *SManagedResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ManagedResourceListInput,
) (*sqlchemy.SQuery, error) {
	return _managedResourceFilterByAccount(ctx, manager.getManagerIdFileName(), q, query, "", nil)
}

func (manager *SManagedResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	switch field {
	case "manager":
		managerQuery := CloudproviderManager.Query("name", "id").SubQuery()
		q.AppendField(managerQuery.Field("name", field)).Distinct()
		q = q.Join(managerQuery, sqlchemy.Equals(q.Field(manager.getManagerIdFileName()), managerQuery.Field("id")))
		return q, nil
	case "account":
		accountQuery := CloudaccountManager.Query("name", "id").SubQuery()
		providers := CloudproviderManager.Query("id", "cloudaccount_id").SubQuery()
		q.AppendField(accountQuery.Field("name", field)).Distinct()
		q = q.Join(providers, sqlchemy.Equals(q.Field(manager.getManagerIdFileName()), providers.Field("id")))
		q = q.Join(accountQuery, sqlchemy.Equals(providers.Field("cloudaccount_id"), accountQuery.Field("id")))
		return q, nil
	case "provider", "brand":
		accountQuery := CloudaccountManager.Query(field, "id").Distinct().SubQuery()
		providers := CloudproviderManager.Query("id", "cloudaccount_id").SubQuery()
		q.AppendField(accountQuery.Field(field)).Distinct()
		q = q.Join(providers, sqlchemy.Equals(q.Field(manager.getManagerIdFileName()), providers.Field("id")))
		q = q.Join(accountQuery, sqlchemy.Equals(providers.Field("cloudaccount_id"), accountQuery.Field("id")))
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SManagedResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ManagedResourceListInput,
) (*sqlchemy.SQuery, error) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, nil
	}
	orderQ := CloudproviderManager.Query("id")
	subOrderQ := orderQ.SubQuery()
	orderQ, orders, fields := manager.GetOrderBySubQuery(orderQ, subOrderQ, orderQ.Field("id"), userCred, query, nil, nil)
	q = q.LeftJoin(subOrderQ, sqlchemy.Equals(q.Field(manager.getManagerIdFileName()), subOrderQ.Field("id")))
	q = db.OrderByFields(q, orders, fields)
	return q, nil
}

func (manager *SManagedResourceBaseManager) GetOrderBySubQuery(
	q *sqlchemy.SQuery,
	subq *sqlchemy.SSubQuery,
	subqField sqlchemy.IQueryField,
	userCred mcclient.TokenCredential,
	query api.ManagedResourceListInput,
	orders []string,
	fields []sqlchemy.IQueryField,
) (*sqlchemy.SQuery, []string, []sqlchemy.IQueryField) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, orders, fields
	}
	providers := CloudproviderManager.Query().SubQuery()
	accounts := CloudaccountManager.Query().SubQuery()
	q = q.LeftJoin(providers, sqlchemy.Equals(subqField, providers.Field("id")))
	q = q.LeftJoin(accounts, sqlchemy.Equals(providers.Field("cloudaccount_id"), accounts.Field("id")))
	q = q.AppendField(providers.Field("name").Label("manager"))
	q = q.AppendField(accounts.Field("name").Label("account"))
	q = q.AppendField(accounts.Field("provider"))
	q = q.AppendField(accounts.Field("brand"))
	orders = append(orders, query.OrderByManager, query.OrderByAccount, query.OrderByProvider, query.OrderByBrand)
	fields = append(fields, subq.Field("manager"), subq.Field("account"), subq.Field("provider"), subq.Field("brand"))
	return q, orders, fields
}

func (manager *SManagedResourceBaseManager) GetOrderByFields(query api.ManagedResourceListInput) []string {
	return []string{query.OrderByManager, query.OrderByAccount, query.OrderByProvider, query.OrderByBrand}
}

func (model *SManagedResourceBase) GetChangeOwnerCandidateDomainIds() []string {
	provider := model.GetCloudprovider()
	if provider == nil {
		return nil
	}
	account := model.GetCloudaccount()
	if account == nil {
		return nil
	}
	var candidateIds []string
	switch account.ShareMode {
	case api.CLOUD_ACCOUNT_SHARE_MODE_ACCOUNT_DOMAIN:
		candidateIds = append(candidateIds, account.DomainId)
	case api.CLOUD_ACCOUNT_SHARE_MODE_PROVIDER_DOMAIN:
		candidateIds = append(candidateIds, provider.DomainId)
	case api.CLOUD_ACCOUNT_SHARE_MODE_SYSTEM:
		if account.PublicScope != string(rbacscope.ScopeSystem) {
			candidateIds = account.GetSharedDomains()
			candidateIds = append(candidateIds, account.DomainId)
		}
	}
	return candidateIds
}

func (manager *SManagedResourceBaseManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	if keys.ContainsAny(manager.GetExportKeys()...) {
		cloudprovidersQ := CloudproviderManager.Query("id", "name", "cloudaccount_id").SubQuery()
		q = q.LeftJoin(cloudprovidersQ, sqlchemy.Equals(q.Field(manager.getManagerIdFileName()), cloudprovidersQ.Field("id")))
		if keys.Contains("manager") {
			q = q.AppendField(cloudprovidersQ.Field("name", "manager"))
		}
		if keys.Contains("account") || keys.Contains("provider") || keys.Contains("brand") {
			cloudaccountsQ := CloudaccountManager.Query("id", "name", "provider", "brand").SubQuery()
			q = q.LeftJoin(cloudaccountsQ, sqlchemy.Equals(cloudprovidersQ.Field("cloudaccount_id"), cloudaccountsQ.Field("id")))
			if keys.Contains("account") {
				q = q.AppendField(cloudaccountsQ.Field("name", "account"))
			}
			if keys.Contains("provider") {
				q = q.AppendField(cloudaccountsQ.Field("provider"))
			}
			if keys.Contains("brand") {
				q = q.AppendField(cloudaccountsQ.Field("provider"))
			}
		}
	}
	return q, nil
}

func (manager *SManagedResourceBaseManager) GetExportKeys() []string {
	return []string{"manager", "account", "provider", "brand"}
}

func _managedResourceFilterByDomain(managerIdFieldName string, q *sqlchemy.SQuery, query apis.DomainizedResourceListInput, filterField string, subqFunc func() *sqlchemy.SQuery) (*sqlchemy.SQuery, error) {
	domainStr := query.ProjectDomainId
	if len(domainStr) > 0 {
		domain, err := db.TenantCacheManager.FetchDomainByIdOrName(context.Background(), domainStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("domains", domainStr)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		accounts := CloudaccountManager.Query("id")
		accounts = CloudaccountManager.filterByDomainId(accounts, domain.GetId())
		subq := CloudproviderManager.Query("id").In("cloudaccount_id", accounts.SubQuery())
		/*subq := providers.Query(providers.Field("id"))
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
		))*/
		if len(filterField) == 0 {
			q = q.Filter(sqlchemy.OR(
				sqlchemy.IsNullOrEmpty(q.Field(managerIdFieldName)),
				sqlchemy.In(q.Field(managerIdFieldName), subq.SubQuery()),
			))
		} else {
			sq := subqFunc()
			sq = sq.Filter(sqlchemy.OR(sqlchemy.In(sq.Field(managerIdFieldName), subq.SubQuery()), sqlchemy.IsNullOrEmpty(sq.Field(managerIdFieldName))))
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

func _filterByProviderStrs(managerIdFieldName string, q *sqlchemy.SQuery, filterField string, subqFunc func() *sqlchemy.SQuery, fieldName string, providerStrs []string) *sqlchemy.SQuery {
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
		filters = append(filters, sqlchemy.In(sq.Field(managerIdFieldName), subq.SubQuery()))
	}
	if oneCloud {
		filters = append(filters, sqlchemy.IsNullOrEmpty(sq.Field(managerIdFieldName)))
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

func managedResourceFilterByAccount(ctx context.Context, q *sqlchemy.SQuery, input api.ManagedResourceListInput, filterField string, subqFunc func() *sqlchemy.SQuery) (*sqlchemy.SQuery, error) {
	return _managedResourceFilterByAccount(ctx, "manager_id", q, input, filterField, subqFunc)
}

func _managedResourceFilterByAccount(ctx context.Context, managerIdFieldName string, q *sqlchemy.SQuery, input api.ManagedResourceListInput, filterField string, subqFunc func() *sqlchemy.SQuery) (*sqlchemy.SQuery, error) {
	cloudproviderStrs := input.CloudproviderId
	managerIds := []string{}
	for _, cloudproviderStr := range cloudproviderStrs {
		if len(cloudproviderStr) == 0 {
			continue
		}
		provider, err := CloudproviderManager.FetchByIdOrName(ctx, nil, cloudproviderStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(CloudproviderManager.Keyword(), cloudproviderStr)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		if len(filterField) == 0 {
			managerIds = append(managerIds, provider.GetId())
		} else {
			sq := subqFunc()
			sq = sq.Filter(sqlchemy.Equals(sq.Field(managerIdFieldName), provider.GetId()))
			q = q.Filter(sqlchemy.In(q.Field(filterField), sq.SubQuery()))
		}
	}

	if len(managerIds) > 0 {
		q = q.In(managerIdFieldName, managerIds)
	}

	cloudaccountArr := input.CloudaccountId
	if len(cloudaccountArr) > 0 {
		cpq := CloudaccountManager.Query().SubQuery()
		subcpq := cpq.Query(cpq.Field("id")).Filter(sqlchemy.OR(
			sqlchemy.In(cpq.Field("id"), stringutils2.RemoveUtf8Strings(cloudaccountArr)),
			sqlchemy.In(cpq.Field("name"), cloudaccountArr),
		)).SubQuery()
		subq := CloudproviderManager.Query("id").In("cloudaccount_id", subcpq).SubQuery()
		if len(filterField) == 0 {
			q = q.Filter(sqlchemy.In(q.Field(managerIdFieldName), subq))
		} else {
			sq := subqFunc()
			sq = sq.Filter(sqlchemy.In(sq.Field(managerIdFieldName), subq))
			q = q.Filter(sqlchemy.In(q.Field(filterField), sq.SubQuery()))
		}
	}

	if len(input.Providers) > 0 {
		q = _filterByProviderStrs(managerIdFieldName, q, filterField, subqFunc, "provider", input.Providers)
	}

	if len(input.Brands) > 0 {
		q = _filterByProviderStrs(managerIdFieldName, q, filterField, subqFunc, "brand", input.Brands)
	}

	if input.IsManaged != nil {
		if *input.IsManaged {
			q = q.IsNotEmpty(managerIdFieldName)
		} else {
			q = q.IsNullOrEmpty(managerIdFieldName)
		}
	}

	q = _filterByCloudType(managerIdFieldName, q, input, filterField, subqFunc)

	q, err := _managedResourceFilterByDomain(managerIdFieldName, q, input.DomainizedResourceListInput, filterField, subqFunc)
	if err != nil {
		return nil, errors.Wrap(err, "managedResourceFilterByDomain")
	}

	return q, nil
}

func managedResourceFilterByZone(ctx context.Context, q *sqlchemy.SQuery, query api.ZonalFilterListInput, filterField string, subqFunc func() *sqlchemy.SQuery) (*sqlchemy.SQuery, error) {
	zoneList := query.ZoneList()
	if len(query.ZoneIds) >= 1 {
		zoneQ := ZoneManager.Query("id")
		zoneQ = zoneQ.Filter(sqlchemy.OR(
			sqlchemy.In(zoneQ.Field("id"), zoneList),
			sqlchemy.In(zoneQ.Field("name"), zoneList),
		))
		if len(filterField) == 0 {
			q = q.Filter(sqlchemy.In(q.Field("zone_id"), zoneQ.SubQuery()))
		} else {
			sq := subqFunc()
			sq = sq.Filter(sqlchemy.In(sq.Field("zone_id"), zoneQ.SubQuery()))
			q = q.Filter(sqlchemy.In(q.Field(filterField), sq.SubQuery()))
		}
	} else if len(query.ZoneId) > 0 {
		zoneObj, _, err := ValidateZoneResourceInput(ctx, nil, query.ZoneResourceInput)
		if err != nil {
			return nil, errors.Wrap(err, "ValidateZoneResourceInput")
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

func managedResourceFilterByRegion(ctx context.Context, q *sqlchemy.SQuery, query api.RegionalFilterListInput, filterField string, subqFunc func() *sqlchemy.SQuery) (*sqlchemy.SQuery, error) {
	regionIds := []string{}
	for _, region := range query.CloudregionId {
		if len(region) == 0 {
			continue
		}
		regionObj, err := ValidateCloudregionId(ctx, nil, region)
		if err != nil {
			return nil, errors.Wrap(err, "ValidateCloudregionResourceInput")
		}
		regionIds = append(regionIds, regionObj.GetId())
	}
	if len(filterField) == 0 {
		if len(regionIds) > 0 {
			q = q.In("cloudregion_id", regionIds)
		}
	} else {
		if len(regionIds) > 0 {
			sq := subqFunc()
			sq = sq.Filter(sqlchemy.In(sq.Field("cloudregion_id"), regionIds))
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

func _filterByCloudType(managerIdFieldName string, q *sqlchemy.SQuery, input api.ManagedResourceListInput, filterField string, subqFunc func() *sqlchemy.SQuery) *sqlchemy.SQuery {
	cloudEnvStr := input.CloudEnv

	switch cloudEnvStr {
	case api.CLOUD_ENV_PUBLIC_CLOUD:
		if len(filterField) == 0 {
			q = q.Filter(sqlchemy.In(q.Field(managerIdFieldName), CloudproviderManager.GetPublicProviderIdsQuery()))
		} else {
			sq := subqFunc()
			sq = sq.Filter(sqlchemy.In(sq.Field(managerIdFieldName), CloudproviderManager.GetPublicProviderIdsQuery()))
			q = q.Filter(sqlchemy.In(q.Field(filterField), sq.SubQuery()))
		}
	case api.CLOUD_ENV_PRIVATE_CLOUD:
		if len(filterField) == 0 {
			q = q.Filter(sqlchemy.In(q.Field(managerIdFieldName), CloudproviderManager.GetPrivateProviderIdsQuery()))
		} else {
			sq := subqFunc()
			sq = sq.Filter(sqlchemy.In(sq.Field(managerIdFieldName), CloudproviderManager.GetPrivateProviderIdsQuery()))
			q = q.Filter(sqlchemy.In(q.Field(filterField), sq.SubQuery()))
		}
	case api.CLOUD_ENV_ON_PREMISE:
		if len(filterField) == 0 {
			q = q.Filter(
				sqlchemy.OR(
					sqlchemy.In(q.Field(managerIdFieldName), CloudproviderManager.GetOnPremiseProviderIdsQuery()),
					sqlchemy.IsNullOrEmpty(q.Field(managerIdFieldName)),
				),
			)
		} else {
			sq := subqFunc()
			sq = sq.Filter(
				sqlchemy.OR(
					sqlchemy.In(sq.Field(managerIdFieldName), CloudproviderManager.GetOnPremiseProviderIdsQuery()),
					sqlchemy.IsNullOrEmpty(sq.Field(managerIdFieldName)),
				),
			)
			q = q.Filter(sqlchemy.In(q.Field(filterField), sq.SubQuery()))
		}
	case api.CLOUD_ENV_PRIVATE_ON_PREMISE:
		if len(filterField) == 0 {
			q = q.Filter(
				sqlchemy.OR(
					sqlchemy.In(q.Field(managerIdFieldName), CloudproviderManager.GetPrivateOrOnPremiseProviderIdsQuery()),
					sqlchemy.IsNullOrEmpty(q.Field(managerIdFieldName)),
				),
			)
		} else {
			sq := subqFunc()
			sq = sq.Filter(
				sqlchemy.OR(
					sqlchemy.In(sq.Field(managerIdFieldName), CloudproviderManager.GetPrivateOrOnPremiseProviderIdsQuery()),
					sqlchemy.IsNullOrEmpty(sq.Field(managerIdFieldName)),
				),
			)
			q = q.Filter(sqlchemy.In(q.Field(filterField), sq.SubQuery()))
		}
	}

	/*if input.IsManaged {
		if len(filterField) == 0 {
			q = q.Filter(sqlchemy.IsNotEmpty(q.Field(managerIdFieldName)))
		} else {
			sq := subqFunc()
			sq = sq.Filter(sqlchemy.IsNotEmpty(sq.Field(managerIdFieldName)))
			q = q.Filter(sqlchemy.In(q.Field(filterField), sq.SubQuery()))
		}
	}*/

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
	if len(extId) == 0 {
		return ""
	}
	pos := strings.LastIndexByte(extId, '/')
	if pos > 0 {
		return extId[pos+1:]
	} else {
		return extId
	}
}

func MakeCloudProviderInfo(region *SCloudregion, zone *SZone, provider *SCloudprovider) SCloudProviderInfo {
	info := SCloudProviderInfo{}

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

		account, _ := provider.GetCloudaccount()
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

func fetchByVpcManagerId(manager db.IModelManager, providerId string, receiver interface{}) error {
	vpc := VpcManager.Query().SubQuery()
	q := manager.Query()
	q = q.Join(vpc, sqlchemy.Equals(vpc.Field("id"), q.Field("vpc_id"))).Filter(sqlchemy.Equals(vpc.Field("manager_id"), providerId))
	return db.FetchModelObjects(manager, q, receiver)
}

func fetchByLbVpcManagerId(manager db.IModelManager, providerId string, receiver interface{}) error {
	vpc := VpcManager.Query().SubQuery()
	lb := LoadbalancerManager.Query().SubQuery()
	q := manager.Query()
	q = q.Join(lb, sqlchemy.Equals(lb.Field("id"), q.Field("loadbalancer_id"))).
		Join(vpc, sqlchemy.Equals(vpc.Field("id"), lb.Field("vpc_id"))).
		Filter(sqlchemy.Equals(vpc.Field("manager_id"), providerId))
	return db.FetchModelObjects(manager, q, receiver)
}
