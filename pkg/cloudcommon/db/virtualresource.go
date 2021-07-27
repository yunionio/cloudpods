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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SVirtualResourceBaseManager struct {
	SStatusStandaloneResourceBaseManager
	SProjectizedResourceBaseManager
}

func NewVirtualResourceBaseManager(dt interface{}, tableName string, keyword string, keywordPlural string) SVirtualResourceBaseManager {
	return SVirtualResourceBaseManager{
		SStatusStandaloneResourceBaseManager: NewStatusStandaloneResourceBaseManager(dt,
			tableName, keyword, keywordPlural),
	}
}

type SVirtualResourceBase struct {
	SStatusStandaloneResourceBase
	SProjectizedResourceBase

	// 云上同步资源是否在本地被更改过配置, local: 更改过, cloud: 未更改过
	// example: local
	ProjectSrc string `width:"10" charset:"ascii" nullable:"false" list:"user" default:"" json:"project_src"`

	// 是否是系统资源
	IsSystem bool `nullable:"true" default:"false" list:"admin" create:"optional" json:"is_system"`

	// 资源放入回收站时间
	PendingDeletedAt time.Time `json:"pending_deleted_at" list:"user"`
	// 资源是否处于回收站中
	PendingDeleted bool `nullable:"false" default:"false" index:"true" get:"user" list:"user" json:"pending_deleted"`
	// 资源是否被冻结
	Freezed bool `nullable:"false" default:"false" get:"user" list:"user" json:"freezed"`
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

/*func (manager *SVirtualResourceBaseManager) FilterByName(q *sqlchemy.SQuery, name string) *sqlchemy.SQuery {
	q = manager.SStatusStandaloneResourceBaseManager.FilterByName(q, name)
	return q
}*/

func (manager *SVirtualResourceBaseManager) AllowGetPropertyStatistics(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return IsAdminAllowGetSpec(userCred, manager, "statistics")
}

func (manager *SVirtualResourceBaseManager) GetPropertyStatistics(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (map[string]apis.StatusStatistic, error) {
	im, ok := manager.GetVirtualObject().(IModelManager)
	if !ok {
		im = manager
	}

	var err error
	q := manager.Query()
	q, err = ListItemFilter(im, ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}

	sq := q.SubQuery()
	statQ := sq.Query(sq.Field("status"), sqlchemy.COUNT("total_count", sq.Field("id")), sqlchemy.SUM("pending_deleted_count", sq.Field("pending_deleted")))
	_, queryScope, err := FetchCheckQueryOwnerScope(ctx, userCred, query, manager, rbacutils.ActionList, true)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	statQ = manager.FilterByOwner(statQ, userCred, queryScope)
	statQ = manager.FilterBySystemAttributes(statQ, userCred, query, queryScope)
	statQ = manager.FilterByHiddenSystemAttributes(statQ, userCred, query, queryScope)
	statQ = statQ.GroupBy(sq.Field("status"))
	ret := []struct {
		Status              string
		TotalCount          int64
		PendingDeletedCount int64
	}{}
	err = statQ.All(&ret)
	if err != nil {
		return nil, errors.Wrapf(err, "q.All")
	}
	type sStatistic struct {
		TotalCount          int64
		PendingDeletedCount int64
	}
	result := map[string]apis.StatusStatistic{}
	for _, s := range ret {
		result[s.Status] = apis.StatusStatistic{
			TotalCount:          s.TotalCount,
			PendingDeletedCount: s.PendingDeletedCount,
		}
	}
	return result, nil
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

func (model *SVirtualResourceBase) SetSystemInfo(isSystem bool) error {
	_, err := Update(model, func() error {
		model.IsSystem = isSystem
		return nil
	})
	return err
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

func (manager *SVirtualResourceBaseManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input apis.VirtualResourceCreateInput) (apis.VirtualResourceCreateInput, error) {
	var err error
	if input.IsSystem != nil && *input.IsSystem && !IsAdminAllowCreate(userCred, manager) {
		return input, httperrors.NewNotSufficientPrivilegeError("non-admin user not allowed to create system object")
	}
	input.StatusStandaloneResourceCreateInput, err = manager.SStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StatusStandaloneResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ValidateCreateData")
	}
	return input, nil
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
	model.ProjectSrc = string(apis.OWNER_SOURCE_LOCAL)
	return model.SStatusStandaloneResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
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

func (model *SVirtualResourceBase) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (apis.VirtualResourceDetails, error) {
	return apis.VirtualResourceDetails{}, nil
}

func (manager *SVirtualResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []apis.VirtualResourceDetails {
	ret := make([]apis.VirtualResourceDetails, len(objs))
	stdRows := manager.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	projRows := manager.SProjectizedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range objs {
		ret[i] = apis.VirtualResourceDetails{
			StatusStandaloneResourceDetails: stdRows[i],
			ProjectizedResourceInfo:         projRows[i],
		}
	}
	return ret
}

func (model *SVirtualResourceBase) PreCheckPerformAction(
	ctx context.Context, userCred mcclient.TokenCredential,
	action string, query jsonutils.JSONObject, data jsonutils.JSONObject,
) error {
	if err := model.SStandaloneResourceBase.PreCheckPerformAction(ctx, userCred, action, query, data); err != nil {
		return err
	}
	if model.Freezed && action != "unfreeze" {
		return httperrors.NewBadRequestError("Virtual resource freezed, can't do %s", action)
	}
	return nil
}

func (model *SVirtualResourceBase) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return !model.Freezed && (model.IsOwner(userCred) || IsAdminAllowUpdate(userCred, model))
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

func (model *SVirtualResourceBase) AllowPerformFreeze(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformFreezeInput) bool {
	return IsAdminAllowPerform(userCred, model, "freeze")
}

// freezed update and perform action operation except for unfreeze
func (model *SVirtualResourceBase) PerformFreeze(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformFreezeInput) (jsonutils.JSONObject, error) {
	if model.Freezed {
		return nil, httperrors.NewBadRequestError("virtual resource already freezed")
	}
	_, err := Update(model, func() error {
		model.Freezed = true
		return nil
	})
	if err != nil {
		return nil, err
	}
	OpsLog.LogEvent(model, ACT_FREEZE, "perform freeze", userCred)
	logclient.AddActionLogWithContext(ctx, model, logclient.ACT_FREEZE, "perform freeze", userCred, true)
	return nil, nil
}

func (model *SVirtualResourceBase) AllowPerformUnfreeze(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformUnfreezeInput) bool {
	return IsAdminAllowPerform(userCred, model, "unfreeze")
}

func (model *SVirtualResourceBase) PerformUnfreeze(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformUnfreezeInput) (jsonutils.JSONObject, error) {
	if !model.Freezed {
		return nil, httperrors.NewBadRequestError("virtual resource not freezed")
	}
	_, err := Update(model, func() error {
		model.Freezed = false
		return nil
	})
	if err != nil {
		return nil, err
	}
	OpsLog.LogEvent(model, ACT_UNFREEZE, "perform unfreeze", userCred)
	logclient.AddActionLogWithContext(ctx, model, logclient.ACT_UNFREEZE, "perform unfreeze", userCred, true)
	return nil, nil
}

func (model *SVirtualResourceBase) AllowPerformChangeOwner(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformChangeProjectOwnerInput) bool {
	return IsAdminAllowPerform(userCred, model, "change-owner")
}

func (model *SVirtualResourceBase) PerformChangeOwner(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformChangeProjectOwnerInput) (jsonutils.JSONObject, error) {
	if model.GetIStandaloneModel().IsShared() {
		return nil, errors.Wrap(httperrors.ErrForbidden, "cannot change owner of shared resource")
	}

	manager := model.GetModelManager()

	data := jsonutils.Marshal(input)
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
			model.ProjectSrc = string(apis.OWNER_SOURCE_LOCAL)
			return nil
		})
		return nil, nil
	}

	var requireScope rbacutils.TRbacScope
	if ownerId.GetProjectDomainId() != model.DomainId {
		// change domain, do check
		candidates := model.GetIVirtualModel().GetChangeOwnerCandidateDomainIds()
		if len(candidates) > 0 && !utils.IsInStringArray(ownerId.GetProjectDomainId(), candidates) {
			return nil, errors.Wrap(httperrors.ErrForbidden, "target domain not in change owner candidate list")
		}
		requireScope = rbacutils.ScopeSystem
	} else {
		requireScope = rbacutils.ScopeDomain
	}

	allowScope := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), model.KeywordPlural(), policy.PolicyActionPerform, "change-owner")
	if requireScope.HigherThan(allowScope) {
		return nil, errors.Wrapf(httperrors.ErrNotSufficientPrivilege, "require %s allow %s", requireScope, allowScope)
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
	if sharedModel, ok := model.GetIVirtualModel().(ISharableBaseModel); ok {
		if err := SharedResourceManager.CleanModelShares(ctx, userCred, sharedModel); err != nil {
			return nil, err
		}
	}

	// cancel usage
	model.cleanModelUsages(ctx, userCred)

	_, err = Update(model, func() error {
		model.DomainId = ownerId.GetProjectDomainId()
		model.ProjectId = ownerId.GetProjectId()
		model.ProjectSrc = string(apis.OWNER_SOURCE_LOCAL)
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "Update")
	}

	// add usage
	model.RecoverUsages(ctx, userCred)

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
	return DeleteModel(ctx, userCred, model.GetIVirtualModel())
}

func (model *SVirtualResourceBase) AllowPerformCancelDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (model *SVirtualResourceBase) PerformCancelDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if model.PendingDeleted && !model.Deleted {
		err := model.DoCancelPendingDelete(ctx, userCred)
		if err != nil {
			return nil, errors.Wrap(err, "model.DoCancelPendingDelete")
		}
		model.RecoverUsages(ctx, userCred)
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
		err := model.MarkCancelPendingDelete(ctx, userCred)
		if err != nil {
			return errors.Wrap(err, "MarkCancelPendingDelete")
		}
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
	if model.ProjectSrc != string(apis.OWNER_SOURCE_LOCAL) && ownerId != nil && len(ownerId.GetProjectId()) > 0 {
		diff, _ := Update(model, func() error {
			model.ProjectSrc = string(apis.OWNER_SOURCE_CLOUD)
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

func (manager *SVirtualResourceBaseManager) OrderByExtraFields(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query apis.VirtualResourceListInput) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, err
	}
	q, err = manager.SProjectizedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ProjectizedResourceListInput)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (manager *SVirtualResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.VirtualResourceListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SProjectizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ProjectizedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SProjectizedResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

func (manager *SVirtualResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SProjectizedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (model *SVirtualResourceBase) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input apis.VirtualResourceBaseUpdateInput,
) (apis.VirtualResourceBaseUpdateInput, error) {
	var err error
	input.StatusStandaloneResourceBaseUpdateInput, err = model.SStatusStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, input.StatusStandaloneResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SStatusStandaloneResourceBase.ValidateUpdateData")
	}
	return input, nil
}

func (model *SVirtualResourceBase) AllowGetDetailsChangeOwnerCandidateDomains(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return model.IsOwner(userCred) || IsAdminAllowGetSpec(userCred, model, "change-owner-candidate-domains")
}

func (model *SVirtualResourceBase) GetDetailsChangeOwnerCandidateDomains(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (apis.ChangeOwnerCandidateDomainsOutput, error) {
	return IOwnerResourceBaseModelGetChangeOwnerCandidateDomains(model.GetIVirtualModel())
}

func (manager *SVirtualResourceBaseManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusStandaloneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemExportKeys")
	}
	q, err = manager.SProjectizedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SProjectizedResourceBaseManager.ListItemExportKeys")
	}

	if keys.Contains("user_tags") {
		guestUserTagsQuery := Metadata.Query().Startswith("id", manager.keyword+"::").
			Startswith("key", USER_TAG_PREFIX).GroupBy("id")
		guestUserTagsQuery.AppendField(sqlchemy.SubStr("resource_id", guestUserTagsQuery.Field("id"), len(manager.keyword)+3, 0))
		guestUserTagsQuery.AppendField(
			sqlchemy.GROUP_CONCAT("user_tags", sqlchemy.CONCAT("",
				sqlchemy.SubStr("", guestUserTagsQuery.Field("key"), len(USER_TAG_PREFIX)+1, 0),
				sqlchemy.NewStringField(":"),
				guestUserTagsQuery.Field("value"),
			)))
		subQ := guestUserTagsQuery.SubQuery()
		q.LeftJoin(subQ, sqlchemy.Equals(q.Field("id"), subQ.Field("resource_id")))
		q.AppendField(subQ.Field("user_tags"))
	}

	return q, nil
}

func (manager *SVirtualResourceBaseManager) GetExportExtraKeys(ctx context.Context, keys stringutils2.SSortedStrings, rowMap map[string]string) *jsonutils.JSONDict {
	res := manager.SStatusStandaloneResourceBaseManager.GetExportExtraKeys(ctx, keys, rowMap)
	if userTags, ok := rowMap["user_tags"]; ok && len(userTags) > 0 {
		res.Set("user_tags", jsonutils.NewString(userTags))
	}
	return res
}
