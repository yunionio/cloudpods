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
	"yunion.io/x/onecloud/pkg/hostman/diskutils/libguestfs"
	"yunion.io/x/onecloud/pkg/hostman/diskutils/nbd"
	"yunion.io/x/onecloud/pkg/hostman/guestfs"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/consts"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
)

type SKVMGuestDisk struct {
	readOnly     bool
	kvmImagePath string
	topImagePath string
	deployer     IDeployer
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
			log.Errorf("fail to create overlay qcow2 for kvm disk readonly access")
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

func newDeployer(imageInfo qemuimg.SImageInfo, driver string) IDeployer {
	switch driver {
	case consts.DEPLOY_DRIVER_NBD:
		return nbd.NewNBDDriver(imageInfo)
	case consts.DEPLOY_DRIVER_LIBGUESTFS:
		return libguestfs.NewLibguestfsDriver(imageInfo)
	default:
		return nbd.NewNBDDriver(imageInfo)
	}
}

func (d *SKVMGuestDisk) IsLVMPartition() bool {
	return d.deployer.IsLVMPartition()
}

func (d *SKVMGuestDisk) Connect() error {
	return d.deployer.Connect()
}

func (d *SKVMGuestDisk) Disconnect() error {
	return d.deployer.Disconnect()
}

func (d *SKVMGuestDisk) DetectIsUEFISupport(rootfs fsdriver.IRootFsDriver) bool {
	partitions := d.deployer.GetPartitions()
	for i := 0; i < len(partitions); i++ {
		if partitions[i].IsMounted() {
			if rootfs.DetectIsUEFISupport(partitions[i]) {
				return true
			}
		} else {
			if partitions[i].Mount() {
				support := rootfs.DetectIsUEFISupport(partitions[i])
				if err := partitions[i].Umount(); err != nil {
					log.Errorf("failed umount %s: %s", partitions[i].GetPartDev(), err)
				}
				if support {
					return true
				}
			}
		}
	}
	return false
}

func (d *SKVMGuestDisk) MountRootfs() (fsdriver.IRootFsDriver, error) {
	return d.MountKvmRootfs()
}

func (d *SKVMGuestDisk) MountKvmRootfs() (fsdriver.IRootFsDriver, error) {
	return d.mountKvmRootfs(false)
}

func (d *SKVMGuestDisk) mountKvmRootfs(readonly bool) (fsdriver.IRootFsDriver, error) {
	partitions := d.deployer.GetPartitions()
	errs := []error{}
	for i := 0; i < len(partitions); i++ {
		log.Infof("detect partition %s", partitions[i].GetPartDev())
		mountFunc := partitions[i].Mount
		if readonly {
			mountFunc = partitions[i].MountPartReadOnly
		}
		if mountFunc() {
			fs, err := guestfs.DetectRootFs(partitions[i])
			if err == nil {
				log.Infof("Use rootfs %s, partition %s", fs, partitions[i].GetPartDev())
				return fs, nil
			}
			errs = append(errs, err)
			if err := partitions[i].Umount(); err != nil {
				log.Errorf("failed umount %s: %s", partitions[i].GetPartDev(), err)
			}
		}
	}
	if len(partitions) == 0 {
		return nil, errors.Wrap(errors.ErrNotFound, "not found any partition")
	}
	var err error = errors.ErrNotFound
	if len(errs) > 0 {
		err = errors.Wrapf(errors.ErrNotFound, errors.NewAggregate(errs).Error())
	}
	return nil, err
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

func (d *SKVMGuestDisk) MakePartition(fs string) error {
	return d.deployer.MakePartition(fs)
}

func (d *SKVMGuestDisk) FormatPartition(fs, uuid string) error {
	return d.deployer.FormatPartition(fs, uuid)
}

func (d *SKVMGuestDisk) ResizePartition() error {
	return d.deployer.ResizePartition()
}

func (d *SKVMGuestDisk) Zerofree() {
	d.deployer.Zerofree()
}
