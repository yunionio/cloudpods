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

package fsutils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/hostman/diskutils/deploy_iface"
	"yunion.io/x/onecloud/pkg/hostman/guestfs"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/regutils2"
)

var ext4UsageTypeLargefileSize int64 = 1024 * 1024 * 1024 * 1024 * 4
var ext4UsageTypeHugefileSize int64 = 1024 * 1024 * 1024 * 512

func SetExt4UsageTypeThresholds(largefile, hugefile int64) {
	if largefile > 0 {
		ext4UsageTypeLargefileSize = largefile
	}
	if hugefile > 0 {
		ext4UsageTypeHugefileSize = hugefile
	}
}

func IsPartedFsString(fsstr string) bool {
	return utils.IsInStringArray(strings.ToLower(fsstr), []string{
		"ext2", "ext3", "ext4", "xfs",
		"fat16", "fat32",
		"hfs", "hfs+", "hfsx",
		"linux-swap", "linux-swap(v1)",
		"ntfs", "reiserfs", "ufs", "btrfs",
	})
}

func ParseDiskPartition(dev string, lines []string) ([][]string, string) {
	var (
		parts        = [][]string{}
		label        string
		labelPartten = regexp.MustCompile(`Partition Table:\s+(?P<label>\w+)`)
		partten      = regexp.MustCompile(`(?P<idx>\d+)\s+(?P<start>\d+)s\s+(?P<end>\d+)s\s+(?P<count>\d+)s`)
	)

	for _, line := range lines {
		if len(label) == 0 {
			m := regutils2.GetParams(labelPartten, line)
			if len(m) > 0 {
				label = m["label"]
			}
		}
		m := regutils2.GetParams(partten, line)
		if len(m) > 0 {
			var (
				idx     = m["idx"]
				start   = m["start"]
				end     = m["end"]
				count   = m["count"]
				devname = dev
			)
			if '0' <= dev[len(dev)-1] && dev[len(dev)-1] <= '9' {
				devname += "p"
			}
			devname += idx
			data := regexp.MustCompile(`\s+`).Split(strings.TrimSpace(line), -1)

			var disktype, fs, flag string
			var offset = 0
			if len(data) > 4 {
				if label == "msdos" {
					disktype = data[4]
					if len(data) > 5 && IsPartedFsString(data[5]) {
						fs = data[5]
						offset += 1
					}
					if len(data) > 5+offset {
						flag = data[5+offset]
					}
				} else if label == "gpt" {
					if IsPartedFsString(data[4]) {
						fs = data[4]
						offset += 1
					}
					if len(data) > 4+offset {
						disktype = data[4+offset]
					}
					if len(data) > 4+offset+1 {
						flag = data[4+offset+1]
					}
				}
			}
			var bootable = ""
			if len(flag) > 0 && strings.Index(flag, "boot") >= 0 {
				bootable = "true"
			}
			parts = append(parts, []string{idx, bootable, start, end, count, disktype, fs,
				devname})
		}
	}
	return parts, label
}

func Mkpartition(imagePath, fsFormat string) error {
	var (
		parted    = "parted"
		labelType = "gpt"
		diskType  = fileutils2.FsFormatToDiskType(fsFormat)
	)

	if len(diskType) == 0 {
		return fmt.Errorf("Unknown fsFormat %s", fsFormat)
	}

	// 创建一个新磁盘分区表类型, ex: mbr gpt msdos ...
	output, err := procutils.NewCommand(parted, "-s", imagePath, "mklabel", labelType).Output()
	if err != nil {
		log.Errorf("mklabel %s %s error: %s, %s", imagePath, fsFormat, err, output)
		return errors.Wrapf(err, "parted mklabel failed: %s", output)
	}

	// 创建一个part-type类型的分区, part-type可以是："primary", "logical", "extended"
	// 如果指定fs-type(即diskType)则在创建分区的同时进行格式化
	output, err = procutils.NewCommand(parted, "-s", "-a", "cylinder",
		imagePath, "mkpart", "primary", diskType, "0", "100%").Output()
	if err != nil {
		log.Errorf("mkpart %s %s error: %s, %s", imagePath, fsFormat, err, output)
		return errors.Wrapf(err, "parted mkpart failed: %s", output)
	}
	return nil
}

func ext4UsageType(path string) string {
	out, err := procutils.NewCommand("blockdev", "--getsize64", path).Output()
	if err != nil {
		log.Errorf("failed get blockdev %s size: %s", path, err)
		return ""
	}
	size, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		log.Errorf("failed parse blocksize %s", out)
		return ""
	}
	if size > ext4UsageTypeLargefileSize {
		// node_ratio 1M
		return "largefile"
	} else if size > ext4UsageTypeHugefileSize {
		// node_ratio 64K
		return "huge"
	}
	return ""
}

func FormatPartition(path, fs, uuid string) error {
	var cmd, cmdUuid []string
	switch {
	case fs == "swap":
		cmd = []string{"mkswap", "-U", uuid}
	case fs == "ext2":
		cmd = []string{"mkfs.ext2"}
		cmdUuid = []string{"tune2fs", "-U", uuid}
	case fs == "ext3":
		cmd = []string{"mkfs.ext3"}
		cmdUuid = []string{"tune2fs", "-U", uuid}
	case fs == "ext4":
		cmd = []string{"mkfs.ext4", "-O", "^64bit", "-E", "lazy_itable_init=1"}
		/*
			// see /etc/mke2fs.conf, default inode_ratio is 16384
			If  this option is is not specified, mke2fs will pick a single default usage type based on the size
			of the filesystem to be created.  If the filesystem size is less than  or  equal  to  3  megabytes,
			mke2fs will use the filesystem type floppy.  If the filesystem size is greater than 3 but less than
			or equal to 512 megabytes, mke2fs(8) will use the filesystem type small.  If the filesystem size is
			greater  than or equal to 4 terabytes but less than 16 terabytes, mke2fs(8) will use the filesystem
			type big.  If the filesystem size is greater than or equal to 16 terabytes, mke2fs(8) will use  the
			filesystem type huge.  Otherwise, mke2fs(8) will use the default filesystem type default.
		*/
		if usageType := ext4UsageType(path); usageType != "" {
			cmd = append(cmd, "-T", usageType)
		}

		cmdUuid = []string{"tune2fs", "-U", uuid}
	case fs == "ext4dev":
		cmd = []string{"mkfs.ext4dev", "-E", "lazy_itable_init=1"}
		cmdUuid = []string{"tune2fs", "-U", uuid}
	case strings.HasPrefix(fs, "fat"):
		cmd = []string{"mkfs.msdos"}
	// #case fs == "ntfs":
	// #    cmd = []string{"/sbin/mkfs.ntfs"}
	case fs == "xfs":
		cmd = []string{"mkfs.xfs", "-f", "-m", "crc=0", "-i", "projid32bit=0", "-n", "ftype=1"}
		cmdUuid = []string{"xfs_admin", "-U", uuid}
	}

	if len(cmd) > 0 {
		var cmds = cmd
		cmds = append(cmds, path)
		if output, err := procutils.NewCommand(cmds[0], cmds[1:]...).Output(); err != nil {
			log.Errorf("%v failed: %s, %s", cmds, err, output)
			return errors.Wrapf(err, "format partition failed %s", output)
		}
		if len(cmdUuid) > 0 {
			cmds = cmdUuid
			cmds = append(cmds, path)
			if output, err := procutils.NewCommand(cmds[0], cmds[1:]...).Output(); err != nil {
				log.Errorf("%v failed: %s, %s", cmds, err, output)
				return errors.Wrapf(err, "format partition set uuid failed %s", output)
			}
		}
		return nil
	}
	return fmt.Errorf("Unknown fs %s", fs)
}

func DetectIsUEFISupport(rootfs fsdriver.IRootFsDriver, partitions []fsdriver.IDiskPartition) bool {
	for i := 0; i < len(partitions); i++ {
		if partitions[i].IsMounted() {
			if rootfs.DetectIsUEFISupport(partitions[i]) {
				return true
			}
		} else {
			if partitions[i].Mount() {
				support := rootfs.DetectIsUEFISupport(partitions[i])
				if err := partitions[i].Umount(); err != nil {
					log.Errorf("failed umount %s: %s", partitions[i].GetPartDev(), err)
				}
				if support {
					return true
				}
			}
		}
	}
	return false
}

func MountRootfs(readonly bool, partitions []fsdriver.IDiskPartition) (fsdriver.IRootFsDriver, error) {
	errs := []error{}
	for i := 0; i < len(partitions); i++ {
		log.Infof("detect partition %s", partitions[i].GetPartDev())
		mountFunc := partitions[i].Mount
		if readonly {
			mountFunc = partitions[i].MountPartReadOnly
		}
		if mountFunc() {
			fs, err := guestfs.DetectRootFs(partitions[i])
			if err == nil {
				log.Infof("Use rootfs %s, partition %s", fs, partitions[i].GetPartDev())
				return fs, nil
			}
			errs = append(errs, err)
			if err := partitions[i].Umount(); err != nil {
				log.Errorf("failed umount %s: %s", partitions[i].GetPartDev(), err)
			}
		}
	}
	if len(partitions) == 0 {
		return nil, errors.Wrap(errors.ErrNotFound, "not found any partition")
	}
	var err error = errors.ErrNotFound
	if len(errs) > 0 {
		err = errors.Wrapf(errors.ErrNotFound, errors.NewAggregate(errs).Error())
	}
	return nil, err
}

func DeployGuestfs(d deploy_iface.IDeployer, req *apis.DeployParams) (res *apis.DeployGuestFsResponse, err error) {
	root, err := d.MountRootfs(false)
	if err != nil {
		if errors.Cause(err) == errors.ErrNotFound && req.DeployInfo.IsInit {
			// if init deploy, ignore no partition error
			log.Errorf("disk.MountRootfs not found partition, not init, quit")
			return nil, nil
		}
		log.Errorf("Failed mounting rootfs for %s disk: %s", req.GuestDesc.Hypervisor, err)
		return nil, err
	}
	defer d.UmountRootfs(root)
	ret, err := guestfs.DoDeployGuestFs(root, req.GuestDesc, req.DeployInfo)
	if err != nil {
		log.Errorf("guestfs.DoDeployGuestFs fail %s", err)
		return nil, err
	}
	if ret == nil {
		log.Errorf("guestfs.DoDeployGuestFs return empty results")
		return nil, nil
	}
	return ret, nil
}

func ResizeFs(d deploy_iface.IDeployer, diskId string) (*apis.Empty, error) {
	unmount := func(root fsdriver.IRootFsDriver) error {
		err := d.UmountRootfs(root)
		if err != nil {
			return errors.Wrap(err, "unmount rootfs")
		}
		return nil
	}

	var rootPartDev string
	root, err := d.MountRootfs(false)
	if err != nil && errors.Cause(err) != errors.ErrNotFound {
		return new(apis.Empty), errors.Wrapf(err, "disk.MountRootfs")
	} else if err == nil {
		rootPartDev = root.GetPartition().GetPartDev()
		if !root.IsResizeFsPartitionSupport() {
			err := unmount(root)
			if err != nil {
				return new(apis.Empty), err
			}
			return new(apis.Empty), errors.ErrNotSupported
		}

		// must umount rootfs before resize partition
		err = unmount(root)
		if err != nil {
			return new(apis.Empty), err
		}
	}
	err = d.ResizePartition(diskId, rootPartDev)
	if err != nil {
		return new(apis.Empty), errors.Wrap(err, "resize disk partition")
	}
	return new(apis.Empty), nil
}

func FormatFs(d deploy_iface.IDeployer, req *apis.FormatFsParams) (*apis.Empty, error) {
	err := d.MakePartition(req.FsFormat)
	if err != nil {
		return new(apis.Empty), errors.Wrap(err, "MakePartition")
	}
	err = d.FormatPartition(req.FsFormat, req.Uuid)
	if err != nil {
		return new(apis.Empty), errors.Wrap(err, "FormatPartition")
	}
	return new(apis.Empty), nil
}

func SaveToGlance(d deploy_iface.IDeployer, req *apis.SaveToGlanceParams) (*apis.SaveToGlanceResponse, error) {
	var (
		osInfo  string
		relInfo *apis.ReleaseInfo
	)

	ret := &apis.SaveToGlanceResponse{
		OsInfo:      osInfo,
		ReleaseInfo: relInfo,
	}

	root, err := d.MountRootfs(false)
	if err != nil {
		if errors.Cause(err) == errors.ErrNotFound {
			return ret, nil
		}
		return ret, errors.Wrapf(err, "MountKvmRootfs")
	}
	defer d.UmountRootfs(root)

	osInfo = root.GetOs()
	relInfo = root.GetReleaseInfo(root.GetPartition())
	if req.Compress {
		err = root.PrepareFsForTemplate(root.GetPartition())
		if err != nil {
			log.Errorf("PrepareFsForTemplate %s", err)
		}
	}
	if req.Compress {
		d.Zerofree()
	}
	return ret, err
}

func ProbeImageInfo(d deploy_iface.IDeployer) (*apis.ImageInfo, error) {
	// Fsck is executed during mount
	rootfs, err := d.MountRootfs(false)
	if err != nil {
		if errors.Cause(err) == errors.ErrNotFound {
			return new(apis.ImageInfo), nil
		}
		return new(apis.ImageInfo), errors.Wrapf(err, "d.MountKvmRootfs")
	}
	partition := rootfs.GetPartition()
	imageInfo := &apis.ImageInfo{
		OsInfo:               rootfs.GetReleaseInfo(partition),
		OsType:               rootfs.GetOs(),
		IsLvmPartition:       d.IsLVMPartition(),
		IsReadonly:           partition.IsReadonly(),
		IsInstalledCloudInit: rootfs.IsCloudinitInstall(),
	}
	d.UmountRootfs(rootfs)

	// In case of deploy driver is guestfish, we can't mount
	// multi partition concurrent, so we need umount rootfs first
	imageInfo.IsUefiSupport = d.DetectIsUEFISupport(rootfs)
	imageInfo.PhysicalPartitionType = partition.GetPhysicalPartitionType()
	log.Infof("ProbeImageInfo response %s", imageInfo)
	return imageInfo, nil
}
