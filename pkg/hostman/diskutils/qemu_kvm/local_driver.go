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

package qemu_kvm

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/hostman/diskutils/fsutils"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/kvmpart"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type LocalDiskDriver struct {
	partitions    []fsdriver.IDiskPartition
	lvmPartitions []fsdriver.IDiskPartition
}

func NewLocalDiskDriver() *LocalDiskDriver {
	return &LocalDiskDriver{
		partitions:    make([]fsdriver.IDiskPartition, 0),
		lvmPartitions: make([]fsdriver.IDiskPartition, 0),
	}
}

func (d *LocalDiskDriver) Connect(desc *apis.GuestDesc) error {
	out, err := procutils.NewCommand("sh", "-c", "cat /proc/partitions | grep -v name | awk '{print $4}'").Output()
	if err != nil {
		return errors.Wrap(err, "cat proc partitions")
	}
	lines := strings.Split(string(out), "\n")
	partDevs := make([]string, 0)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "sd") {
			partDevs = append(partDevs, line)
		}
	}
	sortPartDevs := stringutils2.NewSortedStrings(partDevs)
	for _, partDev := range sortPartDevs {
		part := kvmpart.NewKVMGuestDiskPartition(fmt.Sprintf("/dev/%s", partDev), "", false)
		d.partitions = append(d.partitions, part)
		log.Infof("found part dev %s", part.GetPartDev())
	}
	d.setupLVMS()
	if len(d.lvmPartitions) > 0 {
		d.partitions = append(d.partitions, d.lvmPartitions...)
	}
	return nil
}

func (d *LocalDiskDriver) Disconnect() error {
	return nil
}

func (d *LocalDiskDriver) GetPartitions() []fsdriver.IDiskPartition {
	return d.partitions
}

func (d *LocalDiskDriver) IsLVMPartition() bool {
	return len(d.lvmPartitions) > 0
}

func (d *LocalDiskDriver) Zerofree() {
	startTime := time.Now()
	for _, part := range d.partitions {
		part.Zerofree()
	}
	log.Infof("Zerofree %d partitions takes %f seconds", len(d.partitions), time.Now().Sub(startTime).Seconds())
}

func (d *LocalDiskDriver) ResizePartition() error {
	if d.IsLVMPartition() {
		// do not resize LVM partition
		return nil
	}
	return fsutils.ResizeDiskFs("/dev/sda", 0, false)
}

func (d *LocalDiskDriver) FormatPartition(fs, uuid string, features *apis.FsFeatures) error {
	return fsutils.FormatPartition("/dev/sda1", fs, uuid, features)
}

func (d *LocalDiskDriver) MakePartition(fs string) error {
	return fsutils.Mkpartition("/dev/sda", fs)
}

func (d *LocalDiskDriver) DetectIsUEFISupport(rootfs fsdriver.IRootFsDriver) bool {
	return fsutils.DetectIsUEFISupport(rootfs, d.GetPartitions())
}

func (d *LocalDiskDriver) MountRootfs(readonly bool) (fsdriver.IRootFsDriver, error) {
	return fsutils.MountRootfs(readonly, d.GetPartitions())
}

func (d *LocalDiskDriver) UmountRootfs(fd fsdriver.IRootFsDriver) error {
	if part := fd.GetPartition(); part != nil {
		return part.Umount()
	}
	return nil
}

func (d *LocalDiskDriver) DeployGuestfs(req *apis.DeployParams) (res *apis.DeployGuestFsResponse, err error) {
	return fsutils.DeployGuestfs(d, req)
}

func (d *LocalDiskDriver) ResizeFs() (*apis.Empty, error) {
	return fsutils.ResizeFs(d)
}

func (d *LocalDiskDriver) SaveToGlance(req *apis.SaveToGlanceParams) (*apis.SaveToGlanceResponse, error) {
	return fsutils.SaveToGlance(d, req)
}

func (d *LocalDiskDriver) FormatFs(req *apis.FormatFsParams) (*apis.Empty, error) {
	return fsutils.FormatFs(d, req)
}

func (d *LocalDiskDriver) ProbeImageInfo(req *apis.ProbeImageInfoPramas) (*apis.ImageInfo, error) {
	return fsutils.ProbeImageInfo(d)
}
