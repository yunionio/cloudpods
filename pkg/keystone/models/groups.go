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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
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
	GroupManager.SetVirtualObject(GroupManager)
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

	Displayname string `with:"128" charset:"utf8" nullable:"true" list:"domain" update:"domain" create:"domain_optional"`
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
	if group.IsReadOnly() {
		return httperrors.NewForbiddenError("readonly")
	}
	return group.SIdentityBaseResource.ValidateDeleteCondition(ctx)
}

func (group *SGroup) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := AssignmentManager.projectRemoveAllGroup(ctx, userCred, group)
	if err != nil {
		return errors.Wrap(err, "AssignmentManager.projectRemoveAllGroup")
	}

	err = UsergroupManager.delete("", group.Id)
	if err != nil {
		return errors.Wrap(err, "UsergroupManager.delete")
	}

	return group.SIdentityBaseResource.Delete(ctx, userCred)
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
	usrCnt, _ := group.GetUserCount()
	extra.Add(jsonutils.NewInt(int64(usrCnt)), "user_count")
	prjCnt, _ := group.GetProjectCount()
	extra.Add(jsonutils.NewInt(int64(prjCnt)), "project_count")
	return extra
}

func (manager *SGroupManager) RegisterExternalGroup(ctx context.Context, idpId string, domainId string, groupId string, groupName string) (*SGroup, error) {
	lockman.LockClass(ctx, manager, idpId)
	defer lockman.ReleaseClass(ctx, manager, idpId)

	pubId, err := IdmappingManager.RegisterIdMap(ctx, idpId, groupId, api.IdMappingEntityGroup)
	if err != nil {
		return nil, errors.Wrap(err, "IdmappingManager.registerIdMap")
	}

	group := SGroup{}
	group.SetModelManager(manager, &group)

	q := manager.RawQuery().Equals("id", pubId)
	err = q.First(&group)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "Query")
	}
	if err == sql.ErrNoRows {
		group.Id = pubId
		group.DomainId = domainId
		group.Name = groupName
		group.Displayname = groupName

		err = manager.TableSpec().Insert(&group)
		if err != nil {
			return nil, errors.Wrap(err, "Insert")
		}
	} else {
		_, err = db.Update(&group, func() error {
			group.DomainId = domainId
			group.Name = groupName
			group.Displayname = groupName
			group.MarkUnDelete()
			return nil
		})
		if err != nil {
			return nil, errors.Wrap(err, "update")
		}
	}

	return &group, nil
}

func (group *SGroup) ValidateUpdateCondition(ctx context.Context) error {
	// if group.IsReadOnly() {
	// 	return httperrors.NewForbiddenError("readonly")
	// }
	return group.SIdentityBaseResource.ValidateUpdateCondition(ctx)
}

func (group *SGroup) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if group.IsReadOnly() {
		for _, k := range []string{
			"name",
			"displayname",
		} {
			if data.Contains(k) {
				return nil, httperrors.NewForbiddenError("field %s is readonly", k)
			}
		}
	}
	return group.SIdentityBaseResource.ValidateUpdateData(ctx, userCred, query, data)
}

func (manager *SGroupManager) fetchGroupById(gid string) *SGroup {
	obj, _ := GroupManager.FetchById(gid)
	if obj != nil {
		return obj.(*SGroup)
	}
	return nil
}

func (manager *SGroupManager) NamespaceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeDomain
}

func (group *SGroup) getIdmapping() (*SIdmapping, error) {
	return IdmappingManager.FetchEntity(group.Id, api.IdMappingEntityGroup)
}

func (group *SGroup) IsReadOnly() bool {
	idmap, _ := group.getIdmapping()
	if idmap != nil {
		return true
	}
	return false
}

func (group *SGroup) LinkedWithIdp(idpId string) bool {
	idmap, _ := group.getIdmapping()
	if idmap != nil && idmap.IdpId == idpId {
		return true
	}
	return false
}

func (manager *SGroupManager) FetchCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, objs []db.IModel, fields stringutils2.SSortedStrings) []*jsonutils.JSONDict {
	rows := manager.SIdentityBaseResourceManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields)
	return expandIdpAttributes(rows, objs, fields, api.IdMappingEntityGroup)
}

func (manager *SGroupManager) FetchGroupsInDomain(domainId string, excludes []string) ([]SGroup, error) {
	q := manager.Query().Equals("domain_id", domainId).NotIn("id", excludes)
	grps := make([]SGroup, 0)
	err := db.FetchModelObjects(manager, q, &grps)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return grps, nil
}

func (group *SGroup) UnlinkIdp(idpId string) error {
	return IdmappingManager.deleteAny(idpId, api.IdMappingEntityGroup, group.Id)
}

func (group *SGroup) AllowPerformJoin(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) bool {
	return db.IsAdminAllowPerform(userCred, group, "join")
}

func (group *SGroup) PerformJoin(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	err := joinProjects(group, false, ctx, userCred, data)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (group *SGroup) AllowPerformLeave(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) bool {
	return db.IsAdminAllowPerform(userCred, group, "leave")
}

func (group *SGroup) PerformLeave(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	err := leaveProjects(group, false, ctx, userCred, data)
	if err != nil {
		return nil, err
	}
	return nil, nil
}
