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

package nbd

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/hostman/diskutils/fsutils"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/kvmpart"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
	"yunion.io/x/onecloud/pkg/util/qemutils"
	"yunion.io/x/onecloud/pkg/util/version"
)

const MAX_TRIES = 3

type NBDDriver struct {
	partitions            []fsdriver.IDiskPartition
	lvms                  []*SKVMGuestLVMPartition
	imageRootBackFilePath string
	imageInfo             qemuimg.SImageInfo
	acquiredLvm           bool
	nbdDev                string
}

func NewNBDDriver(imageInfo qemuimg.SImageInfo) *NBDDriver {
	return &NBDDriver{
		imageInfo:  imageInfo,
		partitions: make([]fsdriver.IDiskPartition, 0),
	}
}

var lvmTool *SLVMImageConnectUniqueToolSet

func init() {
	lvmTool = NewLVMImageConnectUniqueToolSet()
}

func (d *NBDDriver) Connect() error {
	pathType := lvmTool.GetPathType(d.rootImagePath())
	if pathType == LVM_PATH || pathType == PATH_TYPE_UNKNOWN {
		lvmTool.Acquire(d.rootImagePath())
		d.acquiredLvm = true
	}

	d.nbdDev = GetNBDManager().AcquireNbddev()
	if len(d.nbdDev) == 0 {
		return errors.Errorf("Cannot get nbd device")
	}
	if err := QemuNbdConnect(d.imageInfo, d.nbdDev); err != nil {
		return err
	}

	var tried uint = 0
	for len(d.partitions) == 0 && tried < MAX_TRIES {
		time.Sleep((1 << tried) * time.Second)
		err := d.findPartitions()
		if err != nil {
			log.Errorln(err.Error())
			return err
		}
		tried += 1
	}

	if pathType == LVM_PATH {
		if _, err := d.setupLVMS(); err != nil {
			return err
		}
	} else if pathType == PATH_TYPE_UNKNOWN {
		hasLVM, err := d.setupLVMS()
		if err != nil {
			return err
		}

		// no lvm partition found and has partitions
		if !hasLVM && len(d.partitions) > 0 {
			d.cacheNonLVMImagePath()
		}
	}
	return nil
}

func (d *NBDDriver) findPartitions() error {
	if len(d.nbdDev) == 0 {
		return fmt.Errorf("Want find partitions but dosen't have nbd dev")
	}
	dev := filepath.Base(d.nbdDev)
	devpath := filepath.Dir(d.nbdDev)
	files, err := os.ReadDir(devpath)
	if err != nil {
		return errors.Wrapf(err, "read dir %s", devpath)
	}
	for i := 0; i < len(files); i++ {
		if files[i].Name() != dev && strings.HasPrefix(files[i].Name(), dev+"p") {
			var part = kvmpart.NewKVMGuestDiskPartition(path.Join(devpath, files[i].Name()), "", false)
			d.partitions = append(d.partitions, part)
		}
	}

	return nil
}

func (d *NBDDriver) rootImagePath() string {
	if len(d.imageRootBackFilePath) > 0 {
		return d.imageRootBackFilePath
	}

	d.imageRootBackFilePath = d.imageInfo.Path
	img, err := qemuimg.NewQemuImage(d.imageInfo.Path)
	if err != nil {
		return d.imageRootBackFilePath
	}

	for len(img.BackFilePath) > 0 {
		d.imageRootBackFilePath = img.BackFilePath
		img, err = qemuimg.NewQemuImage(img.BackFilePath)
		if err != nil {
			break
		}
	}
	return d.imageRootBackFilePath
}

func (d *NBDDriver) isNonLvmImagePath() bool {
	pathType := lvmTool.GetPathType(d.rootImagePath())
	return pathType == NON_LVM_PATH
}

func (d *NBDDriver) cacheNonLVMImagePath() {
	lvmTool.CacheNonLvmImagePath(d.rootImagePath())
}

func (d *NBDDriver) setupLVMS() (bool, error) {
	// Scan all devices and send the metadata to lvmetad
	output, err := procutils.NewCommand("pvscan", "--cache").Output()
	if err != nil {
		log.Errorf("pvscan error %s", output)
		return false, err
	}

	lvmPartitions := []fsdriver.IDiskPartition{}
	for _, part := range d.partitions {
		vg, err := findVg(part.GetPartDev())
		if err != nil {
			log.Errorf("unable to find vg from %s: %v", part.GetPartDev(), err)
			continue
		}
		log.Infof("find vg %s from %s", vg.Name, part.GetPartDev())
		lvm := NewKVMGuestLVMPartition(part.GetPartDev(), vg)
		d.lvms = append(d.lvms, lvm)
		if lvm.SetupDevice() {
			if subparts := lvm.FindPartitions(); len(subparts) > 0 {
				for i := 0; i < len(subparts); i++ {
					lvmPartitions = append(lvmPartitions, subparts[i])
				}
			} else {
				log.Infof("wait a second and try again")
				time.Sleep(time.Second)
				subparts := lvm.FindPartitions()
				if len(subparts) > 0 {
					for i := 0; i < len(subparts); i++ {
						lvmPartitions = append(lvmPartitions, subparts[i])
					}
				}
			}
		}
	}

	if len(lvmPartitions) > 0 {
		d.partitions = append(d.partitions, lvmPartitions...)
		return true, nil
	} else {
		return false, nil
	}
}

func (d *NBDDriver) findLVMPartitions(partDev string) string {
	return findVgname(partDev)
}

func (d *NBDDriver) Disconnect() error {
	if len(d.nbdDev) > 0 {
		defer d.lvmDisconnectNotify()
		d.putdownLVMs()
		return d.disconnect()
	} else {
		return nil
	}
}

func (d *NBDDriver) lvmDisconnectNotify() {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Catch panic on LvmDisconnectNotify %v \n %s", r, debug.Stack())
		}
	}()
	pathType := lvmTool.GetPathType(d.rootImagePath())
	if d.acquiredLvm || pathType != NON_LVM_PATH {
		lvmTool.Release(d.rootImagePath())
	}
}

func (d *NBDDriver) disconnect() error {
	if err := QemuNbdDisconnect(d.nbdDev); err != nil {
		return err
	}
	GetNBDManager().ReleaseNbddev(d.nbdDev)
	d.nbdDev = ""
	d.partitions = d.partitions[len(d.partitions):]
	return nil
}

func (d *NBDDriver) putdownLVMs() {
	for _, lvm := range d.lvms {
		lvm.PutdownDevice()
	}
	d.lvms = []*SKVMGuestLVMPartition{}
}

func (d *NBDDriver) GetPartitions() []fsdriver.IDiskPartition {
	return d.partitions
}

func (d *NBDDriver) MakePartition(fs string) error {
	return fsutils.Mkpartition(d.nbdDev, fs)
}

func (d *NBDDriver) FormatPartition(fs, uuid string) error {
	return fsutils.FormatPartition(fmt.Sprintf("%sp1", d.nbdDev), fs, uuid)
}

func (d *NBDDriver) ResizePartition() error {
	if d.IsLVMPartition() {
		// do not resize LVM partition
		return nil
	}
	return fsutils.ResizeDiskFs(d.nbdDev, 0)
}

func (d *NBDDriver) Zerofree() {
	startTime := time.Now()
	for _, part := range d.partitions {
		part.Zerofree()
	}
	log.Infof("Zerofree %d partitions takes %f seconds", len(d.partitions), time.Now().Sub(startTime).Seconds())
}

func (d *NBDDriver) IsLVMPartition() bool {
	return len(d.lvms) > 0
}

func getQemuNbdVersion() (string, error) {
	output, err := procutils.NewRemoteCommandAsFarAsPossible(qemutils.GetQemuNbd(), "--version").Output()
	if err != nil {
		log.Errorf("qemu-nbd version failed %s %s", output, err.Error())
		return "", errors.Wrapf(err, "qemu-nbd version failed %s", output)
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) > 0 {
		parts := strings.Split(lines[0], " ")
		return parts[1], nil
	}
	return "", errors.Error("empty version output")
}

func QemuNbdConnect(imageInfo qemuimg.SImageInfo, nbddev string) error {
	nbdVer, err := getQemuNbdVersion()
	if err != nil {
		return errors.Wrap(err, "getQemuNbdVersion")
	}
	var cmd []string
	if strings.HasPrefix(imageInfo.Path, "rbd:") || getImageFormat(imageInfo) == "raw" {
		//qemu-nbd 连接ceph时 /etc/ceph/ceph.conf 必须存在
		if strings.HasPrefix(imageInfo.Path, "rbd:") {
			err := procutils.NewRemoteCommandAsFarAsPossible("mkdir", "-p", "/etc/ceph").Run()
			if err != nil {
				log.Errorf("Failed to mkdir /etc/ceph: %s", err)
				return errors.Wrap(err, "Failed to mkdir /etc/ceph: %s")
			}
			err = procutils.NewRemoteCommandAsFarAsPossible("test", "-f", "/etc/ceph/ceph.conf").Run()
			if err != nil {
				err = procutils.NewRemoteCommandAsFarAsPossible("touch", "/etc/ceph/ceph.conf").Run()
				if err != nil {
					log.Errorf("failed to create /etc/ceph/ceph.conf: %s", err)
					return errors.Wrap(err, "failed to create /etc/ceph/ceph.conf")
				}
			}
		}
		cmd = []string{qemutils.GetQemuNbd(), "-c", nbddev, "-f", "raw", imageInfo.Path}
	} else if imageInfo.Encrypted() {
		cmd = []string{
			qemutils.GetQemuNbd(), "-c", nbddev,
			"--object", imageInfo.SecretOptions(),
			"--image-opts", imageInfo.ImageOptions(),
		}
	} else {
		cmd = []string{qemutils.GetQemuNbd(), "-c", nbddev, imageInfo.Path}
	}
	if version.GE(nbdVer, "4.0.0") {
		cmd = append(cmd, "--fork")
	}
	output, err := procutils.NewRemoteCommandAsFarAsPossible(cmd[0], cmd[1:]...).Output()
	if err != nil {
		log.Errorf("qemu-nbd connect failed %s %s", output, err.Error())
		return errors.Wrapf(err, "qemu-nbd connect failed %s", output)
	}
	return nil
}

func getImageFormat(imageInfo qemuimg.SImageInfo) string {
	img, err := qemuimg.NewQemuImage(imageInfo.Path)
	if err == nil {
		return string(img.Format)
	}
	log.Errorf("NBD getImageFormat fail: %s", err)
	return ""
}

func QemuNbdDisconnect(nbddev string) error {
	output, err := procutils.NewRemoteCommandAsFarAsPossible(qemutils.GetQemuNbd(), "-d", nbddev).Output()
	if err != nil {
		return errors.Wrapf(err, "qemu-nbd disconnect %s", output)
	}
	return nil
}
