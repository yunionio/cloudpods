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
	"path"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/hostman/diskutils/fsutils/driver"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/xfsutils"
)

// use proc driver do resizefs
func ResizeDiskFs(diskPath string, sizeMb int) error {
	fsutilDriver := NewFsutilDriver(driver.NewProcDriver())
	return fsutilDriver.ResizeDiskFs(diskPath, sizeMb)
}

func (d *SFsutilDriver) ResizeDiskFs(diskPath string, sizeMb int) error {
	fPath, fs, err := d.ResizeDiskPartition(diskPath, sizeMb)
	if err != nil {
		return err
	}
	err, _ = d.ResizePartitionFs(fPath, fs, false, false)
	if err != nil {
		return errors.Wrapf(err, "resize fs %s", fs)
	}
	return nil
}

func (d *SFsutilDriver) ResizeDiskWithDiskId(diskId string, rootPartDev string, onlineResize bool) error {
	// find partition need resize
	resizeDev, err := d.GetResizeDevBySerial(diskId)
	if err != nil {
		return err
	}
	if resizeDev == "" {
		log.Errorf("failed find disk serial %s", diskId)
		return nil
	}
	partDev, fsType, err := d.ResizeDiskPartition(resizeDev, 0)
	if err != nil {
		return err
	}
	if partDev == "" || fsType == "" {
		if fsType == "" && partDev != "" {
			// fsType empty and partDev not empty is lvm device
			resizeDev = partDev
		}

		if !d.IsLvmPvDevice(resizeDev) {
			fsType = d.GetFsFormat(resizeDev)
			err, _ := d.ResizePartitionFs(resizeDev, fsType, false, onlineResize)
			return err
		}
		if err := d.Pvresize(resizeDev); err != nil {
			return err
		}

		vg := d.GetVgOfPvDevice(resizeDev)
		if vg == "" {
			return nil
		}
		lvs, err := d.GetVgLvs(vg)
		if err != nil {
			log.Errorf("failed get vg lvs %s: %s", vg, err)
		}
		if len(lvs) == 0 {
			log.Infof("disk %s has no lv, skip resize", diskId)
			return nil
		}

		var resizeLv string
		if rootPartDev != "" {
			for i := range lvs {
				if lvs[i].LvPath == rootPartDev {
					resizeLv = rootPartDev
					break
				}
			}
		}
		if resizeLv == "" {
			if len(lvs) != 1 {
				log.Errorf("disk %s has multi lv and no rootfs, skip resize partition", diskId)
				return nil
			} else {
				resizeLv = lvs[0].LvPath
			}
		}
		err = d.ExtendLv(resizeLv)
		if err != nil {
			return err
		}
		fsType = d.GetFsFormat(resizeLv)
		err, _ = d.ResizePartitionFs(resizeLv, fsType, false, onlineResize)
		return err
	} else {
		err, _ = d.ResizePartitionFs(partDev, fsType, false, onlineResize)
		return err
	}
}

func (d *SFsutilDriver) GetResizeDevBySerial(diskId string) (string, error) {
	out, err := d.Exec("sh", "-c", "lsblk -d -o NAME,SERIAL  | awk 'NR>1'")
	if err != nil {
		return "", errors.Wrapf(err, "ResizePartition lsblk %s", err)
	}
	lines := strings.Split(string(out), "\n")
	diskSerial := strings.ReplaceAll(diskId, "-", "")
	resizeDev := ""
	for i := range lines {
		segs := strings.Fields(lines[i])
		if len(segs) == 0 {
			continue
		}
		log.Errorf("segs %v", segs)
		if len(segs) == 1 {
			// fetch vpd 80 serial id
			ret, err := d.Exec("sg_inq", "-u", "-p", "0x80", path.Join("/dev/", segs[0]))
			if err != nil {
				log.Infof("failed exec sg_inq: %s %s", ret, err)
				continue
			}
			serialStr := strings.TrimSpace(string(ret))
			serialSegs := strings.Split(serialStr, "=")
			log.Errorf("serial segs %v", serialSegs)
			if len(serialSegs) == 2 && serialSegs[1] == diskSerial {
				resizeDev = path.Join("/dev/", segs[0])
				break
			}
		}
		devName, serial := segs[0], segs[1]
		log.Infof("lsblk segs: %s %s |", devName, serial)
		if strings.HasPrefix(diskSerial, serial) {
			resizeDev = path.Join("/dev/", devName)
			break
		}
	}
	return resizeDev, nil
}

func (d *SFsutilDriver) IsLvmPvDevice(device string) bool {
	return d.Run("pvs", device) == nil
}

func (d *SFsutilDriver) Pvresize(device string) error {
	out, err := d.Exec("partprobe")
	if err != nil {
		return errors.Wrapf(err, "failed resize pv partprobe %s", out)
	}
	out, err = d.Exec("pvscan")
	if err != nil {
		return errors.Wrapf(err, "failed resize pv pvscan %s", out)
	}
	out, err = d.Exec("pvresize", device)
	if err != nil {
		return errors.Wrapf(err, "failed resize pv %s", out)
	}
	return nil
}

func (d *SFsutilDriver) GetVgOfPvDevice(device string) string {
	out, err := d.Exec("pvs", "--noheadings", "-o", "vg_name", device)
	if err != nil {
		log.Errorf("get vg from pv %s device failed: %s %s", device, out, err)
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (d *SFsutilDriver) ResizeDiskPartition(diskPath string, sizeMb int) (string, string, error) {
	var cmds = []string{"parted", "-a", "none", "-s", diskPath, "--", "unit", "s", "print"}
	lines, err := d.Exec(cmds[0], cmds[1:]...)
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
			return "", "", nil
		}
		log.Errorf("resize disk fs fail, output: %s , error: %s", lines, err)
		return "", "", err
	}
	parts, label := ParseDiskPartition(diskPath, strings.Split(string(lines), "\n"))
	log.Infof("Parts: %v label: %s", parts, label)
	if label == "gpt" {
		retCode, stdout, stderr, e := d.ExecInputWait("gdisk", []string{diskPath}, []string{"r", "e", "Y", "w", "Y", "Y"})
		if e != nil {
			return "", "", errors.Wrap(e, "gdisk exec failed")
		}
		log.Infof("gdisk: %s %s", stdout, stderr)
		if retCode != 1 && retCode != 0 {
			return "", "", errors.Errorf("Exit Code %d: %s\n%s", retCode, stdout, stderr)
		}
	}
	if len(parts) > 0 && (label == "gpt" ||
		(label == "msdos" && parts[len(parts)-1][5] == "primary")) {
		var part = parts[len(parts)-1]
		if part[5] == "lvm" || IsSupportResizeFs(part[6]) {
			// growpart script replace parted resizepart
			output, err := d.Exec("growpart", diskPath, part[0])
			if err != nil {
				return "", "", errors.Wrapf(err, "growpart failed %s", output)
			}
			return part[7], part[6], nil
		}
	}
	return "", "", nil
}

func (d *SFsutilDriver) ResizePartitionFs(fpath, fs string, raiseError, onlineResize bool) (error, bool) {
	log.Errorf("ResizePartitionFs fstype %s", fs)
	if len(fs) == 0 {
		return nil, false
	}
	var (
		cmds     = [][]string{}
		uuids, _ = d.GetDevUuid(fpath)
	)
	if strings.HasPrefix(fs, "linux-swap") {
		if v, ok := uuids["UUID"]; ok {
			cmds = [][]string{{"mkswap", "-U", v, fpath}}
		} else {
			cmds = [][]string{{"mkswap", fpath}}
		}
	} else if strings.HasPrefix(fs, "ext") {
		if !onlineResize {
			if !d.FsckExtFs(fpath) {
				if raiseError {
					return fmt.Errorf("Failed to fsck ext fs %s", fpath), false
				} else {
					return nil, false
				}
			}
		}
		cmds = [][]string{{"resize2fs", fpath}}
	} else if fs == "xfs" {
		uuid := uuids["UUID"]
		if len(uuid) > 0 {
			xfsutils.LockXfsPartition(uuid)
			defer xfsutils.UnlockXfsPartition(uuid)
		}
		if !onlineResize {
			var tmpPoint = fmt.Sprintf("/tmp/%s", strings.Replace(fpath, "/", "_", -1))
			if _, err := d.Exec("mountpoint", tmpPoint); err == nil {
				output, err := d.Exec("umount", "-f", tmpPoint)
				if err != nil {
					log.Errorf("failed umount %s: %s, %s", tmpPoint, err, output)
					return err, false
				}
			}
			d.FsckXfsFs(fpath)
			cmds = [][]string{{"mkdir", "-p", tmpPoint},
				{"mount", fpath, tmpPoint},
				{"sleep", "2"},
				{"xfs_growfs", tmpPoint},
				{"sleep", "2"},
				{"umount", tmpPoint},
				{"sleep", "2"},
				{"rm", "-fr", tmpPoint}}
		} else {
			cmds = [][]string{{"xfs_growfs", fpath}}
		}
	} else if fs == "ntfs" {
		// the following cmds may cause disk damage on Windows 10 with new version of NTFS
		// comment out the following codes only impact Windows 2003
		// as windows 2003 deprecated, so choose to sacrifies windows 2003
		// cmds = [][]string{{"ntfsresize", "-c", fpath}, {"ntfsresize", "-P", "-f", fpath}}
	}

	if len(cmds) > 0 {
		for _, cmd := range cmds {
			output, err := d.Exec(cmd[0], cmd[1:]...)
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

func (d *SFsutilDriver) FsckExtFs(fpath string) bool {
	log.Debugf("Exec command: %v", []string{"e2fsck", "-f", "-p", fpath})
	retCode, stdout, stderr, err := d.ExecInputWait("e2fsck", []string{"-f", "-p", fpath}, nil)
	if err != nil {
		log.Errorf("exec e2fsck failed %s", err)
		return false
	}
	if retCode < 4 {
		return true
	}
	log.Errorf("failed e2fsck retcode %d %s %s", retCode, stdout, stderr)
	return false
}

// https://bugs.launchpad.net/ubuntu/+source/xfsprogs/+bug/1718244
// use xfs_repair -n instead
func (d *SFsutilDriver) FsckXfsFs(fpath string) bool {
	if output, err := d.Exec("xfs_check", fpath); err != nil {
		log.Errorf("xfs_check failed: %s, %s, try xfs_repair -n <dev> instead", err, output)
		if output, err := procutils.NewCommand("xfs_repair", "-n", fpath).Output(); err != nil {
			log.Errorf("xfs_repair -n dev failed: %s, %s", err, output)
			// repair the xfs
			d.Exec("xfs_repair", fpath)
			return false
		}
	}
	return true
}

func GetFsFormat(diskPath string) string {
	fsutilDriver := NewFsutilDriver(driver.NewProcDriver())
	return fsutilDriver.GetFsFormat(diskPath)
}

func (d *SFsutilDriver) GetFsFormat(diskPath string) string {
	ret, err := d.Exec("blkid", "-o", "value", "-s", "TYPE", diskPath)
	if err != nil {
		log.Errorf("failed exec blkid of dev %s: %s, %s", diskPath, err, ret)
		return ""
	}
	var res string
	for _, line := range strings.Split(string(ret), "\n") {
		res += line
	}
	return res
}

func (d *SFsutilDriver) GetDevUuid(dev string) (map[string]string, error) {
	lines, err := d.Exec("blkid", dev)
	if err != nil {
		log.Errorf("GetDevUuid %s error: %v", dev, err)
		return map[string]string{}, errors.Wrapf(err, "blkid")
	}
	for _, l := range strings.Split(string(lines), "\n") {
		if strings.HasPrefix(l, dev) {
			var ret = map[string]string{}
			for _, part := range strings.Split(l, " ") {
				data := strings.Split(part, "=")
				if len(data) == 2 && strings.HasSuffix(data[0], "UUID") {
					if data[1][0] == '"' || data[1][0] == '\'' {
						ret[data[0]] = data[1][1 : len(data[1])-1]
					} else {
						ret[data[0]] = data[1]
					}
				}
			}
			return ret, nil
		}
	}
	return map[string]string{}, nil
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
