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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/yunionconf"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-model-singular=scopedpolicy
// +onecloud:swagger-gen-model-plural=scopedpolicies
type SScopedPolicyManager struct {
	db.SInfrasResourceBaseManager
}

var ScopedPolicyManager *SScopedPolicyManager

func init() {
	ScopedPolicyManager = &SScopedPolicyManager{
		SInfrasResourceBaseManager: db.NewInfrasResourceBaseManager(
			SScopedPolicy{},
			"scopedpolicies_tbl",
			"scopedpolicy",
			"scopedpolicies",
		),
	}
	ScopedPolicyManager.SetVirtualObject(ScopedPolicyManager)
}

type SScopedPolicy struct {
	db.SInfrasResourceBase

	// 策略类别
	Category string `width:"64" charset:"utf8" nullable:"false" list:"domain" create:"domain_required" index:"true"`

	// 策略内容
	Policies jsonutils.JSONObject `charset:"utf8" nullable:"true" list:"domain" update:"domain" create:"domain_optional"`
}

func (manager *SScopedPolicyManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.ScopedPolicyCreateInput,
) (api.ScopedPolicyCreateInput, error) {
	var err error

	if len(input.Category) == 0 {
		return input, errors.Wrap(httperrors.ErrInputParameter, "empty category")
	}

	input.InfrasResourceBaseCreateInput, err = manager.SInfrasResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.InfrasResourceBaseCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SInfrasResourceBaseManager.ValidateCreateData")
	}

	return input, nil
}

func (policy *SScopedPolicy) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ScopedPolicyUpdateInput) (api.ScopedPolicyUpdateInput, error) {
	var err error
	input.InfrasResourceBaseUpdateInput, err = policy.SInfrasResourceBase.ValidateUpdateData(ctx, userCred, query, input.InfrasResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SInfrasResourceBase.ValidateUpdateData")
	}
	return input, nil
}

func (policy *SScopedPolicy) ValidateDeleteCondition(ctx context.Context) error {
	cnt, err := policy.getReferenceCount()
	if err != nil {
		return httperrors.NewInternalServerError("getReferenceCount fail %s", err)
	}
	if cnt > 0 {
		return httperrors.NewNotEmptyError("policy is referenced")
	}
	return policy.SInfrasResourceBase.ValidateDeleteCondition(ctx)
}

func (policy *SScopedPolicy) getReferenceCount() (int, error) {
	return ScopedPolicyBindingManager.getReferenceCount(policy.Id)
}

// 范围策略列表
func (manager *SScopedPolicyManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ScopedPolicyListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SInfrasResourceBaseManager.ListItemFilter(ctx, q, userCred, query.InfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SInfrasResourceBaseManager.ListItemFilter")
	}

	if len(query.Category) > 0 {
		q = q.Filter(sqlchemy.ContainsAny(q.Field("category"), query.Category))
	}

	return q, nil
}

func (manager *SScopedPolicyManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ScopedPolicyListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SInfrasResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.InfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SInfrasResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SScopedPolicyManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SInfrasResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (manager *SScopedPolicyManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ScopedPolicyDetails {
	rows := make([]api.ScopedPolicyDetails, len(objs))

	stdRows := manager.SInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.ScopedPolicyDetails{
			InfrasResourceBaseDetails: stdRows[i],
		}
		rows[i].RefCount, _ = objs[i].(*SScopedPolicy).getReferenceCount()
	}

	return rows
}

func (manager *SScopedPolicyManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SInfrasResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SInfrasResourceBaseManager.ListItemExportKeys")
	}
	return q, nil
}

func (policy *SScopedPolicy) AllowPerformBind(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.ScopedPolicyBindInput,
) bool {
	return true
}

func (policy *SScopedPolicy) PerformBind(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.ScopedPolicyBindInput,
) (jsonutils.JSONObject, error) {
	switch input.Scope {
	case rbacutils.ScopeSystem:
		err := ScopedPolicyBindingManager.bind(ctx, policy.Category, policy.Id, "", "")
		if err != nil {
			return nil, errors.Wrap(err, "bind system")
		}
	case rbacutils.ScopeDomain:
		for i := range input.TargetIds {
			if input.TargetIds[i] == api.ANY_DOMAIN_ID {
				err := ScopedPolicyBindingManager.bind(ctx, policy.Category, policy.Id, api.ANY_DOMAIN_ID, "")
				if err != nil {
					return nil, errors.Wrap(err, "bind all domain")
				}
			} else {
				domainObj, err := db.TenantCacheManager.FetchDomainByIdOrName(ctx, input.TargetIds[i])
				if err != nil {
					return nil, errors.Wrap(err, "FetchDomainByIdOrName")
				}
				err = ScopedPolicyBindingManager.bind(ctx, policy.Category, policy.Id, domainObj.GetId(), "")
				if err != nil {
					return nil, errors.Wrapf(err, "bind domain %s", domainObj.GetName())
				}
			}
		}
	case rbacutils.ScopeProject:
		for i := range input.TargetIds {
			if input.TargetIds[i] == api.ANY_PROJECT_ID {
				err := ScopedPolicyBindingManager.bind(ctx, policy.Category, policy.Id, api.ANY_DOMAIN_ID, api.ANY_PROJECT_ID)
				if err != nil {
					return nil, errors.Wrap(err, "bind all project")
				}
			} else {
				tenantObj, err := db.TenantCacheManager.FetchTenantByIdOrName(ctx, input.TargetIds[i])
				if err != nil {
					return nil, errors.Wrap(err, "FetchTenantByIdOrName")
				}
				err = ScopedPolicyBindingManager.bind(ctx, policy.Category, policy.Id, tenantObj.DomainId, tenantObj.Id)
				if err != nil {
					return nil, errors.Wrapf(err, "bind project %s", tenantObj.GetName())
				}
			}
		}
	}
	return nil, nil
}
