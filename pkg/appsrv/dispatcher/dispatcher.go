package dispatcher

import (
	"context"
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func AddModelDispatcher(prefix string, app *appsrv.Application, manager IModelDispatchHandler) {
	metadata := map[string]interface{}{"manager": manager}
	tags := map[string]string{"resource": manager.KeywordPlural()}
	// list
	h := app.AddHandler2("GET",
		fmt.Sprintf("%s/%s", prefix, manager.KeywordPlural()),
		manager.Filter(listHandler), metadata, "list", tags)
	manager.CustomizeHandlerInfo(h)

	ctxs := manager.ContextKeywordPlural()
	// list in context
	if ctxs != nil && len(ctxs) > 0 {
		for _, ctx := range ctxs {
			h = app.AddHandler2("GET",
				fmt.Sprintf("%s/%s/<resid>/%s", prefix, ctx, manager.KeywordPlural()),
				manager.Filter(listInContextHandler), metadata, fmt.Sprintf("list_in_%s", ctx), tags)
			manager.CustomizeHandlerInfo(h)
		}
	}
	// Head
	h = app.AddHandler2("HEAD",
		fmt.Sprintf("%s/%s/<resid>", prefix, manager.KeywordPlural()),
		manager.Filter(headHandler), metadata, "head_details", tags)
	manager.CustomizeHandlerInfo(h)
	// Get
	h = app.AddHandler2("GET",
		fmt.Sprintf("%s/%s/<resid>", prefix, manager.KeywordPlural()),
		manager.Filter(getHandler), metadata, "get_details", tags)
	manager.CustomizeHandlerInfo(h)
	// get spec
	h = app.AddHandler2("GET",
		fmt.Sprintf("%s/%s/<resid>/<spec>", prefix, manager.KeywordPlural()),
		manager.Filter(getSpecHandler), metadata, "get_specific", tags)
	manager.CustomizeHandlerInfo(h)
	// create
	// create multi
	h = app.AddHandler2("POST",
		fmt.Sprintf("%s/%s", prefix, manager.KeywordPlural()),
		manager.Filter(createHandler), metadata, "create", tags)
	manager.CustomizeHandlerInfo(h)
	// create in context
	if ctxs != nil && len(ctxs) > 0 {
		for _, ctx := range ctxs {
			h = app.AddHandler2("POST",
				fmt.Sprintf("%s/%s/<resid>/%s", prefix, ctx, manager.KeywordPlural()),
				manager.Filter(createInContextHandler), metadata, fmt.Sprintf("create_in_%s", ctx), tags)
			manager.CustomizeHandlerInfo(h)
		}
	}
	// batchPerformAction
	h = app.AddHandler2("POST",
		fmt.Sprintf("%s/%s/<action>", prefix, manager.KeywordPlural()),
		manager.Filter(performClassActionHandler), metadata, "perform_class_action", tags)
	manager.CustomizeHandlerInfo(h)
	// performAction
	// create in context
	h = app.AddHandler2("POST",
		fmt.Sprintf("%s/%s/<resid>/<action>", prefix, manager.KeywordPlural()),
		manager.Filter(performActionHandler), metadata, "perform_action", tags)
	manager.CustomizeHandlerInfo(h)
	// batchUpdate
	/* app.AddHandler2("PUT",
	 	fmt.Sprintf("%s/%s", prefix, manager.KeywordPlural()),
		manager.Filter(updateClassHandler), metadata, "update_class", tags)
	*/
	// update
	h = app.AddHandler2("PUT",
		fmt.Sprintf("%s/%s/<resid>", prefix, manager.KeywordPlural()),
		manager.Filter(updateHandler), metadata, "update", tags)
	manager.CustomizeHandlerInfo(h)
	// batch Delete
	/* app.AddHandler2("DELTE",
	fmt.Sprintf("%s/%s", prefix, manager.KeywordPlural()),
	manager.Filter(batachDeleteHandler), metadata, "batch_delete", tags)
	*/
	// Delete
	h = app.AddHandler2("DELETE",
		fmt.Sprintf("%s/%s/<resid>", prefix, manager.KeywordPlural()),
		manager.Filter(deleteHandler), metadata, "delete", tags)
	manager.CustomizeHandlerInfo(h)
}

func fetchEnv(ctx context.Context, w http.ResponseWriter, r *http.Request) (IModelDispatchHandler, map[string]string, jsonutils.JSONObject, jsonutils.JSONObject) {
	params, query, body := appsrv.FetchEnv(ctx, w, r)
	metadata := appctx.AppContextMetadata(ctx)
	manager, ok := metadata["manager"].(IModelDispatchHandler)
	if !ok {
		log.Fatalf("No manager found for URL: %s", r.URL)
	}
	return manager, params, query, body
}

func mergeQueryParams(params map[string]string, query jsonutils.JSONObject, excludes ...string) jsonutils.JSONObject {
	if query == nil {
		query = jsonutils.NewDict()
	}
	queryDict := query.(*jsonutils.JSONDict)
	for k, v := range params {
		if !utils.IsInStringArray(k, excludes) {
			queryDict.Add(jsonutils.NewString(v), k[1:len(k)-1])
		}
	}
	return queryDict
}

func listHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, _ := fetchEnv(ctx, w, r)
	handleList(ctx, w, manager, "", mergeQueryParams(params, query))
}

func handleList(ctx context.Context, w http.ResponseWriter, manager IModelDispatchHandler, ctxId string, query jsonutils.JSONObject) {
	listResult, err := manager.List(ctx, query, ctxId)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	appsrv.SendJSON(w, modules.ListResult2JSONWithKey(listResult, manager.KeywordPlural()))
}

func listInContextHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, _ := fetchEnv(ctx, w, r)
	ctxId := params["<resid>"]
	handleList(ctx, w, manager, ctxId, mergeQueryParams(params, query, "<resid>"))
}

func wrapBody(body jsonutils.JSONObject, key string) jsonutils.JSONObject {
	if body != nil {
		ret := jsonutils.NewDict()
		ret.Add(body, key)
		return ret
	} else {
		return nil
	}
}

func headHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	defer func() {
		w.Header().Set("Content-Length", "0")
		w.Write([]byte{})
	}()

	manager, params, query, _ := fetchEnv(ctx, w, r)
	_, err := manager.Get(ctx, params["<resid>"], mergeQueryParams(params, query, "<resid>"), true)
	if err != nil {
		jsonErr := httperrors.NewGeneralError(err)
		httperrors.SendHTTPErrorHeader(w, jsonErr.Code)
		return
	}
}

func getHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, _ := fetchEnv(ctx, w, r)
	result, err := manager.Get(ctx, params["<resid>"], mergeQueryParams(params, query, "<resid>"), false)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	if result != nil {
		appParams := appsrv.AppContextGetParams(ctx)
		var body jsonutils.JSONObject
		if appParams != nil && appParams.OverrideResponseBodyWrapper {
			body = result
		} else {
			body = wrapBody(result, manager.Keyword())
		}
		appsrv.SendJSON(w, body)
	}
}

func getSpecHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, _ := fetchEnv(ctx, w, r)
	result, err := manager.GetSpecific(ctx, params["<resid>"], params["<spec>"], mergeQueryParams(params, query, "<resid>", "<spec>"))
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	if result != nil {
		appsrv.SendJSON(w, wrapBody(result, manager.Keyword()))
	}
}

func createHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, body := fetchEnv(ctx, w, r)
	handleCreate(ctx, w, manager, "", mergeQueryParams(params, query), body, r)
}

func handleCreate(ctx context.Context, w http.ResponseWriter, manager IModelDispatchHandler, ctxId string, query jsonutils.JSONObject, body jsonutils.JSONObject, r *http.Request) {
	count := int64(1)
	var data jsonutils.JSONObject
	var err error
	if body != nil {
		count, _ = body.Int("count")
		data, err = body.Get(manager.Keyword())
		if err != nil {
			httperrors.InvalidInputError(w,
				fmt.Sprintf("No request key: %s", manager.Keyword()))
			return
		}
	} else {
		data, err = manager.FetchCreateHeaderData(ctx, r.Header)
		if err != nil {
			httperrors.InvalidInputError(w,
				fmt.Sprintf("In valid request header: %s", err))
			return
		}
	}
	if count <= 1 {
		result, err := manager.Create(ctx, query, data, ctxId)
		if err != nil {
			httperrors.GeneralServerError(w, err)
			return
		}
		appsrv.SendJSON(w, wrapBody(result, manager.Keyword()))
	} else {
		results, err := manager.BatchCreate(ctx, query, data, int(count), ctxId)
		if err != nil {
			httperrors.GeneralServerError(w, err)
			return
		}
		ret := jsonutils.NewArray()
		for i := 0; i < len(results); i++ {
			res := jsonutils.NewDict()
			res.Add(jsonutils.NewInt(int64(results[i].Status)), "status")
			res.Add(results[i].Data, "body")
			ret.Add(res)
		}
		appsrv.SendJSON(w, wrapBody(ret, manager.KeywordPlural()))
	}
}

func createInContextHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, body := fetchEnv(ctx, w, r)
	ctxId := params["<resid>"]
	handleCreate(ctx, w, manager, ctxId, mergeQueryParams(params, query, "<resid>"), body, r)
}

func performClassActionHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, body := fetchEnv(ctx, w, r)
	var data jsonutils.JSONObject
	if body != nil {
		data, _ = body.Get(manager.KeywordPlural())
		if data == nil {
			data = body.(*jsonutils.JSONDict)
		}
	} else {
		data = jsonutils.NewDict()
	}
	results, err := manager.PerformClassAction(ctx, params["<action>"], mergeQueryParams(params, query, "<action>"), data)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	if results == nil {
		results = jsonutils.NewDict()
	}
	appsrv.SendJSON(w, wrapBody(results, manager.KeywordPlural()))
}

func performActionHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, body := fetchEnv(ctx, w, r)
	var data jsonutils.JSONObject
	if body != nil {
		data, _ = body.Get(manager.Keyword())
		if data == nil {
			data = body.(*jsonutils.JSONDict)
		}
	} else {
		data = jsonutils.NewDict()
	}
	result, err := manager.PerformAction(ctx, params["<resid>"], params["<action>"], mergeQueryParams(params, query, "<resid>", "<action>"), data)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	appsrv.SendJSON(w, wrapBody(result, manager.Keyword()))
}

/*
func updateClassHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, _, query, body := fetchEnv(ctx, w, r)
    data, err := body.Get(manager.KeywordPlural())
    if err != nil {
        httperrors.InvalidInputError(w,
                fmt.Sprintf("No request key: %s", manager.KeywordPlural()))
        return
    }
    result, err := manager.UpdateClass(ctx, tr, data)
    if err != nil {
        httperrors.GeneralServerError(w, err)
        return
    }
    appsrv.SendJSON(w, wrapBody(result, manager.KeywordPlural()))
}
*/

func updateHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, body := fetchEnv(ctx, w, r)
	var data jsonutils.JSONObject
	var err error
	if body != nil {
		data, err = body.Get(manager.Keyword())
		if err != nil {
			httperrors.InvalidInputError(w,
				fmt.Sprintf("No request key: %s", manager.Keyword()))
			return
		}
	} else {
		data, err = manager.FetchUpdateHeaderData(ctx, r.Header)
		if err != nil {
			httperrors.InvalidInputError(w,
				fmt.Sprintf("In valid request header: %s", err))
			return
		}
	}
	result, err := manager.Update(ctx, params["<resid>"], mergeQueryParams(params, query, "<resid>"), data)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	appsrv.SendJSON(w, wrapBody(result, manager.Keyword()))
}

/*
func deleteClassHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	tr, manager, _, query, body := fetchEnv(ctx, w, r)
	var data jsonutils.JSONObject
	if body != nil {
		data, err := body.Get(manager.KeywordPlural())
		if err != nil {
			httperrors.InvalidInputError(w,
                fmt.Sprintf("No request key: %s", manager.KeywordPlural()))
			return
		}
	}
    result, err := manager.DeleteClass(ctx, tr, query, data)
    if err != nil {
        httperrors.GeneralServerError(w, err)
        return
    }
    appsrv.SendJSON(w, wrapBody(result, manager.KeywordPlural()))
}
*/

func deleteHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, body := fetchEnv(ctx, w, r)
	var data jsonutils.JSONObject
	var err error
	if body != nil {
		data, err = body.Get(manager.Keyword())
		if err != nil {
			httperrors.InvalidInputError(w,
				fmt.Sprintf("No request key: %s", manager.Keyword()))
			return
		}
	} else {
		data = jsonutils.NewDict()
	}
	result, err := manager.Delete(ctx, params["<resid>"], mergeQueryParams(params, query, "<resid>"), data)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	appsrv.SendJSON(w, wrapBody(result, manager.Keyword()))
}
