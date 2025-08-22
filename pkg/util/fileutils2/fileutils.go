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
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/regutils"

	"yunion.io/x/onecloud/pkg/util/procutils"
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
	mounts, err := procutils.NewCommand("mount").Output()
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
	devs, _ := procutils.NewCommand("cat", "/proc/partitions").Output()
	if idx := strings.Index(string(devs), devStr); idx > 0 {
		return true
	}
	return false
}

func GetAllBlkdevsIoSchedulers() ([]string, error) {
	if _, err := os.Stat("/sys/block"); !os.IsNotExist(err) {
		blockDevs, err := os.ReadDir("/sys/block")
		if err != nil {
			log.Errorf("ReadDir /sys/block error: %s", err)
			return nil, errors.Wrap(err, "ioutil.ReadDir(/sys/block)")
		}
		for _, b := range blockDevs {
			// check is a block device
			if !Exists(path.Join("/sys/block", b.Name(), "device")) {
				continue
			}
			if IsBlockDevMounted(b.Name()) {
				conf, err := GetBlkdevConfig(b.Name(), "queue/scheduler")
				if err != nil {
					log.Errorf("Get %s queue/scheduler fail %s", b.Name(), err)
				} else {
					algs := make([]string, 0)
					for _, alg := range strings.Split(strings.TrimSpace(conf), " ") {
						if len(alg) > 0 {
							if alg[0] == '[' {
								alg = alg[1 : len(alg)-1]
							}
							algs = append(algs, alg)
						}
					}
					return algs, nil
				}
			}
		}
		log.Errorf("no block device avaiable")
		return nil, nil
	} else {
		log.Errorf("stat /sys/block fail %s", err)
		return nil, errors.Wrap(err, "stat /sys/block fail")
	}
}

func ChangeAllBlkdevsParams(params map[string]string) {
	if _, err := os.Stat("/sys/block"); !os.IsNotExist(err) {
		blockDevs, err := ioutil.ReadDir("/sys/block")
		if err != nil {
			log.Errorf("ReadDir /sys/block error: %s", err)
			return
		}
		for _, b := range blockDevs {
			if !Exists(path.Join("/sys/block", b.Name(), "device")) {
				continue
			}
			for k, v := range params {
				ChangeBlkdevParameter(b.Name(), k, v)
			}
		}
	}
}

func BlockDevIsSsd(dev string) bool {
	rotational := path.Join("/sys/block", dev, "queue", "rotational")
	res, err := FileGetContents(rotational)
	if err != nil {
		log.Errorf("FileGetContents fail %s %s", rotational, err)
		return false
	}
	return strings.TrimSpace(res) == "0"
}

func ChangeSsdBlkdevsParams(params map[string]string) {
	if _, err := os.Stat("/sys/block"); !os.IsNotExist(err) {
		blockDevs, err := os.ReadDir("/sys/block")
		if err != nil {
			log.Errorf("ReadDir /sys/block error: %s", err)
			return
		}
		for _, b := range blockDevs {
			if !Exists(path.Join("/sys/block", b.Name(), "device")) {
				continue
			}
			if !BlockDevIsSsd(b.Name()) {
				continue
			}
			for k, v := range params {
				ChangeBlkdevParameter(b.Name(), k, v)
			}
		}
	}
}

func ChangeHddBlkdevsParams(params map[string]string) {
	if _, err := os.Stat("/sys/block"); !os.IsNotExist(err) {
		blockDevs, err := os.ReadDir("/sys/block")
		if err != nil {
			log.Errorf("ReadDir /sys/block error: %s", err)
			return
		}
		for _, b := range blockDevs {
			if !Exists(path.Join("/sys/block", b.Name(), "device")) {
				continue
			}
			if BlockDevIsSsd(b.Name()) {
				continue
			}
			for k, v := range params {
				ChangeBlkdevParameter(b.Name(), k, v)
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

func GetBlkdevConfig(dev, key string) (string, error) {
	p := path.Join("/sys/block", dev, key)
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		return FileGetContents(p)
	} else {
		return "", err
	}
}

func FileGetContents(file string) (string, error) {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func FileGetIntContent(file string) (int, error) {
	content, err := FileGetContents(file)
	if err != nil {
		return -1, errors.Wrap(err, "FileGetContents")
	}
	val, err := strconv.Atoi(strings.TrimSpace(content))
	if err != nil {
		return -1, errors.Wrapf(err, "convert %s to int", content)
	}
	return val, nil
}

func GetFsFormat(diskPath string) string {
	ret, err := procutils.NewCommand("blkid", "-o", "value", "-s", "TYPE", diskPath).Output()
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

func FormatHostsFile(content string, ips []string, hostname, hostdomain string) string {
	hf := make(HostsFile, 0)
	hf.Parse(content)
	hf.Add("127.0.0.1", "localhost")
	isV6 := false
	for _, ip := range ips {
		if regutils.MatchIP6Addr(ip) {
			isV6 = true
		}
	}
	if isV6 {
		hf.Add("::1", "localhost", "ip6-localhost", "ip6-loopback")
	}
	for _, ip := range ips {
		hf.Add(ip, hostdomain, hostname)
	}
	return hf.String()
}

func FsFormatToDiskType(fsFormat string) string {
	switch {
	case fsFormat == "swap":
		return "linux-swap"
	case strings.HasPrefix(fsFormat, "ext") || fsFormat == "xfs" || fsFormat == "f2fs":
		return "ext2"
	case strings.HasPrefix(fsFormat, "fat"):
		return "fat32"
	case fsFormat == "ntfs":
		return fsFormat
	default:
		return ""
	}
}

func GetDevOfPath(spath string) string {
	spath, err := filepath.Abs(spath)
	if err != nil {
		log.Errorln(err)
		return ""
	}
	lines, err := procutils.NewCommand("mount").Output()
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
	devInfo, err := procutils.NewCommand("ls", "-l", dev).Output()
	if err != nil {
		log.Errorln(err)
		return ""
	}
	devInfos := strings.Split(string(devInfo), "\n")
	data := strings.Split(string(devInfos[0]), " ")
	data[4] = data[4][:len(data[4])-1]
	return strings.Join(data, ":")
}

func GetDevUuid(dev string) (map[string]string, error) {
	lines, err := procutils.NewCommand("blkid", dev).Output()
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

func IsIsoFile(sPath string) bool {
	file, err := os.Open(sPath)
	if err != nil {
		return false
	}
	defer file.Close()
	file.Seek(0x8001, 0)
	buffer := make([]byte, 5)
	_, err = file.Read(buffer)
	if err != nil {
		return false
	}
	return bytes.Equal(buffer, []byte("CD001"))
}

func IsTarGzipFile(fPath string) bool {
	f, err := os.Open(fPath)
	if err != nil {
		return false
	}
	defer f.Close()

	gzf, err := gzip.NewReader(f)
	if err != nil {
		return false
	}

	return IsTarStream(gzf)
}

func IsTarStream(f io.Reader) bool {
	tarReader := tar.NewReader(f)
	_, err := tarReader.Next()
	if err != nil {
		return false
	}
	return true
}

func IsTarFile(fPath string) bool {
	f, err := os.Open(fPath)
	if err != nil {
		return false
	}
	defer f.Close()
	return IsTarStream(f)
}
