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

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
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
	if count > 1 && userCred != nil {
		q = manager.FilterByOwner(q, userCred)
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
		q, err = listItemQueryFilters(manager, ctx, q, userCred, query)
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
		q, err = listItemQueryFilters(manager, ctx, q, userCred, query)
		if err != nil {
			return nil, err
		}
	}
	q = manager.FilterByName(q, idStr)
	count, err := q.CountWithError()
	if err != nil {
		return nil, err
	}
	if count > 1 {
		q = manager.FilterByOwner(q, userCred)
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

func FetchProjectInfo(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	tenantId := jsonutils.GetAnyString(data, []string{"project", "project_id", "tenant", "tenant_id"})
	if len(tenantId) > 0 {
		t, err := TenantCacheManager.FetchTenantByIdOrName(ctx, tenantId)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("project", tenantId)
			}
			return nil, errors.Wrap(err, "FetchTenantByIdOrName")
		}
		ownerId := SOwnerId{
			Domain:    t.Domain,
			DomainId:  t.DomainId,
			ProjectId: t.Id,
			Project:   t.Name,
		}
		return &ownerId, nil
	}
	return FetchDomainInfo(ctx, data)
}

func FetchDomainInfo(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	domainId := jsonutils.GetAnyString(data, []string{"domain", "domain_id", "project_domain", "project_domain_id"})
	if len(domainId) > 0 {
		domain, err := TenantCacheManager.FetchDomainByIdOrName(ctx, domainId)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("domain", domainId)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		owner := SOwnerId{DomainId: domain.Id, Domain: domain.Name}
		return &owner, nil
	}
	return nil, nil
}

func FetchUsageOwnerScope(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) (mcclient.IIdentityProvider, rbacutils.TRbacScope, error) {
	return FetchQueryOwnerScope(ctx, userCred, data, "usages", policy.PolicyActionGet)
}

func FetchQueryOwnerScope(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject, resource string, action string) (mcclient.IIdentityProvider, rbacutils.TRbacScope, error) {
	var scope rbacutils.TRbacScope
	ownerId, err := FetchProjectInfo(ctx, data)
	if err != nil {
		return nil, scope, err
	}
	ownerScope := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), resource, action)
	if ownerId != nil {
		var requestScope rbacutils.TRbacScope
		if len(ownerId.GetProjectId()) > 0 {
			// project level
			scope = rbacutils.ScopeProject
			if ownerId.GetProjectId() == userCred.GetProjectId() {
				requestScope = rbacutils.ScopeProject
			} else if ownerId.GetProjectDomainId() == userCred.GetProjectDomainId() {
				requestScope = rbacutils.ScopeDomain
			} else {
				requestScope = rbacutils.ScopeSystem
			}
		} else {
			// domain level if len(ownerId.GetProjectDomainId()) > 0 {
			scope = rbacutils.ScopeDomain
			if ownerId.GetProjectDomainId() == userCred.GetProjectDomainId() {
				requestScope = rbacutils.ScopeDomain
			} else {
				requestScope = rbacutils.ScopeSystem
			}
		}
		if requestScope.HigherThan(ownerScope) {
			return nil, scope, httperrors.NewForbiddenError("not enough privilleges")
		}
	} else {
		ownerId = userCred
		scope = ownerScope
	}
	return ownerId, scope, nil
}
