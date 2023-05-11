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
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/onecloud/pkg/util/tagutils"
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
	var resScope rbacscope.TRbacScope
	for i := range rows {
		if model, ok := objs[i].(ISharableBaseModel); ok {
			if len(resType) == 0 {
				resType = model.Keyword()
			}
			if len(resScope) == 0 {
				resScope = model.GetModelManager().ResourceScope()
			}
			resIds[i] = model.GetId()
		}
	}

	q := SharedResourceManager.Query()
	q = q.Equals("resource_type", resType)

	sharedResourceMap := make(map[string][]SSharedResource)
	err := FetchQueryObjectsByIds(q, "resource_id", resIds, &sharedResourceMap)
	if err != nil {
		log.Errorf("FetchQueryObjectsByIds for shared resource fail %s", err)
		return rows
	}

	targetTenantIds := stringutils2.NewSortedStrings([]string{})
	targetDomainIds := stringutils2.NewSortedStrings([]string{})

	for _, srs := range sharedResourceMap {
		for _, sr := range srs {
			switch sr.TargetType {
			case SharedTargetProject:
				targetTenantIds = stringutils2.Append(targetTenantIds, sr.TargetProjectId)
			case SharedTargetDomain:
				targetDomainIds = stringutils2.Append(targetDomainIds, sr.TargetProjectId)
			}
		}
	}

	var tenantMap map[string]STenant
	var domainMap map[string]STenant

	if len(targetTenantIds) > 0 {
		tenantMap = DefaultProjectsFetcher(ctx, targetTenantIds, false)
	}
	if len(targetDomainIds) > 0 {
		domainMap = DefaultProjectsFetcher(ctx, targetDomainIds, true)
	}

	for i := range rows {
		resId := resIds[i]
		if srs, ok := sharedResourceMap[resId]; ok {
			projects := make([]apis.SharedProject, 0)
			domains := make([]apis.SharedDomain, 0)
			for _, sr := range srs {
				switch sr.TargetType {
				case SharedTargetProject:
					project := apis.SharedProject{}
					project.Id = sr.TargetProjectId
					if tenant, ok := tenantMap[sr.TargetProjectId]; ok {
						project.Name = tenant.Name
						project.Domain = tenant.Domain
						project.DomainId = tenant.DomainId
					}
					projects = append(projects, project)
				case SharedTargetDomain:
					domain := apis.SharedDomain{}
					domain.Id = sr.TargetProjectId
					if tenant, ok := domainMap[sr.TargetProjectId]; ok {
						domain.Name = tenant.Name
					}
					domains = append(domains, domain)
				}
			}
			rows[i].SharedProjects = projects
			rows[i].SharedDomains = domains
		}
	}

	return rows
}

func SharableManagerValidateCreateData(
	manager IStandaloneModelManager,
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input apis.SharableResourceBaseCreateInput,
) (apis.SharableResourceBaseCreateInput, error) {
	resScope := manager.ResourceScope()
	reqScope := resScope
	isPublic := true
	switch resScope {
	case rbacscope.ScopeProject:
		if input.PublicScope == string(rbacscope.ScopeSystem) {
			input.IsPublic = &isPublic
			reqScope = rbacscope.ScopeSystem
		} else if input.PublicScope == string(rbacscope.ScopeDomain) {
			if consts.GetNonDefaultDomainProjects() {
				// only if non_default_domain_projects turned on, allow sharing to domain
				input.IsPublic = &isPublic
				reqScope = rbacscope.ScopeDomain
			} else {
				input.IsPublic = &isPublic
				reqScope = rbacscope.ScopeSystem
			}
		} else if input.IsPublic != nil && *input.IsPublic && len(input.PublicScope) == 0 {
			// backward compatible, if only is_public is true, make it share to system
			input.IsPublic = &isPublic
			input.PublicScope = string(rbacscope.ScopeSystem)
			reqScope = rbacscope.ScopeSystem
		} else {
			input.IsPublic = nil
			input.PublicScope = "" // string(rbacscope.ScopeNone)
		}
	case rbacscope.ScopeDomain:
		if consts.GetNonDefaultDomainProjects() {
			// only if non_default_domain_projects turned on, allow sharing domain resources
			if input.PublicScope == string(rbacscope.ScopeSystem) {
				input.IsPublic = &isPublic
				reqScope = rbacscope.ScopeSystem
			} else if input.IsPublic != nil && *input.IsPublic && len(input.PublicScope) == 0 {
				// backward compatible, if only is_public is true, make it share to system
				input.IsPublic = &isPublic
				input.PublicScope = string(rbacscope.ScopeSystem)
				reqScope = rbacscope.ScopeSystem
			} else {
				input.IsPublic = nil
				input.PublicScope = "" // string(rbacscope.ScopeNone)
			}
		} else {
			// if non_default_domain_projects turned off, all domain resources shared to system
			input.IsPublic = &isPublic
			input.PublicScope = string(rbacscope.ScopeSystem)
			reqScope = rbacscope.ScopeSystem
		}
	default:
		return input, errors.Wrap(httperrors.ErrInputParameter, "the resource is not sharable")
	}
	if input.IsPublic != nil && *input.IsPublic {
		// TODO: deal with policyTags
		allowScope, _ := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionPerform, "public")
		if reqScope.HigherThan(allowScope) {
			return input, errors.Wrapf(httperrors.ErrNotSufficientPrivilege, "require %s allow %s", reqScope, allowScope)
		}
	}
	return input, nil
}

func SharableManagerFilterByOwner(manager IStandaloneModelManager, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, owner mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	if owner != nil {
		resScope := manager.ResourceScope()
		if resScope == rbacscope.ScopeProject && scope == rbacscope.ScopeProject {
			ownerProjectId := owner.GetProjectId()
			if len(ownerProjectId) > 0 {
				subq := SharedResourceManager.Query("resource_id")
				subq = subq.Equals("resource_type", manager.Keyword())
				subq = subq.Equals("target_project_id", ownerProjectId)
				subq = subq.Equals("target_type", SharedTargetProject)
				subq2 := SharedResourceManager.Query("resource_id")
				subq2 = subq2.Equals("resource_type", manager.Keyword())
				subq2 = subq2.Equals("target_project_id", owner.GetProjectDomainId())
				subq2 = subq2.Equals("target_type", SharedTargetDomain)
				q = q.Filter(sqlchemy.OR(
					sqlchemy.Equals(q.Field("tenant_id"), ownerProjectId),
					sqlchemy.AND(
						sqlchemy.IsTrue(q.Field("is_public")),
						sqlchemy.Equals(q.Field("public_scope"), rbacscope.ScopeSystem),
					),
					sqlchemy.AND(
						sqlchemy.IsTrue(q.Field("is_public")),
						sqlchemy.Equals(q.Field("public_scope"), rbacscope.ScopeDomain),
						sqlchemy.OR(
							sqlchemy.Equals(q.Field("domain_id"), owner.GetProjectDomainId()),
							sqlchemy.In(q.Field("id"), subq2.SubQuery()),
						),
					),
					sqlchemy.In(q.Field("id"), subq.SubQuery()),
				))
				if userCred != nil {
					result := policy.PolicyManager.Allow(scope, userCred, consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionList)
					if !result.ObjectTags.IsEmpty() {
						policyTagFilters := tagutils.STagFilters{}
						policyTagFilters.AddFilters(result.ObjectTags)
						q = ObjectIdQueryWithTagFilters(q, "id", manager.Keyword(), policyTagFilters)
					}
				}
			}
		} else if (resScope == rbacscope.ScopeDomain && (scope == rbacscope.ScopeProject || scope == rbacscope.ScopeDomain)) || (resScope == rbacscope.ScopeProject && scope == rbacscope.ScopeDomain) {
			// domain view
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
						sqlchemy.Equals(q.Field("public_scope"), rbacscope.ScopeSystem),
					),
					sqlchemy.AND(
						sqlchemy.IsTrue(q.Field("is_public")),
						sqlchemy.In(q.Field("id"), subq.SubQuery()),
					),
				))
				if userCred != nil {
					result := policy.PolicyManager.Allow(scope, userCred, consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionList)
					if !result.ProjectTags.IsEmpty() && resScope == rbacscope.ScopeProject {
						policyTagFilters := tagutils.STagFilters{}
						policyTagFilters.AddFilters(result.ProjectTags)
						q = ObjectIdQueryWithTagFilters(q, "tenant_id", "project", policyTagFilters)
					}
					if !result.ObjectTags.IsEmpty() {
						policyTagFilters := tagutils.STagFilters{}
						policyTagFilters.AddFilters(result.ObjectTags)
						q = ObjectIdQueryWithTagFilters(q, "id", manager.Keyword(), policyTagFilters)
					}
				}
			}
		} else {
			log.Debugf("res_scope: %s view_scope: %s", resScope, scope)
			// system view
			if userCred != nil {
				result := policy.PolicyManager.Allow(scope, userCred, consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionList)
				log.Debugf("policy result: %s", jsonutils.Marshal(result))
				if !result.DomainTags.IsEmpty() && (resScope == rbacscope.ScopeDomain || resScope == rbacscope.ScopeProject) && scope == rbacscope.ScopeSystem {
					subq := manager.Query("id")
					policyTagFilters := tagutils.STagFilters{}
					policyTagFilters.AddFilters(result.DomainTags)
					subq = ObjectIdQueryWithTagFilters(subq, "domain_id", "domain", policyTagFilters)
					q = q.Filter(sqlchemy.OR(
						sqlchemy.In(q.Field("id"), subq.SubQuery()),
						sqlchemy.AND(
							sqlchemy.IsTrue(q.Field("is_public")),
							sqlchemy.Equals(q.Field("public_scope"), rbacscope.ScopeSystem),
						),
					))
				}
				if !result.ProjectTags.IsEmpty() && resScope == rbacscope.ScopeProject {
					policyTagFilters := tagutils.STagFilters{}
					policyTagFilters.AddFilters(result.ProjectTags)
					q = ObjectIdQueryWithTagFilters(q, "tenant_id", "project", policyTagFilters)
				}
				if !result.ObjectTags.IsEmpty() {
					policyTagFilters := tagutils.STagFilters{}
					policyTagFilters.AddFilters(result.ObjectTags)
					q = ObjectIdQueryWithTagFilters(q, "id", manager.Keyword(), policyTagFilters)
				}
			}
		}
	}
	return q
}

type SSharableBaseResource struct {
	// 是否共享
	IsPublic bool `default:"false" nullable:"false" list:"user" create:"domain_optional"`
	// 默认共享范围
	PublicScope string `width:"16" charset:"ascii" nullable:"false" default:"system" list:"user" create:"domain_optional"`
	// 共享设置的来源, local: 本地设置, cloud: 从云上同步过来
	// example: local
	PublicSrc string `width:"10" charset:"ascii" nullable:"true" list:"user" json:"public_src"`
}

type ISharableBaseModel interface {
	IStandaloneModel
	ISharableBase
}

type ISharableBase interface {
	SetShare(scoe rbacscope.TRbacScope)
	GetIsPublic() bool
	GetPublicScope() rbacscope.TRbacScope
	GetSharableTargetDomainIds() []string
	GetRequiredSharedDomainIds() []string
	GetSharedDomains() []string
}

func ISharableChangeOwnerCandidateDomainIds(model ISharableBaseModel) []string {
	var candidates []string
	if model.GetIsPublic() {
		switch model.GetPublicScope() {
		case rbacscope.ScopeSystem:
			return candidates
		case rbacscope.ScopeDomain:
			candidates = model.GetSharedDomains()
		}
	}
	ownerId := model.GetOwnerId()
	if ownerId != nil && len(ownerId.GetProjectDomainId()) > 0 {
		candidates = append(candidates, ownerId.GetProjectDomainId())
	}
	return candidates
}

func ISharableMergeChangeOwnerCandidateDomainIds(model ISharableBaseModel, candidates ...[]string) []string {
	var ret stringutils2.SSortedStrings
	for i := range candidates {
		if len(candidates[i]) > 0 {
			cand := stringutils2.NewSortedStrings(candidates[i])
			ownerId := model.GetOwnerId()
			if ownerId != nil && len(ownerId.GetProjectDomainId()) > 0 && !cand.Contains(ownerId.GetProjectDomainId()) {
				cand = stringutils2.Append(cand, ownerId.GetProjectDomainId())
			}
			if len(ret) > 0 {
				ret = stringutils2.Intersect(ret, cand)
			} else {
				ret = stringutils2.NewSortedStrings(cand)
			}
		}
	}
	return ret
}

func ISharableMergeShareRequireDomainIds(requiredIds ...[]string) []string {
	var ret stringutils2.SSortedStrings
	for i := range requiredIds {
		if len(requiredIds[i]) > 0 {
			req := stringutils2.NewSortedStrings(requiredIds[i])
			if ret == nil {
				ret = req
			} else {
				ret = stringutils2.Merge(ret, req)
			}
		} else {
			return nil
		}
	}
	return ret
}

func SharableModelIsSharable(model ISharableBaseModel, reqUsrId mcclient.IIdentityProvider) bool {
	if model.GetIsPublic() && model.GetPublicScope() == rbacscope.ScopeSystem {
		return true
	}
	ownerId := model.GetOwnerId()
	if model.GetIsPublic() && model.GetPublicScope() == rbacscope.ScopeDomain {
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
	if model.GetPublicScope() == rbacscope.ScopeProject {
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

func (m *SSharableBaseResource) SetShare(scope rbacscope.TRbacScope) {
	pub := false
	if scope != rbacscope.ScopeNone {
		pub = true
	}
	m.IsPublic = pub
	m.PublicScope = string(scope)
	m.PublicSrc = string(apis.OWNER_SOURCE_LOCAL)
}

func (m SSharableBaseResource) GetIsPublic() bool {
	return m.IsPublic
}

func (m SSharableBaseResource) GetPublicScope() rbacscope.TRbacScope {
	return rbacscope.String2Scope(m.PublicScope)
}

func SharablePerformPublic(model ISharableBaseModel, ctx context.Context, userCred mcclient.TokenCredential, input apis.PerformPublicProjectInput) error {
	var err error

	resourceScope := model.GetModelManager().ResourceScope()
	targetScope := rbacscope.String2ScopeDefault(input.Scope, rbacscope.ScopeSystem)
	if resourceScope.HigherThan(targetScope) {
		return errors.Wrapf(httperrors.ErrNotSupported, "cannot share %s resource to %s", resourceScope, targetScope)
	}

	if len(input.SharedProjectIds) > 0 && len(input.SharedDomainIds) > 0 {
		return errors.Wrap(httperrors.ErrInputParameter, "cannot set shared_projects and shared_domains at the same time")
	} else if len(input.SharedProjectIds) > 0 && targetScope != rbacscope.ScopeProject {
		targetScope = rbacscope.ScopeProject
	} else if len(input.SharedDomainIds) > 0 && targetScope != rbacscope.ScopeDomain {
		targetScope = rbacscope.ScopeDomain
	}

	shareResult := apis.PerformPublicProjectInput{}
	shareResult.Scope = string(targetScope)

	candidateIds := model.GetSharableTargetDomainIds()
	requireIds := model.GetRequiredSharedDomainIds()

	switch targetScope {
	case rbacscope.ScopeProject:
		if len(requireIds) == 0 {
			return errors.Wrap(httperrors.ErrForbidden, "require to be shared to system")
		} else if len(requireIds) > 1 {
			return errors.Wrap(httperrors.ErrForbidden, "require to be shared to other domain")
		}
		// if len(input.SharedProjects) == 0 {
		//	return errors.Wrap(httperrors.ErrEmptyRequest, "empty shared target project list")
		// }
		shareResult.SharedProjectIds, err = SharedResourceManager.shareToTarget(ctx, userCred, model, SharedTargetProject, input.SharedProjectIds, nil, nil)
		if err != nil {
			return errors.Wrap(err, "shareToTarget")
		}
		if len(shareResult.SharedProjectIds) == 0 {
			targetScope = rbacscope.ScopeNone
		}
	case rbacscope.ScopeDomain:
		if !consts.GetNonDefaultDomainProjects() {
			return errors.Wrap(httperrors.ErrForbidden, "not allow to share to domain when non_default_domain_projects turned off")
		}
		if len(requireIds) == 0 {
			return errors.Wrap(httperrors.ErrForbidden, "require to be shared to system")
		}
		_, err = SharedResourceManager.shareToTarget(ctx, userCred, model, SharedTargetProject, nil, nil, nil)
		if err != nil {
			return errors.Wrap(err, "shareToTarget clean projects")
		}
		shareResult.SharedDomainIds, err = SharedResourceManager.shareToTarget(ctx, userCred, model, SharedTargetDomain, input.SharedDomainIds, candidateIds, requireIds)
		if err != nil {
			return errors.Wrap(err, "shareToTarget add domains")
		}
		if len(shareResult.SharedDomainIds) == 0 && resourceScope == rbacscope.ScopeDomain {
			targetScope = rbacscope.ScopeNone
		}
	case rbacscope.ScopeSystem:
		if len(candidateIds) > 0 {
			return httperrors.NewForbiddenError("sharing is limited to domains %s", jsonutils.Marshal(candidateIds))
		}
		_, err = SharedResourceManager.shareToTarget(ctx, userCred, model, SharedTargetProject, nil, nil, nil)
		if err != nil {
			return errors.Wrap(err, "shareToTarget clean projects")
		}
		_, err = SharedResourceManager.shareToTarget(ctx, userCred, model, SharedTargetDomain, nil, nil, nil)
		if err != nil {
			return errors.Wrap(err, "shareToTarget clean domainss")
		}
	}

	allowScope, policyTags := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), model.KeywordPlural(), policy.PolicyActionPerform, "public")
	if targetScope.HigherThan(allowScope) {
		return errors.Wrapf(httperrors.ErrNotSufficientPrivilege, "require %s allow %s", targetScope, allowScope)
	}
	requireScope := model.GetPublicScope()
	if requireScope.HigherThan(allowScope) {
		return errors.Wrapf(httperrors.ErrNotSufficientPrivilege, "require %s allow %s", requireScope, allowScope)
	}

	err = objectConfirmPolicyTags(ctx, model, policyTags)
	if err != nil {
		return errors.Wrap(err, "objectConfirmPolicyTags")
	}

	_, err = Update(model, func() error {
		model.SetShare(targetScope)
		return nil
	})

	if err != nil {
		return errors.Wrap(err, "Update")
	}

	if targetScope != rbacscope.ScopeNone {
		OpsLog.LogEvent(model, ACT_PUBLIC, shareResult, userCred)
		logclient.AddActionLogWithContext(ctx, model, logclient.ACT_PUBLIC, shareResult, userCred, true)
	}

	model.GetIStandaloneModel().ClearSchedDescCache()
	return nil
}

func SharablePerformPrivate(model ISharableBaseModel, ctx context.Context, userCred mcclient.TokenCredential) error {
	if !model.GetIsPublic() && model.GetPublicScope() == rbacscope.ScopeNone {
		return nil
	}

	resourceScope := model.GetModelManager().ResourceScope()
	if resourceScope == rbacscope.ScopeDomain && !consts.GetNonDefaultDomainProjects() {
		return errors.Wrap(httperrors.ErrForbidden, "not allow to private domain resource")
	}

	requireIds := model.GetRequiredSharedDomainIds()
	if len(requireIds) == 0 {
		return errors.Wrap(httperrors.ErrForbidden, "require to be shared to system")
	} else if len(requireIds) > 1 {
		return errors.Wrap(httperrors.ErrForbidden, "require to be shared to other domain")
	}

	requireScope := model.GetPublicScope()
	allowScope, policyTags := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), model.GetModelManager().KeywordPlural(), policy.PolicyActionPerform, "private")
	if requireScope.HigherThan(allowScope) {
		return errors.Wrapf(httperrors.ErrNotSufficientPrivilege, "require %s allow %s", requireScope, allowScope)
	}

	err := objectConfirmPolicyTags(ctx, model, policyTags)
	if err != nil {
		return errors.Wrap(err, "objectConfirmPolicyTags")
	}

	err = SharedResourceManager.CleanModelShares(ctx, userCred, model)
	if err != nil {
		return errors.Wrap(err, "CleanModelShares")
	}

	diff, err := Update(model, func() error {
		model.SetShare(rbacscope.ScopeNone)
		return nil
	})

	if err != nil {
		return errors.Wrap(err, "Update")
	}

	OpsLog.LogEvent(model, ACT_PRIVATE, diff, userCred)
	logclient.AddActionLogWithContext(ctx, model, logclient.ACT_PRIVATE, diff, userCred, true)

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
	switch model.GetPublicScope() {
	case rbacscope.ScopeSystem:
		if model.GetIsPublic() {
			return true
		}
	case rbacscope.ScopeDomain:
		if model.GetModelManager().ResourceScope() == rbacscope.ScopeProject {
			return true
		}
	}
	return false
}

func SharableModelCustomizeCreate(model ISharableBaseModel, ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if !data.Contains("public_scope") {
		resScope := model.GetModelManager().ResourceScope()
		if resScope == rbacscope.ScopeDomain && consts.GetNonDefaultDomainProjects() {
			// only if non_default_domain_projects turned on, do the following
			isManaged := false
			if managedModel, ok := model.(IManagedResourceBase); ok {
				isManaged = managedModel.IsManaged()
			}
			if !isManaged && IsAdminAllowPerform(ctx, userCred, model, "public") && ownerId.GetProjectDomainId() == userCred.GetProjectDomainId() {
				model.SetShare(rbacscope.ScopeSystem)
				data.(*jsonutils.JSONDict).Set("public_scope", jsonutils.NewString(string(rbacscope.ScopeSystem)))
			}
		}
	}
	if !data.Contains("public_scope") {
		model.SetShare(rbacscope.ScopeNone)
	}
	return nil
}
