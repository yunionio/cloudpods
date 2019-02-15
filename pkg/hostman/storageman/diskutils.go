package storageman

import (
	"fmt"
	"io/ioutil"
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
	"yunion.io/x/onecloud/pkg/util/qemutils"
)

const MAX_TRIES = 3

type SKVMGuestDisk struct {
	imagePath  string
	nbdDev     string
	partitions []*guestfs.SKVMGuestDiskPartition
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

	var cmd []string
	if strings.HasPrefix(d.imagePath, "rbd:") || d.getImageFormat() == "raw" {
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
	d.setupLVMS()
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
	for i, j := 0, len(d.partitions)-1; i < j; i, j = i+1, j-1 {
		d.partitions[i], d.partitions[j] = d.partitions[j], d.partitions[i]
	}
	return nil
}

func (d *SKVMGuestDisk) setupLVMS() error {
	//TODO?? 可能不需要开发这里
	return fmt.Errorf("not implement right now")
}

func (d *SKVMGuestDisk) Disconnect() bool {
	if len(d.nbdDev) > 0 {
		// TODO?? PutdownLVMS ??
		_, err := procutils.NewCommand(qemutils.GetQemuNbd(), "-d", d.nbdDev).Run()
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
				log.Infof("Use rootfs %s", fs)
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
	log.Infof("Zerofree takes %f seconds", time.Now().Sub(startTime).Seconds())
}
