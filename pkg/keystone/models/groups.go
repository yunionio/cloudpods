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
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
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

	// 用户组的显示名称
	Displayname string `with:"128" charset:"utf8" nullable:"true" list:"domain" update:"domain" create:"domain_optional"`
}

func (manager *SGroupManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{UserManager},
		{ProjectManager},
	}
}

// 用户组列表
func (manager *SGroupManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.GroupListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SIdentityBaseResourceManager.ListItemFilter(ctx, q, userCred, query.IdentityBaseResourceListInput)
	if err != nil {
		return nil, err
	}

	if len(query.Displayname) > 0 {
		q = q.Equals("displayname", query.Displayname)
	}

	userIdStr := query.UserId
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

	projIdStr := query.ProjectId
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

	if len(query.IdpId) > 0 {
		idpObj, err := IdentityProviderManager.FetchByIdOrName(ctx, userCred, query.IdpId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s", IdentityProviderManager.Keyword(), query.IdpId)
			} else {
				return nil, errors.Wrap(err, "IdentityProviderManager.FetchByIdOrName")
			}
		}
		subq := IdmappingManager.FetchPublicIdsExcludesQuery(idpObj.GetId(), api.IdMappingEntityGroup, nil)
		q = q.In("id", subq.SubQuery())
	}

	return q, nil
}

func (manager *SGroupManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.GroupListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SIdentityBaseResourceManager.OrderByExtraFields(ctx, q, userCred, query.IdentityBaseResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SIdentityBaseResourceManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SGroupManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SIdentityBaseResourceManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (group *SGroup) GetUserCount() (int, error) {
	q := UsergroupManager.Query().Equals("group_id", group.Id)
	return q.CountWithError()
}

func (group *SGroup) GetProjectCount() (int, error) {
	q := AssignmentManager.fetchGroupProjectIdsQuery(group.Id)
	return q.CountWithError()
}

func (group *SGroup) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	if group.IsReadOnly() {
		return httperrors.NewForbiddenError("readonly")
	}
	return group.SIdentityBaseResource.ValidateDeleteCondition(ctx, nil)
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

func (manager *SGroupManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.GroupDetails {
	rows := make([]api.GroupDetails, len(objs))
	identRows := manager.SIdentityBaseResourceManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	idList := make([]string, len(rows))
	for i := range rows {
		rows[i] = api.GroupDetails{
			IdentityBaseResourceDetails: identRows[i],
		}
		group := objs[i].(*SGroup)
		idList[i] = group.Id
		rows[i].UserCount, _ = group.GetUserCount()
		rows[i].ProjectCount, _ = group.GetProjectCount()
	}
	idpRows := expandIdpAttributes(api.IdMappingEntityGroup, idList, fields)
	for i := range rows {
		rows[i].IdpResourceInfo = idpRows[i]
	}
	return rows
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

		err = manager.TableSpec().Insert(ctx, &group)
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

func (group *SGroup) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.GroupUpdateInput) (api.GroupUpdateInput, error) {
	data := jsonutils.Marshal(input)
	if group.IsReadOnly() {
		for _, k := range []string{
			"name",
			"displayname",
		} {
			if data.Contains(k) {
				return input, httperrors.NewForbiddenError("field %s is readonly", k)
			}
		}
	}

	var err error
	input.IdentityBaseUpdateInput, err = group.SIdentityBaseResource.ValidateUpdateData(ctx, userCred, query, input.IdentityBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SIdentityBaseResource.ValidateUpdateData")
	}

	return input, nil
}

func (manager *SGroupManager) fetchGroupById(gid string) *SGroup {
	obj, _ := GroupManager.FetchById(gid)
	if obj != nil {
		return obj.(*SGroup)
	}
	return nil
}

func (manager *SGroupManager) NamespaceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeDomain
}

func (group *SGroup) getIdmapping() (*SIdmapping, error) {
	return IdmappingManager.FetchFirstEntity(group.Id, api.IdMappingEntityGroup)
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

// 组加入项目
func (group *SGroup) PerformJoin(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.SJoinProjectsInput,
) (jsonutils.JSONObject, error) {
	err := joinProjects(group, false, ctx, userCred, input)
	if err != nil {
		return nil, errors.Wrap(err, "joinProjects")
	}
	return nil, nil
}

// 组退出项目
func (group *SGroup) PerformLeave(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.SLeaveProjectsInput,
) (jsonutils.JSONObject, error) {
	err := leaveProjects(group, false, ctx, userCred, input)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (manager *SGroupManager) FilterByOwner(ctx context.Context, q *sqlchemy.SQuery, man db.FilterByOwnerProvider, userCred mcclient.TokenCredential, owner mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	if owner != nil && scope == rbacscope.ScopeProject {
		// if user has project level privilege, returns all groups in user's project
		subq := AssignmentManager.fetchProjectGroupIdsQuery(owner.GetProjectId())
		q = q.In("id", subq.SubQuery())
		return q
	}
	return manager.SIdentityBaseResourceManager.FilterByOwner(ctx, q, man, userCred, owner, scope)
}

func (group *SGroup) GetUsages() []db.IUsage {
	if group.Deleted {
		return nil
	}
	usage := SIdentityQuota{Group: 1}
	usage.SetKeys(quotas.SBaseDomainQuotaKeys{DomainId: group.DomainId})
	return []db.IUsage{
		&usage,
	}
}

func (manager *SGroupManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.GroupCreateInput,
) (api.GroupCreateInput, error) {
	var err error

	input.IdentityBaseResourceCreateInput, err = manager.SIdentityBaseResourceManager.ValidateCreateData(ctx, userCred, ownerId, query, input.IdentityBaseResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SIdentityBaseResourceManager.ValidateCreateData")
	}

	quota := &SIdentityQuota{
		SBaseDomainQuotaKeys: quotas.SBaseDomainQuotaKeys{DomainId: ownerId.GetProjectDomainId()},
		Group:                1,
	}
	err = quotas.CheckSetPendingQuota(ctx, userCred, quota)
	if err != nil {
		return input, errors.Wrap(err, "CheckSetPendingQuota")
	}

	return input, nil
}

func (group *SGroup) PostCreate(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) {
	group.SIdentityBaseResource.PostCreate(ctx, userCred, ownerId, query, data)

	quota := &SIdentityQuota{
		SBaseDomainQuotaKeys: quotas.SBaseDomainQuotaKeys{DomainId: ownerId.GetProjectDomainId()},
		Group:                1,
	}
	err := quotas.CancelPendingUsage(ctx, userCred, quota, quota, true)
	if err != nil {
		log.Errorf("CancelPendingUsage fail %s", err)
	}
}

// 组添加用户
func (group *SGroup) PerformAddUsers(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.PerformGroupAddUsersInput,
) (jsonutils.JSONObject, error) {
	users := make([]*SUser, 0)
	for _, uid := range input.UserIds {
		usr, err := UserManager.FetchByIdOrName(ctx, userCred, uid)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, errors.Wrapf(httperrors.ErrResourceNotFound, "user %s", uid)
			} else {
				return nil, errors.Wrap(err, "UserManager.FetchByIdOrName")
			}
		}
		users = append(users, usr.(*SUser))
	}
	for i := range users {
		err := UsergroupManager.add(ctx, userCred, users[i], group)
		if err != nil {
			return nil, errors.Wrap(err, "UsergroupManager.add")
		}
	}
	return nil, nil
}

// 组删除用户
func (group *SGroup) PerformRemoveUsers(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.PerformGroupRemoveUsersInput,
) (jsonutils.JSONObject, error) {
	users := make([]*SUser, 0)
	for _, uid := range input.UserIds {
		usr, err := UserManager.FetchByIdOrName(ctx, userCred, uid)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, errors.Wrapf(httperrors.ErrResourceNotFound, "user %s", uid)
			} else {
				return nil, errors.Wrap(err, "UserManager.FetchByIdOrName")
			}
		}
		users = append(users, usr.(*SUser))
	}
	for i := range users {
		err := UsergroupManager.remove(ctx, userCred, users[i], group)
		if err != nil {
			return nil, errors.Wrap(err, "UsergroupManager.add")
		}
	}
	return nil, nil
}
