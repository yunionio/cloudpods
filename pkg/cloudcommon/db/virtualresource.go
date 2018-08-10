package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/log"
	"github.com/yunionio/pkg/util/timeutils"
	"github.com/yunionio/pkg/utils"
	"github.com/yunionio/sqlchemy"

	"github.com/yunionio/onecloud/pkg/cloudcommon/db/lockman"
	"github.com/yunionio/onecloud/pkg/httperrors"
	"github.com/yunionio/onecloud/pkg/mcclient"
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

	ProjectId string `name:"tenant_id" width:"128" charset:"ascii" nullable:"false" index:"true" list:"admin"`

	IsSystem bool `nullable:"true" default:"false" list:"admin"`

	PendingDeletedAt time.Time ``
	PendingDeleted   bool      `nullable:"false" default:"false" index:"true" get:"admin"`
}

func (model *SVirtualResourceBase) IsOwner(userCred mcclient.TokenCredential) bool {
	return userCred.IsSystemAdmin() || userCred.GetProjectId() == model.ProjectId
}

func (model *SVirtualResourceBase) IsAdmin(userCred mcclient.TokenCredential) bool {
	return userCred.IsSystemAdmin() || (userCred.GetProjectId() == model.ProjectId && userCred.IsAdmin())
}

func (model *SVirtualResourceBase) GetOwnerProjectId() string {
	return model.ProjectId
}

func (manager *SVirtualResourceBaseManager) FilterByOwner(q *sqlchemy.SQuery, ownerProjId string) *sqlchemy.SQuery {
	q = q.Equals("tenant_id", ownerProjId)
	q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(q.Field("pending_deleted")), sqlchemy.IsFalse(q.Field("pending_deleted"))))
	q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(q.Field("is_system")), sqlchemy.IsFalse(q.Field("is_system"))))
	return q
}

func (manager *SVirtualResourceBaseManager) FetchByName(ownerProjId string, idStr string) (IModel, error) {
	return fetchByName(manager, ownerProjId, idStr)
}

func (manager *SVirtualResourceBaseManager) FetchByIdOrName(ownerProjId string, idStr string) (IModel, error) {
	return fetchByIdOrName(manager, ownerProjId, idStr)
}

func (manager *SVirtualResourceBaseManager) GetOwnerId(userCred mcclient.TokenCredential) string {
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
				return nil, httperrors.NewTenantNotFoundError(fmt.Sprintf("tenant %s not found", tenant))
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
	if err == nil && isSystem && !userCred.IsSystemAdmin() {
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
	return model.SStandaloneResourceBase.CustomizeCreate(ctx, userCred, ownerProjId, query, data)
}

func (manager *SVirtualResourceBaseManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	isAdmin, err := query.Bool("admin")
	if err == nil && isAdmin && !userCred.IsSystemAdmin() {
		return false
	}
	return true
}

func (model *SVirtualResourceBase) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return model.IsOwner(userCred)
}

func (manager *SVirtualResourceBaseManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return true
}

func (model *SVirtualResourceBase) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return model.IsOwner(userCred)
}

func (model *SVirtualResourceBase) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return model.IsOwner(userCred)
}

func (model *SVirtualResourceBase) AllowGetDetailsMetadata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return model.IsOwner(userCred)
}

func (model *SVirtualResourceBase) AllowPerformMetadata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return model.IsOwner(userCred)
}

func (model *SVirtualResourceBase) GetTenantCache(ctx context.Context) (*STenant, error) {
	log.Debugf("Get tenant by Id %s", model.ProjectId)
	return TenantCacheManager.FetchTenantById(ctx, model.ProjectId)
}

func (model *SVirtualResourceBase) getMoreDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	if userCred.IsSystemAdmin() {
		log.Debugf("GetCustomizeColumns")
		tobj, err := model.GetTenantCache(ctx)
		if err == nil {
			log.Debugf("GetTenantFromCache %s", jsonutils.Marshal(tobj))
			extra.Add(jsonutils.NewString(tobj.GetName()), "tenant")
		} else {
			log.Errorf("GetTenantCache fail %s", err)
		}
	}
	return extra
}

func (model *SVirtualResourceBase) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := model.SStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return model.getMoreDetails(ctx, userCred, query, extra)
}

func (model *SVirtualResourceBase) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := model.SStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	return model.getMoreDetails(ctx, userCred, query, extra)
}

func (model *SVirtualResourceBase) AllowPerformChangeOwner(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return userCred.IsSystemAdmin()
}

func (model *SVirtualResourceBase) PerformChangeOwner(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	tenant := jsonutils.GetAnyString(data, []string{"project", "tenant", "project_id", "tenant_id"})
	if len(tenant) == 0 {
		return nil, httperrors.NewInputParameterError("missing parameter tenant")
	}
	tobj, _ := TenantCacheManager.FetchTenantByIdOrName(ctx, tenant)
	if tobj == nil {
		return nil, httperrors.NewTenantNotFoundError(fmt.Sprintf("tenant %s not found", tenant))
	}
	q := model.GetModelManager().Query().Equals("name", model.GetName())
	q = q.Equals("tenant_id", tobj.GetId())
	q = q.NotEquals("id", model.GetId())
	if q.Count() > 0 {
		return nil, httperrors.NewConflictError(fmt.Sprintf("duplicate name %s", model.GetName()))
	}
	former, _ := TenantCacheManager.FetchTenantById(ctx, model.ProjectId)
	if former == nil {
		log.Errorf("tenant_id %s not found", model.ProjectId)
		formerObj := NewTenant(model.ProjectId, "unknown")
		former = &formerObj
	}
	_, err := model.GetModelManager().TableSpec().Update(model, func() error {
		model.ProjectId = tobj.GetId()
		return nil
	})
	if err != nil {
		return nil, err
	}
	OpsLog.SyncOwner(model, former, userCred)
	return nil, nil
}

func (model *SVirtualResourceBase) DoPendingDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := model.PendingDelete()
	if err == nil {
		OpsLog.LogEvent(model, ACT_PENDING_DELETE, nil, userCred)
	}
	return err
}

func (model *SVirtualResourceBase) PendingDelete() error {
	_, err := model.GetModelManager().TableSpec().Update(model, func() error {
		model.PendingDeleted = true
		model.PendingDeletedAt = timeutils.UtcNow()
		return nil
	})
	if err != nil {
		log.Errorf("PendingDelete fail %s", err)
	}
	return err
}

func (model *SVirtualResourceBase) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	if !model.PendingDeleted {
		model.DoPendingDelete(ctx, userCred)
	}
	return model.SStandaloneResourceBase.Delete(ctx, userCred)
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
		OpsLog.LogEvent(model, ACT_CANCEL_DELETE, nil, userCred)
	}
	return err
}

func (model *SVirtualResourceBase) VirtualModelManager() IVirtualModelManager {
	return model.GetModelManager().(IVirtualModelManager)
}

func (model *SVirtualResourceBase) CancelPendingDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	ownerProjId := model.GetOwnerProjectId()
	lockman.LockClass(ctx, model.GetModelManager(), ownerProjId)
	defer lockman.ReleaseClass(ctx, model.GetModelManager(), ownerProjId)
	_, err := model.GetModelManager().TableSpec().Update(model, func() error {
		model.Name = GenerateName(model.GetModelManager(), ownerProjId, model.Name)
		model.PendingDeleted = false
		return nil
	})
	return err
}
