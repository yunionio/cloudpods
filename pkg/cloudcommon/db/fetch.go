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
	"reflect"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/rbacscope"
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
		q = manager.FilterByOwner(q, manager, nil, userCred, manager.NamespaceScope())
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
		if gotypes.IsNil(query) {
			query = jsonutils.NewDict()
		}
		ownerId, _, err, _ := FetchCheckQueryOwnerScope(ctx, userCred, query, manager, rbacutils.ActionGet, true)
		if err != nil {
			return nil, httperrors.NewGeneralError(err)
		}
		q = manager.FilterByOwner(q, manager, userCred, ownerId, manager.NamespaceScope())
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
	if err != nil {
		return nil, err
	}
	if err := CheckRecordChecksumConsistent(item); err != nil {
		return nil, err
	}
	return item, nil
}

func FetchUserInfo(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	userStr, key := jsonutils.GetAnyString2(data, []string{
		"user_id",
		"user",
	})
	if len(userStr) > 0 {
		data.(*jsonutils.JSONDict).Remove(key)
		u, err := DefaultUserFetcher(ctx, userStr)
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
	return FetchProjectInfo(ctx, data)
}

var (
	ProjectFetchKeys = []string{
		"project_id",
		"tenant_id",
		"project",
		"tenant",
	}
	DomainFetchKeys = []string{
		"project_domain_id",
		"domain_id",
		"project_domain",
		"domain",
	}
)

func FetchProjectInfo(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	tenantId, key := jsonutils.GetAnyString2(data, ProjectFetchKeys)
	if len(tenantId) > 0 {
		data.(*jsonutils.JSONDict).Remove(key)
		domainId, _ := jsonutils.GetAnyString2(data, DomainFetchKeys)
		t, err := DefaultProjectFetcher(ctx, tenantId, domainId)
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
		// 当资源的域和归属云账号的域不同时，会导致查找不到该资源
		// data.(*jsonutils.JSONDict).Set("project_domain", jsonutils.NewString(t.DomainId))
		return &ownerId, nil
	}
	return FetchDomainInfo(ctx, data)
}

func FetchDomainInfo(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	domainId, key := jsonutils.GetAnyString2(data, DomainFetchKeys)
	if len(domainId) > 0 {
		data.(*jsonutils.JSONDict).Remove(key)
		domain, err := DefaultDomainFetcher(ctx, domainId)
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

func (m *sUsageManager) ResourceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeProject
}

func (m *sUsageManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	return FetchProjectInfo(ctx, data)
}

func FetchUsageOwnerScope(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) (mcclient.IIdentityProvider, rbacscope.TRbacScope, error, rbacutils.SPolicyResult) {
	return FetchCheckQueryOwnerScope(ctx, userCred, data, &sUsageManager{}, policy.PolicyActionGet, true)
}

type IScopedResourceManager interface {
	KeywordPlural() string
	ResourceScope() rbacscope.TRbacScope
	FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error)
}

func UsagePolicyCheck(userCred mcclient.TokenCredential, manager IScopedResourceManager, scope rbacscope.TRbacScope) rbacutils.SPolicyResult {
	allowScope, policyTagFilters := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionList)
	if scope.HigherThan(allowScope) {
		return rbacutils.SPolicyResult{Result: rbacutils.Deny}
	}
	return policyTagFilters
}

func FetchCheckQueryOwnerScope(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	data jsonutils.JSONObject,
	manager IScopedResourceManager,
	action string,
	doCheckRbac bool,
) (mcclient.IIdentityProvider, rbacscope.TRbacScope, error, rbacutils.SPolicyResult) {
	var scope rbacscope.TRbacScope

	var allowScope rbacscope.TRbacScope
	var requireScope rbacscope.TRbacScope
	var queryScope rbacscope.TRbacScope
	var policyTagFilters rbacutils.SPolicyResult

	resScope := manager.ResourceScope()

	allowScope, policyTagFilters = policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), manager.KeywordPlural(), action)

	ownerId, err := manager.FetchOwnerId(ctx, data)
	if err != nil {
		return nil, queryScope, errors.Wrap(err, "FetchOwnerId"), policyTagFilters
	}
	if ownerId != nil {
		switch resScope {
		case rbacscope.ScopeProject, rbacscope.ScopeDomain:
			if len(ownerId.GetProjectId()) > 0 {
				queryScope = rbacscope.ScopeProject
				if ownerId.GetProjectId() == userCred.GetProjectId() {
					requireScope = rbacscope.ScopeProject
				} else if ownerId.GetProjectDomainId() == userCred.GetProjectDomainId() {
					requireScope = rbacscope.ScopeDomain
				} else {
					requireScope = rbacscope.ScopeSystem
				}
			} else if len(ownerId.GetProjectDomainId()) > 0 {
				queryScope = rbacscope.ScopeDomain
				if ownerId.GetProjectDomainId() == userCred.GetProjectDomainId() {
					requireScope = rbacscope.ScopeDomain
				} else {
					requireScope = rbacscope.ScopeSystem
				}
			}
		case rbacscope.ScopeUser:
			queryScope = rbacscope.ScopeUser
			if ownerId.GetUserId() == userCred.GetUserId() {
				requireScope = rbacscope.ScopeUser
			} else {
				requireScope = rbacscope.ScopeSystem
			}
		}
	} else {
		ownerId = userCred
		reqScopeStr, _ := data.GetString("scope")
		if len(reqScopeStr) > 0 {
			if reqScopeStr == "max" || reqScopeStr == "maxallowed" {
				queryScope = allowScope
			} else {
				queryScope = rbacscope.String2Scope(reqScopeStr)
			}
		} else if data.Contains("admin") {
			isAdmin := jsonutils.QueryBoolean(data, "admin", false)
			if isAdmin && allowScope.HigherThan(rbacscope.ScopeProject) {
				queryScope = allowScope
			}
		} else if action == policy.PolicyActionGet {
			queryScope = allowScope
		} else {
			queryScope = resScope
		}
		// if resScope.HigherThan(queryScope) {
		// 	queryScope = resScope
		// }
		requireScope = queryScope
	}
	if doCheckRbac && (requireScope.HigherThan(allowScope) || policyTagFilters.Result.IsDeny()) {
		return nil, scope, httperrors.NewForbiddenError("not enough privilege to do %s:%s:%s (require:%s,allow:%s,query:%s)",
			consts.GetServiceType(), manager.KeywordPlural(), action,
			requireScope, allowScope, queryScope), policyTagFilters
	}
	return ownerId, queryScope, nil, policyTagFilters
}

func mapKeys(idMap map[string]string) []string {
	keys := make([]string, len(idMap))
	idx := 0
	for k := range idMap {
		keys[idx] = k
		idx += 1
	}
	return keys
}

func FetchIdNameMap2(manager IStandaloneModelManager, ids []string) (map[string]string, error) {
	idMap := make(map[string]string, len(ids))
	for _, id := range ids {
		idMap[id] = ""
	}
	return FetchIdNameMap(manager, idMap)
}

func FetchIdNameMap(manager IStandaloneModelManager, idMap map[string]string) (map[string]string, error) {
	return FetchIdFieldMap(manager, "name", idMap)
}

func FetchIdFieldMap2(manager IStandaloneModelManager, field string, ids []string) (map[string]string, error) {
	idMap := make(map[string]string, len(ids))
	for _, id := range ids {
		idMap[id] = ""
	}
	return FetchIdFieldMap(manager, field, idMap)
}

func FetchIdFieldMap(manager IStandaloneModelManager, field string, idMap map[string]string) (map[string]string, error) {
	q := manager.Query("id", field).In("id", mapKeys(idMap))
	rows, err := q.Rows()
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return idMap, nil
		} else {
			return idMap, errors.Wrap(err, "Query")
		}
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var name string
		err := rows.Scan(&id, &name)
		if err != nil {
			return idMap, errors.Wrap(err, "rows.Scan")
		}
		idMap[id] = name
	}
	return idMap, nil
}

func FetchModelObjectsByIds(modelManager IModelManager, fieldName string, ids []string, targets interface{}) error {
	err := FetchQueryObjectsByIds(modelManager.Query(), fieldName, ids, targets)
	if err != nil {
		return errors.Wrap(err, "FetchQueryObjectsByIds")
	}
	// try to call model's SetModelManager
	targetValue := reflect.Indirect(reflect.ValueOf(targets))
	for _, key := range targetValue.MapKeys() {
		modelValueV := targetValue.MapIndex(key)
		if modelValueV.Kind() != reflect.Struct {
			break
		}
		newModelValue := reflect.New(modelValueV.Type()).Elem()
		newModelValue.Set(modelValueV)
		modelValue := newModelValue.Addr().Interface()
		if model, ok := modelValue.(IModel); ok {
			model.SetModelManager(modelManager, model)
			targetValue.SetMapIndex(key, reflect.Indirect(reflect.ValueOf(model)))
		}
	}
	return nil
}

func FetchQueryObjectsByIds(q *sqlchemy.SQuery, fieldName string, ids []string, targets interface{}) error {
	if len(ids) == 0 {
		return nil
	}

	targetValue := reflect.Indirect(reflect.ValueOf(targets))
	if targetValue.Kind() != reflect.Map {
		return errors.Wrap(httperrors.ErrBadRequest, "receiver should be a map")
	}

	isTargetSlice := false
	modelType := targetValue.Type().Elem()
	if modelType.Kind() == reflect.Slice {
		isTargetSlice = true
		modelType = modelType.Elem()
	}

	query := q.In(fieldName, ids)
	rows, err := query.Rows()
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}
	defer rows.Close()

	targetsValue := reflect.Indirect(reflect.ValueOf(targets))
	for rows.Next() {
		mMap, err := query.Row2Map(rows)
		if err != nil {
			return errors.Wrap(err, "query.Row2Map")
		}
		fieldValue := mMap[fieldName]
		m := reflect.New(modelType).Interface() // a pointer
		err = query.RowMap2Struct(mMap, m)
		if err != nil {
			return errors.Wrap(err, "query.RowMap2Struct")
		}
		keyValue := reflect.ValueOf(fieldValue)
		valValue := reflect.Indirect(reflect.ValueOf(m))
		if isTargetSlice {
			sliceValue := targetValue.MapIndex(keyValue)
			if !sliceValue.IsValid() {
				sliceValue = reflect.New(reflect.SliceOf(modelType)).Elem()
			}
			sliceValue = reflect.Append(sliceValue, valValue)
			targetsValue.SetMapIndex(keyValue, sliceValue)
		} else {
			targetsValue.SetMapIndex(keyValue, valValue)
		}
	}
	return nil
}

func FetchStandaloneObjectsByIds(modelManager IModelManager, ids []string, targets interface{}) error {
	return FetchModelObjectsByIds(modelManager, "id", ids, targets)
}

func FetchField(modelMan IModelManager, field string, qCallback func(q *sqlchemy.SQuery) *sqlchemy.SQuery) ([]string, error) {
	q := modelMan.Query(field)
	if qCallback != nil {
		q = qCallback(q)
	}
	rows, err := q.Rows()
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "q.Rows")
	}
	defer rows.Close()

	values := []string{}
	for rows.Next() {
		var value sql.NullString
		err := rows.Scan(&value)
		if err != nil {
			return values, errors.Wrap(err, "rows.Scan")
		}
		if value.Valid {
			values = append(values, value.String)
		}
	}
	return values, nil
}

func FetchDistinctField(modelManager IModelManager, field string) ([]string, error) {
	return FetchField(modelManager, field, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.Distinct()
	})
}
