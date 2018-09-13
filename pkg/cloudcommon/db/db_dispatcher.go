package db

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/filterclause"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/logclient"
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

func (dispatcher *DBModelDispatcher) ContextKeywordPlural() []string {
	ctxMans := dispatcher.modelManager.GetContextManager()
	if ctxMans != nil {
		keys := make([]string, len(ctxMans))
		for i := 0; i < len(ctxMans); i += 1 {
			keys[i] = ctxMans[i].KeywordPlural()
		}
		return keys
	}
	return nil
}

/*
const (
	AUTH_TOKEN = appctx.AppContextKey("X_AUTH_TOKEN")
)
*/

func (dispatcher *DBModelDispatcher) Filter(f appsrv.FilterHandler) appsrv.FilterHandler {
	return auth.Authenticate(f)
	/*return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		tokenStr := r.Header.Get("X-Auth-Token")
		if len(tokenStr) == 0 {
			httperrors.UnauthorizedError(w, "Unauthorized")
			return
		}
		token, err := auth.Verify(tokenStr)
		if err != nil {
			log.Errorf("Verify token failed: %s", err)
			httperrors.UnauthorizedError(w, "InvalidToken")
			return
		}
		ctx = context.WithValue(ctx, AUTH_TOKEN, token)
		f(ctx, w, r)
	} */
}

func fetchUserCredential(ctx context.Context) mcclient.TokenCredential {
	// token, ok := ctx.Value(AUTH_TOKEN).(mcclient.TokenCredential)
	token := auth.FetchUserCredential(ctx)
	if token == nil {
		log.Fatalf("user token credential not found?")
	}
	return token
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
	for k, v := range qdata {
		searchable, _ := utils.InStringArray(k, listF)
		if searchable {
			f := q.Field(k)
			if f != nil {
				strV, _ := v.GetString()
				if len(strV) > 0 {
					q = q.Equals(k, strV)
				}
			}
		}
	}
	return q, nil
}

/**
 * Column metadata fields to control list, search, details, update, create
 *  list: user | admin
 *  search: user | admin
 *  get: user | admin
 *  create: required | optional | admin_required | admin_optional
 *  update: user | admin
 *
 */
func listFields(manager IModelManager, userCred mcclient.TokenCredential) []string {
	ret := make([]string, 0)
	for _, col := range manager.TableSpec().Columns() {
		tags := col.Tags()
		list, _ := tags["list"]
		if !utils.IsInStringArray(list, []string{"user", "admin", ""}) {
			log.Warningf("Invalid list value %s for field %s", list, col.Name())
		}
		if list == "user" || (list == "admin" && userCred.IsSystemAdmin()) {
			ret = append(ret, col.Name())
		}
	}
	log.Debugf("listFields: %s", ret)
	return ret
}

func searchFields(manager IModelManager, userCred mcclient.TokenCredential) []string {
	ret := make([]string, 0)
	for _, col := range manager.TableSpec().Columns() {
		tags := col.Tags()
		list := tags["list"]
		search := tags["search"]
		if list == "user" || search == "user" || ((list == "admin" || search == "admin") && userCred.IsSystemAdmin()) {
			ret = append(ret, col.Name())
		}
	}
	return ret
}

func getDetailFields(manager IModelManager, userCred mcclient.TokenCredential) []string {
	ret := make([]string, 0)
	for _, col := range manager.TableSpec().Columns() {
		tags := col.Tags()
		list := tags["list"]
		get := tags["get"]
		if list == "user" || get == "user" || ((list == "admin" || get == "admin") && userCred.IsSystemAdmin()) {
			ret = append(ret, col.Name())
		}
	}
	return ret
}

func createRequireFields(manager IModelManager, userCred mcclient.TokenCredential) []string {
	ret := make([]string, 0)
	for _, col := range manager.TableSpec().Columns() {
		tags := col.Tags()
		create, _ := tags["create"]
		if create == "required" || (create == "admin_required" && userCred.IsSystemAdmin()) {
			ret = append(ret, col.Name())
		}
	}
	log.Debugf("CreateRequiredFields for %s: %s", manager.Keyword(), ret)
	return ret
}

func createFields(manager IModelManager, userCred mcclient.TokenCredential) []string {
	ret := make([]string, 0)
	for _, col := range manager.TableSpec().Columns() {
		tags := col.Tags()
		create, _ := tags["create"]
		update := tags["update"]
		if update == "user" || (update == "admin" && userCred.IsSystemAdmin()) || create == "required" || create == "optional" || ((create == "admin_required" || create == "admin_optional") && userCred.IsSystemAdmin()) {
			ret = append(ret, col.Name())
		}
	}
	log.Debugf("CreateFields for %s: %s", manager.Keyword(), ret)
	return ret
}

func updateFields(manager IModelManager, userCred mcclient.TokenCredential) []string {
	ret := make([]string, 0)
	for _, col := range manager.TableSpec().Columns() {
		tags := col.Tags()
		update := tags["update"]
		if update == "user" || (update == "admin" && userCred.IsSystemAdmin()) {
			ret = append(ret, col.Name())
		}
	}
	return ret
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
			ok, _ := utils.InStringArray(fc.GetField(), schFields)
			if ok {
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
			if ok, _ := utils.InStringArray(jfc.GetField(), schFields); ok {
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

func listItemQueryFilters(manager IModelManager, ctx context.Context, q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {

	if !jsonutils.QueryBoolean(query, "admin", false) {
		q = manager.FilterByOwner(q, manager.GetOwnerId(userCred))
	}
	q, err := manager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
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

func query2List(manager IModelManager, ctx context.Context, userCred mcclient.TokenCredential, q *sqlchemy.SQuery, query jsonutils.JSONObject) ([]jsonutils.JSONObject, error) {
	rows, err := q.Rows()
	if err != nil {
		return nil, err
	}
	listF := listFields(manager, userCred)
	fieldFilter := jsonutils.GetQueryStringArray(query, "field")
	if len(fieldFilter) > 0 && userCred.IsSystemAdmin() {
		// only sysadmin can specify list Fields
		listF = fieldFilter
	}
	showDetails := false
	showDetailsJson, _ := query.Get("details")
	if showDetailsJson != nil {
		showDetails, _ = showDetailsJson.Bool()
	} else {
		showDetails = true
	}
	item, err := NewModelObject(manager)
	if err != nil {
		return nil, err
	}
	itemInitValue := reflect.Indirect(reflect.ValueOf(item))
	item, err = NewModelObject(manager)
	if err != nil {
		return nil, err
	}
	results := make([]jsonutils.JSONObject, 0)
	for rows.Next() {
		itemValue := reflect.Indirect(reflect.ValueOf(item))
		itemValue.Set(itemInitValue)
		err = q.Row2Struct(rows, item)
		if err != nil {
			return nil, err
		}
		jsonData := jsonutils.Marshal(item)
		jsonDict, ok := jsonData.(*jsonutils.JSONDict)
		if !ok {
			return nil, fmt.Errorf("invalid model data structure, not a dict")
		}
		jsonDict = jsonDict.CopyIncludes(listF...)
		if showDetails {
			extraDict := item.GetCustomizeColumns(ctx, userCred, query)
			if extraDict != nil {
				jsonDict.Update(extraDict)
			}
			jsonDict = getModelExtraDetails(item, ctx, jsonDict)
		}
		results = append(results, jsonDict)
	}
	return results, nil
}

func fetchContextObjectId(manager IModelManager, ctx context.Context, userCred mcclient.TokenCredential, ctxId string, queryDict *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	ctxMans := manager.GetContextManager()
	if ctxMans == nil {
		return nil, fmt.Errorf("No context manager")
	}
	find := false
	keys := make([]string, 0)
	for i := 0; i < len(ctxMans); i += 1 {
		ctxObj, err := fetchItem(ctxMans[i], ctx, userCred, ctxId, nil)
		if err != nil {
			if err == sql.ErrNoRows {
				keys = append(keys, ctxMans[i].KeywordPlural())
				continue
			} else {
				return nil, err
			}
		} else {
			find = true
			queryDict.Add(jsonutils.NewString(ctxObj.GetId()), fmt.Sprintf("%s_id", ctxObj.GetModelManager().Keyword()))
			if len(ctxObj.GetModelManager().Alias()) > 0 {
				queryDict.Add(jsonutils.NewString(ctxObj.GetId()), fmt.Sprintf("%s_id", ctxObj.GetModelManager().Alias()))
			}
		}
	}
	if !find {
		return nil, httperrors.NewResourceNotFoundError("Resource %s not found in %s", ctxId, strings.Join(keys, ", "))
	} else {
		return queryDict, nil
	}
}

func listItems(manager IModelManager, ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, ctxId string) (*modules.ListResult, error) {
	var maxLimit int64 = 2048
	limit, _ := query.Int("limit")
	offset, _ := query.Int("offset")
	q := manager.Query()
	var err error
	queryDict, ok := query.(*jsonutils.JSONDict)
	if !ok {
		return nil, fmt.Errorf("invalid query format")
	}
	if len(ctxId) > 0 {
		queryDict, err = fetchContextObjectId(manager, ctx, userCred, ctxId, queryDict)
		if err != nil {
			return nil, err
		}
	}
	queryDict, err = manager.ValidateListConditions(ctx, userCred, queryDict)
	if err != nil {
		return nil, err
	}
	q, err = listItemQueryFilters(manager, ctx, q, userCred, queryDict)
	if err != nil {
		return nil, err
	}
	totalCnt := int64(q.Count())
	log.Debugf("total count %d", totalCnt)
	if totalCnt == 0 {
		emptyList := modules.ListResult{Data: []jsonutils.JSONObject{}}
		return &emptyList, nil
	}
	if totalCnt > maxLimit && (limit <= 0 || limit > maxLimit) {
		limit = maxLimit
	}
	orderBy := jsonutils.GetQueryStringArray(queryDict, "order_by")
	order := sqlchemy.SQL_ORDER_DESC
	if len(orderBy) == 0 {
		colSpec := manager.TableSpec().ColumnSpec("id")
		if err == nil && colSpec != nil && colSpec.IsNumeric() {
			orderBy = []string{"id"}
		} else {
			orderBy = []string{"created_at"}
		}
	} else {
		orderStr, _ := queryDict.GetString("order")
		if orderStr == "asc" {
			order = sqlchemy.SQL_ORDER_ASC
		}
	}
	if order == sqlchemy.SQL_ORDER_ASC {
		for _, orderByField := range orderBy {
			q = q.Asc(orderByField)
		}
	} else {
		for _, orderByField := range orderBy {
			q = q.Desc(orderByField)
		}
	}
	if limit > 0 {
		q = q.Limit(int(limit))
	}
	if offset > 0 {
		q = q.Offset(int(offset))
	}
	retList, err := query2List(manager, ctx, userCred, q, queryDict)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	retResult := modules.ListResult{Data: retList, Total: int(totalCnt), Limit: int(limit), Offset: int(offset)}
	return &retResult, nil
}

func (dispatcher *DBModelDispatcher) List(ctx context.Context, query jsonutils.JSONObject, ctxId string) (*modules.ListResult, error) {
	userCred := fetchUserCredential(ctx)
	if !dispatcher.modelManager.AllowListItems(ctx, userCred, query) {
		return nil, httperrors.NewForbiddenError("Not allow to list")
	}
	items, err := listItems(dispatcher.modelManager, ctx, userCred, query, ctxId)
	if err != nil {
		log.Errorf("Fail to list items: %s", err)
		return nil, httperrors.NewGeneralError(err)
	}
	return items, nil
}

func getModelExtraDetails(item IModel, ctx context.Context, extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	err := item.ValidateDeleteCondition(ctx)
	if err != nil {
		extra.Add(jsonutils.JSONFalse, "can_delete")
	} else {
		extra.Add(jsonutils.JSONTrue, "can_delete")
	}
	err = item.ValidateUpdateCondition(ctx)
	if err != nil {
		extra.Add(jsonutils.JSONFalse, "can_update")
	} else {
		extra.Add(jsonutils.JSONTrue, "can_update")
	}
	return extra
}

func getItemDetails(manager IModelManager, item IModel, ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	jsonData := jsonutils.Marshal(item)
	jsonDict, ok := jsonData.(*jsonutils.JSONDict)
	if !ok {
		return nil, fmt.Errorf("fail to convert model to json")
	}
	jsonDict = jsonDict.CopyIncludes(getDetailFields(manager, userCred)...)
	extraDict := item.GetExtraDetails(ctx, userCred, query)
	if extraDict != nil {
		jsonDict.Update(extraDict)
	}
	jsonDict = getModelExtraDetails(item, ctx, jsonDict)
	return jsonDict, nil
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
		return nil, httperrors.NewForbiddenError(fmt.Sprintf("%s not allow to get property %s", dispatcher.Keyword(), property))
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

func (dispatcher *DBModelDispatcher) Get(ctx context.Context, idStr string, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
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
		return nil, httperrors.NewResourceNotFoundError(fmt.Sprintf("%s %s not found",
			dispatcher.modelManager.Keyword(), idStr))
	} else if err != nil {
		return nil, err
	}
	// log.Debugf("Get found %s", model)
	if !model.AllowGetDetails(ctx, userCred, query) {
		return nil, httperrors.NewForbiddenError("Not allow to get details")
	}
	return getItemDetails(dispatcher.modelManager, model, ctx, userCred, query)
}

func (dispatcher *DBModelDispatcher) GetSpecific(ctx context.Context, idStr string, spec string, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	userCred := fetchUserCredential(ctx)
	model, err := fetchItem(dispatcher.modelManager, ctx, userCred, idStr, query)
	if err == sql.ErrNoRows {
		return nil, httperrors.NewResourceNotFoundError(fmt.Sprintf("%s %s not found",
			dispatcher.modelManager.Keyword(), idStr))
	} else if err != nil {
		return nil, err
	}

	specCamel := utils.Kebab2Camel(spec, "-")
	funcName := fmt.Sprintf("AllowGetDetails%s", specCamel)
	modelValue := reflect.ValueOf(model)
	funcValue := modelValue.MethodByName(funcName)
	if !funcValue.IsValid() || funcValue.IsNil() {
		return nil, httperrors.NewSpecNotFoundError(fmt.Sprintf("%s %s %s not found", dispatcher.Keyword(), idStr, spec))
	}

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
		return nil, httperrors.NewForbiddenError(fmt.Sprintf("%s not allow to get spec %s", dispatcher.Keyword(), spec))
	}

	funcName = fmt.Sprintf("GetDetails%s", specCamel)
	funcValue = modelValue.MethodByName(funcName)
	if !funcValue.IsValid() || funcValue.IsNil() {
		return nil, httperrors.NewSpecNotFoundError(fmt.Sprintf("%s %s %s not found", dispatcher.Keyword(), idStr, spec))
	}

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
			return nil, nil
		} else {
			return resVal.(jsonutils.JSONObject), nil
		}
	}
}

func fetchOwnerProjectId(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) (string, error) {
	var projId string
	if data != nil {
		projId, _ = data.GetString("project")
		if len(projId) == 0 {
			projId, _ = data.GetString("tenant")
		}
	}
	if len(projId) == 0 {
		return userCred.GetProjectId(), nil
	}
	if !userCred.IsSystemAdmin() {
		return "", httperrors.NewForbiddenError("Delegation not allowed")
	}
	t, _ := TenantCacheManager.FetchTenantByIdOrName(ctx, projId)
	if t == nil {
		return "", httperrors.NewNotFoundError(fmt.Sprintf("Project %s not found", projId))
	}
	return t.GetId(), nil
}

func NewModelObject(modelManager IModelManager) (IModel, error) {
	m, ok := reflect.New(modelManager.TableSpec().DataType()).Interface().(IModel)
	m.SetModelManager(modelManager)
	if !ok {
		return nil, fmt.Errorf("inconsistent type")
	}
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

func doCreateItem(manager IModelManager, ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) (IModel, error) {
	dataDict := data.(*jsonutils.JSONDict)
	var err error

	generateName, _ := dataDict.GetString("generate_name")
	if len(generateName) > 0 {
		dataDict.Remove("generate_name")
		dataDict.Add(jsonutils.NewString(GenerateName(manager, ownerProjId, generateName)), "name")
	} else {
		name, _ := data.GetString("name")
		if len(name) > 0 {
			err = newNameValidator(manager, ownerProjId, name)
			if err != nil {
				return nil, err
			}
		}
	}

	dataDict, err = manager.ValidateCreateData(ctx, userCred, ownerProjId, query, dataDict)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
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
	err = model.CustomizeCreate(ctx, userCred, ownerProjId, query, dataDict)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	err = manager.TableSpec().Insert(model)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	model.PostCreate(ctx, userCred, ownerProjId, query, dataDict)
	return model, nil
}

func (dispatcher *DBModelDispatcher) Create(ctx context.Context, query jsonutils.JSONObject, data jsonutils.JSONObject, ctxId string) (jsonutils.JSONObject, error) {
	userCred := fetchUserCredential(ctx)

	ownerProjId, err := fetchOwnerProjectId(ctx, userCred, data)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	if len(ctxId) > 0 {
		dataDict, ok := data.(*jsonutils.JSONDict)
		if !ok {
			log.Errorf("fail to convert body into jsondict")
			return nil, fmt.Errorf("fail to parse body")
		}
		data, err = fetchContextObjectId(dispatcher.modelManager, ctx, userCred, ctxId, dataDict)
		if err != nil {
			log.Errorf("fail to find context object %s", ctxId)
			return nil, err
		}
	}

	lockman.LockClass(ctx, dispatcher.modelManager, ownerProjId)
	defer lockman.ReleaseClass(ctx, dispatcher.modelManager, ownerProjId)

	if !dispatcher.modelManager.AllowCreateItem(ctx, userCred, query, data) {
		return nil, httperrors.NewForbiddenError("Not allow to create item")
	}

	model, err := doCreateItem(dispatcher.modelManager, ctx, userCred, ownerProjId, query, data)
	if err != nil {
		log.Errorf("fail to doCreateItem %s", err)
		return nil, httperrors.NewGeneralError(err)
	}
	OpsLog.LogEvent(model, ACT_CREATE, model.GetShortDesc(), userCred)
	logclient.AddActionLog(model, logclient.ACT_CREATE, "", userCred, true)
	dispatcher.modelManager.OnCreateComplete(ctx, []IModel{model}, userCred, query, data)
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

func (dispatcher *DBModelDispatcher) BatchCreate(ctx context.Context, query jsonutils.JSONObject, data jsonutils.JSONObject, count int, ctxId string) ([]modules.SubmitResult, error) {
	userCred := fetchUserCredential(ctx)

	ownerProjId, err := fetchOwnerProjectId(ctx, userCred, data)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	if len(ctxId) > 0 {
		dataDict, ok := data.(*jsonutils.JSONDict)
		if !ok {
			return nil, fmt.Errorf("fail to parse body")
		}
		data, err = fetchContextObjectId(dispatcher.modelManager, ctx, userCred, ctxId, dataDict)
		if err != nil {
			return nil, err
		}
	}

	lockman.LockClass(ctx, dispatcher.modelManager, ownerProjId)
	defer lockman.ReleaseClass(ctx, dispatcher.modelManager, ownerProjId)

	if !dispatcher.modelManager.AllowCreateItem(ctx, userCred, query, data) {
		return nil, httperrors.NewForbiddenError("Not allow to create item")
	}

	multiData, err := expandMultiCreateParams(data, count)
	if err != nil {
		return nil, err
	}
	results := make([]modules.SubmitResult, count)
	models := make([]IModel, 0)
	for i, cdata := range multiData {
		model, err := doCreateItem(dispatcher.modelManager, ctx, userCred, ownerProjId, query, cdata)
		result := modules.SubmitResult{}
		if err != nil {
			jsonErr, ok := err.(*httputils.JSONClientError)
			if ok {
				result.Status = jsonErr.Code
				result.Data = jsonutils.Marshal(jsonErr)
			} else {
				result.Status = 500
				result.Data = jsonutils.NewString(err.Error())
			}
		} else {
			models = append(models, model)
			body, err := getItemDetails(dispatcher.modelManager, model, ctx, userCred, query)
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
		dispatcher.modelManager.OnCreateComplete(ctx, models, userCred, query, data)
	}
	return results, nil
}

func (dispatcher *DBModelDispatcher) PerformClassAction(ctx context.Context, action string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	userCred := fetchUserCredential(ctx)

	ownerProjId, err := fetchOwnerProjectId(ctx, userCred, data)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	lockman.LockClass(ctx, dispatcher.modelManager, ownerProjId)
	defer lockman.ReleaseClass(ctx, dispatcher.modelManager, ownerProjId)

	managerValue := reflect.ValueOf(dispatcher.modelManager)
	if action == "check-create-data" {
		manager := dispatcher.modelManager
		if body, err := data.(*jsonutils.JSONDict).Get(manager.Keyword()); err != nil {
			return nil, httperrors.NewGeneralError(err)
		} else {
			return manager.ValidateCreateData(ctx, userCred, ownerProjId, query, body.(*jsonutils.JSONDict))
		}
	}
	return objectPerformAction(dispatcher, managerValue, ctx, userCred, action, query, data)
}

func (dispatcher *DBModelDispatcher) PerformAction(ctx context.Context, idStr string, action string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	userCred := fetchUserCredential(ctx)
	model, err := fetchItem(dispatcher.modelManager, ctx, userCred, idStr, query)
	if err == sql.ErrNoRows {
		return nil, httperrors.NewResourceNotFoundError(fmt.Sprintf("%s %s not found",
			dispatcher.modelManager.Keyword(), idStr))
	} else if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	lockman.LockObject(ctx, model)
	defer lockman.ReleaseObject(ctx, model)

	modelValue := reflect.ValueOf(model)
	result, err := objectPerformAction(dispatcher, modelValue, ctx, userCred, action, query, data)
	if err == nil && result == nil {
		return getItemDetails(dispatcher.modelManager, model, ctx, userCred, query)
	} else {
		return result, err
	}
}

func objectPerformAction(dispatcher *DBModelDispatcher, modelValue reflect.Value, ctx context.Context, userCred mcclient.TokenCredential, action string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	isGeneral := false

	const generalFuncName = "PerformAction"
	const generalAllowFuncName = "AllowPerformAction"

	funcName := fmt.Sprintf("Perform%s", utils.Kebab2Camel(action, "-"))
	allowFuncName := "Allow" + funcName
	funcValue := modelValue.MethodByName(allowFuncName)
	if !funcValue.IsValid() || funcValue.IsNil() {
		funcValue = modelValue.MethodByName(generalAllowFuncName)
		if !funcValue.IsValid() || funcValue.IsNil() {
			msg := fmt.Sprintf("%s allow perform action %s not found", dispatcher.Keyword(), action)
			log.Errorf(msg)
			return nil, httperrors.NewActionNotFoundError(msg)
		} else {
			isGeneral = true
		}
	}

	var params []reflect.Value

	if isGeneral {
		params = []reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(userCred),
			reflect.ValueOf(action),
			reflect.ValueOf(query),
			reflect.ValueOf(data),
		}
	} else {
		params = []reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(userCred),
			reflect.ValueOf(query),
			reflect.ValueOf(data),
		}
	}

	outs := funcValue.Call(params)
	if len(outs) != 1 {
		return nil, httperrors.NewInternalServerError("Invald %s return value", allowFuncName)
	}
	if !outs[0].Bool() {
		return nil, httperrors.NewForbiddenError(fmt.Sprintf("%s not allow to perform action %s", dispatcher.Keyword(), action))
	}

	if isGeneral {
		funcValue = modelValue.MethodByName(generalFuncName)
	} else {
		funcName = fmt.Sprintf("Perform%s", utils.Kebab2Camel(action, "-"))
		funcValue = modelValue.MethodByName(funcName)
	}
	if !funcValue.IsValid() || funcValue.IsNil() {
		return nil, httperrors.NewActionNotFoundError(fmt.Sprintf("%s perform action %s not found", dispatcher.Keyword(), action))
	}

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
			return nil, nil
		} else {
			return resVal.(jsonutils.JSONObject), nil
		}
	}
}

func updateItem(manager IModelManager, item IModel, ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	var err error

	err = item.ValidateUpdateCondition(ctx)

	if err != nil {
		log.Errorf("validate update condition error: %s", err)
		logclient.AddActionLog(item, logclient.ACT_UPDATE, err.Error(), userCred, false)
		return nil, httperrors.NewGeneralError(err)
	}

	dataDict, ok := data.(*jsonutils.JSONDict)
	if !ok {
		logclient.AddActionLog(item, logclient.ACT_UPDATE, "Invalid data JSONObject", userCred, false)
		return nil, httperrors.NewInternalServerError("Invalid data JSONObject")
	}

	name, _ := data.GetString("name")
	if len(name) > 0 {
		err = alterNameValidator(item, name)
		if err != nil {
			logclient.AddActionLog(item, logclient.ACT_UPDATE, err.Error(), userCred, false)
			return nil, err
		}
	}

	dataDict, err = item.ValidateUpdateData(ctx, userCred, query, dataDict)
	if err != nil {
		errMsg := fmt.Sprintf("validate update data error: %s", err)
		log.Errorf(errMsg)
		logclient.AddActionLog(item, logclient.ACT_UPDATE, errMsg, userCred, false)
		return nil, httperrors.NewGeneralError(err)
	}
	item.PreUpdate(ctx, userCred, query, dataDict)
	diff, err := manager.TableSpec().Update(item, func() error {
		filterData := dataDict.CopyIncludes(updateFields(manager, userCred)...)
		err = filterData.Unmarshal(item)
		if err != nil {
			errMsg := fmt.Sprintf("unmarshal fail: %s", err)
			logclient.AddActionLog(item, logclient.ACT_UPDATE, errMsg, userCred, false)
			log.Errorf(errMsg)
			return httperrors.NewGeneralError(err)
		}
		return nil
	})

	if err != nil {
		log.Errorf("save update error: %s", err)
		return nil, httperrors.NewGeneralError(err)
	}
	if diff != nil {
		diffStr := sqlchemy.UpdateDiffString(diff)
		if len(diffStr) > 0 {
			OpsLog.LogEvent(item, ACT_UPDATE, diffStr, userCred)
			logclient.AddActionLog(item, logclient.ACT_UPDATE, diffStr, userCred, true)
		}
	} else {
		logclient.AddActionLog(item, logclient.ACT_UPDATE, "", userCred, true)
	}
	return getItemDetails(manager, item, ctx, userCred, query)
}

func (dispatcher *DBModelDispatcher) Update(ctx context.Context, idStr string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	userCred := fetchUserCredential(ctx)
	model, err := fetchItem(dispatcher.modelManager, ctx, userCred, idStr, query)
	if err == sql.ErrNoRows {
		return nil, httperrors.NewResourceNotFoundError(fmt.Sprintf("%s %s not found",
			dispatcher.modelManager.Keyword(), idStr))
	} else if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	if !model.AllowUpdateItem(ctx, userCred) {
		return nil, httperrors.NewForbiddenError(fmt.Sprintf("Not allow to update item"))
	}

	lockman.LockObject(ctx, model)
	defer lockman.ReleaseObject(ctx, model)

	return updateItem(dispatcher.modelManager, model, ctx, userCred, query, data)
}

func DeleteModel(ctx context.Context, userCred mcclient.TokenCredential, item IModel) error {
	manager := item.GetModelManager()
	log.Debugf("Ready to delete %s %s %#v", jsonutils.Marshal(item), item, manager)
	_, err := manager.TableSpec().Update(item, func() error {
		return item.MarkDelete()
	})
	if err != nil {
		msg := fmt.Sprintf("save update error %s", err)
		log.Errorf(msg)
		logclient.AddActionLog(item, logclient.ACT_DELETE, msg, userCred, false)
		return httperrors.NewGeneralError(err)
	}
	OpsLog.LogEvent(item, ACT_DELETE, item.GetShortDesc(), userCred)
	logclient.AddActionLog(item, logclient.ACT_DELETE, item.GetShortDesc(), userCred, true)
	return nil
}

func deleteItem(manager IModelManager, model IModel, ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	log.Debugf("deleteItem %s", jsonutils.Marshal(model))
	if !model.AllowDeleteItem(ctx, userCred, query, data) {
		log.Errorf("not allow to delete")
		return nil, httperrors.NewForbiddenError(fmt.Sprintf("%s not allow to delete", manager.KeywordPlural(), model.GetId()))
	}

	err := model.ValidateDeleteCondition(ctx)
	if err != nil {
		log.Errorf("validate delete condition error: %s", err)
		return nil, httperrors.NewNotAcceptableError(err.Error())
	}

	err = model.CustomizeDelete(ctx, userCred, query, data)
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

	log.Debugf("deleteItem before Delete %s %s", jsonutils.Marshal(model), reflect.TypeOf(model))

	// err = DeleteModel(ctx, userCred, model)
	err = model.Delete(ctx, userCred)
	if err != nil {
		log.Errorf("Delete error %s", err)
		return nil, err
	}

	model.PostDelete(ctx, userCred)

	return details, nil
}

func (dispatcher *DBModelDispatcher) Delete(ctx context.Context, idstr string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	userCred := fetchUserCredential(ctx)
	model, err := fetchItem(dispatcher.modelManager, ctx, userCred, idstr, query)
	if err == sql.ErrNoRows {
		return nil, httperrors.NewResourceNotFoundError(fmt.Sprintf("%s %s not found",
			dispatcher.modelManager.Keyword(), idstr))
	} else if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	log.Debugf("Delete %s", model.GetShortDesc())

	lockman.LockObject(ctx, model)
	defer lockman.ReleaseObject(ctx, model)

	return deleteItem(dispatcher.modelManager, model, ctx, userCred, query, data)
}
