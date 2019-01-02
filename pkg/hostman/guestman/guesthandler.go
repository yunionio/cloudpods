package guestman

import (
	"context"
	"fmt"
	"net/http"
	"path"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

var keyWords = []string{"servers"}

type strDict map[string]string
type actionFunc func(context.Context, string, jsonutils.JSONObject) (interface{}, error)

func AddGuestTaskHandler(prefix string, app *appsrv.Application) {
	for _, keyWord := range keyWords {
		app.AddHandler("GET",
			fmt.Sprintf("%s/%s/<sid>/status", prefix, keyWord),
			auth.Authenticate(getStatus))

		app.AddHandler("POST",
			fmt.Sprintf("%s/%s/cpu-node-balance", prefix, keyWord),
			auth.Authenticate(cpusetBalance))

		app.AddHandler("POST",
			fmt.Sprintf("%s/%s/servers/<sid>/<action>", prefix, keyWord),
			auth.Authenticate(guestActions))

		app.AddHandler("DELETE",
			fmt.Sprintf("%s/%s/servers/<sid>", prefix, keyWord),
			auth.Authenticate(deleteGuest))
	}
}

func guestActions(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, _, body := appsrv.FetchEnv(ctx, w, r)
	if body == nil {
		body = jsonutils.NewDict()
	}
	var sid = params["<sid>"]
	var action = params["<action>"]
	if f, ok := actionFuncs[action]; !ok {
		hostutils.Response(ctx, w, httperrors.NewNotFoundError("Not found"))
	} else {
		res, err := f(ctx, sid, body)
		if err != nil {
			hostutils.Response(ctx, w, err)
		} else if res != nil {
			hostutils.Response(ctx, w, res)
		} else {
			hostutils.ResponseOk(ctx, w)
		}
	}
}

func getStatus(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, _, _ := appsrv.FetchEnv(ctx, w, r)
	var status = guestManger.Status(params["<sid>"])
	appsrv.SendStruct(w, strDict{"status": status})
}

func cpusetBalance(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	hostutils.DelayTask(ctx, guestManger.CpusetBalance, nil)
	hostutils.ResponseOk(ctx, w)
}

func deleteGuest(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, _, body := appsrv.FetchEnv(ctx, w, r)
	var sid = params["<sid>"]
	var migrated = jsonutils.QueryBoolean(body, "migrated", false)
	guest, err := guestManger.Delete(sid)
	if err != nil {
		hostutils.Response(ctx, w, err)
	} else {
		hostutils.DelayTask(ctx, guest.CleanGuest, migrated)
		hostutils.Response(ctx, w, map[string]bool{"delay_clean": true})
	}
}

func guestCreate(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	err := guestManger.PrepareCreate(sid)
	if err != nil {
		return nil, err
	}
	hostutils.DelayTask(ctx, guestManger.GuestDeploy, &SGuestDeploy{sid, body, true})
	return nil, nil
}

func guestDeploy(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	err := guestManger.PrepareDeploy(sid)
	if err != nil {
		return nil, err
	}
	hostutils.DelayTask(ctx, guestManger.GuestDeploy, &SGuestDeploy{sid, body, false})
	return nil, nil
}

func guestStart(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	return guestManger.GuestStart(ctx, sid, body)
}

func guestStop(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	timeout, err := body.Int("timeout")
	if err != nil {
		timeout = 30
	}
	return nil, guestManger.GuestStop(ctx, sid, timeout)
}

func guestMonitor(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
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

func guestSync(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	if !guestManger.IsGuestExist(sid) {
		return nil, httperrors.NewNotFoundError("Guest %s not found", sid)
	}
	hostutils.DelayTask(ctx, guestManger.GuestSync, &SGuestSync{sid, body})
	return nil, nil
}

func guestSuspend(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	if !guestManger.IsGuestExist(sid) {
		return nil, httperrors.NewNotFoundError("Guest %s not found", sid)
	}
	hostutils.DelayTaskWithoutTask(ctx, guestManger.GuestSuspend, sid)
	return nil, nil
}

// snapshot 相关的有待重构，放入diskhandler中去，先不写
func guestSnapshot(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	if !guestManger.IsGuestExist(sid) {
		return nil, httperrors.NewNotFoundError("Guest %s not found", sid)
	}
	hostutils.DelayTask(ctx, guestManger.DoSnapshot, params)
}

func guestSrcPrepareMigrate(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
}

func guestDestPrepareMigrate(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
}

func guestLiveMigrate(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
}

func guestResume(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
}

func guestStartNbdServer(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
}

func guestDriveMirror(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
}

var actionFuncs = map[string]actionFunc{
	"create":  guestCreate,
	"deploy":  guestDeploy,
	"start":   guestStart,
	"stop":    guestStop,
	"monitor": guestMonitor,
	"sync":    guestSync,
	"suspend": guestSuspend,

	"snapshot": guestSnapshot,
	// "delete-snapshot":      guestDeleteSnapshot,
	// "reload-disk-snapshot": guestReloadDiskSnapshot,
	// "remove-statefile":     guestRemoveStatefile,
	// "io-throttle":          guestIoThrottle,

	"src-prepare-migrate":  guestSrcPrepareMigrate,
	"dest-prepare-migrate": guestDestPrepareMigrate,
	"live-migrate":         guestLiveMigrate,
	"resume":               guestResume,
	"start-nbd-server":     guestStartNbdServer,
	"drive-mirror":         guestDriveMirror,
}
