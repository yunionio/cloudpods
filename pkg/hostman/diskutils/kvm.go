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

package diskutils

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	cloudconsts "yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/hostman/diskutils/deploy_iface"
	"yunion.io/x/onecloud/pkg/hostman/diskutils/libguestfs"
	"yunion.io/x/onecloud/pkg/hostman/diskutils/nbd"
	"yunion.io/x/onecloud/pkg/hostman/diskutils/qemu_kvm"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/consts"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
)

type SKVMGuestDisk struct {
	readOnly     bool
	kvmImagePath string
	topImagePath string
	deployer     deploy_iface.IDeployer
}

func NewKVMGuestDisk(imageInfo qemuimg.SImageInfo, driver string, readOnly bool) (*SKVMGuestDisk, error) {
	originImage := imageInfo.Path
	imagePath := imageInfo.Path
	if readOnly {
		// if readonly, create a top image over the original image, open device as RW
		tmpFileDir, err := ioutil.TempDir(cloudconsts.DeployTempDir(), "kvm_disks")
		if err != nil {
			log.Errorf("fail to obtain tempFile for readonly kvm disk: %s", err)
			return nil, errors.Wrap(err, "ioutil.TempDir")
		}
		tmpFileName := filepath.Join(tmpFileDir, "disk")
		img, err := qemuimg.NewQemuImage(tmpFileName)
		if err != nil {
			log.Errorf("fail to init qemu image %s", tmpFileName)
			return nil, errors.Wrap(err, "NewQemuImage")
		}
		err = img.CreateQcow2(0, false, imageInfo.Path, imageInfo.Password, imageInfo.EncryptFormat, imageInfo.EncryptAlg)
		if err != nil {
			log.Errorf("fail to create overlay qcow2 for kvm disk readonly access: %s", err)
			return nil, errors.Wrap(err, "CreateQcow2")
		}
		originImage = imagePath
		imagePath = tmpFileName
		imageInfo.Path = tmpFileName
	}
	return &SKVMGuestDisk{
		readOnly:     readOnly,
		kvmImagePath: originImage,
		topImagePath: imagePath,
		deployer:     newDeployer(imageInfo, driver),
	}, nil
}

func (d *SKVMGuestDisk) Cleanup() {
	if d.readOnly {
		// if readonly, discard the top image when cleanup
		os.RemoveAll(filepath.Dir(d.topImagePath))
	}
}

var _ deploy_iface.IDeployer = (*qemu_kvm.QemuKvmDriver)(nil)
var _ deploy_iface.IDeployer = (*qemu_kvm.LocalDiskDriver)(nil)

func newDeployer(imageInfo qemuimg.SImageInfo, driver string) deploy_iface.IDeployer {
	switch driver {
	case consts.DEPLOY_DRIVER_NBD:
		return nbd.NewNBDDriver(imageInfo)
	case consts.DEPLOY_DRIVER_LIBGUESTFS:
		return libguestfs.NewLibguestfsDriver(imageInfo)
	case consts.DEPLOY_DRIVER_QEMU_KVM:
		return qemu_kvm.NewQemuKvmDriver(imageInfo)
	case consts.DEPLOY_DRIVER_LOCAL_DISK:
		return qemu_kvm.NewLocalDiskDriver()
	default:
		return nbd.NewNBDDriver(imageInfo)
	}
}

func (d *SKVMGuestDisk) IsLVMPartition() bool {
	return d.deployer.IsLVMPartition()
}

func (d *SKVMGuestDisk) Connect(guestDesc *apis.GuestDesc) error {
	return d.deployer.Connect(guestDesc)
}

func (d *SKVMGuestDisk) Disconnect() error {
	return d.deployer.Disconnect()
}

func (d *SKVMGuestDisk) MountRootfs() (fsdriver.IRootFsDriver, error) {
	return d.MountKvmRootfs()
}

func (d *SKVMGuestDisk) MountKvmRootfs() (fsdriver.IRootFsDriver, error) {
	return d.mountKvmRootfs(false)
}

func (d *SKVMGuestDisk) mountKvmRootfs(readonly bool) (fsdriver.IRootFsDriver, error) {
	return d.deployer.MountRootfs(readonly)
}

func (d *SKVMGuestDisk) MountKvmRootfsReadOnly() (fsdriver.IRootFsDriver, error) {
	return d.mountKvmRootfs(true)
}

func (d *SKVMGuestDisk) UmountKvmRootfs(fd fsdriver.IRootFsDriver) error {
	if part := fd.GetPartition(); part != nil {
		return part.Umount()
	}
	return nil
}

func (d *SKVMGuestDisk) UmountRootfs(fd fsdriver.IRootFsDriver) error {
	if fd == nil {
		return nil
	}
	return d.UmountKvmRootfs(fd)
}

func (d *SKVMGuestDisk) DeployGuestfs(req *apis.DeployParams) (res *apis.DeployGuestFsResponse, err error) {
	return d.deployer.DeployGuestfs(req)
}

func (d *SKVMGuestDisk) ResizeFs() (*apis.Empty, error) {
	return d.deployer.ResizeFs()
}

func (d *SKVMGuestDisk) FormatFs(req *apis.FormatFsParams) (*apis.Empty, error) {
	return d.deployer.FormatFs(req)
}

func (d *SKVMGuestDisk) SaveToGlance(req *apis.SaveToGlanceParams) (*apis.SaveToGlanceResponse, error) {
	return d.deployer.SaveToGlance(req)
}

func (d *SKVMGuestDisk) ProbeImageInfo(req *apis.ProbeImageInfoPramas) (*apis.ImageInfo, error) {
	return d.deployer.ProbeImageInfo(req)
}
