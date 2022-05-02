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
	"net/http"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/hostman/guestman"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

type strDict map[string]string
type actionFunc func(context.Context, mcclient.TokenCredential, string, jsonutils.JSONObject) (interface{}, error)

var (
	keyWords = []string{"servers"}
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

		app.AddHandler("DELETE",
			fmt.Sprintf("%s/%s/<sid>", prefix, keyWord),
			auth.Authenticate(deleteGuest))

		for action, f := range map[string]actionFunc{
			"create":                guestCreate,
			"deploy":                guestDeploy,
			"rebuild":               guestRebuild,
			"start":                 guestStart,
			"stop":                  guestStop,
			"monitor":               guestMonitor,
			"sync":                  guestSync,
			"suspend":               guestSuspend,
			"io-throttle":           guestIoThrottle,
			"snapshot":              guestSnapshot,
			"delete-snapshot":       guestDeleteSnapshot,
			"reload-disk-snapshot":  guestReloadDiskSnapshot,
			"src-prepare-migrate":   guestSrcPrepareMigrate,
			"dest-prepare-migrate":  guestDestPrepareMigrate,
			"live-migrate":          guestLiveMigrate,
			"resume":                guestResume,
			"drive-mirror":          guestDriveMirror,
			"hotplug-cpu-mem":       guestHotplugCpuMem,
			"cancel-block-jobs":     guestCancelBlockJobs,
			"create-from-libvirt":   guestCreateFromLibvirt,
			"create-form-esxi":      guestCreateFromEsxi,
			"open-forward":          guestOpenForward,
			"list-forward":          guestListForward,
			"close-forward":         guestCloseForward,
			"storage-clone-disk":    guestStorageCloneDisk,
			"cpuset":                guestCPUSet,
			"cpuset-remove":         guestCPUSetRemove,
			"memory-snapshot":       guestMemorySnapshot,
			"memory-snapshot-reset": guestMemorySnapshotReset,
		} {
			app.AddHandler("POST",
				fmt.Sprintf("%s/%s/<sid>/%s", prefix, keyWord, action),
				auth.Authenticate(guestActions(f)),
			)
		}

		app.AddHandler("DELETE",
			fmt.Sprintf("%s/%s/memory-snapshot", prefix, keyWord),
			auth.Authenticate(guestMemorySnapshotDelete))
	}
}

func guestActions(f actionFunc) appsrv.FilterHandler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		params, _, body := appsrv.FetchEnv(ctx, w, r)
		userCred := auth.FetchUserCredential(ctx, nil)
		if body == nil {
			body = jsonutils.NewDict()
		}
		var sid = params["<sid>"]
		res, err := f(ctx, userCred, sid, body)
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
	sid := params["<sid>"]
	hostutils.DelayTaskWithoutReqctx(ctx, guestman.GetGuestManager().StatusWithBlockJobsCount, sid)
	hostutils.ResponseOk(ctx, w)
}

func cpusetBalance(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	hostutils.DelayTask(ctx, guestman.GetGuestManager().CpusetBalance, nil)
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

func guestCreate(ctx context.Context, userCred mcclient.TokenCredential, sid string, body jsonutils.JSONObject) (interface{}, error) {
	if guestman.GetGuestManager().IsGuestExist(sid) {
		return nil, httperrors.NewBadRequestError("Guest %s is exist", sid)
	}
	hostutils.DelayTaskWithWorker(ctx,
		guestman.GetGuestManager().GuestCreate,
		&guestman.SGuestDeploy{
			UserCred: userCred,

			Sid:    sid,
			Body:   body,
			IsInit: true,
		},
		guestman.NbdWorker,
	)
	return nil, nil
}

func guestDeploy(ctx context.Context, userCred mcclient.TokenCredential, sid string, body jsonutils.JSONObject) (interface{}, error) {
	err := guestman.GetGuestManager().PrepareDeploy(sid)
	if err != nil {
		return nil, err
	}
	hostutils.DelayTaskWithWorker(ctx,
		guestman.GetGuestManager().GuestDeploy,
		&guestman.SGuestDeploy{
			UserCred: userCred,

			Sid:    sid,
			Body:   body,
			IsInit: false,
		},
		guestman.NbdWorker,
	)
	return nil, nil
}

func guestRebuild(ctx context.Context, userCred mcclient.TokenCredential, sid string, body jsonutils.JSONObject) (interface{}, error) {
	err := guestman.GetGuestManager().PrepareDeploy(sid)
	if err != nil {
		return nil, err
	}
	hostutils.DelayTaskWithWorker(ctx,
		guestman.GetGuestManager().GuestDeploy,
		&guestman.SGuestDeploy{
			UserCred: userCred,

			Sid:    sid,
			Body:   body,
			IsInit: true,
		},
		guestman.NbdWorker,
	)
	return nil, nil
}

func guestStart(ctx context.Context, userCred mcclient.TokenCredential, sid string, body jsonutils.JSONObject) (interface{}, error) {
	return guestman.GetGuestManager().GuestStart(ctx, userCred, sid, body)
}

func guestStop(ctx context.Context, userCred mcclient.TokenCredential, sid string, body jsonutils.JSONObject) (interface{}, error) {
	timeout, err := body.Int("timeout")
	if err != nil {
		timeout = 30
	}
	return nil, guestman.GetGuestManager().GuestStop(ctx, sid, timeout)
}

func guestMonitor(ctx context.Context, userCred mcclient.TokenCredential, sid string, body jsonutils.JSONObject) (interface{}, error) {
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

func guestSync(ctx context.Context, userCred mcclient.TokenCredential, sid string, body jsonutils.JSONObject) (interface{}, error) {
	if !guestman.GetGuestManager().IsGuestExist(sid) {
		return nil, httperrors.NewNotFoundError("Guest %s not found", sid)
	}
	hostutils.DelayTask(ctx, guestman.GetGuestManager().GuestSync, &guestman.SBaseParms{
		Sid:  sid,
		Body: body,
	})
	return nil, nil
}

func guestSuspend(ctx context.Context, userCred mcclient.TokenCredential, sid string, body jsonutils.JSONObject) (interface{}, error) {
	if !guestman.GetGuestManager().IsGuestExist(sid) {
		return nil, httperrors.NewNotFoundError("Guest %s not found", sid)
	}
	hostutils.DelayTaskWithoutReqctx(ctx, guestman.GetGuestManager().GuestSuspend, sid)
	return nil, nil
}

func guestIoThrottle(ctx context.Context, userCred mcclient.TokenCredential, sid string, body jsonutils.JSONObject) (interface{}, error) {
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
	hostutils.DelayTaskWithoutReqctx(ctx, guestman.GetGuestManager().GuestIoThrottle, &guestman.SGuestIoThrottle{
		Sid:  sid,
		BPS:  bps,
		IOPS: iops,
	})
	return nil, nil
}

func guestSrcPrepareMigrate(ctx context.Context, userCred mcclient.TokenCredential, sid string, body jsonutils.JSONObject) (interface{}, error) {
	if !guestman.GetGuestManager().IsGuestExist(sid) {
		return nil, httperrors.NewNotFoundError("Guest %s not found", sid)
	}
	liveMigrate := jsonutils.QueryBoolean(body, "live_migrate", false)
	liveMigrateEnableTls := jsonutils.QueryBoolean(body, "enable_tls", false)
	hostutils.DelayTask(ctx, guestman.GetGuestManager().SrcPrepareMigrate,
		&guestman.SSrcPrepareMigrate{
			Sid:               sid,
			LiveMigrate:       liveMigrate,
			LiveMigrateUseTLS: liveMigrateEnableTls,
		})
	return nil, nil
}

func guestDestPrepareMigrate(ctx context.Context, userCred mcclient.TokenCredential, sid string, body jsonutils.JSONObject) (interface{}, error) {
	err := guestDestPrepareMigrateInternal(ctx, userCred, sid, body)
	if err != nil {
		guestman.GetGuestManager().CleanServer(sid)
		return nil, errors.Wrapf(err, "guestDestPrepareMigrateInternal %s", sid)
	}
	return nil, nil
}

func guestDestPrepareMigrateInternal(ctx context.Context, userCred mcclient.TokenCredential, sid string, body jsonutils.JSONObject) error {
	if !guestman.GetGuestManager().CanMigrate(sid) {
		return httperrors.NewBadRequestError("Guest exist")
	}
	desc, err := body.Get("desc")
	if err != nil {
		return httperrors.NewMissingParameterError("desc")
	}
	qemuVersion, err := body.GetString("qemu_version")
	if err != nil {
		return httperrors.NewMissingParameterError("qemu_version")
	}
	liveMigrate := jsonutils.QueryBoolean(body, "live_migrate", false)
	isLocal, err := body.Bool("is_local_storage")
	if err != nil {
		return httperrors.NewMissingParameterError("is_local_storage")
	}
	qemuCmdline, err := body.GetString("qemu_cmdline")
	if err != nil {
		return httperrors.NewMissingParameterError("qemu_cmdline")
	}
	var params = &guestman.SDestPrepareMigrate{}
	params.Sid = sid
	params.Desc = desc
	params.QemuVersion = qemuVersion
	params.LiveMigrate = liveMigrate
	params.SourceQemuCmdline = qemuCmdline
	params.EnableTLS = jsonutils.QueryBoolean(body, "enable_tls", false)
	if params.EnableTLS {
		certsObj, err := body.Get("migrate_certs")
		if err != nil {
			return httperrors.NewMissingParameterError("migrate_certs")
		}
		certs := map[string]string{}
		if err := certsObj.Unmarshal(&certs); err != nil {
			return httperrors.NewInputParameterError("unmarshal migrate_certs to map: %s", err)
		}
		params.MigrateCerts = certs
	}
	if isLocal {
		serverUrl, err := body.GetString("server_url")
		if err != nil {
			return httperrors.NewMissingParameterError("server_url")
		} else {
			params.ServerUrl = serverUrl
		}
		snapshotsUri, err := body.GetString("snapshots_uri")
		if err != nil {
			return httperrors.NewMissingParameterError("snapshots_uri")
		} else {
			params.SnapshotsUri = snapshotsUri
		}
		disksUri, err := body.GetString("disks_uri")
		if err != nil {
			return httperrors.NewMissingParameterError("disks_uri")
		} else {
			params.DisksUri = disksUri
		}
		srcSnapshots, err := body.Get("src_snapshots")
		if err != nil {
			return httperrors.NewMissingParameterError("src_snapshots")
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
			return httperrors.NewInputParameterError("Get desc disks error")
		} else {
			targetStorageIds := []string{}
			for i := 0; i < len(disks); i++ {
				targetStorageId, _ := disks[i].GetString("target_storage_id")
				if len(targetStorageId) == 0 {
					return httperrors.NewMissingParameterError("target_storage_id")
				}
				targetStorageIds = append(targetStorageIds, targetStorageId)
				// params.TargetStorageId = targetStorageId
				params.TargetStorageIds = targetStorageIds
			}

		}
		params.RebaseDisks = jsonutils.QueryBoolean(body, "rebase_disks", false)
	}

	msUri, err := body.GetString("memory_snapshots_uri")
	if err != nil {
		return httperrors.NewMissingParameterError("memory_snapshots_uri")
	}
	params.MemorySnapshotsUri = msUri
	msIds, _ := jsonutils.GetStringArray(body, "src_memory_snapshots")
	params.SrcMemorySnapshots = msIds

	hostutils.DelayTask(ctx, guestman.GetGuestManager().DestPrepareMigrate, params)
	return nil
}

func guestLiveMigrate(ctx context.Context, userCred mcclient.TokenCredential, sid string, body jsonutils.JSONObject) (interface{}, error) {
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
	enableTLS := jsonutils.QueryBoolean(body, "enable_tls", false)
	hostutils.DelayTaskWithoutReqctx(ctx, guestman.GetGuestManager().LiveMigrate, &guestman.SLiveMigrate{
		Sid:       sid,
		DestPort:  int(destPort),
		DestIp:    destIp,
		IsLocal:   isLocal,
		EnableTLS: enableTLS,
	})
	return nil, nil
}

func guestResume(ctx context.Context, userCred mcclient.TokenCredential, sid string, body jsonutils.JSONObject) (interface{}, error) {
	if !guestman.GetGuestManager().IsGuestExist(sid) {
		return nil, httperrors.NewNotFoundError("Guest %s not found", sid)
	}
	isLiveMigrate := jsonutils.QueryBoolean(body, "live_migrate", false)
	cleanTLS := jsonutils.QueryBoolean(body, "clean_tls", false)
	guestman.GetGuestManager().Resume(ctx, sid, isLiveMigrate, cleanTLS)
	return nil, nil
}

// func guestStartNbdServer(ctx context.Context, sid string, body jsonutils.JSONObject) (interface{}, error) {
// 	if !guestManger.IsGuestExist(sid) {
// 		return nil, httperrors.NewNotFoundError("Guest %s not found", sid)
// 	}
// 	hostutils.DelayTask(ctx, guestManger.StartNbdServer, sid)
// 	return nil, nil
// }

func guestDriveMirror(ctx context.Context, userCred mcclient.TokenCredential, sid string, body jsonutils.JSONObject) (interface{}, error) {
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
		&guestman.SDriverMirror{
			Sid:          sid,
			NbdServerUri: backupNbdServerUri,
			Desc:         desc,
		})
	return nil, nil
}

func guestCancelBlockJobs(ctx context.Context, userCred mcclient.TokenCredential, sid string, body jsonutils.JSONObject) (interface{}, error) {
	if !guestman.GetGuestManager().IsGuestExist(sid) {
		return nil, httperrors.NewNotFoundError("Guest %s not found", sid)
	}
	hostutils.DelayTaskWithoutReqctx(ctx, guestman.GetGuestManager().CancelBlockJobs, sid)
	return nil, nil
}

func guestHotplugCpuMem(ctx context.Context, userCred mcclient.TokenCredential, sid string, body jsonutils.JSONObject) (interface{}, error) {
	if !guestman.GetGuestManager().IsGuestExist(sid) {
		return nil, httperrors.NewNotFoundError("Guest %s not found", sid)
	}

	if guestman.GetGuestManager().Status(sid) != "running" {
		return nil, httperrors.NewBadRequestError("Guest %s not running", sid)
	}

	addCpuCount, _ := body.Int("add_cpu")
	addMemSize, _ := body.Int("add_mem")
	hostutils.DelayTaskWithoutReqctx(ctx, guestman.GetGuestManager().HotplugCpuMem,
		&guestman.SGuestHotplugCpuMem{
			Sid:         sid,
			AddCpuCount: addCpuCount,
			AddMemSize:  addMemSize,
		})
	return nil, nil
}

func guestReloadDiskSnapshot(ctx context.Context, userCred mcclient.TokenCredential, sid string, body jsonutils.JSONObject) (interface{}, error) {
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
			disk, _ = storageman.GetManager().GetDiskByPath(diskPath)
			break
		}
	}
	if disk == nil {
		return nil, httperrors.NewNotFoundError("Disk not found")
	}

	hostutils.DelayTask(ctx, guestman.GetGuestManager().ReloadDiskSnapshot, &guestman.SReloadDisk{
		Sid:  sid,
		Disk: disk,
	})
	return nil, nil
}

func guestSnapshot(ctx context.Context, userCred mcclient.TokenCredential, sid string, body jsonutils.JSONObject) (interface{}, error) {
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
	log.Infof("guest info: %s", jsonutils.Marshal(guest))

	var disk storageman.IDisk
	disks, _ := guest.Desc.GetArray("disks")
	for _, d := range disks {
		id, _ := d.GetString("disk_id")
		if diskId == id {
			diskPath, _ := d.GetString("path")
			disk, err = storageman.GetManager().GetDiskByPath(diskPath)
			if err != nil {
				return nil, errors.Wrapf(err, "GetDiskByPath(%s)", diskPath)
			}
			break
		}
	}
	if disk == nil {
		return nil, httperrors.NewNotFoundError("Disk not found")
	}

	hostutils.DelayTask(ctx, guestman.GetGuestManager().DoSnapshot, &guestman.SDiskSnapshot{
		Sid:        sid,
		SnapshotId: snapshotId,
		Disk:       disk,
	})
	return nil, nil
}

func guestDeleteSnapshot(ctx context.Context, userCred mcclient.TokenCredential, sid string, body jsonutils.JSONObject) (interface{}, error) {
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
			disk, err = storageman.GetManager().GetDiskByPath(diskPath)
			if err != nil {
				return nil, errors.Wrapf(err, "GetDiskByPath(%s)", diskPath)
			}
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

func guestStorageCloneDisk(ctx context.Context, userCred mcclient.TokenCredential, sid string, body jsonutils.JSONObject) (interface{}, error) {
	input := new(computeapi.ServerChangeDiskStorageInternalInput)
	if err := body.Unmarshal(input); err != nil {
		return nil, err
	}
	srcStorage := storageman.GetManager().GetStorage(input.StorageId)
	if srcStorage == nil {
		return nil, httperrors.NewNotFoundError("Source storage %q not found", input.StorageId)
	}
	srcDisk, err := srcStorage.GetDiskById(input.DiskId)
	if err != nil {
		return nil, errors.Wrapf(err, "Get source disk %q on storage %q", input.DiskId, srcStorage.GetId())
	}
	targetStorage := storageman.GetManager().GetStorage(input.TargetStorageId)
	if targetStorage == nil {
		return nil, httperrors.NewNotFoundError("Target storage %s not found", input.TargetStorageId)
	}
	if input.TargetDiskId == "" {
		return nil, httperrors.NewMissingParameterError("Target disk id is empty")
	}

	params := &guestman.SStorageCloneDisk{
		ServerId:      sid,
		SourceStorage: srcStorage,
		SourceDisk:    srcDisk,
		TargetStorage: targetStorage,
		TargetDiskId:  input.TargetDiskId,
	}
	hostutils.DelayTaskWithoutReqctx(ctx, guestman.GetGuestManager().StorageCloneDisk, params)
	return nil, nil
}

func guestCPUSet(ctx context.Context, userCred mcclient.TokenCredential, sid string, body jsonutils.JSONObject) (interface{}, error) {
	input := new(computeapi.ServerCPUSetInput)
	if err := body.Unmarshal(input); err != nil {
		return nil, err
	}
	gm := guestman.GetGuestManager()
	return gm.CPUSet(ctx, sid, input)
}

func guestCPUSetRemove(ctx context.Context, userCred mcclient.TokenCredential, sid string, body jsonutils.JSONObject) (interface{}, error) {
	gm := guestman.GetGuestManager()
	if err := gm.CPUSetRemove(ctx, sid); err != nil {
		return nil, err
	}
	return nil, nil
}

func guestMemorySnapshot(ctx context.Context, userCred mcclient.TokenCredential, sid string, body jsonutils.JSONObject) (interface{}, error) {
	input := new(hostapi.GuestMemorySnapshotRequest)
	if err := body.Unmarshal(input); err != nil {
		return nil, err
	}
	gm := guestman.GetGuestManager()
	hostutils.DelayTaskWithoutReqctx(ctx, gm.DoMemorySnapshot, &guestman.SMemorySnapshot{
		GuestMemorySnapshotRequest: input,
		Sid:                        sid,
	})
	return nil, nil
}

func guestMemorySnapshotReset(ctx context.Context, userCred mcclient.TokenCredential, sid string, body jsonutils.JSONObject) (interface{}, error) {
	input := new(hostapi.GuestMemorySnapshotResetRequest)
	if err := body.Unmarshal(input); err != nil {
		return nil, err
	}
	if input.InstanceSnapshotId == "" {
		return nil, httperrors.NewMissingParameterError("instance_snapshot_id")
	}
	if input.Path == "" {
		return nil, httperrors.NewMissingParameterError("path")
	}
	gm := guestman.GetGuestManager()
	hostutils.DelayTaskWithoutReqctx(ctx, gm.DoResetMemorySnapshot, &guestman.SMemorySnapshotReset{
		GuestMemorySnapshotResetRequest: input,
		Sid:                             sid,
	})
	return nil, nil
}

func guestMemorySnapshotDelete(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, _, body := appsrv.FetchEnv(ctx, w, r)
	input := new(hostapi.GuestMemorySnapshotDeleteRequest)
	if err := body.Unmarshal(input); err != nil {
		hostutils.Response(ctx, w, err)
		return
	}
	if input.InstanceSnapshotId == "" {
		hostutils.Response(ctx, w, httperrors.NewMissingParameterError("instance_snapshot_id"))
		return
	}
	if input.Path == "" {
		hostutils.Response(ctx, w, httperrors.NewMissingParameterError("path"))
		return
	}
	gm := guestman.GetGuestManager()
	hostutils.DelayTask(ctx, gm.DoDeleteMemorySnapshot, &guestman.SMemorySnapshotDelete{
		GuestMemorySnapshotDeleteRequest: input,
	})
}
