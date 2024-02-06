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
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-ignore
type IIdentityModelManager interface {
	db.IStandaloneModelManager

	GetIIdentityModelManager() IIdentityModelManager
}

type IIdentityModel interface {
	db.IStandaloneModel
	db.IPendingDeletable

	GetDomain() *SDomain

	GetIIdentityModelManager() IIdentityModelManager

	GetIIdentityModel() IIdentityModel
}

type IEnabledIdentityModelManager interface {
	IIdentityModelManager

	GetIEnabledIdentityModelManager() IEnabledIdentityModelManager
}

type IEnabledIdentityModel interface {
	IIdentityModel

	db.IEnabledBaseInterface

	GetIEnabledIdentityModelManager() IEnabledIdentityModelManager

	GetIEnabledIdentityModel() IEnabledIdentityModel
}

// +onecloud:swagger-gen-ignore
type SIdentityBaseResourceManager struct {
	db.SStandaloneResourceBaseManager
	db.SDomainizedResourceBaseManager
	db.SPendingDeletedBaseManager
}

func NewIdentityBaseResourceManager(dt interface{}, tableName string, keyword string, keywordPlural string) SIdentityBaseResourceManager {
	return SIdentityBaseResourceManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(dt, tableName, keyword, keywordPlural),
	}
}

type SIdentityBaseResource struct {
	db.SStandaloneResourceBase
	db.SDomainizedResourceBase
	db.SPendingDeletedBase

	// 额外信息
	Extra *jsonutils.JSONDict `nullable:"true"`
}

// +onecloud:swagger-gen-ignore
type SEnabledIdentityBaseResourceManager struct {
	SIdentityBaseResourceManager
	db.SEnabledResourceBaseManager
}

func NewEnabledIdentityBaseResourceManager(dt interface{}, tableName string, keyword string, keywordPlural string) SEnabledIdentityBaseResourceManager {
	return SEnabledIdentityBaseResourceManager{
		SIdentityBaseResourceManager: NewIdentityBaseResourceManager(dt, tableName, keyword, keywordPlural),
	}
}

type SEnabledIdentityBaseResource struct {
	SIdentityBaseResource

	db.SEnabledResourceBase `"enabled->default":"true" "enabled->list":"user" "enabled->update":"domain" "enabled->create":"domain_optional"`
}

func (model *SIdentityBaseResource) GetIIdentityModelManager() IIdentityModelManager {
	return model.GetModelManager().(IIdentityModelManager)
}

func (model *SIdentityBaseResource) GetIIdentityModel() IIdentityModel {
	return model.GetVirtualObject().(IIdentityModel)
}

func (model *SIdentityBaseResource) GetDomain() *SDomain {
	if len(model.DomainId) > 0 && model.DomainId != api.KeystoneDomainRoot {
		domain, err := DomainManager.FetchDomainById(model.DomainId)
		if err != nil {
			log.Errorf("GetDomain fail %s", err)
		}
		return domain
	}
	return nil
}

func (manager *SIdentityBaseResourceManager) GetIIdentityModelManager() IIdentityModelManager {
	return manager.GetVirtualObject().(IIdentityModelManager)
}

func (manager *SIdentityBaseResourceManager) FetchByName(ctx context.Context, userCred mcclient.IIdentityProvider, idStr string) (db.IModel, error) {
	return db.FetchByName(ctx, manager.GetIIdentityModelManager(), userCred, idStr)
}

func (manager *SIdentityBaseResourceManager) FetchByIdOrName(ctx context.Context, userCred mcclient.IIdentityProvider, idStr string) (db.IModel, error) {
	return db.FetchByIdOrName(ctx, manager.GetIIdentityModelManager(), userCred, idStr)
}

func (manager *SIdentityBaseResourceManager) FilterBySystemAttributes(q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	q = manager.SStandaloneResourceBaseManager.FilterBySystemAttributes(q, userCred, query, scope)
	q = manager.SPendingDeletedBaseManager.FilterBySystemAttributes(manager.GetIStandaloneModelManager(), q, userCred, query, scope)
	return q
}

func (manager *SIdentityBaseResourceManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.IdentityBaseResourceListInput,
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

func (manager *SEnabledIdentityBaseResourceManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.EnabledIdentityBaseResourceListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SIdentityBaseResourceManager.ListItemFilter(ctx, q, userCred, query.IdentityBaseResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SIdentityBaseResourceManager.ListItemFilter")
	}
	q, err = manager.SEnabledResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

func (manager *SIdentityBaseResourceManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.IdentityBaseResourceListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, err
	}
	orderByDomain := query.OrderByDomain
	if sqlchemy.SQL_ORDER_ASC.Equals(orderByDomain) || sqlchemy.SQL_ORDER_DESC.Equals(orderByDomain) {
		domains := DomainManager.Query().SubQuery()
		q = q.LeftJoin(domains, sqlchemy.Equals(q.Field("domain_id"), domains.Field("id")))
		if sqlchemy.SQL_ORDER_ASC.Equals(orderByDomain) {
			q = q.Asc(domains.Field("name"))
		} else {
			q = q.Desc(domains.Field("name"))
		}
	}
	return q, nil
}

func (manager *SEnabledIdentityBaseResourceManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.EnabledIdentityBaseResourceListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SIdentityBaseResourceManager.OrderByExtraFields(ctx, q, userCred, query.IdentityBaseResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SIdentityBaseResourceManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SIdentityBaseResourceManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	if field == "domain" {
		domainQuery := DomainManager.Query("name", "id").Distinct().SubQuery()
		q.AppendField(domainQuery.Field("name", "domain"))
		q = q.Join(domainQuery, sqlchemy.Equals(q.Field("domain_id"), domainQuery.Field("id")))
		q.GroupBy(domainQuery.Field("name"))
		return q, nil
	}
	q, err = manager.SStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SEnabledIdentityBaseResourceManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := manager.SIdentityBaseResourceManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SIdentityBaseResourceManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.IdentityBaseResourceCreateInput) (api.IdentityBaseResourceCreateInput, error) {
	domain, _ := DomainManager.FetchDomainById(ownerId.GetProjectDomainId())
	if domain.Enabled.IsFalse() {
		return input, httperrors.NewInvalidStatusError("domain is disabled")
	}
	var err error
	input.StandaloneResourceCreateInput, err = manager.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StandaloneResourceCreateInput)
	if err != nil {
		return input, err
	}
	return input, nil
}

func (manager *SEnabledIdentityBaseResourceManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.EnabledIdentityBaseResourceCreateInput) (api.EnabledIdentityBaseResourceCreateInput, error) {
	var err error
	input.IdentityBaseResourceCreateInput, err = manager.SIdentityBaseResourceManager.ValidateCreateData(ctx, userCred, ownerId, query, input.IdentityBaseResourceCreateInput)
	if err != nil {
		return input, err
	}
	return input, nil
}

func (model *SIdentityBaseResource) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.IdentityBaseUpdateInput,
) (api.IdentityBaseUpdateInput, error) {
	var err error
	input.StandaloneResourceBaseUpdateInput, err = model.SStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, input.StandaloneResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SStandaloneResourceBase.ValidateUpdateData")
	}
	return input, nil
}

func (model *SEnabledIdentityBaseResource) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.EnabledIdentityBaseUpdateInput,
) (api.EnabledIdentityBaseUpdateInput, error) {
	var err error
	input.IdentityBaseUpdateInput, err = model.SIdentityBaseResource.ValidateUpdateData(ctx, userCred, query, input.IdentityBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SIdentityBaseResource.ValidateUpdateData")
	}
	return input, nil
}

/*func(manager *SIdentityBaseResourceManager) NamespaceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeSystem
}*/

func (manager *SIdentityBaseResourceManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.IdentityBaseResourceDetails {
	rows := make([]api.IdentityBaseResourceDetails, len(objs))

	stdRows := manager.SStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	domainRows := manager.SDomainizedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.IdentityBaseResourceDetails{
			StandaloneResourceDetails: stdRows[i],
			DomainizedResourceInfo:    domainRows[i],
		}
	}

	return rows
}

func (manager *SEnabledIdentityBaseResourceManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.EnabledIdentityBaseResourceDetails {
	rows := make([]api.EnabledIdentityBaseResourceDetails, len(objs))

	identRows := manager.SIdentityBaseResourceManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.EnabledIdentityBaseResourceDetails{
			IdentityBaseResourceDetails: identRows[i],
		}
	}

	return rows
}

func (model *SIdentityBaseResource) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	model.DomainId = ownerId.GetProjectDomainId()
	return model.SStandaloneResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

/*
func (self *SIdentityBaseResource) ValidateDeleteCondition(ctx context.Context) error {
	return self.SStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SIdentityBaseResource) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return self.SStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}
*/

func (ident *SEnabledIdentityBaseResource) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	if ident.Enabled.IsTrue() {
		return httperrors.NewResourceBusyError("resource is enabled")
	}
	return ident.SIdentityBaseResource.ValidateDeleteCondition(ctx, nil)
}

func (model *SIdentityBaseResource) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	model.SStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
}

func (model *SIdentityBaseResource) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	model.SStandaloneResourceBase.PostUpdate(ctx, userCred, query, data)
}

func (model *SIdentityBaseResource) PostDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	model.SStandaloneResourceBase.PostDelete(ctx, userCred)
}

func (manager *SIdentityBaseResourceManager) totalCount(scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider) int {
	q := manager.Query()
	if scope != rbacscope.ScopeSystem {
		q = q.Equals("domain_id", ownerId.GetProjectDomainId())
	}
	cnt, _ := q.CountWithError()
	return cnt
}

func (manager *SIdentityBaseResourceManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
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

func (manager *SIdentityBaseResourceManager) GetPropertyDomainTagValuePairs(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	return db.GetPropertyTagValuePairs(
		manager.GetIIdentityModelManager(),
		"domain",
		"domain_id",
		ctx,
		userCred,
		query,
	)
}

func (manager *SIdentityBaseResourceManager) GetPropertyDomainTagValueTree(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	return db.GetPropertyTagValueTree(
		manager.GetIIdentityModelManager(),
		"domain",
		"domain_id",
		"",
		ctx,
		userCred,
		query,
	)
}

func (model *SIdentityBaseResource) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	if !model.PendingDeleted {
		newName := model.Name
		if !strings.Contains(model.Name, "-deleted-") {
			newName = fmt.Sprintf("%s-deleted-%s", model.Name, timeutils.ShortDate(timeutils.UtcNow()))
		}
		err := model.SPendingDeletedBase.MarkPendingDelete(model.GetIStandaloneModel(), ctx, userCred, newName)
		if err != nil {
			return errors.Wrap(err, "MarkPendingDelete")
		}
	}
	err := db.Metadata.RemoveAll(ctx, model, userCred)
	if err != nil {
		return errors.Wrapf(err, "Metadata.RemoveAll")
	}
	return nil // DeleteModel(ctx, userCred, model.GetIVirtualModel())
}

func (model *SIdentityBaseResource) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	if !model.PendingDeleted {
		err := model.SPendingDeletedBase.MarkPendingDelete(model.GetIStandaloneModel(), ctx, userCred, "")
		if err != nil {
			return errors.Wrap(err, "MarkPendingDelete")
		}
	}
	return db.DeleteModel(ctx, userCred, model.GetIIdentityModel())
}

func (manager *SEnabledIdentityBaseResourceManager) GetIEnabledIdentityModelManager() IEnabledIdentityModelManager {
	return manager.GetVirtualObject().(IEnabledIdentityModelManager)
}

func (model *SEnabledIdentityBaseResource) GetIEnabledIdentityModelManager() IEnabledIdentityModelManager {
	return model.GetModelManager().(IEnabledIdentityModelManager)
}

func (model *SEnabledIdentityBaseResource) GetIEnabledIdentityModel() IEnabledIdentityModel {
	return model.GetVirtualObject().(IEnabledIdentityModel)
}

// 启用资源
func (model *SEnabledIdentityBaseResource) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformEnableInput) (jsonutils.JSONObject, error) {
	err := db.EnabledPerformEnable(model.GetIEnabledIdentityModel(), ctx, userCred, true)
	if err != nil {
		return nil, errors.Wrap(err, "EnabledPerformEnable")
	}
	return nil, nil
}

// 禁用资源
func (model *SEnabledIdentityBaseResource) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformDisableInput) (jsonutils.JSONObject, error) {
	err := db.EnabledPerformEnable(model.GetIEnabledIdentityModel(), ctx, userCred, false)
	if err != nil {
		return nil, errors.Wrap(err, "EnabledPerformEnable")
	}
	return nil, nil
}

func (model *SIdentityBaseResource) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	desc := model.SStandaloneAnonResourceBase.GetShortDesc(ctx)
	if model.DomainId != api.KeystoneDomainRoot {
		desc.Add(jsonutils.NewString(model.DomainId), "domain_id")
		domain := model.GetIIdentityModel().GetDomain()
		if domain != nil {
			desc.Add(jsonutils.NewString(domain.Name), "domain")
		}
	}
	return desc
}

func (model *SEnabledIdentityBaseResource) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	desc := model.SIdentityBaseResource.GetShortDesc(ctx)
	desc.Add(jsonutils.NewBool(model.Enabled.Bool()), "enabled")
	return desc
}
