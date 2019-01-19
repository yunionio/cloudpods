package downloadhandler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
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
	}
}

func customizeHandlerInfo(info *appsrv.SHandlerInfo) {
	switch info.GetName(nil) {
	case "disk_download", "download", "snapshot_download":
		info.SetProcessTimeout(time.Minute * 30).SetWorkerManager(streamingWorkerMan)
	}
}

func download(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var (
		params, _, _ = appsrv.FetchEnv(ctx, w, r)
		sid          = params["<id>"]
		action       = params["<action>"]
		zlib         = r.Header.Get("X-Compress-Content")
		compress     bool
	)

	if zlib == "zlib" {
		compress = true
	}

	switch action {
	case "images":
		// ImagecacheDownloadProvider()
	case "servers":
	default:
		hostutils.Response(ctx, w, httperrors.NewNotFoundError("%s Not found", action))
	}
}

func diskDownload(ctx context.Context, w http.ResponseWriter, r *http.Request) {
}

func snapshotDownload(ctx context.Context, w http.ResponseWriter, r *http.Request) {
}

func diskHead(ctx context.Context, w http.ResponseWriter, r *http.Request) {
}

func snapshotHead(ctx context.Context, w http.ResponseWriter, r *http.Request) {
}
