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
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	policyman "yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/locale"
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

	// swagger:ignore
	// Deprecated
	Type string `width:"255" charset:"utf8" nullable:"false" list:"user" create:"domain_required" update:"domain"`

	// 权限定义
	Blob jsonutils.JSONObject `nullable:"false" list:"user" create:"domain_required" update:"domain"`

	// 权限范围
	Scope rbacutils.TRbacScope `nullable:"true" list:"user" create:"domain_required" update:"domain"`

	// 是否为系统权限
	IsSystem tristate.TriState `nullable:"false" default:"false" list:"domain" update:"admin" create:"admin_optional"`
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
		if !gotypes.IsNil(policies[i].Extra) {
			continue
		}
		db.Update(&policies[i], func() error {
			policies[i].Name = policies[i].Type
			policies[i].Description, _ = policies[i].Extra.GetString("description")
			return nil
		})
	}

	err = manager.initializeRolePolicyGroup()
	if err != nil {
		return err
	}

	return nil
}

func (manager *SPolicyManager) initializeRolePolicyGroup() error {
	ctx := context.Background()
	q := manager.Query()
	q = q.IsNullOrEmpty("scope")
	policies := make([]SPolicy, 0)
	err := db.FetchModelObjects(manager, q, &policies)
	if err != nil {
		return err
	}

	for i := range policies {
		var policy rbacutils.SRbacPolicy
		err := policy.Decode(policies[i].Blob)
		if err != nil {
			log.Errorf("Decode policy %s failed %s", policies[i].Name, err)
			continue
		}
		failed := false
		if len(policy.Roles) == 0 && len(policy.Projects) == 0 {
			// match any
			roles, err := policies[i].fetchMatchableRoles()
			if err != nil {
				log.Errorf("policy fetchMatchableRoles fail %s", err)
				failed = true
			} else {
				for _, r := range roles {
					err = RolePolicyManager.newRecord(ctx, r.Id, "", policies[i].Id, tristate.NewFromBool(policy.Auth), policy.Ips)
					if err != nil {
						log.Errorf("insert role policy fail %s", err)
						failed = true
					}
				}
			}
		} else if len(policy.Roles) > 0 && len(policy.Projects) == 0 {
			for _, r := range policy.Roles {
				role, err := RoleManager.FetchRoleByName(r, policies[i].DomainId, "")
				if err != nil {
					log.Errorf("fetch role %s fail %s", r, err)
					continue
				}
				err = RolePolicyManager.newRecord(ctx, role.Id, "", policies[i].Id, tristate.True, policy.Ips)
				if err != nil {
					log.Errorf("insert role policy fail %s", err)
					failed = true
				}
			}
		} else if len(policy.Roles) == 0 && len(policy.Projects) > 0 {
			for _, p := range policy.Projects {
				project, err := ProjectManager.FetchProjectByName(p, policies[i].DomainId, "")
				if err != nil {
					log.Errorf("fetch porject %s fail %s", p, err)
					continue
				}
				roles, err := policies[i].fetchMatchableRoles()
				if err != nil {
					log.Errorf("policy fetchMatchableRoles fail %s", err)
					failed = true
				} else {
					for _, r := range roles {
						err = RolePolicyManager.newRecord(ctx, r.Id, project.Id, policies[i].Id, tristate.True, policy.Ips)
						if err != nil {
							log.Errorf("insert role policy fail %s", err)
							failed = true
						}
					}
				}
			}
		} else if len(policy.Roles) > 0 && len(policy.Projects) > 0 {
			for _, r := range policy.Roles {
				role, err := RoleManager.FetchRoleByName(r, policies[i].DomainId, "")
				if err != nil {
					log.Errorf("fetch role %s fail %s", r, err)
					continue
				}
				for _, p := range policy.Projects {
					project, err := ProjectManager.FetchProjectByName(p, policies[i].DomainId, "")
					if err != nil {
						log.Errorf("fetch project %s fail %s", p, err)
						continue
					}
					err = RolePolicyManager.newRecord(ctx, role.Id, project.Id, policies[i].Id, tristate.True, policy.Ips)
					if err != nil {
						log.Errorf("insert role policy fail %s", err)
						failed = true
					}
				}
			}
		}
		if !failed {
			db.Update(&policies[i], func() error {
				policies[i].Scope = policy.Scope
				// do not rewrite blob, make backward compatible
				// policies[i].Blob = policy.Rules.Encode()
				return nil
			})
		}
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

func validatePolicyVioldatePrivilege(userCred mcclient.TokenCredential, policyScope rbacutils.TRbacScope, policy rbacutils.TPolicy) error {
	if userCred.GetUserName() == api.SystemAdminUser && userCred.GetDomainId() == api.DEFAULT_DOMAIN_ID {
		return nil
	}
	_, policyGroup, err := RolePolicyManager.GetMatchPolicyGroup(userCred, false)
	if err != nil {
		return errors.Wrap(err, "GetMatchPolicyGroup")
	}
	noViolate := false
	assignPolicySet := rbacutils.TPolicySet{policy}
	for _, scope := range []rbacutils.TRbacScope{
		rbacutils.ScopeSystem,
		rbacutils.ScopeDomain,
		rbacutils.ScopeProject,
	} {
		isViolate := false
		policySet, ok := policyGroup[scope]
		if !ok || len(policySet) == 0 || policySet.ViolatedBy(assignPolicySet) {
			isViolate = true
		}
		if !isViolate {
			noViolate = true
		}
		if scope == policyScope {
			break
		}
	}
	if !noViolate {
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
	if len(input.Type) == 0 && len(input.Name) == 0 {
		return input, httperrors.NewInputParameterError("missing input field type")
	}
	if len(input.Name) == 0 {
		input.Name = input.Type
	}

	policy, err := rbacutils.DecodePolicyData(input.Blob)
	if err != nil {
		return input, httperrors.NewInputParameterError("fail to decode policy data")
	}

	input.Scope = rbacutils.String2ScopeDefault(string(input.Scope), rbacutils.ScopeProject)

	err = validatePolicyVioldatePrivilege(userCred, input.Scope, policy)
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

	requireScope := input.Scope
	if input.IsSystem != nil && *input.IsSystem {
		requireScope = rbacutils.ScopeSystem
	}
	allowScope := policyman.PolicyManager.AllowScope(userCred, api.SERVICE_TYPE, manager.KeywordPlural(), policyman.PolicyActionCreate)
	if requireScope.HigherThan(allowScope) {
		return input, errors.Wrapf(httperrors.ErrNotSufficientPrivilege, "require %s allow %s", requireScope, allowScope)
	}

	return input, nil
}

func (policy *SPolicy) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.PolicyUpdateInput) (api.PolicyUpdateInput, error) {
	var requireScope rbacutils.TRbacScope

	if len(input.Scope) > 0 {
		input.Scope = rbacutils.String2ScopeDefault(string(input.Scope), rbacutils.ScopeProject)
		requireScope = input.Scope
	}
	if input.IsSystem != nil && *input.IsSystem {
		requireScope = rbacutils.ScopeSystem
	}

	if len(requireScope) > 0 {
		allowScope := policyman.PolicyManager.AllowScope(userCred, api.SERVICE_TYPE, policy.KeywordPlural(), policyman.PolicyActionUpdate)
		if requireScope.HigherThan(allowScope) {
			return input, errors.Wrapf(httperrors.ErrNotSufficientPrivilege, "require %s allow %s", requireScope, allowScope)
		}
	}

	if input.Blob != nil {
		p, err := rbacutils.DecodePolicyData(input.Blob)
		if err != nil {
			return input, httperrors.NewInputParameterError("fail to decode policy data")
		}
		/* if p.IsSystemWidePolicy() && policyman.PolicyManager.Allow(rbacutils.ScopeSystem, userCred, consts.GetServiceType(), policy.GetModelManager().KeywordPlural(), policyman.PolicyActionUpdate) == rbacutils.Deny {
			return nil, httperrors.NewNotSufficientPrivilegeError("not allow to update system-wide policy")
		} */
		scope := input.Scope
		if len(scope) == 0 {
			scope = policy.Scope
		}
		err = validatePolicyVioldatePrivilege(userCred, scope, p)
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

	// policyman.PolicyManager.SyncOnce()
}

func (policy *SPolicy) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	policy.SEnabledIdentityBaseResource.PostUpdate(ctx, userCred, query, data)
	// policyman.PolicyManager.SyncOnce()
}

func (policy *SPolicy) PostDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	policy.SEnabledIdentityBaseResource.PostDelete(ctx, userCred)
	// policyman.PolicyManager.SyncOnce()
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
	// policyman.PolicyManager.SyncOnce()
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
	// policyman.PolicyManager.SyncOnce()
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
	// if policy.IsShared() {
	// 	return httperrors.NewInvalidStatusError("cannot delete shared policy")
	// }
	if policy.IsSystem.IsTrue() {
		return httperrors.NewForbiddenError("cannot delete system policy")
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
	if len(query.RoleId) > 0 {
		_, err := validators.ValidateModel(userCred, RoleManager, &query.RoleId)
		if err != nil {
			return nil, err
		}
		sq := RolePolicyManager.Query("policy_id").Equals("role_id", query.RoleId).SubQuery()
		q = q.In("id", sq)
	}
	if len(query.Type) > 0 {
		q = q.In("type", query.Type)
	}
	if query.IsSystem != nil {
		if *query.IsSystem {
			q = q.IsTrue("is_system")
		} else {
			q = q.IsFalse("is_system")
		}
	}
	if len(query.Scope) == 0 {
		query.Scope = string(policyman.PolicyManager.AllowScope(userCred, consts.GetServiceType(), manager.KeywordPlural(), policyman.PolicyActionList))
	}
	switch query.Scope {
	case string(rbacutils.ScopeProject):
		q = q.Equals("scope", rbacutils.ScopeProject)
	case string(rbacutils.ScopeDomain):
		q = q.NotEquals("scope", rbacutils.ScopeSystem)
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
	q = db.SharableManagerFilterByOwner(manager, q, owner, scope)
	return q
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

func (policy *SPolicy) getPolicy() (rbacutils.TPolicy, error) {
	pc, err := rbacutils.DecodePolicyData(policy.Blob)
	if err != nil {
		return nil, errors.Wrap(err, "Decode")
	}
	return pc, nil
}

func (policy *SPolicy) GetChangeOwnerCandidateDomainIds() []string {
	return db.ISharableChangeOwnerCandidateDomainIds(policy)
}

func (policy *SPolicy) fetchMatchableRoles() ([]SRole, error) {
	q := RoleManager.Query()
	candDomains := policy.GetChangeOwnerCandidateDomainIds()
	if len(candDomains) > 0 {
		q = q.In("domain_id", candDomains)
	}
	roles := make([]SRole, 0)
	err := db.FetchModelObjects(RoleManager, q, &roles)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return nil, errors.Wrap(err, "FetchRoles")
	}
	return roles, nil
}

func (policy *SPolicy) AllowPerformBindRole(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformPublicDomainInput) bool {
	return true
}

// 绑定角色
func (policy *SPolicy) PerformBindRole(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.PolicyBindRoleInput) (jsonutils.JSONObject, error) {
	var projectId string
	prefList := make([]netutils.IPV4Prefix, 0)
	for _, ipStr := range input.Ips {
		pref, err := netutils.NewIPV4Prefix(ipStr)
		if err != nil {
			return nil, errors.Wrapf(httperrors.ErrInputParameter, "invalid prefix %s", ipStr)
		}
		prefList = append(prefList, pref)
	}
	if len(input.ProjectId) > 0 {
		proj, err := ProjectManager.FetchByIdOrName(userCred, input.ProjectId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, errors.Wrapf(httperrors.ErrNotFound, "%s %s", ProjectManager.Keyword(), input.ProjectId)
			} else {
				return nil, errors.Wrap(err, "ProjectManager.FetchByIdOrName")
			}
		}
		projectId = proj.GetId()
	}
	if len(input.RoleId) == 0 {
		return nil, errors.Wrap(httperrors.ErrInputParameter, "missing role_id")
	}
	role, err := RoleManager.FetchByIdOrName(userCred, input.RoleId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, errors.Wrapf(httperrors.ErrNotFound, "%s %s", RoleManager.Keyword(), input.RoleId)
		} else {
			return nil, errors.Wrap(err, "RoleManager.FetchByIdOrName")
		}
	}
	err = RolePolicyManager.newRecord(ctx, role.GetId(), projectId, policy.Id, tristate.True, prefList)
	if err != nil {
		return nil, errors.Wrap(err, "newRecord")
	}
	return nil, nil
}

func (policy *SPolicy) GetI18N(ctx context.Context) *jsonutils.JSONDict {
	r := jsonutils.NewDict()
	act18 := locale.PredefinedPolicyI18nTable.Lookup(ctx, policy.Description)
	r.Set("description", jsonutils.NewString(act18))
	return r
}
