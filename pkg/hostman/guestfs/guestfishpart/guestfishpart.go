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

package guestfishpart

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/hostman/diskutils/libguestfs/guestfish"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/kvmpart"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

type SGuestfishDiskPartition struct {
	*kvmpart.SLocalGuestFS

	fish *guestfish.Guestfish
	// device name in guest fish
	dev string
	// one of part in guest fish filesystems
	partDev string
	// guest fish detected filesystem type
	fs string
	// is partition mounted on host filesystem
	mounted bool
	// mount as readonly
	readonly bool

	mountErrorChan chan error
}

var _ fsdriver.IDiskPartition = &SGuestfishDiskPartition{}

/* dev like /dev/sda, partDev like /dev/sda1 */
func NewGuestfishDiskPartition(
	dev, partDev, fs string, fish *guestfish.Guestfish,
) *SGuestfishDiskPartition {
	mountPath := fmt.Sprintf("/tmp/%s", strings.Replace(partDev, "/", "_", -1))
	return &SGuestfishDiskPartition{
		SLocalGuestFS:  kvmpart.NewLocalGuestFS(mountPath),
		dev:            dev,
		partDev:        partDev,
		fs:             fs,
		fish:           fish,
		mountErrorChan: make(chan error, 1),
	}
}

func (d *SGuestfishDiskPartition) GetPartDev() string {
	return d.partDev
}

func (d *SGuestfishDiskPartition) IsMounted() bool {
	return d.mounted
}

func (d *SGuestfishDiskPartition) fsck() error {
	switch d.fs {
	case "hfsplus", "ext2", "ext3", "ext4":
		return d.fish.Fsck(d.partDev, d.fs)
	case "ntfs":
		return d.fish.Ntfsfix(d.partDev)
	}
	return nil
}

func (d *SGuestfishDiskPartition) Mount() bool {
	err := d.fsck()
	if err != nil {
		log.Errorf("fsck error: %s", err)
		return false
	}
	err = d.mount(false)
	if err != nil {
		log.Errorf("mount error：%s", err)
		return false
	}
	return true
}

func (d *SGuestfishDiskPartition) mount(readonly bool) error {
	if output, err := procutils.NewCommand("mkdir", "-p", d.GetMountPath()).Output(); err != nil {
		return errors.Wrapf(err, "mkdir %s failed: %s", d.GetMountPath(), output)
	}
	err := d.fish.Mount(d.partDev)
	if err != nil {
		return err
	}

	err = d.fish.MountLocal(d.GetMountPath(), readonly)
	if err != nil {
		return err
	}

	go func() {
		d.mountErrorChan <- d.fish.MountLocalRun()
	}()

	select {
	case err = <-d.mountErrorChan:
		return err
	case <-time.After(time.Second * 1):
		cmd := exec.Command("ls", d.GetMountPath())
		accessibleChan := make(chan error)
		go func() {
			accessibleChan <- cmd.Run()
		}()
		select {
		case err = <-accessibleChan:
			log.Infof("mount filesystem %s accessable: %v", d.GetMountPath(), err)
		case <-time.After(3 * time.Second):
			log.Errorf("mount filesystem %s not accessable", d.GetMountPath())
			err = cmd.Process.Kill()
			if err != nil {
				log.Errorf("failed kill ls process %s", err)
			}
			return errors.Errorf("mount filesystem %s not accessable", d.GetMountPath())
		}

		log.Infof("may be mount success")
	}

	d.mounted = true
	return nil
}

func (d *SGuestfishDiskPartition) MountPartReadOnly() bool {
	if len(d.fs) == 0 || utils.IsInStringArray(d.fs, []string{"swap", "btrfs"}) {
		return false
	}
	err := d.mount(true)
	if err != nil {
		log.Errorf("SGuestfishDiskPartition mount as readonly error: %s", err)
		return false
	}
	d.readonly = true
	return true
}

func (d *SGuestfishDiskPartition) Umount() error {
	if !d.IsMounted() {
		return nil
	}

	var tries = 0
	var err error
	for tries < 10 {
		tries += 1
		output, err := procutils.NewCommand("umount", d.GetMountPath()).Output()
		if err != nil {
			err = errors.Wrapf(err, "failed umount %s: %s %s", d.GetMountPath(), output, err)
			time.Sleep(time.Second * 1)
		} else {
			d.mounted = false
			return nil
		}
	}
	return err
}

func (d *SGuestfishDiskPartition) IsReadonly() bool {
	return d.readonly
}

func (d *SGuestfishDiskPartition) GetPhysicalPartitionType() string {
	ret, err := d.fish.SfdiskL(d.dev)
	if err != nil {
		log.Errorf("failed sfdisk-l %s: %s", d.dev, err)
		return ""
	}
	var partType string
	for i := 0; i < len(ret); i++ {
		if idx := strings.Index(ret[i], "Disk label type:"); idx > 0 {
			partType = strings.TrimSpace(string(ret[i])[idx+len("Disk label type:"):])
		}
	}
	if partType == "dos" {
		return "mbr"
	} else {
		return partType
	}
}

func (d *SGuestfishDiskPartition) Zerofree() {
	if !d.IsMounted() {
		switch d.fs {
		case "swap":
			d.zerofreeSwap()
		case "ext2", "ext3", "ext4":
			d.zerofreeExt()
		case "xfs", "ntfs":
			d.zerofreeSpace()
		}
	}
}

func (d *SGuestfishDiskPartition) zerofreeSwap() {
	res, err := d.fish.Blkid(d.partDev)
	if err != nil {
		log.Errorf("failed get blkid %s", err)
		return
	}
	var label, uuid string
	for i := 0; i < len(res); i++ {
		if strings.HasPrefix(res[i], "UUID:") {
			uuid = strings.TrimSpace(strings.Split(res[i], "")[1])
		} else if strings.HasPrefix(res[i], "LABEL:") {
			label = strings.TrimSpace(strings.Split(res[i], "")[1])
		}
	}
	if len(uuid) == 0 {
		log.Warningf("zerofree swap missing uuid")
		return
	}
	err = d.fish.Mkswap(d.partDev, uuid, label)
	if err != nil {
		log.Errorf("mkswap failed %s", err)
	}
}

func (d *SGuestfishDiskPartition) zerofreeExt() {
	err := d.fish.Zerofree(d.partDev)
	if err != nil {
		log.Errorf("zerofree %s failed %s", d.partDev, err)
	}
}

// mount and zero-free-space
func (d *SGuestfishDiskPartition) zerofreeSpace() {
	if err := d.fish.Mount(d.partDev); err != nil {
		log.Errorf("failed mount partDev %s", err)
		return
	}
	if err := d.fish.ZeroFreeSpace("/"); err != nil {
		log.Errorf("guestfish zero free space failed %s", err)
	}
}
