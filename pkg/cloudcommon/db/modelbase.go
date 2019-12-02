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
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/object"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SModelBase struct {
	object.SObject

	manager IModelManager `ignore:"true"` // pointer to modelmanager
}

type SModelBaseManager struct {
	object.SObject

	tableSpec     *sqlchemy.STableSpec
	keyword       string
	keywordPlural string
	alias         string
	aliasPlural   string
}

func NewModelBaseManager(model interface{}, tableName string, keyword string, keywordPlural string) SModelBaseManager {
	ts := sqlchemy.NewTableSpecFromStruct(model, tableName)
	modelMan := SModelBaseManager{tableSpec: ts, keyword: keyword, keywordPlural: keywordPlural}
	return modelMan
}

func (manager *SModelBaseManager) GetIModelManager() IModelManager {
	virt := manager.GetVirtualObject()
	if virt == nil {
		panic(fmt.Sprintf("Forgot to call SetVirtualObject?"))
	}
	r, ok := virt.(IModelManager)
	if !ok {
		panic(fmt.Sprintf("Cannot convert virtual object to IModelManager: %#v", virt))
	}
	return r
}

func (manager *SModelBaseManager) SetAlias(alias string, aliasPlural string) {
	manager.alias = alias
	manager.aliasPlural = aliasPlural
}

func (manager *SModelBaseManager) TableSpec() *sqlchemy.STableSpec {
	return manager.tableSpec
}

func (manager *SModelBaseManager) Keyword() string {
	return manager.keyword
}

func (manager *SModelBaseManager) KeywordPlural() string {
	return manager.keywordPlural
}

func (manager *SModelBaseManager) GetContextManagers() [][]IModelManager {
	return nil
}

func (manager *SModelBaseManager) Alias() string {
	return manager.alias
}

func (manager *SModelBaseManager) AliasPlural() string {
	return manager.aliasPlural
}

func (manager *SModelBaseManager) ValidateName(name string) error {
	return nil
}

func (model *SModelBase) MarkDeletePreventionOn() {
	return
}

func (model *SModelBase) MarkDeletePreventionOff() {
	return
}

// list hooks
func (manager *SModelBaseManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return false
}

func (manager *SModelBaseManager) ValidateListConditions(ctx context.Context, userCred mcclient.TokenCredential, query *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return query, nil
}

func (manager *SModelBaseManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	return q, nil
}

func (manager *SModelBaseManager) ListItemFilterV2(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *apis.ModelBaseListInput) (*sqlchemy.SQuery, error) {
	return q, nil
}

func (manager *SModelBaseManager) CustomizeFilterList(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*CustomizeListFilters, error) {
	return NewCustomizeListFilters(), nil
}

func (manager *SModelBaseManager) ExtraSearchConditions(ctx context.Context, q *sqlchemy.SQuery, like string) []sqlchemy.ICondition {
	return nil
}

// fetch hook
func (manager *SModelBaseManager) getTable() *sqlchemy.STable {
	return manager.tableSpec.Instance()
}

func (manager *SModelBaseManager) Query(fieldNames ...string) *sqlchemy.SQuery {
	instance := manager.getTable()
	fields := make([]sqlchemy.IQueryField, len(fieldNames))
	for i, f := range fieldNames {
		fields[i] = instance.Field(f)
	}
	return instance.Query(fields...)
}

func (manager *SModelBaseManager) FilterById(q *sqlchemy.SQuery, idStr string) *sqlchemy.SQuery {
	return q
}

func (manager *SModelBaseManager) FilterByNotId(q *sqlchemy.SQuery, idStr string) *sqlchemy.SQuery {
	return q
}

func (manager *SModelBaseManager) FilterByName(q *sqlchemy.SQuery, name string) *sqlchemy.SQuery {
	return q
}

func (manager *SModelBaseManager) FilterByOwner(q *sqlchemy.SQuery, ownerId mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	return q
}

func (manager *SModelBaseManager) FilterBySystemAttributes(q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	return q
}

func (manager *SModelBaseManager) FilterByHiddenSystemAttributes(q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	return q
}

func (manager *SModelBaseManager) FilterByParentId(q *sqlchemy.SQuery, parentId string) *sqlchemy.SQuery {
	return q
}

func (manager *SModelBaseManager) FetchById(idStr string) (IModel, error) {
	return nil, sql.ErrNoRows
}

func (manager *SModelBaseManager) FetchByName(userCred mcclient.IIdentityProvider, idStr string) (IModel, error) {
	return nil, sql.ErrNoRows
}

func (manager *SModelBaseManager) FetchByIdOrName(userCred mcclient.IIdentityProvider, idStr string) (IModel, error) {
	return nil, sql.ErrNoRows
}

// create hooks
func (manager *SModelBaseManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (manager *SModelBaseManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input apis.ModelBaseCreateInput) (apis.ModelBaseCreateInput, error) {
	return input, nil
}

func (manager *SModelBaseManager) OnCreateComplete(ctx context.Context, items []IModel, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	// do nothing
}

func (manager *SModelBaseManager) AllowPerformAction(ctx context.Context, userCred mcclient.TokenCredential, action string, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (manager *SModelBaseManager) PerformAction(ctx context.Context, userCred mcclient.TokenCredential, action string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, httperrors.NewActionNotFoundError("Action %s not found", action)
}

func (manager *SModelBaseManager) AllowPerformCheckCreateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return IsAllowClassPerform(rbacutils.ScopeSystem, userCred, manager, "check-create-data")
}

func (manager *SModelBaseManager) InitializeData() error {
	return nil
}

func (manager *SModelBaseManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q = q.AppendField(q.QueryFields()...)
	return q, nil
}

func (manager *SModelBaseManager) OrderByExtraFields(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	return q, nil
}

func (manager *SModelBaseManager) GetExportExtraKeys(ctx context.Context, query jsonutils.JSONObject, rowMap map[string]string) *jsonutils.JSONDict {
	return jsonutils.NewDict()
}

func (manager *SModelBaseManager) CustomizeHandlerInfo(info *appsrv.SHandlerInfo) {
	info.SetProcessTimeoutCallback(manager.GetIModelManager().SetHandlerProcessTimeout)
}

func (manager *SModelBaseManager) SetHandlerProcessTimeout(info *appsrv.SHandlerInfo, r *http.Request) time.Duration {
	if r.Method == http.MethodGet && len(r.URL.Query().Get("export_keys")) > 0 {
		return time.Hour * 2
	}
	return -time.Second
}

func (manager *SModelBaseManager) FetchCreateHeaderData(ctx context.Context, header http.Header) (jsonutils.JSONObject, error) {
	return nil, nil
}

func (manager *SModelBaseManager) FetchUpdateHeaderData(ctx context.Context, header http.Header) (jsonutils.JSONObject, error) {
	return nil, nil
}

func (manager *SModelBaseManager) IsCustomizedGetDetailsBody() bool {
	return false
}

func (manager *SModelBaseManager) ListSkipLog(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return false
}

func (manager *SModelBaseManager) GetSkipLog(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return false
}

func (manager *SModelBaseManager) FetchCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, objs []IModel, fields stringutils2.SSortedStrings) []*jsonutils.JSONDict {
	ret := make([]*jsonutils.JSONDict, len(objs))
	for i := range objs {
		ret[i] = jsonutils.NewDict()
	}
	return ret
}

func (manager *SModelBaseManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	return nil, nil
}

func (manager *SModelBaseManager) FetchParentId(ctx context.Context, data jsonutils.JSONObject) string {
	return ""
}

func (manager *SModelBaseManager) NamespaceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeSystem
}

func (manager *SModelBaseManager) ResourceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeSystem
}

func (manager *SModelBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	return q, httperrors.NewBadRequestError("unsupport field %s", field)
}

func (manager *SModelBaseManager) AllowGetPropertyDistinctField(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (manager *SModelBaseManager) GetPagingConfig() *SPagingConfig {
	return nil
}

func (manager *SModelBaseManager) GetPropertyDistinctField(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	im, ok := manager.GetVirtualObject().(IModelManager)
	if !ok {
		im = manager
	}
	fn, err := query.GetArray("field")
	efs, _ := query.GetArray("extra_field")
	fields := make([]string, len(fn))

	// validate field
	for i, f := range fn {
		fields[i], err = f.GetString()
		if err != nil {
			return nil, httperrors.NewInputParameterError("can't get string field")
		}
		var hasField = false
		for _, field := range manager.getTable().Fields() {
			if field.Name() == fields[i] {
				hasField = true
				break
			}
		}
		if !hasField {
			return nil, httperrors.NewBadRequestError("model has no field %s", fields[i])
		}
	}

	var res = jsonutils.NewDict()
	q := im.Query()
	q, err = ListItemQueryFilters(im, ctx, q, userCred, query, policy.PolicyActionList)
	if err != nil {
		return nil, err
	}
	var backupQuery = *q

	// query field
	for i := 0; i < len(fields); i++ {
		var nq = backupQuery
		nq.AppendField(nq.Field(fields[i]))
		of, err := nq.Distinct().AllStringMap()
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil && err != sql.ErrNoRows {
			return nil, httperrors.NewInternalServerError("Query database error %s", err)
		}
		ofa := make([]string, len(of))
		for j := 0; j < len(of); j++ {
			ofa[j] = of[j][fields[i]]
		}
		res.Set(fields[i], jsonutils.Marshal(ofa))
	}

	// query extra field
	for i := 0; i < len(efs); i++ {
		nq := backupQuery
		fe, _ := efs[i].GetString()
		nqp, err := im.QueryDistinctExtraField(&nq, fe)
		if err != nil {
			return nil, err
		}
		ef, err := nqp.AllStringMap()
		if err == sql.ErrNoRows {
			continue
		}
		efa := make([]string, len(ef))
		for i := 0; i < len(ef); i++ {
			efa[i] = ef[i][fe]
		}
		if err != nil && err != sql.ErrNoRows {
			return nil, httperrors.NewInternalServerError("Query database error %s", err)
		}
		res.Set(fe, jsonutils.Marshal(efa))
	}
	return res, nil
}

func (manager *SModelBaseManager) BatchPreValidate(
	ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject, data *jsonutils.JSONDict, count int,
) (func(), error) {
	return nil, nil
}

func (manager *SModelBaseManager) BatchCreateValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, nil
}

func (model *SModelBase) GetId() string {
	return ""
}

func (model *SModelBase) Keyword() string {
	return model.GetModelManager().Keyword()
}

func (model *SModelBase) KeywordPlural() string {
	return model.GetModelManager().KeywordPlural()
}

func (model *SModelBase) GetName() string {
	return ""
}

func (model *SModelBase) SetModelManager(man IModelManager, virtual IModel) {
	model.manager = man
	model.SetVirtualObject(virtual)
}

func (model *SModelBase) GetModelManager() IModelManager {
	return model.manager
}

func (model *SModelBase) GetIModel() IModel {
	return model.GetVirtualObject().(IModel)
}

func (model *SModelBase) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	desc := jsonutils.NewDict()
	desc.Add(jsonutils.NewString(model.Keyword()), "res_name")
	return desc
}

func (model *SModelBase) GetShortDescV2(ctx context.Context) *apis.ModelBaseShortDescDetail {
	return &apis.ModelBaseShortDescDetail{ResName: model.Keyword()}
}

// list hooks
func (model *SModelBase) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := jsonutils.NewDict()
	return getModelExtraDetails(model.GetIModel(), ctx, extra)
}

// get hooks
func (model *SModelBase) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return false
}

func (model *SModelBase) GetExtraDetailsV2(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, out *apis.ModelBaseDetails) error {
	out.CanDelete = true
	out.CanUpdate = true
	err := model.GetIModel().ValidateDeleteCondition(ctx)
	if err != nil {
		out.CanDelete = false
		out.DeleteFailReason = err.Error()
	}
	err = model.GetIModel().ValidateUpdateCondition(ctx)
	if err != nil {
		out.CanUpdate = false
		out.UpdateFailReason = err.Error()
	}
	return nil
}

func (model *SModelBase) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra := jsonutils.NewDict()
	return getModelExtraDetails(model.GetIModel(), ctx, extra), nil
}

func (model *SModelBase) GetExtraDetailsHeaders(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) map[string]string {
	return nil
}

// create hooks
func (model *SModelBase) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return nil
}

func (model *SModelBase) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {

}

func (model *SModelBase) AllowPerformAction(ctx context.Context, userCred mcclient.TokenCredential, action string, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (model *SModelBase) PerformAction(ctx context.Context, userCred mcclient.TokenCredential, action string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, httperrors.NewActionNotFoundError("Action %s not found", action)
}

// update hooks
func (model *SModelBase) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return false
}

func (model *SModelBase) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return data, nil
}

func (model *SModelBase) PreUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	// do nothing
}

func (model *SModelBase) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	// do nothing
}

// delete hooks
func (model *SModelBase) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (model *SModelBase) ValidateUpdateCondition(ctx context.Context) error {
	return nil
}

func (model *SModelBase) ValidateDeleteCondition(ctx context.Context) error {
	return nil
}

func (model *SModelBase) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	// do nothing
	return nil
}

func (model *SModelBase) PreDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	// do nothing
}

func (model *SModelBase) PostDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	// do nothing
}

func (model *SModelBase) MarkDelete() error {
	// do nothing
	return nil
}

func (model *SModelBase) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (model *SModelBase) GetOwnerId() mcclient.IIdentityProvider {
	return nil
}

func (model *SModelBase) GetParentId() string {
	return ""
}

func (model *SModelBase) IsSharable(ownerId mcclient.IIdentityProvider) bool {
	return false
}

func (model *SModelBase) CustomizedGetDetailsBody(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, nil
}

func (model *SModelBase) UpdateInContext(ctx context.Context, userCred mcclient.TokenCredential, ctxObjs []IModel, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, nil
}

func (model *SModelBase) DeleteInContext(ctx context.Context, userCred mcclient.TokenCredential, ctxObjs []IModel, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, nil
}
