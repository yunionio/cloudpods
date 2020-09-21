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

package dispatcher

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/i18n"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

func AddModelDispatcher(prefix string, app *appsrv.Application, manager IModelDispatchHandler) {
	metadata := map[string]interface{}{"manager": manager}
	tags := map[string]string{"resource": manager.KeywordPlural()}
	// list
	h := app.AddHandler2("GET",
		fmt.Sprintf("%s/%s", prefix, manager.KeywordPlural()),
		manager.Filter(listHandler), metadata, "list", tags)
	manager.CustomizeHandlerInfo(h)

	ctxss := manager.ContextKeywordPlurals()
	// list in context
	for _, ctxs := range ctxss {
		segs := make([]string, 0)
		segs = append(segs, prefix)
		for i, ctx := range ctxs {
			segs = append(segs, ctx, fmt.Sprintf("<resid_%d>", i))
		}
		segs = append(segs, manager.KeywordPlural())
		h = app.AddHandler2("GET", strings.Join(segs, "/"),
			manager.Filter(listInContextHandler), metadata, fmt.Sprintf("list_in_%s", strings.Join(ctxs, "_")), tags)
		manager.CustomizeHandlerInfo(h)
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
	for _, ctxs := range ctxss {
		segs := make([]string, 0)
		segs = append(segs, prefix)
		for i, ctx := range ctxs {
			segs = append(segs, ctx, fmt.Sprintf("<resid_%d>", i))
		}
		segs = append(segs, manager.KeywordPlural())
		h = app.AddHandler2("POST", strings.Join(segs, "/"),
			manager.Filter(createInContextHandler), metadata, fmt.Sprintf("create_in_%s", strings.Join(ctxs, "_")), tags)
		manager.CustomizeHandlerInfo(h)
	}

	// batchPerformAction
	h = app.AddHandler2("POST",
		fmt.Sprintf("%s/%s/<action>", prefix, manager.KeywordPlural()),
		manager.Filter(performClassActionHandler), metadata, "perform_class_action", tags)
	manager.CustomizeHandlerInfo(h)
	// performAction
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
	// patch
	h = app.AddHandler2("PATCH",
		fmt.Sprintf("%s/%s/<resid>", prefix, manager.KeywordPlural()),
		manager.Filter(updateHandler), metadata, "patch", tags)
	manager.CustomizeHandlerInfo(h)

	// update/patch in context
	for _, ctxs := range ctxss {
		segs := make([]string, 0)
		segs = append(segs, prefix)
		for i, ctx := range ctxs {
			segs = append(segs, ctx, fmt.Sprintf("<resid_%d>", i))
		}
		segs = append(segs, manager.KeywordPlural(), "<resid>")
		h = app.AddHandler2("PUT", strings.Join(segs, "/"),
			manager.Filter(updateInContextHandler), metadata, fmt.Sprintf("update_in_%s", strings.Join(ctxs, "_")), tags)
		manager.CustomizeHandlerInfo(h)
		h = app.AddHandler2("PATCH", strings.Join(segs, "/"),
			manager.Filter(updateInContextHandler), metadata, fmt.Sprintf("patch_in_%s", strings.Join(ctxs, "_")), tags)
		manager.CustomizeHandlerInfo(h)
	}

	// update spec
	h = app.AddHandler2("PUT",
		fmt.Sprintf("%s/%s/<resid>/<spec>", prefix, manager.KeywordPlural()),
		manager.Filter(updateSpecHandler), metadata, "update_spec", tags)
	manager.CustomizeHandlerInfo(h)
	// patch spec
	h = app.AddHandler2("PATCH",
		fmt.Sprintf("%s/%s/<resid>/<spec>", prefix, manager.KeywordPlural()),
		manager.Filter(updateSpecHandler), metadata, "patch_spec", tags)
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

	// delete in context
	for _, ctxs := range ctxss {
		segs := make([]string, 0)
		segs = append(segs, prefix)
		for i, ctx := range ctxs {
			segs = append(segs, ctx, fmt.Sprintf("<resid_%d>", i))
		}
		segs = append(segs, manager.KeywordPlural(), "<resid>")
		h = app.AddHandler2("DELETE", strings.Join(segs, "/"),
			manager.Filter(deleteInContextHandler), metadata, fmt.Sprintf("delete_in_%s", strings.Join(ctxs, "_")), tags)
		manager.CustomizeHandlerInfo(h)
	}

	// Delete Spec
	h = app.AddHandler2("DELETE",
		fmt.Sprintf("%s/%s/<resid>/<spec>", prefix, manager.KeywordPlural()),
		manager.Filter(deleteSpecHandler), metadata, "delete_spec", tags)
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
	handleList(ctx, w, manager, nil, mergeQueryParams(params, query))
}

func handleList(ctx context.Context, w http.ResponseWriter, manager IModelDispatchHandler, ctxIds []SResourceContext, query jsonutils.JSONObject) {
	listResult, err := manager.List(ctx, query, ctxIds)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	appsrv.SendJSON(w, modulebase.ListResult2JSONWithKey(listResult, manager.KeywordPlural()))
}

func fetchContextIds(segs []string, params map[string]string) ([]SResourceContext, []string) {
	ctxIds := make([]SResourceContext, 0)
	keys := make([]string, 0)
	idx := 0
	key := fmt.Sprintf("<resid_%d>", idx)
	for i := 0; i < len(segs); i += 1 {
		if segs[i] == key {
			ctxIds = append(ctxIds, SResourceContext{Type: segs[i-1], Id: params[key]})
			keys = append(keys, key)
			idx += 1
			key = fmt.Sprintf("<resid_%d>", idx)
		}
	}
	return ctxIds, keys
}

func listInContextHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, _ := fetchEnv(ctx, w, r)
	ctxIds, ctxKeys := fetchContextIds(appctx.AppContextCurrentRoot(ctx), params)
	handleList(ctx, w, manager, ctxIds, mergeQueryParams(params, query, ctxKeys...))
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

func sendJSON(ctx context.Context, w http.ResponseWriter, result jsonutils.JSONObject, keyword string) {
	appParams := appsrv.AppContextGetParams(ctx)
	var body jsonutils.JSONObject
	if appParams != nil && appParams.OverrideResponseBodyWrapper {
		body = result
	} else {
		body = wrapBody(result, keyword)
	}
	appsrv.SendJSON(w, body)
}

func getHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, _ := fetchEnv(ctx, w, r)
	result, err := manager.Get(ctx, params["<resid>"], mergeQueryParams(params, query, "<resid>"), false)
	if err != nil {
		e := httperrors.NewGeneralError(err)
		httperrors.JsonClientError(ctx, w, e)
		return
	}
	if result != nil {
		sendJSON(ctx, w, result, manager.Keyword())
	}
}

func getSpecHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, _ := fetchEnv(ctx, w, r)
	result, err := manager.GetSpecific(ctx, params["<resid>"], params["<spec>"], mergeQueryParams(params, query, "<resid>", "<spec>"))
	if err != nil {
		e := httperrors.NewGeneralError(err)
		httperrors.JsonClientError(ctx, w, e)
		return
	}
	if result != nil {
		sendJSON(ctx, w, result, manager.Keyword())
	}
}

func writeErrNoRequestKey(ctx context.Context, w http.ResponseWriter, r *http.Request, key string) {
	ctx = i18n.WithRequestLang(ctx, r)
	httperrors.InvalidInputError(ctx, w,
		"No request key: %s", key)
}

func writeErrInvalidRequestHeader(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
	ctx = i18n.WithRequestLang(ctx, r)
	httperrors.InvalidInputError(ctx, w,
		"Invalid request header: %v", err)
}

func createHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, body := fetchEnv(ctx, w, r)
	handleCreate(ctx, w, manager, nil, mergeQueryParams(params, query), body, r)
}

func handleCreate(ctx context.Context, w http.ResponseWriter, manager IModelDispatchHandler, ctxIds []SResourceContext, query jsonutils.JSONObject, body jsonutils.JSONObject, r *http.Request) {
	count := int64(1)
	var data jsonutils.JSONObject
	var err error
	if body != nil {
		count, _ = body.Int("count")
		data, err = body.Get(manager.Keyword())
		if err != nil {
			writeErrNoRequestKey(ctx, w, r, manager.Keyword())
			return
		}
	} else {
		data, err = manager.FetchCreateHeaderData(ctx, r.Header)
		if err != nil {
			writeErrInvalidRequestHeader(ctx, w, r, err)
			return
		}
	}
	if count <= 1 {
		result, err := manager.Create(ctx, query, data, ctxIds)
		if err != nil {
			httperrors.GeneralServerError(ctx, w, err)
			return
		}
		appsrv.SendJSON(w, wrapBody(result, manager.Keyword()))
	} else {
		results, err := manager.BatchCreate(ctx, query, data, int(count), ctxIds)
		if err != nil {
			httperrors.GeneralServerError(ctx, w, err)
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
	ctxIds, ctxKeys := fetchContextIds(appctx.AppContextCurrentRoot(ctx), params)
	handleCreate(ctx, w, manager, ctxIds, mergeQueryParams(params, query, ctxKeys...), body, r)
}

func performClassActionHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, body := fetchEnv(ctx, w, r)
	var data jsonutils.JSONObject
	if body != nil {
		if body.Contains(manager.KeywordPlural()) {
			data, _ = body.Get(manager.KeywordPlural())
			if data == nil {
				data = body.(*jsonutils.JSONDict)
			}
		} else {
			data = body
		}
	} else {
		data = jsonutils.NewDict()
	}
	results, err := manager.PerformClassAction(ctx, params["<action>"], mergeQueryParams(params, query, "<action>"), data)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	if results == nil {
		results = jsonutils.NewDict()
	}
	sendJSON(ctx, w, results, manager.KeywordPlural())
	// appsrv.SendJSON(w, wrapBody(results, manager.KeywordPlural()))
}

func performActionHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, body := fetchEnv(ctx, w, r)
	var data jsonutils.JSONObject
	if body != nil {
		if body.Contains(manager.Keyword()) {
			data, _ = body.Get(manager.Keyword())
			if data == nil {
				data = body.(*jsonutils.JSONDict)
			}
		} else {
			data = body
		}
	} else {
		data = jsonutils.NewDict()
	}
	result, err := manager.PerformAction(ctx, params["<resid>"], params["<action>"], mergeQueryParams(params, query, "<resid>", "<action>"), data)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	sendJSON(ctx, w, result, manager.Keyword())
	// appsrv.SendJSON(w, wrapBody(result, manager.Keyword()))
}

func updateHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, body := fetchEnv(ctx, w, r)
	handleUpdate(ctx, w, manager, params["<resid>"], nil, mergeQueryParams(params, query, "<resid>"), body, r)
}

func handleUpdate(ctx context.Context, w http.ResponseWriter, manager IModelDispatchHandler, resId string, ctxIds []SResourceContext, query jsonutils.JSONObject, body jsonutils.JSONObject, r *http.Request) {
	var data jsonutils.JSONObject
	var err error
	if body != nil {
		if body.Contains(manager.Keyword()) {
			data, err = body.Get(manager.Keyword())
			if err != nil {
				writeErrNoRequestKey(ctx, w, r, manager.Keyword())
				return
			}
		} else {
			data = body
		}
	} else {
		data, err = manager.FetchUpdateHeaderData(ctx, r.Header)
		if err != nil {
			writeErrInvalidRequestHeader(ctx, w, r, err)
			return
		}
	}
	result, err := manager.Update(ctx, resId, query, data, ctxIds)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	sendJSON(ctx, w, result, manager.Keyword())
	// appsrv.SendJSON(w, wrapBody(result, manager.Keyword()))
}

func updateInContextHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, body := fetchEnv(ctx, w, r)
	ctxIds, ctxKeys := fetchContextIds(appctx.AppContextCurrentRoot(ctx), params)
	ctxKeys = append(ctxKeys, "<resid>")
	handleUpdate(ctx, w, manager, params["<resid>"], ctxIds, mergeQueryParams(params, query, ctxKeys...), body, r)
}

func updateSpecHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, body := fetchEnv(ctx, w, r)
	var data jsonutils.JSONObject
	var err error
	if body != nil {
		if body.Contains(manager.Keyword()) {
			data, err = body.Get(manager.Keyword())
			if err != nil {
				writeErrNoRequestKey(ctx, w, r, manager.Keyword())
				return
			}
		} else {
			data = body
		}
	} else {
		data, err = manager.FetchUpdateHeaderData(ctx, r.Header)
		if err != nil {
			writeErrInvalidRequestHeader(ctx, w, r, err)
			return
		}
	}
	result, err := manager.UpdateSpec(ctx, params["<resid>"], params["<spec>"], mergeQueryParams(params, query, "<resid>", "<spec>"), data)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	sendJSON(ctx, w, result, manager.Keyword())
	// appsrv.SendJSON(w, wrapBody(result, manager.Keyword()))
}

func deleteHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, body := fetchEnv(ctx, w, r)
	handleDelete(ctx, w, manager, params["<resid>"], nil, mergeQueryParams(params, query, "<resid>"), body, r)
}

func handleDelete(ctx context.Context, w http.ResponseWriter, manager IModelDispatchHandler, resId string, ctxIds []SResourceContext, query jsonutils.JSONObject, body jsonutils.JSONObject, r *http.Request) {
	var data jsonutils.JSONObject
	var err error
	if body != nil {
		if body.Contains(manager.Keyword()) {
			data, err = body.Get(manager.Keyword())
			if err != nil {
				writeErrNoRequestKey(ctx, w, r, manager.Keyword())
				return
			}
		} else {
			data = body
		}
	} else {
		data = jsonutils.NewDict()
	}
	result, err := manager.Delete(ctx, resId, query, data, ctxIds)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	sendJSON(ctx, w, result, manager.Keyword())
	// appsrv.SendJSON(w, wrapBody(result, manager.Keyword()))
}

func deleteInContextHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, body := fetchEnv(ctx, w, r)
	ctxIds, ctxKeys := fetchContextIds(appctx.AppContextCurrentRoot(ctx), params)
	ctxKeys = append(ctxKeys, "<resid>")
	handleDelete(ctx, w, manager, params["<resid>"], ctxIds, mergeQueryParams(params, query, ctxKeys...), body, r)
}

func deleteSpecHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, body := fetchEnv(ctx, w, r)
	var data jsonutils.JSONObject
	var err error
	if body != nil {
		if body.Contains(manager.Keyword()) {
			data, err = body.Get(manager.Keyword())
			if err != nil {
				writeErrNoRequestKey(ctx, w, r, manager.Keyword())
				return
			}
		} else {
			data = body
		}
	} else {
		data = jsonutils.NewDict()
	}
	result, err := manager.DeleteSpec(ctx, params["<resid>"], params["<spec>"], mergeQueryParams(params, query, "<resid>", "<spec>"), data)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	sendJSON(ctx, w, result, manager.Keyword())
	// appsrv.SendJSON(w, wrapBody(result, manager.Keyword()))
}
