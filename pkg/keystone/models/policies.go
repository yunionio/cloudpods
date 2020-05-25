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
	"database/sql"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	policyman "yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SPolicyManager struct {
	SEnabledIdentityBaseResourceManager
	db.SSharableBaseResourceManager
}

var PolicyManager *SPolicyManager

func init() {
	PolicyManager = &SPolicyManager{
		SEnabledIdentityBaseResourceManager: NewEnabledIdentityBaseResourceManager(
			SPolicy{},
			"policy",
			"policy",
			"policies",
		),
	}
	PolicyManager.SetVirtualObject(PolicyManager)
}

/*
+-------+--------------+------+-----+---------+-------+
| Field | Type         | Null | Key | Default | Extra |
+-------+--------------+------+-----+---------+-------+
| id    | varchar(64)  | NO   | PRI | NULL    |       |
| type  | varchar(255) | NO   |     | NULL    |       |
| blob  | text         | NO   |     | NULL    |       |
| extra | text         | YES  |     | NULL    |       |
+-------+--------------+------+-----+---------+-------+
*/

type SPolicy struct {
	SEnabledIdentityBaseResource
	db.SSharableBaseResource `"is_public=>create":"domain_optional" "public_scope=>create":"domain_optional"`

	Type string               `width:"255" charset:"utf8" nullable:"false" list:"user" create:"domain_required" update:"domain"`
	Blob jsonutils.JSONObject `nullable:"false" list:"user" create:"domain_required" update:"domain"`
}

func (manager *SPolicyManager) InitializeData() error {
	q := manager.Query()
	q = q.IsNullOrEmpty("name")
	policies := make([]SPolicy, 0)
	err := db.FetchModelObjects(manager, q, &policies)
	if err != nil {
		return err
	}
	for i := range policies {
		db.Update(&policies[i], func() error {
			policies[i].Name = policies[i].Type
			policies[i].Description, _ = policies[i].Extra.GetString("description")
			return nil
		})
	}
	return nil
}

func (manager *SPolicyManager) FetchEnabledPolicies() ([]SPolicy, error) {
	q := manager.Query().IsTrue("enabled")

	policies := make([]SPolicy, 0)
	err := db.FetchModelObjects(manager, q, &policies)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return policies, nil
}

func validatePolicyVioldatePrivilege(userCred mcclient.TokenCredential, policy *rbacutils.SRbacPolicy) error {
	if userCred.GetUserName() == api.SystemAdminUser && userCred.GetDomainId() == api.DEFAULT_DOMAIN_ID {
		return nil
	}
	opsScope, opsPolicySet := policyman.PolicyManager.GetMatchedPolicySet(userCred)
	if opsScope != rbacutils.ScopeSystem && policy.Scope.HigherThan(opsScope) {
		return errors.Wrapf(httperrors.ErrNotSufficientPrivilege, "cannot create policy scope higher than %s", opsScope)
	}
	assignPolicySet := rbacutils.TPolicySet{policy}
	if !opsScope.HigherThan(policy.Scope) && opsPolicySet.ViolatedBy(assignPolicySet) {
		return errors.Wrap(httperrors.ErrNotSufficientPrivilege, "policy violates operator's policy")
	}
	return nil
}

func (manager *SPolicyManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.PolicyCreateInput,
) (api.PolicyCreateInput, error) {
	var err error
	if len(input.Type) == 0 {
		return input, httperrors.NewInputParameterError("missing input field type")
	}
	input.Name = input.Type
	policy := rbacutils.SRbacPolicy{}
	err = policy.Decode(input.Blob)
	if err != nil {
		return input, httperrors.NewInputParameterError("fail to decode policy data")
	}

	err = validatePolicyVioldatePrivilege(userCred, &policy)
	if err != nil {
		return input, errors.Wrap(err, "validatePolicyVioldatePrivilege")
	}

	err = db.ValidateCreateDomainId(ownerId.GetProjectDomainId())
	if err != nil {
		return input, errors.Wrap(err, "ValidateCreateDomainId")
	}
	input.EnabledIdentityBaseResourceCreateInput, err = manager.SEnabledIdentityBaseResourceManager.ValidateCreateData(ctx, userCred, ownerId, query, input.EnabledIdentityBaseResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SEnabledIdentityBaseResourceManager.ValidateCreateData")
	}
	input.SharableResourceBaseCreateInput, err = db.SharableManagerValidateCreateData(manager, ctx, userCred, ownerId, query, input.SharableResourceBaseCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SharableManagerValidateCreateData")
	}

	quota := &SIdentityQuota{
		SBaseDomainQuotaKeys: quotas.SBaseDomainQuotaKeys{DomainId: ownerId.GetProjectDomainId()},
		Policy:               1,
	}
	err = quotas.CheckSetPendingQuota(ctx, userCred, quota)
	if err != nil {
		return input, errors.Wrap(err, "CheckSetPendingQuota")
	}

	return input, nil
}

func (policy *SPolicy) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.PolicyUpdateInput) (api.PolicyUpdateInput, error) {
	if input.Blob != nil {
		p := rbacutils.SRbacPolicy{}
		err := p.Decode(input.Blob)
		if err != nil {
			return input, httperrors.NewInputParameterError("fail to decode policy data")
		}
		/* if p.IsSystemWidePolicy() && policyman.PolicyManager.Allow(rbacutils.ScopeSystem, userCred, consts.GetServiceType(), policy.GetModelManager().KeywordPlural(), policyman.PolicyActionUpdate) == rbacutils.Deny {
			return nil, httperrors.NewNotSufficientPrivilegeError("not allow to update system-wide policy")
		} */
		err = validatePolicyVioldatePrivilege(userCred, &p)
		if err != nil {
			return input, errors.Wrap(err, "validatePolicyVioldatePrivilege")
		}
	}
	if len(input.Type) > 0 {
		input.Name = input.Type
	}
	var err error
	input.EnabledIdentityBaseUpdateInput, err = policy.SEnabledIdentityBaseResource.ValidateUpdateData(ctx, userCred, query, input.EnabledIdentityBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SEnabledIdentityBaseResource.ValidateUpdateData")
	}
	return input, nil
}

func (policy *SPolicy) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	policy.SEnabledIdentityBaseResource.PostCreate(ctx, userCred, ownerId, query, data)

	quota := &SIdentityQuota{
		SBaseDomainQuotaKeys: quotas.SBaseDomainQuotaKeys{DomainId: ownerId.GetProjectDomainId()},
		Policy:               1,
	}
	err := quotas.CancelPendingUsage(ctx, userCred, quota, quota, true)
	if err != nil {
		log.Errorf("CancelPendingUsage fail %s", err)
	}

	policyman.PolicyManager.SyncOnce()
}

func (policy *SPolicy) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	policy.SEnabledIdentityBaseResource.PostUpdate(ctx, userCred, query, data)
	policyman.PolicyManager.SyncOnce()
}

func (policy *SPolicy) PostDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	policy.SEnabledIdentityBaseResource.PostDelete(ctx, userCred)
	policyman.PolicyManager.SyncOnce()
}

func (policy *SPolicy) IsSharable(reqUsrId mcclient.IIdentityProvider) bool {
	return db.SharableModelIsSharable(policy, reqUsrId)
}

func (policy *SPolicy) IsShared() bool {
	return db.SharableModelIsShared(policy)
}

func (policy *SPolicy) AllowPerformPublic(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformPublicDomainInput) bool {
	return true
}

// 共享Policy
func (policy *SPolicy) PerformPublic(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformPublicDomainInput) (jsonutils.JSONObject, error) {
	err := db.SharablePerformPublic(policy, ctx, userCred, apis.PerformPublicProjectInput{PerformPublicDomainInput: input})
	if err != nil {
		return nil, errors.Wrap(err, "SharablePerformPublic")
	}
	policyman.PolicyManager.SyncOnce()
	return nil, nil
}

func (policy *SPolicy) AllowPerformPrivate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformPrivateInput) bool {
	return true
}

// 设置policy为私有
func (policy *SPolicy) PerformPrivate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformPrivateInput) (jsonutils.JSONObject, error) {
	err := db.SharablePerformPrivate(policy, ctx, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "SharablePerformPrivate")
	}
	policyman.PolicyManager.SyncOnce()
	return nil, nil
}

func (policy *SPolicy) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	db.SharableModelCustomizeCreate(policy, ctx, userCred, ownerId, query, data)
	return policy.SEnabledIdentityBaseResource.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (policy *SPolicy) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	db.SharedResourceManager.CleanModelShares(ctx, userCred, policy)
	return policy.SEnabledIdentityBaseResource.Delete(ctx, userCred)
}

func (policy *SPolicy) ValidateDeleteCondition(ctx context.Context) error {
	if policy.IsShared() {
		return httperrors.NewInvalidStatusError("cannot delete shared policy")
	}
	if policy.Enabled.IsTrue() {
		return httperrors.NewInvalidStatusError("cannot delete enabled policy")
	}
	return policy.SEnabledIdentityBaseResource.ValidateDeleteCondition(ctx)
}

// 权限策略列表
func (manager *SPolicyManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.PolicyListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SEnabledIdentityBaseResourceManager.ListItemFilter(ctx, q, userCred, query.EnabledIdentityBaseResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledIdentityBaseResourceManager.ListItemFilter")
	}
	q, err = manager.SSharableBaseResourceManager.ListItemFilter(ctx, q, userCred, query.SharableResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableBaseResourceManager.ListItemFilter")
	}
	if len(query.Type) > 0 {
		q = q.In("type", query.Type)
	}
	return q, nil
}

func (manager *SPolicyManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.PolicyListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SEnabledIdentityBaseResourceManager.OrderByExtraFields(ctx, q, userCred, query.EnabledIdentityBaseResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledIdentityBaseResourceManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SPolicyManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SEnabledIdentityBaseResourceManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (policy *SPolicy) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.PolicyDetails, error) {
	return api.PolicyDetails{}, nil
}

func (manager *SPolicyManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.PolicyDetails {
	rows := make([]api.PolicyDetails, len(objs))
	identRows := manager.SEnabledIdentityBaseResourceManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	shareRows := manager.SSharableBaseResourceManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.PolicyDetails{
			EnabledIdentityBaseResourceDetails: identRows[i],
			SharableResourceBaseInfo:           shareRows[i],
		}
	}
	return rows
}

func (policy *SPolicy) GetUsages() []db.IUsage {
	if policy.Deleted {
		return nil
	}
	usage := SIdentityQuota{Policy: 1}
	usage.SetKeys(quotas.SBaseDomainQuotaKeys{DomainId: policy.DomainId})
	return []db.IUsage{
		&usage,
	}
}

func (manager *SPolicyManager) FilterByOwner(q *sqlchemy.SQuery, owner mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	return db.SharableManagerFilterByOwner(manager, q, owner, scope)
}

func (policy *SPolicy) GetSharableTargetDomainIds() []string {
	return nil
}

func (policy *SPolicy) GetRequiredSharedDomainIds() []string {
	return []string{policy.DomainId}
}

func (policy *SPolicy) GetSharedDomains() []string {
	return db.SharableGetSharedProjects(policy, db.SharedTargetDomain)
}
