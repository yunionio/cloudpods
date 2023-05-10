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
	"yunion.io/x/pkg/util/rbacscope"
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
	PendingDeletedAt time.Time `json:"pending_deleted_at" list:"user" update:"admin"`
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

func (manager *SVirtualResourceBaseManager) GetPropertyStatistics(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*apis.StatusStatistic, error) {
	im, ok := manager.GetVirtualObject().(IModelManager)
	if !ok {
		im = manager
	}

	var err error
	q := manager.Query()
	q, err = ListItemQueryFilters(im, ctx, q, userCred, query, policy.PolicyActionList)
	if err != nil {
		return nil, err
	}

	sq := q.SubQuery()
	statQ := sq.Query(sq.Field("status"), sqlchemy.COUNT("total_count", sq.Field("id")), sqlchemy.SUM("pending_deleted_count", sq.Field("pending_deleted")))
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
	result := &apis.StatusStatistic{
		StatusInfo: []apis.StatusStatisticStatusInfo{},
	}
	for _, s := range ret {
		result.StatusInfo = append(result.StatusInfo, apis.StatusStatisticStatusInfo{
			Status:              s.Status,
			TotalCount:          s.TotalCount,
			PendingDeletedCount: s.PendingDeletedCount,
		})
	}
	return result, nil
}

func (manager *SVirtualResourceBaseManager) GetPropertyProjectStatistics(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) ([]apis.ProjectStatistic, error) {
	im, ok := manager.GetVirtualObject().(IModelManager)
	if !ok {
		im = manager
	}

	tenants := TenantCacheManager.GetTenantQuery().Equals("domain_id", userCred.GetProjectDomainId()).SubQuery()

	_q, err := ListItemQueryFilters(im, ctx, im.Query(), userCred, query, policy.PolicyActionList)
	if err != nil {
		return nil, err
	}

	sq := _q.SubQuery()

	q := sq.Query(
		sq.Field("tenant_id"),
		sqlchemy.COUNT("count"),
	).GroupBy(sq.Field("tenant_id"))

	q.Join(tenants, sqlchemy.Equals(q.Field("tenant_id"), tenants.Field("id")))

	q.AppendField(tenants.Field("id"))
	q.AppendField(tenants.Field("name"))

	result := []apis.ProjectStatistic{}
	return result, q.All(&result)
}

func (manager *SVirtualResourceBaseManager) GetPropertyDomainStatistics(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) ([]apis.ProjectStatistic, error) {
	im, ok := manager.GetVirtualObject().(IModelManager)
	if !ok {
		im = manager
	}

	domains := TenantCacheManager.GetDomainQuery().SubQuery()

	_q, err := ListItemQueryFilters(im, ctx, im.Query(), userCred, query, policy.PolicyActionList)
	if err != nil {
		return nil, err
	}

	sq := _q.SubQuery()

	q := sq.Query(
		sq.Field("domain_id"),
		sqlchemy.COUNT("count"),
	).GroupBy(sq.Field("domain_id"))

	q.Join(domains, sqlchemy.Equals(q.Field("domain_id"), domains.Field("id")))

	q.AppendField(domains.Field("id"))
	q.AppendField(domains.Field("name"))

	result := []apis.ProjectStatistic{}
	return result, q.All(&result)
}

func (manager *SVirtualResourceBaseManager) FilterByHiddenSystemAttributes(q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	q = manager.SStatusStandaloneResourceBaseManager.FilterByHiddenSystemAttributes(q, userCred, query, scope)

	isSystem := jsonutils.QueryBoolean(query, "system", false)
	if isSystem {
		var isAllow bool
		allowScope, result := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionList, "system")
		if result.Result.IsAllow() && !scope.HigherThan(allowScope) {
			isAllow = true
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

func (manager *SVirtualResourceBaseManager) FilterBySystemAttributes(q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	q = manager.SStatusStandaloneResourceBaseManager.FilterBySystemAttributes(q, userCred, query, scope)

	var pendingDelete string
	if query != nil {
		pendingDelete, _ = query.GetString("pending_delete")
	}
	pendingDeleteLower := strings.ToLower(pendingDelete)
	if pendingDeleteLower == "all" || pendingDeleteLower == "any" || utils.ToBool(pendingDeleteLower) {
		var isAllow bool
		allowScope, result := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionList, "pending_delete")
		if result.Result.IsAllow() && !scope.HigherThan(allowScope) {
			isAllow = true
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
	if input.IsSystem != nil && *input.IsSystem && IsAdminAllowCreate(userCred, manager).Result.IsDeny() {
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

func (model *SVirtualResourceBase) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	project, err := model.GetTenantCache(ctx)
	if err != nil {
		log.Errorf("unable to GetTenantCache: %s", err.Error())
		return
	}
	err = InheritFromTo(ctx, project, model)
	if err != nil {
		log.Errorf("unable to inherit class metadata from poject %s: %s", project.GetId(), err.Error())
	}
	model.SStatusStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
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

func (model *SVirtualResourceBase) GetTenantCache(ctx context.Context) (*STenant, error) {
	// log.Debugf("Get tenant by Id %s", model.ProjectId)
	return TenantCacheManager.FetchTenantById(ctx, model.ProjectId)
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

	var requireScope rbacscope.TRbacScope
	if ownerId.GetProjectDomainId() != model.DomainId {
		// change domain, do check
		candidates := model.GetIVirtualModel().GetChangeOwnerCandidateDomainIds()
		if len(candidates) > 0 && !utils.IsInStringArray(ownerId.GetProjectDomainId(), candidates) {
			return nil, errors.Wrap(httperrors.ErrForbidden, "target domain not in change owner candidate list")
		}
		requireScope = rbacscope.ScopeSystem
	} else {
		requireScope = rbacscope.ScopeDomain
	}

	allowScope, policyTags := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), model.KeywordPlural(), policy.PolicyActionPerform, "change-owner")
	if requireScope.HigherThan(allowScope) {
		return nil, errors.Wrapf(httperrors.ErrNotSufficientPrivilege, "require %s allow %s", requireScope, allowScope)
	}

	err = objectConfirmPolicyTags(ctx, model, policyTags)
	if err != nil {
		return nil, errors.Wrap(err, "objectConfirmPolicyTags")
	}

	q := manager.Query().Equals("name", model.GetName())
	q = manager.FilterByOwner(q, manager, userCred, ownerId, manager.NamespaceScope())
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
	} else {
		// check fromer's class metadata
		cm, err := former.GetAllClassMetadata()
		if err != nil {
			return nil, errors.Wrap(err, "unable to GetAllClassMetadata")
		}
		if len(cm) > 0 {
			return nil, httperrors.NewForbiddenError("can't change owner for resource in project with class metadata")
		}
	}

	toer, err := TenantCacheManager.FetchTenantById(ctx, ownerId.GetProjectId())
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get project %s", ownerId.GetProjectId())
	}
	toCm, err := toer.GetAllClassMetadata()
	if err != nil {
		return nil, errors.Wrap(err, "unable to GetAllClassMetadata")
	}
	if model.Keyword() != "image" && len(toCm) > 0 {
		return nil, httperrors.NewForbiddenError("can't change resource's owner as that in project with class metadata")
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

	// set class metadata

	err = model.SetClassMetadataAll(ctx, toCm, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "unable to SetClassMetadataAll")
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

	lockman.LockRawObject(ctx, manager.Keyword(), "name")
	defer lockman.ReleaseRawObject(ctx, manager.Keyword(), "name")

	newName, err := GenerateName(ctx, manager, ownerId, model.Name)
	if err != nil {
		return errors.Wrapf(err, "GenerateNam")
	}
	diff, err := Update(model, func() error {
		model.Name = newName
		model.PendingDeleted = false
		model.PendingDeletedAt = time.Time{}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "MarkCancelPendingDelete.Update")
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
		metadata, _ := GetVisiableMetadata(ctx, tc, nil)
		desc.Set("project_tags", jsonutils.Marshal(metadata))
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

	return q, nil
}

func (manager *SVirtualResourceBaseManager) GetExportExtraKeys(ctx context.Context, keys stringutils2.SSortedStrings, rowMap map[string]string) *jsonutils.JSONDict {
	res := manager.SStatusStandaloneResourceBaseManager.GetExportExtraKeys(ctx, keys, rowMap)
	if userTags, ok := rowMap["user_tags"]; ok && len(userTags) > 0 {
		res.Set("user_tags", jsonutils.NewString(userTags))
	}
	return res
}

func (manager *SVirtualResourceBaseManager) GetPropertyProjectTagValuePairs(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	return GetPropertyTagValuePairs(
		manager.GetIVirtualModelManager(),
		"project",
		"tenant_id",
		ctx,
		userCred,
		query,
	)
}

func (manager *SVirtualResourceBaseManager) GetPropertyProjectTagValueTree(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	return GetPropertyTagValueTree(
		manager.GetIVirtualModelManager(),
		"project",
		"tenant_id",
		ctx,
		userCred,
		query,
	)
}
