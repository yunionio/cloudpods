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
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/locale"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SRoleManager struct {
	SIdentityBaseResourceManager
	db.SSharableBaseResourceManager
}

var RoleManager *SRoleManager

func init() {
	RoleManager = &SRoleManager{
		SIdentityBaseResourceManager: NewIdentityBaseResourceManager(
			SRole{},
			"role",
			"role",
			"roles",
		),
	}
	RoleManager.SetVirtualObject(RoleManager)
}

/*
+------------+--------------+------+-----+----------+-------+
| Field      | Type         | Null | Key | Default  | Extra |
+------------+--------------+------+-----+----------+-------+
| id         | varchar(64)  | NO   | PRI | NULL     |       |
| name       | varchar(255) | NO   | MUL | NULL     |       |
| extra      | text         | YES  |     | NULL     |       |
| domain_id  | varchar(64)  | NO   |     | <<null>> |       |
| created_at | datetime     | YES  |     | NULL     |       |
+------------+--------------+------+-----+----------+-------+
*/

type SRole struct {
	SIdentityBaseResource    `"name->update":""`
	db.SSharableBaseResource `"is_public->create":"domain_optional" "public_scope->create":"domain_optional"`
}

func (manager *SRoleManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{ProjectManager, UserManager},
		{ProjectManager, GroupManager},
	}
}

const (
	ROLE_DEFAULT_DOMAIN_ID = "<<null>>"
)

func (manager *SRoleManager) InitializeData() error {
	q := manager.Query()
	q = q.IsNull("description").IsNotNull("extra")
	roles := make([]SRole, 0)
	err := db.FetchModelObjects(manager, q, &roles)
	if err != nil {
		return errors.Wrap(err, "query")
	}
	for i := range roles {
		if gotypes.IsNil(roles[i].Extra) {
			continue
		}
		desc, _ := roles[i].Extra.GetString("description")
		_, err = db.Update(&roles[i], func() error {
			roles[i].Description = desc
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "update description")
		}
	}
	err = manager.initializeDomainId()
	if err != nil {
		return errors.Wrap(err, "InitializeDomainId")
	}
	err = manager.initSysRole(context.TODO())
	if err != nil {
		return errors.Wrap(err, "initSysRole")
	}
	return nil
}

func (manager *SRoleManager) initializeDomainId() error {
	q := manager.Query().Equals("domain_id", ROLE_DEFAULT_DOMAIN_ID)
	roles := make([]SRole, 0)
	err := db.FetchModelObjects(manager, q, &roles)
	if err != nil {
		return err
	}
	for i := range roles {
		db.Update(&roles[i], func() error {
			roles[i].DomainId = api.DEFAULT_DOMAIN_ID
			return nil
		})
	}
	return nil
}

func (manager *SRoleManager) initSysRole(ctx context.Context) error {
	q := manager.Query().Equals("name", api.SystemAdminRole)
	q = q.Equals("domain_id", api.DEFAULT_DOMAIN_ID)
	cnt, err := q.CountWithError()
	if err != nil {
		return errors.Wrap(err, "query")
	}
	if cnt == 1 {
		return nil
	}
	if cnt > 2 {
		// ???
		log.Fatalf("duplicate system role???")
	}
	// insert
	role := SRole{}
	role.Name = api.SystemAdminRole
	role.IsPublic = true
	role.PublicScope = string(rbacscope.ScopeSystem)
	role.DomainId = api.DEFAULT_DOMAIN_ID
	role.Description = "Boostrap system default admin role"
	role.SetModelManager(manager, &role)

	err = manager.TableSpec().Insert(ctx, &role)
	if err != nil {
		return errors.Wrap(err, "insert")
	}
	return nil
}

func (role *SRole) GetUserCount() (int, error) {
	q := AssignmentManager.fetchRoleUserIdsQuery(role.Id)
	return q.CountWithError()
}

func (role *SRole) GetGroupCount() (int, error) {
	q := AssignmentManager.fetchRoleGroupIdsQuery(role.Id)
	return q.CountWithError()
}

func (role *SRole) GetProjectCount() (int, error) {
	q := AssignmentManager.fetchRoleProjectIdsQuery(role.Id)
	return q.CountWithError()
}

func (role *SRole) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.RoleUpdateInput) (api.RoleUpdateInput, error) {
	if len(input.Name) > 0 {
		return input, httperrors.NewForbiddenError("cannot alter name of role")
	}
	var err error
	input.IdentityBaseUpdateInput, err = role.SIdentityBaseResource.ValidateUpdateData(ctx, userCred, query, input.IdentityBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SIdentityBaseResource.ValidateUpdateData")
	}

	return input, nil
}

func (role *SRole) IsSystemRole() bool {
	return role.Name == api.SystemAdminRole && role.DomainId == api.DEFAULT_DOMAIN_ID
}

func (role *SRole) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	// if role.IsShared() {
	// 	return httperrors.NewInvalidStatusError("cannot delete shared role")
	// }
	if role.IsSystemRole() {
		return httperrors.NewForbiddenError("cannot delete system role")
	}
	usrCnt, _ := role.GetUserCount()
	if usrCnt > 0 {
		return httperrors.NewNotEmptyError("role is being assigned to user")
	}
	grpCnt, _ := role.GetGroupCount()
	if grpCnt > 0 {
		return httperrors.NewNotEmptyError("role is being assigned to group")
	}
	return role.SIdentityBaseResource.ValidateDeleteCondition(ctx, nil)
}

func (manager *SRoleManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.RoleDetails {
	rows := make([]api.RoleDetails, len(objs))

	identRows := manager.SIdentityBaseResourceManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	shareRows := manager.SSharableBaseResourceManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.RoleDetails{
			IdentityBaseResourceDetails: identRows[i],
			SharableResourceBaseInfo:    shareRows[i],
		}
		role := objs[i].(*SRole)
		rows[i].UserCount, _ = role.GetUserCount()
		rows[i].GroupCount, _ = role.GetGroupCount()
		rows[i].ProjectCount, _ = role.GetProjectCount()
		names, _, _ := RolePolicyManager.GetMatchPolicyGroup2(false, []string{role.Id}, "", "", time.Time{}, true)
		rows[i].Policies = names
		mp := make([]string, 0)
		for _, v := range names {
			mp = append(mp, v...)
		}
		rows[i].MatchPolicies = mp
	}

	return rows
}

// 角色列表
func (manager *SRoleManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.RoleListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SIdentityBaseResourceManager.ListItemFilter(ctx, q, userCred, query.IdentityBaseResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SIdentityBaseResourceManager.ListItemFilter")
	}

	q, err = manager.SSharableBaseResourceManager.ListItemFilter(ctx, q, userCred, query.SharableResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableBaseResourceManager.ListItemFilter")
	}

	var projectId string
	projectStr := query.ProjectId
	if len(projectStr) > 0 {
		project, err := ProjectManager.FetchProjectById(projectStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(ProjectManager.Keyword(), projectStr)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		projectId = project.Id
	}

	userStr := query.UserId
	if len(projectId) > 0 && len(userStr) > 0 {
		userObj, err := UserManager.FetchById(userStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(UserManager.Keyword(), userStr)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		subq := AssignmentManager.fetchUserProjectRoleIdsQuery(userObj.GetId(), projectId)
		q = q.In("id", subq.SubQuery())
	}

	groupStr := query.GroupId
	if len(projectId) > 0 && len(groupStr) > 0 {
		groupObj, err := GroupManager.FetchById(groupStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(GroupManager.Keyword(), groupStr)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		subq := AssignmentManager.fetchGroupProjectRoleIdsQuery(groupObj.GetId(), projectId)
		q = q.In("id", subq.SubQuery())
	}

	return q, nil
}

func (manager *SRoleManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.RoleListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SIdentityBaseResourceManager.OrderByExtraFields(ctx, q, userCred, query.IdentityBaseResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SIdentityBaseResourceManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SRoleManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SIdentityBaseResourceManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (role *SRole) UpdateInContext(ctx context.Context, userCred mcclient.TokenCredential, ctxObjs []db.IModel, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if len(ctxObjs) != 2 {
		return nil, httperrors.NewInputParameterError("not supported update context")
	}
	project, ok := ctxObjs[0].(*SProject)
	if !ok {
		return nil, httperrors.NewInputParameterError("not supported update context %s", ctxObjs[0].Keyword())
	}
	if project.DomainId != role.DomainId {
		projectOwner := &db.SOwnerId{
			ProjectId: project.Id,
			DomainId:  project.DomainId,
		}
		if !role.IsSharable(projectOwner) {
			return nil, httperrors.NewInputParameterError("inconsistent domain for project and roles")
		}
	}
	err := validateJoinProject(userCred, project, []string{role.Id})
	if err != nil {
		return nil, errors.Wrap(err, "validateJoinProject")
	}
	switch obj := ctxObjs[1].(type) {
	case *SUser:
		return nil, AssignmentManager.ProjectAddUser(ctx, userCred, project, obj, role)
	case *SGroup:
		return nil, AssignmentManager.projectAddGroup(ctx, userCred, project, obj, role)
	default:
		return nil, httperrors.NewInputParameterError("not supported secondary update context %s", ctxObjs[0].Keyword())
	}
}

func (role *SRole) DeleteInContext(ctx context.Context, userCred mcclient.TokenCredential, ctxObjs []db.IModel, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if len(ctxObjs) != 2 {
		return nil, httperrors.NewInputParameterError("not supported update context")
	}
	project, ok := ctxObjs[0].(*SProject)
	if !ok {
		return nil, httperrors.NewInputParameterError("not supported update context %s", ctxObjs[0].Keyword())
	}
	switch obj := ctxObjs[1].(type) {
	case *SUser:
		return nil, AssignmentManager.projectRemoveUser(ctx, userCred, project, obj, role)
	case *SGroup:
		return nil, AssignmentManager.projectRemoveGroup(ctx, userCred, project, obj, role)
	default:
		return nil, httperrors.NewInputParameterError("not supported secondary update context %s", ctxObjs[0].Keyword())
	}
}

func (manager *SRoleManager) FetchRoleByName(roleName string, domainId, domainName string) (*SRole, error) {
	obj, err := db.NewModelObject(manager)
	if err != nil {
		return nil, err
	}
	domain, err := DomainManager.FetchDomain(domainId, domainName)
	if err != nil {
		return nil, err
	}
	q := manager.Query().Equals("name", roleName).Equals("domain_id", domain.Id)
	err = q.First(obj)
	if err != nil {
		return nil, err
	}
	return obj.(*SRole), err
}

func (manager *SRoleManager) FetchRoleById(roleId string) (*SRole, error) {
	obj, err := db.NewModelObject(manager)
	if err != nil {
		return nil, err
	}
	q := manager.Query().Equals("id", roleId)
	err = q.First(obj)
	if err != nil {
		return nil, err
	}
	return obj.(*SRole), err
}

func (manager *SRoleManager) FetchRole(roleId, roleName string, domainId, domainName string) (*SRole, error) {
	if len(roleId) > 0 {
		return manager.FetchRoleById(roleId)
	}
	if len(roleName) > 0 {
		return manager.FetchRoleByName(roleName, domainId, domainName)
	}
	return nil, fmt.Errorf("no role Id or name provided")
}

func (role *SRole) IsShared() bool {
	return db.SharableModelIsShared(role)
}

func (role *SRole) IsSharable(reqUsrId mcclient.IIdentityProvider) bool {
	return db.SharableModelIsSharable(role, reqUsrId)
}

func (role *SRole) PerformPublic(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformPublicDomainInput) (jsonutils.JSONObject, error) {
	err := db.SharablePerformPublic(role, ctx, userCred, apis.PerformPublicProjectInput{PerformPublicDomainInput: input})
	if err != nil {
		return nil, errors.Wrap(err, "SharablePerformPublic")
	}
	return nil, nil
}

func (role *SRole) PerformPrivate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformPrivateInput) (jsonutils.JSONObject, error) {
	err := db.SharablePerformPrivate(role, ctx, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "SharablePerformPrivate")
	}
	return nil, nil
}

func (role *SRole) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	db.SharableModelCustomizeCreate(role, ctx, userCred, ownerId, query, data)
	return role.SIdentityBaseResource.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (role *SRole) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	db.SharedResourceManager.CleanModelShares(ctx, userCred, role)
	return role.SIdentityBaseResource.RealDelete(ctx, userCred)
}

func (manager *SRoleManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.RoleCreateInput,
) (api.RoleCreateInput, error) {
	err := db.ValidateCreateDomainId(ownerId.GetProjectDomainId())
	if err != nil {
		return input, errors.Wrap(err, "ValidateCreateDomainId")
	}

	input.IdentityBaseResourceCreateInput, err = manager.SIdentityBaseResourceManager.ValidateCreateData(ctx, userCred, ownerId, query, input.IdentityBaseResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SIdentityBaseResourceManager.ValidateCreateData")
	}
	input.SharableResourceBaseCreateInput, err = db.SharableManagerValidateCreateData(manager, ctx, userCred, ownerId, query, input.SharableResourceBaseCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SharableManagerValidateCreateData")
	}

	quota := &SIdentityQuota{
		SBaseDomainQuotaKeys: quotas.SBaseDomainQuotaKeys{DomainId: ownerId.GetProjectDomainId()},
		Role:                 1,
	}
	err = quotas.CheckSetPendingQuota(ctx, userCred, quota)
	if err != nil {
		return input, errors.Wrap(err, "CheckSetPendingQuota")
	}

	return input, nil
}

func (role *SRole) GetUsages() []db.IUsage {
	if role.Deleted {
		return nil
	}
	usage := SIdentityQuota{Role: 1}
	usage.SetKeys(quotas.SBaseDomainQuotaKeys{DomainId: role.DomainId})
	return []db.IUsage{
		&usage,
	}
}

func (role *SRole) PostCreate(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) {
	role.SIdentityBaseResource.PostCreate(ctx, userCred, ownerId, query, data)

	quota := &SIdentityQuota{
		SBaseDomainQuotaKeys: quotas.SBaseDomainQuotaKeys{DomainId: ownerId.GetProjectDomainId()},
		Role:                 1,
	}
	err := quotas.CancelPendingUsage(ctx, userCred, quota, quota, true)
	if err != nil {
		log.Errorf("CancelPendingUsage fail %s", err)
	}
}

func (manager *SRoleManager) FilterByOwner(ctx context.Context, q *sqlchemy.SQuery, man db.FilterByOwnerProvider, userCred mcclient.TokenCredential, owner mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	return db.SharableManagerFilterByOwner(ctx, manager, q, userCred, owner, scope)
}

func (role *SRole) GetSharableTargetDomainIds() []string {
	return nil
}

func (role *SRole) GetRequiredSharedDomainIds() []string {
	return []string{role.DomainId}
}

func (role *SRole) GetSharedDomains() []string {
	return db.SharableGetSharedProjects(role, db.SharedTargetDomain)
}

func validateRolePolicies(userCred mcclient.TokenCredential, policyIds []string) error {
	_, assignPolicies, err := RolePolicyManager.GetPolicyGroupByIds(policyIds, false)
	if err != nil {
		return errors.Wrapf(err, "RolePolicyManager.GetPolicyGroupByIds %s", policyIds)
	}
	return validateAssignPolicies(userCred, "", assignPolicies)
}

func (role *SRole) PerformSetPolicies(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.RolePerformSetPoliciesInput) (jsonutils.JSONObject, error) {
	if len(input.Action) == 0 {
		input.Action = api.ROLE_SET_POLICY_ACTION_DEFAULT
	}
	inputPolicyIds := stringutils2.NewSortedStrings(nil)
	normalInputIds := stringutils2.NewSortedStrings(nil)
	normalInputs := make(map[string]sRolePerformAddPolicyInput, len(input.Policies))
	for i := range input.Policies {
		normalInput, err := role.normalizeRoleAddPolicyInput(ctx, userCred, input.Policies[i])
		if err != nil {
			return nil, errors.Wrapf(err, "normalizeRoleAddPolicyInput at %d", i)
		}
		inputPolicyIds = stringutils2.Append(inputPolicyIds, normalInput.policyId)
		idstr := normalInput.getId()
		if _, ok := normalInputs[idstr]; !ok {
			normalInputs[idstr] = normalInput
			normalInputIds = stringutils2.Append(normalInputIds, idstr)
		} else {
			log.Warningf("duplicate input key id %s", idstr)
		}
	}

	existRpList, err := RolePolicyManager.fetchByRoleId(role.Id)
	if err != nil {
		return nil, errors.Wrap(err, "RolePolicyManager.fetchByRoleId")
	}

	existPolicyIds := stringutils2.NewSortedStrings(nil)
	existRpIds := stringutils2.NewSortedStrings(nil)
	existRpMap := make(map[string]*SRolePolicy)
	for i := range existRpList {
		existPolicyIds = stringutils2.Append(existPolicyIds, existRpList[i].PolicyId)
		idstr := existRpList[i].GetId()
		if _, ok := existRpMap[idstr]; !ok {
			existRpMap[idstr] = &existRpList[i]
			existRpIds = stringutils2.Append(existRpIds, idstr)
		}
	}

	addedIds, updatedIds, deletedIds := stringutils2.Split(normalInputIds, existRpIds)

	isBootStrap, err := RolePolicyManager.isBootstrapRolePolicy()
	if !isBootStrap {
		// validate
		var newPolicyIds []string
		if input.Action == api.ROLE_SET_POLICY_ACTION_REPLACE {
			newPolicyIds = inputPolicyIds
		} else {
			newPolicyIds = stringutils2.Append(newPolicyIds, inputPolicyIds...)
			newPolicyIds = stringutils2.Append(newPolicyIds, existPolicyIds...)
		}
		err = validateRolePolicies(userCred, newPolicyIds)
		if err != nil {
			return nil, errors.Wrap(err, "validateRolePolicies")
		}
	}

	if input.Action == api.ROLE_SET_POLICY_ACTION_REPLACE {
		for _, idstr := range deletedIds {
			toDel := existRpMap[idstr]
			err := RolePolicyManager.deleteRecord(ctx, toDel.RoleId, toDel.ProjectId, toDel.PolicyId)
			if err != nil {
				return nil, errors.Wrap(err, "RolePolicyManager.deleteRecord")
			}
		}
	}

	for _, idstr := range updatedIds {
		toUpdate := normalInputs[idstr]
		err := RolePolicyManager.newRecord(ctx, toUpdate.roleId, toUpdate.projectId, toUpdate.policyId, tristate.True, toUpdate.prefixes, toUpdate.validSince, toUpdate.validUntil)
		if err != nil {
			return nil, errors.Wrap(err, "RolePolicyManager.updateRecord")
		}
	}

	for _, idstr := range addedIds {
		toAdd := normalInputs[idstr]
		err := RolePolicyManager.newRecord(ctx, toAdd.roleId, toAdd.projectId, toAdd.policyId, tristate.True, toAdd.prefixes, toAdd.validSince, toAdd.validUntil)
		if err != nil {
			return nil, errors.Wrap(err, "RolePolicyManager.newRecord")
		}
	}

	return nil, nil
}

type sRolePerformAddPolicyInput struct {
	prefixes  []netutils.IPV4Prefix
	roleId    string
	projectId string
	policyId  string

	validSince time.Time
	validUntil time.Time
}

func (s sRolePerformAddPolicyInput) getId() string {
	return fmt.Sprintf("%s:%s:%s", s.roleId, s.projectId, s.policyId)
}

func (role *SRole) normalizeRoleAddPolicyInput(ctx context.Context, userCred mcclient.TokenCredential, input api.RolePerformAddPolicyInput) (sRolePerformAddPolicyInput, error) {
	output := sRolePerformAddPolicyInput{}
	prefList := make([]netutils.IPV4Prefix, 0)
	for _, ipStr := range input.Ips {
		pref, err := netutils.NewIPV4Prefix(ipStr)
		if err != nil {
			return output, errors.Wrapf(httperrors.ErrInputParameter, "invalid prefix %s", ipStr)
		}
		prefList = append(prefList, pref)
	}
	if len(input.ProjectId) > 0 {
		proj, err := ProjectManager.FetchByIdOrName(ctx, userCred, input.ProjectId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return output, errors.Wrapf(httperrors.ErrNotFound, "%s %s", ProjectManager.Keyword(), input.ProjectId)
			} else {
				return output, errors.Wrap(err, "ProjectManager.FetchByIdOrName")
			}
		}
		output.projectId = proj.GetId()
	}
	if len(input.PolicyId) == 0 {
		return output, errors.Wrap(httperrors.ErrInputParameter, "missing policy_id")
	}
	policy, err := PolicyManager.FetchByIdOrName(ctx, userCred, input.PolicyId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return output, errors.Wrapf(httperrors.ErrNotFound, "%s %s", PolicyManager.Keyword(), input.PolicyId)
		} else {
			return output, errors.Wrap(err, "PolicyManager.FetchByIdOrName")
		}
	}
	output.roleId = role.Id
	output.prefixes = prefList
	output.policyId = policy.GetId()
	output.validSince = input.ValidSince
	output.validUntil = input.ValidUntil
	return output, nil
}

func (role *SRole) PerformAddPolicy(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.RolePerformAddPolicyInput) (jsonutils.JSONObject, error) {
	normalInput, err := role.normalizeRoleAddPolicyInput(ctx, userCred, input)
	if err != nil {
		return nil, errors.Wrap(err, "normalizeRoleAddPolicyInput")
	}

	isBootStrap, err := RolePolicyManager.isBootstrapRolePolicy()
	if !isBootStrap {
		// validate
		rps, err := RolePolicyManager.fetchByRoleId(role.Id)
		if err != nil {
			return nil, errors.Wrap(err, "fetchByRoleId")
		}
		newPolicyIds := stringutils2.NewSortedStrings(nil)
		for i := range rps {
			newPolicyIds = stringutils2.Append(newPolicyIds, rps[i].PolicyId)
		}
		newPolicyIds = stringutils2.Append(newPolicyIds, normalInput.policyId)
		err = validateRolePolicies(userCred, newPolicyIds)
		if err != nil {
			return nil, errors.Wrap(err, "validateRolePolicies")
		}
	}

	err = RolePolicyManager.newRecord(ctx, normalInput.roleId, normalInput.projectId, normalInput.policyId, tristate.True, normalInput.prefixes, normalInput.validSince, normalInput.validUntil)
	if err != nil {
		return nil, errors.Wrap(err, "newRecord")
	}
	return nil, nil
}

func (role *SRole) PerformRemovePolicy(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.RolePerformRemovePolicyInput) (jsonutils.JSONObject, error) {
	if len(input.ProjectId) > 0 {
		proj, err := ProjectManager.FetchByIdOrName(ctx, userCred, input.ProjectId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, errors.Wrapf(httperrors.ErrNotFound, "%s %s", ProjectManager.Keyword(), input.ProjectId)
			} else {
				return nil, errors.Wrap(err, "ProjectManager.FetchByIdOrName")
			}
		}
		input.ProjectId = proj.GetId()
	}
	if len(input.PolicyId) == 0 {
		return nil, errors.Wrap(httperrors.ErrInputParameter, "missing policy_id")
	}
	policy, err := PolicyManager.FetchByIdOrName(ctx, userCred, input.PolicyId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, errors.Wrapf(httperrors.ErrNotFound, "%s %s", PolicyManager.Keyword(), input.PolicyId)
		} else {
			return nil, errors.Wrap(err, "PolicyManager.FetchByIdOrName")
		}
	}
	err = RolePolicyManager.deleteRecord(ctx, role.Id, input.ProjectId, policy.GetId())
	if err != nil {
		return nil, errors.Wrap(err, "deleteRecord")
	}
	return nil, nil
}

func (role *SRole) GetChangeOwnerCandidateDomainIds() []string {
	return db.ISharableChangeOwnerCandidateDomainIds(role)
}

func (role *SRole) GetI18N(ctx context.Context) *jsonutils.JSONDict {
	r := jsonutils.NewDict()
	act18 := locale.PredefinedRoleI18nTable.Lookup(ctx, role.Description)
	r.Set("description", jsonutils.NewString(act18))
	return r
}
