package db

import (
	"context"
	"database/sql"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/sqlchemy"
)

type SModelBase struct {
	// empty struct
	manager IModelManager `ignore:"true"` // pointer to modelmanager
}

type SModelBaseManager struct {
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

func (manager *SModelBaseManager) GetContextManager() []IModelManager {
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

func (manager *SModelBaseManager) FilterByOwner(q *sqlchemy.SQuery, owner string) *sqlchemy.SQuery {
	return q
}

func (manager *SModelBaseManager) GetOwnerId(userCred mcclient.IIdentityProvider) string {
	return ""
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

func (manager *SModelBaseManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return data, nil
}

func (manager *SModelBaseManager) OnCreateComplete(ctx context.Context, items []IModel, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	// do nothing
}

func (manager *SModelBaseManager) AllowPerformAction(ctx context.Context, userCred mcclient.TokenCredential, action string, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (manager *SModelBaseManager) PerformAction(ctx context.Context, userCred mcclient.TokenCredential, action string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, nil
}

func (manager *SModelBaseManager) AllowPerformCheckCreateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return IsAdminAllowClassPerform(userCred, manager, "check-create-data")
}

func (manager *SModelBaseManager) InitializeData() error {
	return nil
}

func (manager *SModelBaseManager) DoCreate(ctx context.Context, userCred mcclient.TokenCredential, kwargs jsonutils.JSONObject, data jsonutils.JSONObject, realManager IModelManager) (IModel, error) {
	return nil, fmt.Errorf("Do create not implement?")
}

func (manager *SModelBaseManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	return q, nil
}

func (manager *SModelBaseManager) GetExportExtraKeys(ctx context.Context, query jsonutils.JSONObject, rowMap map[string]string) *jsonutils.JSONDict {
	return jsonutils.NewDict()
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

func (model *SModelBase) SetModelManager(man IModelManager) {
	model.manager = man
}

func (model *SModelBase) GetModelManager() IModelManager {
	return model.manager
}

func (model *SModelBase) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	desc := jsonutils.NewDict()
	desc.Add(jsonutils.NewString(model.Keyword()), "res_name")
	return desc
}

// list hooks
func (model *SModelBase) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	return jsonutils.NewDict()
}

// get hooks
func (model *SModelBase) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return false
}

func (model *SModelBase) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	return jsonutils.NewDict()
}

// create hooks
func (model *SModelBase) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return nil
}

func (model *SModelBase) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) {

}

func (model *SModelBase) AllowPerformAction(ctx context.Context, userCred mcclient.TokenCredential, action string, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (model *SModelBase) PerformAction(ctx context.Context, userCred mcclient.TokenCredential, action string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, nil
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

func (model *SModelBase) GetOwnerProjectId() string {
	return ""
}
