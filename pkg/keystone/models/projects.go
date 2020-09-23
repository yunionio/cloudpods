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
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/pinyinutils"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
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

	ParentId string `width:"64" charset:"ascii" list:"domain" create:"domain_optional"`

	IsDomain tristate.TriState `default:"false" nullable:"false"`
}

func (manager *SProjectManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{UserManager},
		{GroupManager},
	}
}

func (manager *SProjectManager) InitializeData() error {
	return manager.initSysProject(context.TODO())
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

func (proj *SProject) ValidateDeleteCondition(ctx context.Context) error {
	if proj.IsAdminProject() {
		return httperrors.NewForbiddenError("cannot delete system project")
	}
	external, _, _ := proj.getExternalResources()
	if len(external) > 0 {
		return httperrors.NewNotEmptyError("project contains external resources")
	}
	usrCnt, _ := proj.GetUserCount()
	if usrCnt > 0 {
		return httperrors.NewNotEmptyError("project contains user")
	}
	grpCnt, _ := proj.GetGroupCount()
	if grpCnt > 0 {
		return httperrors.NewNotEmptyError("project contains group")
	}
	return proj.SIdentityBaseResource.ValidateDeleteCondition(ctx)
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

	for i := range rows {
		rows[i] = api.ProjectDetails{
			IdentityBaseResourceDetails: identRows[i],
		}
		rows[i] = projectExtra(objs[i].(*SProject), rows[i])
	}

	return rows
}

func (proj *SProject) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.ProjectDetails, error) {
	return api.ProjectDetails{}, nil
}

func projectExtra(proj *SProject, out api.ProjectDetails) api.ProjectDetails {
	out.GroupCount, _ = proj.GetGroupCount()
	out.UserCount, _ = proj.GetUserCount()
	external, update, _ := proj.getExternalResources()
	if len(external) > 0 {
		out.ExtResource = jsonutils.Marshal(external)
		out.ExtResourcesLastUpdate = update
		if update.IsZero() {
			update = time.Now()
		}
		nextUpdate := update.Add(time.Duration(options.Options.FetchScopeResourceCountIntervalSeconds) * time.Second)
		out.ExtResourcesNextUpdate = nextUpdate
	}
	return out
}

func (proj *SProject) getExternalResources() (map[string]int, time.Time, error) {
	return ScopeResourceManager.getScopeResource("", proj.Id, "")
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

func (self *SProject) PostCreate(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) {
	self.SIdentityBaseResource.PostCreate(ctx, userCred, ownerId, query, data)

	quota := &SIdentityQuota{Project: 1}
	quota.SetKeys(quotas.SBaseDomainQuotaKeys{DomainId: ownerId.GetProjectDomainId()})
	err := quotas.CancelPendingUsage(ctx, userCred, quota, quota, true)
	if err != nil {
		log.Errorf("CancelPendingUsage fail %s", err)
	}
}

func validateJoinProject(userCred mcclient.TokenCredential, project *SProject, roleIds []string) error {
	_, opsPolicies, _ := RolePolicyManager.GetMatchPolicyGroup(userCred, false)
	_, assignPolicies, _ := RolePolicyManager.GetMatchPolicyGroup2(false, roleIds, project.Id, "", false)
	opsScope := opsPolicies.HighestScope()
	assignScope := assignPolicies.HighestScope()
	if assignScope.HigherThan(opsScope) {
		return errors.Wrap(httperrors.ErrNotSufficientPrivilege, "assigning roles requires higher privilege scope")
	} else if assignScope == opsScope && opsPolicies[opsScope].ViolatedBy(assignPolicies[assignScope]) {
		return errors.Wrap(httperrors.ErrNotSufficientPrivilege, "assigning roles violates operator's policy")
	}
	return nil
}

func (project *SProject) AllowPerformJoin(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.SProjectAddUserGroupInput,
) bool {
	return db.IsAdminAllowPerform(userCred, project, "join")
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
		obj, err := RoleManager.FetchByIdOrName(userCred, input.Roles[i])
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
		obj, err := UserManager.FetchByIdOrName(userCred, input.Users[i])
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
		obj, err := GroupManager.FetchByIdOrName(userCred, input.Groups[i])
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

	return nil, nil
}

func (project *SProject) AllowPerformLeave(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) bool {
	return db.IsAdminAllowPerform(userCred, project, "leave")
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
		userObj, err := UserManager.FetchByIdOrName(userCred, input.UserRoles[i].User)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(UserManager.Keyword(), input.UserRoles[i].User)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		roleObj, err := RoleManager.FetchByIdOrName(userCred, input.UserRoles[i].Role)
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
		groupObj, err := GroupManager.FetchByIdOrName(userCred, input.GroupRoles[i].Group)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(GroupManager.Keyword(), input.GroupRoles[i].Group)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		roleObj, err := RoleManager.FetchByIdOrName(userCred, input.GroupRoles[i].Role)
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
	if manager.NamespaceScope() == rbacutils.ScopeDomain {
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
