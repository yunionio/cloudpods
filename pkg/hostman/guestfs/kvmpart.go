package guestfs

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

type SKVMGuestDiskPartition struct {
	*SLocalGuestFS
	partDev string
	fs      string

	readonly bool
}

func NewKVMGuestDiskPartition(devPath string) *SKVMGuestDiskPartition {
	var res = new(SKVMGuestDiskPartition)
	res.partDev = devPath
	res.fs = res.getFsFormat()
	fileutils2.CleanFailedMountpoints()
	mountPath := fmt.Sprintf("/tmp/%s", strings.Replace(devPath, "/", "_", -1))
	res.SLocalGuestFS = NewLocalGuestFS(mountPath)
	return res
}

func (p *SKVMGuestDiskPartition) IsReadonly() bool {
	return IsPartitionReadonly(p)
}

func (p *SKVMGuestDiskPartition) getFsFormat() string {
	return fileutils2.GetFsFormat(p.partDev)
}

func (p *SKVMGuestDiskPartition) Mount() bool {
	if len(p.fs) == 0 || utils.IsInStringArray(p.fs, []string{"swap", "btrfs"}) {
		return false
	}
	err := p.fsck()
	if err != nil {
		log.Errorf("SKVMGuestDiskPartition fsck error: %s", err)
		return false
	}
	err = p.mount(false)
	if err != nil {
		log.Errorf("SKVMGuestDiskPartition mount error: %s", err)
		return false
	}
	if p.IsReadonly() {
		log.Errorf("SKVMGuestDiskPartition %s is readonly, try mount as ro", p.partDev)
		p.Umount()
		err = p.mount(true)
		if err != nil {
			log.Errorf("SKVMGuestDiskPartition mount as ro error %s", err)
			return false
		} else {
			p.readonly = true
		}
	}
	return true
}

func (p *SKVMGuestDiskPartition) mount(readonly bool) error {
	if err := exec.Command("mkdir", "-p", p.mountPath).Run(); err != nil {
		return err
	}
	var cmds = []string{"mount", "-t"}
	var opt, fsType string
	if readonly {
		opt = "ro"
	}
	fsType = p.fs
	if fsType == "ntfs" {
		fsType = "ntfs-3g"
		if !readonly {
			opt = "recover,remove_hiberfile,noatime,windows_names"
		}
	} else if fsType == "hfsplus" && !readonly {
		opt = "force,rw"
	}
	cmds = append(cmds, fsType)
	if len(opt) > 0 {
		cmds = append(cmds, "-o", opt)
	}
	cmds = append(cmds, p.partDev, p.mountPath)
	return exec.Command(cmds[0], cmds[1:]...).Run()
}

func (p *SKVMGuestDiskPartition) fsck() error {
	var checkCmd, fixCmd []string
	switch p.fs {
	case "hfsplus":
		checkCmd = []string{"fsck.hfsplus", "-q", p.partDev}
		fixCmd = []string{"fsck.hfsplus", "-fpy", p.partDev}
	case "ext2", "ext3", "ext4":
		checkCmd = []string{"e2fsck", "-n", p.partDev}
		fixCmd = []string{"e2fsck", "-fp", p.partDev}
	case "ntfs":
		checkCmd = []string{"ntfsfix", "-n", p.partDev}
		fixCmd = []string{"ntfsfix", p.partDev}
	}
	if len(checkCmd) > 0 {
		_, err := exec.Command(checkCmd[0], checkCmd[1:]...).Output()
		if err != nil {
			log.Warningf("FS %s dirty, try to repair ...", p.partDev)
			for i := 0; i < 3; i++ {
				_, err := exec.Command(fixCmd[0], fixCmd[1:]...).Output()
				if err == nil {
					break
				} else {
					return err
				}
			}
		}
	}
	return nil
}

func (p *SKVMGuestDiskPartition) Exists(sPath string, caseInsensitive bool) bool {
	sPath = p.getLocalPath(sPath, caseInsensitive)
	if len(sPath) > 0 {
		if _, err := os.Stat(sPath); !os.IsNotExist(err) {
			return true
		}
	}
	return false
}

func (p *SKVMGuestDiskPartition) IsMounted() bool {
	if _, err := os.Stat(p.mountPath); os.IsNotExist(err) {
		return false
	}
	err := exec.Command("mountpoint", p.mountPath).Run()
	if err == nil {
		return true
	} else {
		log.Errorln(err)
	}
	return false
}

func (p *SKVMGuestDiskPartition) Umount() bool {
	if p.IsMounted() {
		var tries = 0
		for tries < 10 {
			tries += 1
			err := exec.Command("umount", p.mountPath).Run()
			if err == nil {
				exec.Command("blockdev", "--flushbufs", p.partDev).Run()
				os.Remove(p.mountPath)
				return true
			} else {
				time.Sleep(time.Second * 1)
			}
		}
	}
	return false
}

func (p *SKVMGuestDiskPartition) Zerofree() {
	if !p.IsMounted() {
		switch p.fs {
		case "swap":
			p.zerofreeSwap()
		case "ext2", "ext3", "ext4":
			p.zerofreeExt()
		case "ntfs":
			p.zerofreeNtfs()
		}
	}
}

func (p *SKVMGuestDiskPartition) zerofreeSwap() {
	uuids := fileutils2.GetDevUuid(p.partDev)
	err := exec.Command("shred", "-n", "0", "-z", p.partDev).Run()
	if err != nil {
		log.Errorf("zerofree swap error: %s", err)
		return
	}
	cmd := []string{"mkswap"}
	if uuid, ok := uuids["UUID"]; ok {
		cmd = append(cmd, "-U", uuid)
	}
	cmd = append(cmd, p.partDev)
	err = exec.Command(cmd[0], cmd[1:]...).Run()
	if err != nil {
		log.Errorf("zerofree swap error: %s", err)
	}
}

func (p *SKVMGuestDiskPartition) zerofreeExt() {
	err := exec.Command("zerofree", p.partDev).Run()
	if err != nil {
		log.Errorf("zerofree ext error: %s", err)
		return
	}
}

func (p *SKVMGuestDiskPartition) zerofreeNtfs() {
	err := exec.Command("ntfswipe", "-f", "-l", "-m", "-p", "-s", "-q",
		p.partDev).Run()
	if err != nil {
		log.Errorf("zerofree ntfs error: %s", err)
		return
	}
}
