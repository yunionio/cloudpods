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

package diskhandlers

import (
	"context"
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/regutils"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/workmanager"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/hostman/guestman"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

var (
	keyWords         = []string{"disks"}
	snapshotKeywords = []string{"snapshots"}

	actionFuncs = map[string]actionFunc{
		"create":            diskCreate,
		"delete":            diskDelete,
		"resize":            diskResize,
		"save-prepare":      diskSavePrepare,
		"reset":             diskReset,
		"snapshot":          diskSnapshot,
		"delete-snapshot":   diskDeleteSnapshot,
		"cleanup-snapshots": diskCleanupSnapshots,
		"backup":            diskBackup,
	}
)

type actionFunc func(context.Context, storageman.IStorage, string, storageman.IDisk, jsonutils.JSONObject) (interface{}, error)

func AddDiskHandler(prefix string, app *appsrv.Application) {
	for _, keyWord := range keyWords {
		for _, seg := range []string{"iso_cache", "image_cache"} {
			app.AddHandler("POST",
				fmt.Sprintf("%s/%s/%s", prefix, keyWord, seg),
				auth.Authenticate(perfetchImageCache))

			app.AddHandler("DELETE",
				fmt.Sprintf("%s/%s/%s", prefix, keyWord, seg),
				auth.Authenticate(deleteImageCache))
		}

		app.AddHandler("POST",
			fmt.Sprintf("%s/%s/<storageId>/upload", prefix, keyWord),
			auth.Authenticate(saveToGlance))

		app.AddHandler("POST",
			fmt.Sprintf("%s/%s/<storageId>/<action>/<diskId>", prefix, keyWord),
			auth.Authenticate(performDiskActions))
		app.AddHandler("GET",
			fmt.Sprintf("%s/%s/<storageId>/<diskId>/status", prefix, keyWord),
			auth.Authenticate(getDiskStatus))
	}
	for _, keyWord := range snapshotKeywords {
		app.AddHandler("GET",
			fmt.Sprintf("%s/%s/<storageId>/<diskId>/<snapshotId>/status", prefix, keyWord),
			auth.Authenticate(getSnapshotStatus),
		)
	}

}

func performImageCache(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	performAction string,
) {
	_, _, body := appsrv.FetchEnv(ctx, w, r)

	disk, err := body.Get("disk")
	if err != nil {
		httperrors.MissingParameterError(ctx, w, "disk")
		return
	}
	scId, err := disk.GetString("storagecache_id")
	if err != nil {
		httperrors.MissingParameterError(ctx, w, "disk")
		return
	}
	storagecache := storageman.GetManager().GetStoragecacheById(scId)
	if storagecache == nil {
		httperrors.NotFoundError(ctx, w, "Storagecache %s not found", scId)
		return
	}

	var performTask workmanager.DelayTaskFunc
	if performAction == "perfetch" {
		performTask = storagecache.PrefetchImageCache
	} else {
		performTask = storagecache.DeleteImageCache
	}

	hostutils.DelayTask(ctx, performTask, disk)
	hostutils.ResponseOk(ctx, w)
}

func perfetchImageCache(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	performImageCache(ctx, w, r, "perfetch")
}

func deleteImageCache(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	performImageCache(ctx, w, r, "delete")

}

func getDiskStatus(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, _, _ := appsrv.FetchEnv(ctx, w, r)
	var (
		storageId = params["<storageId>"]
		diskId    = params["<diskId>"]
	)

	storage := storageman.GetManager().GetStorage(storageId)
	if storage == nil {
		hostutils.Response(ctx, w, httperrors.NewNotFoundError("Storage %s not found", storageId))
		return
	}
	ret := jsonutils.NewDict()
	_, err := storage.GetDiskById(diskId)
	if err != nil {
		if errors.Cause(err) != cloudprovider.ErrNotFound {
			hostutils.Response(ctx, w, httperrors.NewGeneralError(errors.Wrapf(err, "GetDiskById(%s)", diskId)))
			return
		}
		ret.Set("status", jsonutils.NewString(compute.DISK_NOT_EXIST))
	} else {
		// Note: the statuses of disk on host are either exist or not exist
		ret.Set("status", jsonutils.NewString(compute.DISK_EXIST))
	}
	hostutils.Response(ctx, w, ret)
}

func getSnapshotStatus(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, _, _ := appsrv.FetchEnv(ctx, w, r)
	var (
		storageId  = params["<storageId>"]
		diskId     = params["<diskId>"]
		snapshotId = params["<snapshotId>"]
	)

	storage := storageman.GetManager().GetStorage(storageId)
	if storage == nil {
		hostutils.Response(ctx, w, httperrors.NewNotFoundError("Storage %s not found", storageId))
		return
	}

	var (
		ret    = jsonutils.NewDict()
		status string
	)
	if exist, err := storage.IsSnapshotExist(diskId, snapshotId); exist {
		status = compute.SNAPSHOT_EXIST
	} else if !exist && err == nil {
		status = compute.SNAPSHOT_NOT_EXIST
	} else {
		log.Errorf("fetch snapshot exist failed %s", err)
		status = compute.SNAPSHOT_UNKNOWN
	}
	ret.Set("status", jsonutils.NewString(status))
	hostutils.Response(ctx, w, ret)
}

func saveToGlance(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, _, body := appsrv.FetchEnv(ctx, w, r)
	var (
		storageId   = params["<storageId>"]
		diskInfo, _ = body.Get("disk")
	)
	storage := storageman.GetManager().GetStorage(storageId)
	if storage == nil {
		hostutils.Response(ctx, w, httperrors.NewNotFoundError("Storage %s not found", storageId))
		return
	}
	if diskInfo == nil {
		hostutils.Response(ctx, w, httperrors.NewMissingParameterError("disk"))
		return
	}

	hostutils.DelayTaskWithoutReqctx(ctx, storage.SaveToGlance, diskInfo)
	hostutils.ResponseOk(ctx, w)
}

func performDiskActions(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, _, body := appsrv.FetchEnv(ctx, w, r)
	if body == nil {
		body = jsonutils.NewDict()
	}

	var (
		storageId = params["<storageId>"] // seg1
		action    = params["<action>"]    // seg2
		diskId    = params["<diskId>"]    // seg3
	)
	if !regutils.MatchUUID(storageId) {
		hostutils.Response(ctx, w, httperrors.NewNotFoundError("Not found"))
		return
	}
	storage := storageman.GetManager().GetStorage(storageId)
	if storage == nil {
		hostutils.Response(ctx, w, httperrors.NewNotFoundError("Storage %s not found", storageId))
		return
	}

	var disk storageman.IDisk
	var err error

	rebuild, _ := body.Bool("disk", "rebuild")
	if action != "create" || rebuild {
		disk, err = storage.GetDiskById(diskId)
		if err != nil {
			hostutils.Response(ctx, w, httperrors.NewGeneralError(errors.Wrapf(err, "GetDiskById(%s)", diskId)))
			return
		}
	}

	f, ok := actionFuncs[action]
	if !ok {
		hostutils.Response(ctx, w, httperrors.NewNotFoundError("Action %s Not found", action))
		return
	}
	res, err := f(ctx, storage, diskId, disk, body)
	if err != nil {
		hostutils.Response(ctx, w, err)
		return
	}
	if res != nil {
		hostutils.Response(ctx, w, res)
		return
	}
	hostutils.ResponseOk(ctx, w)
}

func diskCreate(ctx context.Context, storage storageman.IStorage, diskId string, disk storageman.IDisk, body jsonutils.JSONObject) (interface{}, error) {
	params := storageman.SDiskCreateByDiskinfo{
		DiskId:   diskId,
		Disk:     disk,
		DiskInfo: compute.DiskAllocateInput{},
		Storage:  storage,
	}
	err := body.Unmarshal(&params.DiskInfo, "disk")
	if err != nil {
		return nil, errors.Wrapf(err, "body.Unmarshal")
	}
	hostutils.DelayTask(ctx, storage.CreateDiskByDiskinfo, &params)
	return nil, nil
}

func diskDelete(ctx context.Context, storage storageman.IStorage, diskId string, disk storageman.IDisk, body jsonutils.JSONObject) (interface{}, error) {
	if disk != nil {
		hostutils.DelayTask(ctx, disk.Delete, compute.DiskDeleteInput{})
	} else {
		hostutils.DelayTask(ctx, nil, nil)
	}
	return nil, nil
}

func diskResize(ctx context.Context, storage storageman.IStorage, diskId string, disk storageman.IDisk, body jsonutils.JSONObject) (interface{}, error) {
	diskInfo, err := body.Get("disk")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("disk")
	}
	serverId, _ := diskInfo.GetString("server_id")
	if len(serverId) > 0 && guestman.GetGuestManager().Status(serverId) == "running" {
		sizeMb, _ := diskInfo.Int("size")
		return guestman.GetGuestManager().OnlineResizeDisk(ctx, serverId, diskId, sizeMb)
	} else {
		hostutils.DelayTask(ctx, disk.Resize, diskInfo)
		return nil, nil
	}
}

func diskSavePrepare(ctx context.Context, storage storageman.IStorage, diskId string, disk storageman.IDisk, body jsonutils.JSONObject) (interface{}, error) {
	diskInfo, err := body.Get("disk")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("disk")
	}
	hostutils.DelayTask(ctx, disk.PrepareSaveToGlance, diskInfo)
	return nil, nil
}

func diskReset(ctx context.Context, storage storageman.IStorage, diskId string, disk storageman.IDisk, body jsonutils.JSONObject) (interface{}, error) {
	snapshotId, err := body.GetString("snapshot_id")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("snapshot_id")
	}
	backingDiskId, _ := body.GetString("backing_disk_id")
	hostutils.DelayTask(ctx, disk.ResetFromSnapshot, &storageman.SDiskReset{
		SnapshotId:    snapshotId,
		BackingDiskId: backingDiskId,
		Input:         body,
	})
	return nil, nil
}

func diskSnapshot(ctx context.Context, storage storageman.IStorage, diskId string, disk storageman.IDisk, body jsonutils.JSONObject) (interface{}, error) {
	snapshotId, err := body.GetString("snapshot_id")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("snapshot_id")
	}
	hostutils.DelayTask(ctx, disk.DiskSnapshot, snapshotId)
	return nil, nil
}

func diskStorageBackup(ctx context.Context, storage storageman.IStorage, diskId string, disk storageman.IDisk, body jsonutils.JSONObject) (interface{}, error) {
	backupId, err := body.GetString("backup_id")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("backup_id")
	}
	backupStorageId, err := body.GetString("backup_storage_id")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("backup_storage_id")
	}
	backupStorageAccessInfo, err := body.Get("backup_storage_access_info")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("backup_storage_access_info")
	}
	hostutils.DelayTask(ctx, storage.StorageBackup, &storageman.SStorageBackup{
		BackupId:                backupId,
		BackupStorageId:         backupStorageId,
		BackupStorageAccessInfo: backupStorageAccessInfo.(*jsonutils.JSONDict),
	})
	return nil, nil
}

func diskStorageBackupRecovery(ctx context.Context, storage storageman.IStorage, diskId string, disk storageman.IDisk, body jsonutils.JSONObject) (interface{}, error) {
	backupId, err := body.GetString("backup_id")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("backup_id")
	}
	backupStorageId, err := body.GetString("backup_storage_id")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("backup_storage_id")
	}
	backupStorageAccessInfo, err := body.Get("backup_storage_access_info")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("backup_storage_access_info")
	}
	hostutils.DelayTask(ctx, storage.StorageBackupRecovery, storageman.SStorageBackup{
		BackupId:                backupId,
		BackupStorageId:         backupStorageId,
		BackupStorageAccessInfo: backupStorageAccessInfo.(*jsonutils.JSONDict),
	})
	return nil, nil
}

func diskBackup(ctx context.Context, storage storageman.IStorage, diskId string, disk storageman.IDisk, body jsonutils.JSONObject) (interface{}, error) {
	snapshotId, err := body.GetString("snapshot_id")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("snapshot_id")
	}
	backupId, err := body.GetString("backup_id")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("backup_id")
	}
	backupStorageId, err := body.GetString("backup_storage_id")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("backup_storage_id")
	}
	backupStorageAccessInfo, err := body.Get("backup_storage_access_info")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("backup_storage_access_info")
	}
	hostutils.DelayTask(ctx, disk.DiskBackup, &storageman.SDiskBakcup{
		BackupId:                backupId,
		SnapshotId:              snapshotId,
		BackupStorageId:         backupStorageId,
		BackupStorageAccessInfo: backupStorageAccessInfo.(*jsonutils.JSONDict),
	})
	return nil, nil
}

func diskDeleteSnapshot(ctx context.Context, storage storageman.IStorage, diskId string, disk storageman.IDisk, body jsonutils.JSONObject) (interface{}, error) {
	snapshotId, err := body.GetString("snapshot_id")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("snapshot_id")
	}
	hostutils.DelayTask(ctx, disk.DiskDeleteSnapshot, snapshotId)
	return nil, nil
}

func diskCleanupSnapshots(ctx context.Context, storage storageman.IStorage, diskId string, disk storageman.IDisk, body jsonutils.JSONObject) (interface{}, error) {
	convertSnapshots, err := body.GetArray("convert_snapshots")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("convert_snapshots")
	}
	deleteSnapshots, err := body.GetArray("delete_snapshots")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("delete_snapshots")
	}
	hostutils.DelayTask(ctx, disk.CleanupSnapshots, &storageman.SDiskCleanupSnapshots{
		ConvertSnapshots: convertSnapshots,
		DeleteSnapshots:  deleteSnapshots,
	})
	return nil, nil
}
