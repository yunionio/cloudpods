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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

func FetchJointByIds(manager IJointModelManager, masterId, slaveId string, query jsonutils.JSONObject) (IJointModel, error) {
	obj, err := NewModelObject(manager)
	if err != nil {
		return nil, err
	}
	jointObj, ok := obj.(IJointModel)
	if !ok {
		return nil, fmt.Errorf("FetchByIds not a IJointModel")
	}
	q := manager.Query()
	masterField := q.Field(manager.GetIJointModelManager().GetMasterFieldName()) // queryField(q, manager.GetMasterManager())
	if masterField == nil {
		return nil, fmt.Errorf("cannot find master id")
	}
	slaveField := q.Field(manager.GetIJointModelManager().GetSlaveFieldName()) // queryField(q, manager.GetSlaveManager())
	if slaveField == nil {
		return nil, fmt.Errorf("cannot find slave id")
	}
	cond := sqlchemy.AND(sqlchemy.Equals(masterField, masterId), sqlchemy.Equals(slaveField, slaveId))
	q = q.Filter(cond)
	q = manager.FilterByParams(q, query)
	count, err := q.CountWithError()
	if err != nil {
		return nil, err
	}
	if count > 1 {
		return nil, sqlchemy.ErrDuplicateEntry
	} else if count == 0 {
		return nil, sql.ErrNoRows
	}
	err = q.First(jointObj)
	if err != nil {
		return nil, err
	}
	return jointObj, nil
}

func FetchById(manager IModelManager, idStr string) (IModel, error) {
	q := manager.Query()
	q = manager.FilterById(q, idStr)
	count, err := q.CountWithError()
	if err != nil {
		return nil, err
	}
	if count == 1 {
		obj, err := NewModelObject(manager)
		if err != nil {
			return nil, err
		}
		err = q.First(obj)
		if err != nil {
			return nil, err
		} else {
			return obj, nil
		}
	} else if count > 1 {
		return nil, sqlchemy.ErrDuplicateEntry
	} else {
		return nil, sql.ErrNoRows
	}
}

func FetchByName(manager IModelManager, userCred mcclient.IIdentityProvider, idStr string) (IModel, error) {
	q := manager.Query()
	q = manager.FilterByName(q, idStr)
	count, err := q.CountWithError()
	if err != nil {
		return nil, err
	}
	if count > 0 && userCred != nil {
		q = manager.FilterByOwner(q, userCred, manager.NamespaceScope())
		q = manager.FilterBySystemAttributes(q, nil, nil, manager.ResourceScope())
		count, err = q.CountWithError()
		if err != nil {
			return nil, err
		}
	}
	if count == 1 {
		obj, err := NewModelObject(manager)
		if err != nil {
			return nil, err
		}
		err = q.First(obj)
		if err != nil {
			return nil, err
		} else {
			return obj, nil
		}
	} else if count > 1 {
		return nil, sqlchemy.ErrDuplicateEntry
	} else {
		return nil, sql.ErrNoRows
	}
}

func FetchByIdOrName(manager IModelManager, userCred mcclient.IIdentityProvider, idStr string) (IModel, error) {
	if stringutils2.IsUtf8(idStr) {
		return FetchByName(manager, userCred, idStr)
	}
	obj, err := FetchById(manager, idStr)
	if err == sql.ErrNoRows {
		return FetchByName(manager, userCred, idStr)
	} else {
		return obj, err
	}
}

func fetchItemById(manager IModelManager, ctx context.Context, userCred mcclient.TokenCredential, idStr string, query jsonutils.JSONObject) (IModel, error) {
	q := manager.Query()
	var err error
	if query != nil && !query.IsZero() {
		// if isListRbacAllowed(manager, userCred, true) {
		// 	query.(*jsonutils.JSONDict).Set("admin", jsonutils.JSONTrue)
		// }
		q, err = listItemQueryFilters(manager, ctx, q, userCred, query, policy.PolicyActionGet, false)
		if err != nil {
			return nil, err
		}
	}
	q = manager.FilterById(q, idStr)
	count, err := q.CountWithError()
	if err != nil {
		return nil, err
	}
	if count == 1 {
		item, err := NewModelObject(manager)
		if err != nil {
			return nil, err
		}
		err = q.First(item)
		if err != nil {
			return nil, err
		}
		return item, nil
	} else if count > 1 {
		return nil, sqlchemy.ErrDuplicateEntry
	} else {
		return nil, sql.ErrNoRows
	}
}

func fetchItemByName(manager IModelManager, ctx context.Context, userCred mcclient.TokenCredential, idStr string, query jsonutils.JSONObject) (IModel, error) {
	q := manager.Query()
	var err error
	if query != nil && !query.IsZero() {
		q, err = listItemQueryFilters(manager, ctx, q, userCred, query, policy.PolicyActionGet, false)
		if err != nil {
			return nil, err
		}
	}
	q = manager.FilterByName(q, idStr)
	count, err := q.CountWithError()
	if err != nil {
		return nil, err
	}
	if count > 0 {
		q = manager.FilterByOwner(q, userCred, manager.NamespaceScope())
		q = manager.FilterBySystemAttributes(q, nil, nil, manager.ResourceScope())
		count, err = q.CountWithError()
		if err != nil {
			return nil, err
		}
	}
	if count == 1 {
		item, err := NewModelObject(manager)
		if err != nil {
			return nil, err
		}
		err = q.First(item)
		if err != nil {
			return nil, err
		}
		return item, nil
	} else if count > 1 {
		return nil, sqlchemy.ErrDuplicateEntry
	} else {
		return nil, sql.ErrNoRows
	}
}

func fetchItem(manager IModelManager, ctx context.Context, userCred mcclient.TokenCredential, idStr string, query jsonutils.JSONObject) (IModel, error) {
	item, err := fetchItemById(manager, ctx, userCred, idStr, query)
	if err != nil {
		item, err = fetchItemByName(manager, ctx, userCred, idStr, query)
	}
	return item, err
}

func FetchUserInfo(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	userStr, key := jsonutils.GetAnyString2(data, []string{"user", "user_id"})
	if len(userStr) > 0 {
		data.(*jsonutils.JSONDict).Remove(key)
		u, err := UserCacheManager.FetchUserByIdOrName(userStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("user", userStr)
			}
			return nil, errors.Wrap(err, "UserCacheManager.FetchUserByIdOrName")
		}
		ownerId := SOwnerId{
			UserDomain:   u.Domain,
			UserDomainId: u.DomainId,
			UserId:       u.Id,
			User:         u.Name,
		}
		return &ownerId, nil
	}
	return nil, nil
}

func FetchProjectInfo(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	tenantId, key := jsonutils.GetAnyString2(data, []string{"project", "project_id", "tenant", "tenant_id"})
	if len(tenantId) > 0 {
		data.(*jsonutils.JSONDict).Remove(key)
		t, err := TenantCacheManager.FetchTenantByIdOrName(ctx, tenantId)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("project", tenantId)
			}
			return nil, errors.Wrap(err, "TenantCacheManager.FetchTenantByIdOrName")
		}
		ownerId := SOwnerId{
			Domain:    t.Domain,
			DomainId:  t.DomainId,
			ProjectId: t.Id,
			Project:   t.Name,
		}
		data.(*jsonutils.JSONDict).Set("project", jsonutils.NewString(t.Id))
		data.(*jsonutils.JSONDict).Set("project_domain", jsonutils.NewString(t.DomainId))
		return &ownerId, nil
	}
	return FetchDomainInfo(ctx, data)
}

func FetchDomainInfo(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	domainId, key := jsonutils.GetAnyString2(data, []string{"domain_id", "project_domain", "project_domain_id"})
	if len(domainId) > 0 {
		data.(*jsonutils.JSONDict).Remove(key)
		domain, err := TenantCacheManager.FetchDomainByIdOrName(ctx, domainId)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("domain", domainId)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		owner := SOwnerId{DomainId: domain.Id, Domain: domain.Name}
		data.(*jsonutils.JSONDict).Set("project_domain", jsonutils.NewString(domain.Id))
		return &owner, nil
	}
	return nil, nil
}

type sUsageManager struct{}

func (m *sUsageManager) KeywordPlural() string {
	return "usages"
}

func (m *sUsageManager) ResourceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeProject
}

func (m *sUsageManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	return FetchProjectInfo(ctx, data)
}

func FetchUsageOwnerScope(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) (mcclient.IIdentityProvider, rbacutils.TRbacScope, error) {
	return FetchCheckQueryOwnerScope(ctx, userCred, data, &sUsageManager{}, policy.PolicyActionGet, true)
}

type IScopedResourceManager interface {
	KeywordPlural() string
	ResourceScope() rbacutils.TRbacScope
	FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error)
}

func FetchCheckQueryOwnerScope(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject, manager IScopedResourceManager, action string, doCheckRbac bool) (mcclient.IIdentityProvider, rbacutils.TRbacScope, error) {
	var scope rbacutils.TRbacScope

	var allowScope rbacutils.TRbacScope
	var requireScope rbacutils.TRbacScope
	var queryScope rbacutils.TRbacScope

	resScope := manager.ResourceScope()

	if consts.IsRbacEnabled() {
		allowScope = policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), manager.KeywordPlural(), action)
	} else {
		if userCred.HasSystemAdminPrivilege() {
			allowScope = rbacutils.ScopeSystem
		} else {
			allowScope = rbacutils.ScopeProject
			if resScope == rbacutils.ScopeUser {
				allowScope = rbacutils.ScopeUser
			}
		}
	}

	// var ownerId mcclient.IIdentityProvider
	// var err error

	ownerId, err := manager.FetchOwnerId(ctx, data)
	if err != nil {
		return nil, queryScope, err
	}
	if ownerId != nil {
		switch resScope {
		case rbacutils.ScopeProject, rbacutils.ScopeDomain:
			if len(ownerId.GetProjectId()) > 0 {
				queryScope = rbacutils.ScopeProject
				if ownerId.GetProjectId() == userCred.GetProjectId() {
					requireScope = rbacutils.ScopeProject
				} else if ownerId.GetProjectDomainId() == userCred.GetProjectDomainId() {
					requireScope = rbacutils.ScopeDomain
				} else {
					requireScope = rbacutils.ScopeSystem
				}
			} else if len(ownerId.GetProjectDomainId()) > 0 {
				queryScope = rbacutils.ScopeDomain
				if ownerId.GetProjectDomainId() == userCred.GetProjectDomainId() {
					requireScope = rbacutils.ScopeDomain
				} else {
					requireScope = rbacutils.ScopeSystem
				}
			}
		case rbacutils.ScopeUser:
			queryScope = rbacutils.ScopeUser
			if ownerId.GetUserId() == userCred.GetUserId() {
				requireScope = rbacutils.ScopeUser
			} else {
				requireScope = rbacutils.ScopeSystem
			}
		}
	}

	if ownerId == nil {
		ownerId = userCred
		reqScopeStr, _ := data.GetString("scope")
		if len(reqScopeStr) > 0 {
			queryScope = rbacutils.String2Scope(reqScopeStr)
		} else if data.Contains("admin") {
			isAdmin := jsonutils.QueryBoolean(data, "admin", false)
			if isAdmin && allowScope.HigherThan(rbacutils.ScopeProject) {
				queryScope = allowScope
			}
		} else if action == policy.PolicyActionGet {
			queryScope = allowScope
		}
		if resScope.HigherThan(queryScope) {
			queryScope = resScope
		}
		requireScope = queryScope
	}
	if doCheckRbac && requireScope.HigherThan(allowScope) {
		return nil, scope, httperrors.NewForbiddenError(fmt.Sprintf("not enough privilleges(require:%s,allow:%s,query:%s)", requireScope, allowScope, queryScope))
	}
	return ownerId, queryScope, nil
}
