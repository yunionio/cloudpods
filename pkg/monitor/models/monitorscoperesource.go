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

	"yunion.io/x/jsonutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SMonitorScopedResourceManager struct {
	db.SScopedResourceBaseManager
}

type SMonitorScopedResource struct {
	db.SScopedResourceBase
}

func (m *SMonitorScopedResourceManager) FilterByOwner(q *sqlchemy.SQuery, userCred mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
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

func (s *SMonitorScopedResource) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	scope, _ := data.GetString("scope")
	switch rbacutils.TRbacScope(scope) {
	case rbacutils.ScopeSystem:
		s.DomainId = ""
		s.ProjectId = ""
	case rbacutils.ScopeDomain:
		s.DomainId = ownerId.GetProjectDomainId()
		s.ProjectId = ""
	case rbacutils.ScopeProject:
		s.DomainId = ownerId.GetProjectDomainId()
		s.ProjectId = ownerId.GetProjectId()
	}
	return nil
}

func (manager *SMonitorScopedResourceManager) ResourceScope() rbacutils.TRbacScope {
	return manager.SScopedResourceBaseManager.ResourceScope()
}

func (manager *SMonitorScopedResourceManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	return manager.SScopedResourceBaseManager.FetchOwnerId(ctx, data)
}

func (model *SMonitorScopedResource) GetOwnerId() mcclient.IIdentityProvider {
	return model.SScopedResourceBase.GetOwnerId()
}
