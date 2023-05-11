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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SScopedResourceBaseManager struct {
	SProjectizedResourceBaseManager
}

// +onecloud:model-api-gen
type SScopedResourceBase struct {
	SProjectizedResourceBase `"domain_id->default":""`
}

type sUniqValues struct {
	Scope   string
	Project string
	Domain  string
}

func (m *SScopedResourceBaseManager) FetchUniqValues(ctx context.Context, data jsonutils.JSONObject) jsonutils.JSONObject {
	parentScope := rbacscope.ScopeSystem
	scope, _ := data.GetString("scope")
	if scope != "" {
		parentScope = rbacscope.TRbacScope(scope)
	}
	uniqValues := sUniqValues{}
	switch parentScope {
	case rbacscope.ScopeSystem:
	case rbacscope.ScopeDomain:
		domain, _ := data.GetString("project_domain")
		uniqValues.Domain = domain
	case rbacscope.ScopeProject:
		project, _ := data.GetString("project")
		uniqValues.Project = project
	}
	uniqValues.Scope = scope
	return jsonutils.Marshal(uniqValues)
}

func (m *SScopedResourceBaseManager) FilterByScope(q *sqlchemy.SQuery, scope rbacscope.TRbacScope, scopeResId string) *sqlchemy.SQuery {
	isNotNullOrEmpty := func(field string) sqlchemy.ICondition {
		return sqlchemy.AND(sqlchemy.IsNotNull(q.Field(field)), sqlchemy.IsNotEmpty(q.Field(field)))
	}
	switch scope {
	case rbacscope.ScopeSystem:
		q = q.IsNullOrEmpty("domain_id").IsNullOrEmpty("tenant_id")
	case rbacscope.ScopeDomain:
		q = q.IsNullOrEmpty("tenant_id").Filter(isNotNullOrEmpty("domain_id"))
		if scopeResId != "" {
			q = q.Equals("domain_id", scopeResId)
		}
	case rbacscope.ScopeProject:
		q = q.Filter(isNotNullOrEmpty("domain_id")).Filter(isNotNullOrEmpty("tenant_id"))
		if scopeResId != "" {
			q = q.Equals("tenant_id", scopeResId)
		}
	}
	return q
}

func (m *SScopedResourceBaseManager) FilterByUniqValues(q *sqlchemy.SQuery, values jsonutils.JSONObject) *sqlchemy.SQuery {
	uniqValues := &sUniqValues{}
	values.Unmarshal(uniqValues)
	if len(uniqValues.Domain) > 0 {
		return m.FilterByScope(q, rbacscope.TRbacScope(uniqValues.Scope), uniqValues.Domain)
	} else if len(uniqValues.Project) > 0 {
		return m.FilterByScope(q, rbacscope.TRbacScope(uniqValues.Scope), uniqValues.Project)
	} else {
		return m.FilterByScope(q, rbacscope.TRbacScope(uniqValues.Scope), "")
	}
}

func (m *SScopedResourceBase) IsOwner(userCred mcclient.TokenCredential) bool {
	scope := m.GetResourceScope()
	switch scope {
	case rbacscope.ScopeDomain:
		return userCred.GetProjectDomainId() == m.GetDomainId()
	case rbacscope.ScopeProject:
		return userCred.GetProjectId() == m.GetProjectId()
	}
	// system scope
	return userCred.HasSystemAdminPrivilege()
}

func (m *SScopedResourceBaseManager) FilterByOwner(q *sqlchemy.SQuery, man FilterByOwnerProvider, userCred mcclient.TokenCredential, owner mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	if owner == nil {
		return q
	}
	switch scope {
	case rbacscope.ScopeDomain:
		q = q.Equals("domain_id", owner.GetProjectDomainId())
	case rbacscope.ScopeProject:
		q = q.Equals("tenant_id", owner.GetProjectId())
	}
	return q
}

func (m *SScopedResourceBaseManager) ValidateCreateData(man IScopedResourceManager, ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input apis.ScopedResourceCreateInput) (apis.ScopedResourceCreateInput, error) {
	if input.Scope == "" {
		input.Scope = string(rbacscope.ScopeSystem)
	}
	if !utils.IsInStringArray(input.Scope, []string{
		string(rbacscope.ScopeSystem),
		string(rbacscope.ScopeDomain),
		string(rbacscope.ScopeProject)}) {
		return input, httperrors.NewInputParameterError("invalid scope %s", input.Scope)
	}
	var allowCreate bool
	switch rbacscope.TRbacScope(input.Scope) {
	case rbacscope.ScopeSystem:
		allowCreate = IsAdminAllowCreate(userCred, man).Result.IsAllow()
	case rbacscope.ScopeDomain:
		allowCreate = IsDomainAllowCreate(userCred, man).Result.IsAllow()
		input.ProjectDomainId = ownerId.GetDomainId()
	case rbacscope.ScopeProject:
		allowCreate = IsProjectAllowCreate(userCred, man).Result.IsAllow()
		input.ProjectDomainId = ownerId.GetDomainId()
		input.ProjectId = ownerId.GetProjectId()
	}
	if !allowCreate {
		return input, httperrors.NewForbiddenError("not allow create %s in scope %s", man.ResourceScope(), input.Scope)
	}
	return input, nil
}

func getScopedResourceScope(domainId, projectId string) rbacscope.TRbacScope {
	if domainId == "" && projectId == "" {
		return rbacscope.ScopeSystem
	}
	if domainId != "" && projectId == "" {
		return rbacscope.ScopeDomain
	}
	if domainId != "" && projectId != "" {
		return rbacscope.ScopeProject
	}
	return rbacscope.ScopeNone
}

func (s *SScopedResourceBase) GetResourceScope() rbacscope.TRbacScope {
	return getScopedResourceScope(s.DomainId, s.ProjectId)
}

func (s *SScopedResourceBase) GetDomainId() string {
	return s.DomainId
}

func (s *SScopedResourceBase) GetProjectId() string {
	return s.ProjectId
}

func (s *SScopedResourceBase) SetResourceScope(domainId, projectId string) error {
	s.DomainId = domainId
	s.ProjectId = projectId
	return nil
}

func (s *SScopedResourceBase) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	scope, _ := data.GetString("scope")
	switch rbacscope.TRbacScope(scope) {
	case rbacscope.ScopeSystem:
		s.DomainId = ""
		s.ProjectId = ""
	case rbacscope.ScopeDomain:
		s.DomainId = ownerId.GetDomainId()
		s.ProjectId = ""
	case rbacscope.ScopeProject:
		s.DomainId = ownerId.GetDomainId()
		s.ProjectId = ownerId.GetProjectId()
	}
	return nil
}

func (s *SScopedResourceBase) GetMoreColumns(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	if s.ProjectId != "" {
		extra.Add(jsonutils.NewString(s.ProjectId), "project_id")
	}
	return extra
}

type IScopedResourceModel interface {
	IModel

	GetDomainId() string
	GetProjectId() string
	SetResourceScope(domainId, projectId string) error
}

func PerformSetScope(
	ctx context.Context,
	obj IScopedResourceModel,
	userCred mcclient.TokenCredential,
	data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	domainId := jsonutils.GetAnyString(data, DomainFetchKeys) // []string{"domain_id", "domain", "project_domain_id", "project_domain"})
	projectId := jsonutils.GetAnyString(data, ProjectFetchKeys)
	if projectId != "" {
		project, err := DefaultProjectFetcher(ctx, projectId, domainId)
		if err != nil {
			return nil, err
		}
		projectId = project.GetProjectId()
		domainId = project.GetProjectDomainId()
	}
	if domainId != "" {
		domain, err := DefaultDomainFetcher(ctx, domainId)
		if err != nil {
			return nil, err
		}
		domainId = domain.GetProjectDomainId()
	}
	scopeToSet := getScopedResourceScope(domainId, projectId)
	var err error
	switch scopeToSet {
	case rbacscope.ScopeSystem:
		err = setScopedResourceToSystem(ctx, obj, userCred)
	case rbacscope.ScopeDomain:
		err = setScopedResourceToDomain(ctx, obj, userCred, domainId)
	case rbacscope.ScopeProject:
		err = setScopedResourceToProject(ctx, obj, userCred, projectId)
	}
	return nil, err
}

func setScopedResourceIds(model IScopedResourceModel, userCred mcclient.TokenCredential, domainId, projectId string) error {
	diff, err := Update(model, func() error {
		model.SetResourceScope(domainId, projectId)
		return nil
	})
	if err == nil {
		OpsLog.LogEvent(model, ACT_UPDATE, diff, userCred)
	}
	return err
}

func setScopedResourceToSystem(ctx context.Context, model IScopedResourceModel, userCred mcclient.TokenCredential) error {
	if !IsAdminAllowPerform(ctx, userCred, model, "set-scope") {
		return httperrors.NewForbiddenError("Not allow set scope to system")
	}
	if model.GetProjectId() == "" && model.GetDomainId() == "" {
		return nil
	}
	return setScopedResourceIds(model, userCred, "", "")
}

func setScopedResourceToDomain(ctx context.Context, model IScopedResourceModel, userCred mcclient.TokenCredential, domainId string) error {
	if !IsDomainAllowPerform(ctx, userCred, model, "set-scope") {
		return httperrors.NewForbiddenError("Not allow set scope to domain %s", domainId)
	}
	if model.GetDomainId() == domainId && model.GetProjectId() == "" {
		return nil
	}
	domain, err := TenantCacheManager.FetchDomainById(context.TODO(), domainId)
	if err != nil {
		return err
	}
	return setScopedResourceIds(model, userCred, domain.GetId(), "")
}

func setScopedResourceToProject(ctx context.Context, model IScopedResourceModel, userCred mcclient.TokenCredential, projectId string) error {
	if !IsProjectAllowPerform(ctx, userCred, model, "set-scope") {
		return httperrors.NewForbiddenError("Not allow set scope to project %s", projectId)
	}
	if model.GetProjectId() == projectId {
		return nil
	}
	project, err := TenantCacheManager.FetchTenantById(context.TODO(), projectId)
	if err != nil {
		return err
	}
	return setScopedResourceIds(model, userCred, project.GetProjectDomainId(), projectId)
}

func (m *SScopedResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.ScopedResourceBaseListInput,
) (*sqlchemy.SQuery, error) {
	q, err := m.SProjectizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ProjectizedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SProjectizedResourceBaseManager.ListItemFilter")
	}
	if query.BelongScope != "" {
		q = m.FilterByScope(q, rbacscope.TRbacScope(query.BelongScope), "")
	}
	return q, nil
}

func (m *SScopedResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.ScopedResourceBaseListInput,
) (*sqlchemy.SQuery, error) {
	q, err := m.SProjectizedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ProjectizedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SProjectizedResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SScopedResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []apis.ScopedResourceBaseInfo {
	rows := make([]apis.ScopedResourceBaseInfo, len(objs))

	projRows := manager.SProjectizedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = apis.ScopedResourceBaseInfo{
			ProjectizedResourceInfo: projRows[i],
		}
		var base *SScopedResourceBase
		reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if base != nil {
			if base.ProjectId != "" {
				project, _ := DefaultProjectFetcher(ctx, base.ProjectId, base.DomainId)
				if project != nil {
					rows[i].Project = project.Name
					rows[i].ProjectDomain = project.Domain
				}
			} else if base.DomainId != "" {
				domain, _ := DefaultDomainFetcher(ctx, base.DomainId)
				if domain != nil {
					rows[i].ProjectDomain = domain.Name
				}
			}
			rows[i].Scope = string(base.GetResourceScope())
		}
	}

	return rows
}
