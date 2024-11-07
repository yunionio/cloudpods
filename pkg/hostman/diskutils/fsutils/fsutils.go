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
	"io"
	"io/ioutil"
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
	"yunion.io/x/onecloud/pkg/util/xfsutils"
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

func GetDevSector512Count(dev string) int {
	sizeStr, _ := fileutils2.FileGetContents(fmt.Sprintf("/sys/block/%s/size", dev))
	sizeStr = strings.Trim(sizeStr, "\n")
	size, _ := strconv.Atoi(sizeStr)
	return size
}

func ResizeDiskFs(diskPath string, sizeMb int, mounted bool) error {
	var cmds = []string{"parted", "-a", "none", "-s", diskPath, "--", "unit", "s", "print"}
	lines, err := procutils.NewCommand(cmds[0], cmds[1:]...).Output()
	if err != nil {
		hasPartTable := func() bool {
			for _, line := range strings.Split(string(lines), "\n") {
				if strings.Contains(line, "Partition Table") {
					return true
				}
			}
			return false
		}
		if hasPartTable() {
			return nil
		}
		log.Errorf("resize disk fs fail, output: %s , error: %s", lines, err)
		return err
	}
	parts, label := ParseDiskPartition(diskPath, strings.Split(string(lines), "\n"))
	log.Infof("Parts: %v label: %s", parts, label)
	if label == "gpt" {
		proc := procutils.NewCommand("gdisk", diskPath)
		stdin, err := proc.StdinPipe()
		if err != nil {
			return err
		}
		defer stdin.Close()

		outb, err := proc.StdoutPipe()
		if err != nil {
			return err
		}
		defer outb.Close()

		errb, err := proc.StderrPipe()
		if err != nil {
			return err
		}
		defer errb.Close()

		if err := proc.Start(); err != nil {
			return err
		}
		for _, s := range []string{"r", "e", "Y", "w", "Y", "Y"} {
			io.WriteString(stdin, fmt.Sprintf("%s\n", s))
		}
		stdoutPut, err := ioutil.ReadAll(outb)
		if err != nil {
			return err
		}
		stderrOutPut, err := ioutil.ReadAll(errb)
		if err != nil {
			return err
		}
		log.Infof("gdisk: %s %s", stdoutPut, stderrOutPut)
		if err = proc.Wait(); err != nil {
			if status, succ := proc.GetExitStatus(err); succ {
				if status != 1 {
					return err
				}
			} else {
				return err
			}
		}
	}
	if len(parts) > 0 && (label == "gpt" ||
		(label == "msdos" && parts[len(parts)-1][5] == "primary")) {
		var part = parts[len(parts)-1]
		if IsSupportResizeFs(part[6]) {
			// growpart script replace parted resizepart
			output, err := procutils.NewCommand("growpart", diskPath, part[0]).Output()
			if err != nil {
				return errors.Wrapf(err, "growpart failed %s", output)
			}
			err, _ = ResizePartitionFs(part[7], part[6], false, mounted)
			if err != nil {
				return errors.Wrapf(err, "resize fs %s", part[6])
			}
		}
	}
	return nil
}

func IsSupportResizeFs(fs string) bool {
	if strings.HasPrefix(fs, "linux-swap") {
		return true
	} else if strings.HasPrefix(fs, "ext") {
		return true
	} else if fs == "xfs" {
		return true
	}
	return false
}

func ResizePartitionFs(fpath, fs string, raiseError, mounted bool) (error, bool) {
	if len(fs) == 0 {
		return nil, false
	}
	var (
		cmds     = [][]string{}
		uuids, _ = fileutils2.GetDevUuid(fpath)
	)
	if strings.HasPrefix(fs, "linux-swap") {
		if v, ok := uuids["UUID"]; ok {
			cmds = [][]string{{"mkswap", "-U", v, fpath}}
		} else {
			cmds = [][]string{{"mkswap", fpath}}
		}
	} else if strings.HasPrefix(fs, "ext") {
		if !mounted {
			if !FsckExtFs(fpath) {
				if raiseError {
					return fmt.Errorf("Failed to fsck ext fs %s", fpath), false
				} else {
					return nil, false
				}
			}
		}

		cmds = [][]string{{"resize2fs", fpath}}
	} else if fs == "xfs" {
		var tmpPoint = fmt.Sprintf("/tmp/%s", strings.Replace(fpath, "/", "_", -1))
		if _, err := procutils.NewCommand("mountpoint", tmpPoint).Output(); err == nil {
			output, err := procutils.NewCommand("umount", "-f", tmpPoint).Output()
			if err != nil {
				log.Errorf("failed umount %s: %s, %s", tmpPoint, err, output)
				return err, false
			}
		}
		FsckXfsFs(fpath)
		uuid := uuids["UUID"]
		if len(uuid) > 0 {
			xfsutils.LockXfsPartition(uuid)
			defer xfsutils.UnlockXfsPartition(uuid)
		}
		cmds = [][]string{{"mkdir", "-p", tmpPoint},
			{"mount", fpath, tmpPoint},
			{"sleep", "2"},
			{"xfs_growfs", tmpPoint},
			{"sleep", "2"},
			{"umount", tmpPoint},
			{"sleep", "2"},
			{"rm", "-fr", tmpPoint}}
	} else if fs == "ntfs" {
		// the following cmds may cause disk damage on Windows 10 with new version of NTFS
		// comment out the following codes only impact Windows 2003
		// as windows 2003 deprecated, so choose to sacrifies windows 2003
		// cmds = [][]string{{"ntfsresize", "-c", fpath}, {"ntfsresize", "-P", "-f", fpath}}
	}

	if len(cmds) > 0 {
		for _, cmd := range cmds {
			output, err := procutils.NewCommand(cmd[0], cmd[1:]...).Output()
			if err != nil {
				log.Errorf("resize partition: %s, %s", err, output)
				if raiseError {
					return err, false
				} else {
					return nil, false
				}
			}
		}
	}
	return nil, true
}

func FsckExtFs(fpath string) bool {
	log.Debugf("Exec command: %v", []string{"e2fsck", "-f", "-p", fpath})
	cmd := procutils.NewCommand("e2fsck", "-f", "-p", fpath)

	out, err := cmd.Output()
	if err != nil {
		log.Errorf("e2fsck failed %s: %s", err, out)
		if status, ok := cmd.GetExitStatus(err); ok {
			log.Errorf("e2fsck exit status %d", status)
			if status < 4 {
				return true
			} else {
				return false
			}
		} else {
			return false
		}
	}
	return true
}

// https://bugs.launchpad.net/ubuntu/+source/xfsprogs/+bug/1718244
// use xfs_repair -n instead
func FsckXfsFs(fpath string) bool {
	if output, err := procutils.NewCommand("xfs_check", fpath).Output(); err != nil {
		log.Errorf("xfs_check failed: %s, %s, try xfs_repair -n <dev> instead", err, output)
		if output, err := procutils.NewCommand("xfs_repair", "-n", fpath).Output(); err != nil {
			log.Errorf("xfs_repair -n dev failed: %s, %s", err, output)
			// repair the xfs
			procutils.NewCommand("xfs_repair", fpath).Output()
			return false
		}
	}
	return true
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

func FormatPartition(path, fs, uuid string, fsFeatures *apis.FsFeatures) error {
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
		feature := "^64bit"
		extendOpts := "lazy_itable_init=1"
		if fsFeatures != nil && fsFeatures.Ext4 != nil {
			if fsFeatures.Ext4.CaseInsensitive {
				feature = fmt.Sprintf("%s,casefold,project,quota", feature)
				//feature = fmt.Sprintf("casefold,project,quota")
				extendOpts = fmt.Sprintf("%s,nodiscard,lazy_journal_init=1,encoding_flags=strict,encoding=utf8-12.1", extendOpts)
			}
		}
		cmd = []string{"mkfs.ext4", "-O", feature, "-E", extendOpts}
		log.Infof("===========format partion cmd: %#v", cmd)
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
		cmd = []string{"mkfs.xfs", "-f", "-m", "crc=0", "-i", "projid32bit=0", "-n", "ftype=0"}
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

func ResizeFs(d deploy_iface.IDeployer) (*apis.Empty, error) {
	unmount := func(root fsdriver.IRootFsDriver) error {
		err := d.UmountRootfs(root)
		if err != nil {
			return errors.Wrap(err, "unmount rootfs")
		}
		return nil
	}

	root, err := d.MountRootfs(false)
	if err != nil && errors.Cause(err) != errors.ErrNotFound {
		return new(apis.Empty), errors.Wrapf(err, "disk.MountRootfs")
	} else if err == nil {
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

	err = d.ResizePartition()
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
	err = d.FormatPartition(req.FsFormat, req.Uuid, req.FsFeatures)
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
