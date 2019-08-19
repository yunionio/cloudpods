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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
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
	}
}

func storageVerifyMountPoint(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, query, _ := appsrv.FetchEnv(ctx, w, r)
	mountPoint, err := query.GetString("mount_point")
	if err != nil {
		hostutils.Response(ctx, w, httperrors.NewMissingParameterError("mount_point"))
		return
	}
	output, err := procutils.NewCommand("mountpoint", mountPoint).Run()
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
	storage.SetStorageInfo(storageId, storageName, storageConf)
	resp, err := storage.SyncStorageInfo()
	if err != nil {
		return nil, err
	}
	storageman.GetManager().InitSharedStorageImageCache(storageType, storagecacheId, imagecachePath, storage)
	storageman.GetManager().Storages = append(storageman.GetManager().Storages, storage)
	return resp, nil
}

func storageDetach(ctx context.Context, body jsonutils.JSONObject) (interface{}, error) {
	mountPoint, err := body.GetString("mount_point")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("mount_point")
	}
	storage := storageman.GetManager().GetStorageByPath(mountPoint)

	name, _ := body.GetString("name")
	if storage == nil {
		return nil, httperrors.NewBadRequestError("ShareStorage[%s] Has detach from host ...", name)
	}
	storageman.GetManager().Remove(storage)
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
	storage.SetStorageInfo(storageId, storageName, storageConf)
	mountPoint, _ := ret.GetString("mount_point")
	storage.SetPath(mountPoint)
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
