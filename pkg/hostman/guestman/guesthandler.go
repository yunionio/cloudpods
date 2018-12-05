package guestman

import (
	"context"
	"net/http"
	"path"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

type strDict map[string]string
type actionFunc func(context.Context, string, jsonutils.JSONObject) (interface{}, error)

func AddGuestTaskHandler(prefix string, app *appsrv.Application) {
	app.AddHandler("GET", "/servers/<sid>/status", auth.Authenticate(getStatus))
	app.AddHandler("POST", "/servers/cpu-node-balance", auth.Authenticate(cpusetBalance))
	app.AddHandler("POST", "/servers/<sid>/<action>", auth.Authenticate(guestActions))
	app.AddHandler("DELETE", "/servers/<sid>", auth.Authenticate(deleteGuest))
}

func getStatus(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, _, _ := appsrv.FetchEnv(ctx, w, r)
	var status = guestManger.Status(params["<sid>"])
	appsrv.SendStruct(w, strDict{"status": status})
}

func cpusetBalance(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	wm.RunTask(func() { guestManger.CpusetBalance(ctx) })
	responseOk(ctx, w)
}

func guestActions(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, _, body := appsrv.FetchEnv(ctx, w, r)
	var sid = params["<sid>"]
	var action = params["<action>"]
	if f, ok := actionFuncs[action]; !ok {
		response(ctx, w, httperrors.NewNotFoundError("Not found"))
	} else {
		res, err := f(ctx, sid, body)
		if err != nil {
			response(ctx, w, err)
		} else if res != nil {
			response(ctx, w, res)
		} else {
			responseOk(ctx, w)
		}
	}
}

func deleteGuest(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, _, body := appsrv.FetchEnv(ctx, w, r)
	var sid = params["<sid>"]
	var migrated = jsonutils.QueryBoolean(body, "migrated", false)
	guest, err := guestManger.Delete(sid)
	if err != nil {
		response(ctx, w, err)
	} else {
		wm.RunTask(func() { guest.CleanGuest(ctx, migrated) })
		response(ctx, w, map[string]bool{"delay_clean": true})
	}
}

func doCreate(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	err := guestManger.PrepareCreate(sid)
	if err != nil {
		return nil, httperrors.NewBadRequestError(err.Error())
	}
	wm.RunTask(func() { guestManger.DoDeploy(ctx, sid, body, true) })
	return nil, nil
}

func doDeploy(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	// TODO
	return nil, nil
}

func doStart(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	res, err := guestManger.Start(ctx, sid, body)
	return nil, nil
}

func doStop(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	// TODO
	return nil, nil
}

func doMonitor(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	if body.Contains("cmd") {
		var c = make(chan string)
		cb := func(res string) {
			c <- res
		}
		cmd, _ := body.GetString("cmd")
		err := guestManger.Monitor(sid, cmd, cb)
		if err != nil {
			return nil, err
		} else {
			var res = <-c
			return strDict{"results": path.Join("\n", res)}, nil
		}
	} else {
		return nil, httperrors.NewMissingParameterError("cmd")
	}
}

func responseOk(ctx context.Context, w http.ResponseWriter) {
	response(ctx, w, strDict{"result": "ok"})
}

func response(ctx context.Context, w http.ResponseWriter, res interface{}) {
	if taskId := ctx.Value(appctx.APP_CONTEXT_KEY_TASK_ID); taskId != nil {
		w.Header().Set("X-Request-Id", taskId.(string))
	}
	switch res.(type) {
	case string:
		appsrv.Send(w, res.(string))
	case jsonutils.JSONObject:
		appsrv.SendJSON(w, res.(jsonutils.JSONObject))
	case error:
		httperrors.GeneralServerError(w, res.(error))
	default:
		appsrv.SendStruct(w, res)
	}
}

var actionFuncs = map[string]actionFunc{
	"create":               doCreate,
	"deploy":               doDeploy,
	"start":                doStart,
	"stop":                 doStop,
	"monitor":              doMonitor,
	"sync":                 doSync,
	"suspend":              doSuspend,
	"snapshot":             doSnapshot,
	"delete-snapshot":      doDeleteSnapshot,
	"reload-disk-snapshot": doReloadDiskSnapshot,
	"remove-statefile":     doRemoveStatefile,
	"io-throttle":          doIoThrottle,
	"src-prepare-migrate":  doSrcPrepareMigrate,
	"dest-prepare-migrate": doDestPrepareMigrate,
	"live-migrate":         doLiveMigrate,
	"resume":               doResume,
	"start-nbd-server":     doStartNbdServer,
	"drive-mirror":         doDriveMirror,
}
