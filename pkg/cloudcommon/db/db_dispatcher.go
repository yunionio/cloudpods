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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/filterclause"
	"yunion.io/x/pkg/util/printutils"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/version"
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
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/onecloud/pkg/util/tagutils"
)

type DBModelDispatcher struct {
	manager IModelManager
}

func NewModelHandler(manager IModelManager) *DBModelDispatcher {
	// registerModelManager(manager)
	return &DBModelDispatcher{manager: manager}
}

func (dispatcher *DBModelDispatcher) Keyword() string {
	return dispatcher.manager.Keyword()
}

func (dispatcher *DBModelDispatcher) KeywordPlural() string {
	return dispatcher.manager.KeywordPlural()
}

func (dispatcher *DBModelDispatcher) ContextKeywordPlurals() [][]string {
	ctxMans := dispatcher.manager.GetContextManagers()
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
	return auth.AuthenticateWithDelayDecision(f, true)
}

func (dispatcher *DBModelDispatcher) CustomizeHandlerInfo(handler *appsrv.SHandlerInfo) {
	dispatcher.manager.CustomizeHandlerInfo(handler)
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
		return nil, errors.Wrapf(err, "query.GetMap %s", query.String())
	}

	listF := searchFields(manager, userCred)
	for key := range qdata {
		if !strings.HasPrefix(key, "@") {
			continue
		}
		fn, op := parseSearchFieldkey(key[1:])
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
						arrV[i] = sqlchemy.GetStringValue(colSpec.ConvertFromString(arrV[i]))
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

func ApplyListItemsGeneralFilters(manager IModelManager, q *sqlchemy.SQuery,
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
			if jointModelManager == nil {
				return nil, httperrors.NewResourceNotFoundError("invalid joint resources %s", jfc.GetJointModelName())
			}
			hasKey := false
			for _, colume := range manager.TableSpec().Columns() {
				if colume.Name() == jfc.OriginKey {
					hasKey = true
				}
			}
			if !hasKey {
				return q, httperrors.NewInputParameterError("invalid joint filter %s, because %s doesn't have %s field", f, manager.Keyword(), jfc.OriginKey)
			}
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

func listItemQueryFiltersRaw(manager IModelManager,
	ctx context.Context, q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	action string,
	doCheckRbac bool,
	useRawQuery bool,
) (*sqlchemy.SQuery, error) {
	ownerId, queryScope, err, policyTagFilters := FetchCheckQueryOwnerScope(ctx, userCred, query, manager, action, doCheckRbac)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	if !policyTagFilters.IsEmpty() {
		query.(*jsonutils.JSONDict).Update(policyTagFilters.Json())
		log.Debugf("policyTagFilers: %s", query)
	}

	if !useRawQuery {
		// Specifically for joint resource, these filters will exclude
		// deleted resources by joining with master/slave tables
		q = manager.FilterByOwner(q, manager, userCred, ownerId, queryScope)
		q = manager.FilterBySystemAttributes(q, userCred, query, queryScope)
		q = manager.FilterByHiddenSystemAttributes(q, userCred, query, queryScope)
	}
	if query.Contains("export_keys") {
		exportKeys, _ := query.GetString("export_keys")
		keys := stringutils2.NewSortedStrings(strings.Split(exportKeys, ","))
		q, err = manager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "ListItemExportKeys")
		}
	}

	// XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
	// TURN ON automatic filter by column name, ONLY if query key starts with @!!!!
	// example: @name=abc&@city=111
	// XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
	q, err = listItemsQueryByColumn(manager, q, userCred, query)
	if err != nil {
		return nil, errors.Wrap(err, "listItemsQueryByColumn")
	}

	searches := jsonutils.GetQueryStringArray(query, "search")
	if len(searches) > 0 {
		q, err = applyListItemsSearchFilters(manager, ctx, q, userCred, searches)
		if err != nil {
			return nil, errors.Wrap(err, "applyListItemsSearchFilters")
		}
	}
	filterAny, _ := query.Bool("filter_any")
	filters := jsonutils.GetQueryStringArray(query, "filter")
	if len(filters) > 0 {
		q, err = ApplyListItemsGeneralFilters(manager, q, userCred, filters, filterAny)
		if err != nil {
			return nil, errors.Wrap(err, "ApplyListItemsGeneralFilters")
		}
	}
	jointFilter := jsonutils.GetQueryStringArray(query, "joint_filter")
	if len(jointFilter) > 0 {
		q, err = applyListItemsGeneralJointFilters(manager, q, userCred, jointFilter, filterAny)
		if err != nil {
			return nil, errors.Wrap(err, "applyListItemsGeneralJointFilters")
		}
	}
	q, err = ListItemFilter(manager, ctx, q, userCred, query)
	if err != nil {
		return nil, errors.Wrap(err, "ListItemFilter")
	}

	if isShowDetails(query) {
		managerVal := reflect.ValueOf(manager)
		fName := "ExtendListQuery"
		funcVal := managerVal.MethodByName(fName)
		if funcVal.IsValid() && !funcVal.IsNil() {
			oldq := q
			fields, _ := GetDetailFields(manager, userCred)
			for _, f := range fields {
				q = q.AppendField(q.Field(f).Label(f))
			}
			q, err = ExtendListQuery(manager, ctx, q, userCred, query)
			if err != nil {
				if errors.Cause(err) != MethodNotFoundError {
					return nil, errors.Wrap(err, "ExtendQuery")
				} else {
					// else ignore
					q = oldq
				}
			} else {
				// force query no details
				query.(*jsonutils.JSONDict).Set("details", jsonutils.JSONFalse)
			}
		}
	}

	return q, nil
}

func isShowDetails(query jsonutils.JSONObject) bool {
	showDetails := false
	showDetailsJson, _ := query.Get("details")
	if showDetailsJson != nil {
		showDetails, _ = showDetailsJson.Bool()
	} else {
		showDetails = true
	}
	return showDetails
}

func listItemQueryFilters(manager IModelManager,
	ctx context.Context, q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	action string,
	doCheckRbac bool,
) (*sqlchemy.SQuery, error) {
	return listItemQueryFiltersRaw(manager, ctx, q, userCred, query, action, doCheckRbac, false)
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
	metaFields, excludeFields := listFields(manager, userCred)
	fieldFilter := jsonutils.GetQueryStringArray(query, "field")
	allowListResult := IsAllowList(rbacscope.ScopeSystem, userCred, manager)
	listF := mergeFields(metaFields, fieldFilter, allowListResult.Result.IsAllow())
	listExcludes, _, _ := stringutils2.Split(stringutils2.NewSortedStrings(excludeFields), listF)

	showDetails := isShowDetails(query)

	var items []interface{}
	extraResults := make([]*jsonutils.JSONDict, 0)
	rows, err := q.Rows()
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	defer rows.Close()

	var exportKeys stringutils2.SSortedStrings
	if query.Contains("export_keys") {
		exportKeyStr, _ := query.GetString("export_keys")
		exportKeys = stringutils2.NewSortedStrings(strings.Split(exportKeyStr, ","))
	}

	for rows.Next() {
		item, err := NewModelObject(manager)
		if err != nil {
			return nil, err
		}
		if len(exportKeys) > 0 {
			rowMap, err := q.Row2Map(rows)
			if err != nil {
				return nil, err
			}
			extraData := jsonutils.NewDict()
			for k, v := range rowMap {
				if len(v) > 0 {
					extraData.Add(jsonutils.NewString(v), k)
				}
			}
			extraKeys := manager.GetExportExtraKeys(ctx, exportKeys, rowMap)
			if extraKeys != nil {
				extraData.Update(extraKeys)
			}
			err = q.RowMap2Struct(rowMap, item)
			if err != nil {
				return nil, err
			}
			extraResults = append(extraResults, extraData)
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

		if err := CheckRecordChecksumConsistent(item); err != nil {
			return nil, err
		}

		items = append(items, item)
	}
	results := make([]*jsonutils.JSONDict, len(items))
	if showDetails || len(exportKeys) > 0 {
		var sortedListFields stringutils2.SSortedStrings
		if len(exportKeys) > 0 {
			sortedListFields = exportKeys
		} else if len(fieldFilter) > 0 {
			sortedListFields = stringutils2.NewSortedStrings(fieldFilter)
		}
		extraRows, err := FetchCustomizeColumns(manager, ctx, userCred, query, items, sortedListFields, true)
		if err != nil {
			return nil, errors.Wrap(err, "FetchCustomizeColumns")
		}

		if len(extraRows) == len(results) {
			for i := range results {
				jsonDict := extraRows[i].CopyExcludes(listExcludes...)
				if i < len(extraResults) {
					extraResults[i].Update(jsonDict)
					jsonDict = extraResults[i]
				}
				if len(fieldFilter) > 0 {
					jsonDict = jsonDict.CopyIncludes(fieldFilter...)
				}
				results[i] = jsonDict
			}
		} else {
			return nil, httperrors.NewInternalServerError("FetchCustomizeColumns return incorrect number of results")
		}
	} else {
		for i := range items {
			jsonDict := jsonutils.Marshal(items[i]).(*jsonutils.JSONDict).CopyExcludes(listExcludes...)
			if len(fieldFilter) > 0 {
				jsonDict = jsonDict.CopyIncludes(fieldFilter...)
			}
			results[i] = jsonDict
		}
	}
	for i := range items {
		i18nDict := items[i].(IModel).GetI18N(ctx)
		if i18nDict != nil {
			jsonDict := results[i]
			jsonDict.Set("_i18n", i18nDict)
		}
	}
	r := make([]jsonutils.JSONObject, len(items))
	for i := range results {
		r[i] = results[i]
	}
	return r, nil
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

func ListItems(manager IModelManager, ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, ctxIds []dispatcher.SResourceContext) (*printutils.ListResult, error) {
	// 获取常规参数
	var err error
	var maxLimit int64 = consts.GetMaxPagingLimit()
	limit, _ := query.Int("limit")
	forceNoPaging := jsonutils.QueryBoolean(query, "force_no_paging", false)
	offset, _ := query.Int("offset")
	pagingMarker, _ := query.GetString("paging_marker")
	pagingOrderStr, _ := query.GetString("paging_order")
	pagingOrder := sqlchemy.QueryOrderType(strings.ToUpper(pagingOrderStr))

	// export data only
	exportLimit, err := query.Int("export_limit")
	if query.Contains("export_keys") && err == nil {
		limit = exportLimit
	}

	var (
		q           *sqlchemy.SQuery
		useRawQuery bool
	)
	{
		// query senders are responsible for clear up other constraint
		// like setting "pendinge_delete" to "all"
		queryDelete, _ := query.GetString("delete")
		if queryDelete == "all" && policy.PolicyManager.Allow(rbacscope.ScopeSystem, userCred, consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionList).Result.IsAllow() {
			useRawQuery = true
		}
	}

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

	pagingConf := manager.GetPagingConfig()
	if pagingConf == nil {
		if limit <= 0 {
			limit = consts.GetDefaultPagingLimit()
		}
	} else {
		if limit <= 0 {
			limit = int64(pagingConf.DefaultLimit)
		}
		if pagingOrder != sqlchemy.SQL_ORDER_ASC && pagingOrder != sqlchemy.SQL_ORDER_DESC {
			pagingOrder = pagingConf.Order
		}
	}

	splitable := manager.GetSplitTable()
	if splitable != nil {
		// handle splitable query, query each subtable, then union results
		metas, err := splitable.GetTableMetas()
		if err != nil {
			return nil, errors.Wrap(err, "splitable.GetTableMetas")
		}
		var subqs []sqlchemy.IQuery
		for _, meta := range metas {
			ts := splitable.GetTableSpec(meta)
			subq := ts.Query()
			subq, err = listItemQueryFiltersRaw(manager, ctx, subq, userCred, queryDict.Copy(), policy.PolicyActionList, true, useRawQuery)
			if err != nil {
				return nil, errors.Wrap(err, "listItemQueryFiltersRaw")
			}
			if pagingConf != nil {
				if limit > 0 {
					subq = subq.Limit(int(limit) + 1)
				}
				if len(pagingMarker) > 0 {
					markers := decodePagingMarker(pagingMarker)
					for markerIdx, marker := range markers {
						if markerIdx < len(pagingConf.MarkerFields) {
							if pagingOrder == sqlchemy.SQL_ORDER_ASC {
								subq = subq.GE(pagingConf.MarkerFields[markerIdx], marker)
							} else {
								subq = subq.LE(pagingConf.MarkerFields[markerIdx], marker)
							}
						}
					}
				}
				for _, f := range pagingConf.MarkerFields {
					if pagingOrder == sqlchemy.SQL_ORDER_ASC {
						subq = subq.Asc(f)
					} else {
						subq = subq.Desc(f)
					}
				}
			}
			if limit > 0 {
				subq = subq.Limit(int(limit) + 1)
			}
			subqs = append(subqs, subq)
		}
		union, err := sqlchemy.UnionWithError(subqs...)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				emptyList := printutils.ListResult{Data: []jsonutils.JSONObject{}}
				return &emptyList, nil
			} else {
				return nil, errors.Wrap(err, "sqlchemy.UnionWithError")
			}
		}
		q = union.Query()
	} else {
		if useRawQuery {
			q = manager.RawQuery()
		} else {
			q = manager.Query()
		}
	}

	q, err = listItemQueryFiltersRaw(manager, ctx, q, userCred, queryDict, policy.PolicyActionList, true, useRawQuery)
	if err != nil {
		return nil, errors.Wrap(err, "listItemQueryFiltersRaw")
	}

	var totalCnt int
	var totalJson jsonutils.JSONObject
	if pagingConf == nil {
		// calculate total
		totalQ := q.CountQuery()
		totalCnt, totalJson, err = manager.CustomizedTotalCount(ctx, userCred, query, totalQ)
		if err != nil {
			return nil, errors.Wrap(err, "CustomizedTotalCount")
		}
		//log.Debugf("total count %d", totalCnt)
		if totalCnt == 0 {
			emptyList := printutils.ListResult{Data: []jsonutils.JSONObject{}}
			return &emptyList, nil
		}
	}
	if int64(totalCnt) > maxLimit && (limit <= 0 || limit > maxLimit) && !forceNoPaging {
		limit = maxLimit
	}

	// orders defined in pagingConf should have the highest priority
	if pagingConf != nil {
		for _, f := range pagingConf.MarkerFields {
			if pagingOrder == sqlchemy.SQL_ORDER_ASC {
				q = q.Asc(f)
			} else {
				q = q.Desc(f)
			}
		}
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
		if pagingConf != nil && utils.IsInStringArray(orderByField, pagingConf.MarkerFields) {
			// skip markerField in pagingConf
			continue
		}
		colSpec := manager.TableSpec().ColumnSpec(orderByField)
		if colSpec == nil {
			orderQuery.Set(fmt.Sprintf("order_by_%s", orderByField), jsonutils.NewString(string(order)))
		}
	}
	q, err = OrderByExtraFields(manager, ctx, q, userCred, orderQuery)
	if err != nil {
		return nil, errors.Wrap(err, "OrderByExtraFields")
	}
	if orderBy == nil {
		orderBy = []string{}
	}
	if !q.IsGroupBy() {
		if primaryCol != nil && primaryCol.IsNumeric() {
			orderBy = append(orderBy, primaryCol.Name())
		} else if manager.TableSpec().ColumnSpec("created_at") != nil {
			orderBy = append(orderBy, "created_at")
			if manager.TableSpec().ColumnSpec("name") != nil {
				orderBy = append(orderBy, "name")
			}
			if primaryCol != nil {
				orderBy = append(orderBy, primaryCol.Name())
			}
		}
	}
	for _, orderByField := range orderBy {
		if pagingConf != nil && utils.IsInStringArray(orderByField, pagingConf.MarkerFields) {
			// skip markerField in pagingConf
			continue
		}
		for _, field := range q.QueryFields() {
			if orderByField == field.Name() {
				if order == sqlchemy.SQL_ORDER_ASC {
					q = q.Asc(field)
				} else {
					q = q.Desc(field)
				}
				break
			}
		}
	}

	if forceNoPaging {
		limit = 0
	}

	if pagingConf != nil {
		if limit > 0 {
			q = q.Limit(int(limit) + 1)
		}
		if len(pagingMarker) > 0 {
			markers := decodePagingMarker(pagingMarker)
			for markerIdx, marker := range markers {
				if markerIdx < len(pagingConf.MarkerFields) {
					if pagingOrder == sqlchemy.SQL_ORDER_ASC {
						q = q.GE(pagingConf.MarkerFields[markerIdx], marker)
					} else {
						q = q.LE(pagingConf.MarkerFields[markerIdx], marker)
					}
				}
			}
		}
		retList, err := Query2List(manager, ctx, userCred, q, queryDict, false)
		if err != nil {
			return nil, httperrors.NewGeneralError(err)
		}
		nextMarkers := make([]string, 0)
		if limit > 0 && int64(len(retList)) > limit {
			for _, markerField := range pagingConf.MarkerFields {
				nextMarker, _ := retList[limit].GetString(markerField)
				nextMarkers = append(nextMarkers, nextMarker)
			}
			retList = retList[:limit]
		}
		nextMarker := encodePagingMarker(nextMarkers)
		retResult := printutils.ListResult{
			Data: retList, Limit: int(limit),
			NextMarker:  nextMarker,
			MarkerField: strings.Join(pagingConf.MarkerFields, ","),
			MarkerOrder: string(pagingOrder),
		}
		return &retResult, nil
	}

	customizeFilters, err := manager.CustomizeFilterList(ctx, q, userCred, queryDict)
	if err != nil {
		return nil, errors.Wrap(err, "CustomizeFilterList")
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
	if query.Contains("export_keys") {
		retList = getExportCols(query, retList)
	}
	paginate := false
	if !customizeFilters.IsEmpty() {
		// query not use Limit and Offset, do manual pagination
		paginate = true
	}
	return calculateListResult(retList, totalCnt, totalJson, int(limit), int(offset), paginate), nil
}

// 构造list返回详情
func calculateListResult(data []jsonutils.JSONObject, total int, totalJson jsonutils.JSONObject, limit, offset int, paginate bool) *printutils.ListResult {
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
		if limit > 0 && total-offset > limit {
			data = data[:limit]
		} else {
			data = data[:total-offset]
		}
	}

	retResult := printutils.ListResult{
		Data:   data,
		Total:  total,
		Limit:  limit,
		Offset: offset,
		Totals: totalJson,
	}

	return &retResult
}

func getExportCols(query jsonutils.JSONObject, retList []jsonutils.JSONObject) []jsonutils.JSONObject {
	var exportKeys stringutils2.SSortedStrings
	exportKeyStr, _ := query.GetString("export_keys")
	exportKeys = stringutils2.NewSortedStrings(strings.Split(exportKeyStr, ","))
	for i := range retList {
		if len(exportKeys) > 0 {
			retList[i] = retList[i].(*jsonutils.JSONDict).CopyIncludes(exportKeys...)
		}
	}
	return retList
}

func (dispatcher *DBModelDispatcher) List(ctx context.Context, query jsonutils.JSONObject, ctxIds []dispatcher.SResourceContext) (*printutils.ListResult, error) {
	// 获取用户信息
	userCred := fetchUserCredential(ctx)
	manager := dispatcher.manager.GetImmutableInstance(ctx, userCred, query)

	// list详情
	items, err := ListItems(manager, ctx, userCred, query, ctxIds)
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrapf(err, "ListItems"))
	}

	if userCred != nil && userCred.HasSystemAdminPrivilege() && manager.ListSkipLog(ctx, userCred, query) {
		appParams := appsrv.AppContextGetParams(ctx)
		if appParams != nil {
			appParams.SkipLog = true
		}
	}
	return items, nil
}

func getModelItemDetails(manager IModelManager, item IModel, ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isHead bool) (jsonutils.JSONObject, error) {
	appParams := appsrv.AppContextGetParams(ctx)
	if appParams == nil && isHead {
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

func GetItemDetails(manager IModelManager, item IModel, ctx context.Context, userCred mcclient.TokenCredential) (jsonutils.JSONObject, error) {
	return getItemDetails(manager, item, ctx, userCred, jsonutils.NewDict())
}

func getItemDetails(manager IModelManager, item IModel, ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	metaFields, excludeFields := GetDetailFields(manager, userCred)
	fieldFilter := jsonutils.GetQueryStringArray(query, "field")

	var sortedListFields stringutils2.SSortedStrings
	if len(fieldFilter) > 0 {
		sortedListFields = stringutils2.NewSortedStrings(fieldFilter)
	}
	extraRows, err := FetchCustomizeColumns(manager, ctx, userCred, query, []interface{}{item}, sortedListFields, false)
	if err != nil {
		return nil, errors.Wrap(err, "FetchCustomizeColumns")
	}
	if len(extraRows) == 1 {
		getFields := mergeFields(metaFields, fieldFilter, IsAllowGet(ctx, rbacscope.ScopeSystem, userCred, item))
		excludes, _, _ := stringutils2.Split(stringutils2.NewSortedStrings(excludeFields), getFields)
		return extraRows[0].CopyExcludes(excludes...), nil
	}
	return nil, httperrors.NewInternalServerError("FetchCustomizeColumns returns incorrect results(expect 1 actual %d)", len(extraRows))
}

func (dispatcher *DBModelDispatcher) tryGetModelProperty(ctx context.Context, property string, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	userCred := fetchUserCredential(ctx)
	funcName := fmt.Sprintf("GetProperty%s", utils.Kebab2Camel(property, "-"))
	manager := dispatcher.manager.GetImmutableInstance(ctx, userCred, query)
	modelValue := reflect.ValueOf(manager)
	// params := []interface{}{ctx, userCred, query}

	funcValue := modelValue.MethodByName(funcName)
	if !funcValue.IsValid() || funcValue.IsNil() {
		return nil, nil
	}

	// _, _, err, _ := FetchCheckQueryOwnerScope(ctx, userCred, query, manager, policy.PolicyActionList, true)
	// if err != nil {
	// 	return nil, err
	// }

	outs, err := callFunc(funcValue, funcName, ctx, userCred, query)
	if err != nil {
		return nil, httperrors.NewInternalServerError("reflect call %s fail %s", funcName, err)
	}
	if len(outs) != 2 {
		return nil, httperrors.NewInternalServerError("Invald %s return value", funcName)
	}

	resVal := outs[0]
	errVal := outs[1].Interface()
	if !gotypes.IsNil(errVal) {
		return nil, errVal.(error)
	} else {
		if gotypes.IsNil(resVal) {
			return nil, httperrors.NewBadRequestError("No return value, so why query?")
		} else {
			return ValueToJSONObject(resVal), nil
		}
	}
}

func (dispatcher *DBModelDispatcher) Get(ctx context.Context, idStr string, query jsonutils.JSONObject, isHead bool) (jsonutils.JSONObject, error) {
	// log.Debugf("Get %s", idStr)
	userCred := fetchUserCredential(ctx)
	manager := dispatcher.manager.GetImmutableInstance(ctx, userCred, query)

	data, err := dispatcher.tryGetModelProperty(ctx, idStr, query)
	if err != nil {
		return nil, err
	} else if data != nil {
		if dataDict, ok := data.(*jsonutils.JSONDict); ok {
			i18nDict := manager.GetI18N(ctx, idStr, data)
			if i18nDict != nil {
				dataDict.Set("_i18n", i18nDict)
			}
		}
		return data, nil
	}

	model, err := fetchItem(manager, ctx, userCred, idStr, query)
	if err == sql.ErrNoRows {
		return nil, httperrors.NewResourceNotFoundError2(manager.Keyword(), idStr)
	} else if err != nil {
		return nil, errors.Wrapf(err, "fetchItem")
	}

	err = isObjectRbacAllowed(ctx, model, userCred, policy.PolicyActionGet)
	if err != nil {
		return nil, err
	}

	if userCred.HasSystemAdminPrivilege() && manager.GetSkipLog(ctx, userCred, query) {
		appParams := appsrv.AppContextGetParams(ctx)
		if appParams != nil {
			appParams.SkipLog = true
		}
	}
	return getModelItemDetails(manager, model, ctx, userCred, query, isHead)
}

func (dispatcher *DBModelDispatcher) GetSpecific(ctx context.Context, idStr string, spec string, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	userCred := fetchUserCredential(ctx)
	manager := dispatcher.manager.GetImmutableInstance(ctx, userCred, query)
	model, err := fetchItem(manager, ctx, userCred, idStr, query)
	if err == sql.ErrNoRows {
		return nil, httperrors.NewResourceNotFoundError2(manager.Keyword(), idStr)
	} else if err != nil {
		return nil, err
	}

	params := []interface{}{ctx, userCred, query}

	specCamel := utils.Kebab2Camel(spec, "-")
	modelValue := reflect.ValueOf(model)

	err = isObjectRbacAllowed(ctx, model, userCred, policy.PolicyActionGet, spec)
	if err != nil {
		return nil, err
	}

	funcName := fmt.Sprintf("GetDetails%s", specCamel)
	funcValue := modelValue.MethodByName(funcName)
	if !funcValue.IsValid() || funcValue.IsNil() {
		return nil, httperrors.NewSpecNotFoundError("%s %s %s not found", dispatcher.Keyword(), idStr, spec)
	}

	outs, err := callFunc(funcValue, funcName, params...)
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
	if manager.ResourceScope() != rbacscope.ScopeSystem {
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

func FetchIModelObjects(modelManager IModelManager, query *sqlchemy.SQuery) ([]IModel, error) {
	// TODO: refactor below duplicated code from FetchModelObjects
	rows, err := query.Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	objs := make([]IModel, 0)
	for rows.Next() {
		m, err := NewModelObject(modelManager)
		if err != nil {
			return nil, err
		}
		err = query.Row2Struct(rows, m)
		if err != nil {
			return nil, err
		}
		objs = append(objs, m)
	}
	return objs, nil
}

func DoCreate(manager IModelManager, ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject, ownerId mcclient.IIdentityProvider) (IModel, error) {
	// 锁住一类实例
	lockman.LockClass(ctx, manager, GetLockClassKey(manager, ownerId))
	defer lockman.ReleaseClass(ctx, manager, GetLockClassKey(manager, ownerId))

	return doCreateItem(manager, ctx, userCred, ownerId, nil, data)
}

func doCreateItem(
	manager IModelManager, ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) (IModel, error) {

	return _doCreateItem(manager, ctx, userCred, ownerId, query, data, false, 1)
}

// 批量创建
func batchCreateDoCreateItem(
	manager IModelManager, ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject, data jsonutils.JSONObject, baseIndex int) (IModel, error) {

	return _doCreateItem(manager, ctx, userCred, ownerId, query, data, true, baseIndex)
}

// 对于modelManager的实际创建过程
func _doCreateItem(
	manager IModelManager, ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject, data jsonutils.JSONObject, batchCreate bool, baseIndex int) (IModel, error) {

	dataDict, ok := data.(*jsonutils.JSONDict)
	if !ok {
		return nil, httperrors.NewGeneralError(fmt.Errorf("fail to decode json data %s", data))
	}

	var err error

	var generateName string
	// 若manager存在name字段且请求包含generate_name,则根据name从数据库中获取相同名称添加后缀
	if manager.HasName() {
		if dataDict.Contains("generate_name") {
			generateName, _ = dataDict.GetString("generate_name")
			if len(generateName) > 0 {
				if manager.EnableGenerateName() {
					lockman.LockRawObject(ctx, manager.Keyword(), "name")
					defer lockman.ReleaseRawObject(ctx, manager.Keyword(), "name")

					// if enable generateName, alway generate name
					newName, err := GenerateName2(ctx, manager, ownerId, generateName, nil, baseIndex)
					if err != nil {
						return nil, errors.Wrap(err, "GenerateName2")
					}
					dataDict.Add(jsonutils.NewString(newName), "name")
				} else {
					// if no name but generate_name provided, use generate_name as name instead
					oldName, _ := dataDict.GetString("name")
					if len(oldName) == 0 {
						dataDict.Add(jsonutils.NewString(generateName), "name")
					}
				}
			}
			// cleanup generate_name
			dataDict.Remove("generate_name")
		}
	}

	funcName := "ValidateCreateData"
	if batchCreate {
		funcName = "BatchCreateValidateCreateData"
	}
	// 校验创建请求入参
	dataDict, err = ValidateCreateData(funcName, manager, ctx, userCred, ownerId, query, dataDict)
	if err != nil {
		return nil, err
	}

	// 若manager用于name字段，确保name唯一
	if manager.HasName() {
		// run name validation after validate create data
		uniqValues := manager.FetchUniqValues(ctx, dataDict)
		name, _ := dataDict.GetString("name")
		if len(name) > 0 {
			err = NewNameValidator(manager, ownerId, name, uniqValues)
			if err != nil {
				return nil, err
			}
		}
	}

	// 检查models定义中tag指定required
	err = jsonutils.CheckRequiredFields(dataDict, createRequireFields(manager, userCred))
	if err != nil {
		return nil, httperrors.NewInputParameterError("%v", err)
	}
	// 初始化model
	model, err := NewModelObject(manager)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	// 检查models定义中tag指定create
	filterData := dataDict.CopyIncludes(createFields(manager, userCred)...)
	err = filterData.Unmarshal(model)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	// 实际创建前钩子
	err = model.CustomizeCreate(ctx, userCred, ownerId, query, dataDict)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	dryRun := struct {
		DryRun bool
	}{}
	data.Unmarshal(&dryRun)
	if dryRun.DryRun {
		return model, nil
	}

	// 插入数据库记录
	if manager.CreateByInsertOrUpdate() {
		err = manager.TableSpec().InsertOrUpdate(ctx, model)
	} else {
		err = manager.TableSpec().Insert(ctx, model)
	}
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	if manager.HasName() {
		// HACK: save generateName
		if len(generateName) > 0 && manager.EnableGenerateName() {
			if standaloneMode, ok := model.(IStandaloneModel); ok {
				standaloneMode.SetMetadata(ctx, "generate_name", generateName, userCred)
			}
		}
	}
	// HACK: set data same as dataDict
	data.(*jsonutils.JSONDict).Update(dataDict)
	return model, nil
}

func (dispatcher *DBModelDispatcher) FetchCreateHeaderData(ctx context.Context, header http.Header) (jsonutils.JSONObject, error) {
	return dispatcher.manager.FetchCreateHeaderData(ctx, header)
}

func (dispatcher *DBModelDispatcher) Create(ctx context.Context, query jsonutils.JSONObject, data jsonutils.JSONObject, ctxIds []dispatcher.SResourceContext) (jsonutils.JSONObject, error) {
	// 获取用户信息
	userCred := fetchUserCredential(ctx)
	manager := dispatcher.manager.GetMutableInstance(ctx, userCred, query, data)

	ownerId, err := fetchOwnerId(ctx, manager, userCred, data)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	if len(ctxIds) > 0 {
		dataDict, ok := data.(*jsonutils.JSONDict)
		if !ok {
			return nil, httperrors.NewGeneralError(fmt.Errorf("fail to parse body %s", data))
		}
		data, err = fetchContextObjectsIds(manager, ctx, userCred, ctxIds, dataDict)
		if err != nil {
			return nil, httperrors.NewGeneralError(errors.Wrapf(err, "fetchContextObjectsIds"))
		}
	}

	// 用户角色校验
	var policyResult rbacutils.SPolicyResult
	policyResult, err = isClassRbacAllowed(ctx, manager, userCred, ownerId, policy.PolicyActionCreate)
	if err != nil {
		return nil, errors.Wrap(err, "isClassRbacAllowed")
	}

	// initialize pending usage in context any way
	if InitPendingUsagesInContext != nil {
		ctx = InitPendingUsagesInContext(ctx)
	}
	dryRun := jsonutils.QueryBoolean(data, "dry_run", false)

	// inject tag filters imposed by policy
	data.(*jsonutils.JSONDict).Update(policyResult.Json())

	// 资源实际创建函数
	model, err := DoCreate(manager, ctx, userCred, query, data, ownerId)
	if err != nil {
		// validate failed, clean pending usage
		if CancelPendingUsagesInContext != nil {
			e := CancelPendingUsagesInContext(ctx, userCred)
			if e != nil {
				err = errors.Wrapf(err, "CancelPendingUsagesInContext fail %s", e.Error())
			}
		}
		failErr := manager.OnCreateFailed(ctx, userCred, ownerId, query, data)
		if failErr != nil {
			err = errors.Wrapf(err, failErr.Error())
		}
		return nil, httperrors.NewGeneralError(err)
	}

	// 伪创建
	if dryRun {
		// dry run, clean pending usage
		if CancelPendingUsagesInContext != nil {
			err := CancelPendingUsagesInContext(ctx, userCred)
			if err != nil {
				return nil, errors.Wrap(err, "CancelPendingUsagesInContext")
			}
		}
		return getItemDetails(manager, model, ctx, userCred, query)
	}

	// 资源创建完成后所需执行的任务（创建完成指在数据库中存在数据）
	func() {
		lockman.LockObject(ctx, model)
		defer lockman.ReleaseObject(ctx, model)

		model.PostCreate(ctx, userCred, ownerId, query, data)
	}()

	// 添加操作日志与消息通知
	{
		notes := model.GetShortDesc(ctx)
		OpsLog.LogEvent(model, ACT_CREATE, notes, userCred)
		logclient.AddActionLogWithContext(ctx, model, logclient.ACT_CREATE, notes, userCred, true)
	}
	manager.OnCreateComplete(ctx, []IModel{model}, userCred, ownerId, query, []jsonutils.JSONObject{data})
	return getItemDetails(manager, model, ctx, userCred, query)
}

func expandMultiCreateParams(manager IModelManager,
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
	count int,
) ([]jsonutils.JSONObject, error) {
	jsonDict, ok := data.(*jsonutils.JSONDict)
	if !ok {
		return nil, httperrors.NewInputParameterError("body is not a json?")
	}
	if manager.HasName() {
		name, _ := jsonDict.GetString("generate_name")
		if len(name) == 0 {
			name, _ = jsonDict.GetString("name")
			if len(name) == 0 {
				return nil, httperrors.NewInputParameterError("Missing name or generate_name")
			}
			jsonDict.Add(jsonutils.NewString(name), "generate_name")
			jsonDict.RemoveIgnoreCase("name")
		}
	}
	ret := make([]jsonutils.JSONObject, count)
	for i := 0; i < count; i += 1 {
		input, err := ExpandBatchCreateData(manager, ctx, userCred, ownerId, query, jsonDict.Copy(), i)
		if err != nil {
			if errors.Cause(err) == MethodNotFoundError {
				ret[i] = jsonDict.Copy()
			} else {
				return nil, errors.Wrap(err, "ExpandBatchCreateData")
			}
		} else {
			ret[i] = input
		}
	}
	return ret, nil
}

func (dispatcher *DBModelDispatcher) BatchCreate(ctx context.Context, query jsonutils.JSONObject, data jsonutils.JSONObject, count int, ctxIds []dispatcher.SResourceContext) ([]printutils.SubmitResult, error) {
	userCred := fetchUserCredential(ctx)
	manager := dispatcher.manager.GetMutableInstance(ctx, userCred, query, data)

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

	var policyResult rbacutils.SPolicyResult
	policyResult, err = isClassRbacAllowed(ctx, manager, userCred, ownerId, policy.PolicyActionCreate)
	if err != nil {
		return nil, errors.Wrap(err, "isClassRbacAllowd")
	}

	data.(*jsonutils.JSONDict).Update(policyResult.Json())

	type sCreateResult struct {
		model IModel
		err   error
	}

	var (
		multiData []jsonutils.JSONObject
		// onBatchCreateFail func()
		// validateError error
	)

	if InitPendingUsagesInContext != nil {
		ctx = InitPendingUsagesInContext(ctx)
	}

	createResults, err := func() ([]sCreateResult, error) {
		lockman.LockClass(ctx, manager, GetLockClassKey(manager, ownerId))
		defer lockman.ReleaseClass(ctx, manager, GetLockClassKey(manager, ownerId))

		// invoke only Once
		err = manager.BatchPreValidate(ctx, userCred, ownerId, query, data.(*jsonutils.JSONDict), count)
		if err != nil {
			return nil, errors.Wrap(err, "manager.BatchPreValidate")
		}

		multiData, err = expandMultiCreateParams(manager, ctx, userCred, ownerId, query, data, count)
		if err != nil {
			return nil, errors.Wrap(err, "expandMultiCreateParams")
		}

		// one fail, then all fail
		ret := make([]sCreateResult, len(multiData))
		for i := range multiData {
			var model IModel
			log.Debugf("batchCreateDoCreateItem %d %s", i, multiData[i].String())
			model, err = batchCreateDoCreateItem(manager, ctx, userCred, ownerId, query, multiData[i], i+1)
			if err == nil {
				ret[i] = sCreateResult{model: model, err: nil}
			} else {
				break
			}
		}
		if err != nil {
			for i := range ret {
				if ret[i].model != nil {
					DeleteModel(ctx, userCred, ret[i].model)
					ret[i].model = nil
				}
				ret[i].err = err
			}
			return nil, errors.Wrap(err, "batchCreateDoCreateItem")
		} else {
			return ret, nil
		}
	}()

	if err != nil {
		failErr := manager.OnCreateFailed(ctx, userCred, ownerId, query, data)
		if failErr != nil {
			err = errors.Wrapf(err, failErr.Error())
		}
		return nil, httperrors.NewGeneralError(errors.Wrap(err, "createResults"))
	}

	results := make([]printutils.SubmitResult, count)
	models := make([]IModel, 0)
	for i, res := range createResults {
		result := printutils.SubmitResult{}
		if res.err != nil {
			jsonErr := httperrors.NewGeneralError(res.err)
			result.Status = jsonErr.Code
			result.Data = jsonutils.Marshal(jsonErr)
		} else {
			lockman.LockObject(ctx, res.model)
			defer lockman.ReleaseObject(ctx, res.model)

			res.model.PostCreate(ctx, userCred, ownerId, query, data)

			models = append(models, res.model)
			body, err := getItemDetails(manager, res.model, ctx, userCred, query)
			if err != nil {
				result.Status = 500
				result.Data = jsonutils.NewString(err.Error()) // no translation here
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

		manager.OnCreateComplete(ctx, models, userCred, ownerId, query, multiData)
	}
	return results, nil
}

func (dispatcher *DBModelDispatcher) PerformClassAction(ctx context.Context, action string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	// 伪创建，校验创建参数
	if action == "check-create-data" {
		dataDict := data.(*jsonutils.JSONDict)
		dataDict.Set("dry_run", jsonutils.JSONTrue)
		return dispatcher.Create(ctx, query, dataDict, nil)
	}

	// 获取用户信息
	userCred := fetchUserCredential(ctx)
	manager := dispatcher.manager.GetMutableInstance(ctx, userCred, query, data)

	ownerId, err := fetchOwnerId(ctx, manager, userCred, data)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	// 锁住一类
	lockman.LockClass(ctx, manager, GetLockClassKey(manager, ownerId))
	defer lockman.ReleaseClass(ctx, manager, GetLockClassKey(manager, ownerId))

	managerValue := reflect.ValueOf(manager)
	return objectPerformAction(manager, nil, managerValue, ctx, userCred, action, query, data)
}

func (dispatcher *DBModelDispatcher) PerformAction(ctx context.Context, idStr string, action string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	userCred := fetchUserCredential(ctx)
	manager := dispatcher.manager.GetMutableInstance(ctx, userCred, query, data)
	model, err := fetchItem(manager, ctx, userCred, idStr, nil)
	if err == sql.ErrNoRows {
		return nil, httperrors.NewResourceNotFoundError2(manager.Keyword(), idStr)
	} else if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	lockman.LockObject(ctx, model)
	defer lockman.ReleaseObject(ctx, model)

	if err := model.PreCheckPerformAction(ctx, userCred, action, query, data); err != nil {
		return nil, err
	}
	// 通过action与实例执行请求
	return objectPerformAction(manager, model, reflect.ValueOf(model), ctx, userCred, action, query, data)
}

func objectPerformAction(manager IModelManager, model IModel, modelValue reflect.Value, ctx context.Context, userCred mcclient.TokenCredential, action string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return reflectDispatcher(manager, model, modelValue, ctx, userCred, policy.PolicyActionPerform, "PerformAction", "Perform", action, query, data)
}

func reflectDispatcher(
	// dispatcher *DBModelDispatcher,
	manager IModelManager,
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
		manager, model, modelValue, ctx, userCred, operator, generalFuncName, funcPrefix, spec, query, data)
	if model != nil && err == nil && result == nil {
		return getItemDetails(manager, model, ctx, userCred, query)
	} else {
		return result, err
	}
}

func reflectDispatcherInternal(
	// dispatcher *DBModelDispatcher,
	manager IModelManager,
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
	// 优先通过action查找该model下的PerformXXX方法
	funcName := fmt.Sprintf("%s%s", funcPrefix, utils.Kebab2Camel(spec, "-"))
	funcValue := modelValue.MethodByName(funcName)

	// 若不存在该方法则根据generalFuncName查找model下的PerformAction方法
	if !funcValue.IsValid() || funcValue.IsNil() {
		funcValue = modelValue.MethodByName(generalFuncName)
		if !funcValue.IsValid() || funcValue.IsNil() {
			return nil, httperrors.NewActionNotFoundError("%s %s %s not found, please check service version, current version: %s",
				manager.Keyword(), operator, spec, version.GetShortString())
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

	// 若perform指定一类资源，则当前用户对一类资源的权限，否则校验用户对该资源的权限
	var result rbacutils.SPolicyResult
	if model == nil {
		ownerId, err := fetchOwnerId(ctx, manager, userCred, data)
		if err != nil {
			return nil, httperrors.NewGeneralError(err)
		}
		_, err = isClassRbacAllowed(ctx, manager, userCred, ownerId, operator, spec)
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		result, err = isObjectRbacAllowedResult(ctx, model, userCred, operator, spec)
		if err != nil {
			return nil, err
		}
	}

	// 调用反射的方法
	outs, err := callFunc(funcValue, funcName, params...)
	if err != nil {
		return nil, err
	}
	// perform方法返回值为jsonutils.JSONObject,error
	// 对于perform方法返回值数量不为2时，默认不合法
	if len(outs) != 2 {
		return nil, httperrors.NewInternalServerError("Invald %s return value", funcName)
	}
	resVal := outs[0]
	errVal := outs[1].Interface()
	if !gotypes.IsNil(errVal) {
		return nil, errVal.(error)
	} else {
		if model != nil {
			if _, ok := model.(IStandaloneModel); ok {
				Metadata.rawSetValues(ctx, model.Keyword(), model.GetId(), tagutils.TagsetMap2MapString(result.ObjectTags.Flattern()), false, "")
				if model.Keyword() == "project" {
					Metadata.rawSetValues(ctx, model.Keyword(), model.GetId(), tagutils.TagsetMap2MapString(result.ProjectTags.Flattern()), false, "")
				} else if model.Keyword() == "domain" {
					Metadata.rawSetValues(ctx, model.Keyword(), model.GetId(), tagutils.TagsetMap2MapString(result.DomainTags.Flattern()), false, "")
				}
			}
		}
		if gotypes.IsNil(resVal.Interface()) {
			return nil, nil
		} else {
			return ValueToJSONObject(resVal), nil
		}
	}
}

func updateItem(manager IModelManager, item IModel, ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	var err error

	// 校验update入参钩子
	err = item.ValidateUpdateCondition(ctx)
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrapf(err, "ValidateUpdateCondition"))
	}

	dataDict, ok := data.(*jsonutils.JSONDict)
	if !ok {
		return nil, httperrors.NewInternalServerError("Invalid data JSONObject")
	}

	dataDict, err = ValidateUpdateData(item, ctx, userCred, query, dataDict)
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrapf(err, "ValidateUpdateData"))
	}

	item.PreUpdate(ctx, userCred, query, dataDict)

	diff, err := Update(item, func() error {
		filterData := dataDict.CopyIncludes(updateFields(manager, userCred)...)
		err = filterData.Unmarshal(item)
		if err != nil {
			return httperrors.NewGeneralError(errors.Wrapf(err, "filterData.Unmarshal"))
		}
		return nil
	})
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrapf(err, "Update"))
	}
	for _, skip := range skipLogFields(manager) {
		delete(diff, skip)
	}
	if len(diff) > 0 {
		OpsLog.LogEvent(item, ACT_UPDATE, diff, userCred)
		logclient.AddActionLogWithContext(ctx, item, logclient.ACT_UPDATE, diff, userCred, true)
		CallUpdateNotifyHook(ctx, userCred, item)
	}

	item.PostUpdate(ctx, userCred, query, data)

	return getItemDetails(manager, item, ctx, userCred, query)
}

func (dispatcher *DBModelDispatcher) FetchUpdateHeaderData(ctx context.Context, header http.Header) (jsonutils.JSONObject, error) {
	return dispatcher.manager.FetchUpdateHeaderData(ctx, header)
}

func (dispatcher *DBModelDispatcher) Update(ctx context.Context, idStr string, query jsonutils.JSONObject, data jsonutils.JSONObject, ctxIds []dispatcher.SResourceContext) (jsonutils.JSONObject, error) {
	userCred := fetchUserCredential(ctx)
	if data == nil {
		data = jsonutils.NewDict()
	}
	manager := dispatcher.manager.GetMutableInstance(ctx, userCred, query, data)
	model, err := fetchItem(manager, ctx, userCred, idStr, nil)
	if err == sql.ErrNoRows {
		return nil, httperrors.NewResourceNotFoundError2(manager.Keyword(), idStr)
	} else if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	result, err := isObjectRbacAllowedResult(ctx, model, userCred, policy.PolicyActionUpdate)
	if err != nil {
		return nil, err
	}
	data.(*jsonutils.JSONDict).Update(result.Json())

	if len(ctxIds) > 0 {
		ctxObjs, err := fetchContextObjects(manager, ctx, userCred, ctxIds)
		if err != nil {
			return nil, httperrors.NewGeneralError(err)
		}

		return model.UpdateInContext(ctx, userCred, ctxObjs, query, data)
	} else {
		lockman.LockObject(ctx, model)
		defer lockman.ReleaseObject(ctx, model)

		return updateItem(manager, model, ctx, userCred, query, data)
	}
}

func (dispatcher *DBModelDispatcher) UpdateSpec(ctx context.Context, idStr string, spec string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	userCred := fetchUserCredential(ctx)
	manager := dispatcher.manager.GetMutableInstance(ctx, userCred, query, data)

	model, err := fetchItem(manager, ctx, userCred, idStr, nil)
	if err == sql.ErrNoRows {
		return nil, httperrors.NewResourceNotFoundError2(manager.Keyword(), idStr)
	} else if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	lockman.LockObject(ctx, model)
	defer lockman.ReleaseObject(ctx, model)

	return objectUpdateSpec(manager, model, reflect.ValueOf(model), ctx, userCred, spec, query, data)
}

func objectUpdateSpec(manager IModelManager, model IModel, modelValue reflect.Value, ctx context.Context, userCred mcclient.TokenCredential, spec string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return reflectDispatcher(manager, model, modelValue, ctx, userCred, policy.PolicyActionUpdate, "UpdateSpec", "Update", spec, query, data)
}

func DeleteModel(ctx context.Context, userCred mcclient.TokenCredential, item IModel) error {
	// cleanModelUsages(ctx, userCred, item)
	_, err := Update(item, func() error {
		return item.MarkDelete()
	})
	if err != nil {
		return httperrors.NewGeneralError(errors.Wrapf(err, "db.Update"))
	}
	if userCred != nil {
		OpsLog.LogEvent(item, ACT_DELETE, item.GetShortDesc(ctx), userCred)
		logclient.AddSimpleActionLog(item, logclient.ACT_DELETE, item.GetShortDesc(ctx), userCred, true)
	}
	if _, ok := item.(IStandaloneModel); ok && len(item.GetId()) > 0 {
		err := Metadata.RemoveAll(ctx, item, userCred)
		if err != nil {
			return errors.Wrapf(err, "Metadata.RemoveAll")
		}
	}
	return nil
}

func RealDeleteModel(ctx context.Context, userCred mcclient.TokenCredential, item IModel) error {
	if len(item.GetId()) == 0 {
		return DeleteModel(ctx, userCred, item)
	}
	_, err := sqlchemy.GetDB().Exec(
		fmt.Sprintf(
			"delete from %s where id = ?",
			item.GetModelManager().TableSpec().Name(),
		), item.GetId(),
	)
	if err != nil {
		return httperrors.NewGeneralError(errors.Wrapf(err, "db.Update"))
	}
	if userCred != nil {
		OpsLog.LogEvent(item, ACT_DELETE, item.GetShortDesc(ctx), userCred)
		logclient.AddSimpleActionLog(item, logclient.ACT_DELETE, item.GetShortDesc(ctx), userCred, true)
	}
	if _, ok := item.(IStandaloneModel); ok && len(item.GetId()) > 0 {
		err := Metadata.RemoveAll(ctx, item, userCred)
		if err != nil {
			return errors.Wrapf(err, "Metadata.RemoveAll")
		}
	}
	return nil
}

func deleteItem(manager IModelManager, model IModel, ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	// 获取实例详情
	details, err := getItemDetails(manager, model, ctx, userCred, query)
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrapf(err, "getItemDetails"))
	}

	// 删除校验
	err = ValidateDeleteCondition(model, ctx, details)
	if err != nil {
		return nil, err
	}

	// 删除前钩子
	err = CustomizeDelete(model, ctx, userCred, query, data)
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrapf(err, "CustomizeDelete"))
	}

	model.PreDelete(ctx, userCred)

	// 实际删除
	err = model.Delete(ctx, userCred)
	if err != nil {
		return nil, errors.Wrapf(err, "Delete")
	}

	// 删除后钩子
	model.PostDelete(ctx, userCred)

	// 避免设置删除状态没有正常返回
	jsonutils.Update(details, model)
	return details, nil
}

func (dispatcher *DBModelDispatcher) Delete(ctx context.Context, idstr string, query jsonutils.JSONObject, data jsonutils.JSONObject, ctxIds []dispatcher.SResourceContext) (jsonutils.JSONObject, error) {
	userCred := fetchUserCredential(ctx)
	manager := dispatcher.manager.GetMutableInstance(ctx, userCred, query, data)

	// 找到实例
	model, err := fetchItem(manager, ctx, userCred, idstr, nil)
	if err == sql.ErrNoRows {
		return nil, httperrors.NewResourceNotFoundError2(manager.Keyword(), idstr)
	} else if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	// 校验角色
	err = isObjectRbacAllowed(ctx, model, userCred, policy.PolicyActionDelete)
	if err != nil {
		return nil, err
	}

	if len(ctxIds) > 0 {
		ctxObjs, err := fetchContextObjects(manager, ctx, userCred, ctxIds)
		if err != nil {
			return nil, httperrors.NewGeneralError(err)
		}

		return model.DeleteInContext(ctx, userCred, ctxObjs, query, data)
	} else {
		lockman.LockObject(ctx, model)
		defer lockman.ReleaseObject(ctx, model)

		return deleteItem(manager, model, ctx, userCred, query, data)
	}
}

func (dispatcher *DBModelDispatcher) DeleteSpec(ctx context.Context, idstr string, spec string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	userCred := fetchUserCredential(ctx)
	manager := dispatcher.manager.GetMutableInstance(ctx, userCred, query, data)

	model, err := fetchItem(manager, ctx, userCred, idstr, nil)
	if err == sql.ErrNoRows {
		return nil, httperrors.NewResourceNotFoundError2(manager.Keyword(), idstr)
	} else if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	lockman.LockObject(ctx, model)
	defer lockman.ReleaseObject(ctx, model)

	return objectDeleteSpec(manager, model, reflect.ValueOf(model), ctx, userCred, spec, query, data)
}

func objectDeleteSpec(manager IModelManager, model IModel, modelValue reflect.Value, ctx context.Context, userCred mcclient.TokenCredential, spec string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return reflectDispatcher(manager, model, modelValue, ctx, userCred, policy.PolicyActionDelete, "DeleteSpec", "Delete", spec, query, data)
}
