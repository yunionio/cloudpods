package guestman

import (
	"context"
	"fmt"
	"net/http"
	"path"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
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
			fmt.Sprintf("%s/%s/<sid>/<action>", prefix, keyWord),
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
		hostutils.Response(ctx, w, httperrors.NewNotFoundError("%s Not found", action))
	} else {
		log.Infof("Guest %s Do %s", sid, action)
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
			log.Errorln(res)
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
	hostutils.DelayTask(ctx, guestManger.GuestSync, &SBaseParms{sid, body})
	return nil, nil
}

func guestSuspend(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	if !guestManger.IsGuestExist(sid) {
		return nil, httperrors.NewNotFoundError("Guest %s not found", sid)
	}
	hostutils.DelayTaskWithoutReqctx(ctx, guestManger.GuestSuspend, sid)
	return nil, nil
}

func guestSrcPrepareMigrate(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	if !guestManger.IsGuestExist(sid) {
		return nil, httperrors.NewNotFoundError("Guest %s not found", sid)
	}
	liveMigrate := jsonutils.QueryBoolean(body, "live_migrate", false)
	hostutils.DelayTask(ctx, guestManger.SrcPrepareMigrate,
		&SSrcPrepareMigrate{sid, liveMigrate})
	return nil, nil
}

func guestDestPrepareMigrate(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	if !guestManger.CanMigrate(sid) {
		return nil, httperrors.NewBadRequestError("Guest exist")
	}
	desc, err := body.Get("desc")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("desc")
	}
	qemuVersion, err := body.GetString("qemu_version")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("qemu_version")
	}
	liveMigrate := jsonutils.QueryBoolean(body, "live_migrate", false)
	isLocal, err := body.Bool("is_local_storage")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("is_local_storage")
	}
	var params = &SDestPrepareMigrate{}
	params.Sid = sid
	params.Desc = desc
	params.QemuVersion = qemuVersion
	params.LiveMigrate = liveMigrate
	if isLocal {
		serverUrl, err := body.GetString("server_url")
		if err != nil {
			return nil, httperrors.NewMissingParameterError("server_url")
		} else {
			params.ServerUrl = serverUrl
		}
		snapshotsUri, err := body.GetString("snapshots_uri")
		if err != nil {
			return nil, httperrors.NewMissingParameterError("snapshots_uri")
		} else {
			params.SnapshotsUri = snapshotsUri
		}
		disksUri, err := body.GetString("disks_uri")
		if err != nil {
			return nil, httperrors.NewMissingParameterError("disks_uri")
		} else {
			params.DisksUri = disksUri
		}
		srcSnapshots, err := body.Get("src_snapshots")
		if err != nil {
			return nil, httperrors.NewMissingParameterError("src_snapshots")
		} else {
			params.SrcSnapshots = srcSnapshots
		}
		disksBack, err := body.Get("disks_back")
		if err != nil {
			return nil, httperrors.NewMissingParameterError("disks_back")
		} else {
			params.DisksBackingFile = disksBack
		}
		disks, err := desc.GetArray("disks")
		if err != nil {
			return nil, httperrors.NewInputParameterError("Get desc disks error")
		} else {
			targetStorageId, _ := disks[0].GetString("target_storage_id")
			if len(targetStorageId) == 0 {
				return nil, httperrors.NewInputParameterError("Disk desc missing target storage id")
			}
			params.TargetStorageId = targetStorageId
		}
	}
	hostutils.DelayTask(ctx, guestManger.DestPrepareMigrate, params)
	return nil, nil
}

func guestLiveMigrate(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	if !guestManger.IsGuestExist(sid) {
		return nil, httperrors.NewNotFoundError("Guest %s not found", sid)
	}
	destPort, err := body.Int("live_migrate_dest_port")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("live_migrate_dest_port")
	}
	destIp, err := body.GetString("dest_ip")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("dest_ip")
	}
	isLocal, err := body.Bool("is_local_storage")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("is_local_storage")
	}
	hostutils.DelayTaskWithoutReqctx(ctx, guestManger.LiveMigrate, &SLiveMigrate{
		Sid: sid, DestPort: int(destPort), DestIp: destIp, IsLocal: isLocal,
	})
	return nil, nil
}

func guestResume(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	if !guestManger.IsGuestExist(sid) {
		return nil, httperrors.NewNotFoundError("Guest %s not found", sid)
	}
	isLiveMigrate := jsonutils.QueryBoolean(body, "live_migrate", false)
	guestManger.Resume(ctx, sid, isLiveMigrate)
	return nil, nil
}

// func guestStartNbdServer(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
// 	if !guestManger.IsGuestExist(sid) {
// 		return nil, httperrors.NewNotFoundError("Guest %s not found", sid)
// 	}
// 	hostutils.DelayTask(ctx, guestManger.StartNbdServer, sid)
// 	return nil, nil
// }

func guestDriveMirror(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	if !guestManger.IsGuestExist(sid) {
		return nil, httperrors.NewNotFoundError("Guest %s not found", sid)
	}
	backupNbdServerUri, err := body.GetString("backup_ndb_server_uri")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("backup_ndb_server_uri")
	}
	hostutils.DelayTaskWithoutReqctx(ctx, guestManger.StartDriveMirror,
		&SDriverMirror{sid, backupNbdServerUri})
	return nil, nil
}

func guestReloadDiskSnapshot(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	if !guestManger.IsGuestExist(sid) {
		return nil, httperrors.NewNotFoundError("Guest %s not found", sid)
	}
	diskId, err := body.GetString("disk_id")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("disk_id")
	}

	var disk storageman.IDisk
	guest := guestManger.Servers[sid]
	disks, _ := guest.Desc.GetArray("disks")
	for _, d := range disks {
		id, _ := d.GetString("disk_id")
		if diskId == id {
			diskPath, _ := d.GetString("path")
			disk = storageman.GetManager().GetDiskByPath(diskPath)
			break
		}
	}
	if disk == nil {
		return nil, httperrors.NewNotFoundError("Disk not found")
	}

	hostutils.DelayTaskWithoutReqctx(ctx, guestManger.ReloadDiskSnapshot, &SReloadDisk{sid, disk})
	return nil, nil
}

func guestSnapshot(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	if !guestManger.IsGuestExist(sid) {
		return nil, httperrors.NewNotFoundError("Guest %s not found", sid)
	}
	snapshotId, err := body.GetString("snapshot_id")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("snapshot_id")
	}
	diskId, err := body.GetString("disk_id")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("disk_id")
	}

	var disk storageman.IDisk
	guest := guestManger.Servers[sid]
	disks, _ := guest.Desc.GetArray("disks")
	for _, d := range disks {
		id, _ := d.GetString("disk_id")
		if diskId == id {
			diskPath, _ := d.GetString("path")
			disk = storageman.GetManager().GetDiskByPath(diskPath)
			break
		}
	}
	if disk == nil {
		return nil, httperrors.NewNotFoundError("Disk not found")
	}

	hostutils.DelayTask(ctx, guestManger.DoSnapshot, &SDiskSnapshot{sid, snapshotId, disk})
	return nil, nil
}

func guestDeleteSnapshot(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	deleteSnapshot, err := body.GetString("delete_snapshot")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("delete_snapshot")
	}
	diskId, err := body.GetString("disk_id")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("disk_id")
	}

	var disk storageman.IDisk
	guest := guestManger.Servers[sid]
	disks, _ := guest.Desc.GetArray("disks")
	for _, d := range disks {
		id, _ := d.GetString("disk_id")
		if diskId == id {
			diskPath, _ := d.GetString("path")
			disk = storageman.GetManager().GetDiskByPath(diskPath)
			break
		}
	}
	if disk == nil {
		return nil, httperrors.NewNotFoundError("Disk not found")
	}

	params := &SDeleteDiskSnapshot{
		Sid:            sid,
		DeleteSnapshot: deleteSnapshot,
		Disk:           disk,
	}

	if !jsonutils.QueryBoolean(body, "auto_delete", false) {
		convertSnapshot, err := body.GetString("convert_snapshot")
		if err != nil {
			return nil, httperrors.NewMissingParameterError("convert_snapshot")
		}
		params.ConvertSnapshot = convertSnapshot
		pendingDelete, err := body.Bool("pending_delete")
		if err != nil {
			return nil, httperrors.NewMissingParameterError("pending_delete")
		}
		params.PendingDelete = pendingDelete
	}
	hostutils.DelayTask(ctx, guestManger.DeleteSnapshot, params)
	return nil, nil
}

var actionFuncs = map[string]actionFunc{
	"create":  guestCreate,
	"deploy":  guestDeploy,
	"start":   guestStart,
	"stop":    guestStop,
	"monitor": guestMonitor,
	"sync":    guestSync,
	"suspend": guestSuspend,

	"snapshot":             guestSnapshot,
	"delete-snapshot":      guestDeleteSnapshot,
	"reload-disk-snapshot": guestReloadDiskSnapshot,
	// "remove-statefile":     guestRemoveStatefile,
	// "io-throttle":          guestIoThrottle,

	"src-prepare-migrate":  guestSrcPrepareMigrate,
	"dest-prepare-migrate": guestDestPrepareMigrate,
	"live-migrate":         guestLiveMigrate,
	"resume":               guestResume,
	// "start-nbd-server":     guestStartNbdServer,
	"drive-mirror": guestDriveMirror,
}
