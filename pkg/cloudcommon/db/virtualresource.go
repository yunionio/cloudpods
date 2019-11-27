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

package db

import (
	"context"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	identityapi "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type TProjectSource string

const (
	PROJECT_SOURCE_LOCAL = TProjectSource("local")
	PROJECT_SOURCE_CLOUD = TProjectSource("cloud")
)

type SVirtualResourceBaseManager struct {
	SStatusStandaloneResourceBaseManager
	SProjectizedResourceBaseManager
}

func NewVirtualResourceBaseManager(dt interface{}, tableName string, keyword string, keywordPlural string) SVirtualResourceBaseManager {
	return SVirtualResourceBaseManager{SStatusStandaloneResourceBaseManager: NewStatusStandaloneResourceBaseManager(dt,
		tableName, keyword, keywordPlural)}
}

type SVirtualResourceBase struct {
	SStatusStandaloneResourceBase
	SProjectizedResourceBase

	ProjectSrc string `width:"10" charset:"ascii" nullable:"false" list:"user" default:""`

	IsSystem bool `nullable:"true" default:"false" list:"admin" create:"optional"`

	PendingDeletedAt time.Time ``
	PendingDeleted   bool      `nullable:"false" default:"false" index:"true" get:"user"`
}

func (model *SVirtualResourceBase) IsOwner(userCred mcclient.TokenCredential) bool {
	return userCred.GetProjectId() == model.ProjectId
}

func (manager *SVirtualResourceBaseManager) GetIVirtualModelManager() IVirtualModelManager {
	return manager.GetVirtualObject().(IVirtualModelManager)
}

/*func (manager *SVirtualResourceBaseManager) FilterByOwner(q *sqlchemy.SQuery, owner mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	q = manager.SProjectizedResourceBaseManager.FilterByOwner(q, owner, scope)
	return q
}
*/

func (manager *SVirtualResourceBaseManager) FilterByName(q *sqlchemy.SQuery, name string) *sqlchemy.SQuery {
	q = manager.SStatusStandaloneResourceBaseManager.FilterByName(q, name)
	q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(q.Field("pending_deleted")), sqlchemy.IsFalse(q.Field("pending_deleted"))))
	return q
}

func (manager *SVirtualResourceBaseManager) FilterByHiddenSystemAttributes(q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	q = manager.SStatusStandaloneResourceBaseManager.FilterByHiddenSystemAttributes(q, userCred, query, scope)

	isSystem := jsonutils.QueryBoolean(query, "system", false)
	if isSystem {
		var isAllow bool
		if consts.IsRbacEnabled() {
			allowScope := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionList, "system")
			if !scope.HigherThan(allowScope) {
				isAllow = true
			}
		} else {
			if userCred.HasSystemAdminPrivilege() {
				isAllow = true
			}
		}
		if !isAllow {
			isSystem = false
		}
	}
	if !isSystem {
		q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(q.Field("is_system")), sqlchemy.IsFalse(q.Field("is_system"))))
	}
	return q
}

func (model *SVirtualResourceBase) SetProjectInfo(ctx context.Context, userCred mcclient.TokenCredential, projectId, domainId string) error {
	_, err := Update(model, func() error {
		model.ProjectId = projectId
		model.DomainId = domainId
		return nil
	})
	return err
}

func (manager *SVirtualResourceBaseManager) FilterBySystemAttributes(q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	q = manager.SStatusStandaloneResourceBaseManager.FilterBySystemAttributes(q, userCred, query, scope)

	var pendingDelete string
	if query != nil {
		pendingDelete, _ = query.GetString("pending_delete")
	}
	pendingDeleteLower := strings.ToLower(pendingDelete)
	if pendingDeleteLower == "all" || pendingDeleteLower == "any" || utils.ToBool(pendingDeleteLower) {
		var isAllow bool
		if consts.IsRbacEnabled() {
			allowScope := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionList, "pending_delete")
			if !scope.HigherThan(allowScope) {
				isAllow = true
			}
		} else {
			if userCred.HasSystemAdminPrivilege() {
				isAllow = true
			}
		}
		if !isAllow {
			pendingDeleteLower = ""
		}
	}

	if pendingDeleteLower == "all" || pendingDeleteLower == "any" {
	} else if utils.ToBool(pendingDeleteLower) {
		q = q.IsTrue("pending_deleted")
	} else {
		q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(q.Field("pending_deleted")), sqlchemy.IsFalse(q.Field("pending_deleted"))))
	}
	return q
}

func (manager *SVirtualResourceBaseManager) FetchByName(userCred mcclient.IIdentityProvider, idStr string) (IModel, error) {
	return FetchByName(manager, userCred, idStr)
}

func (manager *SVirtualResourceBaseManager) FetchByIdOrName(userCred mcclient.IIdentityProvider, idStr string) (IModel, error) {
	return FetchByIdOrName(manager, userCred, idStr)
}

func (manager *SVirtualResourceBaseManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	isSystem, err := data.Bool("is_system")
	if err == nil && isSystem && !IsAdminAllowCreate(userCred, manager) {
		return nil, httperrors.NewNotSufficientPrivilegeError("non-admin user not allowed to create system object")
	}
	return manager.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data)
}

func (model *SVirtualResourceBase) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	model.DomainId = ownerId.GetProjectDomainId()
	model.ProjectId = ownerId.GetProjectId()
	isSystem, err := data.Bool("is_system")
	if err == nil && isSystem {
		model.IsSystem = true
	} else {
		model.IsSystem = false
	}
	model.ProjectSrc = string(PROJECT_SOURCE_LOCAL)
	return model.SStandaloneResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (manager *SVirtualResourceBaseManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	isAdmin, err := query.Bool("admin")
	if err == nil && isAdmin && !IsAdminAllowList(userCred, manager) {
		return false
	}
	return true
}

func (model *SVirtualResourceBase) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return model.IsOwner(userCred) || IsAdminAllowGet(userCred, model)
}

func (manager *SVirtualResourceBaseManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return true
}

func (manager *SVirtualResourceBaseManager) FetchCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, objs []IModel, fields stringutils2.SSortedStrings) []*jsonutils.JSONDict {
	rows := manager.SStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields)
	if len(fields) == 0 || fields.Contains("tenant") || fields.Contains("project_domain") {
		projectIds := stringutils2.SSortedStrings{}
		for i := range objs {
			idStr := objs[i].GetOwnerId().GetProjectId()
			projectIds = stringutils2.Append(projectIds, idStr)
		}
		projects := FetchProjects(projectIds, false)
		if projects != nil {
			for i := range rows {
				idStr := objs[i].GetOwnerId().GetProjectId()
				if proj, ok := projects[idStr]; ok {
					if len(fields) == 0 || fields.Contains("project_domain") {
						rows[i].Add(jsonutils.NewString(proj.Domain), "project_domain")
					}
					if len(fields) == 0 || fields.Contains("tenant") {
						rows[i].Add(jsonutils.NewString(proj.Name), "tenant")
					}
				}

			}
		}
	}
	return rows
}

func FetchProjects(projectIds []string, isDomain bool) map[string]STenant {
	deadline := time.Now().UTC().Add(-consts.GetTenantCacheExpireSeconds())
	q := TenantCacheManager.Query().In("id", projectIds).GT("last_check", deadline)
	if isDomain {
		q = q.Equals("domain_id", identityapi.KeystoneDomainRoot)
	} else {
		q = q.NotEquals("domain_id", identityapi.KeystoneDomainRoot)
	}
	projects := make([]STenant, 0)
	err := FetchModelObjects(TenantCacheManager, q, &projects)
	if err != nil {
		return nil
	}
	ret := make(map[string]STenant)
	for i := range projects {
		ret[projects[i].Id] = projects[i]
	}
	ctx := context.Background()
	for _, pid := range projectIds {
		if _, ok := ret[pid]; !ok {
			// not found
			var t *STenant
			if isDomain {
				t, _ = TenantCacheManager.fetchDomainFromKeystone(ctx, pid)
			} else {
				t, _ = TenantCacheManager.fetchTenantFromKeystone(ctx, pid)
			}
			if t != nil {
				ret[t.Id] = *t
			}
		}
	}
	return ret
}

func (model *SVirtualResourceBase) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return model.IsOwner(userCred) || IsAdminAllowUpdate(userCred, model)
}

func (model *SVirtualResourceBase) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return model.IsOwner(userCred) || IsAdminAllowDelete(userCred, model)
}

func (model *SVirtualResourceBase) AllowGetDetailsMetadata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return model.IsOwner(userCred) || IsAdminAllowGetSpec(userCred, model, "metadata")
}

func (model *SVirtualResourceBase) AllowPerformMetadata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return model.IsOwner(userCred) || IsAdminAllowPerform(userCred, model, "metadata")
}

func (model *SVirtualResourceBase) AllowGetDetailsStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return model.IsOwner(userCred) || IsAdminAllowGetSpec(userCred, model, "status")
}

func (model *SVirtualResourceBase) GetTenantCache(ctx context.Context) (*STenant, error) {
	// log.Debugf("Get tenant by Id %s", model.ProjectId)
	return TenantCacheManager.FetchTenantById(ctx, model.ProjectId)
}

func (model *SVirtualResourceBase) getMoreDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	//if IsAdminAllowGet(userCred, model) {
	// log.Debugf("GetCustomizeColumns")
	// tobj, err := model.GetTenantCache(ctx)
	// if err == nil {
	// log.Debugf("GetTenantFromCache %s", jsonutils.Marshal(tobj))
	// extra.Add(jsonutils.NewString(tobj.GetName()), "tenant")
	// } else {
	// 	log.Errorf("GetTenantCache fail %s", err)
	// }
	// }
	admin, _ := query.GetString("admin")
	if utils.ToBool(admin) { // admin
		pendingDelete, _ := query.GetString("pending_delete")
		pendingDeleteLower := strings.ToLower(pendingDelete)
		if pendingDeleteLower == "all" || pendingDeleteLower == "any" {
			extra.Set("pending_deleted", jsonutils.NewBool(model.PendingDeleted))
		}
	}
	return extra
}

func (model *SVirtualResourceBase) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := model.SStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return model.getMoreDetails(ctx, userCred, query, extra)
}

func (model *SVirtualResourceBase) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := model.SStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return model.getMoreDetails(ctx, userCred, query, extra), nil
}

func (model *SVirtualResourceBase) AllowPerformChangeOwner(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return IsAdminAllowPerform(userCred, model, "change-owner")
}

func (model *SVirtualResourceBase) PerformChangeOwner(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	manager := model.GetModelManager()

	log.Debugf("SVirtualResourceBase change_owner %s %s %#v", query, data, manager)
	ownerId, err := manager.FetchOwnerId(ctx, data)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	if len(ownerId.GetProjectId()) == 0 {
		return nil, httperrors.NewInputParameterError("missing new project/tenant")
	}
	if ownerId.GetProjectId() == model.ProjectId {
		// do nothing
		Update(model, func() error {
			model.ProjectSrc = string(PROJECT_SOURCE_LOCAL)
			return nil
		})
		return nil, nil
	}
	q := manager.Query().Equals("name", model.GetName())
	q = manager.FilterByOwner(q, ownerId, manager.NamespaceScope())
	q = manager.FilterBySystemAttributes(q, nil, nil, manager.ResourceScope())
	q = q.NotEquals("id", model.GetId())
	cnt, err := q.CountWithError()
	if err != nil {
		return nil, httperrors.NewInternalServerError("check name duplication error: %s", err)
	}
	if cnt > 0 {
		return nil, httperrors.NewDuplicateNameError("name", model.GetName())
	}
	former, _ := TenantCacheManager.FetchTenantById(ctx, model.ProjectId)
	if former == nil {
		log.Warningf("tenant_id %s not found", model.ProjectId)
		formerObj := NewTenant(model.ProjectId, "unknown", model.DomainId, "unknown")
		former = &formerObj
	}

	// clean shared projects before update project id
	if err := SharedResourceManager.CleanModelSharedProjects(ctx, userCred, model); err != nil {
		return nil, err
	}

	_, err = Update(model, func() error {
		model.DomainId = ownerId.GetProjectDomainId()
		model.ProjectId = ownerId.GetProjectId()
		model.ProjectSrc = string(PROJECT_SOURCE_LOCAL)
		return nil
	})
	if err != nil {
		return nil, err
	}

	OpsLog.SyncOwner(model, former, userCred)
	notes := struct {
		OldProjectId string
		OldProject   string
		NewProjectId string
		NewProject   string
		OldDomainId  string
		OldDomain    string
		NewDomainId  string
		NewDomain    string
	}{
		OldProjectId: former.Id,
		OldProject:   former.Name,
		NewProjectId: ownerId.GetProjectId(),
		NewProject:   ownerId.GetProjectName(),
		OldDomainId:  former.DomainId,
		OldDomain:    former.Domain,
		NewDomainId:  ownerId.GetProjectDomainId(),
		NewDomain:    ownerId.GetProjectDomain(),
	}
	logclient.AddActionLogWithContext(ctx, model, logclient.ACT_CHANGE_OWNER, notes, userCred, true)
	return nil, nil
}

func (model *SVirtualResourceBase) DoPendingDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return model.MarkPendingDelete(userCred)
}

func (model *SVirtualResourceBase) MarkPendingDelete(userCred mcclient.TokenCredential) error {
	if !model.PendingDeleted {
		diff, err := Update(model, func() error {
			model.PendingDeleted = true
			model.PendingDeletedAt = timeutils.UtcNow()
			return nil
		})
		if err != nil {
			log.Errorf("MarkPendingDelete update fail %s", err)
			return err
		}
		OpsLog.LogEvent(model, ACT_PENDING_DELETE, diff, userCred)
		logclient.AddSimpleActionLog(model, logclient.ACT_PENDING_DELETE, "", userCred, true)
	}
	return nil
}

func (model *SVirtualResourceBase) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	if !model.PendingDeleted {
		model.DoPendingDelete(ctx, userCred)
	}
	return DeleteModel(ctx, userCred, model)
}

func (model *SVirtualResourceBase) AllowPerformCancelDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (model *SVirtualResourceBase) PerformCancelDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if model.PendingDeleted && !model.Deleted {
		err := model.DoCancelPendingDelete(ctx, userCred)
		return nil, err
	}
	return nil, nil
}

func (model *SVirtualResourceBase) DoCancelPendingDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := model.CancelPendingDelete(ctx, userCred)
	if err == nil {
		OpsLog.LogEvent(model, ACT_CANCEL_DELETE, model.GetShortDesc(ctx), userCred)
	}
	return err
}

func (model *SVirtualResourceBase) VirtualModelManager() IVirtualModelManager {
	return model.GetModelManager().(IVirtualModelManager)
}

func (model *SVirtualResourceBase) GetIVirtualModel() IVirtualModel {
	return model.GetVirtualObject().(IVirtualModel)
}

func (model *SVirtualResourceBase) CancelPendingDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	if model.PendingDeleted && !model.Deleted {
		return model.MarkCancelPendingDelete(ctx, userCred)
	}
	return nil
}

func (model *SVirtualResourceBase) MarkCancelPendingDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	manager := model.GetModelManager()
	ownerId := model.GetOwnerId()

	lockman.LockClass(ctx, manager, GetLockClassKey(manager, ownerId))
	defer lockman.ReleaseClass(ctx, manager, GetLockClassKey(manager, ownerId))

	newName, err := GenerateName(manager, ownerId, model.Name)
	if err != nil {
		return err
	}
	diff, err := Update(model, func() error {
		model.Name = newName
		model.PendingDeleted = false
		model.PendingDeletedAt = time.Time{}
		return nil
	})
	if err != nil {
		log.Errorf("MarkCancelPendingDelete fail %s", err)
		return err
	}
	OpsLog.LogEvent(model, ACT_CANCEL_DELETE, diff, userCred)
	return nil
}

func (model *SVirtualResourceBase) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	desc := model.SStatusStandaloneResourceBase.GetShortDesc(ctx)
	desc.Add(jsonutils.NewString(model.ProjectId), "owner_tenant_id")
	tc, _ := TenantCacheManager.FetchTenantById(ctx, model.ProjectId)
	if tc != nil {
		desc.Add(jsonutils.NewString(tc.GetName()), "owner_tenant")
	}
	return desc
}

func (model *SVirtualResourceBase) SyncCloudProjectId(userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider) {
	if model.ProjectSrc != string(PROJECT_SOURCE_LOCAL) && ownerId != nil && len(ownerId.GetProjectId()) > 0 {
		diff, _ := Update(model, func() error {
			model.ProjectSrc = string(PROJECT_SOURCE_CLOUD)
			model.ProjectId = ownerId.GetProjectId()
			model.DomainId = ownerId.GetProjectDomainId()
			return nil
		})
		if len(diff) > 0 {
			OpsLog.LogEvent(model, ACT_SYNC_OWNER, diff, userCred)
		}
	}
}

// GetPendingDeleted implements IPendingDeltable
func (model *SVirtualResourceBase) GetPendingDeleted() bool {
	return model.PendingDeleted
}

// GetPendingDeletedAt implements IPendingDeltable
func (model *SVirtualResourceBase) GetPendingDeletedAt() time.Time {
	return model.PendingDeletedAt
}

func (manager *SVirtualResourceBaseManager) OrderByExtraFields(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	orderByTenant, _ := query.GetString("order_by_tenant")
	if sqlchemy.SQL_ORDER_ASC.Equals(orderByTenant) || sqlchemy.SQL_ORDER_DESC.Equals(orderByTenant) {
		tenantCaches := TenantCacheManager.Query().SubQuery()
		q = q.LeftJoin(tenantCaches, sqlchemy.AND(
			sqlchemy.Equals(q.Field("tenant_id"), tenantCaches.Field("id")),
			sqlchemy.NotEquals(tenantCaches.Field("domain_id"), identityapi.KeystoneDomainRoot),
		))
		if sqlchemy.SQL_ORDER_ASC.Equals(orderByTenant) {
			q = q.Asc(tenantCaches.Field("name"))
		} else {
			q = q.Desc(tenantCaches.Field("name"))
		}
	}

	orderByDomain, _ := query.GetString("order_by_domain")
	if sqlchemy.SQL_ORDER_ASC.Equals(orderByDomain) || sqlchemy.SQL_ORDER_DESC.Equals(orderByDomain) {
		tenantCaches := TenantCacheManager.Query().SubQuery()
		q = q.LeftJoin(tenantCaches, sqlchemy.AND(
			sqlchemy.Equals(q.Field("domain_id"), tenantCaches.Field("id")),
			sqlchemy.Equals(tenantCaches.Field("domain_id"), identityapi.KeystoneDomainRoot),
		))
		if sqlchemy.SQL_ORDER_ASC.Equals(orderByTenant) {
			q = q.Asc(tenantCaches.Field("name"))
		} else {
			q = q.Desc(tenantCaches.Field("name"))
		}
	}

	return q, nil
}
