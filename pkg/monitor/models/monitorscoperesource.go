package models

import (
	"context"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/sqlchemy"
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

func (manager *SMonitorScopedResourceManager) ResourceScope() rbacutils.TRbacScope {
	return manager.SScopedResourceBaseManager.ResourceScope()
}

func (manager *SMonitorScopedResourceManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	return manager.SScopedResourceBaseManager.FetchOwnerId(ctx, data)
}

func (model *SMonitorScopedResource) GetOwnerId() mcclient.IIdentityProvider {
	return model.SScopedResourceBase.GetOwnerId()
}
