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

package db

import (
	"context"
	"database/sql"
	"fmt"
	"runtime/debug"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	identityapi "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

var (
	DefaultProjectFetcher  func(ctx context.Context, id string) (*STenant, error)
	DefaultDomainFetcher   func(ctx context.Context, id string) (*STenant, error)
	DefaultProjectsFetcher func(ctx context.Context, idList []string, isDomain bool) map[string]STenant
	DefaultDomainQuery     func(fields ...string) *sqlchemy.SQuery
	DefaultProjectQuery    func(fields ...string) *sqlchemy.SQuery
)

type STenantCacheManager struct {
	SKeystoneCacheObjectManager
}

type STenant struct {
	SKeystoneCacheObject
}

func NewTenant(idStr string, name string, domainId string, domainName string) STenant {
	return STenant{SKeystoneCacheObject: NewKeystoneCacheObject(idStr, name, domainId, domainName)}
}

func NewDomain(idStr, name string) STenant {
	return NewTenant(idStr, name, identityapi.KeystoneDomainRoot, identityapi.KeystoneDomainRoot)
}

func (tenant *STenant) GetModelManager() IModelManager {
	return TenantCacheManager
}

var TenantCacheManager *STenantCacheManager

func init() {
	TenantCacheManager = &STenantCacheManager{NewKeystoneCacheObjectManager(STenant{}, "tenant_cache_tbl", "tenant", "tenants")}
	// log.Debugf("Initialize tenant cache manager %s %s", TenantCacheManager.KeywordPlural(), TenantCacheManager)

	TenantCacheManager.SetVirtualObject(TenantCacheManager)

	DefaultProjectFetcher = TenantCacheManager.FetchTenantByIdOrName
	DefaultDomainFetcher = TenantCacheManager.FetchDomainByIdOrName
	DefaultProjectsFetcher = fetchProjects
	DefaultDomainQuery = TenantCacheManager.GetDomainQuery
	DefaultProjectQuery = TenantCacheManager.GetTenantQuery
}

func RegistUserCredCacheUpdater() {
	auth.RegisterAuthHook(onAuthCompleteUpdateCache)
}

func onAuthCompleteUpdateCache(ctx context.Context, userCred mcclient.TokenCredential) {
	TenantCacheManager.updateTenantCache(ctx, userCred)
	UserCacheManager.updateUserCache(userCred)
}

func (manager *STenantCacheManager) InitializeData() error {
	q := manager.Query().IsNullOrEmpty("domain_id")
	tenants := make([]STenant, 0)
	err := FetchModelObjects(manager, q, &tenants)
	if err != nil && err != sql.ErrNoRows {
		return errors.Wrap(err, "query")
	}
	for i := range tenants {
		_, err := Update(&tenants[i], func() error {
			tenants[i].DomainId = identityapi.DEFAULT_DOMAIN_ID
			tenants[i].Domain = identityapi.DEFAULT_DOMAIN_NAME
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "update")
		}
	}
	return nil
}

func (manager *STenantCacheManager) updateTenantCache(ctx context.Context, userCred mcclient.TokenCredential) {
	manager.Save(ctx, userCred.GetProjectId(), userCred.GetProjectName(),
		userCred.GetProjectDomainId(), userCred.GetProjectDomain())
}

func (manager *STenantCacheManager) GetTenantQuery(fields ...string) *sqlchemy.SQuery {
	return manager.Query(fields...).NotEquals("domain_id", identityapi.KeystoneDomainRoot)
}

func (manager *STenantCacheManager) GetDomainQuery(fields ...string) *sqlchemy.SQuery {
	return manager.Query(fields...).Equals("domain_id", identityapi.KeystoneDomainRoot)
}

func (manager *STenantCacheManager) fetchTenant(ctx context.Context, idStr string, isDomain bool, noExpireCheck bool, filter func(q *sqlchemy.SQuery) *sqlchemy.SQuery) (*STenant, error) {
	var q *sqlchemy.SQuery
	if isDomain {
		q = manager.GetDomainQuery()
	} else {
		q = manager.GetTenantQuery()
	}
	q = filter(q)
	caches := []STenant{}
	err := FetchModelObjects(manager, q, &caches)
	if err != nil {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	normal := []STenant{}
	for _, cache := range caches {
		if noExpireCheck || !cache.IsExpired() {
			normal = append(normal, cache)
		}
	}
	if len(normal) > 1 {
		return nil, errors.Wrapf(httperrors.ErrDuplicateName, "duplicate tenant/domain name (%d)", len(normal))
	}
	if len(normal) == 1 {
		return &normal[0], nil
	}
	if isDomain {
		return manager.fetchDomainFromKeystone(ctx, idStr)
	} else {
		return manager.fetchTenantFromKeystone(ctx, idStr)
	}
}

func (t *STenant) IsExpired() bool {
	if t.LastCheck.IsZero() {
		return true
	}
	now := time.Now().UTC()
	if t.LastCheck.Add(consts.GetTenantCacheExpireSeconds()).Before(now) {
		return true
	}
	return false
}

func (manager *STenantCacheManager) FetchTenantByIdOrName(ctx context.Context, idStr string) (*STenant, error) {
	return manager.fetchTenant(ctx, idStr, false, false, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		if stringutils2.IsUtf8(idStr) {
			return q.Equals("name", idStr)
		} else {
			return q.Filter(sqlchemy.OR(
				sqlchemy.Equals(q.Field("id"), idStr),
				sqlchemy.Equals(q.Field("name"), idStr),
			))
		}
	})
}

func (manager *STenantCacheManager) FetchTenantById(ctx context.Context, idStr string) (*STenant, error) {
	return manager.fetchTenantById(ctx, idStr, false)
}

func (manager *STenantCacheManager) FetchTenantByIdWithoutExpireCheck(ctx context.Context, idStr string) (*STenant, error) {
	// noExpireCheck should be true
	return manager.fetchTenantById(ctx, idStr, true)
}

func (manager *STenantCacheManager) fetchTenantById(ctx context.Context, idStr string, noExpireCheck bool) (*STenant, error) {
	return manager.fetchTenant(ctx, idStr, false, noExpireCheck, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.Filter(sqlchemy.Equals(q.Field("id"), idStr))
	})
}

func (manager *STenantCacheManager) FetchTenantByName(ctx context.Context, idStr string) (*STenant, error) {
	return manager.fetchTenant(ctx, idStr, false, false, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.Filter(sqlchemy.Equals(q.Field("name"), idStr))
	})
}

func (manager *STenantCacheManager) fetchTenantFromKeystone(ctx context.Context, idStr string) (*STenant, error) {
	if len(idStr) == 0 {
		log.Debugf("fetch empty tenant!!!!\n%s", debug.Stack())
		return nil, fmt.Errorf("Empty idStr")
	}

	// It is to query all domain's project.
	query := jsonutils.NewDict()
	query.Set("scope", jsonutils.NewString("system"))

	s := auth.GetAdminSession(ctx, consts.GetRegion(), "v1")
	tenant, err := modules.Projects.GetById(s, idStr, query)
	if err != nil {
		if je, ok := err.(*httputils.JSONClientError); ok && je.Code == 404 {
			return nil, sql.ErrNoRows
		}
		log.Errorf("fetch project %s fail %s", idStr, err)
		return nil, errors.Wrap(err, "modules.Projects.Get")
	}
	tenantId, _ := tenant.GetString("id")
	tenantName, _ := tenant.GetString("name")
	domainId, _ := tenant.GetString("domain_id")
	domainName, _ := tenant.GetString("project_domain")
	// manager.Save(ctx, domainId, domainName, identityapi.KeystoneDomainRoot, identityapi.KeystoneDomainRoot)
	return manager.Save(ctx, tenantId, tenantName, domainId, domainName)
}

func (manager *STenantCacheManager) FetchDomainByIdOrName(ctx context.Context, idStr string) (*STenant, error) {
	return manager.fetchTenant(ctx, idStr, true, false, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		if stringutils2.IsUtf8(idStr) {
			return q.Equals("name", idStr)
		} else {
			return q.Filter(sqlchemy.OR(
				sqlchemy.Equals(q.Field("id"), idStr),
				sqlchemy.Equals(q.Field("name"), idStr),
			))
		}
	})
}

func (manager *STenantCacheManager) FetchDomainById(ctx context.Context, idStr string) (*STenant, error) {
	return manager.fetchDomainById(ctx, idStr, false)
}

func (manager *STenantCacheManager) FetchDomainByIdWithoutExpireCheck(ctx context.Context, idStr string) (*STenant, error) {
	return manager.fetchDomainById(ctx, idStr, true)
}

func (manager *STenantCacheManager) fetchDomainById(ctx context.Context, idStr string, noExpireCheck bool) (*STenant, error) {
	return manager.fetchTenant(ctx, idStr, true, noExpireCheck, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.Filter(sqlchemy.Equals(q.Field("id"), idStr))
	})
}

func (manager *STenantCacheManager) FetchDomainByName(ctx context.Context, idStr string) (*STenant, error) {
	return manager.fetchTenant(ctx, idStr, true, false, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.Filter(sqlchemy.Equals(q.Field("name"), idStr))
	})
}

func (manager *STenantCacheManager) fetchDomainFromKeystone(ctx context.Context, idStr string) (*STenant, error) {
	if len(idStr) == 0 {
		log.Debugf("fetch empty domain!!!!\n%s", debug.Stack())
		return nil, fmt.Errorf("Empty idStr")
	}
	s := auth.GetAdminSession(ctx, consts.GetRegion(), "v1")
	tenant, err := modules.Domains.GetById(s, idStr, nil)
	if err != nil {
		if je, ok := err.(*httputils.JSONClientError); ok && je.Code == 404 {
			return nil, sql.ErrNoRows
		}
		log.Errorf("fetch project %s fail %s", idStr, err)
		return nil, errors.Wrap(err, "modules.Projects.Get")
	}
	tenantId, err := tenant.GetString("id")
	tenantName, err := tenant.GetString("name")
	return manager.Save(ctx, tenantId, tenantName, identityapi.KeystoneDomainRoot, identityapi.KeystoneDomainRoot)
}

func (manager *STenantCacheManager) Delete(ctx context.Context, idStr string) error {
	lockman.LockRawObject(ctx, manager.KeywordPlural(), idStr)
	defer lockman.ReleaseRawObject(ctx, manager.KeywordPlural(), idStr)

	objo, err := manager.FetchById(idStr)
	if err != nil && err != sql.ErrNoRows {
		log.Errorf("FetchTenantbyId fail %s", err)
		return err
	}
	if err == sql.ErrNoRows {
		return nil
	}
	return objo.Delete(ctx, nil)
}

func (manager *STenantCacheManager) Save(ctx context.Context, idStr string, name string, domainId string, domain string) (*STenant, error) {
	lockman.LockRawObject(ctx, manager.KeywordPlural(), idStr)
	defer lockman.ReleaseRawObject(ctx, manager.KeywordPlural(), idStr)

	objo, err := manager.FetchById(idStr)
	if err != nil && err != sql.ErrNoRows {
		log.Errorf("FetchTenantbyId fail %s", err)
		return nil, err
	}
	now := time.Now().UTC()
	if err == nil {
		obj := objo.(*STenant)
		if obj.Id == idStr && obj.Name == name && obj.Domain == domain && obj.DomainId == domainId {
			Update(obj, func() error {
				obj.LastCheck = now
				return nil
			})
			return obj, nil
		}
		_, err = Update(obj, func() error {
			obj.Id = idStr
			obj.Name = name
			obj.Domain = domain
			obj.DomainId = domainId
			obj.LastCheck = now
			return nil
		})
		if err != nil {
			return nil, err
		} else {
			return obj, nil
		}
	} else {
		objm, err := NewModelObject(manager)
		obj := objm.(*STenant)
		obj.Id = idStr
		obj.Name = name
		obj.Domain = domain
		obj.DomainId = domainId
		obj.LastCheck = now
		err = manager.TableSpec().InsertOrUpdate(ctx, obj)
		if err != nil {
			return nil, errors.Wrap(err, "InsertOrUpdate")
		} else {
			return obj, nil
		}
	}
}

/*func (manager *STenantCacheManager) GenerateProjectUserCred(ctx context.Context, projectName string) (mcclient.TokenCredential, error) {
	project, err := manager.FetchTenantByIdOrName(ctx, projectName)
	if err != nil {
		return nil, err
	}
	return &mcclient.SSimpleToken{
		Project:   project.Name,
		ProjectId: project.Id,
	}, nil
}*/

/*func (tenant *STenant) GetDomain() string {
	if len(tenant.Domain) == 0 {
		return identityapi.DEFAULT_DOMAIN_NAME
	}
	return tenant.Domain
}

func (tenant *STenant) GetDomainId() string {
	if len(tenant.DomainId) == 0 {
		return identityapi.DEFAULT_DOMAIN_ID
	}
	return tenant.DomainId
}*/

func (manager *STenantCacheManager) findFirstProjectOfDomain(domainId string) (*STenant, error) {
	q := manager.Query().Equals("domain_id", domainId)
	tenant := STenant{}
	tenant.SetModelManager(manager, &tenant)
	err := q.First(&tenant)
	if err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (manager *STenantCacheManager) fetchDomainTenantsFromKeystone(ctx context.Context, domainId string) error {
	if len(domainId) == 0 {
		log.Debugf("fetch empty domain!!!!")
		debug.PrintStack()
		return fmt.Errorf("Empty domainId")
	}

	s := auth.GetAdminSession(ctx, consts.GetRegion(), "v1")
	params := jsonutils.Marshal(map[string]string{"domain_id": domainId})
	tenants, err := modules.Projects.List(s, params)
	if err != nil {
		return errors.Wrap(err, "Projects.List")
	}
	for _, tenant := range tenants.Data {
		tenantId, _ := tenant.GetString("id")
		tenantName, _ := tenant.GetString("name")
		domainId, _ := tenant.GetString("domain_id")
		domainName, _ := tenant.GetString("project_domain")
		_, err = manager.Save(ctx, tenantId, tenantName, domainId, domainName)
		if err != nil {
			return err
		}
	}
	return nil
}

func (manager *STenantCacheManager) FindFirstProjectOfDomain(ctx context.Context, domainId string) (*STenant, error) {
	tenant, err := manager.findFirstProjectOfDomain(domainId)
	if err != nil {
		if err == sql.ErrNoRows {
			err = manager.fetchDomainTenantsFromKeystone(ctx, domainId)
			if err != nil {
				return nil, errors.Wrap(err, "fetchDomainTenantsFromKeystone")
			}
			return manager.findFirstProjectOfDomain(domainId)
		}
		return nil, errors.Wrap(err, "findFirstProjectOfDomain.queryFirst")
	}
	return tenant, nil
}

func (tenant *STenant) GetProjectId() string {
	if tenant.IsDomain() {
		return ""
	} else {
		return tenant.Id
	}
}

func (tenant *STenant) GetTenantId() string {
	return tenant.GetProjectId()
}

func (tenant *STenant) GetProjectDomainId() string {
	if tenant.IsDomain() {
		return tenant.Id
	} else {
		return tenant.DomainId
	}
}

func (tenant *STenant) GetTenantName() string {
	return tenant.GetProjectName()
}

func (tenant *STenant) GetProjectName() string {
	if tenant.IsDomain() {
		return ""
	} else {
		return tenant.Name
	}
}

func (tenant *STenant) GetProjectDomain() string {
	if tenant.IsDomain() {
		return tenant.Name
	} else {
		return tenant.Domain
	}
}

func (tenant *STenant) GetUserId() string {
	return ""
}

func (tenant *STenant) GetUserName() string {
	return ""
}

func (tenant *STenant) GetDomainId() string {
	return ""
}

func (tenant *STenant) GetDomainName() string {
	return ""
}

func (tenant *STenant) IsDomain() bool {
	if tenant.DomainId == identityapi.KeystoneDomainRoot {
		return true
	} else {
		return false
	}
}
