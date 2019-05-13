package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/sqlchemy"

	"database/sql"

	"github.com/pkg/errors"
	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SGroupManager struct {
	SIdentityBaseResourceManager
}

var GroupManager *SGroupManager

func init() {
	GroupManager = &SGroupManager{
		SIdentityBaseResourceManager: NewIdentityBaseResourceManager(
			SGroup{},
			"group",
			"group",
			"groups",
		),
	}
}

/*
+-------------+-------------+------+-----+---------+-------+
| Field       | Type        | Null | Key | Default | Extra |
+-------------+-------------+------+-----+---------+-------+
| id          | varchar(64) | NO   | PRI | NULL    |       |
| domain_id   | varchar(64) | NO   | MUL | NULL    |       |
| name        | varchar(64) | NO   |     | NULL    |       |
| description | text        | YES  |     | NULL    |       |
| extra       | text        | YES  |     | NULL    |       |
| created_at  | datetime    | YES  |     | NULL    |       |
+-------------+-------------+------+-----+---------+-------+
*/

type SGroup struct {
	SIdentityBaseResource

	Displayname string `with:"128" charset:"utf8" nullable:"true" list:"admin" update:"admin" create:"admin_optional"`
}

func (manager *SGroupManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{UserManager},
		{ProjectManager},
	}
}

func (manager *SGroupManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := manager.SIdentityBaseResourceManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}

	userIdStr := jsonutils.GetAnyString(query, []string{"user_id"})
	if len(userIdStr) > 0 {
		user, err := UserManager.FetchById(userIdStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(UserManager.Keyword(), userIdStr)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		subq := UsergroupManager.Query("group_id").Equals("user_id", user.GetId())
		q = q.In("id", subq.SubQuery())
	}

	projIdStr := jsonutils.GetAnyString(query, []string{"project_id", "tenant_id"})
	if len(projIdStr) > 0 {
		proj, err := ProjectManager.FetchProjectById(projIdStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(ProjectManager.Keyword(), projIdStr)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		subq := AssignmentManager.fetchProjectGroupIdsQuery(proj.Id)
		q = q.In("id", subq.SubQuery())
	}

	return q, nil
}

func (group *SGroup) GetUserCount() (int, error) {
	q := UsergroupManager.Query().Equals("group_id", group.Id)
	return q.CountWithError()
}

func (group *SGroup) GetProjectCount() (int, error) {
	q := AssignmentManager.fetchGroupProjectIdsQuery(group.Id)
	return q.CountWithError()
}

func (group *SGroup) ValidateDeleteCondition(ctx context.Context) error {
	usrCnt, _ := group.GetUserCount()
	if usrCnt > 0 {
		return httperrors.NewNotEmptyError("group contains user")
	}
	prjCnt, _ := group.GetProjectCount()
	if prjCnt > 0 {
		return httperrors.NewNotEmptyError("group joins project")
	}
	return group.SIdentityBaseResource.ValidateDeleteCondition(ctx)
}

func (group *SGroup) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := group.SIdentityBaseResource.GetCustomizeColumns(ctx, userCred, query)
	return groupExtra(group, extra)
}

func (group *SGroup) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := group.SIdentityBaseResource.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return groupExtra(group, extra), nil
}

func groupExtra(group *SGroup, extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	domain := group.GetDomain()
	if domain != nil {
		extra.Add(jsonutils.NewString(domain.Name), "domain")
	}

	usrCnt, _ := group.GetUserCount()
	extra.Add(jsonutils.NewInt(int64(usrCnt)), "user_count")
	prjCnt, _ := group.GetProjectCount()
	extra.Add(jsonutils.NewInt(int64(prjCnt)), "project_count")
	return extra
}

func (manager *SGroupManager) RegisterExternalGroup(ctx context.Context, domainId string, groupId string, groupName string) (*SGroup, error) {
	lockman.LockClass(ctx, manager, domainId)
	defer lockman.ReleaseClass(ctx, manager, domainId)

	pubId, err := IdmappingManager.registerIdMap(ctx, domainId, groupId, api.IdMappingEntityGroup)
	if err != nil {
		return nil, errors.WithMessage(err, "IdmappingManager.registerIdMap")
	}

	group := SGroup{}
	group.SetModelManager(manager)

	q := manager.Query().Equals("id", pubId)
	err = q.First(&group)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.WithMessage(err, "Query")
	}
	if err == sql.ErrNoRows {
		group.Id = pubId
		group.DomainId = domainId
		group.Name = groupId
		group.Displayname = groupName

		err = manager.TableSpec().Insert(&group)
		if err != nil {
			return nil, errors.WithMessage(err, "Insert")
		}
	}

	return &group, nil
}

func (group *SGroup) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return group.SIdentityBaseResource.ValidateUpdateData(ctx, userCred, query, data)
}

func (manager *SGroupManager) fetchGroupById(gid string) *SGroup {
	obj, _ := GroupManager.FetchById(gid)
	if obj != nil {
		return obj.(*SGroup)
	}
	return nil
}
