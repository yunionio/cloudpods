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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/object"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/version"
	"yunion.io/x/sqlchemy"
	"yunion.io/x/sqlchemy/backends/clickhouse"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/dbutils"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/splitable"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

const (
	COLUMN_RECORD_CHECKSUM = "record_checksum"
	COLUMN_UPDATE_VERSION  = "update_version"
	COLUMN_UPDATED_AT      = "updated_at"
)

type SModelBase struct {
	object.SObject

	manager IModelManager `ignore:"true"` // pointer to modelmanager
}

type SModelBaseManager struct {
	object.SObject

	tableSpec     ITableSpec
	keyword       string
	keywordPlural string
	alias         string
	aliasPlural   string
	extraHook     IModelManagerExtraHook
}

func NewModelBaseManager(model interface{}, tableName string, keyword string, keywordPlural string) SModelBaseManager {
	return NewModelBaseManagerWithDBName(model, tableName, keyword, keywordPlural, sqlchemy.DefaultDB)
}

func NewModelBaseManagerWithDBName(model interface{}, tableName string, keyword string, keywordPlural string, dbName sqlchemy.DBName) SModelBaseManager {
	return NewModelBaseManagerWithSplitableDBName(model, tableName, keyword, keywordPlural, "", "", 0, 0, dbName)
}

func NewModelBaseManagerWithSplitable(model interface{}, tableName string, keyword string, keywordPlural string, indexField string, dateField string, maxDuration time.Duration, maxSegments int) SModelBaseManager {
	return NewModelBaseManagerWithSplitableDBName(model, tableName, keyword, keywordPlural, indexField, dateField, maxDuration, maxSegments, sqlchemy.DefaultDB)
}

func NewModelBaseManagerWithSplitableDBName(model interface{}, tableName string, keyword string, keywordPlural string, indexField string, dateField string, maxDuration time.Duration, maxSegments int, dbName sqlchemy.DBName) SModelBaseManager {
	ts := newTableSpec(model, tableName, indexField, dateField, maxDuration, maxSegments, dbName)
	modelMan := SModelBaseManager{
		tableSpec:     ts,
		keyword:       keyword,
		keywordPlural: keywordPlural,
		extraHook:     NewEmptyExtraHook(),
	}
	return modelMan
}

func NewModelBaseManagerWithClickhouseMapping(manager IModelManager, keyword, keywordPlural string) SModelBaseManager {
	ots := manager.TableSpec()
	var extraOpts sqlchemy.TableExtraOptions
	switch consts.DefaultDBDialect() {
	case "mysql":
		cfg := dbutils.ParseMySQLConnStr(consts.DefaultDBConnStr())
		err := cfg.Validate()
		if err != nil {
			panic(fmt.Sprintf("invalid mysql connection string %s", consts.DefaultDBConnStr()))
		}
		extraOpts = clickhouse.MySQLExtraOptions(cfg.Hostport, cfg.Database, ots.Name(), cfg.Username, cfg.Password)
	default:
		panic(fmt.Sprintf("unsupport dialect %s to be backend of clickhouse", consts.DefaultDBDialect()))
	}
	nts := newClickhouseTableSpecFromMySQL(ots, ots.Name(), ClickhouseDB, extraOpts)
	modelMan := SModelBaseManager{
		tableSpec:     nts,
		keyword:       keyword,
		keywordPlural: keywordPlural,
	}
	return modelMan
}

func (manager *SModelBaseManager) CreateByInsertOrUpdate() bool {
	return true
}

func (manager *SModelBaseManager) IsStandaloneManager() bool {
	return false
}

func (manager *SModelBaseManager) GetIModelManager() IModelManager {
	virt := manager.GetVirtualObject()
	if virt == nil {
		panic(fmt.Sprintf("[%s] Forgot to call SetVirtualObject?", manager.Keyword()))
	}
	r, ok := virt.(IModelManager)
	if !ok {
		panic(fmt.Sprintf("Cannot convert virtual object to IModelManager: %#v", virt))
	}
	return r
}

func (manager *SModelBaseManager) GetImmutableInstance(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) IModelManager {
	return manager.GetIModelManager()
}

func (manager *SModelBaseManager) GetMutableInstance(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) IModelManager {
	return manager.GetIModelManager()
}

func (manager *SModelBaseManager) PrepareQueryContext(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) context.Context {
	return ctx
}

func (manager *SModelBaseManager) SetAlias(alias string, aliasPlural string) {
	manager.alias = alias
	manager.aliasPlural = aliasPlural
}

func (manager *SModelBaseManager) TableSpec() ITableSpec {
	return manager.tableSpec
}

func (manager *SModelBaseManager) GetSplitTable() *splitable.SSplitTableSpec {
	return manager.TableSpec().GetSplitTable()
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

func (manager *SModelBaseManager) EnableGenerateName() bool {
	return true
}

func (manager *SModelBaseManager) HasName() bool {
	return false
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

func (manager *SModelBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input apis.ModelBaseListInput,
) (*sqlchemy.SQuery, error) {
	return q, nil
}

func (manager *SModelBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input apis.ModelBaseListInput,
) (*sqlchemy.SQuery, error) {
	return q, nil
}

func (manager *SModelBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	// no field match
	return q, httperrors.ErrNotFound
}

func (manager *SModelBaseManager) QueryDistinctExtraFields(q *sqlchemy.SQuery, resource string, fields []string) (*sqlchemy.SQuery, error) {
	return q, httperrors.ErrNotImplemented
}

func (manager *SModelBaseManager) CustomizeFilterList(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*CustomizeListFilters, error) {
	return NewCustomizeListFilters(), nil
}

func (manager *SModelBaseManager) ExtraSearchConditions(ctx context.Context, q *sqlchemy.SQuery, like string) []sqlchemy.ICondition {
	return nil
}

func (manager *SModelBaseManager) NewQuery(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, useRawQuery bool) *sqlchemy.SQuery {
	if useRawQuery {
		return manager.Query()
	} else {
		return manager.GetIModelManager().Query()
	}
}

// fetch hook
func (manager *SModelBaseManager) getTable() *sqlchemy.STable {
	return manager.TableSpec().Instance()
}

func (manager *SModelBaseManager) Query(fieldNames ...string) *sqlchemy.SQuery {
	instance := manager.getTable()
	fields := make([]sqlchemy.IQueryField, len(fieldNames))
	for i, f := range fieldNames {
		fields[i] = instance.Field(f)
	}
	return instance.Query(fields...)
}

func (manager *SModelBaseManager) RawQuery(fieldNames ...string) *sqlchemy.SQuery {
	return manager.Query(fieldNames...)
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

func (manager *SModelBaseManager) FilterByOwner(ctx context.Context, q *sqlchemy.SQuery, man FilterByOwnerProvider, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	return q
}

func (manager *SModelBaseManager) FilterBySystemAttributes(q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	return q
}

func (manager *SModelBaseManager) FilterByHiddenSystemAttributes(q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	return q
}

func (manager *SModelBaseManager) FilterByUniqValues(q *sqlchemy.SQuery, uniqValues jsonutils.JSONObject) *sqlchemy.SQuery {
	return q
}

func (manager *SModelBaseManager) FetchById(idStr string) (IModel, error) {
	return nil, sql.ErrNoRows
}

func (manager *SModelBaseManager) FetchByName(ctx context.Context, userCred mcclient.IIdentityProvider, idStr string) (IModel, error) {
	return nil, sql.ErrNoRows
}

func (manager *SModelBaseManager) FetchByIdOrName(ctx context.Context, userCred mcclient.IIdentityProvider, idStr string) (IModel, error) {
	return nil, sql.ErrNoRows
}

// create hooks
func (manager *SModelBaseManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (manager *SModelBaseManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input apis.ModelBaseCreateInput) (apis.ModelBaseCreateInput, error) {
	return input, nil
}

func (manager *SModelBaseManager) OnCreateComplete(ctx context.Context, items []IModel, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data []jsonutils.JSONObject) {
	// do nothing
}

func (manager *SModelBaseManager) PerformAction(ctx context.Context, userCred mcclient.TokenCredential, action string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, httperrors.NewActionNotFoundError("Action %s not found, please check service version, current version: %s", action, version.GetShortString())
}

func (manager *SModelBaseManager) InitializeData() error {
	return nil
}

func (manager *SModelBaseManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
	q = q.AppendField(q.QueryFields()...)
	return q, nil
}

func (manager *SModelBaseManager) GetExportExtraKeys(ctx context.Context, keys stringutils2.SSortedStrings, rowMap map[string]string) *jsonutils.JSONDict {
	return jsonutils.NewDict()
}

func (manager *SModelBaseManager) CustomizeHandlerInfo(info *appsrv.SHandlerInfo) {
	info.SetProcessTimeoutCallback(manager.GetIModelManager().SetHandlerProcessTimeout)
}

func (manager *SModelBaseManager) SetHandlerProcessTimeout(info *appsrv.SHandlerInfo, r *http.Request) time.Duration {
	splitableExportPath := fmt.Sprintf("/%s/splitable-export", manager.KeywordPlural())
	if r.Method == http.MethodGet && (len(r.URL.Query().Get("export_keys")) > 0 ||
		r.URL.Query().Has("force_no_paging") ||
		strings.HasSuffix(r.URL.Path, splitableExportPath)) {
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

func (manager *SModelBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []apis.ModelBaseDetails {
	ret := make([]apis.ModelBaseDetails, len(objs))
	return ret
}

func (manager *SModelBaseManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	return nil, nil
}

func (manager *SModelBaseManager) FetchUniqValues(ctx context.Context, data jsonutils.JSONObject) jsonutils.JSONObject {
	return nil
}

func (manager *SModelBaseManager) NamespaceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeSystem
}

func (manager *SModelBaseManager) ResourceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeSystem
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

	q := im.Query()
	q, err = ListItemQueryFilters(im, ctx, q, userCred, query, policy.PolicyActionList)
	if err != nil {
		return nil, err
	}
	var (
		backupQuery = *q
		res         = jsonutils.NewDict()
	)
	// query field
	for i := 0; i < len(fields); i++ {
		var nq = backupQuery.SubQuery().Query()
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
		nq := backupQuery.SubQuery().Query()
		fe, _ := efs[i].GetString()
		nqp, err := im.QueryDistinctExtraField(nq, fe)
		if err != nil {
			continue
		}
		ef, err := nqp.AllStringMap()
		if errors.Cause(err) == sql.ErrNoRows {
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

func (manager *SModelBaseManager) GetPropertyDistinctFields(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	im, ok := manager.GetVirtualObject().(IModelManager)
	if !ok {
		im = manager
	}
	input := &apis.DistinctFieldsInput{}
	query.Unmarshal(input)
	if len(input.Field) == 0 && (len(input.ExtraResource) == 0 || len(input.ExtraField) == 0) {
		return nil, httperrors.NewMissingParameterError("field")
	}
	// validate field
	for _, fd := range input.Field {
		var hasField = false
		for _, field := range manager.getTable().Fields() {
			if field.Name() == fd {
				hasField = true
				break
			}
		}
		if !hasField {
			return nil, httperrors.NewBadRequestError("model has no field %s", fd)
		}
	}
	var err error
	q := im.Query()
	q, err = ListItemQueryFilters(im, ctx, q, userCred, query, policy.PolicyActionList)
	if err != nil {
		return nil, err
	}
	result := jsonutils.NewDict()
	fields := jsonutils.NewArray()
	if len(input.Field) > 0 {
		sq := q.Copy().ResetFields()
		// query field
		for i := 0; i < len(input.Field); i++ {
			sq = sq.AppendField(sq.Field(input.Field[i]))
		}
		rows, err := sq.Distinct().Rows()
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			mMap, err := sq.Row2Map(rows)
			if err != nil {
				return nil, errors.Wrapf(err, "Row2Map")
			}
			fields.Add(jsonutils.Marshal(mMap))
		}
	}
	result.Set("fields", fields)

	extraFields := jsonutils.NewArray()
	if len(input.ExtraResource) > 0 && len(input.ExtraField) > 0 {
		// query extra field
		sq := q.Copy().ResetFields()
		em := GetModelManager(input.ExtraResource)
		if gotypes.IsNil(em) {
			return nil, httperrors.NewInputParameterError("invalid extra_resource %s", input.ExtraResource)
		}
		for _, field := range input.ExtraField {
			if gotypes.IsNil(em.TableSpec().ColumnSpec(field)) {
				return nil, httperrors.NewInputParameterError("resource %s does not have field %s", input.ExtraResource, field)
			}
		}
		sq, err := im.QueryDistinctExtraFields(sq, input.ExtraResource, input.ExtraField)
		if err != nil {
			return nil, err
		}

		rows, err := sq.Distinct().Rows()
		if err != nil {
			return nil, err
		}

		defer rows.Close()
		for rows.Next() {
			mMap, err := sq.Row2Map(rows)
			if err != nil {
				return nil, errors.Wrapf(err, "Row2Map")
			}
			extraFields.Add(jsonutils.Marshal(mMap))
		}
	}
	result.Set("extra_fields", extraFields)
	return result, nil
}

func (manager *SModelBaseManager) BatchPreValidate(
	ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject, data *jsonutils.JSONDict, count int,
) error {
	return nil
}

func (manager *SModelBaseManager) BatchCreateValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return data, nil
}

func (manager *SModelBaseManager) OnCreateFailed(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return nil
}

func (manager *SModelBaseManager) GetI18N(ctx context.Context, idstr string, resObj jsonutils.JSONObject) *jsonutils.JSONDict {
	return nil
}

func (manager *SModelBaseManager) GetPropertySplitable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	stable := manager.GetIModelManager().GetImmutableInstance(ctx, userCred, query).GetSplitTable()
	if stable == nil {
		// generate a fake metadata tbl record
		man := manager.GetIModelManager().GetImmutableInstance(ctx, userCred, query)
		subq := man.Query().SubQuery()
		q := subq.Query(
			sqlchemy.MIN("start", subq.Field("id")),
			sqlchemy.MAX("end", subq.Field("id")),
			sqlchemy.MIN("start_date", subq.Field("ops_time")),
			sqlchemy.MAX("end_date", subq.Field("ops_time")),
			sqlchemy.COUNT("count", subq.Field("id")),
			sqlchemy.MIN("created_at", subq.Field("ops_time")),
		)
		meta := splitable.STableMetadata{
			Id:    1,
			Table: "action_tbl",
		}
		err := q.First(&meta)
		if err != nil {
			return nil, errors.Wrap(err, "Query metadata")
		}
		metas := []splitable.STableMetadata{meta}
		return jsonutils.Marshal(metas), nil
	}
	metas, err := stable.GetTableMetas()
	if err != nil {
		return nil, errors.Wrap(err, "GetTableMetas")
	}
	return jsonutils.Marshal(metas), nil
}

func (manager *SModelBaseManager) GetPropertySplitableExport(ctx context.Context, userCred mcclient.TokenCredential, input apis.SplitTableExportInput) (jsonutils.JSONObject, error) {
	splitable := manager.GetIModelManager().GetImmutableInstance(ctx, userCred, jsonutils.Marshal(input)).GetSplitTable()
	if splitable == nil {
		return nil, errors.Wrap(httperrors.ErrNotSupported, "not splitable")
	}
	if len(input.Table) == 0 {
		return nil, httperrors.NewMissingParameterError("table")
	}

	metas, err := splitable.GetTableMetas()
	if err != nil {
		return nil, errors.Wrap(err, "GetTableMetas")
	}
	for i := 0; i < len(metas); i += 1 {
		if metas[i].Table == input.Table {
			if input.Limit <= 0 {
				input.Limit = apis.MAX_SPLITABLE_EXPORT_LIMIT
			}
			if input.Offset < 0 {
				input.Offset = 0
			}
			q := splitable.GetTableSpec(metas[i]).Query().Limit(input.Limit).Offset(input.Offset)
			resp, err := q.AllStringMap()
			if err != nil {
				return nil, errors.Wrapf(err, "q.AllStringMap")
			}
			exportId := fmt.Sprintf("%s(%d-%d)", metas[i].Table, metas[i].Start, metas[i].End)
			obj := logclient.NewSimpleObject(exportId, exportId, manager.Keyword())
			logclient.AddActionLogWithContext(ctx, obj, logclient.ACT_EXPORT, nil, userCred, true)
			return jsonutils.Marshal(resp), nil
		}
	}
	return nil, httperrors.NewResourceNotFoundError("table %s not found", input.Table)
}

func (manager *SModelBaseManager) PerformPurgeSplitable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PurgeSplitTableInput) (jsonutils.JSONObject, error) {
	splitable := manager.GetIModelManager().GetImmutableInstance(ctx, userCred, query).GetSplitTable()
	if splitable == nil {
		return jsonutils.Marshal(map[string][]string{"tables": {}}), nil
	}
	ret, err := splitable.Purge(input.Tables)
	if err != nil {
		return nil, errors.Wrapf(err, "Purge")
	}
	return jsonutils.Marshal(map[string][]string{"tables": ret}), nil
}

func (manager *SModelBaseManager) CustomizedTotalCount(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, totalQ *sqlchemy.SQuery) (int, jsonutils.JSONObject, error) {
	ret := apis.TotalCountBase{}
	err := totalQ.First(&ret)
	if err != nil {
		return -1, nil, errors.Wrapf(err, "SModelBaseManager Query total %s", totalQ.DebugString())
	}
	return ret.Count, nil, nil
}

func (manager *SModelBaseManager) RegisterExtraHook(eh IModelManagerExtraHook) {
	manager.extraHook = eh
}

func (manager *SModelBaseManager) GetExtraHook() IModelManagerExtraHook {
	return manager.extraHook
}

func (model SModelBase) GetId() string {
	return ""
}

func (model SModelBase) Keyword() string {
	return model.GetModelManager().Keyword()
}

func (model SModelBase) KeywordPlural() string {
	return model.GetModelManager().KeywordPlural()
}

func (model SModelBase) GetName() string {
	return ""
}

func (model *SModelBase) GetUpdatedAt() time.Time {
	return time.Time{}
}

func (model *SModelBase) GetUpdateVersion() int {
	return 0
}

func (model *SModelBase) GetDeleted() bool {
	return false
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

func (model *SModelBase) GetExtraDetailsHeaders(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) map[string]string {
	return nil
}

// create hooks
func (model *SModelBase) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return nil
}

func (model *SModelBase) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {

}

func (model *SModelBase) PerformAction(ctx context.Context, userCred mcclient.TokenCredential, action string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, httperrors.NewActionNotFoundError("Action %s not found, please check service version, current version: %s", action, version.GetShortString())
}

func (model *SModelBase) PreCheckPerformAction(
	ctx context.Context, userCred mcclient.TokenCredential,
	action string, query jsonutils.JSONObject, data jsonutils.JSONObject,
) error {
	return nil
}

// update hooks
func (model *SModelBase) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return false
}

func (model *SModelBase) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.ModelBaseUpdateInput) (apis.ModelBaseUpdateInput, error) {
	return input, nil
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

func (model *SModelBase) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	return nil
}

func (model *SModelBase) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	// do nothing
	return nil
}

func (model *SModelBase) cleanModelUsages(ctx context.Context, userCred mcclient.TokenCredential) {
	usages := model.GetIModel().GetUsages()
	if CancelUsages != nil && len(usages) > 0 {
		CancelUsages(ctx, userCred, usages)
	}
}

func (model *SModelBase) RecoverUsages(ctx context.Context, userCred mcclient.TokenCredential) {
	usages := model.GetIModel().GetUsages()
	if AddUsages != nil && len(usages) > 0 {
		AddUsages(ctx, userCred, usages)
	}
}

func (model *SModelBase) PreDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	// clean usage on predelete
	// clean usage before fakedelete for pending delete models
	model.cleanModelUsages(ctx, userCred)
}

func (model *SModelBase) PostDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	// do nothing
}

func (model *SModelBase) MarkDelete() error {
	// do nothing
	return nil
}

func (model *SModelBase) MarkPendingDeleted() {
	return
}

func (model *SModelBase) CancelPendingDeleted() {
	return
}

func (model *SModelBase) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (model *SModelBase) GetOwnerId() mcclient.IIdentityProvider {
	return nil
}

func (model *SModelBase) GetUniqValues() jsonutils.JSONObject {
	return nil
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

func (model *SModelBase) GetUsages() []IUsage {
	return nil
}

func (model *SModelBase) GetI18N(ctx context.Context) *jsonutils.JSONDict {
	return nil
}

type SEmptyExtraHook struct{}

func NewEmptyExtraHook() *SEmptyExtraHook {
	return new(SEmptyExtraHook)
}

func (e SEmptyExtraHook) AfterPostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, model IModel, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return nil
}

func (e SEmptyExtraHook) AfterPostDelete(ctx context.Context, userCred mcclient.TokenCredential, model IModel, query jsonutils.JSONObject) error {
	return nil
}
