package storageman

import (
	"context"
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
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

		// TODO
		// app.AddHandler("POST",
		// 	fmt.Sprintf("%s/%s/<storageId>/delete-snapshots", prefix, keyWords),
		// 	auth.Authenticate(storageDeleteSnapshots))
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
	storage := storageManager.NewSharedStorageInstance(mountPoint, storageType)
	if storage == nil {
		return nil, httperrors.NewBadRequestError("'Not Support Storage[%s] mount_point: %s", storageType, mountPoint)
	}

	storagecacheId, _ := body.GetString("storagecache_id")
	imagecachePath, _ := body.GetString("imagecache_path")
	storageManager.InitSharedStorageImageCache(storageType, storagecacheId, imagecachePath, storage)

	storageId, _ := body.GetString("id")
	storageName, _ := body.GetString("name")
	storageConf, _ := body.Get("storage_conf")
	storage.SetStorageInfo(storageId, storageName, storageConf)
	return nil, nil
}

func storageDetach(ctx context.Context, body jsonutils.JSONObject) (interface{}, error) {
	mountPoint, err := body.GetString("mount_point")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("mount_point")
	}
	storage := storageManager.GetStorageByPath(mountPoint)

	name, _ := body.GetString("name")
	if storage == nil {
		return nil, httperrors.NewBadRequestError("ShareStorage[%s] Has detach from host ...", name)
	}
	storageManager.Remove(storage)
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
	storage := storageManager.GetStorage(storageId)
	ret, err := modules.Hoststorages.Get(hostutils.GetComputeSession(context.Background()),
		storageManager.GetHostId(), storageId,
		jsonutils.NewDict(jsonutils.NewPair("details", jsonutils.JSONTrue)))
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
