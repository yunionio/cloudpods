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

package fileutils2

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"

	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/regutils2"
)

func Cleandir(sPath string, keepdir bool) error {
	if f, _ := os.Lstat(sPath); f == nil || f.Mode()&os.ModeSymlink == os.ModeSymlink {
		return nil
	}
	files, _ := ioutil.ReadDir(sPath)
	for _, file := range files {
		fp := path.Join(sPath, file.Name())
		if f, _ := os.Lstat(fp); f.Mode()&os.ModeSymlink == os.ModeSymlink {
			if !keepdir {
				if err := os.Remove(fp); err != nil {
					return err
				}
			}
		} else if f.IsDir() {
			Cleandir(fp, keepdir)
			if !keepdir {
				if err := os.Remove(fp); err != nil {
					return err
				}
			}
		} else {
			if err := os.Remove(fp); err != nil {
				return err
			}
		}
	}
	return nil
}

// TODO: test
func Zerofiles(sPath string) error {
	f, err := os.Lstat(sPath)
	switch {
	case err != nil:
		return err
	case f.Mode()&os.ModeSymlink == os.ModeSymlink:
		// islink
		return nil
	case f.Mode().IsRegular():
		return FilePutContents(sPath, "", false)
	case f.Mode().IsDir():
		files, err := ioutil.ReadDir(sPath)
		if err != nil {
			return err
		}
		for _, file := range files {
			if file.Mode()&os.ModeSymlink == os.ModeSymlink {
				continue
			} else if file.Mode().IsRegular() {
				if err := FilePutContents(path.Join(sPath, file.Name()), "", false); err != nil {
					return err
				}
			} else if file.Mode().IsDir() {
				return Zerofiles(path.Join(sPath, file.Name()))
			}
		}
	}
	return nil
}

func FileSetContents(filename string, content string) error {
	return FilePutContents(filename, content, false)
}

func FileAppendContents(filename string, content string) error {
	return FilePutContents(filename, content, true)
}

func FilePutContents(filename string, content string, modAppend bool) error {
	var mode = os.O_WRONLY | os.O_CREATE
	if modAppend {
		mode = mode | os.O_APPEND
	} else {
		mode = mode | os.O_TRUNC
	}
	fd, err := os.OpenFile(filename, mode, 0644)
	if err != nil {
		return err
	}
	defer fd.Close()
	_, err = fd.WriteString(content)
	return err
}

func IsBlockDevMounted(dev string) bool {
	devPath := "/dev/" + dev
	mounts, err := procutils.NewCommand("mount").Run()
	if err != nil {
		return false
	}
	for _, s := range strings.Split(string(mounts), "\n") {
		if strings.HasPrefix(s, devPath) {
			return true
		}
	}
	return false
}

func IsBlockDeviceUsed(dev string) bool {
	if strings.HasPrefix(dev, "/dev/") {
		dev = dev[strings.LastIndex(dev, "/")+1:]
	}
	devStr := fmt.Sprintf(" %s\n", dev)
	devs, _ := procutils.NewCommand("cat", "/proc/partitions").Run()
	if idx := strings.Index(string(devs), devStr); idx > 0 {
		return true
	}
	return false
}

func ChangeAllBlkdevsParams(params map[string]string) {
	if _, err := os.Stat("/sys/block"); !os.IsNotExist(err) {
		blockDevs, err := ioutil.ReadDir("/sys/block")
		if err != nil {
			log.Errorln(err)
			return
		}
		for _, b := range blockDevs {
			if IsBlockDevMounted(b.Name()) {
				for k, v := range params {
					ChangeBlkdevParameter(b.Name(), k, v)
				}
			}
		}
	}
}

func ChangeBlkdevParameter(dev, key, value string) {
	p := path.Join("/sys/block", dev, key)
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		err = FilePutContents(p, value, false)
		if err != nil {
			log.Errorf("Fail to set %s of %s to %s:%s", key, dev, value, err)
		}
		log.Infof("Set %s of %s to %s", key, dev, value)
	}
}

func FileGetContents(file string) (string, error) {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func GetFsFormat(diskPath string) string {
	ret, err := procutils.NewCommand("blkid", "-o", "value", "-s", "TYPE", diskPath).Run()
	if err != nil {
		return ""
	}
	var res string
	for _, line := range strings.Split(string(ret), "\n") {
		res += line
	}
	return res
}

func CleanFailedMountpoints() {
	var mtfile = "/etc/mtab"
	if _, err := os.Stat(mtfile); os.IsNotExist(err) {
		mtfile = "/proc/mounts"
	}
	f, err := os.Open(mtfile)
	if err != nil {
		log.Errorf("CleanFailedMountpoints error: %s", err)
	}
	reader := bufio.NewReader(f)
	line, _, err := reader.ReadLine()
	for err != nil {
		m := strings.Split(string(line), " ")
		if len(m) > 1 {
			mp := m[1]
			if _, err := os.Stat(mp); os.IsNotExist(err) {
				log.Warningf("Mount point %s not exists", mp)
			}
			procutils.NewCommand("umount", mp).Run()
		}
	}
}

type HostsFile map[string][]string

func (hf HostsFile) Parse(content string) {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		data := regexp.MustCompile(`\s+`).Split(line, -1)
		for len(data) > 0 && data[len(data)-1] == "" {
			data = data[:len(data)-1]
		}
		if len(data) > 1 {
			hf[data[0]] = data[1:]
		}
	}
}

func (hf HostsFile) Add(name string, value ...string) {
	hf[name] = value
}

func (hf HostsFile) String() string {
	var ret = ""
	for k, v := range hf {
		if len(v) > 0 {
			ret += fmt.Sprintf("%s\t%s\n", k, strings.Join(v, "\t"))
		}
	}
	return ret
}

func Writable(path string) bool {
	return unix.Access(path, unix.W_OK) == nil
}

func FsFormatToDiskType(fsFormat string) string {
	switch {
	case fsFormat == "swap":
		return "linux-swap"
	case strings.HasPrefix(fsFormat, "ext") || fsFormat == "xfs":
		return "ext2"
	case strings.HasPrefix(fsFormat, "fat"):
		return "fat32"
	case fsFormat == "ntfs":
		return fsFormat
	default:
		return ""
	}
}

func Mkpartition(imagePath, fsFormat string) error {
	var (
		parted    = "/sbin/parted"
		labelType = "gpt"
		diskType  = FsFormatToDiskType(fsFormat)
	)

	if len(diskType) == 0 {
		return fmt.Errorf("Unknown fsFormat %s", fsFormat)
	}

	// 创建一个新磁盘分区表类型, ex: mbr gpt msdos ...
	_, err := procutils.NewCommand(parted, "-s", imagePath, "mklabel", labelType).Run()
	if err != nil {
		log.Errorf("mklabel %s %s error %s", imagePath, fsFormat, err)
		return err
	}

	// 创建一个part-type类型的分区, part-type可以是："primary", "logical", "extended"
	// 如果指定fs-type(即diskType)则在创建分区的同时进行格式化
	_, err = procutils.NewCommand(parted, "-s", "-a", "cylinder",
		imagePath, "mkpart", "primary", diskType, "0", "100%").Run()
	if err != nil {
		log.Errorf("mkpart %s %s error %s", imagePath, fsFormat, err)
		return err
	}
	return nil
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
		cmdUuid = []string{"tune2fs", "-U", uuid}
	case fs == "ext4dev":
		cmd = []string{"mkfs.ext4dev", "-E", "lazy_itable_init=1"}
		cmdUuid = []string{"tune2fs", "-U", uuid}
	case strings.HasPrefix(fs, "fat"):
		cmd = []string{"mkfs.msdos"}
	// #case fs == "ntfs":
	// #    cmd = []string{"/sbin/mkfs.ntfs"}
	case fs == "xfs":
		cmd = []string{"/sbin/mkfs.xfs", "-f", "-m", "crc=0", "-i", "projid32bit=0", "-n", "ftype=0"}
		cmdUuid = []string{"xfs_admin", "-U", uuid}
	}

	if len(cmd) > 0 {
		var cmds = cmd
		cmds = append(cmds, path)
		if _, err := procutils.NewCommand(cmds[0], cmds[1:]...).Run(); err != nil {
			log.Errorln(err)
			return err
		}
		if len(cmdUuid) > 0 {
			cmds = cmdUuid
			cmds = append(cmds, path)
			if _, err := procutils.NewCommand(cmds[0], cmds[1:]...).Run(); err != nil {
				log.Errorln(err)
				return err
			}
		}
		return nil
	}
	return fmt.Errorf("Unknown fs %s", fs)
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

func ParseDiskPartition(dev string, lines []byte) ([][]string, string) {
	var (
		parts        = [][]string{}
		label        string
		labelPartten = regexp.MustCompile(`Partition Table:\s+(?P<label>\w+)`)
		partten      = regexp.MustCompile(`(?P<idx>\d+)\s+(?P<start>\d+)s\s+(?P<end>\d+)s\s+(?P<count>\d+)s`)
	)

	for _, line := range strings.Split(string(lines), "\n") {
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
	sizeStr, _ := FileGetContents(fmt.Sprintf("/sys/block/%s/size", dev))
	sizeStr = strings.Trim(sizeStr, "\n")
	size, _ := strconv.Atoi(sizeStr)
	return size
}

func ResizeDiskFs(diskPath string, sizeMb int) error {
	var cmds = []string{"parted", "-a", "none", "-s", diskPath, "--", "unit", "s", "print"}
	lines, err := procutils.NewCommand(cmds[0], cmds[1:]...).Run()
	if err != nil {
		log.Errorf("resize disk fs fail: %s", err)
		return err
	}
	parts, label := ParseDiskPartition(diskPath, lines)
	log.Infof("Parts: %v label: %s", parts, label)
	maxSector := GetDevSector512Count(path.Base(diskPath))
	if label == "gpt" {
		proc := exec.Command("gdisk", diskPath)
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
			log.Errorln(err)
			if exiterr, ok := err.(*exec.ExitError); ok {
				ws := exiterr.Sys().(syscall.WaitStatus)
				if ws.ExitStatus() != 1 {
					return err
				}
			} else {
				return err
			}
		}
	}
	if len(parts) > 0 && (label == "gpt" ||
		(label == "msdos" && parts[len(parts)-1][5] == "primary")) {
		var (
			part = parts[len(parts)-1]
			end  int
		)
		if sizeMb > 0 {
			end = sizeMb * 1024 * 2
		} else if label == "gpt" {
			end = maxSector - 35
		} else {
			end = maxSector - 1
		}
		if label == "msdos" && end >= 4294967296 {
			end = 4294967295
		}
		cmds = []string{"parted", "-a", "none", "-s", diskPath, "--",
			"unit", "s", "rm", part[0], "mkpart", part[5]}

		if len(part[6]) > 0 {
			cmds = append(cmds, part[6])
		}
		cmds = append(cmds, part[2], fmt.Sprintf("%ds", end))
		if len(part[1]) > 0 {
			cmds = append(cmds, "set", part[0], "boot", "on")
		}
		log.Infof("resize disk partition: %s", cmds)
		output, err := procutils.NewCommand(cmds[0], cmds[1:]...).Run()
		if err != nil {
			return errors.Wrapf(err, "parted failed %s", output)
		}
		if len(part[6]) > 0 {
			err, _ := ResizePartitionFs(part[7], part[6], false)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func FsckExtFs(fpath string) bool {
	log.Debugf("Exec command: %v", []string{"e2fsck", "-f", "-p", fpath})
	cmd := exec.Command("e2fsck", "-f", "-p", fpath)
	if err := cmd.Start(); err != nil {
		log.Errorln(err)
		return false
	} else {
		err = cmd.Wait()
		if err != nil {
			if exiterr, ok := err.(*exec.ExitError); ok {
				ws := exiterr.Sys().(syscall.WaitStatus)
				if ws.ExitStatus() < 4 {
					return true
				}
			}
			log.Errorln(err)
			return false
		} else {
			return true
		}
	}
}

func FsckXfsFs(fpath string) bool {
	if _, err := procutils.NewCommand("xfs_check", fpath).Run(); err != nil {
		log.Errorln(err)
		procutils.NewCommand("xfs_repair", fpath).Run()
		return false
	}
	return true
}

func ResizePartitionFs(fpath, fs string, raiseError bool) (error, bool) {
	if len(fs) == 0 {
		return nil, false
	}
	var (
		cmds  = [][]string{}
		uuids = GetDevUuid(fpath)
	)
	if strings.HasPrefix(fs, "linux-swap") {
		if v, ok := uuids["UUID"]; ok {
			cmds = [][]string{{"mkswap", "-U", v, fpath}}
		} else {
			cmds = [][]string{{"mkswap", fpath}}
		}
	} else if strings.HasPrefix(fs, "ext") {
		if !FsckExtFs(fpath) {
			if raiseError {
				return fmt.Errorf("Failed to fsck ext fs %s", fpath), false
			} else {
				return nil, false
			}
		}
		cmds = [][]string{{"resize2fs", fpath}}
	} else if fs == "xfs" {
		var tmpPoint = fmt.Sprintf("/tmp/%s", strings.Replace(fpath, "/", "_", -1))
		if _, err := procutils.NewCommand("mountpoint", tmpPoint).Run(); err == nil {
			_, err = procutils.NewCommand("umount", "-f", tmpPoint).Run()
			if err != nil {
				log.Errorln(err)
				return err, false
			}
		}
		FsckXfsFs(fpath)
		cmds = [][]string{{"mkdir", "-p", tmpPoint},
			{"mount", fpath, tmpPoint},
			{"sleep", "2"},
			{"xfs_growfs", tmpPoint},
			{"sleep", "2"},
			{"umount", tmpPoint},
			{"sleep", "2"},
			{"rm", "-fr", tmpPoint}}
	}

	if len(cmds) > 0 {
		for _, cmd := range cmds {
			_, err := procutils.NewCommand(cmd[0], cmd[1:]...).Run()
			if err != nil {
				log.Errorln(err)
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

func GetDevUuid(dev string) map[string]string {
	lines, err := procutils.NewCommand("blkid", dev).Run()
	if err != nil {
		return nil
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
			return ret
		}
	}
	return nil
}

func GetDevOfPath(spath string) string {
	spath, err := filepath.Abs(spath)
	if err != nil {
		log.Errorln(err)
		return ""
	}
	lines, err := procutils.NewCommand("mount").Run()
	if err != nil {
		log.Errorln(err)
		return ""
	}
	var (
		maxMatchLen int
		matchDev    string
	)

	for _, line := range strings.Split(string(lines), "\n") {
		segs := strings.Split(line, " ")
		if len(segs) < 3 {
			continue
		}
		if strings.HasPrefix(segs[0], "/dev/") {
			if strings.HasPrefix(spath, segs[2]) {
				matchLen := len(segs[2])
				if maxMatchLen < matchLen {
					maxMatchLen = matchLen
					matchDev = segs[0]
				}
			}
		}
	}
	return matchDev
}

func GetDevId(spath string) string {
	dev := GetDevOfPath(spath)
	if len(dev) == 0 {
		return ""
	}
	devInfo, err := procutils.NewCommand("ls", "-l", dev).Run()
	if err != nil {
		log.Errorln(err)
		return ""
	}
	devInfos := strings.Split(string(devInfo), "\n")
	data := strings.Split(string(devInfos[0]), " ")
	data[4] = data[4][:len(data[4])-1]
	return strings.Join(data, ":")
}
