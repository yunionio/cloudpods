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

package libguestfs

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"path"
	"path/filepath"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/sortedmap"

	"yunion.io/x/onecloud/pkg/hostman/diskutils/fsutils"
	"yunion.io/x/onecloud/pkg/hostman/diskutils/libguestfs/guestfish"
	"yunion.io/x/onecloud/pkg/hostman/diskutils/nbd"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/guestfishpart"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/kvmpart"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
)

const (
	DiskLabelLength = 6
	letterBytes     = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

type SLibguestfsDriver struct {
	imageInfo qemuimg.SImageInfo

	nbddev    string
	diskLabel string
	lvmParts  []string
	fsmap     *sortedmap.SSortedMap
	fish      *guestfish.Guestfish
	device    string

	parts []fsdriver.IDiskPartition
}

func NewLibguestfsDriver(imageInfo qemuimg.SImageInfo) *SLibguestfsDriver {
	return &SLibguestfsDriver{
		imageInfo: imageInfo,
	}
}

func (d *SLibguestfsDriver) Connect(*apis.GuestDesc) error {
	fish, err := guestfsManager.AcquireFish()
	if err != nil {
		return err
	}
	d.fish = fish

	d.nbddev = nbd.GetNBDManager().AcquireNbddev()
	if err != nil {
		return errors.Errorf("Cannot get nbd device")
	}
	log.Debugf("acquired device %s", d.nbddev)

	err = nbd.QemuNbdConnect(d.imageInfo, d.nbddev)
	if err != nil {
		return err
	}

	lable := RandStringBytes(DiskLabelLength)
	err = fish.AddDrive(d.nbddev, lable, false)
	if err != nil {
		return err
	}
	d.diskLabel = lable

	if err = fish.LvmClearFilter(); err != nil {
		return err
	}

	devices, err := fish.ListDevices()
	if err != nil {
		return err
	}
	if len(devices) == 0 {
		return errors.Errorf("fish list devices no device found")
	}
	d.device = devices[0]

	fsmap, err := fish.ListFilesystems()
	if err != nil {
		return err
	}
	d.fsmap = fsmap
	log.Debugf("fsmap output %#v", d.fsmap)

	lvs, err := fish.Lvs()
	if err != nil {
		return err
	}
	d.lvmParts = lvs

	keys := d.fsmap.Keys()
	for i := 0; i < len(keys); i++ {
		partDev := keys[i]
		ifs, _ := d.fsmap.Get(keys[i])
		fs := ifs.(string)
		log.Debugf("new partition %s %s %s", d.device, partDev, fs)

		/* guestfish run ntfs mount to host is too slow
		 * use host nbd partition replace */
		if fs == "ntfs" && len(d.lvmParts) == 0 {
			log.Infof("has ntfs, use nbd parts")
			d.parts, err = d.findNbdPartitions()
			if err != nil {
				return err
			}
			if err = guestfsManager.ReleaseFish(d.fish); err != nil {
				log.Errorf("release fish failed %s", err)
			}
			d.diskLabel = ""
			d.fish = nil
			break
		}
		part := guestfishpart.NewGuestfishDiskPartition(d.device, partDev, fs, fish)
		d.parts = append(d.parts, part)
	}

	return nil
}

func (d *SLibguestfsDriver) findNbdPartitions() ([]fsdriver.IDiskPartition, error) {
	if len(d.nbddev) == 0 {
		return nil, fmt.Errorf("Want find partitions but dosen't have nbd dev")
	}
	dev := filepath.Base(d.nbddev)
	devpath := filepath.Dir(d.nbddev)
	files, err := ioutil.ReadDir(devpath)
	if err != nil {
		return nil, errors.Wrapf(err, "read dir %s", devpath)
	}

	parts := make([]fsdriver.IDiskPartition, 0)
	for i := 0; i < len(files); i++ {
		if files[i].Name() != dev && strings.HasPrefix(files[i].Name(), dev+"p") {
			var part = kvmpart.NewKVMGuestDiskPartition(path.Join(devpath, files[i].Name()), "", false)
			parts = append(parts, part)
		}
	}
	return parts, nil
}

func (d *SLibguestfsDriver) Disconnect() error {
	if len(d.diskLabel) > 0 {
		if err := guestfsManager.ReleaseFish(d.fish); err != nil {
			log.Errorf("release fish failed %s", err)
		}
		d.diskLabel = ""
		d.fish = nil
	}
	if len(d.nbddev) > 0 {
		if err := nbd.QemuNbdDisconnect(d.nbddev); err != nil {
			return err
		}
		nbd.GetNBDManager().ReleaseNbddev(d.nbddev)
	}
	return nil
}

func (d *SLibguestfsDriver) GetPartitions() []fsdriver.IDiskPartition {
	return d.parts
}

func (d *SLibguestfsDriver) IsLVMPartition() bool {
	return len(d.lvmParts) > 0
}

func (d *SLibguestfsDriver) Zerofree() {
	startTime := time.Now()
	for _, part := range d.parts {
		part.Zerofree()
	}
	log.Infof("libguestfs zerofree %d partitions takes %f seconds",
		len(d.parts), time.Now().Sub(startTime).Seconds())
}

func (d *SLibguestfsDriver) ResizePartition() error {
	if d.IsLVMPartition() {
		// do not try to resize LVM partition
		return nil
	}
	return fsutils.ResizeDiskFs(d.nbddev, 0, false)
}

func (d *SLibguestfsDriver) FormatPartition(fs, uuid string, features *apis.FsFeatures) error {
	return fsutils.FormatPartition(fmt.Sprintf("%sp1", d.nbddev), fs, uuid, features)
}

func (d *SLibguestfsDriver) MakePartition(fsFormat string) error {
	return fsutils.Mkpartition(d.nbddev, fsFormat)
}

func (d *SLibguestfsDriver) FormatPartition2(fs, uuid string) error {
	partDev := fmt.Sprintf("%s1", d.device)
	switch fs {
	case "swap":
		return d.fish.Mkswap(partDev, uuid, "")
	case "ext2", "ext3", "ext4", "xfs", "fat":
		return d.fish.Mkfs(partDev, fs)
	}
	return errors.Errorf("Unknown fs %s", fs)
}

func (d *SLibguestfsDriver) MakePartition2(fsFormat string) error {
	var (
		labelType = "gpt"
		diskType  = fileutils2.FsFormatToDiskType(fsFormat)
	)
	if len(diskType) == 0 {
		return errors.Errorf("Unknown fsFormat %s", fsFormat)
	}

	err := d.fish.PartDisk(d.device, labelType)
	if err != nil {
		return err
	}
	return nil
}

func (d *SLibguestfsDriver) DetectIsUEFISupport(rootfs fsdriver.IRootFsDriver) bool {
	return fsutils.DetectIsUEFISupport(rootfs, d.GetPartitions())
}

func (d *SLibguestfsDriver) MountRootfs(readonly bool) (fsdriver.IRootFsDriver, error) {
	return fsutils.MountRootfs(readonly, d.GetPartitions())
}

func (d *SLibguestfsDriver) UmountRootfs(fd fsdriver.IRootFsDriver) error {
	if part := fd.GetPartition(); part != nil {
		return part.Umount()
	}
	return nil
}

func (d *SLibguestfsDriver) DeployGuestfs(req *apis.DeployParams) (res *apis.DeployGuestFsResponse, err error) {
	return fsutils.DeployGuestfs(d, req)
}

func (d *SLibguestfsDriver) ResizeFs() (*apis.Empty, error) {
	return fsutils.ResizeFs(d)
}

func (d *SLibguestfsDriver) SaveToGlance(req *apis.SaveToGlanceParams) (*apis.SaveToGlanceResponse, error) {
	return fsutils.SaveToGlance(d, req)
}

func (d *SLibguestfsDriver) FormatFs(req *apis.FormatFsParams) (*apis.Empty, error) {
	return fsutils.FormatFs(d, req)
}

func (d *SLibguestfsDriver) ProbeImageInfo(req *apis.ProbeImageInfoPramas) (*apis.ImageInfo, error) {
	return fsutils.ProbeImageInfo(d)
}
