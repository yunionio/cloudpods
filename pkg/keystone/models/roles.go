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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
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
	SIdentityBaseResource
	db.SSharableBaseResource
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
	err = manager.initSysRole()
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

func (manager *SRoleManager) initSysRole() error {
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
	role.DomainId = api.DEFAULT_DOMAIN_ID
	role.Description = "Boostrap system default admin role"
	role.SetModelManager(manager, &role)

	err = manager.TableSpec().Insert(&role)
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

func (role *SRole) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if data.Contains("name") {
		return nil, httperrors.NewForbiddenError("cannot alter name of role")
	}
	return role.SIdentityBaseResource.ValidateUpdateData(ctx, userCred, query, data)
}

func (role *SRole) IsSystemRole() bool {
	return role.Name == api.SystemAdminRole && role.DomainId == api.DEFAULT_DOMAIN_ID
}

func (role *SRole) ValidateDeleteCondition(ctx context.Context) error {
	if role.IsPublic {
		return httperrors.NewInvalidStatusError("cannot delete shared role")
	}
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
	return role.SIdentityBaseResource.ValidateDeleteCondition(ctx)
}

func (role *SRole) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := role.SIdentityBaseResource.GetCustomizeColumns(ctx, userCred, query)
	return roleExtra(role, extra)
}

func (role *SRole) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := role.SIdentityBaseResource.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return roleExtra(role, extra), nil
}

func roleExtra(role *SRole, extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	usrCnt, _ := role.GetUserCount()
	extra.Add(jsonutils.NewInt(int64(usrCnt)), "user_count")
	grpCnt, _ := role.GetGroupCount()
	extra.Add(jsonutils.NewInt(int64(grpCnt)), "group_count")
	prjCnt, _ := role.GetProjectCount()
	extra.Add(jsonutils.NewInt(int64(prjCnt)), "project_count")
	policies := policy.PolicyManager.RoleMatchPolicies(role.Name)
	if len(policies) > 0 {
		extra.Add(jsonutils.NewStringArray(policies), "match_policies")
	}
	return extra
}

func (manager *SRoleManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := manager.SIdentityBaseResourceManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}

	var projectId string
	projectStr := jsonutils.GetAnyString(query, []string{"project_id"})
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

	userStr := jsonutils.GetAnyString(query, []string{"user_id"})
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

	groupStr := jsonutils.GetAnyString(query, []string{"group_id"})
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

func (role *SRole) UpdateInContext(ctx context.Context, userCred mcclient.TokenCredential, ctxObjs []db.IModel, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if len(ctxObjs) != 2 {
		return nil, httperrors.NewInputParameterError("not supported update context")
	}
	project, ok := ctxObjs[0].(*SProject)
	if !ok {
		return nil, httperrors.NewInputParameterError("not supported update context %s", ctxObjs[0].Keyword())
	}
	if project.DomainId != role.DomainId && !role.GetIsPublic() {
		return nil, httperrors.NewInputParameterError("inconsistent domain for project and roles")
	}
	switch obj := ctxObjs[1].(type) {
	case *SUser:
		return nil, AssignmentManager.projectAddUser(ctx, userCred, project, obj, role)
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

func (role *SRole) AllowPerformPublic(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.SharableAllowPerformPublic(role, userCred)
}

func (role *SRole) PerformPublic(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	res, err := db.SharablePerformPublic(role, ctx, userCred, query, data)
	if err == nil {
		policy.PolicyManager.SyncOnce()
	}
	return res, err
}

func (role *SRole) AllowPerformPrivate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.SharableAllowPerformPrivate(role, userCred)
}

func (role *SRole) PerformPrivate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	res, err := db.SharablePerformPrivate(role, ctx, userCred, query, data)
	if err == nil {
		policy.PolicyManager.SyncOnce()
	}
	return res, err
}

func (manager *SRoleManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	err := db.ValidateCreateDomainId(ownerId.GetProjectDomainId())
	if err != nil {
		return nil, err
	}
	input := api.IdentityBaseResourceCreateInput{}
	err = data.Unmarshal(&input)
	if err != nil {
		return nil, httperrors.NewInternalServerError("unmarshal IdentityBaseResourceCreateInput fail %s", err)
	}
	input, err = manager.SIdentityBaseResourceManager.ValidateCreateData(ctx, userCred, ownerId, query, input)
	if err != nil {
		return nil, err
	}
	data.Update(jsonutils.Marshal(input))
	return data, nil
}
