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

	execlient "yunion.io/x/executor/client"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

type SHostImageOptions struct {
	common_options.HostCommonOptions
	LocalImagePath     []string `help:"Local Image Paths"`
	LVMVolumeGroups    []string `help:"LVM Volume Groups(vgs)"`
	SnapshotDirSuffix  string   `help:"Snapshot dir name equal diskId concat snapshot dir suffix" default:"_snap"`
	HostImageNbdPidDir string   `help:"Host-image nbd pid files dir " default:"/var/run/onecloud/host-image"`
	CommonConfigFile   string   `help:"common config file for container"`
}

var (
	HostImageOptions SHostImageOptions
	nbdExportManager *SNbdExportManager
)

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
	log.Infof("exec socket path: %s", HostImageOptions.ExecutorSocketPath)
	if HostImageOptions.EnableRemoteExecutor {
		execlient.Init(HostImageOptions.ExecutorSocketPath)
		execlient.SetTimeoutSeconds(HostImageOptions.ExecutorConnectTimeoutSeconds)
		procutils.SetRemoteExecutor()
	}

	nbdExportManager = NewNbdExportManager()
	output, err := procutils.NewCommand("mkdir", "-p", HostImageOptions.HostImageNbdPidDir).Output()
	if err != nil {
		log.Fatalf("failed to create path %s: %s %s", HostImageOptions.HostImageNbdPidDir, output, err)
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
	app.AddHandler("POST", fmt.Sprintf("%s/disks/<sid>/nbd-export", prefix), auth.Authenticate(imageNbdExport))
	app.AddHandler("POST", fmt.Sprintf("%s/snapshots/<diskId>/<sid>/nbd-export", prefix), auth.Authenticate(imageNbdExport))

	app.AddHandler("POST", fmt.Sprintf("%s/disks/<sid>/nbd-close", prefix), auth.Authenticate(imageNbdClose))
	app.AddHandler("POST", fmt.Sprintf("%s/snapshots/<diskId>/<sid>/nbd-close", prefix), auth.Authenticate(imageNbdClose))
}

func getDiskPath(diskId string) string {
	for _, imagePath := range HostImageOptions.LocalImagePath {
		diskPath := path.Join(imagePath, diskId)
		if _, err := procutils.RemoteStat(diskPath); err == nil {
			return diskPath
		}
	}
	for _, vg := range HostImageOptions.LVMVolumeGroups {
		diskPath := path.Join("/dev", vg, diskId)
		if _, err := procutils.RemoteStat(diskPath); err == nil {
			return diskPath
		}
	}
	return ""
}

func getSnapshotPath(diskId, snapshotId string) string {
	for _, imagePath := range HostImageOptions.LocalImagePath {
		diskPath := path.Join(imagePath, "snapshots",
			diskId+HostImageOptions.SnapshotDirSuffix, snapshotId)
		if _, err := procutils.RemoteStat(diskPath); err == nil {
			return diskPath
		}
	}
	for _, vg := range HostImageOptions.LVMVolumeGroups {
		diskPath := path.Join("/dev", vg, "snap_"+snapshotId)
		if _, err := procutils.RemoteStat(diskPath); err == nil {
			return diskPath
		}
	}
	return ""
}

func inputCheck(ctx context.Context, w http.ResponseWriter, r *http.Request) (string, string, error) {
	params, _, body := appsrv.FetchEnv(ctx, w, r)
	var sid = params["<sid>"]
	var imagePath string
	var remoteDiskId string

	remoteDiskId, _ = body.GetString("disk_id")
	if remoteDiskId == "" {
		return "", "", httperrors.NewMissingParameterError("disk_id")
	}

	if diskId, ok := params["<diskId>"]; ok {
		imagePath = getSnapshotPath(diskId, sid)
	} else {
		imagePath = getDiskPath(sid)
	}
	if len(imagePath) == 0 {
		return "", "", httperrors.NewNotFoundError("Disk not found")
	}
	return imagePath, remoteDiskId, nil
}

func imageNbdExport(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	imagePath, targetDiskId, err := inputCheck(ctx, w, r)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	imageInfo := qemuimg.SImageInfo{
		Path: imagePath,
	}
	encryptKey := r.Header.Get("X-Encrypt-Key")
	if len(encryptKey) > 0 {
		imageInfo.Password = encryptKey
		imageInfo.EncryptAlg = seclib2.TSymEncAlg(r.Header.Get("X-Encrypt-Alg"))
	}

	nbdPort, err := nbdExportManager.QemuNbdStartExport(imageInfo, targetDiskId)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	log.Infof("Image %s request nbd export with port %d", targetDiskId, nbdPort)

	ret := jsonutils.NewDict()
	ret.Set("nbd_port", jsonutils.NewInt(int64(nbdPort)))
	appsrv.SendJSON(w, ret)
}

func imageNbdClose(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, targetDiskId, err := inputCheck(ctx, w, r)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	err = nbdExportManager.QemuNbdCloseExport(targetDiskId)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	log.Infof("Image %s request nbd close export with port", targetDiskId)
	appsrv.SendStruct(w, map[string]string{"result": "ok"})
}
