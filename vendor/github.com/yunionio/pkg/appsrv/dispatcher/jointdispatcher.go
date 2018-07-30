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
	params, query, body := _fetchEnv(ctx, w, r)
	metadata := appctx.AppContextMetadata(ctx)
	manager, ok := metadata["manager"].(IJointModelDispatchHandler)
	if !ok {
		log.Errorf("No manager found for URL: %s", r.URL)
	}
	return manager, params, query, body
}

func jointListHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, _, query, _ := fetchJointEnv(ctx, w, r)
	listResult, err := manager.List(ctx, query, "")
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	appsrv.SendJSON(w, modules.ListResult2JSONWithKey(listResult, manager.KeywordPlural()))
}

func jointListDescendentHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, _ := fetchJointEnv(ctx, w, r)
	var listResult *modules.ListResult
	var err error
	if _, ok := params["<master_id>"]; ok {
		listResult, err = manager.ListMasterDescendent(ctx, params["<master_id>"], query)
	} else if _, ok := params["<slave_id>"]; ok {
		listResult, err = manager.ListSlaveDescendent(ctx, params["<slave_id>"], query)
	}
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	appsrv.SendJSON(w, modules.ListResult2JSONWithKey(listResult, manager.KeywordPlural()))
}

func jointGetHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, _ := fetchJointEnv(ctx, w, r)
	result, err := manager.Get(ctx, params["<master_id>"], params["<slave_id>"], query)
	if err != nil {
		httperrors.GeneralServerError(w, err)
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
	result, err := manager.Attach(ctx, params["<master_id>"], params["<slave_id>"], query, data)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	appsrv.SendJSON(w, wrapBody(result, manager.Keyword()))
}

func updateJointHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, body := fetchJointEnv(ctx, w, r)
	data, err := body.Get(manager.Keyword())
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	result, err := manager.Update(ctx, params["<master_id>"], params["<slave_id>"], query, data)
	if err != nil {
		httperrors.GeneralServerError(w, err)
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
	result, err := manager.Detach(ctx, params["<master_id>"], params["<slave_id>"], query, data)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	appsrv.SendJSON(w, wrapBody(result, manager.Keyword()))
}
