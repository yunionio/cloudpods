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
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SDomainLevelResourceBaseManager struct {
	SStandaloneResourceBaseManager
	SDomainizedResourceBaseManager
}

func NewDomainLevelResourceBaseManager(
	dt interface{},
	tableName string,
	keyword string,
	keywordPlural string,
) SDomainLevelResourceBaseManager {
	return SDomainLevelResourceBaseManager{
		SStandaloneResourceBaseManager: NewStandaloneResourceBaseManager(dt, tableName, keyword, keywordPlural),
	}
}

type SDomainLevelResourceBase struct {
	SStandaloneResourceBase
	SDomainizedResourceBase

	// 归属Domain信息的来源, local: 本地设置, cloud: 从云上同步过来
	// example: local
	DomainSrc string `width:"10" charset:"ascii" nullable:"true" list:"user" default:"" json:"domain_src"`
}

func (manager *SDomainLevelResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.DomainLevelResourceListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SDomainizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.DomainizedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SDomainizedResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

func (manager *SDomainLevelResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SDomainizedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SDomainLevelResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.DomainLevelResourceListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SDomainizedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.DomainizedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SDomainizedResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (model *SDomainLevelResourceBase) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	model.DomainId = ownerId.GetProjectDomainId()
	model.DomainSrc = string(apis.OWNER_SOURCE_LOCAL)
	return model.SStandaloneResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (model *SDomainLevelResourceBase) PerformChangeOwner(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformChangeDomainOwnerInput) (jsonutils.JSONObject, error) {
	if !consts.GetNonDefaultDomainProjects() {
		return nil, errors.Wrap(httperrors.ErrForbidden, "not allow to change owner of domain resource if non_default_domain_projects is turned off")
	}
	if model.GetIStandaloneModel().IsShared() {
		return nil, errors.Wrap(httperrors.ErrForbidden, "cannot change owner of shared resource")
	}

	manager := model.GetModelManager()

	data := jsonutils.Marshal(input)
	log.Debugf("SDomainLevelResourceBase change_owner %s %s %#v", query, data, manager)
	ownerId, err := manager.FetchOwnerId(ctx, data)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	if len(ownerId.GetProjectDomainId()) == 0 {
		return nil, httperrors.NewInputParameterError("missing new domain")
	}
	if ownerId.GetProjectDomainId() == model.DomainId {
		// do nothing
		Update(model, func() error {
			model.DomainSrc = string(apis.OWNER_SOURCE_LOCAL)
			return nil
		})
		return nil, nil
	}

	// change domain, do check
	candidates := model.GetIDomainLevelModel().GetChangeOwnerCandidateDomainIds()
	if len(candidates) > 0 && !utils.IsInStringArray(ownerId.GetProjectDomainId(), candidates) {
		return nil, errors.Wrap(httperrors.ErrForbidden, "target domain not in change owner candidate list")
	}
	requires := model.GetIDomainLevelModel().GetChangeOwnerRequiredDomainIds()
	log.Debugf("%s required domains: %s", model.Keyword(), requires)
	if len(requires) > 0 && !utils.IsInStringArray(ownerId.GetProjectDomainId(), requires) {
		return nil, errors.Wrap(httperrors.ErrForbidden, "target domain not in change owner required list")
	}

	if !IsAdminAllowPerform(ctx, userCred, model, "change-owner") {
		return nil, errors.Wrap(httperrors.ErrNotSufficientPrivilege, "require system privileges")
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
	former, _ := TenantCacheManager.FetchDomainById(ctx, model.DomainId)
	if former == nil {
		log.Warningf("domain_id %s not found", model.DomainId)
		formerObj := NewDomain(model.DomainId, "unknown")
		former = &formerObj
	}

	// clean shared projects before update domain id
	if sharedModel, ok := model.GetIDomainLevelModel().(ISharableBaseModel); ok {
		if err := SharedResourceManager.CleanModelShares(ctx, userCred, sharedModel); err != nil {
			return nil, err
		}
	}

	// cancel usage
	model.cleanModelUsages(ctx, userCred)

	_, err = Update(model, func() error {
		model.DomainId = ownerId.GetProjectDomainId()
		model.DomainSrc = string(apis.OWNER_SOURCE_LOCAL)
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "Update")
	}

	// add usage
	model.RecoverUsages(ctx, userCred)

	OpsLog.SyncOwner(model, former, userCred)
	notes := struct {
		OldDomainId string
		OldDomain   string
		NewDomainId string
		NewDomain   string
	}{
		OldDomainId: former.DomainId,
		OldDomain:   former.Domain,
		NewDomainId: ownerId.GetProjectDomainId(),
		NewDomain:   ownerId.GetProjectDomain(),
	}
	logclient.AddActionLogWithContext(ctx, model, logclient.ACT_CHANGE_OWNER, notes, userCred, true)
	return nil, nil
}

func (manager *SDomainLevelResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []apis.DomainLevelResourceDetails {
	rows := make([]apis.DomainLevelResourceDetails, len(objs))
	stdRows := manager.SStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	domainRows := manager.SDomainizedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = apis.DomainLevelResourceDetails{
			StandaloneResourceDetails: stdRows[i],
			DomainizedResourceInfo:    domainRows[i],
		}
	}
	return rows
}

func (manager *SDomainLevelResourceBaseManager) GetIDomainLevelModelManager() IDomainLevelModelManager {
	return manager.GetVirtualObject().(IDomainLevelModelManager)
}

func (manager *SDomainLevelResourceBaseManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input apis.DomainLevelResourceCreateInput,
) (apis.DomainLevelResourceCreateInput, error) {
	var err error
	input.StandaloneResourceCreateInput, err = manager.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StandaloneResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SStandaloneResourceBaseManager.ValidateCreateData")
	}
	return input, nil
}

func (model *SDomainLevelResourceBase) DomainLevelModelManager() IDomainLevelModelManager {
	return model.GetModelManager().(IDomainLevelModelManager)
}

func (model *SDomainLevelResourceBase) IsOwner(userCred mcclient.TokenCredential) bool {
	return model.DomainId == userCred.GetProjectDomainId()
}

func (model *SDomainLevelResourceBase) SyncCloudDomainId(userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider) {
	if model.DomainSrc != string(apis.OWNER_SOURCE_LOCAL) && ownerId != nil && len(ownerId.GetProjectDomainId()) > 0 {
		diff, _ := Update(model, func() error {
			model.DomainSrc = string(apis.OWNER_SOURCE_CLOUD)
			model.DomainId = ownerId.GetProjectDomainId()
			return nil
		})
		if len(diff) > 0 {
			OpsLog.LogEvent(model, ACT_SYNC_OWNER, diff, userCred)
		}
	}
}

func (model *SDomainLevelResourceBase) GetIDomainLevelModel() IDomainLevelModel {
	return model.GetVirtualObject().(IDomainLevelModel)
}

func (model *SDomainLevelResourceBase) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input apis.DomainLevelResourceBaseUpdateInput,
) (apis.DomainLevelResourceBaseUpdateInput, error) {
	var err error
	input.StandaloneResourceBaseUpdateInput, err = model.SStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, input.StandaloneResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SStandaloneResourceBase.ValidateUpdateData")
	}
	return input, nil
}

func (model *SDomainLevelResourceBase) GetDetailsChangeOwnerCandidateDomains(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (apis.ChangeOwnerCandidateDomainsOutput, error) {
	return IOwnerResourceBaseModelGetChangeOwnerCandidateDomains(model.GetIDomainLevelModel())
}

func (manager *SDomainLevelResourceBaseManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.ListItemExportKeys")
	}
	q, err = manager.SDomainizedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SDomainizedResourceBaseManager.ListItemExportKeys")
	}
	return q, nil
}

func (manager *SDomainLevelResourceBaseManager) GetPropertyDomainTagValuePairs(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	return GetPropertyTagValuePairs(
		manager.GetIDomainLevelModelManager(),
		"domain",
		"domain_id",
		ctx,
		userCred,
		query,
	)
}

func (manager *SDomainLevelResourceBaseManager) GetPropertyDomainTagValueTree(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	return GetPropertyTagValueTree(
		manager.GetIDomainLevelModelManager(),
		"domain",
		"domain_id",
		ctx,
		userCred,
		query,
	)
}
