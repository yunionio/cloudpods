package models

import (
	"context"
	"database/sql"

	"yunion.io/x/jsonutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SRoleManager struct {
	SIdentityBaseResourceManager
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
	q = q.IsNull("description")
	roles := make([]SRole, 0)
	err := db.FetchModelObjects(manager, q, &roles)
	if err != nil {
		return err
	}
	for i := range roles {
		desc, _ := roles[i].Extra.GetString("description")
		db.Update(&roles[i], func() error {
			roles[i].Description = desc
			return nil
		})
	}
	return manager.InitializeDomainId()
}

func (manager *SRoleManager) InitializeDomainId() error {
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

func (role *SRole) ValidateDeleteCondition(ctx context.Context) error {
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
	domain := role.GetDomain()
	if domain != nil {
		extra.Add(jsonutils.NewString(domain.Name), "domain")
	}

	usrCnt, _ := role.GetUserCount()
	extra.Add(jsonutils.NewInt(int64(usrCnt)), "user_count")
	grpCnt, _ := role.GetGroupCount()
	extra.Add(jsonutils.NewInt(int64(grpCnt)), "group_count")
	prjCnt, _ := role.GetProjectCount()
	extra.Add(jsonutils.NewInt(int64(prjCnt)), "project_count")
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
