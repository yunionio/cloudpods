package storageman

import (
	"context"
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/regutils"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

var keyWords = []string{"disks"}

type actionFunc func(context.Context, IStorage, string, IDisk, jsonutils.JSONObject) (interface{}, error)

func AddDiskHandler(prefix string, app *appsrv.Application) {
	for _, keyWord := range keyWords {
		for _, seg := range []string{"iso_cache", "image_cache"} {
			app.AddHandler("POST",
				fmt.Sprintf("%s/%s/%s", prefix, keyWord, seg),
				auth.Authenticate(perfetchImageCache))

			app.AddHandler("GET",
				fmt.Sprintf("%s/%s/%s", prefix, keyWord, seg),
				auth.Authenticate(deleteImageCache))
		}

		// excuse me ？
		app.AddHandler("POST",
			fmt.Sprintf("%s/%s/<storageId>/upload", prefix, keyWord),
			auth.Authenticate(saveToGlance))

		app.AddHandler("POST",
			fmt.Sprintf("%s/%s/<storageId>/<action>/<diskId>", prefix, keyWord),
			auth.Authenticate(perfomrDiskActions))
	}
}

func perfomrDiskActions(ctx context.Context, w http.ResponseWriter, r *http.Request) {
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
	}
	storage := storageManager.GetStorage(storageId)
	if storage == nil {
		hostutils.Response(ctx, w, httperrors.NewNotFoundError("Storage %s not found", storageId))
	}
	disk := storage.GetDiskById(diskId)

	if f, ok := actionFuncs[action]; !ok {
		hostutils.Response(ctx, w, httperrors.NewNotFoundError("Not found"))
	} else {
		res, err := f(ctx, storage, diskId, disk, body)
		if err != nil {
			hostutils.Response(ctx, w, err)
		} else if res != nil {
			hostutils.Response(ctx, w, res)
		} else {
			hostutils.ResponseOk(ctx, w)
		}
	}
}

func diskCreate(ctx context.Context, storage IStorage, diskId string, disk IDisk, body jsonutils.JSONObject) (interface{}, error) {
	diskInfo, err := body.Get("disk")
	if err != nil {
		return nil, httperrors.NewInputParameterError("Missing disk")
	}
	hostutils.DelayTask(ctx, storage.CreateDiskByDiskinfo,
		&SDiskCreateByDiskinfo{diskId, disk, diskInfo, storage})
	return nil, nil
}

func diskDelete(ctx context.Context, storage IStorage, diskId string, disk IDisk, body jsonutils.JSONObject) (interface{}, error) {
	hostutils.DelayTask(ctx, disk.Delete, nil)
	return nil, nil
}

func diskResize(ctx context.Context, storage IStorage, diskId string, disk IDisk, body jsonutils.JSONObject) (interface{}, error) {
	diskInfo, err := body.Get("disk")
	if err != nil {
		return nil, httperrors.NewInputParameterError("Missing disk")
	}
	hostutils.DelayTask(ctx, disk.Resize, diskInfo)
	return nil, nil
}

var actionFuncs = map[string]actionFunc{
	"create": diskCreate,
	"delete": diskDelete,
	"resize": diskResize,
}
