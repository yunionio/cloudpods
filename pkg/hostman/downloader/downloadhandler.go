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

package downloader

import (
	"context"
	"fmt"
	"net/http"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

var (
	keyWords           = []string{"download"}
	streamingWorkerMan *appsrv.SWorkerManager
)

func init() {
	streamingWorkerMan = appsrv.NewWorkerManager("streaming_worker", 20, 1024, false)
}

func AddDownloadHandler(prefix string, app *appsrv.Application) {
	for _, kerword := range keyWords {
		hi := app.AddHandler2("GET", fmt.Sprintf("%s/%s/<action>/<id>", prefix, kerword),
			auth.Authenticate(download), nil, "download", nil)
		customizeHandlerInfo(hi)

		hi = app.AddHandler2("GET", fmt.Sprintf("%s/%s/disks/<storageId>/<diskId>",
			prefix, kerword), auth.Authenticate(diskDownload), nil, "disk_download", nil)
		customizeHandlerInfo(hi)

		hi = app.AddHandler2("GET", fmt.Sprintf(
			"%s/%s/snapshots/<storageId>/<diskId>/<snapshotId>",
			prefix, kerword), auth.Authenticate(snapshotDownload),
			nil, "snapshot_download", nil)
		customizeHandlerInfo(hi)

		app.AddHandler("HEAD", fmt.Sprintf("%s/%s/disks/<storageId>/<diskId>",
			prefix, kerword), auth.Authenticate(diskHead))
		app.AddHandler("HEAD",
			fmt.Sprintf("%s/%s/snapshots/<storageId>/<diskId>/<snapshotId>",
				prefix, kerword), auth.Authenticate(snapshotHead))
		app.AddHandler("HEAD",
			fmt.Sprintf("%s/%s/images/<id>", prefix, kerword), auth.Authenticate(imageCacheHead))
	}
}

func customizeHandlerInfo(info *appsrv.SHandlerInfo) {
	switch info.GetName(nil) {
	case "disk_download", "download", "snapshot_download":
		info.SetProcessNoTimeout().SetWorkerManager(streamingWorkerMan)
	}
}

func isCompress(r *http.Request) bool {
	return r.Header.Get("X-Compress-Content") == "zlib"
}

func download(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var (
		params, _, _ = appsrv.FetchEnv(ctx, w, r)
		id           = params["<id>"]
		action       = params["<action>"]
		rateLimit    = options.HostOptions.BandwidthLimit
		compress     = isCompress(r)
	)

	switch action {
	case "images":
		hand := NewImageCacheDownloadProvider(w, compress, rateLimit, id)
		if !fileutils2.Exists(hand.downloadFilePath()) {
			httperrors.NotFoundError(ctx, w, "Image cache %s not found", id)
		} else {
			if err := hand.Start(); err != nil {
				hostutils.Response(ctx, w, err)
			}
		}
	case "servers":
		hand := NewGuestDownloadProvider(w, compress, rateLimit, id)
		if !fileutils2.Exists(hand.fullPath()) {
			httperrors.NotFoundError(ctx, w, "Guest %s not found", id)
		} else {
			if err := hand.Start(); err != nil {
				hostutils.Response(ctx, w, err)
			}
		}
	default:
		hostutils.Response(ctx, w, httperrors.NewNotFoundError("%s Not found", action))
	}
}

func diskPrecheck(
	ctx context.Context, w http.ResponseWriter, r *http.Request,
) (storageman.IDisk, error) {
	var (
		params, _, _ = appsrv.FetchEnv(ctx, w, r)
		storageId    = params["<storageId>"]
		diskId       = params["<diskId>"]
	)
	storage := storageman.GetManager().GetStorage(storageId)
	if storage == nil {
		return nil, httperrors.NewNotFoundError("Storage %s not found", storageId)
	}
	disk, err := storage.GetDiskById(diskId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetDiskById(%s)", diskId)
	}
	return disk, nil
}

func diskDownload(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	disk, err := diskPrecheck(ctx, w, r)
	if err != nil {
		hostutils.Response(ctx, w, err)
	} else {
		var compress = isCompress(r)
		hand := NewImageDownloadProvider(w,
			compress, options.HostOptions.BandwidthLimit, disk, "")
		if err := hand.Start(); err != nil {
			hostutils.Response(ctx, w, err)
		}
	}
}

func diskHead(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	disk, err := diskPrecheck(ctx, w, r)
	if err != nil {
		hostutils.Response(ctx, w, err)
	} else {
		var compress = isCompress(r)
		hand := NewImageDownloadProvider(w,
			compress, options.HostOptions.BandwidthLimit, disk, "")
		if err := hand.HandlerHead(); err != nil {
			hostutils.Response(ctx, w, err)
		}
	}
}

func snapshotPrecheck(
	ctx context.Context, w http.ResponseWriter, r *http.Request,
) (string, error) {
	var (
		params, _, _ = appsrv.FetchEnv(ctx, w, r)
		storageId    = params["<storageId>"]
		diskId       = params["<diskId>"]
		snapshotId   = params["<snapshotId>"]
	)

	storage := storageman.GetManager().GetStorage(storageId)
	if storage == nil {
		return "", httperrors.NewNotFoundError("Storage %s not found", storageId)
	}
	return storage.GetSnapshotPathByIds(diskId, snapshotId), nil
}

func snapshotDownload(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	snapshotPath, err := snapshotPrecheck(ctx, w, r)
	if err != nil {
		hostutils.Response(ctx, w, err)
	} else {
		var compress = isCompress(r)
		hand := NewSnapshotDownloadProvider(w,
			compress, options.HostOptions.BandwidthLimit, snapshotPath)
		if err := hand.Start(); err != nil {
			hostutils.Response(ctx, w, err)
		}
	}
}

func snapshotHead(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	snapshotPath, err := snapshotPrecheck(ctx, w, r)
	if err != nil {
		hostutils.Response(ctx, w, err)
	} else {
		var compress = isCompress(r)
		hand := NewSnapshotDownloadProvider(w,
			compress, options.HostOptions.BandwidthLimit, snapshotPath)
		if err := hand.HandlerHead(); err != nil {
			hostutils.Response(ctx, w, err)
		}
	}
}

func imageCacheHead(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, _, _ := appsrv.FetchEnv(ctx, w, r)
	imageId := params["<id>"]
	rateLimit := options.HostOptions.BandwidthLimit
	compress := isCompress(r)

	hand := NewImageCacheDownloadProvider(w, compress, rateLimit, imageId)

	if err := hand.HandlerHead(); err != nil {
		hostutils.Response(ctx, w, err)
	}
}
