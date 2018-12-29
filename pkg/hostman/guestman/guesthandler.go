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

func guestActions(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, _, body := appsrv.FetchEnv(ctx, w, r)
	if body == nil {
		body = jsonutils.NewDict()
	}
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

func getStatus(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, _, _ := appsrv.FetchEnv(ctx, w, r)
	var status = guestManger.Status(params["<sid>"])
	appsrv.SendStruct(w, strDict{"status": status})
}

func cpusetBalance(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	wm.DelayTask(ctx, guestManger.CpusetBalance, nil)
	responseOk(ctx, w)
}

func deleteGuest(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, _, body := appsrv.FetchEnv(ctx, w, r)
	var sid = params["<sid>"]
	var migrated = jsonutils.QueryBoolean(body, "migrated", false)
	guest, err := guestManger.Delete(sid)
	if err != nil {
		response(ctx, w, err)
	} else {
		wm.DelayTask(ctx, guest.CleanGuest, migrated)
		response(ctx, w, map[string]bool{"delay_clean": true})
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

func doCreate(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	err := guestManger.PrepareCreate(sid)
	if err != nil {
		return nil, err
	}
	wm.DelayTask(ctx, guestManger.GuestDeploy, &SGuestDeploy{sid, body, true})
	return nil, nil
}

func doDeploy(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	err := guestManger.PrepareDeploy(sid)
	if err != nil {
		return nil, err
	}
	wm.DelayTask(ctx, guestManger.GuestDeploy, &SGuestDeploy{sid, body, false})
	return nil, nil
}

func doStart(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	return guestManger.GuestStart(ctx, sid, body)
}

func doStop(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	timeout, err := body.Int("timeout")
	if err != nil {
		timeout = 30
	}
	return nil, guestManger.GuestStop(ctx, sid, timeout)
}

func doMonitor(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	if !guestManger.IsGuestExist(sid) {
		return nil, httperrors.NewNotFoundError("Guest %s not found", sid)
	}

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

func doSync(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	if !guestManger.IsGuestExist(sid) {
		return nil, httperrors.NewNotFoundError("Guest %s not found", sid)
	}
	wm.DelayTask(ctx, guestManger.GuestSync, &SGuestSync{sid, body})
	return nil, nil
}

func doSuspend(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	if !guestManger.IsGuestExist(sid) {
		return nil, httperrors.NewNotFoundError("Guest %s not found", sid)
	}
	wm.DelayTaskWithoutTask(ctx, guestManger.GuestSuspend, sid)
	return nil, nil
}

// snapshot 相关的有待重构，放入diskhandler中去，先不写
func doSnapshot(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	if !guestManger.IsGuestExist(sid) {
		return nil, httperrors.NewNotFoundError("Guest %s not found", sid)
	}
	wm.DelayTask(ctx, guestManger.DoSnapshot, params)
}

func doSrcPrepareMigrate(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
}

func doDestPrepareMigrate(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
}

func doLiveMigrate(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
}

func doResume(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
}

func doStartNbdServer(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
}

func doDriveMirror(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
}

var actionFuncs = map[string]actionFunc{
	"create":  doCreate,
	"deploy":  doDeploy,
	"start":   doStart,
	"stop":    doStop,
	"monitor": doMonitor,
	"sync":    doSync,
	"suspend": doSuspend,

	"snapshot": doSnapshot,
	// "delete-snapshot":      doDeleteSnapshot,
	// "reload-disk-snapshot": doReloadDiskSnapshot,
	// "remove-statefile":     doRemoveStatefile,
	// "io-throttle":          doIoThrottle,

	"src-prepare-migrate":  doSrcPrepareMigrate,
	"dest-prepare-migrate": doDestPrepareMigrate,
	"live-migrate":         doLiveMigrate,
	"resume":               doResume,
	"start-nbd-server":     doStartNbdServer,
	"drive-mirror":         doDriveMirror,
}
