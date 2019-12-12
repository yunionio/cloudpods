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
	"reflect"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/filterclause"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/appsrv/dispatcher"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

var (
	CancelUsages func(ctx context.Context, userCred mcclient.TokenCredential, usages []IUsage)
)

type DBModelDispatcher struct {
	modelManager IModelManager
}

func NewModelHandler(manager IModelManager) *DBModelDispatcher {
	// registerModelManager(manager)
	return &DBModelDispatcher{modelManager: manager}
}

func (dispatcher *DBModelDispatcher) Keyword() string {
	return dispatcher.modelManager.Keyword()
}

func (dispatcher *DBModelDispatcher) KeywordPlural() string {
	return dispatcher.modelManager.KeywordPlural()
}

func (dispatcher *DBModelDispatcher) ContextKeywordPlurals() [][]string {
	ctxMans := dispatcher.modelManager.GetContextManagers()
	if ctxMans != nil {
		keys := make([][]string, len(ctxMans))
		for i := 0; i < len(ctxMans); i += 1 {
			keys[i] = make([]string, len(ctxMans[i]))
			for j := 0; j < len(ctxMans[i]); j += 1 {
				keys[i][j] = ctxMans[i][j].KeywordPlural()
			}
		}
		return keys
	}
	return nil
}

func (dispatcher *DBModelDispatcher) Filter(f appsrv.FilterHandler) appsrv.FilterHandler {
	if consts.IsRbacEnabled() {
		return auth.AuthenticateWithDelayDecision(f, true)
	} else {
		return auth.Authenticate(f)
	}
}

func (dispatcher *DBModelDispatcher) CustomizeHandlerInfo(handler *appsrv.SHandlerInfo) {
	dispatcher.modelManager.CustomizeHandlerInfo(handler)
}

func fetchUserCredential(ctx context.Context) mcclient.TokenCredential {
	return policy.FetchUserCredential(ctx)
}

var (
	searchOps = map[string]string{
		"contains":   "contains",
		"startswith": "startswith",
		"endswith":   "endswith",
		"empty":      "isnullorempty",
	}
)

func parseSearchFieldkey(key string) (string, string) {
	for op, fn := range searchOps {
		if strings.HasSuffix(key, "__"+op) {
			key = key[:len(key)-(2+len(op))]
			return key, fn
		} else if strings.HasSuffix(key, "__i"+op) {
			key = key[:len(key)-(3+len(op))]
			return key, fn
		}
	}
	return key, ""
}

func listItemsQueryByColumn(manager IModelManager, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	if query == nil {
		return q, nil
	}
	qdata, err := query.GetMap()
	if err != nil {
		return nil, err
	}

	listF := searchFields(manager, userCred)
	for key := range qdata {
		fn, op := parseSearchFieldkey(key)
		if listF.Contains(fn) {
			colSpec := manager.TableSpec().ColumnSpec(fn)
			if colSpec != nil {
				arrV := jsonutils.GetQueryStringArray(query, key)
				if len(op) > 0 {
					strV := strings.Join(arrV, ",")
					filter := fmt.Sprintf("%s.%s(%s)", fn, op, strV)
					fc := filterclause.ParseFilterClause(filter)
					if fc != nil {
						cond := fc.QueryCondition(q)
						if cond != nil {
							q = q.Filter(cond)
						}
					}
				} else if len(arrV) > 1 {
					for i := range arrV {
						arrV[i] = colSpec.ConvertFromString(arrV[i])
					}
					q = q.In(fn, arrV)
				} else if len(arrV) == 1 {
					strV := colSpec.ConvertFromString(arrV[0])
					q = q.Equals(fn, strV)
				}
			}
		}
	}
	return q, nil
}

func applyListItemsSearchFilters(manager IModelManager, ctx context.Context, q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential, likes []string) (*sqlchemy.SQuery, error) {
	conds := make([]sqlchemy.ICondition, 0)
	for _, like := range likes {
		like = strings.TrimSpace(like)
		if len(like) > 0 {
			isAscii := utils.IsAscii(like)
			for _, colName := range searchFields(manager, userCred) {
				colSpec := manager.TableSpec().ColumnSpec(colName)
				if colSpec != nil && colSpec.IsSearchable() && (!colSpec.IsAscii() || (colSpec.IsAscii() && isAscii)) {
					conds = append(conds, sqlchemy.Contains(q.Field(colName), like))
				}
			}
			extraConds := manager.ExtraSearchConditions(ctx, q, like)
			if len(extraConds) > 0 {
				conds = append(conds, extraConds...)
			}
		}
	}
	if len(conds) > 0 {
		q = q.Filter(sqlchemy.OR(conds...))
	}
	return q, nil
}

func applyListItemsGeneralFilters(manager IModelManager, q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential, filters []string, filterAny bool) (*sqlchemy.SQuery, error) {
	conds := make([]sqlchemy.ICondition, 0)
	schFields := searchFields(manager, userCred) // only filter searchable fields
	for _, f := range filters {
		fc := filterclause.ParseFilterClause(f)
		if fc != nil {
			if schFields.Contains(fc.GetField()) {
				cond := fc.QueryCondition(q)
				if cond != nil {
					conds = append(conds, cond)
				}
			}
		}
	}
	if len(conds) > 0 {
		if filterAny {
			q = q.Filter(sqlchemy.OR(conds...))
		} else {
			q = q.Filter(sqlchemy.AND(conds...))
		}
	}
	return q, nil
}

func applyListItemsGeneralJointFilters(manager IModelManager, q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential, jointFilters []string, filterAny bool) (*sqlchemy.SQuery, error) {
	for _, f := range jointFilters {
		jfc := filterclause.ParseJointFilterClause(f)
		if jfc != nil {
			jointModelManager := GetModelManager(jfc.GetJointModelName())
			schFields := searchFields(jointModelManager, userCred)
			if schFields.Contains(jfc.GetField()) {
				sq := jointModelManager.Query(jfc.RelatedKey)
				cond := jfc.GetJointFilter(sq)
				if cond != nil {
					sq = sq.Filter(cond)
					if filterAny {
						q = q.Filter(sqlchemy.OR(sqlchemy.In(q.Field(jfc.OriginKey), sq)))
					} else {
						q = q.Filter(sqlchemy.AND(sqlchemy.In(q.Field(jfc.OriginKey), sq)))
					}
				}
			}
		}
	}
	return q, nil
}

func ListItemQueryFilters(manager IModelManager,
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	action string,
) (*sqlchemy.SQuery, error) {
	return listItemQueryFilters(manager, ctx, q, userCred, query, action, false)
}

func listItemQueryFilters(manager IModelManager,
	ctx context.Context, q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	action string,
	doCheckRbac bool,
) (*sqlchemy.SQuery, error) {
	ownerId, queryScope, err := FetchCheckQueryOwnerScope(ctx, userCred, query, manager, action, doCheckRbac)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	q = manager.FilterByOwner(q, ownerId, queryScope)
	// apply all filters
	q = manager.FilterBySystemAttributes(q, userCred, query, queryScope)
	q = manager.FilterByHiddenSystemAttributes(q, userCred, query, queryScope)

	q, err = ListItemFilter(manager, ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	if query.Contains("export_keys") {
		q, err = manager.ListItemExportKeys(ctx, q, userCred, query)
		if err != nil {
			return nil, err
		}
	}
	q, err = listItemsQueryByColumn(manager, q, userCred, query)
	if err != nil {
		return nil, err
	}
	searches := jsonutils.GetQueryStringArray(query, "search")
	if len(searches) > 0 {
		q, err = applyListItemsSearchFilters(manager, ctx, q, userCred, searches)
		if err != nil {
			return nil, err
		}
	}
	filterAny, _ := query.Bool("filter_any")
	filters := jsonutils.GetQueryStringArray(query, "filter")
	if len(filters) > 0 {
		q, err = applyListItemsGeneralFilters(manager, q, userCred, filters, filterAny)
	}
	jointFilter := jsonutils.GetQueryStringArray(query, "joint_filter")
	if len(jointFilter) > 0 {
		q, _ = applyListItemsGeneralJointFilters(manager, q, userCred, jointFilter, filterAny)
	}
	return q, nil
}

func mergeFields(metaFields, queryFields []string, isSysAdmin bool) stringutils2.SSortedStrings {
	meta := stringutils2.NewSortedStrings(metaFields)
	if len(queryFields) == 0 {
		return meta
	}

	query := stringutils2.NewSortedStrings(queryFields)
	_, mAndQ, qNoM := stringutils2.Split(meta, query)

	if !isSysAdmin {
		return mAndQ
	}

	// only sysadmin can specify list Fields
	return stringutils2.Merge(mAndQ, qNoM)
}

func Query2List(manager IModelManager, ctx context.Context, userCred mcclient.TokenCredential, q *sqlchemy.SQuery, query jsonutils.JSONObject, delayFetch bool) ([]jsonutils.JSONObject, error) {
	metaFields := listFields(manager, userCred)
	fieldFilter := jsonutils.GetQueryStringArray(query, "field")
	listF := mergeFields(metaFields, fieldFilter, IsAllowList(rbacutils.ScopeSystem, userCred, manager))

	showDetails := false
	showDetailsJson, _ := query.Get("details")
	if showDetailsJson != nil {
		showDetails, _ = showDetailsJson.Bool()
	} else {
		showDetails = true
	}
	items := make([]IModel, 0)
	results := make([]jsonutils.JSONObject, 0)
	rows, err := q.Rows()
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		item, err := NewModelObject(manager)
		if err != nil {
			return nil, err
		}

		extraData := jsonutils.NewDict()
		if query.Contains("export_keys") {
			RowMap, err := q.Row2Map(rows)
			if err != nil {
				return nil, err
			}
			extraKeys := manager.GetExportExtraKeys(ctx, query, RowMap)
			if extraKeys != nil {
				extraData.Update(extraKeys)
			}
			err = q.RowMap2Struct(RowMap, item)
			if err != nil {
				return nil, err
			}
		} else {
			err = q.Row2Struct(rows, item)
			if err != nil {
				return nil, err
			}
			if delayFetch {
				err = Fetch(item)
				if err != nil {
					return nil, err
				}
			}
		}

		jsonDict := jsonutils.Marshal(item).(*jsonutils.JSONDict)
		jsonDict = jsonDict.CopyIncludes([]string(listF)...)
		jsonDict.Update(extraData)
		if showDetails && !query.Contains("export_keys") {
			extraDict := item.GetCustomizeColumns(ctx, userCred, query)
			if extraDict != nil {
				// Fix for Now
				extraDict.Update(jsonDict)
				jsonDict = extraDict
			}
			// jsonDict = getModelExtraDetails(item, ctx, jsonDict)
		}
		results = append(results, jsonDict)
		items = append(items, item)
	}
	if showDetails && !query.Contains("export_keys") {
		extraRows := manager.FetchCustomizeColumns(ctx, userCred, query, items, stringutils2.NewSortedStrings(fieldFilter))
		// log.Debugf("manager.FetchCustomizeColumns: %s %s", extraRows, listF)
		if len(extraRows) == len(results) {
			for i := range results {
				results[i].(*jsonutils.JSONDict).Update(extraRows[i])
			}
		}
	}
	return results, nil
}

func fetchContextObjectsIds(manager IModelManager, ctx context.Context, userCred mcclient.TokenCredential, ctxIds []dispatcher.SResourceContext, queryDict *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	var err error
	for i := 0; i < len(ctxIds); i += 1 {
		queryDict, err = fetchContextObjectIds(manager, ctx, userCred, ctxIds[i], queryDict)
		if err != nil {
			return nil, err
		}
	}
	return queryDict, nil
}

func fetchContextObjectIds(manager IModelManager, ctx context.Context, userCred mcclient.TokenCredential, ctxId dispatcher.SResourceContext, queryDict *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	ctxObj, err := fetchContextObject(manager, ctx, userCred, ctxId)
	if err != nil {
		return nil, err
	}
	queryDict.Add(jsonutils.NewString(ctxObj.GetId()), fmt.Sprintf("%s_id", ctxObj.GetModelManager().Keyword()))
	if len(ctxObj.GetModelManager().Alias()) > 0 {
		queryDict.Add(jsonutils.NewString(ctxObj.GetId()), fmt.Sprintf("%s_id", ctxObj.GetModelManager().Alias()))
	}
	return queryDict, nil
}

func fetchContextObjects(manager IModelManager, ctx context.Context, userCred mcclient.TokenCredential, ctxIds []dispatcher.SResourceContext) ([]IModel, error) {
	ctxObjs := make([]IModel, len(ctxIds))
	for i := 0; i < len(ctxIds); i += 1 {
		ctxObj, err := fetchContextObject(manager, ctx, userCred, ctxIds[i])
		if err != nil {
			return nil, err
		}
		ctxObjs[i] = ctxObj
	}
	return ctxObjs, nil
}

func fetchContextObject(manager IModelManager, ctx context.Context, userCred mcclient.TokenCredential, ctxId dispatcher.SResourceContext) (IModel, error) {
	ctxMans := manager.GetContextManagers()
	if ctxMans == nil {
		return nil, httperrors.NewInternalServerError("No context manager")
	}
	for i := 0; i < len(ctxMans); i += 1 {
		for j := 0; j < len(ctxMans[i]); j += 1 {
			if ctxMans[i][j].KeywordPlural() == ctxId.Type {
				ctxObj, err := fetchItem(ctxMans[i][j], ctx, userCred, ctxId.Id, nil)
				if err != nil {
					if err == sql.ErrNoRows {
						return nil, httperrors.NewResourceNotFoundError2(ctxMans[i][j].Keyword(), ctxId.Id)
					} else {
						return nil, err
					}
				}
				return ctxObj, nil
			}
		}
	}
	return nil, httperrors.NewInternalServerError("No such context %s(%s)", ctxId.Type, ctxId.Id)
}

func ListItems(manager IModelManager, ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, ctxIds []dispatcher.SResourceContext) (*modulebase.ListResult, error) {
	var err error
	var maxLimit int64 = 2048
	limit, _ := query.Int("limit")
	offset, _ := query.Int("offset")
	pagingMarker, _ := query.GetString("paging_marker")
	q := manager.Query()

	queryDict, ok := query.(*jsonutils.JSONDict)
	if !ok {
		return nil, fmt.Errorf("invalid query format")
	}

	if len(ctxIds) > 0 {
		queryDict, err = fetchContextObjectsIds(manager, ctx, userCred, ctxIds, queryDict)
		if err != nil {
			return nil, err
		}
	}

	queryDict, err = manager.ValidateListConditions(ctx, userCred, queryDict)
	if err != nil {
		return nil, err
	}

	q, err = listItemQueryFilters(manager, ctx, q, userCred, queryDict, policy.PolicyActionList, true)
	if err != nil {
		return nil, err
	}

	var totalCnt int
	pagingConf := manager.GetPagingConfig()
	if pagingConf == nil {
		totalCnt, err = q.CountWithError()
		if err != nil {
			return nil, err
		}
		// log.Debugf("total count %d", totalCnt)
		if totalCnt == 0 {
			emptyList := modulebase.ListResult{Data: []jsonutils.JSONObject{}}
			return &emptyList, nil
		}
	} else {
		if limit <= 0 {
			limit = int64(pagingConf.DefaultLimit)
		}
	}
	if int64(totalCnt) > maxLimit && (limit <= 0 || limit > maxLimit) {
		limit = maxLimit
	}

	// export data only
	exportLimit, err := query.Int("export_limit")
	if query.Contains("export_keys") && err == nil {
		limit = exportLimit
	}

	var primaryCol sqlchemy.IColumnSpec
	primaryCols := manager.TableSpec().PrimaryColumns()
	if len(primaryCols) == 1 {
		primaryCol = primaryCols[0]
	}

	orderBy := jsonutils.GetQueryStringArray(queryDict, "order_by")
	order := sqlchemy.SQL_ORDER_DESC
	orderStr, _ := queryDict.GetString("order")
	if orderStr == "asc" {
		order = sqlchemy.SQL_ORDER_ASC
	}
	orderQuery := query.(*jsonutils.JSONDict).Copy()
	for _, orderByField := range orderBy {
		if pagingConf != nil && orderByField == pagingConf.MarkerField {
			// skip markerField
			continue
		}
		colSpec := manager.TableSpec().ColumnSpec(orderByField)
		if colSpec == nil {
			orderQuery.Set(fmt.Sprintf("order_by_%s", orderByField), jsonutils.NewString(string(order)))
		}
	}
	q, err = manager.OrderByExtraFields(ctx, q, userCred, orderQuery)
	if err != nil {
		return nil, err
	}
	if orderBy == nil {
		orderBy = []string{}
	}
	if primaryCol != nil && primaryCol.IsNumeric() {
		orderBy = append(orderBy, primaryCol.Name())
	} else if manager.TableSpec().ColumnSpec("created_at") != nil {
		orderBy = append(orderBy, "created_at")
		if primaryCol != nil {
			orderBy = append(orderBy, primaryCol.Name())
		}
	}
	for _, orderByField := range orderBy {
		if pagingConf != nil && pagingConf.MarkerField == orderByField {
			// skip markerField
			continue
		}
		colSpec := manager.TableSpec().ColumnSpec(orderByField)
		if colSpec != nil {
			if order == sqlchemy.SQL_ORDER_ASC {
				q = q.Asc(orderByField)
			} else {
				q = q.Desc(orderByField)
			}
		}
	}
	if pagingConf != nil {
		if pagingConf.Order == sqlchemy.SQL_ORDER_ASC {
			q = q.Asc(pagingConf.MarkerField)
		} else {
			q = q.Desc(pagingConf.MarkerField)
		}
	}

	if pagingConf != nil {
		q = q.Limit(int(limit) + 1)
		if len(pagingMarker) > 0 {
			if pagingConf.Order == sqlchemy.SQL_ORDER_ASC {
				q = q.GE(pagingConf.MarkerField, pagingMarker)
			} else {
				q = q.LE(pagingConf.MarkerField, pagingMarker)
			}
		}
		retList, err := Query2List(manager, ctx, userCred, q, queryDict, false)
		if err != nil {
			return nil, httperrors.NewGeneralError(err)
		}
		nextMarker := ""
		if int64(len(retList)) > limit {
			markerObj, _ := retList[limit].Get(pagingConf.MarkerField)
			if markerObj != nil {
				nextMarker = fmt.Sprintf("%s", markerObj)
			}
			retList = retList[:limit]
		}
		retResult := modulebase.ListResult{
			Data: retList, Limit: int(limit),
			NextMarker:  nextMarker,
			MarkerField: pagingConf.MarkerField,
			MarkerOrder: string(pagingConf.Order),
		}
		return &retResult, nil
	}

	customizeFilters, err := manager.CustomizeFilterList(ctx, q, userCred, queryDict)
	if err != nil {
		return nil, err
	}
	delayFetch := false
	if customizeFilters.IsEmpty() {
		if limit > 0 {
			q = q.Limit(int(limit))
		}
		if offset > 0 {
			q = q.Offset(int(offset))
			if primaryCol != nil && !query.Contains("export_keys") && consts.QueryOffsetOptimization {
				q.AppendField(q.Field(primaryCol.Name())) // query primary key only
				delayFetch = true
				log.Debugf("apply queryOffsetOptimization")
			}
		}
	}
	retList, err := Query2List(manager, ctx, userCred, q, queryDict, delayFetch)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	retCount := len(retList)

	// apply customizeFilters
	retList, err = customizeFilters.DoApply(retList)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	if len(retList) != retCount {
		totalCnt = len(retList)
	}
	paginate := false
	if !customizeFilters.IsEmpty() {
		// query not use Limit and Offset, do manual pagination
		paginate = true
	}
	return calculateListResult(retList, int64(totalCnt), limit, offset, paginate), nil
}

func calculateListResult(data []jsonutils.JSONObject, total, limit, offset int64, paginate bool) *modulebase.ListResult {
	if paginate {
		// do offset first
		if offset > 0 {
			if total > offset {
				data = data[offset:]
			} else {
				data = []jsonutils.JSONObject{}
			}
		}
		// do limit
		if limit > 0 && total > limit {
			data = data[:limit]
		}
	}

	retResult := modulebase.ListResult{Data: data, Total: int(total), Limit: int(limit), Offset: int(offset)}

	return &retResult
}

func (dispatcher *DBModelDispatcher) List(ctx context.Context, query jsonutils.JSONObject, ctxIds []dispatcher.SResourceContext) (*modulebase.ListResult, error) {
	userCred := fetchUserCredential(ctx)

	items, err := ListItems(dispatcher.modelManager, ctx, userCred, query, ctxIds)
	if err != nil {
		log.Errorf("Fail to list items: %s", err)
		return nil, httperrors.NewGeneralError(err)
	}

	if userCred != nil && userCred.HasSystemAdminPrivilege() && dispatcher.modelManager.ListSkipLog(ctx, userCred, query) {
		appParams := appsrv.AppContextGetParams(ctx)
		if appParams != nil {
			appParams.SkipLog = true
		}
	}
	return items, nil
}

func getModelExtraDetails(item IModel, ctx context.Context, extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	err := item.ValidateDeleteCondition(ctx)
	if err != nil {
		extra.Add(jsonutils.JSONFalse, "can_delete")
		extra.Add(jsonutils.NewString(err.Error()), "delete_fail_reason")
	} else {
		extra.Add(jsonutils.JSONTrue, "can_delete")
	}
	err = item.ValidateUpdateCondition(ctx)
	if err != nil {
		extra.Add(jsonutils.JSONFalse, "can_update")
		extra.Add(jsonutils.NewString(err.Error()), "update_fail_reason")
	} else {
		extra.Add(jsonutils.JSONTrue, "can_update")
	}
	return extra
}

func getModelItemDetails(manager IModelManager, item IModel, ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isHead bool) (jsonutils.JSONObject, error) {
	appParams := appsrv.AppContextGetParams(ctx)
	if appParams == nil && isHead {
		log.Errorf("fail to get http response writer???")
		return nil, httperrors.NewInternalServerError("fail to get http response writer from context")
	}
	hdrs := item.GetExtraDetailsHeaders(ctx, userCred, query)
	for k, v := range hdrs {
		appParams.Response.Header().Add(k, v)
	}

	if isHead {
		appParams.Response.Header().Add("Content-Length", "0")
		appParams.Response.Write([]byte{})
		return nil, nil
	}

	if manager.IsCustomizedGetDetailsBody() {
		return item.CustomizedGetDetailsBody(ctx, userCred, query)
	} else {
		return getItemDetails(manager, item, ctx, userCred, query)
	}
}

func getItemDetails(manager IModelManager, item IModel, ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	extraDict, err := GetExtraDetails(item, ctx, userCred, query)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	if extraDict == nil {
		// override GetExtraDetails
		return nil, nil
	}

	metaFields := GetDetailFields(manager, userCred)
	fieldFilter := jsonutils.GetQueryStringArray(query, "field")
	getFields := mergeFields(metaFields, fieldFilter, IsAllowGet(rbacutils.ScopeSystem, userCred, item))

	jsonDict := jsonutils.Marshal(item).(*jsonutils.JSONDict)
	jsonDict = jsonDict.CopyIncludes(getFields...)
	extraDict.Update(jsonDict)
	// jsonDict = getModelExtraDetails(item, ctx, jsonDict)

	extraRows := manager.FetchCustomizeColumns(ctx, userCred, query, []IModel{item}, stringutils2.NewSortedStrings(fieldFilter))
	if len(extraRows) == 1 {
		extraDict.Update(extraRows[0])
	}

	return extraDict, nil
}

func (dispatcher *DBModelDispatcher) tryGetModelProperty(ctx context.Context, property string, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	funcName := fmt.Sprintf("GetProperty%s", utils.Kebab2Camel(property, "-"))
	allowFuncName := "Allow" + funcName
	modelValue := reflect.ValueOf(dispatcher.modelManager)

	funcValue := modelValue.MethodByName(allowFuncName)
	if !funcValue.IsValid() || funcValue.IsNil() {
		return nil, nil
	}
	userCred := fetchUserCredential(ctx)
	params := []reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(userCred),
		reflect.ValueOf(query),
	}
	outs := funcValue.Call(params)
	if len(outs) != 1 {
		return nil, httperrors.NewInternalServerError("Invald %s return value", funcName)
	}
	if !outs[0].Bool() {
		return nil, httperrors.NewForbiddenError("%s not allow to get property %s", dispatcher.Keyword(), property)
	}

	funcValue = modelValue.MethodByName(funcName)
	outs = funcValue.Call(params)
	if len(outs) != 2 {
		return nil, httperrors.NewInternalServerError("Invald %s return value", funcName)
	}

	resVal := outs[0].Interface()
	errVal := outs[1].Interface()
	if !gotypes.IsNil(errVal) {
		return nil, errVal.(error)
	} else {
		if gotypes.IsNil(resVal) {
			return nil, httperrors.NewBadRequestError("No return value, so why query?")
		} else {
			return resVal.(jsonutils.JSONObject), nil
		}
	}
}

func (dispatcher *DBModelDispatcher) Get(ctx context.Context, idStr string, query jsonutils.JSONObject, isHead bool) (jsonutils.JSONObject, error) {
	// log.Debugf("Get %s", idStr)
	userCred := fetchUserCredential(ctx)

	data, err := dispatcher.tryGetModelProperty(ctx, idStr, query)
	if err != nil {
		return nil, err
	} else if data != nil {
		return data, nil
	}

	model, err := fetchItem(dispatcher.modelManager, ctx, userCred, idStr, query)
	if err == sql.ErrNoRows {
		return nil, httperrors.NewResourceNotFoundError2(dispatcher.modelManager.Keyword(), idStr)
	} else if err != nil {
		return nil, err
	}

	if consts.IsRbacEnabled() {
		err := isObjectRbacAllowed(model, userCred, policy.PolicyActionGet)
		if err != nil {
			return nil, err
		}
	} else if !model.AllowGetDetails(ctx, userCred, query) {
		return nil, httperrors.NewForbiddenError("Not allow to get details")
	}

	if userCred.HasSystemAdminPrivilege() && dispatcher.modelManager.GetSkipLog(ctx, userCred, query) {
		appParams := appsrv.AppContextGetParams(ctx)
		if appParams != nil {
			appParams.SkipLog = true
		}
	}
	return getModelItemDetails(dispatcher.modelManager, model, ctx, userCred, query, isHead)
}

func (dispatcher *DBModelDispatcher) GetSpecific(ctx context.Context, idStr string, spec string, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	userCred := fetchUserCredential(ctx)
	model, err := fetchItem(dispatcher.modelManager, ctx, userCred, idStr, query)
	if err == sql.ErrNoRows {
		return nil, httperrors.NewResourceNotFoundError2(dispatcher.modelManager.Keyword(), idStr)
	} else if err != nil {
		return nil, err
	}

	params := []interface{}{ctx, userCred, query}

	specCamel := utils.Kebab2Camel(spec, "-")
	modelValue := reflect.ValueOf(model)

	if consts.IsRbacEnabled() {
		err := isObjectRbacAllowed(model, userCred, policy.PolicyActionGet, spec)
		if err != nil {
			return nil, err
		}
	} else {
		funcName := fmt.Sprintf("AllowGetDetails%s", specCamel)

		funcValue := modelValue.MethodByName(funcName)
		if !funcValue.IsValid() || funcValue.IsNil() {
			return nil, httperrors.NewSpecNotFoundError("%s %s %s not found", dispatcher.Keyword(), idStr, spec)
		}

		outs, err := callFunc(funcValue, params...)
		if err != nil {
			return nil, err
		}
		if len(outs) != 1 {
			return nil, httperrors.NewInternalServerError("Invald %s return value", funcName)
		}
		if !outs[0].Bool() {
			return nil, httperrors.NewForbiddenError("%s not allow to get spec %s", dispatcher.Keyword(), spec)
		}
	}

	funcName := fmt.Sprintf("GetDetails%s", specCamel)
	funcValue := modelValue.MethodByName(funcName)
	if !funcValue.IsValid() || funcValue.IsNil() {
		return nil, httperrors.NewSpecNotFoundError("%s %s %s not found", dispatcher.Keyword(), idStr, spec)
	}

	outs, err := callFunc(funcValue, params...)
	if err != nil {
		return nil, err
	}
	if len(outs) != 2 {
		return nil, httperrors.NewInternalServerError("Invald %s return value", funcName)
	}

	resVal := outs[0]
	errVal := outs[1].Interface()
	if !gotypes.IsNil(errVal) {
		return nil, errVal.(error)
	} else {
		if gotypes.IsNil(resVal.Interface()) {
			return nil, nil
		} else {
			return ValueToJSONObject(resVal), nil
		}
	}
}

func fetchOwnerId(ctx context.Context, manager IModelManager, userCred mcclient.TokenCredential, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	var ownerId mcclient.IIdentityProvider
	var err error
	if manager.ResourceScope() != rbacutils.ScopeSystem {
		ownerId, err = manager.FetchOwnerId(ctx, data)
		if err != nil {
			return nil, httperrors.NewGeneralError(err)
		}
	}
	if ownerId == nil {
		ownerId = userCred
	}
	return ownerId, nil
}

/*func fetchOwnerProjectId(ctx context.Context, manager IModelManager, userCred mcclient.TokenCredential, data jsonutils.JSONObject) (string, error) {
	var projId string
	if data != nil {
		// projId = jsonutils.GetAnyString(data, []string{"project", "tenant", "project_id", "tenant_id"})
		projId = manager.FetchOwnerId(data)
	}
	ownerProjId := manager.GetOwnerId(userCred)
	if len(projId) == 0 {
		return ownerProjId, nil
	}
	tid, err := manager.ValidateOwnerId(ctx, projId)
	if err != nil {
		return "", httperrors.NewInputParameterError("Invalid owner %s", projId)
	}
	if tid == ownerProjId {
		return ownerProjId, nil
	}
	var isAllow bool
	if consts.IsRbacEnabled() {
		result := policy.PolicyManager.Allow(true, userCred,
			consts.GetServiceType(), policy.PolicyDelegation, "")
		if result == rbacutils.AdminAllow {
			isAllow = true
		}
	} else {
		isAllow = userCred.IsAdminAllow(consts.GetServiceType(), policy.PolicyDelegation, "")
	}
	if !isAllow {
		return "", httperrors.NewForbiddenError("Delegation not allowed")
	}
	return tid, nil
}*/

func NewModelObject(modelManager IModelManager) (IModel, error) {
	m, ok := reflect.New(modelManager.TableSpec().DataType()).Interface().(IModel)
	if !ok {
		return nil, ErrInconsistentDataType
	}
	m.SetModelManager(modelManager, m)
	return m, nil
}

func FetchModelObjects(modelManager IModelManager, query *sqlchemy.SQuery, targets interface{}) error {
	rows, err := query.Rows()
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}
	defer rows.Close()

	targetsValue := reflect.Indirect(reflect.ValueOf(targets))
	for rows.Next() {
		m, err := NewModelObject(modelManager)
		if err != nil {
			return err
		}
		err = query.Row2Struct(rows, m)
		if err != nil {
			return err
		}
		newTargets := reflect.Append(targetsValue, reflect.Indirect(reflect.ValueOf(m)))
		targetsValue.Set(newTargets)
	}
	return nil
}

func DoCreate(manager IModelManager, ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject, ownerId mcclient.IIdentityProvider) (IModel, error) {
	lockman.LockClass(ctx, manager, GetLockClassKey(manager, ownerId))
	defer lockman.ReleaseClass(ctx, manager, GetLockClassKey(manager, ownerId))

	return doCreateItem(manager, ctx, userCred, ownerId, nil, data)
}

func doCreateItem(
	manager IModelManager, ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) (IModel, error) {

	return _doCreateItem(manager, ctx, userCred, ownerId, query, data, false, 1)
}

func batchCreateDoCreateItem(
	manager IModelManager, ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject, data jsonutils.JSONObject, baseIndex int) (IModel, error) {

	return _doCreateItem(manager, ctx, userCred, ownerId, query, data, true, baseIndex)
}

func _doCreateItem(
	manager IModelManager, ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject, data jsonutils.JSONObject, batchCreate bool, baseIndex int) (IModel, error) {

	dataDict, ok := data.(*jsonutils.JSONDict)
	if !ok {
		log.Errorf("doCreateItem: fail to decode json data %s", data)
		return nil, fmt.Errorf("fail to decode json data %s", data)
	}
	var err error

	generateName, _ := dataDict.GetString("generate_name")
	if len(generateName) > 0 {
		dataDict.Remove("generate_name")
		newName, err := GenerateName2(manager, ownerId, generateName, nil, baseIndex)
		if err != nil {
			return nil, err
		}
		dataDict.Add(jsonutils.NewString(newName), "name")
	} /*else {
		name, _ := data.GetString("name")
		if len(name) > 0 {
			err = NewNameValidator(manager, ownerId, name)
			if err != nil {
				return nil, err
			}
		}
	}*/

	if batchCreate {
		dataDict, err = manager.BatchCreateValidateCreateData(ctx, userCred, ownerId, query, dataDict)
	} else {
		dataDict, err = ValidateCreateData(manager, ctx, userCred, ownerId, query, dataDict)
	}

	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	// run name validation after validate create data
	parentId := manager.FetchParentId(ctx, dataDict)
	name, _ := dataDict.GetString("name")
	if len(name) > 0 {
		err = NewNameValidator(manager, ownerId, name, parentId)
		if err != nil {
			return nil, err
		}
	}

	err = jsonutils.CheckRequiredFields(dataDict, createRequireFields(manager, userCred))
	if err != nil {
		return nil, httperrors.NewInputParameterError(err.Error())
	}
	model, err := NewModelObject(manager)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	filterData := dataDict.CopyIncludes(createFields(manager, userCred)...)
	err = filterData.Unmarshal(model)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	err = model.CustomizeCreate(ctx, userCred, ownerId, query, dataDict)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	err = manager.TableSpec().InsertOrUpdate(model)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	// HACK: set data same as dataDict
	data.(*jsonutils.JSONDict).Update(dataDict)
	return model, nil
}

func (dispatcher *DBModelDispatcher) FetchCreateHeaderData(ctx context.Context, header http.Header) (jsonutils.JSONObject, error) {
	return dispatcher.modelManager.FetchCreateHeaderData(ctx, header)
}

func (dispatcher *DBModelDispatcher) Create(ctx context.Context, query jsonutils.JSONObject, data jsonutils.JSONObject, ctxIds []dispatcher.SResourceContext) (jsonutils.JSONObject, error) {
	userCred := fetchUserCredential(ctx)
	manager := dispatcher.modelManager

	ownerId, err := fetchOwnerId(ctx, manager, userCred, data)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	if len(ctxIds) > 0 {
		dataDict, ok := data.(*jsonutils.JSONDict)
		if !ok {
			log.Errorf("fail to convert body into jsondict")
			return nil, fmt.Errorf("fail to parse body")
		}
		data, err = fetchContextObjectsIds(dispatcher.modelManager, ctx, userCred, ctxIds, dataDict)
		if err != nil {
			log.Errorf("fail to find context object %s", ctxIds)
			return nil, err
		}
	}

	if consts.IsRbacEnabled() {
		err := isClassRbacAllowed(dispatcher.modelManager, userCred, ownerId, policy.PolicyActionCreate)
		if err != nil {
			return nil, err
		}
	} else if !dispatcher.modelManager.AllowCreateItem(ctx, userCred, query, data) {
		return nil, httperrors.NewForbiddenError("Not allow to create item")
	}

	model, err := DoCreate(dispatcher.modelManager, ctx, userCred, query, data, ownerId)
	if err != nil {
		log.Errorf("fail to doCreateItem %s", err)
		return nil, httperrors.NewGeneralError(err)
	}

	func() {
		lockman.LockObject(ctx, model)
		defer lockman.ReleaseObject(ctx, model)

		model.PostCreate(ctx, userCred, ownerId, query, data)
	}()

	OpsLog.LogEvent(model, ACT_CREATE, model.GetShortDesc(ctx), userCred)
	dispatcher.modelManager.OnCreateComplete(ctx, []IModel{model}, userCred, ownerId, query, data)
	return getItemDetails(dispatcher.modelManager, model, ctx, userCred, query)
}

func expandMultiCreateParams(data jsonutils.JSONObject, count int) ([]jsonutils.JSONObject, error) {
	jsonDict, ok := data.(*jsonutils.JSONDict)
	if !ok {
		return nil, httperrors.NewInputParameterError("body is not a json?")
	}
	name, _ := jsonDict.GetString("generate_name")
	if len(name) == 0 {
		name, _ = jsonDict.GetString("name")
		if len(name) == 0 {
			return nil, httperrors.NewInputParameterError("Missing name or generate_name")
		}
		jsonDict.Add(jsonutils.NewString(name), "generate_name")
		jsonDict.RemoveIgnoreCase("name")
	}
	ret := make([]jsonutils.JSONObject, count)
	for i := 0; i < count; i += 1 {
		ret[i] = jsonDict.Copy()
	}
	return ret, nil
}

func (dispatcher *DBModelDispatcher) BatchCreate(ctx context.Context, query jsonutils.JSONObject, data jsonutils.JSONObject, count int, ctxIds []dispatcher.SResourceContext) ([]modulebase.SubmitResult, error) {
	userCred := fetchUserCredential(ctx)

	manager := dispatcher.modelManager

	ownerId, err := fetchOwnerId(ctx, manager, userCred, data)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	if len(ctxIds) > 0 {
		dataDict, ok := data.(*jsonutils.JSONDict)
		if !ok {
			return nil, fmt.Errorf("fail to parse body")
		}
		data, err = fetchContextObjectsIds(manager, ctx, userCred, ctxIds, dataDict)
		if err != nil {
			return nil, err
		}
	}

	if consts.IsRbacEnabled() {
		err := isClassRbacAllowed(manager, userCred, ownerId, policy.PolicyActionCreate)
		if err != nil {
			return nil, err
		}
	} else if !manager.AllowCreateItem(ctx, userCred, query, data) {
		return nil, httperrors.NewForbiddenError("Not allow to create item")
	}

	type sCreateResult struct {
		model IModel
		err   error
	}

	var (
		multiData         []jsonutils.JSONObject
		onBatchCreateFail func()
		validateError     error
	)

	createResults, err := func() ([]sCreateResult, error) {
		lockman.LockClass(ctx, manager, GetLockClassKey(manager, ownerId))
		defer lockman.ReleaseClass(ctx, manager, GetLockClassKey(manager, ownerId))

		multiData, err = expandMultiCreateParams(data, count)
		if err != nil {
			return nil, err
		}

		ret := make([]sCreateResult, len(multiData))
		for i, cdata := range multiData {
			if i == 0 {
				onBatchCreateFail, validateError = manager.BatchPreValidate(
					ctx, userCred, ownerId, query, cdata.(*jsonutils.JSONDict), len(multiData))
				if validateError != nil {
					return nil, validateError
				}
			}
			model, err := batchCreateDoCreateItem(manager, ctx, userCred, ownerId, query, cdata, i)
			if err != nil && onBatchCreateFail != nil {
				onBatchCreateFail()
			}
			ret[i] = sCreateResult{model: model, err: err}
		}
		return ret, nil
	}()

	if err != nil {
		return nil, err
	}
	results := make([]modulebase.SubmitResult, count)
	models := make([]IModel, 0)
	for i, res := range createResults {
		result := modulebase.SubmitResult{}
		if res.err != nil {
			jsonErr, ok := res.err.(*httputils.JSONClientError)
			if ok {
				result.Status = jsonErr.Code
				result.Data = jsonutils.Marshal(jsonErr)
			} else {
				result.Status = 500
				result.Data = jsonutils.NewString(res.err.Error())
			}
		} else {
			lockman.LockObject(ctx, res.model)
			defer lockman.ReleaseObject(ctx, res.model)

			res.model.PostCreate(ctx, userCred, ownerId, query, data)

			models = append(models, res.model)
			body, err := getItemDetails(manager, res.model, ctx, userCred, query)
			if err != nil {
				result.Status = 500
				result.Data = jsonutils.NewString(err.Error())
			} else {
				result.Status = 200
				result.Data = body
			}
		}
		results[i] = result
	}
	if len(models) > 0 {
		lockman.LockClass(ctx, manager, GetLockClassKey(manager, ownerId))
		defer lockman.ReleaseClass(ctx, manager, GetLockClassKey(manager, ownerId))

		manager.OnCreateComplete(ctx, models, userCred, ownerId, query, multiData[0])
	}
	return results, nil
}

func managerPerformCheckCreateData(
	manager IModelManager,
	ctx context.Context,
	userCred mcclient.TokenCredential,
	action string,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	body, err := data.(*jsonutils.JSONDict).Get(manager.Keyword())
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	bodyDict := body.(*jsonutils.JSONDict)

	if consts.IsRbacEnabled() {
		err := isClassRbacAllowed(manager, userCred, ownerId, policy.PolicyActionPerform, action)
		if err != nil {
			return nil, err
		}
	} else if !manager.AllowPerformCheckCreateData(ctx, userCred, query, data) {
		return nil, httperrors.NewForbiddenError("not allow to perform %s", action)
	}

	return ValidateCreateData(manager, ctx, userCred, ownerId, query, bodyDict)
}

func (dispatcher *DBModelDispatcher) PerformClassAction(ctx context.Context, action string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	userCred := fetchUserCredential(ctx)
	manager := dispatcher.modelManager

	ownerId, err := fetchOwnerId(ctx, manager, userCred, data)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	lockman.LockClass(ctx, manager, GetLockClassKey(manager, ownerId))
	defer lockman.ReleaseClass(ctx, manager, GetLockClassKey(manager, ownerId))

	if action == "check-create-data" {
		return managerPerformCheckCreateData(dispatcher.modelManager,
			ctx, userCred, action, ownerId, query, data)
	}

	managerValue := reflect.ValueOf(dispatcher.modelManager)
	return objectPerformAction(dispatcher, nil, managerValue, ctx, userCred, action, query, data)
}

func (dispatcher *DBModelDispatcher) PerformAction(ctx context.Context, idStr string, action string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	userCred := fetchUserCredential(ctx)
	model, err := fetchItem(dispatcher.modelManager, ctx, userCred, idStr, nil)
	if err == sql.ErrNoRows {
		return nil, httperrors.NewResourceNotFoundError2(dispatcher.modelManager.Keyword(), idStr)
	} else if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	lockman.LockObject(ctx, model)
	defer lockman.ReleaseObject(ctx, model)

	return objectPerformAction(dispatcher, model, reflect.ValueOf(model), ctx, userCred, action, query, data)
}

func objectPerformAction(dispatcher *DBModelDispatcher, model IModel, modelValue reflect.Value, ctx context.Context, userCred mcclient.TokenCredential, action string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return reflectDispatcher(dispatcher, model, modelValue, ctx, userCred, policy.PolicyActionPerform, "PerformAction", "Perform", action, query, data)
}

func reflectDispatcher(
	dispatcher *DBModelDispatcher,
	model IModel,
	modelValue reflect.Value,
	ctx context.Context,
	userCred mcclient.TokenCredential,
	operator string,
	generalFuncName string,
	funcPrefix string,
	spec string,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	result, err := reflectDispatcherInternal(
		dispatcher, model, modelValue, ctx, userCred, operator, generalFuncName, funcPrefix, spec, query, data)
	if model != nil && err == nil && result == nil {
		return getItemDetails(dispatcher.modelManager, model, ctx, userCred, query)
	} else {
		return result, err
	}
}
func reflectDispatcherInternal(
	dispatcher *DBModelDispatcher,
	model IModel,
	modelValue reflect.Value,
	ctx context.Context,
	userCred mcclient.TokenCredential,
	operator string,
	generalFuncName string,
	funcPrefix string,
	spec string,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	isGeneral := false
	funcName := fmt.Sprintf("%s%s", funcPrefix, utils.Kebab2Camel(spec, "-"))
	funcValue := modelValue.MethodByName(funcName)

	if !funcValue.IsValid() || funcValue.IsNil() {
		funcValue = modelValue.MethodByName(generalFuncName)
		if !funcValue.IsValid() || funcValue.IsNil() {
			msg := fmt.Sprintf("%s %s %s not found", dispatcher.Keyword(), operator, spec)
			log.Errorf(msg)
			return nil, httperrors.NewActionNotFoundError(msg)
		} else {
			isGeneral = true
			funcName = generalFuncName
		}
	}

	var params []interface{}

	if isGeneral {
		params = []interface{}{ctx, userCred, spec, query, data}
	} else {
		params = []interface{}{ctx, userCred, query, data}
	}

	if consts.IsRbacEnabled() {
		if model == nil {
			ownerId, err := fetchOwnerId(ctx, dispatcher.modelManager, userCred, data)
			if err != nil {
				return nil, httperrors.NewGeneralError(err)
			}
			err = isClassRbacAllowed(dispatcher.modelManager, userCred, ownerId, operator, spec)
			if err != nil {
				return nil, err
			}
		} else {
			err := isObjectRbacAllowed(model, userCred, operator, spec)
			if err != nil {
				return nil, err
			}
		}
	} else {
		allowFuncName := "Allow" + funcName
		allowFuncValue := modelValue.MethodByName(allowFuncName)
		if !allowFuncValue.IsValid() || allowFuncValue.IsNil() {
			msg := fmt.Sprintf("%s allow %s %s not found", dispatcher.Keyword(), operator, spec)
			log.Errorf(msg)
			return nil, httperrors.NewActionNotFoundError(msg)
		}

		outs, err := callFunc(allowFuncValue, params...)
		if err != nil {
			return nil, err
		}
		if len(outs) != 1 {
			return nil, httperrors.NewInternalServerError("Invald %s return value", allowFuncName)
		}

		if !outs[0].Bool() {
			return nil, httperrors.NewForbiddenError("%s not allow to %s %s", dispatcher.Keyword(), operator, spec)
		}
	}

	outs, err := callFunc(funcValue, params...)
	if err != nil {
		return nil, err
	}
	if len(outs) != 2 {
		return nil, httperrors.NewInternalServerError("Invald %s return value", funcName)
	}
	resVal := outs[0]
	errVal := outs[1].Interface()
	if !gotypes.IsNil(errVal) {
		return nil, errVal.(error)
	} else {
		if gotypes.IsNil(resVal.Interface()) {
			return nil, nil
		} else {
			return ValueToJSONObject(resVal), nil
		}
	}
}

func updateItem(manager IModelManager, item IModel, ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	var err error

	err = item.ValidateUpdateCondition(ctx)

	if err != nil {
		log.Errorf("validate update condition error: %s", err)
		return nil, httperrors.NewGeneralError(err)
	}

	dataDict, ok := data.(*jsonutils.JSONDict)
	if !ok {
		return nil, httperrors.NewInternalServerError("Invalid data JSONObject")
	}

	name, _ := data.GetString("name")
	if len(name) > 0 {
		err = alterNameValidator(item, name)
		if err != nil {
			return nil, err
		}
	}

	dataDict, err = ValidateUpdateData(item, ctx, userCred, query, dataDict)
	if err != nil {
		errMsg := fmt.Sprintf("validate update data error: %s", err)
		log.Errorf(errMsg)
		return nil, httperrors.NewGeneralError(err)
	}

	item.PreUpdate(ctx, userCred, query, dataDict)

	diff, err := Update(item, func() error {
		filterData := dataDict.CopyIncludes(updateFields(manager, userCred)...)
		err = filterData.Unmarshal(item)
		if err != nil {
			errMsg := fmt.Sprintf("unmarshal fail: %s", err)
			log.Errorf(errMsg)
			return httperrors.NewGeneralError(err)
		}
		return nil
	})

	if err != nil {
		log.Errorf("save update error: %s", err)
		return nil, httperrors.NewGeneralError(err)
	}
	OpsLog.LogEvent(item, ACT_UPDATE, diff, userCred)

	item.PostUpdate(ctx, userCred, query, data)

	return getItemDetails(manager, item, ctx, userCred, query)
}

func (dispatcher *DBModelDispatcher) FetchUpdateHeaderData(ctx context.Context, header http.Header) (jsonutils.JSONObject, error) {
	return dispatcher.modelManager.FetchUpdateHeaderData(ctx, header)
}

func (dispatcher *DBModelDispatcher) Update(ctx context.Context, idStr string, query jsonutils.JSONObject, data jsonutils.JSONObject, ctxIds []dispatcher.SResourceContext) (jsonutils.JSONObject, error) {
	userCred := fetchUserCredential(ctx)
	model, err := fetchItem(dispatcher.modelManager, ctx, userCred, idStr, nil)
	if err == sql.ErrNoRows {
		return nil, httperrors.NewResourceNotFoundError2(dispatcher.modelManager.Keyword(), idStr)
	} else if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	if consts.IsRbacEnabled() {
		err := isObjectRbacAllowed(model, userCred, policy.PolicyActionUpdate)
		if err != nil {
			return nil, err
		}
	} else if !model.AllowUpdateItem(ctx, userCred) {
		return nil, httperrors.NewForbiddenError("Not allow to update item")
	}

	if len(ctxIds) > 0 {
		ctxObjs, err := fetchContextObjects(dispatcher.modelManager, ctx, userCred, ctxIds)
		if err != nil {
			return nil, httperrors.NewGeneralError(err)
		}

		return model.UpdateInContext(ctx, userCred, ctxObjs, query, data)
	} else {
		lockman.LockObject(ctx, model)
		defer lockman.ReleaseObject(ctx, model)

		return updateItem(dispatcher.modelManager, model, ctx, userCred, query, data)
	}
}

func (dispatcher *DBModelDispatcher) UpdateSpec(ctx context.Context, idStr string, spec string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	userCred := fetchUserCredential(ctx)
	model, err := fetchItem(dispatcher.modelManager, ctx, userCred, idStr, nil)
	if err == sql.ErrNoRows {
		return nil, httperrors.NewResourceNotFoundError2(dispatcher.modelManager.Keyword(), idStr)
	} else if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	lockman.LockObject(ctx, model)
	defer lockman.ReleaseObject(ctx, model)

	return objectUpdateSpec(dispatcher, model, reflect.ValueOf(model), ctx, userCred, spec, query, data)
}

func objectUpdateSpec(dispatcher *DBModelDispatcher, model IModel, modelValue reflect.Value, ctx context.Context, userCred mcclient.TokenCredential, spec string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return reflectDispatcher(dispatcher, model, modelValue, ctx, userCred, policy.PolicyActionUpdate, "UpdateSpec", "Update", spec, query, data)
}

func DeleteModel(ctx context.Context, userCred mcclient.TokenCredential, item IModel) error {
	// log.Debugf("Ready to delete %s %s %#v", jsonutils.Marshal(item), item, manager)
	cleanModelUsages(ctx, userCred, item)
	_, err := Update(item, func() error {
		return item.MarkDelete()
	})
	if err != nil {
		msg := fmt.Sprintf("save update error %s", err)
		log.Errorf(msg)
		return httperrors.NewGeneralError(err)
	}
	if userCred != nil {
		OpsLog.LogEvent(item, ACT_DELETE, item.GetShortDesc(ctx), userCred)
	}
	return nil
}

func deleteItem(manager IModelManager, model IModel, ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	// log.Debugf("deleteItem %s", jsonutils.Marshal(model))
	err := model.ValidateDeleteCondition(ctx)
	if err != nil {
		log.Errorf("validate delete condition error: %s", err)
		return nil, err
	}

	err = CustomizeDelete(model, ctx, userCred, query, data)
	if err != nil {
		log.Errorf("customize delete error: %s", err)
		return nil, httperrors.NewNotAcceptableError(err.Error())
	}

	details, err := getItemDetails(manager, model, ctx, userCred, query)
	if err != nil {
		log.Errorf("fail to get item detail before delete: %s", err)
		return nil, httperrors.NewGeneralError(err)
	}

	model.PreDelete(ctx, userCred)

	// err = DeleteModel(ctx, userCred, model)
	err = model.Delete(ctx, userCred)
	if err != nil {
		log.Errorf("Delete error %s", err)
		return nil, err
	}

	model.PostDelete(ctx, userCred)

	return details, nil
}

func (dispatcher *DBModelDispatcher) Delete(ctx context.Context, idstr string, query jsonutils.JSONObject, data jsonutils.JSONObject, ctxIds []dispatcher.SResourceContext) (jsonutils.JSONObject, error) {
	userCred := fetchUserCredential(ctx)
	model, err := fetchItem(dispatcher.modelManager, ctx, userCred, idstr, nil)
	if err == sql.ErrNoRows {
		return nil, httperrors.NewResourceNotFoundError2(dispatcher.modelManager.Keyword(), idstr)
	} else if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	// log.Debugf("Delete %s", model.GetShortDesc(ctx))

	if consts.IsRbacEnabled() {
		err := isObjectRbacAllowed(model, userCred, policy.PolicyActionDelete)
		if err != nil {
			return nil, err
		}
	} else if !model.AllowDeleteItem(ctx, userCred, query, data) {
		return nil, httperrors.NewForbiddenError("%s(%s) not allow to delete", dispatcher.modelManager.KeywordPlural(), model.GetId())
	}

	if len(ctxIds) > 0 {
		ctxObjs, err := fetchContextObjects(dispatcher.modelManager, ctx, userCred, ctxIds)
		if err != nil {
			return nil, httperrors.NewGeneralError(err)
		}

		return model.DeleteInContext(ctx, userCred, ctxObjs, query, data)
	} else {
		lockman.LockObject(ctx, model)
		defer lockman.ReleaseObject(ctx, model)

		return deleteItem(dispatcher.modelManager, model, ctx, userCred, query, data)
	}
}

func (dispatcher *DBModelDispatcher) DeleteSpec(ctx context.Context, idstr string, spec string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	userCred := fetchUserCredential(ctx)
	model, err := fetchItem(dispatcher.modelManager, ctx, userCred, idstr, nil)
	if err == sql.ErrNoRows {
		return nil, httperrors.NewResourceNotFoundError2(dispatcher.modelManager.Keyword(), idstr)
	} else if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	lockman.LockObject(ctx, model)
	defer lockman.ReleaseObject(ctx, model)

	return objectDeleteSpec(dispatcher, model, reflect.ValueOf(model), ctx, userCred, spec, query, data)
}

func objectDeleteSpec(dispatcher *DBModelDispatcher, model IModel, modelValue reflect.Value, ctx context.Context, userCred mcclient.TokenCredential, spec string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return reflectDispatcher(dispatcher, model, modelValue, ctx, userCred, policy.PolicyActionDelete, "DeleteSpec", "Delete", spec, query, data)
}
