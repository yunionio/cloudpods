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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

func AddJointModelDispatcher(prefix string, app *appsrv.Application, manager IJointModelDispatchHandler) {
	metadata := map[string]interface{}{"manager": manager}
	tags := map[string]string{"resource": manager.KeywordPlural()}
	// list
	app.AddHandler2("GET",
		fmt.Sprintf("%s/%s", prefix, manager.KeywordPlural()),
		manager.Filter(jointListHandler),
		metadata, "list_joint", tags)
	// joint list descendent
	app.AddHandler2("GET",
		fmt.Sprintf("%s/%s/<master_id>/%s", prefix,
			manager.MasterKeywordPlural(),
			manager.SlaveKeywordPlural()),
		manager.Filter(jointListDescendentHandler),
		metadata, "list_descendent", tags)
	// joint list descendent
	app.AddHandler2("GET",
		fmt.Sprintf("%s/%s/<slave_id>/%s", prefix,
			manager.SlaveKeywordPlural(),
			manager.MasterKeywordPlural()),
		manager.Filter(jointListDescendentHandler),
		metadata, "list_descendent", tags)
	// joint Get
	app.AddHandler2("GET",
		fmt.Sprintf("%s/%s/<master_id>/%s/<slave_id>", prefix,
			manager.MasterKeywordPlural(),
			manager.SlaveKeywordPlural()),
		manager.Filter(jointGetHandler),
		metadata, "get_joint", tags)
	app.AddHandler2("GET",
		fmt.Sprintf("%s/%s/<slave_id>/%s/<master_id>", prefix,
			manager.SlaveKeywordPlural(),
			manager.MasterKeywordPlural()),
		manager.Filter(jointGetHandler),
		metadata, "get_joint", tags)
	// joint attach
	app.AddHandler2("POST",
		fmt.Sprintf("%s/%s/<master_id>/%s/<slave_id>", prefix,
			manager.MasterKeywordPlural(),
			manager.SlaveKeywordPlural()),
		manager.Filter(attachHandler),
		metadata, "attach", tags)
	app.AddHandler2("POST",
		fmt.Sprintf("%s/%s/<slave_id>/%s/<master_id>", prefix,
			manager.SlaveKeywordPlural(),
			manager.MasterKeywordPlural()),
		manager.Filter(attachHandler),
		metadata, "attach", tags)
	// joint update
	app.AddHandler2("PUT",
		fmt.Sprintf("%s/%s/<master_id>/%s/<slave_id>", prefix,
			manager.MasterKeywordPlural(),
			manager.SlaveKeywordPlural()),
		manager.Filter(updateJointHandler),
		metadata, "update_joint", tags)
	app.AddHandler2("PUT",
		fmt.Sprintf("%s/%s/<slave_id>/%s/<master_id>", prefix,
			manager.SlaveKeywordPlural(),
			manager.MasterKeywordPlural()),
		manager.Filter(updateJointHandler),
		metadata, "update_joint", tags)
	// detach joint
	app.AddHandler2("DELETE",
		fmt.Sprintf("%s/%s/<master_id>/%s/<slave_id>", prefix,
			manager.MasterKeywordPlural(),
			manager.SlaveKeywordPlural()),
		manager.Filter(detachHandler),
		metadata, "detach", tags)
	app.AddHandler2("DELETE",
		fmt.Sprintf("%s/%s/<slave_id>/%s/<master_id>", prefix,
			manager.SlaveKeywordPlural(),
			manager.MasterKeywordPlural()),
		manager.Filter(detachHandler),
		metadata, "detach", tags)
}

func fetchJointEnv(ctx context.Context, w http.ResponseWriter, r *http.Request) (IJointModelDispatchHandler, map[string]string, jsonutils.JSONObject, jsonutils.JSONObject) {
	params, query, body := appsrv.FetchEnv(ctx, w, r)
	metadata := appctx.AppContextMetadata(ctx)
	manager, ok := metadata["manager"].(IJointModelDispatchHandler)
	if !ok {
		log.Errorf("No manager found for URL: %s", r.URL)
	}
	return manager, params, query, body
}

func jointListHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, _ := fetchJointEnv(ctx, w, r)
	listResult, err := manager.List(ctx, mergeQueryParams(params, query), nil)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	appsrv.SendJSON(w, modulebase.ListResult2JSONWithKey(listResult, manager.KeywordPlural()))
}

func jointListDescendentHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, _ := fetchJointEnv(ctx, w, r)
	var listResult *modulebase.ListResult
	var err error
	if _, ok := params["<master_id>"]; ok {
		listResult, err = manager.ListMasterDescendent(ctx, params["<master_id>"], mergeQueryParams(params, query, "<master_id>"))
	} else if _, ok := params["<slave_id>"]; ok {
		listResult, err = manager.ListSlaveDescendent(ctx, params["<slave_id>"], mergeQueryParams(params, query, "<slave_id>"))
	}
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	appsrv.SendJSON(w, modulebase.ListResult2JSONWithKey(listResult, manager.KeywordPlural()))
}

func jointGetHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, _ := fetchJointEnv(ctx, w, r)
	result, err := manager.Get(ctx, params["<master_id>"], params["<slave_id>"], mergeQueryParams(params, query, "<master_id>", "<slave_id>"))
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	appsrv.SendJSON(w, wrapBody(result, manager.Keyword()))
}

func attachHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, body := fetchJointEnv(ctx, w, r)
	log.Debugf("body: %s", body)
	var data jsonutils.JSONObject
	if body != nil {
		data, _ = body.Get(manager.Keyword())
	}
	if data == nil {
		data = jsonutils.NewDict()
	}
	result, err := manager.Attach(ctx, params["<master_id>"], params["<slave_id>"], mergeQueryParams(params, query, "<master_id>", "<slave_id>"), data)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	appsrv.SendJSON(w, wrapBody(result, manager.Keyword()))
}

func updateJointHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, body := fetchJointEnv(ctx, w, r)
	data, err := body.Get(manager.Keyword())
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	result, err := manager.Update(ctx, params["<master_id>"], params["<slave_id>"], mergeQueryParams(params, query, "<master_id>", "<slave_id>"), data)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	appsrv.SendJSON(w, wrapBody(result, manager.Keyword()))
}

func detachHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, body := fetchJointEnv(ctx, w, r)
	var data jsonutils.JSONObject
	if body != nil {
		data, _ = body.Get(manager.Keyword())
	}
	result, err := manager.Detach(ctx, params["<master_id>"], params["<slave_id>"], mergeQueryParams(params, query, "<master_id>", "<slave_id>"), data)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	appsrv.SendJSON(w, wrapBody(result, manager.Keyword()))
}
