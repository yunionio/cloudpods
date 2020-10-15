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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SDomainizedResourceBaseManager struct {
}

type SDomainizedResourceBase struct {
	// 域Id
	DomainId string `width:"64" charset:"ascii" default:"default" nullable:"false" index:"true" list:"user" json:"domain_id"`
}

func (manager *SDomainizedResourceBaseManager) NamespaceScope() rbacutils.TRbacScope {
	if consts.IsDomainizedNamespace() {
		return rbacutils.ScopeDomain
	} else {
		return rbacutils.ScopeSystem
	}
}

func (manager *SDomainizedResourceBaseManager) ResourceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeDomain
}

func (manager *SDomainizedResourceBaseManager) FilterByOwner(q *sqlchemy.SQuery, owner mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	if owner != nil {
		switch scope {
		case rbacutils.ScopeProject, rbacutils.ScopeDomain:
			q = q.Equals("domain_id", owner.GetProjectDomainId())
		}
	}
	return q
}

func (manager *SDomainizedResourceBaseManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	return FetchDomainInfo(ctx, data)
}

func (model *SDomainizedResourceBase) GetOwnerId() mcclient.IIdentityProvider {
	owner := SOwnerId{DomainId: model.DomainId}
	return &owner
}

// returns candiate domain Id list that the resource can change owner to
// nil or empty means any domain
func (model *SDomainizedResourceBase) GetChangeOwnerCandidateDomainIds() []string {
	return nil
}

func (model *SDomainizedResourceBase) GetChangeOwnerRequiredDomainIds() []string {
	return nil
}

func ValidateCreateDomainId(domainId string) error {
	if !consts.GetNonDefaultDomainProjects() && domainId != identity.DEFAULT_DOMAIN_ID {
		return httperrors.NewForbiddenError("project in non-default domain is prohibited")
	}
	return nil
}

func (manager *SDomainizedResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	switch field {
	case "domain":
		tenantCacheQuery := TenantCacheManager.GetDomainQuery("name", "id").SubQuery()
		q = q.AppendField(tenantCacheQuery.Field("name", "domain")).Distinct()
		q = q.Join(tenantCacheQuery, sqlchemy.Equals(q.Field("domain_id"), tenantCacheQuery.Field("id")))
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SDomainizedResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.DomainizedResourceListInput,
) (*sqlchemy.SQuery, error) {
	if len(query.ProjectDomainIds) > 0 {
		// make sure ids are not utf8 string
		idList := stringutils2.RemoveUtf8Strings(query.ProjectDomainIds)
		tenants := TenantCacheManager.GetDomainQuery().SubQuery()
		subq := tenants.Query(tenants.Field("id")).Filter(sqlchemy.OR(
			sqlchemy.In(tenants.Field("id"), idList),
			sqlchemy.In(tenants.Field("name"), query.ProjectDomainIds),
		)).SubQuery()
		q = q.In("domain_id", subq)
	}
	return q, nil
}

func (manager *SDomainizedResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.DomainizedResourceListInput,
) (*sqlchemy.SQuery, error) {
	subq := TenantCacheManager.GetDomainQuery("id", "name").SubQuery()
	if NeedOrderQuery([]string{query.OrderByDomain}) {
		q = q.LeftJoin(subq, sqlchemy.Equals(q.Field("domain_id"), subq.Field("id")))
		q = OrderByFields(q, []string{query.OrderByDomain}, []sqlchemy.IQueryField{subq.Field("name")})
		return q, nil
	}
	return q, nil
}

func (manager *SDomainizedResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []apis.DomainizedResourceInfo {
	ret := make([]apis.DomainizedResourceInfo, len(objs))
	for i := range objs {
		ret[i] = apis.DomainizedResourceInfo{}
	}
	if len(fields) == 0 || fields.Contains("project_domain") {
		domainIds := stringutils2.SSortedStrings{}
		for i := range objs {
			var base *SDomainizedResourceBase
			reflectutils.FindAnonymouStructPointer(objs[i], &base)
			if base != nil && len(base.DomainId) > 0 {
				domainIds = stringutils2.Append(domainIds, base.DomainId)
			}
		}
		domains := DefaultProjectsFetcher(ctx, domainIds, true)
		if domains != nil {
			for i := range objs {
				var base *SDomainizedResourceBase
				reflectutils.FindAnonymouStructPointer(objs[i], &base)
				if base != nil && len(base.DomainId) > 0 {
					if proj, ok := domains[base.DomainId]; ok {
						if len(fields) == 0 || fields.Contains("project_domain") {
							ret[i].ProjectDomain = proj.Name
						}
					}
				}
			}
		}
	}
	return ret
}

func ValidateDomainizedResourceInput(ctx context.Context, input apis.DomainizedResourceInput) (*STenant, apis.DomainizedResourceInput, error) {
	domain, err := DefaultDomainFetcher(ctx, input.ProjectDomainId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, input, httperrors.NewResourceNotFoundError2("domain", input.ProjectDomainId)
		} else {
			return nil, input, errors.Wrap(err, "TenantCacheManager.FetchDomainByIdOrName")
		}
	}
	input.ProjectDomainId = domain.GetId()
	return domain, input, nil
}

func (manager *SDomainizedResourceBaseManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
	if keys.Contains("project_domain") {
		domainsQ := DefaultDomainQuery().SubQuery()
		q = q.LeftJoin(domainsQ, sqlchemy.Equals(q.Field("domain_id"), domainsQ.Field("id")))
		q = q.AppendField(domainsQ.Field("name", "project_domain"))
	}
	return q, nil
}
