package dispatcher

import (
	"context"
	"fmt"
	"net/http"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/log"
	"github.com/yunionio/mcclient/modules"
	"github.com/yunionio/pkg/appctx"
	"github.com/yunionio/pkg/appsrv"
	"github.com/yunionio/pkg/httperrors"
)

func AddModelDispatcher(prefix string, app *appsrv.Application, manager IModelDispatchHandler) {
	metadata := map[string]interface{}{"manager": manager}
	tags := map[string]string{"resource": manager.KeywordPlural()}
	// list
	app.AddHandler2("GET",
		fmt.Sprintf("%s/%s", prefix, manager.KeywordPlural()),
		manager.Filter(listHandler), metadata, "list", tags)
	ctxs := manager.ContextKeywordPlural()
	// list in context
	if ctxs != nil && len(ctxs) > 0 {
		for _, ctx := range ctxs {
			app.AddHandler2("GET",
				fmt.Sprintf("%s/%s/<resid>/%s", prefix, ctx, manager.KeywordPlural()),
				manager.Filter(listInContextHandler), metadata, "list", tags)
		}
	}
	// Get
	app.AddHandler2("GET",
		fmt.Sprintf("%s/%s/<resid>", prefix, manager.KeywordPlural()),
		manager.Filter(getHandler), metadata, "get_details", tags)
	// get spec
	app.AddHandler2("GET",
		fmt.Sprintf("%s/%s/<resid>/<spec>", prefix, manager.KeywordPlural()),
		manager.Filter(getSpecHandler), metadata, "get_specific", tags)
	// create
	// create multi
	app.AddHandler2("POST",
		fmt.Sprintf("%s/%s", prefix, manager.KeywordPlural()),
		manager.Filter(createHandler), metadata, "create", tags)
	// create in context
	if ctxs != nil && len(ctxs) > 0 {
		for _, ctx := range ctxs {
			app.AddHandler2("POST",
				fmt.Sprintf("%s/%s/<resid>/%s", prefix, ctx, manager.KeywordPlural()),
				manager.Filter(createInContextHandler), metadata, "create", tags)
		}
	}
	// batchPerformAction
	app.AddHandler2("POST",
		fmt.Sprintf("%s/%s/<action>", prefix, manager.KeywordPlural()),
		manager.Filter(performClassActionHandler), metadata, "perform_class_action", tags)
	// performAction
	// create in context
	app.AddHandler2("POST",
		fmt.Sprintf("%s/%s/<resid>/<action>", prefix, manager.KeywordPlural()),
		manager.Filter(performActionHandler), metadata, "perform_action", tags)
	// batchUpdate
	/* app.AddHandler2("PUT",
	 	fmt.Sprintf("%s/%s", prefix, manager.KeywordPlural()),
		manager.Filter(updateClassHandler), metadata, "update_class", tags)
	*/
	// update
	app.AddHandler2("PUT",
		fmt.Sprintf("%s/%s/<resid>", prefix, manager.KeywordPlural()),
		manager.Filter(updateHandler), metadata, "update", tags)
	// batch Delete
	/* app.AddHandler2("DELTE",
	fmt.Sprintf("%s/%s", prefix, manager.KeywordPlural()),
	manager.Filter(batachDeleteHandler), metadata, "batch_delete", tags)
	*/
	// Delete
	app.AddHandler2("DELETE",
		fmt.Sprintf("%s/%s/<resid>", prefix, manager.KeywordPlural()),
		manager.Filter(deleteHandler), metadata, "delete", tags)
}

func fetchEnv(ctx context.Context, w http.ResponseWriter, r *http.Request) (IModelDispatchHandler, map[string]string, jsonutils.JSONObject, jsonutils.JSONObject) {
	params, query, body := _fetchEnv(ctx, w, r)
	metadata := appctx.AppContextMetadata(ctx)
	manager, ok := metadata["manager"].(IModelDispatchHandler)
	if !ok {
		log.Fatalf("No manager found for URL: %s", r.URL)
	}
	return manager, params, query, body
}

func _fetchEnv(ctx context.Context, w http.ResponseWriter, r *http.Request) (map[string]string, jsonutils.JSONObject, jsonutils.JSONObject) {
	// trace := appsrv.AppContextTrace(ctx)
	params := appctx.AppContextParams(ctx)
	query, e := jsonutils.ParseQueryString(r.URL.RawQuery)
	if e != nil {
		log.Errorf("Parse query string %s failed: %s", r.URL.RawQuery, e)
	}
	var body jsonutils.JSONObject = nil
	if r.Method == "PUT" || r.Method == "POST" || r.Method == "DELETE" || r.Method == "PATCH" {
		body, e = appsrv.FetchJSON(r)
		if e != nil {
			log.Errorf("Fail to decode JSON request body: %s", e)
		}
	}
	return params, query, body
}

func listHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, _, query, _ := fetchEnv(ctx, w, r)
	handleList(ctx, w, manager, "", query)
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
	handleList(ctx, w, manager, ctxId, query)
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

func getHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, _ := fetchEnv(ctx, w, r)
	result, err := manager.Get(ctx, params["<resid>"], query)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	appsrv.SendJSON(w, wrapBody(result, manager.Keyword()))
}

func getSpecHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, _ := fetchEnv(ctx, w, r)
	result, err := manager.GetSpecific(ctx, params["<resid>"], params["<spec>"], query)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	appsrv.SendJSON(w, wrapBody(result, manager.Keyword()))
}

func createHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, _, query, body := fetchEnv(ctx, w, r)
	handleCreate(ctx, w, manager, "", query, body)
}

func handleCreate(ctx context.Context, w http.ResponseWriter, manager IModelDispatchHandler, ctxId string, query jsonutils.JSONObject, body jsonutils.JSONObject) {
	count, _ := body.Int("count")
	data, err := body.Get(manager.Keyword())
	if err != nil {
		httperrors.InvalidInputError(w,
			fmt.Sprintf("No request key: %s", manager.Keyword()))
		return
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
	handleCreate(ctx, w, manager, ctxId, query, body)
}

func performClassActionHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, body := fetchEnv(ctx, w, r)
	data, _ := body.Get(manager.KeywordPlural())
	if data == nil {
		data = jsonutils.NewDict()
	}
	results, err := manager.PerformClassAction(ctx, params["<action>"], query, data)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	appsrv.SendJSON(w, wrapBody(results, manager.KeywordPlural()))
}

func performActionHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, body := fetchEnv(ctx, w, r)
	data, _ := body.Get(manager.Keyword())
	if data == nil {
		data = jsonutils.NewDict()
	}
	result, err := manager.PerformAction(ctx, params["<resid>"], params["<action>"], query, data)
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
	data, err := body.Get(manager.Keyword())
	if err != nil {
		httperrors.InvalidInputError(w,
			fmt.Sprintf("No request key: %s", manager.Keyword()))
		return
	}
	result, err := manager.Update(ctx, params["<resid>"], query, data)
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
	}
	result, err := manager.Delete(ctx, params["<resid>"], query, data)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	appsrv.SendJSON(w, wrapBody(result, manager.Keyword()))
}
