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

package sshpart

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/baremetal/utils/disktool"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
	"yunion.io/x/onecloud/pkg/hostman/guestfs"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	"yunion.io/x/onecloud/pkg/util/ssh"
	stringutils "yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SSHPartition struct {
	term      *ssh.Client
	partDev   string
	mountPath string
	part      *disktool.Partition
}

func NewSSHPartition(term *ssh.Client, part *disktool.Partition) *SSHPartition {
	p := new(SSHPartition)
	p.term = term
	p.partDev = part.GetDev()
	p.mountPath = fmt.Sprintf("/tmp/%s", strings.Replace(p.partDev, "/", "_", -1))
	p.part = part
	return p
}

func (p *SSHPartition) GetMountPath() string {
	return p.mountPath
}

func (p *SSHPartition) GetFsFormat() (string, error) {
	cmd := fmt.Sprintf("/lib/mos/partfs.sh %s", p.partDev)
	ret, err := p.term.Run(cmd)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(ret[0]), nil
}

func (p *SSHPartition) osChmod(path string, mode uint32) error {
	cmd := fmt.Sprintf("chmod %o %s", (mode & 0777), path)
	_, err := p.term.Run(cmd)
	return err
}

func (p *SSHPartition) osMkdirP(dir string, mode uint32) error {
	cmd := fmt.Sprintf("mkdir -p %s", dir)
	_, err := p.term.Run(cmd)
	if err != nil {
		return err
	}
	if mode != 0 {
		return p.osChmod(dir, mode)
	}
	return nil
}

func (p *SSHPartition) Mkdir(sPath string, mode int, caseInsensitive bool) error {
	segs := strings.Split(sPath, "/")
	sp := ""
	pPath := p.GetLocalPath("/", caseInsensitive)
	var err error
	for _, s := range segs {
		if len(s) > 0 {
			sp = path.Join(sp, s)
			vPath := p.GetLocalPath(sp, caseInsensitive)
			if len(vPath) == 0 {
				err = p.osMkdirP(path.Join(pPath, s), uint32(mode))
				pPath = p.GetLocalPath(sp, caseInsensitive)
			} else {
				pPath = vPath
			}
		}
	}
	return err
}

func (p *SSHPartition) osRmDir(path string) error {
	_, err := p.term.Run(fmt.Sprintf("rm -fr %s", path))
	return err
}

func (p *SSHPartition) osPathExists(path string) bool {
	// test file or symbolic link exists
	_, err := p.term.Run(fmt.Sprintf("test -e %s || test -L %s", path, path))
	if err != nil {
		return false
	}
	return true
}

func (p *SSHPartition) osSymlink(src, dst string) error {
	_, err := p.term.Run(fmt.Sprintf("ln -s %s %s", src, dst))
	return err
}

func (p *SSHPartition) MountPartReadOnly() bool {
	return false // Not implement
}

func (p *SSHPartition) IsReadonly() bool {
	return false // sshpart not implement mount as readonly
}

func (p *SSHPartition) GetPhysicalPartitionType() string {
	return "" // Not implement
}

func (p *SSHPartition) Mount() bool {
	if err := p.osMkdirP(p.mountPath, 0); err != nil {
		log.Errorf("SSHPartition mount error: %s", err)
		return false
	}
	fs, err := p.GetFsFormat()
	if err != nil {
		log.Errorf("SSHPartition mount error: %s", err)
		return false
	}
	fstr := ""
	if fs == "ntfs" {
		fstr = "-t ntfs-3g"
	}
	cmd := fmt.Sprintf("mount %s -o sync %s %s", fstr, p.partDev, p.mountPath)
	log.Infof("Do mount %s", cmd)
	_, err = p.term.Run(cmd)
	if err != nil {
		p.osRmDir(p.mountPath)
		log.Errorf("SSHPartition mount error: %s", err)
		return false
	}
	return true
}

func (p *SSHPartition) Umount() bool {
	if !p.IsMounted() {
		log.Errorf("%s is not mounted", p.mountPath)
		return false
	}
	var err error
	for tries := 0; tries < 10; tries++ {
		cmds := []string{
			"sync",
			"/sbin/sysctl -w vm.drop_caches=3",
			fmt.Sprintf("/bin/umount %s", p.mountPath),
			fmt.Sprintf("/sbin/hdparm -f %s", p.partDev),
			//fmt.Sprintf("/usr/bin/sg_sync %s", p.part.GetDisk().GetDev()),
		}
		_, err = p.term.Run(cmds...)
		if err != nil {
			log.Errorf("umount %s error: %v", p.mountPath, err)
			time.Sleep(1 * time.Second)
		} else {
			if err := p.osRmDir(p.mountPath); err != nil {
				log.Errorf("remove mount path %s: %v", p.mountPath, err)
				return false
			}
			return true
		}
	}
	return err == nil
}

func (p *SSHPartition) IsMounted() bool {
	if !p.osPathExists(p.mountPath) {
		return false
	}
	_, err := p.term.Run(fmt.Sprintf("mountpoint %s", p.mountPath))
	if err != nil {
		return false
	}
	return true
}

func (p *SSHPartition) Chmod(sPath string, mode uint32, caseI bool) error {
	sPath = p.GetLocalPath(sPath, caseI)
	if sPath != "" {
		return p.osChmod(sPath, mode)
	}
	return nil
}

func (p *SSHPartition) osIsDir(path string) bool {
	_, err := p.term.Run(fmt.Sprintf("test -d %s", path))
	if err != nil {
		return false
	}
	return true
}

func (p *SSHPartition) osListDir(path string) ([]string, error) {
	if !p.osIsDir(path) {
		return nil, fmt.Errorf("Path %s is not dir", path)
	}
	ret, err := p.term.Run(fmt.Sprintf("ls -a %s", path))
	if err != nil {
		return nil, err
	}
	files := []string{}
	for _, f := range ret {
		f = strings.TrimSpace(f)
		if !utils.IsInStringArray(f, []string{"", ".", ".."}) {
			files = append(files, f)
		}
	}
	return files, nil
}

func (p *SSHPartition) GetLocalPath(sPath string, caseI bool) string {
	var fullPath = p.mountPath
	pathSegs := strings.Split(sPath, "/")
	for _, seg := range pathSegs {
		if len(seg) == 0 {
			continue
		}

		var realSeg string
		files, err := p.osListDir(fullPath)
		if err != nil {
			log.Errorf("List dir %s error: %v", sPath, err)
			return ""
		}
		for _, f := range files {
			if f == seg || (caseI && (strings.ToLower(f)) == strings.ToLower(seg)) {
				realSeg = f
				break
			}
		}
		if len(realSeg) > 0 {
			fullPath = path.Join(fullPath, realSeg)
		} else {
			return ""
		}
	}
	return fullPath
}

func (f *SSHPartition) Symlink(src string, dst string, caseInsensitive bool) error {
	f.Mkdir(path.Dir(dst), 0755, caseInsensitive)
	odstDir := f.GetLocalPath(path.Dir(dst), caseInsensitive)
	return f.osSymlink(src, path.Join(odstDir, path.Base(dst)))
}

func (p *SSHPartition) Exists(sPath string, caseInsensitive bool) bool {
	sPath = p.GetLocalPath(sPath, caseInsensitive)
	if len(sPath) > 0 {
		return p.osPathExists(sPath)
	}
	return false
}

func (p *SSHPartition) sshFileGetContents(path string) ([]byte, error) {
	cmd := fmt.Sprintf("cat %s", path)
	ret, err := p.term.Run(cmd)
	if err != nil {
		return nil, err
	}
	if len(ret) > 0 && ret[len(ret)-1] == "" {
		ret = ret[0 : len(ret)-1]
	}
	retBytes := []byte(strings.Join(ret, "\n"))
	return retBytes, nil
}

func (p *SSHPartition) FileGetContents(sPath string, caseInsensitive bool) ([]byte, error) {
	sPath = p.GetLocalPath(sPath, caseInsensitive)
	return p.FileGetContentsByPath(sPath)
}

func (p *SSHPartition) FileGetContentsByPath(sPath string) ([]byte, error) {
	if len(sPath) == 0 {
		return nil, fmt.Errorf("Path is not provide")
	}
	return p.sshFileGetContents(sPath)
}

func (p *SSHPartition) sshFilePutContents(sPath, content string, modAppend bool) error {
	op := ">"
	if modAppend {
		op = ">>"
	}

	cmds := []string{}
	var chunkSize int = 8192
	for offset := 0; offset < len(content); offset += chunkSize {
		end := offset + chunkSize
		if end > len(content) {
			end = len(content)
		}
		ll, err := stringutils.EscapeEchoString(content[offset:end])
		if err != nil {
			return fmt.Errorf("EscapeEchoString %q error: %v", content[offset:end], err)
		}
		cmd := fmt.Sprintf(`echo -n -e "%s" %s %s`, ll, op, sPath)
		cmds = append(cmds, cmd)
		if op == ">" {
			op = ">>"
		}
	}
	_, err := p.term.Run(cmds...)
	return err
}

func (p *SSHPartition) FilePutContents(sPath, content string, modAppend, caseInsensitive bool) error {
	sFilePath := p.GetLocalPath(sPath, caseInsensitive)
	if len(sFilePath) > 0 {
		sPath = sFilePath
	} else {
		dirPath := p.GetLocalPath(path.Dir(sPath), caseInsensitive)
		if len(dirPath) > 0 {
			sPath = path.Join(dirPath, path.Base(sPath))
		}
	}
	if len(sPath) > 0 {
		return p.sshFilePutContents(sPath, content, modAppend)
	}
	return fmt.Errorf("Can't put content to %s", sPath)
}

func (p *SSHPartition) ListDir(sPath string, caseInsensitive bool) []string {
	sPath = p.GetLocalPath(sPath, caseInsensitive)
	if len(sPath) > 0 {
		ret, err := p.osListDir(sPath)
		if err != nil {
			log.Errorf("list dir for %s: %v", sPath, err)
			return nil
		}
		return ret
	}
	return nil
}

func (p *SSHPartition) osChown(sPath string, uid, gid int) error {
	cmd := fmt.Sprintf("chown %d.%d %s", uid, gid, sPath)
	_, err := p.term.Run(cmd)
	return err
}

func (p *SSHPartition) Chown(sPath string, uid, gid int, caseInsensitive bool) error {
	sPath = p.GetLocalPath(sPath, caseInsensitive)
	if len(sPath) == 0 {
		return fmt.Errorf("Can't get local path: %s", sPath)
	}
	return p.osChown(sPath, uid, gid)
}

func (p *SSHPartition) osRemove(sPath string) error {
	cmd := fmt.Sprintf("rm %s", sPath)
	_, err := p.term.Run(cmd)
	return err
}

func (p *SSHPartition) Remove(sPath string, caseInsensitive bool) {
	sPath = p.GetLocalPath(sPath, caseInsensitive)
	if len(sPath) > 0 {
		p.osRemove(sPath)
	}
}

func (p *SSHPartition) UserAdd(user string, caseInsensitive bool) error {
	cmd := fmt.Sprintf("/usr/sbin/chroot %s /usr/sbin/useradd -m -s /bin/bash %s", p.mountPath, user)
	_, err := p.term.Run(cmd)
	return err
}

func (p *SSHPartition) Passwd(user, password string, caseInsensitive bool) error {
	newpass := "/tmp/newpass"
	p.sshFilePutContents(newpass, fmt.Sprintf("%s\n%s\n", password, password), false)
	cmd := fmt.Sprintf("/usr/sbin/chroot %s /usr/bin/passwd %s < %s", p.mountPath, user, newpass)
	_, err := p.term.Run(cmd)
	return err
}

func (p *SSHPartition) osStat(sPath string) (os.FileInfo, error) {
	cmd := fmt.Sprintf("ls -a -l -n -i -s -d %s", sPath)
	ret, err := p.term.Run(cmd)
	if err != nil {
		return nil, err
	}
	for _, line := range ret {
		dat := regexp.MustCompile(`\s+`).Split(strings.TrimSpace(line), -1)
		if len(dat) > 7 && ((dat[2][0] != 'l' && dat[len(dat)-1] == sPath) ||
			(dat[2][0] == 'l' && dat[len(dat)-3] == sPath)) {
			stMode, err := modeStr2Bin(dat[2])
			if err != nil {
				return nil, err
			}
			stIno, _ := strconv.Atoi(dat[0])
			stUid, _ := strconv.Atoi(dat[4])
			stGid, _ := strconv.Atoi(dat[5])
			stSize, _ := strconv.Atoi(dat[6])
			info := &sFileInfo{
				name:  sPath,
				size:  int64(stSize),
				mode:  os.FileMode(stMode),
				isDir: dat[2][0] == 'd',
				stat: &syscall.Stat_t{
					Ino:  uint64(stIno),
					Uid:  uint32(stUid),
					Gid:  uint32(stGid),
					Size: int64(stSize),
				},
			}
			return info, nil
		}
	}
	return nil, fmt.Errorf("Can't stat for path %s", sPath)
}

func modeStr2Bin(mode string) (uint32, error) {
	table := []map[byte]uint32{
		{'-': syscall.S_IRUSR, 'd': syscall.S_IFDIR, 'l': syscall.S_IFLNK},
		{'r': syscall.S_IRUSR},
		{'w': syscall.S_IWUSR},
		{'x': syscall.S_IXUSR, 's': syscall.S_ISUID},
		{'r': syscall.S_IRGRP},
		{'w': syscall.S_IWGRP},
		{'x': syscall.S_IXGRP, 's': syscall.S_ISGID},
		{'r': syscall.S_IROTH},
		{'w': syscall.S_IWOTH},
		{'x': syscall.S_IXOTH},
	}
	if len(mode) != len(table) {
		return 0, fmt.Errorf("Invalid mod %q", mode)
	}
	var ret uint32 = 0
	for i := 0; i < len(table); i++ {
		ret |= table[i][mode[i]]
	}
	return ret, nil
}

// sFileInfo implements os.FileInfo interface
type sFileInfo struct {
	name  string
	size  int64
	mode  os.FileMode
	isDir bool
	stat  *syscall.Stat_t
}

func (info sFileInfo) Name() string {
	return info.name
}

func (info sFileInfo) Size() int64 {
	return info.size
}

func (info sFileInfo) Mode() os.FileMode {
	return info.mode
}

func (info sFileInfo) IsDir() bool {
	return info.isDir
}

func (info sFileInfo) ModTime() time.Time {
	// TODO: impl
	return time.Now()
}

func (info sFileInfo) Sys() interface{} {
	return info.stat
}

func (p *SSHPartition) Stat(sPath string, caseInsensitive bool) os.FileInfo {
	sPath = p.GetLocalPath(sPath, caseInsensitive)
	if len(sPath) == 0 {
		return nil
	}
	info, err := p.osStat(sPath)
	if err != nil {
		log.Errorf("stat %s error: %v", sPath, err)
		return nil
	}
	return info
}

func (p *SSHPartition) Zerofiles(dir string, caseI bool) error {
	return nil
}

func (p *SSHPartition) GetReadonly() bool {
	return false
}

func (p *SSHPartition) SupportSerialPorts() bool {
	return true
}

func (p *SSHPartition) Cleandir(dir string, keepdir, caseInsensitive bool) error {
	return nil
}

func MountSSHRootfs(term *ssh.Client, layouts []baremetal.Layout) (*SSHPartition, fsdriver.IRootFsDriver, error) {
	tool := disktool.NewSSHPartitionTool(term)
	tool.FetchDiskConfs(baremetal.GetDiskConfigurations(layouts))
	if err := tool.RetrieveDiskInfo(); err != nil {
		return nil, nil, err
	}
	tool.RetrievePartitionInfo()
	rootDisk := tool.GetRootDisk()
	if err := rootDisk.ReInitInfo(); err != nil {
		return nil, nil, errors.Wrapf(err, "Reinit root disk")
	}
	parts := rootDisk.GetPartitions()
	if len(parts) == 0 {
		return nil, nil, fmt.Errorf("Not found root disk partitions")
	}
	for _, part := range parts {
		dev := NewSSHPartition(term, part)
		if !dev.Mount() {
			continue
		}
		if rootFs := guestfs.DetectRootFs(dev); rootFs != nil {
			log.Infof("Use class %#v", rootFs)
			return dev, rootFs, nil
		} else {
			dev.Umount()
		}
	}
	return nil, nil, fmt.Errorf("Fail to find rootfs")
}
