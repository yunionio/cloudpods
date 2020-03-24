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
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SSharableBaseResourceManager struct{}

func (manager *SSharableBaseResourceManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.SharableResourceBaseListInput,
) (*sqlchemy.SQuery, error) {
	if query.IsPublic != nil {
		if *query.IsPublic == true {
			q = q.IsTrue("is_public")
		} else {
			q = q.IsFalse("is_public")
		}
	}
	if len(query.PublicScope) > 0 {
		q = q.Equals("public_scope", query.PublicScope)
	}
	return q, nil
}

func (manager *SSharableBaseResourceManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []apis.SharableResourceBaseInfo {
	rows := make([]apis.SharableResourceBaseInfo, len(objs))

	var resType string
	resIds := make([]string, len(rows))
	var sharedTarget string
	for i := range rows {
		if model, ok := objs[i].(ISharableBaseModel); ok {
			if len(resType) == 0 {
				resType = model.Keyword()
			}
			if len(sharedTarget) == 0 {
				sharedTarget = string(model.GetModelManager().ResourceScope())
			}
			resIds[i] = model.GetId()
		}
	}

	q := SharedResourceManager.Query()
	q = q.Equals("resource_type", resType)
	q = q.Equals("target_type", sharedTarget)

	sharedResourceMap := make(map[string][]SSharedResource)
	err := FetchQueryObjectsByIds(q, "resource_id", resIds, &sharedResourceMap)
	if err != nil {
		log.Errorf("FetchQueryObjectsByIds for shared resource fail %s", err)
		return rows
	}

	targetIds := stringutils2.NewSortedStrings([]string{})

	for _, srs := range sharedResourceMap {
		for _, sr := range srs {
			targetIds = stringutils2.Append(targetIds, sr.TargetProjectId)
		}
	}

	if len(targetIds) == 0 {
		return rows
	}

	tenantMap := make(map[string]STenant)
	var subq *sqlchemy.SQuery
	switch sharedTarget {
	case SharedTargetProject:
		subq = TenantCacheManager.GetTenantQuery()
	case SharedTargetDomain:
		subq = TenantCacheManager.GetDomainQuery()
	}

	err = FetchQueryObjectsByIds(subq, "id", targetIds, &tenantMap)
	if err != nil {
		log.Errorf("FetchQueryObjectsByIds for tenant_cache fail %s", err)
		return rows
	}

	for i := range rows {
		resId := resIds[i]
		if srs, ok := sharedResourceMap[resId]; ok {
			switch sharedTarget {
			case SharedTargetProject:
				projects := make([]apis.SharedProject, len(srs))
				for si, sr := range srs {
					projects[si].Id = sr.TargetProjectId
					if tenant, ok := tenantMap[sr.TargetProjectId]; ok {
						projects[si].Name = tenant.Name
						projects[si].Domain = tenant.Domain
						projects[si].DomainId = tenant.DomainId
					}
				}
				rows[i].SharedProjects = projects
			case SharedTargetDomain:
				domains := make([]apis.SharedDomain, len(srs))
				for si, sr := range srs {
					domains[si].Id = sr.TargetProjectId
					if tenant, ok := tenantMap[sr.TargetProjectId]; ok {
						domains[si].Name = tenant.Name
					}
				}
				rows[i].SharedDomains = domains
			}
		}
	}

	return rows
}

func SharableManagerFilterByOwner(manager IStandaloneModelManager, q *sqlchemy.SQuery, owner mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	if owner != nil {
		resScope := manager.ResourceScope()
		if resScope == rbacutils.ScopeProject && scope == rbacutils.ScopeProject {
			ownerProjectId := owner.GetProjectId()
			if len(ownerProjectId) > 0 {
				subq := SharedResourceManager.Query("resource_id")
				subq = subq.Equals("resource_type", manager.Keyword())
				subq = subq.Equals("target_project_id", ownerProjectId)
				subq = subq.Equals("target_type", SharedTargetProject)
				q = q.Filter(sqlchemy.OR(
					sqlchemy.Equals(q.Field("tenant_id"), ownerProjectId),
					sqlchemy.AND(
						sqlchemy.IsTrue(q.Field("is_public")),
						sqlchemy.Equals(q.Field("public_scope"), rbacutils.ScopeSystem),
					),
					sqlchemy.AND(
						sqlchemy.IsTrue(q.Field("is_public")),
						sqlchemy.Equals(q.Field("public_scope"), rbacutils.ScopeDomain),
						sqlchemy.Equals(q.Field("domain_id"), owner.GetProjectDomainId()),
					),
					sqlchemy.In(q.Field("id"), subq.SubQuery()),
				))
			}
		} else if (resScope == rbacutils.ScopeDomain && (scope == rbacutils.ScopeProject || scope == rbacutils.ScopeDomain)) || (resScope == rbacutils.ScopeProject && scope == rbacutils.ScopeDomain) {
			ownerDomainId := owner.GetProjectDomainId()
			if len(ownerDomainId) > 0 {
				subq := SharedResourceManager.Query("resource_id")
				subq = subq.Equals("resource_type", manager.Keyword())
				subq = subq.Equals("target_project_id", ownerDomainId)
				subq = subq.Equals("target_type", SharedTargetDomain)
				q = q.Filter(sqlchemy.OR(
					sqlchemy.Equals(q.Field("domain_id"), ownerDomainId),
					sqlchemy.AND(
						sqlchemy.IsTrue(q.Field("is_public")),
						sqlchemy.Equals(q.Field("public_scope"), rbacutils.ScopeSystem),
					),
					sqlchemy.AND(
						sqlchemy.IsTrue(q.Field("is_public")),
						sqlchemy.In(q.Field("id"), subq.SubQuery()),
					),
				))
			}
		}
	}
	return q
}

type SSharableBaseResource struct {
	// 是否共享
	IsPublic bool `default:"false" nullable:"false" list:"user"`
	// 默认共享范围
	PublicScope string `width:"16" charset:"ascii" nullable:"false" default:"system" list:"user"`
}

type ISharableBaseModel interface {
	IStandaloneModel
	ISharableBase
}

type ISharableBase interface {
	SetShare(pub bool, scoe rbacutils.TRbacScope)
	GetIsPublic() bool
	GetPublicScope() rbacutils.TRbacScope
}

func SharableModelIsSharable(model ISharableBaseModel, reqUsrId mcclient.IIdentityProvider) bool {
	if model.GetIsPublic() && model.GetPublicScope() == rbacutils.ScopeSystem {
		return true
	}
	ownerId := model.GetOwnerId()
	if model.GetIsPublic() && model.GetPublicScope() == rbacutils.ScopeDomain {
		if ownerId != nil && ownerId.GetProjectDomainId() == reqUsrId.GetProjectDomainId() {
			return true
		}
		q := SharedResourceManager.Query().Equals("resource_id", model.GetId())
		q = q.Equals("resource_type", model.Keyword())
		q = q.Equals("target_project_id", reqUsrId.GetProjectDomainId())
		q = q.Equals("target_type", SharedTargetDomain)
		cnt, _ := q.CountWithError()
		if cnt > 0 {
			return true
		}
	}
	if model.GetPublicScope() == rbacutils.ScopeProject {
		if ownerId != nil && ownerId.GetProjectId() == reqUsrId.GetProjectId() {
			return true
		}
		q := SharedResourceManager.Query().Equals("resource_id", model.GetId())
		q = q.Equals("resource_type", model.Keyword())
		q = q.Equals("target_project_id", reqUsrId.GetProjectId())
		q = q.Equals("target_type", SharedTargetProject)
		cnt, _ := q.CountWithError()
		if cnt > 0 {
			return true
		}
	}
	return false
}

func (m *SSharableBaseResource) SetShare(pub bool, scope rbacutils.TRbacScope) {
	m.IsPublic = pub
	m.PublicScope = string(scope)
}

func (m SSharableBaseResource) GetIsPublic() bool {
	return m.IsPublic
}

func (m SSharableBaseResource) GetPublicScope() rbacutils.TRbacScope {
	return rbacutils.String2Scope(m.PublicScope)
}

func SharablePerformPublic(model ISharableBaseModel, ctx context.Context, userCred mcclient.TokenCredential, input apis.PerformPublicInput) error {
	var err error

	resourceScope := model.GetModelManager().ResourceScope()
	targetScope := rbacutils.String2ScopeDefault(input.Scope, rbacutils.ScopeSystem)
	if resourceScope.HigherThan(targetScope) {
		return errors.Wrapf(httperrors.ErrNotSupported, "cannot share %s resource to %s", resourceScope, targetScope)
	}

	shareResult := apis.PerformPublicInput{
		Scope: string(targetScope),
	}

	switch targetScope {
	case rbacutils.ScopeProject:
		if len(input.SharedProjects) == 0 {
			return errors.Wrap(httperrors.ErrEmptyRequest, "empty shared target project list")
		}
		shareResult.SharedProjects, err = SharedResourceManager.shareToTarget(ctx, userCred, model, SharedTargetProject, input.SharedProjects)
		if err != nil {
			return errors.Wrap(err, "shareToTarget")
		}
	case rbacutils.ScopeDomain:
		_, err = SharedResourceManager.shareToTarget(ctx, userCred, model, SharedTargetProject, nil)
		if err != nil {
			return errors.Wrap(err, "shareToTarget clean projects")
		}
		shareResult.SharedDomains, err = SharedResourceManager.shareToTarget(ctx, userCred, model, SharedTargetDomain, input.SharedDomains)
		if err != nil {
			return errors.Wrap(err, "shareToTarget add domains")
		}
	case rbacutils.ScopeSystem:
		_, err = SharedResourceManager.shareToTarget(ctx, userCred, model, SharedTargetProject, nil)
		if err != nil {
			return errors.Wrap(err, "shareToTarget clean projects")
		}
		_, err = SharedResourceManager.shareToTarget(ctx, userCred, model, SharedTargetDomain, nil)
		if err != nil {
			return errors.Wrap(err, "shareToTarget clean domainss")
		}
	}

	allowScope := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), model.KeywordPlural(), policy.PolicyActionPerform, "public")
	if targetScope.HigherThan(allowScope) {
		return errors.Wrapf(httperrors.ErrNotSufficientPrivilege, "require %s allow %s", targetScope, allowScope)
	}

	_, err = Update(model, func() error {
		model.SetShare(true, targetScope)
		return nil
	})

	if err != nil {
		return errors.Wrap(err, "Update")
	}

	OpsLog.LogEvent(model, ACT_PUBLIC, shareResult, userCred)

	model.GetIStandaloneModel().ClearSchedDescCache()
	return nil
}

func SharablePerformPrivate(model ISharableBaseModel, ctx context.Context, userCred mcclient.TokenCredential) error {
	if !model.GetIsPublic() {
		return nil
	}

	requireScope := model.GetPublicScope()
	allowScope := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), model.GetModelManager().KeywordPlural(), policy.PolicyActionPerform, "private")
	if requireScope.HigherThan(allowScope) {
		return errors.Wrapf(httperrors.ErrNotSufficientPrivilege, "require %s allow %s", requireScope, allowScope)
	}

	err := SharedResourceManager.CleanModelShares(ctx, userCred, model)
	if err != nil {
		return errors.Wrap(err, "CleanModelShares")
	}

	diff, err := Update(model, func() error {
		model.SetShare(false, rbacutils.ScopeNone)
		return nil
	})

	if err != nil {
		return errors.Wrap(err, "Update")
	}

	OpsLog.LogEvent(model, ACT_PRIVATE, diff, userCred)

	model.GetIStandaloneModel().ClearSchedDescCache()

	return nil
}

func SharableGetSharedProjects(model ISharableBaseModel, targetType string) []string {
	sharedResources := make([]SSharedResource, 0)
	q := SharedResourceManager.Query()
	q = q.Equals("resource_type", model.Keyword())
	q = q.Equals("resource_id", model.GetId())
	q = q.Equals("target_type", targetType)
	err := q.All(&sharedResources)
	if err != nil {
		return nil
	}
	res := make([]string, len(sharedResources))
	for i := range sharedResources {
		res[i] = sharedResources[i].TargetProjectId
	}
	return res
}

func SharableModelIsShared(model ISharableBaseModel) bool {
	q := SharedResourceManager.Query()
	q = q.Equals("resource_type", model.Keyword())
	q = q.Equals("resource_id", model.GetId())
	cnt, _ := q.CountWithError()
	if cnt > 0 {
		return true
	}
	return false
}
