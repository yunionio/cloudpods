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

	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type TProjectSource string

const (
	PROJECT_SOURCE_LOCAL = TProjectSource("local")
	PROJECT_SOURCE_CLOUD = TProjectSource("cloud")
)

type SVirtualResourceBaseManager struct {
	SStatusStandaloneResourceBaseManager
}

func NewVirtualResourceBaseManager(dt interface{}, tableName string, keyword string, keywordPlural string) SVirtualResourceBaseManager {
	return SVirtualResourceBaseManager{SStatusStandaloneResourceBaseManager: NewStatusStandaloneResourceBaseManager(dt,
		tableName, keyword, keywordPlural)}
}

type SVirtualResourceBase struct {
	SStatusStandaloneResourceBase

	ProjectId  string `name:"tenant_id" width:"128" charset:"ascii" nullable:"false" index:"true" list:"user"`
	ProjectSrc string `width:"10" charset:"ascii" nullable:"false" list:"user" default:""`

	IsSystem bool `nullable:"true" default:"false" list:"admin" create:"optional"`

	PendingDeletedAt time.Time ``
	PendingDeleted   bool      `nullable:"false" default:"false" index:"true" get:"admin"`
}

func (model *SVirtualResourceBase) IsOwner(userCred mcclient.TokenCredential) bool {
	return userCred.GetProjectId() == model.ProjectId
}

/*func (model *SVirtualResourceBase) IsAdmin(userCred mcclient.TokenCredential) bool {
	return userCred.IsSystemAdmin() || (userCred.GetProjectId() == model.ProjectId && userCred.IsAdmin())
}*/

func (model *SVirtualResourceBase) GetOwnerProjectId() string {
	return model.ProjectId
}

func (manager *SVirtualResourceBaseManager) FilterByOwner(q *sqlchemy.SQuery, owner string) *sqlchemy.SQuery {
	q = q.Equals("tenant_id", owner)
	q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(q.Field("pending_deleted")), sqlchemy.IsFalse(q.Field("pending_deleted"))))
	q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(q.Field("is_system")), sqlchemy.IsFalse(q.Field("is_system"))))
	return q
}

func (manager *SVirtualResourceBaseManager) FetchByName(userCred mcclient.IIdentityProvider, idStr string) (IModel, error) {
	return FetchByName(manager, userCred, idStr)
}

func (manager *SVirtualResourceBaseManager) FetchByIdOrName(userCred mcclient.IIdentityProvider, idStr string) (IModel, error) {
	return FetchByIdOrName(manager, userCred, idStr)
}

func (manager *SVirtualResourceBaseManager) GetOwnerId(userCred mcclient.IIdentityProvider) string {
	return userCred.GetProjectId()
}

/*func (manager *SVirtualResourceBaseManager) IsOwnerFilter(q *sqlchemy.SQuery, userCred mcclient.TokenCredential) *sqlchemy.SQuery {
	return q.Equals("tenant_id", userCred.GetProjectId())
}*/

func (manager *SVirtualResourceBaseManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	admin, _ := query.GetString("admin")
	if utils.ToBool(admin) { // admin
		tenant := jsonutils.GetAnyString(query, []string{"project", "project_id", "tenant", "tenant_id"})
		if len(tenant) > 0 {
			tobj, _ := TenantCacheManager.FetchTenantByIdOrName(ctx, tenant)
			if tobj != nil {
				q = q.Equals("tenant_id", tobj.GetId())
			} else {
				return nil, httperrors.NewTenantNotFoundError("tenant %s not found", tenant)
			}
		}
		isSystem, err := query.Bool("system")
		if err == nil && isSystem {
			// no filter
		} else {
			q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(q.Field("is_system")), sqlchemy.IsFalse(q.Field("is_system"))))
		}
		pendingDelete, _ := query.GetString("pending_delete")
		pendingDeleteLower := strings.ToLower(pendingDelete)
		if pendingDeleteLower == "all" || pendingDeleteLower == "any" {
			// no filter
		} else {
			if utils.ToBool(pendingDelete) {
				q = q.IsTrue("pending_deleted")
			} else {
				q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(q.Field("pending_deleted")), sqlchemy.IsFalse(q.Field("pending_deleted"))))
			}
		}
	}
	return q, nil
}

func (manager *SVirtualResourceBaseManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	isSystem, err := data.Bool("is_system")
	if err == nil && isSystem && !IsAdminAllowCreate(userCred, manager) {
		return nil, httperrors.NewNotSufficientPrivilegeError("non-admin user not allowed to create system object")
	}
	return manager.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
}

func (model *SVirtualResourceBase) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	model.ProjectId = ownerProjId
	isSystem, err := data.Bool("is_system")
	if err == nil && isSystem {
		model.IsSystem = true
	} else {
		model.IsSystem = false
	}
	model.ProjectSrc = string(PROJECT_SOURCE_LOCAL)
	return model.SStandaloneResourceBase.CustomizeCreate(ctx, userCred, ownerProjId, query, data)
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
	if IsAdminAllowGet(userCred, model) {
		// log.Debugf("GetCustomizeColumns")
		tobj, err := model.GetTenantCache(ctx)
		if err == nil {
			// log.Debugf("GetTenantFromCache %s", jsonutils.Marshal(tobj))
			extra.Add(jsonutils.NewString(tobj.GetName()), "tenant")
		} else {
			log.Errorf("GetTenantCache fail %s", err)
		}
	}
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
	tenant := jsonutils.GetAnyString(data, []string{"project", "tenant", "project_id", "tenant_id"})
	if len(tenant) == 0 {
		return nil, httperrors.NewMissingParameterError("tenant_id")
	}
	tobj, _ := TenantCacheManager.FetchTenantByIdOrName(ctx, tenant)
	if tobj == nil {
		return nil, httperrors.NewTenantNotFoundError("tenant %s not found", tenant)
	}
	if tobj.GetId() == model.ProjectId {
		// do nothing
		Update(model, func() error {
			model.ProjectSrc = string(PROJECT_SOURCE_LOCAL)
			return nil
		})
		return nil, nil
	}
	q := model.GetModelManager().Query().Equals("name", model.GetName())
	q = q.Equals("tenant_id", tobj.GetId())
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
		log.Errorf("tenant_id %s not found", model.ProjectId)
		formerObj := NewTenant(model.ProjectId, "unknown")
		former = &formerObj
	}
	_, err = Update(model, func() error {
		model.ProjectId = tobj.GetId()
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
	}{
		OldProjectId: former.Id,
		OldProject:   former.Name,
		NewProjectId: tobj.GetId(),
		NewProject:   tobj.GetName(),
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

/*func DoCancelPendingDelete(model IVirtualModel, ctx context.Context, userCred mcclient.TokenCredential) error {
	return model.DoCancelPendingDelete(ctx, userCred)
}*/

func (model *SVirtualResourceBase) PerformCancelDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if model.PendingDeleted {
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

func (model *SVirtualResourceBase) CancelPendingDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	if model.PendingDeleted {
		return model.MarkCancelPendingDelete(ctx, userCred)
	}
	return nil
}

func (model *SVirtualResourceBase) MarkCancelPendingDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	ownerProjId := model.GetOwnerProjectId()

	lockman.LockClass(ctx, model.GetModelManager(), ownerProjId)
	defer lockman.ReleaseClass(ctx, model.GetModelManager(), ownerProjId)

	newName, err := GenerateName(model.GetModelManager(), ownerProjId, model.Name)
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

func (model *SVirtualResourceBase) SyncCloudProjectId(userCred mcclient.TokenCredential, projectId string) {
	if model.ProjectSrc != string(PROJECT_SOURCE_LOCAL) && len(projectId) > 0 {
		diff, _ := Update(model, func() error {
			model.ProjectSrc = string(PROJECT_SOURCE_CLOUD)
			if len(projectId) > 0 {
				model.ProjectId = projectId
			}
			return nil
		})
		if len(diff) > 0 {
			OpsLog.LogEvent(model, ACT_SYNC_OWNER, diff, userCred)
		}
	}
}
