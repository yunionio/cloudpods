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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-ignore
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
	AssignmentManager.SetVirtualObject(AssignmentManager)
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

	// 关联类型，分为四类：'UserProject','GroupProject','UserDomain','GroupDomain'
	Type string `width:"16" charset:"ascii" nullable:"false" primary:"true" list:"admin"`
	// 用户或者用户组ID
	ActorId string `width:"64" charset:"ascii" nullable:"false" primary:"true" list:"admin"`
	// 项目或者域ID
	TargetId string `width:"64" charset:"ascii" nullable:"false" primary:"true" list:"admin"`
	// 角色ID
	RoleId string `width:"64" charset:"ascii" nullable:"false" primary:"true" list:"admin"`

	Inherited tristate.TriState `primary:"true" list:"admin"`
}

func (manager *SAssignmentManager) InitializeData() error {
	return manager.initSysAssignment(context.TODO())
}

func (manager *SAssignmentManager) initSysAssignment(ctx context.Context) error {
	adminUser, err := UserManager.FetchUserExtended("", api.SystemAdminUser, api.DEFAULT_DOMAIN_ID, "")
	if err != nil {
		return errors.Wrap(err, "FetchUserExtended")
	}
	adminProject, err := ProjectManager.FetchProjectByName(api.SystemAdminProject, api.DEFAULT_DOMAIN_ID, "")
	if err != nil {
		return errors.Wrap(err, "FetchProjectByName")
	}
	adminRole, err := RoleManager.FetchRoleByName(api.SystemAdminRole, api.DEFAULT_DOMAIN_ID, "")
	if err != nil {
		return errors.Wrap(err, "FetchRoleByName")
	}

	q := manager.Query().Equals("type", api.AssignmentUserProject)
	q = q.Equals("actor_id", adminUser.Id)
	q = q.Equals("target_id", adminProject.Id)
	q = q.Equals("role_id", adminRole.Id)
	q = q.IsFalse("inherited")

	assign := SAssignment{}
	assign.SetModelManager(manager, &assign)

	err = q.First(&assign)
	if err != nil && err != sql.ErrNoRows {
		return errors.Wrap(err, "query")
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

	err = manager.TableSpec().Insert(ctx, &assign)
	if err != nil {
		return errors.Wrap(err, "insert")
	}

	return nil
}

func (manager *SAssignmentManager) fetchUserProjectRoleCount(userId, projId string) (int, error) {
	q := manager.fetchUserProjectRoleIdsQuery(userId, projId)
	return q.CountWithError()
}

func (manager *SAssignmentManager) fetchGroupProjectRoleCount(grpId, projId string) (int, error) {
	q := manager.fetchGroupProjectRoleIdsQuery(grpId, projId)
	return q.CountWithError()
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

	return sqlchemy.Union(subq, subq2).Query().Distinct()
}

func (manager *SAssignmentManager) fetchGroupProjectRoleIdsQuery(groupId, projId string) *sqlchemy.SQuery {
	subq := AssignmentManager.Query("role_id")
	subq = subq.Equals("type", api.AssignmentGroupProject)
	subq = subq.Equals("actor_id", groupId)
	subq = subq.Equals("target_id", projId)
	subq = subq.IsFalse("inherited")
	return subq.Distinct()
}

func (manager *SAssignmentManager) fetchGroupProjectIdsQuery(groupId string) *sqlchemy.SQuery {
	q := manager.Query("target_id")
	q = q.Equals("type", api.AssignmentGroupProject)
	q = q.Equals("actor_id", groupId)
	q = q.IsFalse("inherited")
	return q.Distinct()
}

func (manager *SAssignmentManager) fetchProjectGroupIdsQuery(projId string) *sqlchemy.SQuery {
	q := manager.Query("actor_id")
	q = q.Equals("type", api.AssignmentGroupProject)
	q = q.Equals("target_id", projId)
	q = q.IsFalse("inherited")
	return q.Distinct()
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
	return union.Query().Distinct()
}

func (manager *SAssignmentManager) fetchProjectUserIdsQuery(projId string) *sqlchemy.SQuery {
	return manager.fetchProjectRoleUserIdsQuery(projId, "")
}

func (manager *SAssignmentManager) fetchProjectRoleUserIdsQuery(projId, roleId string) *sqlchemy.SQuery {
	q1 := manager.Query("actor_id")
	q1 = q1.Equals("type", api.AssignmentUserProject)
	q1 = q1.Equals("target_id", projId)
	q1 = q1.IsFalse("inherited")
	if len(roleId) > 0 {
		q1 = q1.Equals("role_id", roleId)
	}

	assigns := AssignmentManager.Query().SubQuery()
	usergroups := UsergroupManager.Query().SubQuery()

	q2 := usergroups.Query(usergroups.Field("user_id", "actor_id"))
	q2 = q2.Join(assigns, sqlchemy.Equals(
		usergroups.Field("group_id"), assigns.Field("actor_id"),
	))
	q2 = q2.Filter(sqlchemy.Equals(assigns.Field("type"), api.AssignmentGroupProject))
	q2 = q2.Filter(sqlchemy.Equals(assigns.Field("target_id"), projId))
	q2 = q2.Filter(sqlchemy.IsFalse(assigns.Field("inherited")))
	if len(roleId) > 0 {
		q2 = q2.Equals("role_id", roleId)
	}

	union := sqlchemy.Union(q1, q2)
	return union.Query().Distinct()
}

func (manager *SAssignmentManager) fetchUserAndGroups(projIds []string) (map[string][]string, map[string][]string, error) {
	q1 := manager.Query().In("type", []string{api.AssignmentGroupProject, api.AssignmentUserProject}).IsFalse("inherited").In("target_id", projIds)
	groupCnt, userCnt := map[string][]string{}, map[string][]string{}
	assignments := []SAssignment{}
	err := q1.All(&assignments)
	if err != nil {
		return groupCnt, userCnt, errors.Wrapf(err, "q1.All")
	}
	for i := range assignments {
		switch assignments[i].Type {
		case api.AssignmentGroupProject:
			_, ok := groupCnt[assignments[i].TargetId]
			if !ok {
				groupCnt[assignments[i].TargetId] = []string{}
			}
			if !utils.IsInStringArray(assignments[i].ActorId, groupCnt[assignments[i].TargetId]) {
				groupCnt[assignments[i].TargetId] = append(groupCnt[assignments[i].TargetId], assignments[i].ActorId)
			}
		case api.AssignmentUserProject:
			_, ok := userCnt[assignments[i].TargetId]
			if !ok {
				userCnt[assignments[i].TargetId] = []string{}
			}
			if !utils.IsInStringArray(assignments[i].ActorId, userCnt[assignments[i].TargetId]) {
				userCnt[assignments[i].TargetId] = append(userCnt[assignments[i].TargetId], assignments[i].ActorId)
			}
		}
	}

	assigns := AssignmentManager.Query().SubQuery()
	usergroups := UsergroupManager.Query().SubQuery()

	q2 := usergroups.Query(usergroups.Field("user_id", "actor_id"))
	q2 = q2.Join(assigns, sqlchemy.Equals(
		usergroups.Field("group_id"), assigns.Field("actor_id"),
	))
	q2 = q2.Filter(sqlchemy.Equals(assigns.Field("type"), api.AssignmentGroupProject))
	q2 = q2.Filter(sqlchemy.In(assigns.Field("target_id"), projIds))
	q2 = q2.Filter(sqlchemy.IsFalse(assigns.Field("inherited")))

	err = q2.All(&assignments)
	if err != nil {
		return groupCnt, userCnt, errors.Wrapf(err, "q2.All")
	}
	for i := range assignments {
		_, ok := userCnt[assignments[i].TargetId]
		if !ok {
			userCnt[assignments[i].TargetId] = []string{}
		}
		if !utils.IsInStringArray(assignments[i].ActorId, userCnt[assignments[i].TargetId]) {
			userCnt[assignments[i].TargetId] = append(userCnt[assignments[i].TargetId], assignments[i].ActorId)
		}
	}
	return groupCnt, userCnt, nil
}

func (manager *SAssignmentManager) ProjectAddUser(ctx context.Context, userCred mcclient.TokenCredential, project *SProject, user *SUser, role *SRole) error {
	err := db.ValidateCreateDomainId(project.DomainId)
	if err != nil {
		return err
	}
	if project.DomainId != user.DomainId {
		// if project.DomainId != api.DEFAULT_DOMAIN_ID && !options.Options.AllowJoinProjectsAcrossDomains {
		//	return httperrors.NewInputParameterError("join user into project of default domain or identical domain")
		// } else
		if !db.IsAllowPerform(ctx, rbacscope.ScopeSystem, userCred, user, "join-project") {
			return httperrors.NewForbiddenError("not enough privilege")
		}
	} else {
		if !db.IsAllowPerform(ctx, rbacscope.ScopeDomain, userCred, user, "join-project") {
			return httperrors.NewForbiddenError("not enough privilege")
		}
	}
	roleCnt, err := manager.fetchUserProjectRoleCount(user.Id, project.Id)
	if err != nil {
		return errors.Wrap(err, "FetchUserProjectRoleCount")
	}
	if roleCnt >= options.Options.MaxUserRolesInProject {
		return errors.Wrapf(httperrors.ErrTooLarge, "user %s has joined project %s %d roles more than %d", user.Name, project.Name, roleCnt, options.Options.MaxUserRolesInProject)
	}
	err = manager.add(ctx, api.AssignmentUserProject, user.Id, project.Id, role.Id)
	if err != nil {
		return errors.Wrap(err, "manager.add")
	}
	db.OpsLog.LogEvent(user, db.ACT_ATTACH, project.GetShortDesc(ctx), userCred)
	db.OpsLog.LogEvent(project, db.ACT_ATTACH, user.GetShortDesc(ctx), userCred)
	if len(project.AdminId) == 0 && role.Name == options.Options.ProjectAdminRole {
		err := project.resetAdminUser(ctx, userCred)
		if err != nil {
			log.Errorf("rsetAdminUser fail: %s", err)
		}
	}
	return nil
}

func (assign *SAssignment) getRole() (*SRole, error) {
	return RoleManager.FetchRoleById(assign.RoleId)
}

func (assign *SAssignment) getProject() (*SProject, error) {
	if assign.Type == api.AssignmentUserProject || assign.Type == api.AssignmentGroupProject {
		return ProjectManager.FetchProjectById(assign.TargetId)
	}
	return nil, nil
}

func (assign *SAssignment) getDomain() (*SDomain, error) {
	if assign.Type == api.AssignmentUserDomain || assign.Type == api.AssignmentGroupDomain {
		return DomainManager.FetchDomainById(assign.TargetId)
	}
	return nil, nil
}

func (manager *SAssignmentManager) batchRemove(ctx context.Context, userCred mcclient.TokenCredential, actorId string, typeStrs []string) error {
	q := manager.Query()
	q = q.In("type", typeStrs)
	q = q.Equals("actor_id", actorId)
	q = q.IsFalse("inherited")
	assigns := make([]SAssignment, 0)
	err := db.FetchModelObjects(manager, q, &assigns)
	if err != nil && err != sql.ErrNoRows {
		return errors.Wrap(err, "db.FetchModelObjects")
	}
	for i := range assigns {
		_, err := db.Update(&assigns[i], func() error {
			assigns[i].MarkDelete()
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "db.Update")
		}
		// clear project admin Id
		role, _ := assigns[i].getRole()
		if role.Name == options.Options.ProjectAdminRole {
			project, _ := assigns[i].getProject()
			if project != nil && project.AdminId == actorId {
				err := project.resetAdminUser(ctx, userCred)
				if err != nil {
					log.Errorf("batchRemove project resetAdminUser fail %s", err)
				}
			}
		}
	}
	return nil
}

func (manager *SAssignmentManager) projectRemoveAllUser(ctx context.Context, userCred mcclient.TokenCredential, user *SUser) error {
	if user.IsAdminUser() {
		return httperrors.NewForbiddenError("sysadmin is protected")
	}
	// allow remove current user from current project. user takes the consequence
	// if user.Id == userCred.GetUserId() {
	// 	return httperrors.NewForbiddenError("cannot remove current user from current project")
	// }
	err := manager.batchRemove(ctx, userCred, user.Id, []string{api.AssignmentUserProject, api.AssignmentUserDomain})
	if err != nil {
		return errors.Wrap(err, "manager.batchRemove")
	}
	db.OpsLog.LogEvent(user, "leave_all_projects", user.GetShortDesc(ctx), userCred)
	return nil
}

func (manager *SAssignmentManager) projectRemoveAllGroup(ctx context.Context, userCred mcclient.TokenCredential, group *SGroup) error {
	err := manager.batchRemove(ctx, userCred, group.Id, []string{api.AssignmentGroupProject, api.AssignmentGroupDomain})
	if err != nil {
		return errors.Wrap(err, "manager.batchRemove")
	}
	db.OpsLog.LogEvent(group, "leave_all_projects", group.GetShortDesc(ctx), userCred)
	return nil
}

func (manager *SAssignmentManager) projectRemoveUser(ctx context.Context, userCred mcclient.TokenCredential, project *SProject, user *SUser, role *SRole) error {
	if project.IsAdminProject() && user.IsAdminUser() && role.IsSystemRole() {
		return httperrors.NewForbiddenError("sysadmin is protected")
	}
	// allow remove current user from current project, user takes the consequence
	// prevent remove current user from current project
	// if project.Id == userCred.GetProjectId() && user.Id == userCred.GetUserId() {
	//	return httperrors.NewForbiddenError("cannot remove current user from current project")
	// }
	if project.DomainId != user.DomainId {
		// if project.DomainId != api.DEFAULT_DOMAIN_ID {
		//    return httperrors.NewInputParameterError("join user into project of default domain or identical domain")
		// } else
		if !db.IsAllowPerform(ctx, rbacscope.ScopeSystem, userCred, user, "leave-project") {
			return httperrors.NewForbiddenError("not enough privilege")
		}
	} else {
		if !db.IsAllowPerform(ctx, rbacscope.ScopeDomain, userCred, user, "leave-project") {
			return httperrors.NewForbiddenError("not enough privilege")
		}
	}
	err := manager.remove(api.AssignmentUserProject, user.Id, project.Id, role.Id)
	if err != nil {
		return errors.Wrap(err, "manager.remove")
	}
	db.OpsLog.LogEvent(user, db.ACT_DETACH, project.GetShortDesc(ctx), userCred)
	db.OpsLog.LogEvent(project, db.ACT_DETACH, user.GetShortDesc(ctx), userCred)
	if project.AdminId == user.Id && role.Name == options.Options.ProjectAdminRole {
		err := project.resetAdminUser(ctx, userCred)
		if err != nil {
			log.Errorf("resetAdminUser fail %s", err)
		}
	}
	return nil
}

func (manager *SAssignmentManager) projectAddGroup(ctx context.Context, userCred mcclient.TokenCredential, project *SProject, group *SGroup, role *SRole) error {
	err := db.ValidateCreateDomainId(project.DomainId)
	if err != nil {
		return err
	}
	if project.DomainId != group.DomainId {
		// if project.DomainId != api.DEFAULT_DOMAIN_ID && !options.Options.AllowJoinProjectsAcrossDomains {
		// 	return httperrors.NewInputParameterError("join group into project of default domain or identical domain")
		// } else
		if !db.IsAllowPerform(ctx, rbacscope.ScopeSystem, userCred, group, "join-project") {
			return httperrors.NewForbiddenError("not enough privilege")
		}
	} else {
		if !db.IsAllowPerform(ctx, rbacscope.ScopeDomain, userCred, group, "join-project") {
			return httperrors.NewForbiddenError("not enough privilege")
		}
	}
	roleCnt, err := manager.fetchGroupProjectRoleCount(group.Id, project.Id)
	if err != nil {
		return errors.Wrap(err, "fetchGroupProjectRoleCount")
	}
	if roleCnt >= options.Options.MaxGroupRolesInProject {
		return errors.Wrapf(httperrors.ErrTooLarge, "group %s has joined project %s %d roles more than %d", group.Name, project.Name, roleCnt, options.Options.MaxGroupRolesInProject)
	}
	err = manager.add(ctx, api.AssignmentGroupProject, group.Id, project.Id, role.Id)
	if err != nil {
		return errors.Wrap(err, "manager.add")
	}
	db.OpsLog.LogEvent(group, db.ACT_ATTACH, project.GetShortDesc(ctx), userCred)
	db.OpsLog.LogEvent(project, db.ACT_ATTACH, group.GetShortDesc(ctx), userCred)
	if len(project.AdminId) == 0 && role.Name == options.Options.ProjectAdminRole {
		err := project.resetAdminUser(ctx, userCred)
		if err != nil {
			log.Errorf("rsetAdminUser fail: %s", err)
		}
	}
	return nil
}

func (manager *SAssignmentManager) projectRemoveGroup(ctx context.Context, userCred mcclient.TokenCredential, project *SProject, group *SGroup, role *SRole) error {
	if project.DomainId != group.DomainId {
		// if project.DomainId != api.DEFAULT_DOMAIN_ID {
		//    return httperrors.NewInputParameterError("join group into project of default domain or identical domain")
		// } else
		if !db.IsAllowPerform(ctx, rbacscope.ScopeSystem, userCred, group, "leave-project") {
			return httperrors.NewForbiddenError("not enough privilege")
		}
	} else {
		if !db.IsAllowPerform(ctx, rbacscope.ScopeDomain, userCred, group, "leave-project") {
			return httperrors.NewForbiddenError("not enough privilege")
		}
	}
	err := manager.remove(api.AssignmentGroupProject, group.Id, project.Id, role.Id)
	if err != nil {
		return errors.Wrap(err, "manager.remove")
	}
	db.OpsLog.LogEvent(group, db.ACT_DETACH, project.GetShortDesc(ctx), userCred)
	db.OpsLog.LogEvent(project, db.ACT_DETACH, group.GetShortDesc(ctx), userCred)
	if len(project.AdminId) > 0 && role.Name == options.Options.ProjectAdminRole {
		err := project.resetAdminUser(ctx, userCred)
		if err != nil {
			log.Errorf("rsetAdminUser fail: %s", err)
		}
	}
	return nil
}

func (manager *SAssignmentManager) remove(typeStr, actorId, projectId, roleId string) error {
	assign := SAssignment{
		Type:      typeStr,
		ActorId:   actorId,
		TargetId:  projectId,
		RoleId:    roleId,
		Inherited: tristate.False,
	}
	assign.SetModelManager(manager, &assign)
	_, err := db.Update(&assign, func() error {
		return assign.MarkDelete()
	})
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	return nil
}

func (manager *SAssignmentManager) add(ctx context.Context, typeStr, actorId, projectId, roleId string) error {
	assign := SAssignment{
		Type:      typeStr,
		ActorId:   actorId,
		TargetId:  projectId,
		RoleId:    roleId,
		Inherited: tristate.False,
	}
	assign.SetModelManager(manager, &assign)
	err := manager.TableSpec().InsertOrUpdate(ctx, &assign)
	if err != nil {
		return errors.Wrap(err, "InsertOrUpdate")
	}
	return nil
}

func AddAdhocHandlers(version string, app *appsrv.Application) {
	app.AddHandler2("GET", fmt.Sprintf("%s/role_assignments", version), auth.Authenticate(roleAssignmentHandler), nil, "list_role_assignments", nil)
}

func roleAssignmentHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, query, _ := appsrv.FetchEnv(ctx, w, r)
	input := api.RoleAssignmentsInput{}
	err := query.Unmarshal(&input)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}

	includeNames := (input.IncludeNames != nil)
	effective := (input.Effective != nil)
	includeSub := (input.IncludeSubtree != nil)
	includeSystem := (input.IncludeSystem != nil)
	includePolicies := (input.IncludePolicies != nil)

	limit := 0
	if input.Limit != nil {
		limit = *input.Limit
	}
	offset := 0
	if input.Offset != nil {
		offset = *input.Offset
	}

	results, total, err := AssignmentManager.FetchAll(
		input.User.Id,
		input.Group.Id,
		input.Role.Id,
		input.Scope.Domain.Id,
		input.Scope.Project.Id,
		input.ProjectDomainId,
		input.Users,
		input.Groups,
		input.Roles,
		input.Domains,
		input.Projects,
		input.ProjectDomains,
		includeNames, effective, includeSub, includeSystem, includePolicies,
		limit, offset)

	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}

	output := api.RoleAssignmentsOutput{}
	output.RoleAssignments = results
	output.Total = total
	output.Limit = limit
	output.Offset = offset
	appsrv.SendJSON(w, jsonutils.Marshal(output))
}

func (manager *SAssignmentManager) queryAll(
	userId, groupId, roleId, domainId, projectId string, projectDomainId string,
	users, groups, roles, domains, projects, projectDomains []string,
) *sqlchemy.SQuery {
	assigments := manager.Query().SubQuery()
	q := assigments.Query(
		assigments.Field("type"),
		sqlchemy.NewFunction(
			sqlchemy.NewCase().When(sqlchemy.OR(
				sqlchemy.Equals(assigments.Field("type"), sqlchemy.NewStringField(api.AssignmentUserProject)),
				sqlchemy.Equals(assigments.Field("type"), sqlchemy.NewStringField(api.AssignmentUserDomain)),
			), assigments.Field("actor_id")).Else(sqlchemy.NewStringField("")),
			"user_id",
			false,
		),
		sqlchemy.NewFunction(
			sqlchemy.NewCase().When(sqlchemy.OR(
				sqlchemy.Equals(assigments.Field("type"), sqlchemy.NewStringField(api.AssignmentGroupProject)),
				sqlchemy.Equals(assigments.Field("type"), sqlchemy.NewStringField(api.AssignmentGroupDomain)),
			), assigments.Field("actor_id")).Else(sqlchemy.NewStringField("")),
			"group_id",
			false,
		),
		sqlchemy.NewFunction(
			sqlchemy.NewCase().When(sqlchemy.OR(
				sqlchemy.Equals(assigments.Field("type"), sqlchemy.NewStringField(api.AssignmentUserDomain)),
				sqlchemy.Equals(assigments.Field("type"), sqlchemy.NewStringField(api.AssignmentGroupDomain)),
			), assigments.Field("target_id")).Else(sqlchemy.NewStringField("")),
			"domain_id",
			false,
		),
		sqlchemy.NewFunction(
			sqlchemy.NewCase().When(sqlchemy.OR(
				sqlchemy.Equals(assigments.Field("type"), sqlchemy.NewStringField(api.AssignmentUserProject)),
				sqlchemy.Equals(assigments.Field("type"), sqlchemy.NewStringField(api.AssignmentGroupProject)),
			), assigments.Field("target_id")).Else(sqlchemy.NewStringField("")),
			"project_id",
			false,
		),
		assigments.Field("role_id"),
	)
	// here use subquery.query to produce a effective reference to case function fields
	q = q.SubQuery().Query()
	if len(userId) > 0 {
		q = q.In("type", []string{api.AssignmentUserProject, api.AssignmentUserDomain}).Equals("user_id", userId)
	}
	if len(users) > 0 {
		subq := UserManager.Query("id")
		subq = subq.Filter(sqlchemy.OR(
			sqlchemy.In(subq.Field("id"), stringutils2.RemoveUtf8Strings(users)),
			sqlchemy.ContainsAny(subq.Field("name"), users),
		))
		q = q.In("type", []string{api.AssignmentUserProject, api.AssignmentUserDomain}).In("user_id", subq.SubQuery())
	}
	if len(groupId) > 0 {
		q = q.In("type", []string{api.AssignmentGroupProject, api.AssignmentGroupDomain}).Equals("group_id", groupId)
	}
	if len(groups) > 0 {
		subq := GroupManager.Query("id")
		subq = subq.Filter(sqlchemy.OR(
			sqlchemy.In(subq.Field("id"), stringutils2.RemoveUtf8Strings(groups)),
			sqlchemy.ContainsAny(subq.Field("name"), groups),
		))
		q = q.In("type", []string{api.AssignmentGroupProject, api.AssignmentGroupDomain}).In("group_id", subq.SubQuery())
	}
	if len(roleId) > 0 {
		q = q.Equals("role_id", roleId)
	}
	if len(roles) > 0 {
		subq := RoleManager.Query("id")
		subq = subq.Filter(sqlchemy.OR(
			sqlchemy.In(subq.Field("id"), stringutils2.RemoveUtf8Strings(roles)),
			sqlchemy.ContainsAny(subq.Field("name"), roles),
		))
		q = q.In("role_id", subq.SubQuery())
	}
	if len(projectId) > 0 {
		q = q.Equals("project_id", projectId).In("type", []string{api.AssignmentUserProject, api.AssignmentGroupProject})
	}
	if len(projects) > 0 {
		subq := ProjectManager.Query("id")
		subq = subq.Filter(sqlchemy.OR(
			sqlchemy.In(subq.Field("id"), stringutils2.RemoveUtf8Strings(projects)),
			sqlchemy.ContainsAny(subq.Field("name"), projects),
		))
		q = q.In("project_id", subq.SubQuery()).In("type", []string{api.AssignmentUserProject, api.AssignmentGroupProject})
	}
	if len(projectDomainId) > 0 {
		subq := ProjectManager.Query("id").Equals("domain_id", projectDomainId)
		q = q.In("project_id", subq.SubQuery()).In("type", []string{api.AssignmentUserProject, api.AssignmentGroupProject})
	}
	if len(projectDomains) > 0 {
		subq := ProjectManager.Query("id")
		domainQ := DomainManager.Query("id", "name").SubQuery()
		subq = subq.Join(domainQ, sqlchemy.Equals(subq.Field("domain_id"), domainQ.Field("id")))
		subq = subq.Filter(sqlchemy.OR(
			sqlchemy.In(domainQ.Field("id"), stringutils2.RemoveUtf8Strings(projectDomains)),
			sqlchemy.ContainsAny(domainQ.Field("name"), projectDomains),
		))
		q = q.In("project_id", subq.SubQuery()).In("type", []string{api.AssignmentUserProject, api.AssignmentGroupProject})
	}

	if len(domainId) > 0 {
		q = q.Equals("domain_id", domainId).In("type", []string{api.AssignmentUserDomain, api.AssignmentGroupDomain})
	}
	if len(domains) > 0 {
		subq := DomainManager.Query("id")
		subq = subq.Filter(sqlchemy.OR(
			sqlchemy.In(subq.Field("id"), stringutils2.RemoveUtf8Strings(domains)),
			sqlchemy.ContainsAny(subq.Field("name"), domains),
		))
		q = q.In("domain_id", subq.SubQuery()).In("type", []string{api.AssignmentUserDomain, api.AssignmentGroupDomain})
	}
	return q
}

func fetchRoleAssignmentPolicies(ra *api.SRoleAssignment) {
	policyNames, _, _ := RolePolicyManager.GetMatchPolicyGroup(ra, time.Time{}, true)
	ra.Policies.Project, _ = policyNames[rbacscope.ScopeProject]
	ra.Policies.Domain, _ = policyNames[rbacscope.ScopeDomain]
	ra.Policies.System, _ = policyNames[rbacscope.ScopeSystem]
}

type sAssignmentInternal struct {
	Type      string `json:"type"`
	UserId    string `json:"user_id"`
	GroupId   string `json:"group_id"`
	DomainId  string `json:"domain_id"`
	ProjectId string `json:"project_id"`
	RoleId    string `json:"role_id"`
}

func (assign *sAssignmentInternal) getRoleAssignment(domains, projects, groups, users, roles map[string]api.SFetchDomainObject, fetchPolicies bool, projectMetadata map[string]map[string]string) api.SRoleAssignment {
	ra := api.SRoleAssignment{}
	ra.Role.Id = assign.RoleId
	ra.Role.Name = roles[assign.RoleId].Name
	ra.Role.Domain.Id = roles[assign.RoleId].DomainId
	ra.Role.Domain.Name = roles[assign.RoleId].Domain
	if len(assign.UserId) > 0 {
		ra.User.Id = assign.UserId
		ra.User.Name = users[assign.UserId].Name
		ra.User.Domain.Id = users[assign.UserId].DomainId
		ra.User.Domain.Name = users[assign.UserId].Domain
	}
	if len(assign.GroupId) > 0 {
		ra.Group.Id = assign.GroupId
		ra.Group.Name = groups[assign.GroupId].Name
		ra.Group.Domain.Id = groups[assign.GroupId].DomainId
		ra.Group.Domain.Name = groups[assign.GroupId].Domain
	}
	if len(assign.ProjectId) > 0 {
		ra.Scope.Project.Id = assign.ProjectId
		ra.Scope.Project.Name = projects[assign.ProjectId].Name
		ra.Scope.Project.Metadata, _ = projectMetadata[assign.ProjectId]
		ra.Scope.Project.Domain.Id = projects[assign.ProjectId].DomainId
		ra.Scope.Project.Domain.Name = projects[assign.ProjectId].Domain
		if fetchPolicies {
			fetchRoleAssignmentPolicies(&ra)
		}
	} else if len(assign.DomainId) > 0 {
		ra.Scope.Domain.Id = assign.DomainId
		ra.Scope.Domain.Name = domains[assign.DomainId].Name
	}
	return ra
}

func (manager *SAssignmentManager) FetchAll(
	userId, groupId, roleId, domainId, projectId string, projectDomainId string,
	userStrs, groupStrs, roleStrs, domainStrs, projectStrs, projectDomainStrs []string,
	includeNames, effective, includeSub, includeSystem, includePolicies bool,
	limit, offset int) ([]api.SRoleAssignment, int64, error) {
	var q *sqlchemy.SQuery
	if effective {
		usrq := manager.queryAll(userId, "", roleId, domainId, projectId, projectDomainId, userStrs, nil, roleStrs, domainStrs, projectStrs, projectDomainStrs).In("type", []string{api.AssignmentUserProject, api.AssignmentUserDomain})

		memberships := UsergroupManager.Query("user_id", "group_id").SubQuery()

		grpproj := manager.queryAll("", groupId, roleId, domainId, projectId, projectDomainId, nil, groupStrs, roleStrs, domainStrs, projectStrs, projectDomainStrs).In("type", []string{api.AssignmentGroupProject, api.AssignmentGroupDomain}).SubQuery()
		q2 := grpproj.Query(
			grpproj.Field("type"),
			memberships.Field("user_id"),
			grpproj.Field("group_id"),
			grpproj.Field("domain_id"),
			grpproj.Field("project_id"),
			grpproj.Field("role_id"),
		)
		q2 = q2.LeftJoin(memberships, sqlchemy.Equals(grpproj.Field("group_id"), memberships.Field("group_id")))
		if len(userId) > 0 {
			q2 = q2.Filter(sqlchemy.Equals(memberships.Field("user_id"), userId))
		}
		if len(userStrs) > 0 {
			subq := UserManager.Query("id")
			subq = subq.Filter(sqlchemy.OR(
				sqlchemy.In(subq.Field("id"), stringutils2.RemoveUtf8Strings(userStrs)),
				sqlchemy.ContainsAny(subq.Field("name"), userStrs),
			))
			q2 = q2.Filter(sqlchemy.In(memberships.Field("user_id"), subq.SubQuery()))
		}

		q = sqlchemy.Union(usrq, q2).Query().Distinct()
	} else {
		q = manager.queryAll(userId, groupId, roleId, domainId, projectId, projectDomainId, userStrs, groupStrs, roleStrs, domainStrs, projectStrs, projectDomainStrs).Distinct()
	}

	if !includeSystem {
		users := UserManager.Query().SubQuery()
		q = q.LeftJoin(users, sqlchemy.Equals(q.Field("user_id"), users.Field("id")))
		q = q.Filter(sqlchemy.OR(
			sqlchemy.IsFalse(users.Field("is_system_account")),
			sqlchemy.IsNull(users.Field("is_system_account")),
		))
	}

	total, err := q.CountWithError()
	if err != nil {
		return nil, -1, errors.Wrap(err, "q.Count")
	}

	if limit > 0 {
		q = q.Limit(limit)
	}
	if offset > 0 {
		q = q.Offset(offset)
	}

	assigns := make([]sAssignmentInternal, 0)
	err = q.All(&assigns)
	if err != nil && err != sql.ErrNoRows {
		return nil, -1, httperrors.NewInternalServerError("query error %s", err)
	}

	domainIds := stringutils2.SSortedStrings{}
	projectIds := stringutils2.SSortedStrings{}
	groupIds := stringutils2.SSortedStrings{}
	userIds := stringutils2.SSortedStrings{}
	roleIds := stringutils2.SSortedStrings{}

	for i := range assigns {
		if len(assigns[i].UserId) > 0 {
			userIds = stringutils2.Append(userIds, assigns[i].UserId)
		}
		if len(assigns[i].GroupId) > 0 {
			groupIds = stringutils2.Append(groupIds, assigns[i].GroupId)
		}
		if len(assigns[i].DomainId) > 0 {
			domainIds = stringutils2.Append(domainIds, assigns[i].DomainId)
		}
		if len(assigns[i].ProjectId) > 0 {
			projectIds = stringutils2.Append(projectIds, assigns[i].ProjectId)
		}
		roleIds = stringutils2.Append(roleIds, assigns[i].RoleId)
	}

	domains, err := fetchObjects(DomainManager, domainIds)
	if err != nil {
		return nil, -1, errors.Wrap(err, "fetchObjects DomainManager")
	}
	projects, err := fetchObjects(ProjectManager, projectIds)
	if err != nil {
		return nil, -1, errors.Wrap(err, "fetchObjects ProjectManager")
	}
	projectMetadatas := fetchProjectMetadatas(projectIds)
	groups, err := fetchObjects(GroupManager, groupIds)
	if err != nil {
		return nil, -1, errors.Wrap(err, "fetchObjects GroupManager")
	}
	users, err := fetchObjects(UserManager, userIds)
	if err != nil {
		return nil, -1, errors.Wrap(err, "fetchObjects UserManager")
	}
	roles, err := fetchObjects(RoleManager, roleIds)
	if err != nil {
		return nil, -1, errors.Wrap(err, "fetchObjects RoleManager")
	}

	results := make([]api.SRoleAssignment, len(assigns))
	for i := range assigns {
		results[i] = assigns[i].getRoleAssignment(domains, projects, groups, users, roles, includePolicies, projectMetadatas)
	}
	return results, int64(total), nil
}

func (manager *SAssignmentManager) isUserInProjectWithRole(userId, projectId, roleId string) (bool, error) {
	q := manager.fetchUserProjectRoleIdsQuery(userId, projectId)
	q = q.Equals("role_id", roleId)

	cnt, err := q.CountWithError()
	if err != nil {
		return false, errors.Wrap(err, "CountWithError")
	}
	if cnt > 0 {
		return true, nil
	} else {
		return false, nil
	}
}

func fetchProjectMetadatas(idList []string) map[string]map[string]string {
	ret := map[string]map[string]string{}
	if len(idList) == 0 {
		return ret
	}
	q := db.Metadata.Query().Equals("obj_type", "project").In("obj_id", idList)
	result := []db.SMetadata{}
	err := q.All(&result)
	if err != nil {
		return ret
	}
	for i := range result {
		_, ok := ret[result[i].ObjId]
		if !ok {
			ret[result[i].ObjId] = map[string]string{}
		}
		ret[result[i].ObjId][result[i].Key] = result[i].Value
	}
	return ret
}

func fetchObjects(manager db.IModelManager, idList []string) (map[string]api.SFetchDomainObject, error) {
	results := make(map[string]api.SFetchDomainObject)
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
	objs := make([]api.SFetchDomainObject, 0)
	err := q.All(&objs)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "query")
	}
	for i := range objs {
		results[objs[i].Id] = objs[i]
	}
	return results, nil
}
