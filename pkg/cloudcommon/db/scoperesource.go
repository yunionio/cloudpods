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
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SScopedResourceBaseManager struct{}

type SScopedResourceBase struct {
	DomainId  string `width:"64" charset:"ascii" nullable:"true" index:"true" list:"user"`
	ProjectId string `name:"tenant_id" width:"128" charset:"ascii" nullable:"true" index:"true" list:"user"`
}

func (m *SScopedResourceBaseManager) ResourceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeProject
}

func (m *SScopedResourceBaseManager) FilterByOwner(q *sqlchemy.SQuery, userCred mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	if userCred == nil {
		return q
	}
	switch scope {
	case rbacutils.ScopeDomain:
		q = q.Filter(sqlchemy.Equals(q.Field("domain_id"), userCred.GetDomainId()))
	case rbacutils.ScopeProject:
		q = q.Filter(sqlchemy.Equals(q.Field("tenant_id"), userCred.GetProjectId()))
	}
	return q
}

func (m *SScopedResourceBaseManager) ValidateCreateData(man IScopedResourceManager, ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	scope, _ := data.GetString("scope")
	if scope == "" {
		scope = string(rbacutils.ScopeSystem)
	}
	if !utils.IsInStringArray(scope, []string{
		string(rbacutils.ScopeSystem),
		string(rbacutils.ScopeDomain),
		string(rbacutils.ScopeProject)}) {
		return nil, httperrors.NewInputParameterError("invalid scope %s", scope)
	}
	var allowCreate bool
	switch rbacutils.TRbacScope(scope) {
	case rbacutils.ScopeSystem:
		allowCreate = IsAdminAllowCreate(userCred, man)
	case rbacutils.ScopeDomain:
		allowCreate = IsDomainAllowCreate(userCred, man)
	case rbacutils.ScopeProject:
		allowCreate = IsProjectAllowCreate(userCred, man)
	}
	if !allowCreate {
		return nil, httperrors.NewForbiddenError("not allow create %s in scope %s", man.ResourceScope(), scope)
	}
	data.Set("scope", jsonutils.NewString(scope))
	return data, nil
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

func (s *SScopedResourceBase) GetOwnerId() mcclient.IIdentityProvider {
	return &SOwnerId{
		DomainId:  s.DomainId,
		ProjectId: s.ProjectId,
	}
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
	case rbacutils.ScopeDomain:
		s.DomainId = ownerId.GetDomainId()
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

func (m *SScopedResourceBaseManager) PerformSetScope(
	ctx context.Context,
	obj IScopedResourceModel,
	userCred mcclient.TokenCredential,
	data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	domainId := jsonutils.GetAnyString(data, []string{"domain_id", "domain", "project_domain_id", "project_domain"})
	projectId := jsonutils.GetAnyString(data, []string{"project_id", "project"})
	if projectId != "" {
		project, err := TenantCacheManager.FetchTenantByIdOrName(ctx, projectId)
		if err != nil {
			return nil, err
		}
		projectId = project.GetId()
		domainId = project.GetDomainId()
	}
	if domainId != "" {
		domain, err := TenantCacheManager.FetchDomainByIdOrName(ctx, domainId)
		if err != nil {
			return nil, err
		}
		domainId = domain.GetId()
	}
	scopeToSet := getScopedResourceScope(domainId, projectId)
	var err error
	switch scopeToSet {
	case rbacutils.ScopeSystem:
		err = m.SetScopedResourceToSystem(obj, userCred)
	case rbacutils.ScopeDomain:
		err = m.SetScopedResourceToDomain(obj, userCred, domainId)
	case rbacutils.ScopeProject:
		err = m.SetScopedResourceToProject(obj, userCred, projectId)
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

func (m *SScopedResourceBaseManager) SetScopedResourceToSystem(model IScopedResourceModel, userCred mcclient.TokenCredential) error {
	if !IsAdminAllowPerform(userCred, model, "set-scope") {
		return httperrors.NewForbiddenError("Not allow set scope to system")
	}
	if model.GetProjectId() == "" && model.GetDomainId() == "" {
		return nil
	}
	return setScopedResourceIds(model, userCred, "", "")
}

func (m *SScopedResourceBaseManager) SetScopedResourceToDomain(model IScopedResourceModel, userCred mcclient.TokenCredential, domainId string) error {
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

func (m *SScopedResourceBaseManager) SetScopedResourceToProject(model IScopedResourceModel, userCred mcclient.TokenCredential, projectId string) error {
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
	return setScopedResourceIds(model, userCred, project.GetDomainId(), projectId)
}
