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
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SInfrasResourceBaseManager struct {
	SDomainLevelResourceBaseManager
	SSharableBaseResourceManager
}

func NewInfrasResourceBaseManager(
	dt interface{},
	tableName string,
	keyword string,
	keywordPlural string,
) SInfrasResourceBaseManager {
	return SInfrasResourceBaseManager{
		SDomainLevelResourceBaseManager: NewDomainLevelResourceBaseManager(dt, tableName, keyword, keywordPlural),
	}
}

type SInfrasResourceBase struct {
	SDomainLevelResourceBase
	SSharableBaseResource `"is_public=>create":"domain_optional" "public_scope=>create":"domain_optional"`
}

func (manager *SInfrasResourceBaseManager) GetIInfrasModelManager() IInfrasModelManager {
	return manager.GetVirtualObject().(IInfrasModelManager)
}

func (manager *SInfrasResourceBaseManager) FilterByOwner(q *sqlchemy.SQuery, owner mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	return SharableManagerFilterByOwner(manager.GetIInfrasModelManager(), q, owner, scope)
}

func (model *SInfrasResourceBase) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return ((model.IsOwner(userCred) || model.IsSharable(userCred)) && IsAllowGet(rbacutils.ScopeDomain, userCred, model)) || IsAllowGet(rbacutils.ScopeSystem, userCred, model)
}

func (model *SInfrasResourceBase) IsSharable(reqUsrId mcclient.IIdentityProvider) bool {
	return SharableModelIsSharable(model.GetIInfrasModel(), reqUsrId)
}

func (model *SInfrasResourceBase) IsShared() bool {
	return SharableModelIsShared(model)
}

func (model *SInfrasResourceBase) AllowPerformPublic(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformPublicDomainInput) bool {
	return true
}

func (model *SInfrasResourceBase) PerformPublic(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformPublicDomainInput) (jsonutils.JSONObject, error) {
	err := SharablePerformPublic(model.GetIInfrasModel(), ctx, userCred, apis.PerformPublicProjectInput{PerformPublicDomainInput: input})
	if err != nil {
		return nil, errors.Wrap(err, "SharablePerformPublic")
	}
	return nil, nil
}

func (model *SInfrasResourceBase) AllowPerformPrivate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformPrivateInput) bool {
	return true
}

func (model *SInfrasResourceBase) PerformPrivate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformPrivateInput) (jsonutils.JSONObject, error) {
	err := SharablePerformPrivate(model.GetIInfrasModel(), ctx, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "SharablePerformPrivate")
	}
	return nil, nil
}

func (model *SInfrasResourceBase) GetIInfrasModel() IInfrasModel {
	return model.GetVirtualObject().(IInfrasModel)
}

func (manager *SInfrasResourceBaseManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input apis.InfrasResourceBaseCreateInput,
) (apis.InfrasResourceBaseCreateInput, error) {
	var err error
	input.DomainLevelResourceCreateInput, err = manager.SDomainLevelResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.DomainLevelResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "manager.SDomainLevelResourceBaseManager.ValidateCreateData")
	}
	input.SharableResourceBaseCreateInput, err = SharableManagerValidateCreateData(manager.GetIInfrasModelManager(), ctx, userCred, ownerId, query, input.SharableResourceBaseCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SharableManagerValidateCreateData")
	}
	return input, nil
}

func (manager *SInfrasResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.InfrasResourceBaseListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SDomainLevelResourceBaseManager.ListItemFilter(ctx, q, userCred, query.DomainLevelResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SDomainLevelResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SSharableBaseResourceManager.ListItemFilter(ctx, q, userCred, query.SharableResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableBaseResourceManager.ListItemFilter")
	}
	return q, nil
}

func (manager *SInfrasResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := manager.SDomainLevelResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SInfrasResourceBaseManager) OrderByExtraFields(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query apis.InfrasResourceBaseListInput) (*sqlchemy.SQuery, error) {
	q, err := manager.SDomainLevelResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.DomainLevelResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SDomainLevelResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SInfrasResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []apis.InfrasResourceBaseDetails {
	rows := make([]apis.InfrasResourceBaseDetails, len(objs))

	domainRows := manager.SDomainLevelResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	shareRows := manager.SSharableBaseResourceManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = apis.InfrasResourceBaseDetails{
			DomainLevelResourceDetails: domainRows[i],
			SharableResourceBaseInfo:   shareRows[i],
		}
	}

	return rows
}

func (model *SInfrasResourceBase) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input apis.InfrasResourceBaseUpdateInput,
) (apis.InfrasResourceBaseUpdateInput, error) {
	var err error
	input.DomainLevelResourceBaseUpdateInput, err = model.SDomainLevelResourceBase.ValidateUpdateData(ctx, userCred, query, input.DomainLevelResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SDomainLevelResourceBase.ValidateUpdateData")
	}
	return input, nil
}

func (model *SInfrasResourceBase) GetSharedDomains() []string {
	return SharableGetSharedProjects(model, SharedTargetDomain)
}

func (model *SInfrasResourceBase) PerformChangeOwner(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input apis.PerformChangeDomainOwnerInput,
) (jsonutils.JSONObject, error) {
	if !consts.GetNonDefaultDomainProjects() {
		return nil, errors.Wrap(httperrors.ErrForbidden, "not allow to change owner of domain resource if non_default_domain_projects is turned off")
	}
	if model.IsShared() {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "cannot change owner when shared!")
	}
	return model.SDomainLevelResourceBase.PerformChangeOwner(ctx, userCred, query, input)
}

func (model *SInfrasResourceBase) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	SharableModelCustomizeCreate(model.GetIInfrasModel(), ctx, userCred, ownerId, query, data)
	return model.SDomainLevelResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (model *SInfrasResourceBase) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	SharedResourceManager.CleanModelShares(ctx, userCred, model.GetIInfrasModel())
	return model.SDomainLevelResourceBase.Delete(ctx, userCred)
}

func (model *SInfrasResourceBase) GetSharedInfo() apis.SShareInfo {
	ret := apis.SShareInfo{}
	ret.IsPublic = model.IsPublic
	ret.PublicScope = rbacutils.String2ScopeDefault(model.PublicScope, rbacutils.ScopeNone)
	ret.SharedDomains = model.GetSharedDomains()
	ret.SharedProjects = nil
	// fix
	if len(ret.SharedDomains) > 0 {
		ret.PublicScope = rbacutils.ScopeDomain
		ret.SharedProjects = nil
		ret.IsPublic = true
	} else if !ret.IsPublic {
		ret.PublicScope = rbacutils.ScopeNone
	}
	return ret
}

func (model *SInfrasResourceBase) SaveSharedInfo(src apis.TOwnerSource, ctx context.Context, userCred mcclient.TokenCredential, si apis.SShareInfo) {
	diff, _ := Update(model, func() error {
		model.PublicSrc = string(src)
		model.IsPublic = si.IsPublic
		model.PublicScope = string(si.PublicScope)
		return nil
	})
	if len(diff) > 0 {
		OpsLog.LogEvent(model, ACT_SYNC_SHARE, diff, userCred)
	}
	SharedResourceManager.shareToTarget(ctx, userCred, model.GetIInfrasModel(), SharedTargetProject, nil, nil, nil)
	SharedResourceManager.shareToTarget(ctx, userCred, model.GetIInfrasModel(), SharedTargetDomain, si.SharedDomains, nil, nil)
}

func (model *SInfrasResourceBase) SyncShareState(ctx context.Context, userCred mcclient.TokenCredential, shareInfo apis.SAccountShareInfo) {
	if !consts.GetNonDefaultDomainProjects() {
		if model.PublicSrc != string(apis.OWNER_SOURCE_LOCAL) {
			model.SaveSharedInfo(apis.OWNER_SOURCE_CLOUD, ctx, userCred, apis.SShareInfo{
				IsPublic:    true,
				PublicScope: rbacutils.ScopeSystem,
			})
		}
		return
	}
	si := shareInfo.GetDomainShareInfo()
	if model.PublicSrc != string(apis.OWNER_SOURCE_LOCAL) {
		model.SaveSharedInfo(apis.OWNER_SOURCE_CLOUD, ctx, userCred, si)
	} else {
		localSi := model.GetSharedInfo()
		if localSi.IsViolate(si) {
			newSi := localSi.Intersect(si)
			newSi.FixDomainShare()
			// reset to cloud base public_src
			model.SaveSharedInfo(apis.OWNER_SOURCE_CLOUD, ctx, userCred, newSi)
		}
	}
}

func (model *SInfrasResourceBase) GetSharableTargetDomainIds() []string {
	return model.GetIInfrasModel().GetChangeOwnerCandidateDomainIds()
}

func (model *SInfrasResourceBase) GetRequiredSharedDomainIds() []string {
	return []string{model.DomainId}
}

/*func (model *SInfrasResourceBase) ValidateDeleteCondition(ctx context.Context) error {
	if model.IsShared() {
		return httperrors.NewForbiddenError("%s %s is shared", model.Keyword(), model.Name)
	}
	return model.SDomainLevelResourceBase.ValidateDeleteCondition(ctx)
}*/
