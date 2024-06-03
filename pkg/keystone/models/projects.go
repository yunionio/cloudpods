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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/pinyinutils"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/onecloud/pkg/util/tagutils"
)

type SProjectManager struct {
	SIdentityBaseResourceManager
}

var ProjectManager *SProjectManager

func init() {
	ProjectManager = &SProjectManager{
		SIdentityBaseResourceManager: NewIdentityBaseResourceManager(
			SProject{},
			"project",
			"project",
			"projects",
		),
	}
	ProjectManager.SetVirtualObject(ProjectManager)
	notifyclient.AddNotifyDBHookResources(ProjectManager.KeywordPlural(), ProjectManager.AliasPlural())
}

/*
+-------------+-------------+------+-----+---------+-------+
| Field       | Type        | Null | Key | Default | Extra |
+-------------+-------------+------+-----+---------+-------+
| id          | varchar(64) | NO   | PRI | NULL    |       |
| name        | varchar(64) | NO   |     | NULL    |       |
| extra       | text        | YES  |     | NULL    |       |
| description | text        | YES  |     | NULL    |       |
| enabled     | tinyint(1)  | YES  |     | NULL    |       |
| domain_id   | varchar(64) | NO   | MUL | NULL    |       |
| parent_id   | varchar(64) | YES  | MUL | NULL    |       |
| is_domain   | tinyint(1)  | NO   |     | 0       |       |
| created_at  | datetime    | YES  |     | NULL    |       |
+-------------+-------------+------+-----+---------+-------+
*/

type SProject struct {
	SIdentityBaseResource

	// 上级项目或域的ID
	ParentId string `width:"64" charset:"ascii" list:"domain" create:"domain_optional"`

	// 该项目是否为域（domain）
	IsDomain tristate.TriState `default:"false"`

	AdminId string `width:"64" charset:"ascii" nullable:"true" list:"domain"`
}

func (manager *SProjectManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{UserManager},
		{GroupManager},
	}
}

func (manager *SProjectManager) InitializeData() error {
	ctx := context.TODO()
	err := manager.initSysProject(ctx)
	if err != nil {
		return errors.Wrap(err, "initSysProject")
	}
	return nil
}

func (manager *SProjectManager) initSysProject(ctx context.Context) error {
	q := manager.Query().Equals("name", api.SystemAdminProject)
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
		log.Fatalf("duplicate system project???")
	}
	// insert
	project := SProject{}
	project.Name = api.SystemAdminProject
	project.DomainId = api.DEFAULT_DOMAIN_ID
	// project.Enabled = tristate.True
	project.Description = "Boostrap system default admin project"
	project.IsDomain = tristate.False
	project.ParentId = api.DEFAULT_DOMAIN_ID
	project.SetModelManager(manager, &project)

	err = manager.TableSpec().Insert(ctx, &project)
	if err != nil {
		return errors.Wrap(err, "insert")
	}
	return nil
}

func (project *SProject) resetAdminUser(ctx context.Context, userCred mcclient.TokenCredential) error {
	role, err := RoleManager.FetchRoleByName(options.Options.ProjectAdminRole, "", "")
	if err != nil {
		return errors.Wrapf(err, "FetchRoleByName %s", options.Options.ProjectAdminRole)
	}
	q := AssignmentManager.fetchProjectRoleUserIdsQuery(project.Id, role.Id)
	userId := struct {
		ActorId string
	}{}
	err = q.First(&userId)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return errors.Wrap(err, "query")
	}
	err = project.setAdminId(ctx, userCred, userId.ActorId)
	if err != nil {
		return errors.Wrap(err, "setAdminId")
	}
	return nil
}

func (manager *SProjectManager) NewQuery(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, useRawQuery bool) *sqlchemy.SQuery {
	return manager.Query()
}

func (manager *SProjectManager) Query(fields ...string) *sqlchemy.SQuery {
	return manager.SIdentityBaseResourceManager.Query(fields...).IsFalse("is_domain")
}

func (manager *SProjectManager) FetchProjectByName(projectName string, domainId, domainName string) (*SProject, error) {
	obj, err := db.NewModelObject(manager)
	if err != nil {
		return nil, errors.Wrap(err, "db.NewModelObject")
	}
	if len(domainId) == 0 && len(domainName) == 0 {
		q := manager.Query().Equals("name", projectName)
		cnt, err := q.CountWithError()
		if err != nil {
			return nil, errors.Wrap(err, "CountWithError")
		}
		if cnt == 0 {
			return nil, sql.ErrNoRows
		}
		if cnt > 1 {
			return nil, sqlchemy.ErrDuplicateEntry
		}
		err = q.First(obj)
		if err != nil {
			return nil, errors.Wrap(err, "q.First")
		}
	} else {
		domain, err := DomainManager.FetchDomain(domainId, domainName)
		if err != nil {
			return nil, errors.Wrap(err, "DomainManager.FetchDomain")
		}
		q := manager.Query().Equals("name", projectName).Equals("domain_id", domain.Id)
		err = q.First(obj)
		if err != nil {
			return nil, errors.Wrap(err, "q.First")
		}
	}
	return obj.(*SProject), nil
}

func (manager *SProjectManager) FetchProjectById(projectId string) (*SProject, error) {
	obj, err := db.NewModelObject(manager)
	if err != nil {
		return nil, err
	}
	q := manager.Query().Equals("id", projectId)
	err = q.First(obj)
	if err != nil {
		return nil, err
	}
	return obj.(*SProject), err
}

func (manager *SProjectManager) FetchProject(projectId, projectName string, domainId, domainName string) (*SProject, error) {
	if len(projectId) > 0 {
		return manager.FetchProjectById(projectId)
	}
	if len(projectName) > 0 {
		return manager.FetchProjectByName(projectName, domainId, domainName)
	}
	return nil, fmt.Errorf("no project Id or name provided")
}

// +onecloud:model-api-gen
type SProjectExtended struct {
	SProject

	DomainName string
}

func (proj *SProject) getDomain() (*SDomain, error) {
	return DomainManager.FetchDomainById(proj.DomainId)
}

func (proj *SProject) FetchExtend() (*SProjectExtended, error) {
	domain, err := proj.getDomain()
	if err != nil {
		return nil, err
	}
	ext := SProjectExtended{
		SProject:   *proj,
		DomainName: domain.Name,
	}
	return &ext, nil
}

// 项目列表
func (manager *SProjectManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ProjectListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SIdentityBaseResourceManager.ListItemFilter(ctx, q, userCred, query.IdentityBaseResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SIdentityBaseResourceManager.ListItemFilter")
	}

	if !query.PolicyProjectTags.IsEmpty() {
		policyFilters := tagutils.STagFilters{}
		policyFilters.AddFilters(query.PolicyProjectTags)
		q = db.ObjectIdQueryWithTagFilters(ctx, q, "id", "project", policyFilters)
	}

	userStr := query.UserId
	if len(userStr) > 0 {
		userObj, err := UserManager.FetchById(userStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(UserManager.Keyword(), userStr)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		subq := AssignmentManager.fetchUserProjectIdsQuery(userObj.GetId())
		if query.Jointable != nil && *query.Jointable {
			user := userObj.(*SUser)
			if user.DomainId == api.DEFAULT_DOMAIN_ID {
				q = q.Equals("domain_id", api.DEFAULT_DOMAIN_ID)
			} else {
				q = q.In("domain_id", []string{user.DomainId, api.DEFAULT_DOMAIN_ID})
			}
			q = q.NotIn("id", subq.SubQuery())
		} else {
			q = q.In("id", subq.SubQuery())
		}
	}

	if len(query.AdminId) > 0 {
		sq := UserManager.Query("id")
		sq = sq.Filter(
			sqlchemy.OR(
				sqlchemy.In(sq.Field("id"), query.AdminId),
				sqlchemy.In(sq.Field("name"), query.AdminId),
			),
		)
		q = q.In("admin_id", sq.SubQuery())
	}

	groupStr := query.GroupId
	if len(groupStr) > 0 {
		groupObj, err := GroupManager.FetchById(groupStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(GroupManager.Keyword(), groupStr)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		subq := AssignmentManager.fetchGroupProjectIdsQuery(groupObj.GetId())
		if query.Jointable != nil && *query.Jointable {
			group := groupObj.(*SGroup)
			if group.DomainId == api.DEFAULT_DOMAIN_ID {
				q = q.Equals("domain_id", api.DEFAULT_DOMAIN_ID)
			} else {
				q = q.In("domain_id", []string{group.DomainId, api.DEFAULT_DOMAIN_ID})
			}
			q = q.NotIn("id", subq.SubQuery())
		} else {
			q = q.In("id", subq.SubQuery())
		}
	}

	if len(query.IdpId) > 0 {
		idpObj, err := IdentityProviderManager.FetchByIdOrName(ctx, userCred, query.IdpId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(IdentityProviderManager.Keyword(), query.IdpId)
			} else {
				return nil, errors.Wrap(err, "IdentityProviderManager.FetchByIdOrName")
			}
		}
		subq := IdmappingManager.FetchPublicIdsExcludesQuery(idpObj.GetId(), api.IdMappingEntityDomain, nil)
		q = q.In("domain_id", subq.SubQuery())
	}

	return q, nil
}

func (manager *SProjectManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ProjectListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SIdentityBaseResourceManager.OrderByExtraFields(ctx, q, userCred, query.IdentityBaseResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SIdentityBaseResourceManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SProjectManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SIdentityBaseResourceManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	if field == "admin" {
		userQuery := UserManager.Query("name", "id").Distinct().SubQuery()
		q.AppendField(userQuery.Field("name", field))
		q = q.Join(userQuery, sqlchemy.Equals(q.Field("admin_id"), userQuery.Field("id")))
		q.GroupBy(userQuery.Field("name"))
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (model *SProject) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	model.ParentId = ownerId.GetProjectDomainId()
	model.IsDomain = tristate.False
	return model.SIdentityBaseResource.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (proj *SProject) GetUserCount() (int, error) {
	q := AssignmentManager.fetchProjectUserIdsQuery(proj.Id)
	return q.CountWithError()
}

func (proj *SProject) GetGroupCount() (int, error) {
	q := AssignmentManager.fetchProjectGroupIdsQuery(proj.Id)
	return q.CountWithError()
}

func (proj *SProject) ValidateDeleteCondition(ctx context.Context, info *api.ProjectDetails) error {
	if proj.IsAdminProject() {
		return httperrors.NewForbiddenError("cannot delete system project")
	}
	/*if len(info.ExtResource) > 0 {
		return httperrors.NewNotEmptyError("project contains external resources")
	}*/
	if info.UserCount > 0 {
		return httperrors.NewNotEmptyError("project contains user")
	}
	if info.GroupCount > 0 {
		return httperrors.NewNotEmptyError("project contains group")
	}
	return proj.SIdentityBaseResource.ValidateDeleteCondition(ctx, nil)
}

func (proj *SProject) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := proj.SIdentityBaseResource.Delete(ctx, userCred)
	if err != nil {
		return errors.Wrap(err, "project delete")
	}
	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:    proj,
		Action: notifyclient.ActionDelete,
	})
	return nil
}

func (proj *SProject) IsAdminProject() bool {
	return proj.Name == api.SystemAdminProject && proj.DomainId == api.DEFAULT_DOMAIN_ID
}

func (proj *SProject) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ProjectUpdateInput) (api.ProjectUpdateInput, error) {
	if len(input.Name) > 0 {
		if proj.IsAdminProject() {
			return input, httperrors.NewForbiddenError("cannot alter system project name")
		}
	}
	var err error
	input.IdentityBaseUpdateInput, err = proj.SIdentityBaseResource.ValidateUpdateData(ctx, userCred, query, input.IdentityBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SIdentityBaseResource.ValidateUpdateData")
	}

	return input, nil
}

func (manager *SProjectManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ProjectDetails {
	rows := make([]api.ProjectDetails, len(objs))

	identRows := manager.SIdentityBaseResourceManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	projIds := make([]string, len(objs))
	adminUserIds := make([]string, 0)
	for i := range rows {
		rows[i] = api.ProjectDetails{
			IdentityBaseResourceDetails: identRows[i],
		}
		proj := objs[i].(*SProject)
		projIds[i] = proj.Id
		if len(proj.AdminId) > 0 {
			adminUserIds = append(adminUserIds, proj.AdminId)
		}
	}

	extResource, extLastUpdate, err := ScopeResourceManager.FetchProjectsScopeResources(projIds)
	if err != nil {
		return rows
	}

	groupCnt, userCnt, err := AssignmentManager.fetchUserAndGroups(projIds)
	if err != nil {
		return rows
	}

	userMaps := make(map[string]SUser)
	err = db.FetchModelObjectsByIds(UserManager, "id", adminUserIds, &userMaps)
	if err != nil {
		log.Errorf("FetchModelObjectsByIds fail %s", err)
	}

	for i := range rows {
		groups, _ := groupCnt[projIds[i]]
		users, _ := userCnt[projIds[i]]
		rows[i].GroupCount = len(groups)
		rows[i].UserCount = len(users)

		rows[i].ExtResource, _ = extResource[projIds[i]]
		rows[i].ExtResourcesLastUpdate, _ = extLastUpdate[projIds[i]]
		if len(rows[i].ExtResource) == 0 {
			if rows[i].ExtResourcesLastUpdate.IsZero() {
				rows[i].ExtResourcesLastUpdate = time.Now()
			}
			nextUpdate := rows[i].ExtResourcesLastUpdate.Add(time.Duration(options.Options.FetchScopeResourceCountIntervalSeconds) * time.Second)
			rows[i].ExtResourcesNextUpdate = nextUpdate
		}
		proj := objs[i].(*SProject)
		if len(proj.AdminId) > 0 {
			if user, ok := userMaps[proj.AdminId]; ok {
				rows[i].Admin = user.Name
				rows[i].AdminDomain = user.GetDomain().Name
				rows[i].AdminDomainId = user.DomainId
			}
		}
		projOrg, err := proj.matchOrganizationNodes()
		if err != nil {
			log.Errorf("matchOrganizationNodes fail %s", err)
		} else {
			rows[i].Organization = projOrg
		}
	}

	return rows
}

func NormalizeProjectName(name string) string {
	name = pinyinutils.Text2Pinyin(name)
	newName := strings.Builder{}
	lastSlash := false
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			newName.WriteRune(c)
			lastSlash = false
		} else if c >= 'A' && c <= 'Z' {
			newName.WriteRune(c - 'A' + 'a')
			lastSlash = false
		} else if !lastSlash {
			newName.WriteRune('-')
			lastSlash = true
		}
	}
	return newName.String()
}

func (manager *SProjectManager) FetchUserProjects(userId string) ([]SProjectExtended, error) {
	projects := manager.Query().SubQuery()
	domains := DomainManager.Query().SubQuery()
	q := projects.Query(
		projects.Field("id"),
		projects.Field("name"),
		projects.Field("domain_id"),
		domains.Field("name").Label("domain_name"),
	)
	q = q.Join(domains, sqlchemy.Equals(projects.Field("domain_id"), domains.Field("id")))
	subq := AssignmentManager.fetchUserProjectIdsQuery(userId)
	q = q.Filter(sqlchemy.In(projects.Field("id"), subq))

	ret := make([]SProjectExtended, 0)
	err := q.All(&ret)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "query.All")
	}
	for i := range ret {
		ret[i].SetModelManager(manager, &ret[i])
	}
	return ret, nil
}

func (manager *SProjectManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.ProjectCreateInput) (api.ProjectCreateInput, error) {
	err := db.ValidateCreateDomainId(ownerId.GetProjectDomainId())
	if err != nil {
		return input, errors.Wrap(err, "ValidateCreateDomainId")
	}
	input.IdentityBaseResourceCreateInput, err = manager.SIdentityBaseResourceManager.ValidateCreateData(ctx, userCred, ownerId, query, input.IdentityBaseResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SIdentityBaseResourceManager.ValidateCreateData")
	}
	quota := &SIdentityQuota{Project: 1}
	quota.SetKeys(quotas.SBaseDomainQuotaKeys{DomainId: ownerId.GetProjectDomainId()})
	err = quotas.CheckSetPendingQuota(ctx, userCred, quota)
	if err != nil {
		return input, errors.Wrap(err, "CheckSetPendingQuota")
	}
	return input, nil
}

func (project *SProject) PostCreate(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) {
	project.SIdentityBaseResource.PostCreate(ctx, userCred, ownerId, query, data)

	quota := &SIdentityQuota{Project: 1}
	quota.SetKeys(quotas.SBaseDomainQuotaKeys{DomainId: ownerId.GetProjectDomainId()})
	err := quotas.CancelPendingUsage(ctx, userCred, quota, quota, true)
	if err != nil {
		log.Errorf("CancelPendingUsage fail %s", err)
	}
	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:    project,
		Action: notifyclient.ActionCreate,
	})
}

func threeMemberSystemValidatePolicies(userCred mcclient.TokenCredential, projectId string, assignPolicies rbacutils.TPolicyGroup) error {
	assignScope := assignPolicies.HighestScope()
	var checkRoles []string
	if assignScope == rbacscope.ScopeSystem {
		checkRoles = options.Options.SystemThreeAdminRoleNames
	} else if assignScope == rbacscope.ScopeDomain {
		checkRoles = options.Options.DomainThreeAdminRoleNames
	} else {
		return nil
	}
	var contains []string
	for _, roleName := range checkRoles {
		role, err := RoleManager.FetchRoleByName(roleName, "", "")
		if err != nil {
			return httperrors.NewResourceNotFoundError2(RoleManager.Keyword(), roleName)
		}
		_, adminPolicies, _ := RolePolicyManager.GetMatchPolicyGroup2(false, []string{role.Id}, projectId, "", time.Time{}, false)
		if adminPolicies[assignScope].Contains(assignPolicies[assignScope]) {
			contains = append(contains, roleName)
		}
	}
	if len(contains) != 1 {
		return errors.Wrapf(httperrors.ErrNotSufficientPrivilege, "assigning roles violates three-member policy: %s", contains)
	}
	return nil
}

func normalValidatePolicies(userCred mcclient.TokenCredential, assignPolicies rbacutils.TPolicyGroup) error {
	_, opsPolicies, err := RolePolicyManager.GetMatchPolicyGroup(userCred, time.Time{}, false)
	if err != nil {
		return errors.Wrap(err, "RolePolicyManager.GetMatchPolicyGroup")
	}
	opsScope := opsPolicies.HighestScope()
	assignScope := assignPolicies.HighestScope()
	if assignScope.HigherThan(opsScope) {
		return errors.Wrap(httperrors.ErrNotSufficientPrivilege, "assigning roles requires higher privilege scope")
	} else if assignScope == opsScope && !opsPolicies[opsScope].Contains(assignPolicies[assignScope]) {
		return errors.Wrap(httperrors.ErrNotSufficientPrivilege, "assigning roles violates operator's policy")
	}
	return nil
}

func validateAssignPolicies(userCred mcclient.TokenCredential, projectId string, assignPolicies rbacutils.TPolicyGroup) error {
	if options.Options.NoPolicyViolationCheck {
		return nil
	}
	if options.Options.ThreeAdminRoleSystem {
		return threeMemberSystemValidatePolicies(userCred, projectId, assignPolicies)
	} else {
		return normalValidatePolicies(userCred, assignPolicies)
	}
}

func validateJoinProject(userCred mcclient.TokenCredential, project *SProject, roleIds []string) error {
	return ValidateJoinProjectRoles(userCred, project.Id, roleIds)
}

func ValidateJoinProjectRoles(userCred mcclient.TokenCredential, projectId string, roleIds []string) error {
	_, assignPolicies, err := RolePolicyManager.GetMatchPolicyGroup2(false, roleIds, projectId, "", time.Time{}, false)
	if err != nil {
		return errors.Wrap(err, "RolePolicyManager.GetMatchPolicyGroup2")
	}
	return validateAssignPolicies(userCred, projectId, assignPolicies)
}

// 将用户或组加入项目
func (project *SProject) PerformJoin(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.SProjectAddUserGroupInput,
) (jsonutils.JSONObject, error) {
	err := input.Validate()
	if err != nil {
		return nil, httperrors.NewInputParameterError("%v", err)
	}

	roleIds := make([]string, 0)
	roles := make([]*SRole, 0)
	for i := range input.Roles {
		obj, err := RoleManager.FetchByIdOrName(ctx, userCred, input.Roles[i])
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(RoleManager.Keyword(), input.Roles[i])
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		role := obj.(*SRole)
		roles = append(roles, role)
		roleIds = append(roleIds, role.Id)
	}

	err = validateJoinProject(userCred, project, roleIds)
	if err != nil {
		return nil, errors.Wrap(err, "validateJoinProject")
	}

	users := make([]*SUser, 0)
	for i := range input.Users {
		obj, err := UserManager.FetchByIdOrName(ctx, userCred, input.Users[i])
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(UserManager.Keyword(), input.Users[i])
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		users = append(users, obj.(*SUser))
	}
	groups := make([]*SGroup, 0)
	for i := range input.Groups {
		obj, err := GroupManager.FetchByIdOrName(ctx, userCred, input.Groups[i])
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(GroupManager.Keyword(), input.Groups[i])
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		groups = append(groups, obj.(*SGroup))
	}

	for i := range users {
		for j := range roles {
			err = AssignmentManager.ProjectAddUser(ctx, userCred, project, users[i], roles[j])
			if err != nil {
				return nil, httperrors.NewGeneralError(err)
			}
		}
	}
	for i := range groups {
		for j := range roles {
			err = AssignmentManager.projectAddGroup(ctx, userCred, project, groups[i], roles[j])
			if err != nil {
				return nil, httperrors.NewGeneralError(err)
			}
		}
	}

	if input.EnableAllUsers {
		for i := range users {
			db.EnabledPerformEnable(users[i], ctx, userCred, true)
		}
	}

	return nil, nil
}

// 将用户或组移出项目
func (project *SProject) PerformLeave(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.SProjectRemoveUserGroupInput,
) (jsonutils.JSONObject, error) {
	err := input.Validate()
	if err != nil {
		return nil, httperrors.NewInputParameterError("%v", err)
	}

	for i := range input.UserRoles {
		userObj, err := UserManager.FetchByIdOrName(ctx, userCred, input.UserRoles[i].User)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(UserManager.Keyword(), input.UserRoles[i].User)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		roleObj, err := RoleManager.FetchByIdOrName(ctx, userCred, input.UserRoles[i].Role)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(RoleManager.Keyword(), input.UserRoles[i].Role)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		err = AssignmentManager.projectRemoveUser(ctx, userCred, project, userObj.(*SUser), roleObj.(*SRole))
		if err != nil {
			return nil, httperrors.NewGeneralError(err)
		}
	}
	for i := range input.GroupRoles {
		groupObj, err := GroupManager.FetchByIdOrName(ctx, userCred, input.GroupRoles[i].Group)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(GroupManager.Keyword(), input.GroupRoles[i].Group)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		roleObj, err := RoleManager.FetchByIdOrName(ctx, userCred, input.GroupRoles[i].Role)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(RoleManager.Keyword(), input.GroupRoles[i].Role)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		err = AssignmentManager.projectRemoveGroup(ctx, userCred, project, groupObj.(*SGroup), roleObj.(*SRole))
		if err != nil {
			return nil, httperrors.NewGeneralError(err)
		}
	}
	return nil, nil
}

func (project *SProject) GetUsages() []db.IUsage {
	if project.Deleted {
		return nil
	}
	usage := SIdentityQuota{Project: 1}
	usage.SetKeys(quotas.SBaseDomainQuotaKeys{DomainId: project.DomainId})
	return []db.IUsage{
		&usage,
	}
}

func (manager *SProjectManager) NewProject(ctx context.Context, projectName string, desc string, domainId string) (*SProject, error) {
	project := &SProject{}
	project.SetModelManager(ProjectManager, project)
	ownerId := &db.SOwnerId{}
	if manager.NamespaceScope() == rbacscope.ScopeDomain {
		ownerId.DomainId = domainId
	}
	project.DomainId = domainId
	project.Description = desc
	project.IsDomain = tristate.False
	project.ParentId = domainId
	var err = func() error {
		lockman.LockRawObject(ctx, manager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, manager.Keyword(), "name")

		newName, err := db.GenerateName(ctx, ProjectManager, ownerId, projectName)
		if err != nil {
			// ignore the error
			log.Errorf("db.GenerateName error %s for default domain project %s", err, projectName)
			newName = projectName
		}
		project.Name = newName

		return ProjectManager.TableSpec().Insert(ctx, project)
	}()
	if err != nil {
		return nil, errors.Wrap(err, "Insert")
	}
	return project, nil
}

func (project *SProject) PerformSetAdmin(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.SProjectSetAdminInput,
) (jsonutils.JSONObject, error) {
	// unset admin
	if len(input.UserId) == 0 {
		return nil, project.setAdminId(ctx, userCred, input.UserId)
	}

	var user *SUser
	var role *SRole

	{
		obj, err := UserManager.FetchByIdOrName(ctx, userCred, input.UserId)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(UserManager.Keyword(), input.UserId)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		user = obj.(*SUser)
	}

	{
		obj, err := RoleManager.FetchByIdOrName(ctx, userCred, options.Options.ProjectAdminRole)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(RoleManager.Keyword(), options.Options.ProjectAdminRole)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		role = obj.(*SRole)
	}

	inProject, err := AssignmentManager.isUserInProjectWithRole(user.Id, project.Id, role.Id)
	if err != nil {
		return nil, errors.Wrap(err, "isUserInProjectWithRole")
	}

	if !inProject {
		err = AssignmentManager.ProjectAddUser(ctx, userCred, project, user, role)
		if err != nil {
			return nil, httperrors.NewGeneralError(err)
		}
	}

	err = project.setAdminId(ctx, userCred, user.Id)
	if err != nil {
		return nil, errors.Wrap(err, "setAdminId")
	}

	return nil, nil
}

func (project *SProject) setAdminId(ctx context.Context, userCred mcclient.TokenCredential, userId string) error {
	if project.AdminId != userId {
		diff, err := db.Update(project, func() error {
			project.AdminId = userId
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "update adminId")
		}
		db.OpsLog.LogEvent(project, db.ACT_UPDATE, diff, userCred)
		logclient.AddSimpleActionLog(project, logclient.ACT_UPDATE, diff, userCred, true)
	}
	return nil
}

func (project *SProject) matchOrganizationNodes() (*api.SProjectOrganization, error) {
	orgs, err := OrganizationManager.FetchOrgnaizations(func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		q = q.Equals("type", api.OrgTypeProject)
		q = q.IsTrue("enabled")
		return q
	})
	if err != nil {
		return nil, errors.Wrap(err, "FetchOrgnaizations")
	}
	if len(orgs) == 0 {
		return nil, nil
	} else if len(orgs) > 1 {
		return nil, errors.Wrap(httperrors.ErrDuplicateResource, "multiple enabled organizations")
	}
	org := &orgs[0]
	tags, err := project.GetAllOrganizationMetadata()
	if err != nil {
		return nil, errors.Wrap(err, "GetAllOrganizationMetadata")
	}
	if len(tags) == 0 {
		return nil, nil
	}
	log.Debugf("matchOrganizationNodes %s", jsonutils.Marshal(tags))
	projOrg, err := org.getProjectOrganization(tags)
	if err != nil {
		return nil, errors.Wrap(err, "getProjectOrganization")
	}
	return projOrg, nil
}

func (manager *SProjectManager) FilterByOwner(ctx context.Context, q *sqlchemy.SQuery, man db.FilterByOwnerProvider, userCred mcclient.TokenCredential, owner mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	if userCred != nil && scope != rbacscope.ScopeSystem && scope != rbacscope.ScopeDomain {
		q = q.Equals("id", owner.GetProjectId())
	}
	return manager.SIdentityBaseResourceManager.FilterByOwner(ctx, q, man, userCred, owner, scope)
}

func (manager *SProjectManager) GetSystemProject() (*SProject, error) {
	q := manager.Query().Equals("name", api.SystemAdminProject)
	ret := &SProject{}
	ret.SetModelManager(manager, ret)
	err := q.First(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SProject) StartProjectCleanTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ProjectCleanTask", self, userCred, nil, "", "", nil)
	if err != nil {
		return err
	}
	return task.ScheduleRun(nil)
}

func (self *SProject) GetEmptyProjects() ([]SProject, error) {
	q := ProjectManager.Query().IsFalse("pending_deleted").NotEquals("name", api.SystemAdminProject)
	scopes := []SScopeResource{}
	ScopeResourceManager.Query().GT("count", 0).All(&scopes)
	ids := []string{}
	for _, scope := range scopes {
		ids = append(ids, scope.ProjectId)
	}
	projects := []SProject{}
	if len(ids) == 0 {
		return projects, nil
	}
	q = q.Filter(sqlchemy.NotIn(q.Field("id"), ids))
	err := db.FetchModelObjects(ProjectManager, q, &projects)
	if err != nil {
		return nil, err
	}
	return projects, nil
}

func (manager *SProjectManager) PerformClean(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.ProjectCleanInput) (jsonutils.JSONObject, error) {
	if !userCred.HasSystemAdminPrivilege() {
		return nil, httperrors.NewForbiddenError("not allow clean projects")
	}
	system, err := manager.GetSystemProject()
	if err != nil {
		return nil, err
	}
	return nil, system.StartProjectCleanTask(ctx, userCred)
}
