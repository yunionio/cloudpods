package storageman

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/hostman/guestfs"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	"yunion.io/x/onecloud/pkg/hostman/storageman/nbd"
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
	imagePath  string
	nbdDev     string
	partitions []*guestfs.SKVMGuestDiskPartition
	lvms       []*SKVMGuestLVMPartition

	imageRootBackFilePath string
}

func NewKVMGuestDisk(imagePath string) *SKVMGuestDisk {
	var ret = new(SKVMGuestDisk)
	ret.imagePath = imagePath
	ret.partitions = make([]*guestfs.SKVMGuestDiskPartition, 0)
	return ret
}

func (d *SKVMGuestDisk) Connect() bool {
	d.nbdDev = nbd.GetNBDManager().AcquireNbddev()
	if len(d.nbdDev) == 0 {
		log.Errorln("Cannot get nbd device")
		return false
	}

	d.connectionPrecheck()

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
	_, err := procutils.NewCommand(cmd[0], cmd[1:]...).Run()
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

	if !d.isNonLvmImagePath() {
		hasLVM, err := d.setupLVMS()
		if !hasLVM && err == nil {
			d.cacheNonLVMImagePath()
		}
	}

	return true
}

func (d *SKVMGuestDisk) getImageFormat() string {
	lines, err := procutils.NewCommand(qemutils.GetQemuImg(), "info", d.imagePath).Run()
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
			var part = guestfs.NewKVMGuestDiskPartition(path.Join(devpath, files[i].Name()))
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

func (d *SKVMGuestDisk) rootBackingFilePath() string {
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
	pathType := lvmTool.GetPathType(d.rootBackingFilePath())
	return pathType == NON_LVM_PATH
}

func (d *SKVMGuestDisk) cacheNonLVMImagePath() {
	lvmTool.CacheNonLvmImagePath(d.rootBackingFilePath())
}

func (d *SKVMGuestDisk) connectionPrecheck() {
	pathType := lvmTool.GetPathType(d.rootBackingFilePath())
	switch pathType {
	case LVM_PATH:
		lvmTool.Wait(d.rootBackingFilePath())
		lvmTool.Add(d.rootBackingFilePath())
	case NON_LVM_PATH:
		return
	case PATH_NOT_FOUND:
		lvmTool.Add(d.rootBackingFilePath())
	}
}

func (d *SKVMGuestDisk) LvmDisconnectNotify() {
	if lvmTool.GetPathType(d.rootBackingFilePath()) == LVM_PATH {
		lvmTool.Signal(d.rootBackingFilePath())
	}
}

func (d *SKVMGuestDisk) setupLVMS() (bool, error) {
	// Scan all devices and send the metadata to lvmetad
	output, err := procutils.NewCommand("pvscan", "--cache").Run()
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

func (d *SKVMGuestDisk) Disconnect() bool {
	if len(d.nbdDev) > 0 {
		d.PutdownLVMs()
		_, err := procutils.NewCommand(qemutils.GetQemuNbd(), "-d", d.nbdDev).Run()
		d.LvmDisconnectNotify()
		if err != nil {
			log.Errorln(err.Error())
			return false
		}
		nbd.GetNBDManager().ReleaseNbddev(d.nbdDev)
		d.nbdDev = ""
		d.partitions = d.partitions[len(d.partitions):]
		return true
	} else {
		return false
	}
}

func (d *SKVMGuestDisk) MountKvmRootfs() fsdriver.IRootFsDriver {
	for i := 0; i < len(d.partitions); i++ {
		if d.partitions[i].Mount() {
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

func (d *SKVMGuestDisk) UmountKvmRootfs(fd fsdriver.IRootFsDriver) {
	if part := fd.GetPartition(); part != nil {
		part.Umount()
	}
}

func (d *SKVMGuestDisk) MakePartition(fs string) error {
	return fileutils2.Mkpartition(d.nbdDev, fs)
}

func (d *SKVMGuestDisk) FormatPartition(fs, uuid string) error {
	return fileutils2.FormatPartition(fmt.Sprintf("%sp1", d.nbdDev), fs, uuid)
}

func (d *SKVMGuestDisk) ResizePartition() error {
	return fileutils2.ResizeDiskFs(d.nbdDev, 0)
}

func (d *SKVMGuestDisk) Zerofree() {
	startTime := time.Now()
	for _, part := range d.partitions {
		part.Zerofree()
	}
	log.Infof("Zerofree %d partitions takes %f seconds", len(d.partitions), time.Now().Sub(startTime).Seconds())
}
