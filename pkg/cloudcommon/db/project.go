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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	identityapi "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/onecloud/pkg/util/tagutils"
)

type SProjectizedResourceBaseManager struct {
	SDomainizedResourceBaseManager
}

type SProjectizedResourceBase struct {
	SDomainizedResourceBase

	// 项目Id
	ProjectId string `name:"tenant_id" width:"128" charset:"ascii" nullable:"false" index:"true" list:"user" json:"tenant_id"`
}

func (model *SProjectizedResourceBase) GetOwnerId() mcclient.IIdentityProvider {
	owner := SOwnerId{DomainId: model.DomainId, ProjectId: model.ProjectId}
	return &owner
}

func (manager *SProjectizedResourceBaseManager) FilterByOwner(ctx context.Context, q *sqlchemy.SQuery, man FilterByOwnerProvider, userCred mcclient.TokenCredential, owner mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	if owner != nil {
		switch scope {
		case rbacscope.ScopeProject:
			q = q.Equals("tenant_id", owner.GetProjectId())
			if userCred != nil {
				result := policy.PolicyManager.Allow(scope, userCred, consts.GetServiceType(), man.KeywordPlural(), policy.PolicyActionList)
				if !result.ObjectTags.IsEmpty() {
					policyTagFilters := tagutils.STagFilters{}
					policyTagFilters.AddFilters(result.ObjectTags)
					q = ObjectIdQueryWithTagFilters(ctx, q, "id", man.Keyword(), policyTagFilters)
				}
			}
		case rbacscope.ScopeDomain:
			q = q.Equals("domain_id", owner.GetProjectDomainId())
			if userCred != nil {
				result := policy.PolicyManager.Allow(scope, userCred, consts.GetServiceType(), man.KeywordPlural(), policy.PolicyActionList)
				if !result.ProjectTags.IsEmpty() {
					policyTagFilters := tagutils.STagFilters{}
					policyTagFilters.AddFilters(result.ProjectTags)
					q = ObjectIdQueryWithTagFilters(ctx, q, "tenant_id", "project", policyTagFilters)
				}
				if !result.ObjectTags.IsEmpty() {
					policyTagFilters := tagutils.STagFilters{}
					policyTagFilters.AddFilters(result.ObjectTags)
					q = ObjectIdQueryWithTagFilters(ctx, q, "id", man.Keyword(), policyTagFilters)
				}
			}
		case rbacscope.ScopeSystem:
			if userCred != nil {
				result := policy.PolicyManager.Allow(scope, userCred, consts.GetServiceType(), man.KeywordPlural(), policy.PolicyActionList)
				if !result.DomainTags.IsEmpty() {
					policyTagFilters := tagutils.STagFilters{}
					policyTagFilters.AddFilters(result.DomainTags)
					q = ObjectIdQueryWithTagFilters(ctx, q, "domain_id", "domain", policyTagFilters)
				}
				if !result.ProjectTags.IsEmpty() {
					policyTagFilters := tagutils.STagFilters{}
					policyTagFilters.AddFilters(result.ProjectTags)
					q = ObjectIdQueryWithTagFilters(ctx, q, "tenant_id", "project", policyTagFilters)
				}
				if !result.ObjectTags.IsEmpty() {
					policyTagFilters := tagutils.STagFilters{}
					policyTagFilters.AddFilters(result.ObjectTags)
					q = ObjectIdQueryWithTagFilters(ctx, q, "id", man.Keyword(), policyTagFilters)
				}
			}
		}
	}
	return q
}

func (manager *SProjectizedResourceBaseManager) ResourceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeProject
}

func (manager *SProjectizedResourceBaseManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	return FetchProjectInfo(ctx, data)
}

func (manager *SProjectizedResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	switch field {
	case "tenant":
		tenantCacheQuery := TenantCacheManager.getTable() // GetTenantQuery("name", "id").Distinct().SubQuery()
		q.AppendField(tenantCacheQuery.Field("name", "tenant")).Distinct()
		q = q.Join(tenantCacheQuery, sqlchemy.Equals(q.Field("tenant_id"), tenantCacheQuery.Field("id")))
		return q, nil
	}
	q, err := manager.SDomainizedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SProjectizedResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.ProjectizedResourceListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SDomainizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.DomainizedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SDomainizedResourceBaseManager.ListItemFilter")
	}
	if len(query.ProjectIds) > 0 {
		// make sure ids are not utf8 string
		idList := stringutils2.RemoveUtf8Strings(query.ProjectIds)
		tenants := TenantCacheManager.GetTenantQuery().SubQuery()
		subq := tenants.Query(tenants.Field("id")).Filter(sqlchemy.OR(
			sqlchemy.In(tenants.Field("id"), idList),
			sqlchemy.In(tenants.Field("name"), query.ProjectIds),
		)).SubQuery()
		q = q.In("tenant_id", subq)
	}
	tagFilters := tagutils.STagFilters{}
	if len(query.ProjectOrganizations) > 0 {
		orgFilters, err := FetchOrganizationTags(ctx, query.ProjectOrganizations, identityapi.OrgTypeProject)
		if err != nil {
			return nil, errors.Wrap(err, "FetchOrganizationTags")
		}
		tagFilters.AddFilters(orgFilters)
	}
	if !query.ProjectTags.IsEmpty() {
		tagFilters.AddFilters(query.ProjectTags)
	}
	if !query.NoProjectTags.IsEmpty() {
		tagFilters.AddNoFilters(query.NoProjectTags)
	}
	q = ObjectIdQueryWithTagFilters(ctx, q, "tenant_id", "project", tagFilters)
	return q, nil
}

func (manager *SProjectizedResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.ProjectizedResourceListInput,
) (*sqlchemy.SQuery, error) {
	orders := []string{query.OrderByProject, query.OrderByDomain}
	if NeedOrderQuery(orders) {
		subq := TenantCacheManager.GetTenantQuery("id", "name", "domain").SubQuery()
		q = q.LeftJoin(subq, sqlchemy.Equals(q.Field("tenant_id"), subq.Field("id")))
		q = OrderByFields(q, orders, []sqlchemy.IQueryField{subq.Field("name"), subq.Field("domain")})
	}
	return q, nil
}

func (manager *SProjectizedResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []apis.ProjectizedResourceInfo {
	ret := make([]apis.ProjectizedResourceInfo, len(objs))
	resIds := make([]string, len(objs))
	if len(fields) == 0 || fields.Contains("project_domain") || fields.Contains("tenant") {
		projectIds := stringutils2.SSortedStrings{}
		for i := range objs {
			var base *SProjectizedResourceBase
			reflectutils.FindAnonymouStructPointer(objs[i], &base)
			if base != nil && len(base.ProjectId) > 0 {
				resIds[i] = getObjectIdstr("project", base.ProjectId)
				projectIds = stringutils2.Append(projectIds, base.ProjectId)
			}
		}
		projects := DefaultProjectsFetcher(ctx, projectIds, false)
		if projects != nil {
			for i := range objs {
				var base *SProjectizedResourceBase
				reflectutils.FindAnonymouStructPointer(objs[i], &base)
				if base != nil && len(base.ProjectId) > 0 {
					if proj, ok := projects[base.ProjectId]; ok {
						if len(fields) == 0 || fields.Contains("project_domain") {
							ret[i].ProjectDomain = proj.Domain
						}
						if len(fields) == 0 || fields.Contains("tenant") {
							ret[i].Project = proj.Name
						}
					}
				}
			}
		}
	}

	if fields == nil || fields.Contains("__meta__") {
		q := Metadata.Query("id", "key", "value")
		metaKeyValues := make(map[string][]SMetadata)
		err := FetchQueryObjectsByIds(q, "id", resIds, &metaKeyValues)
		if err != nil {
			log.Errorf("FetchQueryObjectsByIds metadata fail %s", err)
			return ret
		}

		for i := range objs {
			if metaList, ok := metaKeyValues[resIds[i]]; ok {
				ret[i].ProjectMetadata = map[string]string{}
				for _, meta := range metaList {
					ret[i].ProjectMetadata[meta.Key] = meta.Value
				}
			}
		}
	}

	domainRows := manager.SDomainizedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range ret {
		ret[i].DomainizedResourceInfo = domainRows[i]
	}
	return ret
}

func fetchProjects(ctx context.Context, projectIds []string, isDomain bool) map[string]STenant {
	deadline := time.Now().UTC().Add(-consts.GetTenantCacheExpireSeconds())
	q := TenantCacheManager.Query().In("id", projectIds).GT("last_check", deadline)
	if isDomain {
		q = q.Equals("domain_id", identityapi.KeystoneDomainRoot)
	} else {
		q = q.NotEquals("domain_id", identityapi.KeystoneDomainRoot)
	}
	projects := make([]STenant, 0)
	err := FetchModelObjects(TenantCacheManager, q, &projects)
	if err != nil {
		return nil
	}
	ret := make(map[string]STenant)
	for i := range projects {
		ret[projects[i].Id] = projects[i]
	}
	for _, pid := range projectIds {
		if len(pid) == 0 {
			continue
		}
		if _, ok := ret[pid]; !ok {
			// not found
			var t *STenant
			if isDomain {
				t, _ = TenantCacheManager.fetchDomainFromKeystone(ctx, pid)
			} else {
				t, _ = TenantCacheManager.fetchTenantFromKeystone(ctx, pid, "")
			}
			if t != nil {
				ret[t.Id] = *t
			}
		}
	}
	return ret
}

func ValidateProjectizedResourceInput(ctx context.Context, input apis.ProjectizedResourceCreateInput) (*STenant, apis.ProjectizedResourceInput, error) {
	tenant, err := DefaultProjectFetcher(ctx, input.ProjectId, input.ProjectDomainId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, input.ProjectizedResourceInput, httperrors.NewResourceNotFoundError2("project", input.ProjectId)
		} else {
			return nil, input.ProjectizedResourceInput, errors.Wrap(err, "TenantCacheManager.FetchTenantByIdOrName")
		}
	}
	input.ProjectId = tenant.GetId()
	return tenant, input.ProjectizedResourceInput, nil
}

func (manager *SProjectizedResourceBaseManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
	q, err := manager.SDomainizedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SDomainizedResourceBaseManager.ListItemExportKeys")
	}
	if keys.Contains("tenant") {
		projectsQ := DefaultProjectQuery().SubQuery()
		q = q.LeftJoin(projectsQ, sqlchemy.Equals(q.Field("tenant_id"), projectsQ.Field("id")))
		q = q.AppendField(projectsQ.Field("name", "tenant"))
	}
	return q, nil
}
