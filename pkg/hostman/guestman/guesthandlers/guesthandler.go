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

package guesthandlers

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/hostman/guestman"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

type strDict map[string]string
type actionFunc func(context.Context, string, jsonutils.JSONObject) (interface{}, error)

var (
	keyWords    = []string{"servers"}
	actionFuncs = map[string]actionFunc{
		"create":      guestCreate,
		"deploy":      guestDeploy,
		"start":       guestStart,
		"stop":        guestStop,
		"monitor":     guestMonitor,
		"sync":        guestSync,
		"suspend":     guestSuspend,
		"io-throttle": guestIoThrottle,

		"snapshot":             guestSnapshot,
		"delete-snapshot":      guestDeleteSnapshot,
		"reload-disk-snapshot": guestReloadDiskSnapshot,
		// "remove-statefile":     guestRemoveStatefile,
		"src-prepare-migrate":  guestSrcPrepareMigrate,
		"dest-prepare-migrate": guestDestPrepareMigrate,
		"live-migrate":         guestLiveMigrate,
		"resume":               guestResume,
		// "start-nbd-server":     guestStartNbdServer,
		"drive-mirror":        guestDriveMirror,
		"hotplug-cpu-mem":     guestHotplugCpuMem,
		"create-from-libvirt": guestCreateFromLibvirt,
		"cancel-block-jobs":   guestCancelBlockJobs,
	}
)

func AddGuestTaskHandler(prefix string, app *appsrv.Application) {
	for _, keyWord := range keyWords {
		app.AddHandler("GET",
			fmt.Sprintf("%s/%s/<sid>/status", prefix, keyWord),
			auth.Authenticate(getStatus))

		app.AddHandler("POST",
			fmt.Sprintf("%s/%s/cpu-node-balance", prefix, keyWord),
			auth.Authenticate(cpusetBalance))

		app.AddHandler("POST",
			fmt.Sprintf("%s/%s/prepare-import-from-libvirt", prefix, keyWord),
			auth.Authenticate(guestPrepareImportFormLibvirt))

		app.AddHandler("POST",
			fmt.Sprintf("%s/%s/<sid>/<action>", prefix, keyWord),
			auth.Authenticate(guestActions))

		app.AddHandler("DELETE",
			fmt.Sprintf("%s/%s/<sid>", prefix, keyWord),
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
	var status = guestman.GetGuestManager().Status(params["<sid>"])
	appsrv.SendStruct(w, strDict{"status": status})
}

func cpusetBalance(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	hostutils.DelayTask(ctx, guestman.GetGuestManager().CpusetBalance, nil)
	hostutils.ResponseOk(ctx, w)
}

func guestPrepareImportFormLibvirt(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, _, body := appsrv.FetchEnv(ctx, w, r)
	config := &compute.SLibvirtHostConfig{}
	err := body.Unmarshal(config)
	if err != nil {
		hostutils.Response(ctx, w, httperrors.NewInputParameterError("Parse params to libvirt config error %s", err))
		return
	}
	if len(config.XmlFilePath) == 0 {
		hostutils.Response(ctx, w, httperrors.NewMissingParameterError("xml_file_path"))
		return
	}
	if !fileutils2.Exists(config.XmlFilePath) {
		hostutils.Response(ctx, w,
			httperrors.NewBadRequestError("xml_file_path %s not found", config.XmlFilePath))
		return
	}

	if len(config.Servers) == 0 {
		hostutils.Response(ctx, w, httperrors.NewMissingParameterError("servers"))
		return
	}

	if len(config.MonitorPath) > 0 {
		if _, err := ioutil.ReadDir(config.MonitorPath); err != nil {
			hostutils.Response(ctx, w,
				httperrors.NewBadRequestError("Monitor path %s can't open as dir: %s", config.MonitorPath, err))
			return
		}
	}

	hostutils.DelayTask(ctx, guestman.GetGuestManager().PrepareImportFromLibvirt, config)
	hostutils.ResponseOk(ctx, w)
}

func deleteGuest(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, _, body := appsrv.FetchEnv(ctx, w, r)
	var sid = params["<sid>"]
	var migrated bool
	if body != nil {
		migrated = jsonutils.QueryBoolean(body, "migrated", false)
	}
	guest, err := guestman.GetGuestManager().Delete(sid)
	if err != nil {
		hostutils.Response(ctx, w, err)
	} else {
		hostutils.DelayTask(ctx, guest.CleanGuest, migrated)
		hostutils.Response(ctx, w, map[string]bool{"delay_clean": true})
	}
}

func guestCreate(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	err := guestman.GetGuestManager().PrepareCreate(sid)
	if err != nil {
		return nil, err
	}
	hostutils.DelayTask(ctx, guestman.GetGuestManager().GuestDeploy, &guestman.SGuestDeploy{sid, body, true})
	return nil, nil
}

func guestDeploy(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	err := guestman.GetGuestManager().PrepareDeploy(sid)
	if err != nil {
		return nil, err
	}
	hostutils.DelayTask(ctx, guestman.GetGuestManager().GuestDeploy, &guestman.SGuestDeploy{sid, body, false})
	return nil, nil
}

func guestStart(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	return guestman.GetGuestManager().GuestStart(ctx, sid, body)
}

func guestStop(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	timeout, err := body.Int("timeout")
	if err != nil {
		timeout = 30
	}
	return nil, guestman.GetGuestManager().GuestStop(ctx, sid, timeout)
}

func guestMonitor(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	if !guestman.GetGuestManager().IsGuestExist(sid) {
		return nil, httperrors.NewNotFoundError("Guest %s not found", sid)
	}

	if body.Contains("cmd") {
		var c = make(chan string)
		cb := func(res string) {
			c <- res
		}
		cmd, _ := body.GetString("cmd")
		err := guestman.GetGuestManager().Monitor(sid, cmd, cb)
		if err != nil {
			return nil, err
		} else {
			var res = <-c
			lines := strings.Split(res, "\\r\\n")

			return strDict{"results": strings.Join(lines, "\n")}, nil
		}
	} else {
		return nil, httperrors.NewMissingParameterError("cmd")
	}
}

func guestSync(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	if !guestman.GetGuestManager().IsGuestExist(sid) {
		return nil, httperrors.NewNotFoundError("Guest %s not found", sid)
	}
	hostutils.DelayTask(ctx, guestman.GetGuestManager().GuestSync, &guestman.SBaseParms{sid, body})
	return nil, nil
}

func guestSuspend(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	if !guestman.GetGuestManager().IsGuestExist(sid) {
		return nil, httperrors.NewNotFoundError("Guest %s not found", sid)
	}
	hostutils.DelayTaskWithoutReqctx(ctx, guestman.GetGuestManager().GuestSuspend, sid)
	return nil, nil
}

func guestIoThrottle(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	guest, ok := guestman.GetGuestManager().GetServer(sid)
	if !ok {
		return nil, httperrors.NewNotFoundError("Guest %s not found", sid)
	}
	if !guest.IsRunning() {
		return nil, httperrors.NewInvalidStatusError("Not running")
	}
	bps, err := body.Int("bps")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("bps")
	}
	iops, err := body.Int("iops")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("iops")
	}
	hostutils.DelayTaskWithoutReqctx(ctx, guestman.GetGuestManager().GuestIoThrottle, &guestman.SGuestIoThrottle{sid, bps, iops})
	return nil, nil
}

func guestSrcPrepareMigrate(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	if !guestman.GetGuestManager().IsGuestExist(sid) {
		return nil, httperrors.NewNotFoundError("Guest %s not found", sid)
	}
	liveMigrate := jsonutils.QueryBoolean(body, "live_migrate", false)
	hostutils.DelayTask(ctx, guestman.GetGuestManager().SrcPrepareMigrate,
		&guestman.SSrcPrepareMigrate{sid, liveMigrate})
	return nil, nil
}

func guestDestPrepareMigrate(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	if !guestman.GetGuestManager().CanMigrate(sid) {
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
	var params = &guestman.SDestPrepareMigrate{}
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
			params.DisksBackingFile = jsonutils.NewDict()
		} else {
			params.DisksBackingFile = disksBack
		}
		disks, err := desc.GetArray("disks")
		if err != nil {
			return nil, httperrors.NewInputParameterError("Get desc disks error")
		} else {
			targetStorageId, _ := disks[0].GetString("target_storage_id")
			if len(targetStorageId) == 0 {
				return nil, httperrors.NewMissingParameterError("target_storage_id")
			}
			params.TargetStorageId = targetStorageId
		}
		params.RebaseDisks = jsonutils.QueryBoolean(body, "rebase_disks", false)
	}
	hostutils.DelayTask(ctx, guestman.GetGuestManager().DestPrepareMigrate, params)
	return nil, nil
}

func guestLiveMigrate(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	if !guestman.GetGuestManager().IsGuestExist(sid) {
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
	hostutils.DelayTaskWithoutReqctx(ctx, guestman.GetGuestManager().LiveMigrate, &guestman.SLiveMigrate{
		Sid: sid, DestPort: int(destPort), DestIp: destIp, IsLocal: isLocal,
	})
	return nil, nil
}

func guestResume(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	if !guestman.GetGuestManager().IsGuestExist(sid) {
		return nil, httperrors.NewNotFoundError("Guest %s not found", sid)
	}
	isLiveMigrate := jsonutils.QueryBoolean(body, "live_migrate", false)
	guestman.GetGuestManager().Resume(ctx, sid, isLiveMigrate)
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
	if !guestman.GetGuestManager().IsGuestExist(sid) {
		return nil, httperrors.NewNotFoundError("Guest %s not found", sid)
	}
	backupNbdServerUri, err := body.GetString("backup_nbd_server_uri")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("backup_nbd_server_uri")
	}
	desc, err := body.Get("desc")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("desc")
	}
	hostutils.DelayTaskWithoutReqctx(ctx, guestman.GetGuestManager().StartDriveMirror,
		&guestman.SDriverMirror{sid, backupNbdServerUri, desc})
	return nil, nil
}

func guestCancelBlockJobs(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	if !guestman.GetGuestManager().IsGuestExist(sid) {
		return nil, httperrors.NewNotFoundError("Guest %s not found", sid)
	}
	hostutils.DelayTaskWithoutReqctx(ctx, guestman.GetGuestManager().CancelBlockJobs, sid)
	return nil, nil
}

func guestHotplugCpuMem(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	if !guestman.GetGuestManager().IsGuestExist(sid) {
		return nil, httperrors.NewNotFoundError("Guest %s not found", sid)
	}

	if guestman.GetGuestManager().Status(sid) != "running" {
		return nil, httperrors.NewBadRequestError("Guest %s not running", sid)
	}

	addCpuCount, _ := body.Int("add_cpu")
	addMemSize, _ := body.Int("add_mem")
	hostutils.DelayTaskWithoutReqctx(ctx, guestman.GetGuestManager().HotplugCpuMem,
		&guestman.SGuestHotplugCpuMem{sid, addCpuCount, addMemSize})
	return nil, nil
}

func guestCreateFromLibvirt(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	err := guestman.GetGuestManager().PrepareCreate(sid)
	if err != nil {
		return nil, err
	}

	iGuestDesc, err := body.Get("desc")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("desc")
	}
	guestDesc, ok := iGuestDesc.(*jsonutils.JSONDict)
	if !ok {
		return nil, httperrors.NewInputParameterError("desc is not dict")
	}

	iDisksPath, err := body.Get("disks_path")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("disks_path")
	}
	disksPath, ok := iDisksPath.(*jsonutils.JSONDict)
	if !ok {
		return nil, httperrors.NewInputParameterError("disks_path is not dict")
	}

	monitorPath, _ := body.GetString("monitor_path")
	if len(monitorPath) > 0 && !fileutils2.Exists(monitorPath) {
		return nil, httperrors.NewBadRequestError("Monitor path %s not found", monitorPath)
	}

	hostutils.DelayTask(ctx, guestman.GetGuestManager().GuestCreateFromLibvirt,
		&guestman.SGuestCreateFromLibvirt{sid, monitorPath, guestDesc, disksPath})
	return nil, nil
}

func guestReloadDiskSnapshot(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	diskId, err := body.GetString("disk_id")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("disk_id")
	}
	guest, ok := guestman.GetGuestManager().GetServer(sid)
	if !ok {
		return nil, httperrors.NewNotFoundError("guest %s not found", sid)
	}

	var disk storageman.IDisk
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

	hostutils.DelayTask(ctx, guestman.GetGuestManager().ReloadDiskSnapshot, &guestman.SReloadDisk{sid, disk})
	return nil, nil
}

func guestSnapshot(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
	snapshotId, err := body.GetString("snapshot_id")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("snapshot_id")
	}
	diskId, err := body.GetString("disk_id")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("disk_id")
	}
	guest, ok := guestman.GetGuestManager().GetServer(sid)
	if !ok {
		return nil, httperrors.NewNotFoundError("guest %s not found", sid)
	}

	var disk storageman.IDisk
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

	hostutils.DelayTask(ctx, guestman.GetGuestManager().DoSnapshot, &guestman.SDiskSnapshot{sid, snapshotId, disk})
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
	guest, ok := guestman.GetGuestManager().GetServer(sid)
	if !ok {
		return nil, httperrors.NewNotFoundError("guest %s not found", sid)
	}

	var disk storageman.IDisk
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

	params := &guestman.SDeleteDiskSnapshot{
		Sid:            sid,
		DeleteSnapshot: deleteSnapshot,
		Disk:           disk,
	}

	if !jsonutils.QueryBoolean(body, "auto_deleted", false) {
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
	hostutils.DelayTask(ctx, guestman.GetGuestManager().DeleteSnapshot, params)
	return nil, nil
}
