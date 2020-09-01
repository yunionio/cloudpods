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
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/hostman/diskutils/nbd"
	"yunion.io/x/onecloud/pkg/hostman/guestfs"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
	"yunion.io/x/onecloud/pkg/util/qemutils"
)

const MAX_TRIES = 3

var lvmTool *SLVMImageConnectUniqueToolSet

func init() {
	lvmTool = NewLVMImageConnectUniqueToolSet()
}

type SKVMGuestDisk struct {
	imagePath   string
	nbdDev      string
	partitions  []*guestfs.SKVMGuestDiskPartition
	lvms        []*SKVMGuestLVMPartition
	acquiredLvm bool

	imageRootBackFilePath string
}

func NewKVMGuestDisk(imagePath string) *SKVMGuestDisk {
	var ret = new(SKVMGuestDisk)
	ret.imagePath = imagePath
	ret.partitions = make([]*guestfs.SKVMGuestDiskPartition, 0)
	return ret
}

func (d *SKVMGuestDisk) IsLVMPartition() bool {
	return len(d.lvms) > 0
}

func (d *SKVMGuestDisk) connect() error {
	d.nbdDev = nbd.GetNBDManager().AcquireNbddev()
	if len(d.nbdDev) == 0 {
		return errors.Errorf("Cannot get nbd device")
	}

	var cmd []string
	if strings.HasPrefix(d.imagePath, "rbd:") || d.getImageFormat() == "raw" {
		//qemu-nbd 连接ceph时 /etc/ceph/ceph.conf 必须存在
		if strings.HasPrefix(d.imagePath, "rbd:") {
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
		cmd = []string{qemutils.GetQemuNbd(), "-c", d.nbdDev, "-f", "raw", d.imagePath}
	} else {
		cmd = []string{qemutils.GetQemuNbd(), "-c", d.nbdDev, d.imagePath}
	}
	output, err := procutils.NewRemoteCommandAsFarAsPossible(cmd[0], cmd[1:]...).Output()
	if err != nil {
		log.Errorf("qemu-nbd connect failed %s %s", output, err.Error())
		return errors.Wrapf(err, "qemu-nbd connect failed %s", output)
	}

	var tried uint = 0
	for len(d.partitions) == 0 && tried < MAX_TRIES {
		time.Sleep((1 << tried) * time.Second)
		err = d.findPartitions()
		if err != nil {
			log.Errorln(err.Error())
			return err
		}
		tried += 1
	}
	return nil
}

func (d *SKVMGuestDisk) Connect() error {
	pathType := d.connectionPrecheck()

	if err := d.connect(); err != nil {
		return errors.Wrap(err, "disk.connect")
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

func (d *SKVMGuestDisk) getImageFormat() string {
	lines, err := procutils.NewRemoteCommandAsFarAsPossible(qemutils.GetQemuImg(), "info", d.imagePath).Output()
	if err != nil {
		return ""
	}
	imgStr := strings.Split(string(lines), "\n")
	for i := 0; i < len(imgStr); i++ {
		if strings.HasPrefix(imgStr[i], "file format: ") {
			return imgStr[i][len("file format: "):]
		}
	}
	return ""
}

func (d *SKVMGuestDisk) findPartitions() error {
	if len(d.nbdDev) == 0 {
		return fmt.Errorf("Want find partitions but dosen't have nbd dev")
	}
	dev := filepath.Base(d.nbdDev)
	devpath := filepath.Dir(d.nbdDev)
	files, err := ioutil.ReadDir(devpath)
	if err != nil {
		return errors.Wrapf(err, "read dir %s", devpath)
	}
	for i := 0; i < len(files); i++ {
		if files[i].Name() != dev && strings.HasPrefix(files[i].Name(), dev+"p") {
			var part = guestfs.NewKVMGuestDiskPartition(path.Join(devpath, files[i].Name()), "", false)
			d.partitions = append(d.partitions, part)
		}
	}

	// XXX: HACK reverse partitions
	// for i, j := 0, len(d.partitions)-1; i < j; i, j = i+1, j-1 {
	// 	d.partitions[i], d.partitions[j] = d.partitions[j], d.partitions[i]
	// }
	return nil
}

func (d *SKVMGuestDisk) findLVMPartitions(partDev string) string {
	return findVgname(partDev)
}

func (d *SKVMGuestDisk) rootImagePath() string {
	if len(d.imageRootBackFilePath) > 0 {
		return d.imageRootBackFilePath
	}

	d.imageRootBackFilePath = d.imagePath
	img, err := qemuimg.NewQemuImage(d.imagePath)
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

func (d *SKVMGuestDisk) isNonLvmImagePath() bool {
	pathType := lvmTool.GetPathType(d.rootImagePath())
	return pathType == NON_LVM_PATH
}

func (d *SKVMGuestDisk) cacheNonLVMImagePath() {
	lvmTool.CacheNonLvmImagePath(d.rootImagePath())
}

func (d *SKVMGuestDisk) connectionPrecheck() int {
	pathType := lvmTool.GetPathType(d.rootImagePath())
	if pathType == LVM_PATH || pathType == PATH_TYPE_UNKNOWN {
		lvmTool.Acquire(d.rootImagePath())
		d.acquiredLvm = true
	}
	return pathType
}

func (d *SKVMGuestDisk) LvmDisconnectNotify() {
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

func (d *SKVMGuestDisk) setupLVMS() (bool, error) {
	// Scan all devices and send the metadata to lvmetad
	output, err := procutils.NewCommand("pvscan", "--cache").Output()
	if err != nil {
		log.Errorf("pvscan error %s", output)
		return false, err
	}

	lvmPartitions := []*guestfs.SKVMGuestDiskPartition{}
	for _, part := range d.partitions {
		vgname := d.findLVMPartitions(part.GetPartDev())
		if len(vgname) > 0 {
			lvm := NewKVMGuestLVMPartition(part.GetPartDev(), vgname)
			d.lvms = append(d.lvms, lvm)
			if lvm.SetupDevice() {
				if subparts := lvm.FindPartitions(); len(subparts) > 0 {
					lvmPartitions = append(lvmPartitions, subparts...)
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

func (d *SKVMGuestDisk) PutdownLVMs() {
	for _, lvm := range d.lvms {
		lvm.PutdownDevice()
	}
	d.lvms = []*SKVMGuestLVMPartition{}
}

func (d *SKVMGuestDisk) Disconnect() error {
	if len(d.nbdDev) > 0 {
		defer d.LvmDisconnectNotify()
		d.PutdownLVMs()
		return d.disconnect()
	} else {
		return nil
	}
}

func (d *SKVMGuestDisk) disconnect() error {
	output, err := procutils.NewRemoteCommandAsFarAsPossible(qemutils.GetQemuNbd(), "-d", d.nbdDev).Output()
	if err != nil {
		log.Errorln(err.Error())
		return errors.Wrapf(err, "qemu-nbd disconnect %s", output)
	}
	nbd.GetNBDManager().ReleaseNbddev(d.nbdDev)
	d.nbdDev = ""
	d.partitions = d.partitions[len(d.partitions):]
	return nil

}

func (d *SKVMGuestDisk) DetectIsUEFISupport(rootfs fsdriver.IRootFsDriver) bool {
	for i := 0; i < len(d.partitions); i++ {
		if d.partitions[i].IsMounted() {
			if rootfs.DetectIsUEFISupport(d.partitions[i]) {
				return true
			}
		} else {
			if d.partitions[i].Mount() {
				support := rootfs.DetectIsUEFISupport(d.partitions[i])
				d.partitions[i].Umount()
				if support {
					return true
				}
			}
		}
	}
	return false
}

func (d *SKVMGuestDisk) MountRootfs() fsdriver.IRootFsDriver {
	return d.MountKvmRootfs()
}

func (d *SKVMGuestDisk) MountKvmRootfs() fsdriver.IRootFsDriver {
	return d.mountKvmRootfs(false)
}
func (d *SKVMGuestDisk) mountKvmRootfs(readonly bool) fsdriver.IRootFsDriver {
	for i := 0; i < len(d.partitions); i++ {
		mountFunc := d.partitions[i].Mount
		if readonly {
			mountFunc = d.partitions[i].MountPartReadOnly
		}
		if mountFunc() {
			if fs := guestfs.DetectRootFs(d.partitions[i]); fs != nil {
				log.Infof("Use rootfs %s, partition %s",
					fs, d.partitions[i].GetPartDev())
				return fs
			} else {
				d.partitions[i].Umount()
			}
		}
	}
	return nil
}

func (d *SKVMGuestDisk) MountKvmRootfsReadOnly() fsdriver.IRootFsDriver {
	return d.mountKvmRootfs(true)
}

func (d *SKVMGuestDisk) UmountKvmRootfs(fd fsdriver.IRootFsDriver) {
	if part := fd.GetPartition(); part != nil {
		part.Umount()
	}
}

func (d *SKVMGuestDisk) UmountRootfs(fd fsdriver.IRootFsDriver) {
	if fd == nil {
		return
	}
	d.UmountKvmRootfs(fd)
}

func (d *SKVMGuestDisk) MakePartition(fs string) error {
	return Mkpartition(d.nbdDev, fs)
}

func (d *SKVMGuestDisk) FormatPartition(fs, uuid string) error {
	return FormatPartition(fmt.Sprintf("%sp1", d.nbdDev), fs, uuid)
}

func (d *SKVMGuestDisk) ResizePartition() error {
	return ResizeDiskFs(d.nbdDev, 0)
}

func (d *SKVMGuestDisk) Zerofree() {
	startTime := time.Now()
	for _, part := range d.partitions {
		part.Zerofree()
	}
	log.Infof("Zerofree %d partitions takes %f seconds", len(d.partitions), time.Now().Sub(startTime).Seconds())
}
