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
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/sqlchemy"

	"github.com/pkg/errors"
	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SAssignmentManager struct {
	db.SResourceBaseManager
}

var AssignmentManager *SAssignmentManager

func init() {
	AssignmentManager = &SAssignmentManager{
		SResourceBaseManager: db.NewResourceBaseManager(
			SAssignment{},
			"assignment",
			"assignment",
			"assignments",
		),
	}
}

/*
+-----------+---------------------------------------------------------------+------+-----+---------+-------+
| Field     | Type                                                          | Null | Key | Default | Extra |
+-----------+---------------------------------------------------------------+------+-----+---------+-------+
| type      | enum('UserProject','GroupProject','UserDomain','GroupDomain') | NO   | PRI | NULL    |       |
| actor_id  | varchar(64)                                                   | NO   | PRI | NULL    |       |
| target_id | varchar(64)                                                   | NO   | PRI | NULL    |       |
| role_id   | varchar(64)                                                   | NO   | PRI | NULL    |       |
| inherited | tinyint(1)                                                    | NO   | PRI | NULL    |       |
+-----------+---------------------------------------------------------------+------+-----+---------+-------+
*/

type SAssignment struct {
	db.SResourceBase

	Type     string `width:"16" charset:"ascii" nullable:"false" primary:"true" list:"admin"`
	ActorId  string `width:"64" charset:"ascii" nullable:"false" primary:"true" list:"admin"`
	TargetId string `width:"64" charset:"ascii" nullable:"false" primary:"true" list:"admin"`
	RoleId   string `width:"64" charset:"ascii" nullable:"false" primary:"true" list:"admin"`

	Inherited tristate.TriState `nullable:"false" primary:"true" list:"admin"`
}

func (manager *SAssignmentManager) InitializeData() error {
	return manager.initSysAssignment()
}

func (manager *SAssignmentManager) initSysAssignment() error {
	adminUser, err := UserManager.FetchUserExtended("", options.Options.AdminUserName, options.Options.AdminUserDomainId, "")
	if err != nil {
		return errors.WithMessage(err, "FetchUserExtended")
	}
	adminProject, err := ProjectManager.FetchProjectByName(options.Options.AdminProjectName, options.Options.AdminProjectDomainId, "")
	if err != nil {
		return errors.WithMessage(err, "FetchProjectByName")
	}
	adminRole, err := RoleManager.FetchRoleByName(options.Options.AdminRoleName, options.Options.AdminRoleDomainId, "")
	if err != nil {
		return errors.WithMessage(err, "FetchRoleByName")
	}

	q := manager.Query().Equals("type", api.AssignmentUserProject)
	q = q.Equals("actor_id", adminUser.Id)
	q = q.Equals("target_id", adminProject.Id)
	q = q.Equals("role_id", adminRole.Id)
	q = q.IsFalse("inherited")

	assign := SAssignment{}
	assign.SetModelManager(manager)

	err = q.First(&assign)
	if err != nil && err != sql.ErrNoRows {
		return errors.WithMessage(err, "query")
	}
	if err == nil {
		return nil
	}
	// no data
	assign.Type = api.AssignmentUserProject
	assign.ActorId = adminUser.Id
	assign.TargetId = adminProject.Id
	assign.RoleId = adminRole.Id
	assign.Inherited = tristate.False

	err = manager.TableSpec().Insert(&assign)
	if err != nil {
		return errors.WithMessage(err, "insert")
	}

	return nil
}

func (manager *SAssignmentManager) FetchUserProjectRoles(userId, projId string) ([]SRole, error) {
	subq := manager.fetchUserProjectRoleIdsQuery(userId, projId)
	q := RoleManager.Query().In("id", subq.SubQuery())

	roles := make([]SRole, 0)
	err := db.FetchModelObjects(RoleManager, q, &roles)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	return roles, nil
}

func (manager *SAssignmentManager) fetchRoleUserIdsQuery(roleId string) *sqlchemy.SQuery {
	q := manager.Query("actor_id").Equals("role_id", roleId).Equals("type", api.AssignmentUserProject).Distinct().SubQuery()
	return q.Query()
}

func (manager *SAssignmentManager) fetchRoleGroupIdsQuery(roleId string) *sqlchemy.SQuery {
	q := manager.Query("actor_id").Equals("role_id", roleId).Equals("type", api.AssignmentGroupProject).Distinct().SubQuery()
	return q.Query()
}

func (manager *SAssignmentManager) fetchRoleProjectIdsQuery(roleId string) *sqlchemy.SQuery {
	q := manager.Query("target_id").Equals("role_id", roleId).Distinct().SubQuery()
	return q.Query()
}

func (manager *SAssignmentManager) fetchUserProjectRoleIdsQuery(userId, projId string) *sqlchemy.SQuery {
	subq := AssignmentManager.Query("role_id")
	subq = subq.Equals("type", api.AssignmentUserProject)
	subq = subq.Equals("actor_id", userId)
	subq = subq.Equals("target_id", projId)
	subq = subq.IsFalse("inherited")

	assigns := AssignmentManager.Query().SubQuery()
	usergroups := UsergroupManager.Query().SubQuery()

	subq2 := assigns.Query(assigns.Field("role_id"))
	subq2 = subq2.Join(usergroups, sqlchemy.Equals(
		usergroups.Field("group_id"), assigns.Field("actor_id"),
	))
	subq2 = subq2.Filter(sqlchemy.Equals(assigns.Field("type"), api.AssignmentGroupProject))
	subq2 = subq2.Filter(sqlchemy.Equals(assigns.Field("target_id"), projId))
	subq2 = subq2.Filter(sqlchemy.Equals(usergroups.Field("user_id"), userId))
	subq2 = subq2.Filter(sqlchemy.IsFalse(assigns.Field("inherited")))

	return sqlchemy.Union(subq, subq2).Query()
}

func (manager *SAssignmentManager) fetchGroupProjectRoleIdsQuery(groupId, projId string) *sqlchemy.SQuery {
	subq := AssignmentManager.Query("role_id")
	subq = subq.Equals("type", api.AssignmentGroupProject)
	subq = subq.Equals("actor_id", groupId)
	subq = subq.Equals("target_id", projId)
	subq = subq.IsFalse("inherited")
	return subq
}

func (manager *SAssignmentManager) fetchGroupProjectIdsQuery(groupId string) *sqlchemy.SQuery {
	q := manager.Query("target_id")
	q = q.Equals("type", api.AssignmentGroupProject)
	q = q.Equals("actor_id", groupId)
	q = q.IsFalse("inherited")
	return q
}

func (manager *SAssignmentManager) fetchProjectGroupIdsQuery(projId string) *sqlchemy.SQuery {
	q := manager.Query("actor_id")
	q = q.Equals("type", api.AssignmentGroupProject)
	q = q.Equals("target_id", projId)
	q = q.IsFalse("inherited")
	return q
}

func (manager *SAssignmentManager) fetchUserProjectIdsQuery(userId string) *sqlchemy.SQuery {
	q1 := manager.Query("target_id")
	q1 = q1.Equals("type", api.AssignmentUserProject)
	q1 = q1.Equals("actor_id", userId)
	q1 = q1.IsFalse("inherited")

	assigns := AssignmentManager.Query().SubQuery()
	usergroups := UsergroupManager.Query().SubQuery()

	q2 := assigns.Query(assigns.Field("target_id"))
	q2 = q2.Join(usergroups, sqlchemy.Equals(
		usergroups.Field("group_id"), assigns.Field("actor_id"),
	))
	q2 = q2.Filter(sqlchemy.Equals(assigns.Field("type"), api.AssignmentGroupProject))
	q2 = q2.Filter(sqlchemy.Equals(usergroups.Field("user_id"), userId))
	q2 = q2.Filter(sqlchemy.IsFalse(assigns.Field("inherited")))

	union := sqlchemy.Union(q1, q2)
	return union.Query()
}

func (manager *SAssignmentManager) fetchProjectUserIdsQuery(projId string) *sqlchemy.SQuery {
	q1 := manager.Query("actor_id")
	q1 = q1.Equals("type", api.AssignmentUserProject)
	q1 = q1.Equals("target_id", projId)
	q1 = q1.IsFalse("inherited")

	assigns := AssignmentManager.Query().SubQuery()
	usergroups := UsergroupManager.Query().SubQuery()

	q2 := usergroups.Query(usergroups.Field("user_id", "actor_id"))
	q2 = q2.Join(assigns, sqlchemy.Equals(
		usergroups.Field("group_id"), assigns.Field("actor_id"),
	))
	q2 = q2.Filter(sqlchemy.Equals(assigns.Field("type"), api.AssignmentGroupProject))
	q2 = q2.Filter(sqlchemy.Equals(assigns.Field("target_id"), projId))
	q2 = q2.Filter(sqlchemy.IsFalse(assigns.Field("inherited")))

	union := sqlchemy.Union(q1, q2)
	return union.Query()
}

func (manager *SAssignmentManager) projectAddUser(ctx context.Context, userCred mcclient.TokenCredential, project *SProject, user *SUser, role *SRole) error {
	err := manager.add(api.AssignmentUserProject, user.Id, project.Id, role.Id)
	if err == nil {
		db.OpsLog.LogEvent(user, db.ACT_ATTACH, project.GetShortDesc(ctx), userCred)
		db.OpsLog.LogEvent(project, db.ACT_ATTACH, user.GetShortDesc(ctx), userCred)
	}
	return err
}

func (manager *SAssignmentManager) projectRemoveUser(ctx context.Context, userCred mcclient.TokenCredential, project *SProject, user *SUser, role *SRole) error {
	err := manager.remove(api.AssignmentUserProject, user.Id, project.Id, role.Id)
	if err == nil {
		db.OpsLog.LogEvent(user, db.ACT_DETACH, project.GetShortDesc(ctx), userCred)
		db.OpsLog.LogEvent(project, db.ACT_DETACH, user.GetShortDesc(ctx), userCred)
	}
	return err
}

func (manager *SAssignmentManager) projectAddGroup(ctx context.Context, userCred mcclient.TokenCredential, project *SProject, group *SGroup, role *SRole) error {
	err := manager.add(api.AssignmentGroupProject, group.Id, project.Id, role.Id)
	if err == nil {
		db.OpsLog.LogEvent(group, db.ACT_ATTACH, project.GetShortDesc(ctx), userCred)
		db.OpsLog.LogEvent(project, db.ACT_ATTACH, group.GetShortDesc(ctx), userCred)
	}
	return err
}

func (manager *SAssignmentManager) projectRemoveGroup(ctx context.Context, userCred mcclient.TokenCredential, project *SProject, group *SGroup, role *SRole) error {
	err := manager.remove(api.AssignmentGroupProject, group.Id, project.Id, role.Id)
	if err == nil {
		db.OpsLog.LogEvent(group, db.ACT_DETACH, project.GetShortDesc(ctx), userCred)
		db.OpsLog.LogEvent(project, db.ACT_DETACH, group.GetShortDesc(ctx), userCred)
	}
	return err
}

func (manager *SAssignmentManager) remove(typeStr, actorId, projectId, roleId string) error {
	assign := SAssignment{
		Type:      typeStr,
		ActorId:   actorId,
		TargetId:  projectId,
		RoleId:    roleId,
		Inherited: tristate.False,
	}
	assign.SetModelManager(manager)
	_, err := db.Update(&assign, func() error {
		return assign.MarkDelete()
	})
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	return nil
}

func (manager *SAssignmentManager) add(typeStr, actorId, projectId, roleId string) error {
	assign := SAssignment{
		Type:      typeStr,
		ActorId:   actorId,
		TargetId:  projectId,
		RoleId:    roleId,
		Inherited: tristate.False,
	}
	assign.SetModelManager(manager)
	return manager.TableSpec().InsertOrUpdate(&assign)
}

func AddAdhocHandlers(version string, app *appsrv.Application) {
	app.AddHandler2("GET", fmt.Sprintf("%s/role_assignments", version), roleAssignmentHandler, nil, "list_role_assignments", nil)
}

func roleAssignmentHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, query, _ := appsrv.FetchEnv(ctx, w, r)
	userId, _ := query.GetString("user", "id")
	groupId, _ := query.GetString("group", "id")
	roleId, _ := query.GetString("role", "id")
	domainId, _ := query.GetString("scope", "domain", "id")
	projectId, _ := query.GetString("scope", "project", "id")
	includeNames := query.Contains("include_names")
	effective := query.Contains("effective")
	includeSub := query.Contains("include_subtree")

	results, err := AssignmentManager.FetchAll(userId, groupId, roleId, domainId, projectId, includeNames, effective, includeSub)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	body := jsonutils.NewDict()
	body.Add(jsonutils.Marshal(results), "role_assignments")
	appsrv.SendJSON(w, body)
}

func (manager *SAssignmentManager) queryAll(userId, groupId, roleId, domainId, projectId string) *sqlchemy.SQuery {
	q := manager.Query("type", "actor_id", "target_id", "role_id")
	if len(userId) > 0 {
		q = q.In("type", []string{api.AssignmentUserProject, api.AssignmentUserDomain}).Equals("actor_id", userId)
	}
	if len(groupId) > 0 {
		q = q.In("type", []string{api.AssignmentGroupProject, api.AssignmentGroupDomain}).Equals("actor_id", groupId)
	}
	if len(roleId) > 0 {
		q = q.Equals("role_id", roleId)
	}
	if len(projectId) > 0 {
		q = q.Equals("target_id", projectId).In("type", []string{api.AssignmentUserProject, api.AssignmentGroupProject})
	}
	if len(domainId) > 0 {
		q = q.Equals("target_id", domainId).In("type", []string{api.AssignmentUserDomain, api.AssignmentGroupDomain})
	}
	return q
}

type SIdentityObject struct {
	Id   string
	Name string
}

type SDomainObject struct {
	SIdentityObject
	Domain SIdentityObject
}

type SFetchDomainObject struct {
	SIdentityObject
	Domain   string
	DomainId string
}

type SRoleAssignment struct {
	Scope struct {
		Domain  SIdentityObject
		Project SDomainObject
	}
	User  SDomainObject
	Group SDomainObject
	Role  SDomainObject
}

func (assign *SAssignment) getRoleAssignment(domains, projects, groups, users, roles map[string]SFetchDomainObject) SRoleAssignment {
	ra := SRoleAssignment{}
	switch assign.Type {
	case api.AssignmentUserDomain:
		ra.Scope.Domain.Id = assign.TargetId
		ra.Scope.Domain.Name = domains[assign.TargetId].Name
		ra.User.Id = assign.ActorId
		ra.User.Name = users[assign.ActorId].Name
		ra.User.Domain.Id = users[assign.ActorId].DomainId
		ra.User.Domain.Name = users[assign.ActorId].Domain
	case api.AssignmentUserProject:
		ra.Scope.Project.Id = assign.TargetId
		ra.Scope.Project.Name = projects[assign.TargetId].Name
		ra.Scope.Project.Domain.Id = projects[assign.TargetId].DomainId
		ra.Scope.Project.Domain.Name = projects[assign.TargetId].Domain
		ra.User.Id = assign.ActorId
		ra.User.Name = users[assign.ActorId].Name
		ra.User.Domain.Id = users[assign.ActorId].DomainId
		ra.User.Domain.Name = users[assign.ActorId].Domain
	case api.AssignmentGroupDomain:
		ra.Scope.Domain.Id = assign.TargetId
		ra.Scope.Domain.Name = domains[assign.TargetId].Name
		ra.Group.Id = assign.ActorId
		ra.Group.Name = groups[assign.ActorId].Name
		ra.Group.Domain.Id = groups[assign.ActorId].DomainId
		ra.Group.Domain.Name = groups[assign.ActorId].Domain
	case api.AssignmentGroupProject:
		ra.Scope.Project.Id = assign.TargetId
		ra.Scope.Project.Name = projects[assign.TargetId].Name
		ra.Scope.Project.Domain.Id = projects[assign.TargetId].DomainId
		ra.Scope.Project.Domain.Name = projects[assign.TargetId].Domain
		ra.Group.Id = assign.ActorId
		ra.Group.Name = groups[assign.ActorId].Name
		ra.Group.Domain.Id = groups[assign.ActorId].DomainId
		ra.Group.Domain.Name = groups[assign.ActorId].Domain
	}
	ra.Role.Id = assign.RoleId
	ra.Role.Name = roles[assign.RoleId].Name
	ra.Role.Domain.Id = roles[assign.RoleId].DomainId
	ra.Role.Domain.Name = roles[assign.RoleId].Domain
	return ra
}

func (manager *SAssignmentManager) FetchAll(userId, groupId, roleId, domainId, projectId string, includeNames, effective, includeSub bool) ([]SRoleAssignment, error) {
	var q *sqlchemy.SQuery
	if effective {
		usrq := manager.queryAll(userId, groupId, roleId, domainId, projectId).In("type", []string{api.AssignmentUserProject, api.AssignmentUserDomain})

		grpq := manager.queryAll(userId, groupId, roleId, domainId, projectId).In("type", []string{api.AssignmentUserProject, api.AssignmentUserDomain}).SubQuery()

		memberships := UsergroupManager.Query("user_id", "group_id").SubQuery()

		q2 := grpq.Query(grpq.Field("type"), memberships.Field("user_id", "actor_id"), grpq.Field("target_id"), grpq.Field("role_id"))
		q2 = q2.Join(memberships, sqlchemy.Equals(grpq.Field("actor_id"), memberships.Field("group_id")))

		q = sqlchemy.Union(usrq, q2).Query().Distinct()
	} else {
		q = manager.queryAll(userId, groupId, roleId, domainId, projectId).Distinct()
	}
	assigns := make([]SAssignment, 0)
	err := q.All(&assigns)
	if err != nil && err != sql.ErrNoRows {
		return nil, httperrors.NewInternalServerError("query error %s", err)
	}

	domainIds := stringutils2.SSortedStrings{}
	projectIds := stringutils2.SSortedStrings{}
	groupIds := stringutils2.SSortedStrings{}
	userIds := stringutils2.SSortedStrings{}
	roleIds := stringutils2.SSortedStrings{}

	for i := range assigns {
		switch assigns[i].Type {
		case api.AssignmentGroupProject:
			projectIds = stringutils2.Append(projectIds, assigns[i].TargetId)
			groupIds = stringutils2.Append(groupIds, assigns[i].ActorId)
		case api.AssignmentGroupDomain:
			domainIds = stringutils2.Append(domainIds, assigns[i].TargetId)
			groupIds = stringutils2.Append(groupIds, assigns[i].ActorId)
		case api.AssignmentUserProject:
			projectIds = stringutils2.Append(projectIds, assigns[i].TargetId)
			userIds = stringutils2.Append(userIds, assigns[i].ActorId)
		case api.AssignmentUserDomain:
			domainIds = stringutils2.Append(domainIds, assigns[i].TargetId)
			userIds = stringutils2.Append(userIds, assigns[i].ActorId)
		}
		roleIds = stringutils2.Append(roleIds, assigns[i].RoleId)
	}

	domains, err := fetchObjects(DomainManager, domainIds)
	if err != nil {
		return nil, errors.WithMessage(err, "fetchObjects DomainManager")
	}
	projects, err := fetchObjects(ProjectManager, projectIds)
	if err != nil {
		return nil, errors.WithMessage(err, "fetchObjects ProjectManager")
	}
	groups, err := fetchObjects(GroupManager, groupIds)
	if err != nil {
		return nil, errors.WithMessage(err, "fetchObjects GroupManager")
	}
	users, err := fetchObjects(UserManager, userIds)
	if err != nil {
		return nil, errors.WithMessage(err, "fetchObjects UserManager")
	}
	roles, err := fetchObjects(RoleManager, roleIds)
	if err != nil {
		return nil, errors.WithMessage(err, "fetchObjects RoleManager")
	}

	results := make([]SRoleAssignment, len(assigns))
	for i := range assigns {
		results[i] = assigns[i].getRoleAssignment(domains, projects, groups, users, roles)
	}
	return results, nil
}

func fetchObjects(manager db.IModelManager, idList []string) (map[string]SFetchDomainObject, error) {
	results := make(map[string]SFetchDomainObject)
	if len(idList) == 0 {
		return results, nil
	}
	var q *sqlchemy.SQuery
	if manager == DomainManager {
		q = DomainManager.Query().In("id", idList)
	} else {
		resq := manager.Query().SubQuery()
		domains := DomainManager.Query().SubQuery()

		q = resq.Query(resq.Field("id"), resq.Field("name"), resq.Field("domain_id"), domains.Field("name", "domain"))
		q = q.Join(domains, sqlchemy.Equals(domains.Field("id"), resq.Field("domain_id")))
		q = q.Filter(sqlchemy.IsTrue(domains.Field("is_domain")))
		q = q.Filter(sqlchemy.In(resq.Field("id"), idList))
	}
	objs := make([]SFetchDomainObject, 0)
	err := q.All(&objs)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.WithMessage(err, "query")
	}
	for i := range objs {
		results[objs[i].Id] = objs[i]
	}
	return results, nil
}
