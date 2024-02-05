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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/yunionconf"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-model-singular=scopedpolicybinding
// +onecloud:swagger-gen-model-plural=scopedpolicybindings
type SScopedPolicyBindingManager struct {
	db.SResourceBaseManager
}

type SScopedPolicyBinding struct {
	db.SResourceBase

	Category  string `width:"64" charset:"utf8" nullable:"false" primary:"true" list:"user"`
	DomainId  string `width:"128" charset:"ascii" nullable:"false" primary:"true" list:"user"`
	ProjectId string `width:"128" charset:"ascii" nullable:"false" primary:"true" list:"user"`

	PolicyId string `width:"128" charset:"ascii" nullable:"false" list:"user"`

	Priority int `nullable:"false" list:"user"`
}

var ScopedPolicyBindingManager *SScopedPolicyBindingManager

func init() {
	ScopedPolicyBindingManager = &SScopedPolicyBindingManager{
		SResourceBaseManager: db.NewResourceBaseManager(
			SScopedPolicyBinding{},
			"scopedpolicybinding_tbl",
			"scopedpolicybinding",
			"scopedpolicybindings",
		),
	}
	ScopedPolicyBindingManager.SetVirtualObject(ScopedPolicyBindingManager)
}

func (binding *SScopedPolicyBinding) GetId() string {
	return strings.Join([]string{binding.Category, binding.DomainId, binding.ProjectId}, ":")
}

func (binding *SScopedPolicyBinding) GetName() string {
	return binding.GetId()
}

func (binding *SScopedPolicyBinding) calculatePriority() (int, error) {
	if binding.DomainId == "" && binding.ProjectId == "" {
		// system level
		return 0, nil
	} else if len(binding.DomainId) > 0 && binding.ProjectId == "" {
		// domain level
		if binding.DomainId == api.ANY_DOMAIN_ID {
			return 10, nil
		} else {
			return 11, nil
		}
	} else if len(binding.DomainId) > 0 && len(binding.ProjectId) > 0 {
		// project level
		if binding.ProjectId == api.ANY_PROJECT_ID {
			if binding.DomainId == api.ANY_DOMAIN_ID {
				return 100, nil
			} else {
				return 101, nil
			}
		} else {
			if binding.DomainId == api.ANY_DOMAIN_ID {
				return -1, errors.Wrap(httperrors.ErrInvalidStatus, "domain_id cannot be any domain when project_id is specific")
			} else {
				return 102, nil
			}
		}
	} else {
		return -1, errors.Wrap(httperrors.ErrInvalidStatus, "invalid domain_id and project_id")
	}
}

func (manager *SScopedPolicyBindingManager) bind(ctx context.Context, category, policyId, domainId, projectId string) error {
	binding := SScopedPolicyBinding{
		Category:  category,
		DomainId:  domainId,
		ProjectId: projectId,
		PolicyId:  policyId,
	}
	var err error
	binding.Priority, err = binding.calculatePriority()
	if err != nil {
		return errors.Wrap(err, "calculatePriority")
	}
	binding.SetModelManager(manager, &binding)
	err = manager.TableSpec().InsertOrUpdate(ctx, &binding)
	if err != nil {
		return errors.Wrap(err, "InsertOrUpdate")
	}
	return nil
}

func (manager *SScopedPolicyBindingManager) unbind(ctx context.Context, category, domainId, projectId string) error {
	binding := SScopedPolicyBinding{
		Category:  category,
		DomainId:  domainId,
		ProjectId: projectId,
	}
	binding.SetModelManager(manager, &binding)
	_, err := db.Update(&binding, func() error {
		return binding.MarkDelete()
	})
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil
		}
		return errors.Wrap(err, "MarkDelete")
	}
	return nil
}

func (manager *SScopedPolicyBindingManager) ResourceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeProject
}

func (manager *SScopedPolicyBindingManager) getReferenceCount(policyId string) (int, error) {
	q := manager.Query().Equals("policy_id", policyId)
	return q.CountWithError()
}

// 范围策略列表
func (manager *SScopedPolicyBindingManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ScopedPolicyBindingListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SResourceBaseManager.ListItemFilter")
	}

	var policySubq *sqlchemy.SSubQuery
	if len(query.Name) > 0 {
		subq := ScopedPolicyManager.Query("id")
		subq = subq.Filter(sqlchemy.ContainsAny(subq.Field("name"), query.Name))
		policySubq = subq.SubQuery()
	}

	if len(query.PolicyId) > 0 {
		policyObj, err := ScopedPolicyManager.FetchByIdOrName(ctx, userCred, query.PolicyId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s", ScopedPolicyManager.Keyword(), query.PolicyId)
			} else {
				return nil, errors.Wrap(err, "ScopedPolicyManager.FetchByIdOrName")
			}
		}
		query.PolicyId = policyObj.GetId()
	}

	if len(query.ProjectId) > 0 {
		projObj, err := db.TenantCacheManager.FetchTenantByIdOrNameInDomain(ctx, query.ProjectId, query.DomainId)
		if err != nil {
			return nil, errors.Wrap(err, "TenantCacheManager.FetchTenantByIdOrName")
		}
		query.DomainId = projObj.DomainId
		query.ProjectId = projObj.Id
	} else if len(query.DomainId) > 0 {
		domainObj, err := db.TenantCacheManager.FetchDomainByIdOrName(ctx, query.DomainId)
		if err != nil {
			return nil, errors.Wrap(err, "TenantCacheManager.FetchDomainByIdOrName")
		}
		query.DomainId = domainObj.Id
	} else {
		if len(query.Scope) == 0 {
			query.Scope = rbacscope.ScopeProject
		}
		switch query.Scope {
		case rbacscope.ScopeProject:
			query.ProjectId = userCred.GetProjectId()
			query.DomainId = userCred.GetProjectDomainId()
		case rbacscope.ScopeDomain:
			query.DomainId = userCred.GetProjectDomainId()
		}
	}

	var requireScope rbacscope.TRbacScope
	if len(query.ProjectId) > 0 {
		if query.ProjectId == userCred.GetProjectId() {
			// require project privileges
			requireScope = rbacscope.ScopeProject
		} else if query.DomainId == userCred.GetProjectDomainId() {
			// require domain privileges
			requireScope = rbacscope.ScopeDomain
		} else {
			// require system privileges
			requireScope = rbacscope.ScopeSystem
		}
	} else if len(query.DomainId) > 0 {
		if query.DomainId == userCred.GetProjectDomainId() {
			// require domain privileges
			requireScope = rbacscope.ScopeDomain
		} else {
			// require system privileges
			requireScope = rbacscope.ScopeSystem
		}
	} else {
		requireScope = rbacscope.ScopeSystem
	}

	allowScope, _ := policy.PolicyManager.AllowScope(userCred, api.SERVICE_TYPE, manager.KeywordPlural(), policy.PolicyActionList)
	if requireScope.HigherThan(allowScope) {
		return nil, errors.Wrapf(httperrors.ErrNotSufficientPrivilege, "require: %s allow: %s", requireScope, allowScope)
	}

	effective := (query.Effective != nil && *query.Effective)
	q = filterByOwnerId(q, query.DomainId, query.ProjectId, effective, query.Category, query.PolicyId, policySubq)
	if query.Effective != nil && *query.Effective && (len(query.ProjectId) > 0 || len(query.DomainId) > 0) {
		bindingQ := manager.Query().SubQuery()
		maxq := bindingQ.Query(bindingQ.Field("category"), sqlchemy.MAX("max_priority", bindingQ.Field("priority")))
		maxq = filterByOwnerId(maxq, query.DomainId, query.ProjectId, *query.Effective, query.Category, query.PolicyId, policySubq)
		maxq = maxq.GroupBy(bindingQ.Field("category"))
		subq := maxq.SubQuery()
		q = q.Join(subq, sqlchemy.Equals(q.Field("category"), subq.Field("category")))
		q = q.Filter(sqlchemy.Equals(q.Field("priority"), subq.Field("max_priority")))
	}

	return q, nil
}

func filterByOwnerId(q *sqlchemy.SQuery, domainId, projectId string, effective bool, category []string, policyId string, policySubq *sqlchemy.SSubQuery) *sqlchemy.SQuery {
	if len(projectId) > 0 {
		q = q.Filter(sqlchemy.OR(
			sqlchemy.AND(sqlchemy.IsNullOrEmpty(q.Field("domain_id")), sqlchemy.IsNullOrEmpty(q.Field("project_id"))),
			sqlchemy.AND(sqlchemy.Equals(q.Field("domain_id"), api.ANY_DOMAIN_ID), sqlchemy.OR(
				sqlchemy.Equals(q.Field("project_id"), api.ANY_PROJECT_ID),
				sqlchemy.IsNullOrEmpty(q.Field("project_id")),
			)),
			sqlchemy.AND(sqlchemy.Equals(q.Field("domain_id"), domainId), sqlchemy.OR(
				sqlchemy.IsNullOrEmpty(q.Field("project_id")),
				sqlchemy.Equals(q.Field("project_id"), api.ANY_PROJECT_ID),
				sqlchemy.Equals(q.Field("project_id"), projectId),
			)),
		))
	} else if len(domainId) > 0 {
		q = q.IsNullOrEmpty("project_id")
		q = q.Filter(sqlchemy.OR(
			sqlchemy.IsNullOrEmpty(q.Field("domain_id")),
			sqlchemy.Equals(q.Field("domain_id"), api.ANY_DOMAIN_ID),
			sqlchemy.Equals(q.Field("domain_id"), domainId),
		))
	} else if effective {
		q = q.IsNullOrEmpty("project_id")
		q = q.IsNullOrEmpty("domain_id")
	}
	if len(category) > 0 {
		q = q.In("category", category)
	}
	if len(policyId) > 0 {
		q = q.Equals("policy_id", policyId)
	}
	if policySubq != nil {
		q = q.In("policy_id", policySubq)
	}
	return q
}

func (manager *SScopedPolicyBindingManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ScopedPolicyBindingListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SResourceBaseManager.OrderByExtraFields")
	}

	if db.NeedOrderQuery([]string{query.OrderByScopedpolicy}) {
		subq := ScopedPolicyManager.Query("id", "name").SubQuery()
		q = q.Join(subq, sqlchemy.Equals(subq.Field("id"), q.Field("policy_id")))
		q = db.OrderByFields(q,
			[]string{query.OrderByScopedpolicy},
			[]sqlchemy.IQueryField{subq.Field("name")})
	}

	return q, nil
}

func (manager *SScopedPolicyBindingManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (manager *SScopedPolicyBindingManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ScopedPolicyBindingDetails {
	rows := make([]api.ScopedPolicyBindingDetails, len(objs))

	policyIds := make([]string, 0)
	domainIds := make([]string, 0)
	projectIds := make([]string, 0)
	stdRows := manager.SResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.ScopedPolicyBindingDetails{
			ResourceBaseDetails: stdRows[i],
		}
		binding := objs[i].(*SScopedPolicyBinding)
		rows[i].Id = binding.GetId()
		policyIds = append(policyIds, binding.PolicyId)
		if len(binding.ProjectId) > 0 && binding.ProjectId != api.ANY_PROJECT_ID {
			projectIds = append(projectIds, binding.ProjectId)
		} else if len(binding.DomainId) > 0 && binding.DomainId != api.ANY_DOMAIN_ID {
			domainIds = append(domainIds, binding.DomainId)
		}
	}

	policies := make(map[string]SScopedPolicy)
	err := db.FetchModelObjectsByIds(ScopedPolicyManager, "id", policyIds, policies)
	if err != nil {
		return rows
	}

	domains := db.DefaultProjectsFetcher(ctx, domainIds, true)
	projects := db.DefaultProjectsFetcher(ctx, projectIds, false)

	for i := range rows {
		binding := objs[i].(*SScopedPolicyBinding)
		if policy, ok := policies[binding.PolicyId]; ok {
			rows[i].PolicyName = policy.Name
			rows[i].Policies = policy.Policies
		}
		if len(binding.ProjectId) > 0 && binding.ProjectId != api.ANY_PROJECT_ID {
			if project, ok := projects[binding.ProjectId]; ok {
				rows[i].Project = project.Name
				rows[i].ProjectDomain = project.Domain
			}
		} else if len(binding.DomainId) > 0 && binding.DomainId != api.ANY_DOMAIN_ID {
			if domain, ok := domains[binding.DomainId]; ok {
				rows[i].ProjectDomain = domain.Name
			}
		}

	}

	return rows
}

func (manager *SScopedPolicyBindingManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SResourceBaseManager.ListItemExportKeys")
	}
	return q, nil
}

func (manager *SScopedPolicyBindingManager) FilterById(q *sqlchemy.SQuery, idStr string) *sqlchemy.SQuery {
	parts := strings.Split(idStr, ":")
	if len(parts) == 3 {
		return q.Equals("category", parts[0]).Equals("domain_id", parts[1]).Equals("project_id", parts[2])
	} else {
		return q.Equals("category", idStr)
	}
}

func (manager *SScopedPolicyBindingManager) FilterByNotId(q *sqlchemy.SQuery, idStr string) *sqlchemy.SQuery {
	parts := strings.Split(idStr, ":")
	if len(parts) == 3 {
		return q.Filter(sqlchemy.OR(
			sqlchemy.NotEquals(q.Field("category"), parts[0]),
			sqlchemy.NotEquals(q.Field("domain_id"), parts[1]),
			sqlchemy.NotEquals(q.Field("project_id"), parts[2]),
		))
	} else {
		return q
	}
}

func (manager *SScopedPolicyBindingManager) FilterByName(q *sqlchemy.SQuery, name string) *sqlchemy.SQuery {
	return manager.FilterById(q, name)
}

func (manager *SScopedPolicyBindingManager) GetResourceCount() ([]db.SScopeResourceCount, error) {
	cnt := make([]db.SScopeResourceCount, 0)
	projCnt, err := manager.getProjectResourceCount()
	if err != nil {
		return nil, errors.Wrap(err, "getProjectResourceCount")
	}
	domainCnt, err := manager.getDomainResourceCount()
	if err != nil {
		return nil, errors.Wrap(err, "getDomainResourceCount")
	}
	cnt = append(cnt, projCnt...)
	cnt = append(cnt, domainCnt...)
	return cnt, nil
}

func (manager *SScopedPolicyBindingManager) getProjectResourceCount() ([]db.SScopeResourceCount, error) {
	subq := manager.Query().SubQuery()
	q := subq.Query(subq.Field("project_id").Label("tenant_id")).IsNotEmpty("project_id").NotEquals("project_id", api.ANY_PROJECT_ID)
	return db.CalculateResourceCount(q, "tenant_id")
}

func (manager *SScopedPolicyBindingManager) getDomainResourceCount() ([]db.SScopeResourceCount, error) {
	subq := manager.Query().SubQuery()
	q := subq.Query(subq.Field("domain_id")).IsNotEmpty("domain_id").NotEquals("domain_id", api.ANY_DOMAIN_ID)
	return db.CalculateResourceCount(q, "domain_id")
}
