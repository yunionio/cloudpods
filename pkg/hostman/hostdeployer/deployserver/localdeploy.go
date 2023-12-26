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

package deployserver

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/sirupsen/logrus"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"

	commonconsts "yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/hostman/diskutils"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/consts"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
	"yunion.io/x/onecloud/pkg/util/winutils"
)

type LocalDeploy struct{}

func (d *LocalDeploy) DeployGuestFs(req *deployapi.DeployParams) (res *deployapi.DeployGuestFsResponse, err error) {
	localDisk, err := diskutils.NewKVMGuestDisk(qemuimg.SImageInfo{}, consts.DEPLOY_DRIVER_LOCAL_DISK, false)
	if err != nil {
		return nil, errors.Wrap(err, "new local disk")
	}
	if err := localDisk.Connect(req.GuestDesc); err != nil {
		return nil, errors.Wrapf(err, "local disk connect")
	}
	defer localDisk.Disconnect()
	return localDisk.DeployGuestfs(req)
}

func (d *LocalDeploy) ResizeFs(req *deployapi.ResizeFsParams) (res *deployapi.Empty, err error) {
	localDisk, err := diskutils.NewKVMGuestDisk(qemuimg.SImageInfo{}, consts.DEPLOY_DRIVER_LOCAL_DISK, false)
	if err != nil {
		return nil, errors.Wrap(err, "new local disk")
	}
	if err := localDisk.Connect(nil); err != nil {
		return nil, errors.Wrapf(err, "local disk connect")
	}
	defer localDisk.Disconnect()
	return localDisk.ResizeFs()
}

func (d *LocalDeploy) FormatFs(req *deployapi.FormatFsParams) (*deployapi.Empty, error) {
	localDisk, err := diskutils.NewKVMGuestDisk(qemuimg.SImageInfo{}, consts.DEPLOY_DRIVER_LOCAL_DISK, false)
	if err != nil {
		return nil, errors.Wrap(err, "new local disk")
	}
	if err := localDisk.Connect(nil); err != nil {
		return nil, errors.Wrapf(err, "local disk connect")
	}
	defer localDisk.Disconnect()
	return localDisk.FormatFs(req)
}

func (d *LocalDeploy) SaveToGlance(req *deployapi.SaveToGlanceParams) (*deployapi.SaveToGlanceResponse, error) {
	localDisk, err := diskutils.NewKVMGuestDisk(qemuimg.SImageInfo{}, consts.DEPLOY_DRIVER_LOCAL_DISK, false)
	if err != nil {
		return nil, errors.Wrap(err, "new local disk")
	}
	if err := localDisk.Connect(nil); err != nil {
		return nil, errors.Wrapf(err, "local disk connect")
	}
	defer localDisk.Disconnect()
	return localDisk.SaveToGlance(req)
}

func (d *LocalDeploy) ProbeImageInfo(req *deployapi.ProbeImageInfoPramas) (*deployapi.ImageInfo, error) {
	localDisk, err := diskutils.NewKVMGuestDisk(qemuimg.SImageInfo{}, consts.DEPLOY_DRIVER_LOCAL_DISK, false)
	if err != nil {
		return nil, errors.Wrap(err, "new local disk")
	}
	if err := localDisk.Connect(nil); err != nil {
		return nil, errors.Wrapf(err, "local disk connect")
	}
	defer localDisk.Disconnect()
	return localDisk.ProbeImageInfo(req)
}

func LocalInitEnv() error {
	f, _ := os.OpenFile("/log", os.O_CREATE|os.O_WRONLY, 0777)
	logrus.SetOutput(f)

	// /bin:/sbin:/usr/bin:/usr/sbin
	var paths = []string{
		"/bin",
		"/sbin",
		"/usr/bin",
		"/usr/sbin",
		"/opt/yunion/bin", // for zerofree command
	}
	err := os.Setenv("PATH", strings.Join(paths, ":"))
	if err != nil {
		return errors.Wrap(err, "set env path")
	}

	netutils.SetPrivatePrefixes(DeployOption.CustomizedPrivatePrefixes)
	if err := fsdriver.Init(DeployOption.CloudrootDir); err != nil {
		return errors.Wrap(err, "init fsdriver")
	}
	commonconsts.SetAllowVmSELinux(DeployOption.AllowVmSELinux)
	winutils.SetChntpwPath("/opt/yunion/bin/chntpw.static")
	return nil
}

func StartLocalDeploy(deployAction string) (interface{}, error) {
	localDeployer := LocalDeploy{}
	switch deployAction {
	case "deploy_guest_fs":
		params := new(deployapi.DeployParams)
		if err := json.Unmarshal([]byte(DeployOption.DeployParams), params); err != nil {
			return nil, errors.Wrap(err, "unmarshal params")
		}
		return localDeployer.DeployGuestFs(params)
	case "resize_fs":
		return localDeployer.ResizeFs(nil)
	case "format_fs":
		params := new(deployapi.FormatFsParams)
		if err := json.Unmarshal([]byte(DeployOption.DeployParams), params); err != nil {
			return nil, errors.Wrap(err, "unmarshal params")
		}
		return localDeployer.FormatFs(params)
	case "save_to_glance":
		params := new(deployapi.SaveToGlanceParams)
		if err := json.Unmarshal([]byte(DeployOption.DeployParams), params); err != nil {
			return nil, errors.Wrap(err, "unmarshal params")
		}
		return localDeployer.SaveToGlance(params)
	case "probe_image_info":
		params := new(deployapi.ProbeImageInfoPramas)
		if err := json.Unmarshal([]byte(DeployOption.DeployParams), params); err != nil {
			return nil, errors.Wrap(err, "unmarshal params")
		}
		return localDeployer.ProbeImageInfo(params)
	default:
		return nil, errors.Errorf("unknown deploy action")
	}
}
