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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SSharableVirtualResourceBase struct {
	SVirtualResourceBase
	SSharableBaseResource `"is_public=>create":"optional" "public_scope=>create":"optional"`
	// IsPublic    bool   `default:"false" nullable:"false" create:"domain_optional" list:"user" json:"is_public"`
	// PublicScope string `width:"16" charset:"ascii" nullable:"false" default:"system" create:"domain_optional" list:"user" json:"public_scope"`
}

type SSharableVirtualResourceBaseManager struct {
	SVirtualResourceBaseManager
	SSharableBaseResourceManager
}

func NewSharableVirtualResourceBaseManager(dt interface{}, tableName string, keyword string, keywordPlural string) SSharableVirtualResourceBaseManager {
	return SSharableVirtualResourceBaseManager{SVirtualResourceBaseManager: NewVirtualResourceBaseManager(dt, tableName, keyword, keywordPlural)}
}

func (manager *SSharableVirtualResourceBaseManager) GetISharableVirtualModelManager() ISharableVirtualModelManager {
	return manager.GetVirtualObject().(ISharableVirtualModelManager)
}

func (manager *SSharableVirtualResourceBaseManager) FilterByOwner(q *sqlchemy.SQuery, owner mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	return SharableManagerFilterByOwner(manager.GetISharableVirtualModelManager(), q, owner, scope)
}

func (model *SSharableVirtualResourceBase) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return model.IsOwner(userCred) || model.IsSharable(userCred) || IsAllowGet(rbacutils.ScopeSystem, userCred, model)
}

func (model *SSharableVirtualResourceBase) IsSharable(reqUsrId mcclient.IIdentityProvider) bool {
	return SharableModelIsSharable(model.GetISharableVirtualModel(), reqUsrId)
}

func (model *SSharableVirtualResourceBase) IsShared() bool {
	return SharableModelIsShared(model)
}

func (model *SSharableVirtualResourceBase) AllowPerformPublic(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformPublicProjectInput) bool {
	return true
}

func (model *SSharableVirtualResourceBase) PerformPublic(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformPublicProjectInput) (jsonutils.JSONObject, error) {
	err := SharablePerformPublic(model.GetISharableVirtualModel(), ctx, userCred, input)
	if err != nil {
		return nil, errors.Wrap(err, "SharablePerformPublic")
	}
	return nil, nil
}

func (model *SSharableVirtualResourceBase) AllowPerformPrivate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformPrivateInput) bool {
	return true
}

func (model *SSharableVirtualResourceBase) PerformPrivate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformPrivateInput) (jsonutils.JSONObject, error) {
	err := SharablePerformPrivate(model.GetISharableVirtualModel(), ctx, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "SharablePerformPrivate")
	}
	return nil, nil
}

func (model *SSharableVirtualResourceBase) GetISharableVirtualModel() ISharableVirtualModel {
	return model.GetVirtualObject().(ISharableVirtualModel)
}

func (manager *SSharableVirtualResourceBaseManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input apis.SharableVirtualResourceCreateInput) (apis.SharableVirtualResourceCreateInput, error) {
	var err error
	input.VirtualResourceCreateInput, err = manager.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.VirtualResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "manager.VirtualResourceBaseManager.ValidateCreateData")
	}
	input.SharableResourceBaseCreateInput, err = SharableManagerValidateCreateData(manager.GetISharableVirtualModelManager(), ctx, userCred, ownerId, query, input.SharableResourceBaseCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SharableManagerValidateCreateData")
	}
	return input, nil
}

func (manager *SSharableVirtualResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.SharableVirtualResourceListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SSharableBaseResourceManager.ListItemFilter(ctx, q, userCred, query.SharableResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableBaseResourceManager.ListItemFilter")
	}
	return q, nil
}

func (manager *SSharableVirtualResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := manager.SVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SSharableVirtualResourceBaseManager) OrderByExtraFields(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query apis.SharableVirtualResourceListInput) (*sqlchemy.SQuery, error) {
	q, err := manager.SVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (model *SSharableVirtualResourceBase) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (apis.SharableVirtualResourceDetails, error) {
	return apis.SharableVirtualResourceDetails{}, nil
}

func (manager *SSharableVirtualResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []apis.SharableVirtualResourceDetails {
	rows := make([]apis.SharableVirtualResourceDetails, len(objs))

	virtRows := manager.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	shareRows := manager.SSharableBaseResourceManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = apis.SharableVirtualResourceDetails{
			VirtualResourceDetails:   virtRows[i],
			SharableResourceBaseInfo: shareRows[i],
		}
	}

	return rows
}

func (model *SSharableVirtualResourceBase) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input apis.SharableVirtualResourceBaseUpdateInput,
) (apis.SharableVirtualResourceBaseUpdateInput, error) {
	var err error
	input.VirtualResourceBaseUpdateInput, err = model.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input.VirtualResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SVirtualResourceBase.ValidateUpdateData")
	}
	return input, nil
}

func (model *SSharableVirtualResourceBase) GetSharedProjects() []string {
	return SharableGetSharedProjects(model, SharedTargetProject)
}

func (model *SSharableVirtualResourceBase) GetSharedDomains() []string {
	return SharableGetSharedProjects(model, SharedTargetDomain)
}

func (model *SSharableVirtualResourceBase) PerformChangeOwner(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input apis.PerformChangeProjectOwnerInput,
) (jsonutils.JSONObject, error) {
	if model.IsShared() {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "cannot change owner when shared!")
	}
	return model.SVirtualResourceBase.PerformChangeOwner(ctx, userCred, query, input)
}

func (model *SSharableVirtualResourceBase) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	SharableModelCustomizeCreate(model.GetISharableVirtualModel(), ctx, userCred, ownerId, query, data)
	return model.SVirtualResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (model *SSharableVirtualResourceBase) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	SharedResourceManager.CleanModelShares(ctx, userCred, model.GetISharableVirtualModel())
	return model.SVirtualResourceBase.Delete(ctx, userCred)
}

func (model *SSharableVirtualResourceBase) GetSharedInfo() apis.SShareInfo {
	ret := apis.SShareInfo{}
	ret.IsPublic = model.IsPublic
	ret.PublicScope = rbacutils.String2ScopeDefault(model.PublicScope, rbacutils.ScopeNone)
	ret.SharedDomains = model.GetSharedDomains()
	ret.SharedProjects = model.GetSharedProjects()
	// fix
	if len(ret.SharedDomains) > 0 {
		ret.PublicScope = rbacutils.ScopeDomain
		ret.SharedProjects = nil
		ret.IsPublic = true
	} else if len(ret.SharedProjects) > 0 {
		ret.PublicScope = rbacutils.ScopeProject
		ret.SharedDomains = nil
		ret.IsPublic = true
	} else if !ret.IsPublic {
		ret.PublicScope = rbacutils.ScopeNone
	}
	return ret
}

func (model *SSharableVirtualResourceBase) SaveSharedInfo(src apis.TOwnerSource, ctx context.Context, userCred mcclient.TokenCredential, si apis.SShareInfo) {
	diff, _ := Update(model, func() error {
		model.PublicSrc = string(src)
		model.IsPublic = si.IsPublic
		model.PublicScope = string(si.PublicScope)
		return nil
	})
	if len(diff) > 0 {
		OpsLog.LogEvent(model, ACT_SYNC_SHARE, diff, userCred)
	}
	SharedResourceManager.shareToTarget(ctx, userCred, model.GetISharableVirtualModel(), SharedTargetProject, si.SharedProjects, nil, nil)
	SharedResourceManager.shareToTarget(ctx, userCred, model.GetISharableVirtualModel(), SharedTargetDomain, si.SharedDomains, nil, nil)
}

func (model *SSharableVirtualResourceBase) SyncShareState(ctx context.Context, userCred mcclient.TokenCredential, shareInfo apis.SAccountShareInfo) {
	si := shareInfo.GetProjectShareInfo()
	if model.PublicSrc != string(apis.OWNER_SOURCE_LOCAL) {
		model.SaveSharedInfo(apis.OWNER_SOURCE_CLOUD, ctx, userCred, si)
	} else {
		localSi := model.GetSharedInfo()
		if localSi.IsViolate(si) {
			newSi := localSi.Intersect(si)
			newSi.FixProjectShare()
			// reset to cloud base public_src
			model.SaveSharedInfo(apis.OWNER_SOURCE_CLOUD, ctx, userCred, newSi)
		}
	}
}

func (model *SSharableVirtualResourceBase) GetSharableTargetDomainIds() []string {
	return model.GetISharableVirtualModel().GetChangeOwnerCandidateDomainIds()
}

func (model *SSharableVirtualResourceBase) GetRequiredSharedDomainIds() []string {
	return []string{model.DomainId}
}

/*func (model *SSharableVirtualResourceBase) ValidateDeleteCondition(ctx context.Context) error {
	if model.IsShared() {
		return httperrors.NewForbiddenError("%s %s is shared", model.Keyword(), model.Name)
	}
	return model.SVirtualResourceBase.ValidateDeleteCondition(ctx)
}*/
