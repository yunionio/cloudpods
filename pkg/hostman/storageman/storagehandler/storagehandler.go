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

package storagehandler

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	"yunion.io/x/onecloud/pkg/hostman/storageman/backupstorage"
	"yunion.io/x/onecloud/pkg/hostman/storageman/lvmutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

var (
	storageKeyWords    = []string{"storages"}
	storageActionFuncs = map[string]storageActionFunc{
		"attach": storageAttach,
		"detach": storageDetach,
		"update": storageUpdate,
	}
)

type storageActionFunc func(context.Context, jsonutils.JSONObject) (interface{}, error)

func AddStorageHandler(prefix string, app *appsrv.Application) {
	for _, keyWords := range storageKeyWords {
		app.AddHandler("POST",
			fmt.Sprintf("%s/%s/<action>", prefix, keyWords),
			auth.Authenticate(storageActions))
		app.AddHandler("POST",
			fmt.Sprintf("%s/%s/<storageId>/delete-snapshots", prefix, keyWords),
			auth.Authenticate(storageDeleteSnapshots))
		app.AddHandler("POST",
			fmt.Sprintf("%s/%s/<storageId>/snapshots-recycle", prefix, keyWords),
			auth.Authenticate(storageSnapshotsRecycle))
		app.AddHandler("GET",
			fmt.Sprintf("%s/%s/is-mount-point", prefix, keyWords),
			auth.Authenticate(storageVerifyMountPoint))
		app.AddHandler("GET",
			fmt.Sprintf("%s/%s/is-local-mount-point", prefix, keyWords),
			auth.Authenticate(storageIsLocalMountPoint))
		app.AddHandler("GET",
			fmt.Sprintf("%s/%s/is-vg-exist", prefix, keyWords),
			auth.Authenticate(storageIsVgExist))
		app.AddHandler("POST",
			fmt.Sprintf("%s/%s/delete-backup", prefix, keyWords),
			auth.Authenticate(storageDeleteBackup))
		app.AddHandler("POST",
			fmt.Sprintf("%s/%s/sync-backup", prefix, keyWords),
			auth.Authenticate(storageSyncBackup))
		app.AddHandler("POST",
			fmt.Sprintf("%s/%s/pack-instance-backup", prefix, keyWords),
			auth.Authenticate(storagePackInstanceBackup))
		app.AddHandler("POST",
			fmt.Sprintf("%s/%s/unpack-instance-backup", prefix, keyWords),
			auth.Authenticate(storageUnpackInstanceBackup))
		app.AddHandler("POST",
			fmt.Sprintf("%s/%s/sync-backup-storage", prefix, keyWords),
			auth.Authenticate(storageSyncBackupStorage))
	}
}

func storageIsLocalMountPoint(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, query, _ := appsrv.FetchEnv(ctx, w, r)
	mountPoint, err := query.GetString("mount_point")
	if err != nil {
		hostutils.Response(ctx, w, httperrors.NewMissingParameterError("mount_point"))
		return
	}
	fs, err := procutils.NewRemoteCommandAsFarAsPossible(
		"sh", "-c",
		fmt.Sprintf("df -T %s | awk 'NR==2{print $2}'", mountPoint),
	).Output()
	if err != nil {
		log.Errorf("failed get source of mountpoint %s: %s", mountPoint, err)
		hostutils.Response(ctx, w, httperrors.NewInternalServerError("failed get source of mountpoint %s: %s", mountPoint, err))
		return
	}
	fsStr := strings.TrimSpace(string(fs))
	log.Infof("check %s file system is %s", mountPoint, fsStr)
	if utils.IsInStringArray(fsStr, []string{"ext", "ext2", "ext3", "ext4", "xfs", "btrfs", "jfs", "reiserfs", "ntfs", "fat32", "exfat", "zfs"}) {
		// local file system
		appsrv.SendStruct(w, map[string]interface{}{"is_local_mount_point": true})
	} else {
		appsrv.SendStruct(w, map[string]interface{}{"is_local_mount_point": false})
	}
}

func storageIsVgExist(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, query, _ := appsrv.FetchEnv(ctx, w, r)
	vgName, err := query.GetString("vg_name")
	if err != nil {
		hostutils.Response(ctx, w, httperrors.NewMissingParameterError("vg_name"))
		return
	}
	if err := lvmutils.VgDisplay(vgName); err != nil {
		log.Errorf("vg %s display failed %s", vgName, err)
		hostutils.Response(ctx, w, httperrors.NewInternalServerError(err.Error()))
		return
	}
	hostutils.ResponseOk(ctx, w)
}

func storageVerifyMountPoint(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, query, _ := appsrv.FetchEnv(ctx, w, r)
	mountPoint, err := query.GetString("mount_point")
	if err != nil {
		hostutils.Response(ctx, w, httperrors.NewMissingParameterError("mount_point"))
		return
	}
	output, err := procutils.NewRemoteCommandAsFarAsPossible("mountpoint", mountPoint).Output()
	if err == nil {
		appsrv.SendStruct(w, map[string]interface{}{"is_mount_point": true})
	} else {
		appsrv.SendStruct(w, map[string]interface{}{
			"is_mount_point": false,
			"error":          string(output),
		})
	}
}

func storageActions(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, _, body := appsrv.FetchEnv(ctx, w, r)
	var action = params["<action>"]

	if f, ok := storageActionFuncs[action]; !ok {
		hostutils.Response(ctx, w, httperrors.NewNotFoundError("Not found"))
	} else {
		res, err := f(ctx, body)
		if err != nil {
			hostutils.Response(ctx, w, err)
		} else if res != nil {
			hostutils.Response(ctx, w, res)
		} else {
			hostutils.ResponseOk(ctx, w)
		}
	}
}

func storageAttach(ctx context.Context, body jsonutils.JSONObject) (interface{}, error) {
	mountPoint, err := body.GetString("mount_point")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("mount_point")
	}

	storageType, _ := body.GetString("storage_type")
	storage := storageman.GetManager().NewSharedStorageInstance(mountPoint, storageType)
	if storage == nil {
		return nil, httperrors.NewBadRequestError("'Not Support Storage[%s] mount_point: %s", storageType, mountPoint)
	}

	storagecacheId, _ := body.GetString("storagecache_id")
	imagecachePath, _ := body.GetString("imagecache_path")
	storageId, _ := body.GetString("storage_id")
	storageName, _ := body.GetString("name")
	storageConf, _ := body.Get("storage_conf")
	storage.SetStoragecacheId(storagecacheId)
	if err := storage.SetStorageInfo(storageId, storageName, storageConf); err != nil {
		return nil, err
	}
	/*err = storage.SyncStorageSize()
	if err != nil {
		return nil, errors.Wrapf(err, "SyncStorageSize")
	}*/
	resp, err := storage.SyncStorageInfo()
	if err != nil {
		return nil, err
	}
	storageman.GetManager().InitSharedStorageImageCache(storageType, storagecacheId, imagecachePath, storage)
	storageman.GetManager().Storages = append(storageman.GetManager().Storages, storage)
	return resp, nil
}

func storageDetach(ctx context.Context, body jsonutils.JSONObject) (interface{}, error) {
	info := struct {
		MountPoint string
		StorageId  string
		Name       string
	}{}
	err := body.Unmarshal(&info)
	if err != nil {
		return nil, errors.Wrapf(err, "body.Unmarshal")
	}
	if len(info.StorageId) == 0 {
		return nil, httperrors.NewMissingParameterError("storage_id")
	}
	storage := storageman.GetManager().GetStorage(info.StorageId)
	if storage != nil {
		if err := storage.Detach(); err != nil {
			log.Errorf("detach storage %s failed: %s", storage.GetPath(), err)
		}
		storageman.GetManager().Remove(storage)
	}
	return nil, nil
}

func storageUpdate(ctx context.Context, body jsonutils.JSONObject) (interface{}, error) {
	storageId, err := body.GetString("storage_id")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("storage_id")
	}
	storageConf, err := body.Get("storage_conf")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("storage_conf")
	}
	storage := storageman.GetManager().GetStorage(storageId)
	params := jsonutils.NewDict()
	params.Set("details", jsonutils.JSONTrue)
	ret, err := modules.Hoststorages.Get(hostutils.GetComputeSession(context.Background()),
		storageman.GetManager().GetHostId(), storageId, params)
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	if ret == nil || storage == nil {
		return nil, httperrors.NewNotFoundError("Storage %s not found", storageId)
	}
	storageName, _ := ret.GetString("storage")
	if err := storage.SetStorageInfo(storageId, storageName, storageConf); err != nil {
		return nil, err
	}
	mountPoint, _ := ret.GetString("mount_point")
	storage.SetPath(mountPoint)
	return nil, nil
}

func storageSyncBackup(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, _, body := appsrv.FetchEnv(ctx, w, r)
	backupId, err := body.GetString("backup_id")
	if err != nil {
		hostutils.Response(ctx, w, httperrors.NewMissingParameterError("backup_id"))
		return
	}
	backupStorageId, err := body.GetString("backup_storage_id")
	if err != nil {
		hostutils.Response(ctx, w, httperrors.NewMissingParameterError("backup_storage_id"))
		return
	}
	backupStorageAccessInfo, err := body.Get("backup_storage_access_info")
	if err != nil {
		hostutils.Response(ctx, w, httperrors.NewMissingParameterError("backup_storage_access_info"))
		return
	}
	backupStorage, err := backupstorage.GetBackupStorage(backupStorageId, backupStorageAccessInfo.(*jsonutils.JSONDict))
	if err != nil {
		hostutils.Response(ctx, w, err)
		return
	}
	exist, err := backupStorage.IsExists(backupId)
	if err != nil {
		hostutils.Response(ctx, w, err)
		return
	}
	var (
		ret    = jsonutils.NewDict()
		status string
	)
	if exist {
		status = compute.BACKUP_EXIST
	} else if !exist && err == nil {
		status = compute.BACKUP_NOT_EXIST
	} else {
		log.Errorf("fetch snapshot exist failed %s", err)
		status = compute.BACKUP_STATUS_UNKNOWN
	}
	ret.Set("status", jsonutils.NewString(status))
	hostutils.Response(ctx, w, ret)
}

func storageSyncBackupStorage(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, _, body := appsrv.FetchEnv(ctx, w, r)
	backupStorageId, err := body.GetString("backup_storage_id")
	if err != nil {
		hostutils.Response(ctx, w, httperrors.NewMissingParameterError("backup_storage_id"))
		return
	}
	backupStorageAccessInfo, err := body.Get("backup_storage_access_info")
	if err != nil {
		hostutils.Response(ctx, w, httperrors.NewMissingParameterError("backup_storage_access_info"))
		return
	}
	backupStorage, err := backupstorage.GetBackupStorage(backupStorageId, backupStorageAccessInfo.(*jsonutils.JSONDict))
	if err != nil {
		hostutils.Response(ctx, w, err)
		return
	}
	exist, reason, err := backupStorage.IsOnline()
	if err != nil {
		hostutils.Response(ctx, w, err)
		return
	}
	var (
		ret    = jsonutils.NewDict()
		status string
	)
	if exist {
		status = compute.BACKUPSTORAGE_STATUS_ONLINE
	} else {
		status = compute.BACKUPSTORAGE_STATUS_OFFLINE
	}
	ret.Set("status", jsonutils.NewString(status))
	ret.Set("reason", jsonutils.NewString(reason))
	hostutils.Response(ctx, w, ret)
}

func storagePackInstanceBackup(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, _, body := appsrv.FetchEnv(ctx, w, r)
	if !checkOptions(ctx, w, body, "package_name", "backup_ids", "backup_storage_id", "backup_storage_access_info", "metadata") {
		return
	}
	pb := storageman.SStoragePackInstanceBackup{}
	err := body.Unmarshal(&pb)
	if err != nil {
		hostutils.Response(ctx, w, httperrors.NewInputParameterError(err.Error()))
		return
	}

	hostutils.DelayTask(ctx, packInstanceBackup, &pb)
	hostutils.ResponseOk(ctx, w)
}

func storageUnpackInstanceBackup(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, _, body := appsrv.FetchEnv(ctx, w, r)
	if !checkOptions(ctx, w, body, "package_name", "backup_storage_id", "backup_storage_access_info") {
		return
	}
	pb := storageman.SStorageUnpackInstanceBackup{}
	err := body.Unmarshal(&pb)
	if err != nil {
		hostutils.Response(ctx, w, httperrors.NewInputParameterError(err.Error()))
		return
	}

	hostutils.DelayTask(ctx, unpackInstanceBackup, &pb)
	hostutils.ResponseOk(ctx, w)
}

func packInstanceBackup(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	sbParams := params.(*storageman.SStoragePackInstanceBackup)
	backupStorage, err := backupstorage.GetBackupStorage(sbParams.BackupStorageId, sbParams.BackupStorageAccessInfo)
	if err != nil {
		return nil, errors.Wrap(err, "GetBackupStorage")
	}
	packFileName, err := backupStorage.InstancePack(ctx, sbParams.PackageName, sbParams.BackupIds, &sbParams.Metadata)
	if err != nil {
		return nil, errors.Wrap(err, "InstancePack")
	}
	ret := jsonutils.NewDict()
	ret.Set("pack_file_name", jsonutils.NewString(packFileName))
	return ret, nil
}

func unpackInstanceBackup(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	sbParams := params.(*storageman.SStorageUnpackInstanceBackup)
	backupStorage, err := backupstorage.GetBackupStorage(sbParams.BackupStorageId, sbParams.BackupStorageAccessInfo)
	if err != nil {
		return nil, errors.Wrap(err, "GetBackupStorage")
	}
	metadataOnly := (sbParams.MetadataOnly != nil && *sbParams.MetadataOnly)
	diskBackupIds, metadata, err := backupStorage.InstanceUnpack(ctx, sbParams.PackageName, metadataOnly)
	if err != nil {
		return nil, errors.Wrap(err, "InstanceUnpack")
	}
	ret := jsonutils.NewDict()
	if diskBackupIds != nil {
		ret.Set("disk_backup_ids", jsonutils.Marshal(diskBackupIds))
	}
	ret.Set("metadata", jsonutils.Marshal(metadata))
	return ret, nil
}

func storageDeleteBackup(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, _, body := appsrv.FetchEnv(ctx, w, r)
	backupId, err := body.GetString("backup_id")
	if err != nil {
		hostutils.Response(ctx, w, httperrors.NewMissingParameterError("backup_id"))
		return
	}
	backupStorageId, err := body.GetString("backup_storage_id")
	if err != nil {
		hostutils.Response(ctx, w, httperrors.NewMissingParameterError("backup_storage_id"))
		return
	}
	backupStorageAccessInfo, err := body.Get("backup_storage_access_info")
	if err != nil {
		hostutils.Response(ctx, w, httperrors.NewMissingParameterError("backup_storage_access_info"))
		return
	}
	hostutils.DelayTask(ctx, deleteBackup, &storageman.SStorageBackup{
		BackupId:                backupId,
		BackupStorageId:         backupStorageId,
		BackupStorageAccessInfo: backupStorageAccessInfo.(*jsonutils.JSONDict),
	})
	hostutils.ResponseOk(ctx, w)
}

func checkOptions(ctx context.Context, w http.ResponseWriter, body jsonutils.JSONObject, options ...string) bool {
	for _, option := range options {
		if body.Contains(option) {
			continue
		}
		hostutils.Response(ctx, w, httperrors.NewMissingParameterError(option))
		return false
	}
	return true
}

func deleteBackup(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	sbParams := params.(*storageman.SStorageBackup)
	backupStorage, err := backupstorage.GetBackupStorage(sbParams.BackupStorageId, sbParams.BackupStorageAccessInfo)
	if err != nil {
		return nil, err
	}
	err = backupStorage.RemoveBackup(sbParams.BackupId)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func storageDeleteSnapshots(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, _, body := appsrv.FetchEnv(ctx, w, r)
	var storageId = params["<storageId>"]
	storage := storageman.GetManager().GetStorage(storageId)
	if storage == nil {
		hostutils.Response(ctx, w, httperrors.NewNotFoundError("Stroage Not found"))
		return
	}
	diskId, err := body.GetString("disk_id")
	if err != nil {
		hostutils.Response(ctx, w, httperrors.NewImageNotFoundError("disk_id"))
		return
	}
	hostutils.DelayTask(ctx, storage.DeleteSnapshots, diskId)
	hostutils.ResponseOk(ctx, w)
}

func storageSnapshotsRecycle(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, _, _ := appsrv.FetchEnv(ctx, w, r)
	var storageId = params["<storageId>"]
	storage := storageman.GetManager().GetStorage(storageId)
	if storage == nil {
		hostutils.Response(ctx, w, httperrors.NewNotFoundError("Stroage Not found"))
		return
	}
	go storageman.StorageRequestSnapshotRecycle(ctx, auth.AdminCredential(), storage)
	hostutils.ResponseOk(ctx, w)
}
