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

package kvmpart

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/hostman/guestfs"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/xfsutils"
)

type SKVMGuestDiskPartition struct {
	*SLocalGuestFS
	partDev string
	fs      string
	uuid    string

	readonly  bool
	sourceDev string
	IsLVMPart bool
}

var _ fsdriver.IDiskPartition = &SKVMGuestDiskPartition{}

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
	cmd := fmt.Sprintf(`fdisk -l %s | grep "Disk.*label type:"`, dev)
	output, err := procutils.NewCommand("sh", "-c", cmd).Output()
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
	return guestfs.IsPartitionReadonly(p)
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
		log.Errorf("Mount fs failed: unsupport fs %s on %s", p.fs, p.partDev)
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
	log.Infof("mount fs %s on %s successfully", p.fs, p.partDev)
	return true
}

func (p *SKVMGuestDiskPartition) mount(readonly bool) error {
	if output, err := procutils.NewCommand("mkdir", "-p", p.mountPath).Output(); err != nil {
		return errors.Wrapf(err, "mkdir %s failed: %s", p.mountPath, output)
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

	var err error
	if fsType == "xfs" {
		uuids, _ := fileutils2.GetDevUuid(p.partDev)
		p.uuid = uuids["UUID"]
		if len(p.uuid) > 0 {
			xfsutils.LockXfsPartition(p.uuid)
			defer func() {
				if err != nil {
					xfsutils.UnlockXfsPartition(p.uuid)
				}
			}()
		}
	}

	retrier := func(utils.FibonacciRetrier) (bool, error) {
		output, err := procutils.NewCommand(cmds[0], cmds[1:]...).Output()
		if err == nil {
			return true, nil
		} else {
			log.Errorf("mount fail: %s %s", err, output)
			time.Sleep(time.Millisecond * time.Duration(100+rand.Intn(400)))
			return false, errors.Wrap(err, "")
		}
	}
	_, err = utils.NewFibonacciRetrierMaxTries(3, retrier).Start(context.Background())
	if err != nil {
		return errors.Wrap(err, "mount failed")
	}
	return nil // errors.Wrapf(err, "mount failed: %s", output)
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
		_, err := procutils.NewCommand(checkCmd[0], checkCmd[1:]...).Output()
		if err != nil {
			log.Warningf("FS %s dirty, try to repair ...", p.partDev)
			for i := 0; i < 3; i++ {
				output, err := procutils.NewCommand(fixCmd[0], fixCmd[1:]...).Output()
				if err == nil {
					break
				} else {
					log.Errorf("Try to fix partition %s failed: %s, %s", fixCmd, err, output)
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
	output, err := procutils.NewCommand("mountpoint", p.mountPath).Output()
	if err == nil {
		return true
	} else {
		log.Errorf("%s is not a mountpoint: %s", p.mountPath, output)
		return false
	}
}

func (p *SKVMGuestDiskPartition) Umount() error {
	if !p.IsMounted() {
		return nil
	}

	defer func() {
		if p.fs == "xfs" && len(p.uuid) > 0 {
			xfsutils.UnlockXfsPartition(p.uuid)
		}
	}()

	var tries = 0
	var err error
	var out []byte
	for tries < 10 {
		tries += 1
		log.Infof("umount %s: %s", p.partDev, p.mountPath)
		out, err = procutils.NewCommand("umount", p.mountPath).Output()
		if err == nil {
			if _, err := procutils.NewCommand("blockdev", "--flushbufs", p.partDev).Output(); err != nil {
				log.Warningf("blockdev --flushbufs %s error: %v", p.partDev, err)
			}
			if err := os.Remove(p.mountPath); err != nil {
				log.Warningf("remove mount path %s error: %v", p.mountPath, err)
			}
			log.Infof("umount %s successfully", p.partDev)
			return nil
		} else {
			log.Warningf("failed umount %s: %s %s", p.partDev, err, out)
			time.Sleep(time.Second * 3)
		}
	}
	return errors.Wrapf(err, "umount %s", p.mountPath)
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
	uuids, _ := fileutils2.GetDevUuid(p.partDev)
	output, err := procutils.NewCommand("shred", "-n", "0", "-z", p.partDev).Output()
	if err != nil {
		log.Errorf("zerofree swap error: %s, %s", err, output)
		return
	}
	cmd := []string{"mkswap"}
	if uuid, ok := uuids["UUID"]; ok {
		cmd = append(cmd, "-U", uuid)
	}
	cmd = append(cmd, p.partDev)
	output, err = procutils.NewCommand(cmd[0], cmd[1:]...).Output()
	if err != nil {
		log.Errorf("zerofree swap error: %s, %s", err, output)
	}
}

func (p *SKVMGuestDiskPartition) zerofreeExt() {
	output, err := procutils.NewCommand("zerofree", p.partDev).Output()
	if err != nil {
		log.Errorf("zerofree ext error: %s, %s", err, output)
		return
	}
}

func (p *SKVMGuestDiskPartition) zerofreeNtfs() {
	err := procutils.NewCommand("ntfswipe", "-f", "-l", "-m", "-p", "-s", "-q", p.partDev).Run()
	if err != nil {
		log.Errorf("zerofree ntfs error: %s", err)
		return
	}
}
