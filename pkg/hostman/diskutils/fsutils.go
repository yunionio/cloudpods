package diskutils

import (
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/regutils2"
)

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
	sizeStr, _ := fileutils2.FileGetContents(fmt.Sprintf("/sys/block/%s/size", dev))
	sizeStr = strings.Trim(sizeStr, "\n")
	size, _ := strconv.Atoi(sizeStr)
	return size
}

func ResizeDiskFs(diskPath string, sizeMb int) error {
	var cmds = []string{"parted", "-a", "none", "-s", diskPath, "--", "unit", "s", "print"}
	lines, err := procutils.NewCommand(cmds[0], cmds[1:]...).Output()
	if err != nil {
		log.Errorf("resize disk fs fail: %s", err)
		return err
	}
	parts, label := ParseDiskPartition(diskPath, lines)
	log.Infof("Parts: %v label: %s", parts, label)
	maxSector := GetDevSector512Count(path.Base(diskPath))
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
			if status, succ := procutils.GetExitStatus(err); succ {
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
		output, err := procutils.NewCommand(cmds[0], cmds[1:]...).Output()
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

func ResizePartitionFs(fpath, fs string, raiseError bool) (error, bool) {
	if len(fs) == 0 {
		return nil, false
	}
	var (
		cmds  = [][]string{}
		uuids = fileutils2.GetDevUuid(fpath)
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
		if _, err := procutils.NewCommand("mountpoint", tmpPoint).Output(); err == nil {
			_, err = procutils.NewCommand("umount", "-f", tmpPoint).Output()
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
			_, err := procutils.NewCommand(cmd[0], cmd[1:]...).Output()
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

func FsckExtFs(fpath string) bool {
	log.Debugf("Exec command: %v", []string{"e2fsck", "-f", "-p", fpath})
	cmd := procutils.NewCommand("e2fsck", "-f", "-p", fpath)
	if err := cmd.Start(); err != nil {
		log.Errorln(err)
		return false
	} else {
		err = cmd.Wait()
		if err != nil {
			if status, ok := procutils.GetExitStatus(err); ok {
				if status < 4 {
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
	if _, err := procutils.NewCommand("xfs_check", fpath).Output(); err != nil {
		log.Errorln(err)
		procutils.NewCommand("xfs_repair", fpath).Output()
		return false
	}
	return true
}

func Mkpartition(imagePath, fsFormat string) error {
	var (
		parted    = "/sbin/parted"
		labelType = "gpt"
		diskType  = fileutils2.FsFormatToDiskType(fsFormat)
	)

	if len(diskType) == 0 {
		return fmt.Errorf("Unknown fsFormat %s", fsFormat)
	}

	// 创建一个新磁盘分区表类型, ex: mbr gpt msdos ...
	_, err := procutils.NewCommand(parted, "-s", imagePath, "mklabel", labelType).Output()
	if err != nil {
		log.Errorf("mklabel %s %s error %s", imagePath, fsFormat, err)
		return err
	}

	// 创建一个part-type类型的分区, part-type可以是："primary", "logical", "extended"
	// 如果指定fs-type(即diskType)则在创建分区的同时进行格式化
	err = procutils.NewCommand(parted, "-s", "-a", "cylinder",
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
		if _, err := procutils.NewCommand(cmds[0], cmds[1:]...).Output(); err != nil {
			log.Errorln(err)
			return err
		}
		if len(cmdUuid) > 0 {
			cmds = cmdUuid
			cmds = append(cmds, path)
			if _, err := procutils.NewCommand(cmds[0], cmds[1:]...).Output(); err != nil {
				log.Errorln(err)
				return err
			}
		}
		return nil
	}
	return fmt.Errorf("Unknown fs %s", fs)
}
