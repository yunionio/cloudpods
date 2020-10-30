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
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
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
	parentScope := rbacutils.ScopeSystem
	scope, _ := data.GetString("scope")
	if scope != "" {
		parentScope = rbacutils.TRbacScope(scope)
	}
	uniqValues := sUniqValues{}
	switch parentScope {
	case rbacutils.ScopeSystem:
		uniqValues.Scope = scope
	case rbacutils.ScopeDomain:
		domain, _ := data.GetString("project_domain")
		uniqValues.Domain = domain
	case rbacutils.ScopeProject:
		project, _ := data.GetString("project")
		uniqValues.Project = project
	}
	return jsonutils.Marshal(uniqValues)
}

func (m *SScopedResourceBaseManager) FilterByScope(q *sqlchemy.SQuery, scope rbacutils.TRbacScope, scopeResId string) *sqlchemy.SQuery {
	isNotNullOrEmpty := func(field string) sqlchemy.ICondition {
		return sqlchemy.AND(sqlchemy.IsNotNull(q.Field(field)), sqlchemy.IsNotEmpty(q.Field(field)))
	}
	switch scope {
	case rbacutils.ScopeSystem:
		q = q.IsNullOrEmpty("domain_id").IsNullOrEmpty("tenant_id")
	case rbacutils.ScopeDomain:
		q = q.IsNullOrEmpty("tenant_id").Filter(isNotNullOrEmpty("domain_id"))
		if scopeResId != "" {
			q = q.Equals("domain_id", scopeResId)
		}
	case rbacutils.ScopeProject:
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
		return m.FilterByScope(q, rbacutils.TRbacScope(uniqValues.Scope), uniqValues.Domain)
	} else if len(uniqValues.Project) > 0 {
		return m.FilterByScope(q, rbacutils.TRbacScope(uniqValues.Scope), uniqValues.Project)
	} else {
		return m.FilterByScope(q, rbacutils.TRbacScope(uniqValues.Scope), "")
	}
}

func (m *SScopedResourceBase) IsOwner(userCred mcclient.TokenCredential) bool {
	scope := m.GetResourceScope()
	switch scope {
	case rbacutils.ScopeDomain:
		return userCred.GetProjectDomainId() == m.GetDomainId()
	case rbacutils.ScopeProject:
		return userCred.GetProjectId() == m.GetProjectId()
	}
	// system scope
	return userCred.HasSystemAdminPrivilege()
}

func (m *SScopedResourceBaseManager) FilterByOwner(q *sqlchemy.SQuery, userCred mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	if userCred == nil {
		return q
	}
	switch scope {
	case rbacutils.ScopeDomain:
		q = q.Equals("domain_id", userCred.GetProjectDomainId())
	case rbacutils.ScopeProject:
		q = q.Equals("tenant_id", userCred.GetProjectId())
	}
	return q
}

func (m *SScopedResourceBaseManager) ValidateCreateData(man IScopedResourceManager, ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input apis.ScopedResourceCreateInput) (apis.ScopedResourceCreateInput, error) {
	if input.Scope == "" {
		input.Scope = string(rbacutils.ScopeSystem)
	}
	if !utils.IsInStringArray(input.Scope, []string{
		string(rbacutils.ScopeSystem),
		string(rbacutils.ScopeDomain),
		string(rbacutils.ScopeProject)}) {
		return input, httperrors.NewInputParameterError("invalid scope %s", input.Scope)
	}
	var allowCreate bool
	switch rbacutils.TRbacScope(input.Scope) {
	case rbacutils.ScopeSystem:
		allowCreate = IsAdminAllowCreate(userCred, man)
	case rbacutils.ScopeDomain:
		allowCreate = IsDomainAllowCreate(userCred, man)
		input.ProjectDomainId = ownerId.GetDomainId()
	case rbacutils.ScopeProject:
		allowCreate = IsProjectAllowCreate(userCred, man)
		input.ProjectDomainId = ownerId.GetDomainId()
		input.ProjectId = ownerId.GetProjectId()
	}
	if !allowCreate {
		return input, httperrors.NewForbiddenError("not allow create %s in scope %s", man.ResourceScope(), input.Scope)
	}
	return input, nil
}

func getScopedResourceScope(domainId, projectId string) rbacutils.TRbacScope {
	if domainId == "" && projectId == "" {
		return rbacutils.ScopeSystem
	}
	if domainId != "" && projectId == "" {
		return rbacutils.ScopeDomain
	}
	if domainId != "" && projectId != "" {
		return rbacutils.ScopeProject
	}
	return rbacutils.ScopeNone
}

func (s *SScopedResourceBase) GetResourceScope() rbacutils.TRbacScope {
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
	switch rbacutils.TRbacScope(scope) {
	case rbacutils.ScopeSystem:
		s.DomainId = ""
		s.ProjectId = ""
	case rbacutils.ScopeDomain:
		s.DomainId = ownerId.GetDomainId()
		s.ProjectId = ""
	case rbacutils.ScopeProject:
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
	domainId := jsonutils.GetAnyString(data, []string{"domain_id", "domain", "project_domain_id", "project_domain"})
	projectId := jsonutils.GetAnyString(data, []string{"project_id", "project"})
	if projectId != "" {
		project, err := DefaultProjectFetcher(ctx, projectId)
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
	case rbacutils.ScopeSystem:
		err = setScopedResourceToSystem(obj, userCred)
	case rbacutils.ScopeDomain:
		err = setScopedResourceToDomain(obj, userCred, domainId)
	case rbacutils.ScopeProject:
		err = setScopedResourceToProject(obj, userCred, projectId)
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

func setScopedResourceToSystem(model IScopedResourceModel, userCred mcclient.TokenCredential) error {
	if !IsAdminAllowPerform(userCred, model, "set-scope") {
		return httperrors.NewForbiddenError("Not allow set scope to system")
	}
	if model.GetProjectId() == "" && model.GetDomainId() == "" {
		return nil
	}
	return setScopedResourceIds(model, userCred, "", "")
}

func setScopedResourceToDomain(model IScopedResourceModel, userCred mcclient.TokenCredential, domainId string) error {
	if !IsDomainAllowPerform(userCred, model, "set-scope") {
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

func setScopedResourceToProject(model IScopedResourceModel, userCred mcclient.TokenCredential, projectId string) error {
	if !IsProjectAllowPerform(userCred, model, "set-scope") {
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
		q = m.FilterByScope(q, rbacutils.TRbacScope(query.BelongScope), "")
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
				project, _ := DefaultProjectFetcher(ctx, base.ProjectId)
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
