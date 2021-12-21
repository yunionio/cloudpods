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

package hostimage

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/pierrec/lz4/v4"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/appsrv"
	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

type SHostImageOptions struct {
	common_options.CommonOptions
	LocalImagePath    []string `help:"Local Image Paths"`
	SnapshotDirSuffix string   `help:"Snapshot dir name equal diskId concat snapshot dir suffix" default:"_snap"`
	CommonConfigFile  string   `help:"common config file for container"`
	StreamChunkSize   int      `help:"Download stream chunk size KB" default:"4096"`
	Lz4ChecksumOff    bool     `help:"Turn off lz4 checksum option"`
}

var (
	HostImageOptions   SHostImageOptions
	streamingWorkerMan *appsrv.SWorkerManager
)

func init() {
	streamingWorkerMan = appsrv.NewWorkerManager("streaming_worker", 20, 1024, false)
}

func StartService() {
	consts.SetServiceType("host-image")
	common_options.ParseOptions(&HostImageOptions, os.Args, "host.conf", "host-image")
	if len(HostImageOptions.CommonConfigFile) > 0 {
		baseOpt := HostImageOptions.BaseOptions.BaseOptions
		commonCfg := new(common_options.CommonOptions)
		commonCfg.Config = HostImageOptions.CommonConfigFile
		common_options.ParseOptions(commonCfg, []string{"host"}, "common.conf", "host")
		HostImageOptions.CommonOptions = *commonCfg
		HostImageOptions.BaseOptions.BaseOptions = baseOpt
	}
	HostImageOptions.EnableSsl = false
	HostImageOptions.Port += 40000
	app_common.InitAuth(&HostImageOptions.CommonOptions, func() {
		log.Infof("Auth complete!!")
	})
	app := app_common.InitApp(&HostImageOptions.BaseOptions, false)
	initHandlers(app, "")
	app_common.ServeForever(app, &HostImageOptions.BaseOptions)
}

func initHandlers(app *appsrv.Application, prefix string) {
	app.AddHandler("GET", fmt.Sprintf("%s/disks/<sid>", prefix), auth.Authenticate(getImage)).
		SetProcessNoTimeout().SetWorkerManager(streamingWorkerMan)
	app.AddHandler("GET", fmt.Sprintf("%s/snapshots/<diskId>/<sid>", prefix), auth.Authenticate(getImage)).
		SetProcessNoTimeout().SetWorkerManager(streamingWorkerMan)

	app.AddHandler("HEAD", fmt.Sprintf("%s/disks/<sid>", prefix), auth.Authenticate(getImageMeta))
	app.AddHandler("HEAD", fmt.Sprintf("%s/snapshots/<diskId>/<sid>", prefix), auth.Authenticate(getImageMeta))
	app.AddHandler("POST", fmt.Sprintf("%s/disks/<sid>", prefix), auth.Authenticate(closeImage))
	app.AddHandler("POST", fmt.Sprintf("%s/snapshots/<diskId>/<sid>", prefix), auth.Authenticate(closeImage))
}

func getDiskPath(diskId string) string {
	for _, imagePath := range HostImageOptions.LocalImagePath {
		diskPath := path.Join(imagePath, diskId)
		if _, err := os.Stat(diskPath); !os.IsNotExist(err) {
			return diskPath
		}
	}
	return ""
}

func getSnapshotPath(diskId, snapshotId string) string {
	for _, imagePath := range HostImageOptions.LocalImagePath {
		diskPath := path.Join(imagePath, "snapshots",
			diskId+HostImageOptions.SnapshotDirSuffix, snapshotId)
		if _, err := os.Stat(diskPath); !os.IsNotExist(err) {
			return diskPath
		}
	}
	return ""
}

func inputCheck(ctx context.Context) (string, error) {
	var userCred = auth.FetchUserCredential(ctx, nil)
	if !userCred.HasSystemAdminPrivilege() {
		return "", httperrors.NewForbiddenError("System admin only")
	}

	var params = appctx.AppContextParams(ctx)
	var sid = params["<sid>"]
	var imagePath string
	if diskId, ok := params["<diskId>"]; ok {
		imagePath = getSnapshotPath(diskId, sid)
	} else {
		imagePath = getDiskPath(sid)
	}
	if len(imagePath) == 0 {
		return "", httperrors.NewNotFoundError("Disk not found")
	}
	return imagePath, nil
}

func parseRange(reqRange string) (int64, int64, error) {
	if !strings.HasPrefix(reqRange, "bytes=") {
		return 0, 0, httperrors.NewInputParameterError("Invalid range header")
	}
	reqRange = reqRange[len("bytes="):]
	ranges := strings.Split(reqRange, "-")
	if len(ranges) != 2 {
		return 0, 0, httperrors.NewInputParameterError("Invalid range header")
	}
	startPos, err := strconv.ParseInt(ranges[0], 10, 0)
	if err != nil {
		return 0, 0, httperrors.NewInputParameterError("Invalid range header")
	}
	endPos, err := strconv.ParseInt(ranges[1], 10, 0)
	if err != nil {
		return 0, 0, httperrors.NewInputParameterError("Invalid range header")
	}
	return startPos, endPos, nil
}

func closeImage(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	imagePath, err := inputCheck(ctx)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}

	var f IImage
	if r.Header.Get("X-Read-File") == "true" {
		f = &SFile{}
	} else {
		f = &SQcow2Image{}
	}
	err = f.Load(imagePath, true)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	f.Close()
	w.WriteHeader(http.StatusOK)
}

func getImage(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	imagePath, err := inputCheck(ctx)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}

	var f IImage
	var startPos, endPos int64
	var rateLimit int64 = -1

	if r.Header.Get("X-Read-File") == "true" {
		f = &SFile{}
	} else {
		f = &SQcow2Image{}
	}
	if err = f.Open(imagePath, true); err != nil {
		log.Errorf("Open image error: %s", err)
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	defer f.Close()

	endPos = f.Length() - 1
	reqRange := r.Header.Get("Range")
	if len(reqRange) > 0 {
		startPos, endPos, err = parseRange(reqRange)
		if err != nil {
			log.Errorf("Parse range error: %s", err)
			httperrors.GeneralServerError(ctx, w, err)
			return
		}
	}

	strRateLimit := r.Header.Get("X-Rate-Limit-Mbps")
	if len(strRateLimit) > 0 {
		rateLimit, err = strconv.ParseInt(strRateLimit, 10, 0)
		if err != nil {
			log.Errorf("Parse ratelimit error: %s", err)
			httperrors.InvalidInputError(ctx, w, "Invaild rate limit header")
			return
		}
	}

	streamHeader(w, f, startPos, endPos)
	startStream(w, f, startPos, endPos, rateLimit)
}

func streamHeader(w http.ResponseWriter, f IImage, startPos, endPos int64) {
	var statusCode = http.StatusOK
	w.Header().Set("Content-Type", "application/octet-stream")
	if startPos > 0 || endPos < f.Length()-1 {
		statusCode = http.StatusPartialContent
		w.Header().Set("Content-Range",
			fmt.Sprintf("bytes %d-%d/%d", startPos, endPos, f.Length()))
	}
	w.WriteHeader(statusCode)
}

func startStream(w http.ResponseWriter, f IImage, startPos, endPos, rateLimit int64) {
	var CHUNK_SIZE int64 = int64(HostImageOptions.StreamChunkSize * 1024)
	var readSize int64 = CHUNK_SIZE
	var sendBytes int64
	var lz4Writer = lz4.NewWriter(w)
	var startTime = time.Now()

	opts := []lz4.Option{
		lz4.BlockSizeOption(lz4.Block4Mb),
		lz4.ConcurrencyOption(-1),
		lz4.CompressionLevelOption(lz4.Fast),
	}
	if HostImageOptions.Lz4ChecksumOff {
		log.Infof("Turn off lz4 checksum")
		opts = append(opts,
			lz4.BlockChecksumOption(false),
			lz4.ChecksumOption(false),
		)
	}
	if err := lz4Writer.Apply(opts...); err != nil {
		log.Errorf("lz4Writer.Apply options error: %v", err)
		goto fail
	}

	for startPos < endPos {
		if endPos-startPos < CHUNK_SIZE {
			readSize = endPos - startPos + 1
		}
		buf, err := f.Read(startPos, readSize)
		if err != nil {
			log.Errorf("Read image error: %s", err)
			goto fail
		}
		startPos += readSize
		wSize, err := lz4Writer.Write(buf)
		if err != nil {
			log.Errorf("lz4Write error: %s", err)
			goto fail
		}
		sendBytes += int64(wSize)
		if rateLimit > 0 {
			tmDelta := time.Now().Sub(startTime)
			tms := tmDelta.Seconds()
			vtmDelta := float64(sendBytes*8) / float64(1024.0*1024.0*rateLimit)
			if vtmDelta > tms {
				time.Sleep(time.Duration(vtmDelta - tms))
			}
		}
	}

fail:
	if err := lz4Writer.Close(); err != nil {
		log.Errorf("lz4 Close error: %s", err)
	}
}

func getImageMeta(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	imagePath, err := inputCheck(ctx)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}

	var f IImage
	if r.Header.Get("X-Read-File") == "true" {
		f = &SFile{}
	} else {
		f = &SQcow2Image{}
	}
	if err = f.Open(imagePath, true); err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}

	w.Header().Set("Content-Length", fmt.Sprintf("%d", f.Length()))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Accept-Ranges", "bytes")
	w.WriteHeader(200)
}
