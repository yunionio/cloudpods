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
	"os"
	"path"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/hostman/diskutils/nbd"
	"yunion.io/x/onecloud/pkg/hostman/guestfs"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
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

func (d *SKVMGuestDisk) ConnectWithoutDetectLvm() bool {
	return d.connect()
}

func (d *SKVMGuestDisk) connect() bool {
	d.nbdDev = nbd.GetNBDManager().AcquireNbddev()
	if len(d.nbdDev) == 0 {
		log.Errorln("Cannot get nbd device")
		return false
	}

	var cmd []string
	if strings.HasPrefix(d.imagePath, "rbd:") || d.getImageFormat() == "raw" {
		//qemu-nbd 连接ceph时 /etc/ceph/ceph.conf 必须存在
		if strings.HasPrefix(d.imagePath, "rbd:") {
			if !fileutils2.Exists("/etc/ceph") {
				if err := os.Mkdir("/etc/ceph", 0755); err != nil {
					log.Errorf("failed to mkdir /etc/ceph error: %v", err)
					return false
				}
			}
			if !fileutils2.IsFile("/etc/ceph/ceph.conf") {
				if _, err := os.Create("/etc/ceph/ceph.conf"); err != nil {
					log.Errorf("failed to create /etc/ceph/ceph.conf error: %v", err)
					return false
				}
			}
		}
		cmd = []string{qemutils.GetQemuNbd(), "-c", d.nbdDev, "-f", "raw", d.imagePath}
	} else {
		cmd = []string{qemutils.GetQemuNbd(), "-c", d.nbdDev, d.imagePath}
	}
	_, err := procutils.NewCommand(cmd[0], cmd[1:]...).Output()
	if err != nil {
		log.Errorln(err.Error())
		return false
	}

	var tried uint = 0
	for len(d.partitions) == 0 && tried < MAX_TRIES {
		time.Sleep((1 << tried) * time.Second)
		err = d.findPartitions()
		if err != nil {
			log.Errorln(err.Error())
			return false
		}
		tried += 1
	}
	return true
}

func (d *SKVMGuestDisk) Connect() bool {
	pathType := d.connectionPrecheck()

	if d.connect() == false {
		return false
	}

	if pathType == LVM_PATH {
		d.setupLVMS()
	} else if pathType == PATH_TYPE_UNKNOWN {
		if hasLVM, err := d.setupLVMS(); !hasLVM && err == nil {
			d.cacheNonLVMImagePath()
		}
	}

	return true
}

func (d *SKVMGuestDisk) getImageFormat() string {
	lines, err := procutils.NewCommand(qemutils.GetQemuImg(), "info", d.imagePath).Output()
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
		return err
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

func (d *SKVMGuestDisk) DisconnectWithoutLvm() bool {
	return d.disconnect()
}

func (d *SKVMGuestDisk) Disconnect() bool {
	if len(d.nbdDev) > 0 {
		defer d.LvmDisconnectNotify()
		d.PutdownLVMs()
		return d.disconnect()
	} else {
		return false
	}
}

func (d *SKVMGuestDisk) disconnect() bool {
	_, err := procutils.NewCommand(qemutils.GetQemuNbd(), "-d", d.nbdDev).Output()
	if err != nil {
		log.Errorln(err.Error())
		return false
	}
	nbd.GetNBDManager().ReleaseNbddev(d.nbdDev)
	d.nbdDev = ""
	d.partitions = d.partitions[len(d.partitions):]
	return true

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
