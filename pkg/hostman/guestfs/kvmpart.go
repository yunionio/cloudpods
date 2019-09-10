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

package guestfs

import (
	"fmt"
	"os"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

type SKVMGuestDiskPartition struct {
	*SLocalGuestFS
	partDev string
	fs      string

	readonly  bool
	sourceDev string
	IsLVMPart bool
}

func NewKVMGuestDiskPartition(devPath, sourceDev string, isLVM bool) *SKVMGuestDiskPartition {
	var res = new(SKVMGuestDiskPartition)
	res.partDev = devPath
	res.fs = res.getFsFormat()
	res.sourceDev = sourceDev
	res.IsLVMPart = isLVM
	fileutils2.CleanFailedMountpoints()
	mountPath := fmt.Sprintf("/tmp/%s", strings.Replace(devPath, "/", "_", -1))
	res.SLocalGuestFS = NewLocalGuestFS(mountPath)
	return res
}

func (p *SKVMGuestDiskPartition) GetPhysicalPartitionType() string {
	dev := p.partDev
	if p.IsLVMPart {
		dev = p.sourceDev
	}
	idxP := strings.LastIndexByte(dev, 'p')
	if idxP > 0 {
		dev = dev[:idxP]
	}
	cmd := fmt.Sprintf(`fdisk -l %s | grep "Disk label type:"`, dev)
	output, err := procutils.NewCommand("sh", "-c", cmd).Run()
	if err != nil {
		log.Errorf("get disk label type error %s", output)
		return ""
	}
	idx := strings.Index(string(output), "Disk label type:")
	tp := strings.TrimSpace(string(output)[idx+len("Disk label type:"):])
	if tp == "dos" {
		return "mbr"
	} else {
		return tp
	}
}

func (p *SKVMGuestDiskPartition) GetPartDev() string {
	return p.partDev
}

func (p *SKVMGuestDiskPartition) IsReadonly() bool {
	return IsPartitionReadonly(p)
}

func (p *SKVMGuestDiskPartition) getFsFormat() string {
	return fileutils2.GetFsFormat(p.partDev)
}

func (p *SKVMGuestDiskPartition) MountPartReadOnly() bool {
	if len(p.fs) == 0 || utils.IsInStringArray(p.fs, []string{"swap", "btrfs"}) {
		return false
	}

	// no fsck, becasue read only

	err := p.mount(true)
	if err != nil {
		log.Errorf("SKVMGuestDiskPartition mount as readonly error: %s", err)
		return false
	}
	p.readonly = true
	return true
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
	if _, err := procutils.NewCommand("mkdir", "-p", p.mountPath).Run(); err != nil {
		log.Errorln(err)
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
	_, err := procutils.NewCommand(cmds[0], cmds[1:]...).Run()
	return err
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
		_, err := procutils.NewCommand(checkCmd[0], checkCmd[1:]...).Run()
		if err != nil {
			log.Warningf("FS %s dirty, try to repair ...", p.partDev)
			for i := 0; i < 3; i++ {
				_, err := procutils.NewCommand(fixCmd[0], fixCmd[1:]...).Run()
				if err == nil {
					break
				} else {
					log.Errorf("Try to fix partition %s failed: %s", fixCmd, err)
					continue
				}
			}
		}
	}
	return nil
}

func (p *SKVMGuestDiskPartition) Exists(sPath string, caseInsensitive bool) bool {
	sPath = p.GetLocalPath(sPath, caseInsensitive)
	if len(sPath) > 0 {
		return fileutils2.Exists(sPath)
	}
	return false
}

func (p *SKVMGuestDiskPartition) IsMounted() bool {
	if !fileutils2.Exists(p.mountPath) {
		return false
	}
	_, err := procutils.NewCommand("mountpoint", p.mountPath).Run()
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
			_, err := procutils.NewCommand("umount", p.mountPath).Run()
			if err == nil {
				procutils.NewCommand("blockdev", "--flushbufs", p.partDev).Run()
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
	_, err := procutils.NewCommand("shred", "-n", "0", "-z", p.partDev).Run()
	if err != nil {
		log.Errorf("zerofree swap error: %s", err)
		return
	}
	cmd := []string{"mkswap"}
	if uuid, ok := uuids["UUID"]; ok {
		cmd = append(cmd, "-U", uuid)
	}
	cmd = append(cmd, p.partDev)
	_, err = procutils.NewCommand(cmd[0], cmd[1:]...).Run()
	if err != nil {
		log.Errorf("zerofree swap error: %s", err)
	}
}

func (p *SKVMGuestDiskPartition) zerofreeExt() {
	_, err := procutils.NewCommand("zerofree", p.partDev).Run()
	if err != nil {
		log.Errorf("zerofree ext error: %s", err)
		return
	}
}

func (p *SKVMGuestDiskPartition) zerofreeNtfs() {
	_, err := procutils.NewCommand("ntfswipe", "-f", "-l", "-m", "-p", "-s", "-q",
		p.partDev).Run()
	if err != nil {
		log.Errorf("zerofree ntfs error: %s", err)
		return
	}
}
