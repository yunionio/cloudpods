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

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/baremetal/utils/disktool"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
	"yunion.io/x/onecloud/pkg/hostman/guestfs"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	"yunion.io/x/onecloud/pkg/util/ssh"
	stringutils "yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SSHPartition struct {
	term      ISSHClient // *ssh.Client
	partDev   string
	mountPath string
	isLVM     bool
}

var _ fsdriver.IDiskPartition = &SSHPartition{}

func NewSSHPartition(term ISSHClient, partDev string, isLVM bool) *SSHPartition {
	p := new(SSHPartition)
	p.term = term
	p.partDev = partDev
	p.isLVM = isLVM
	p.mountPath = fmt.Sprintf("/tmp/%s", strings.Replace(p.partDev, "/", "_", -1))
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

func (p *SSHPartition) Umount() error {
	if !p.IsMounted() {
		log.Warningf("%s is not mounted", p.mountPath)
		return nil
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
				log.Warningf("remove mount path %s: %v", p.mountPath, err)
			}
			return nil
		}
	}
	return err
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

func (p *SSHPartition) GetPartDev() string {
	return ""
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
	odstDir := path.Dir(dst)
	if err := f.Mkdir(odstDir, 0755, caseInsensitive); err != nil {
		return errors.Wrapf(err, "Mkdir %s", odstDir)
	}
	if f.Exists(dst, caseInsensitive) {
		f.Remove(dst, caseInsensitive)
	}
	odstDir = f.GetLocalPath(odstDir, caseInsensitive)
	dst = path.Join(odstDir, path.Base(dst))
	return f.osSymlink(src, dst)
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
	if len(content) == 0 {
		cmd := fmt.Sprintf(`echo -n -e "" %s %s`, op, sPath)
		cmds = append(cmds, cmd)
	} else {
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

func (p *SSHPartition) CheckOrAddUser(user, homeDir string, isSys bool) (realHomeDir string, err error) {
	var exist bool
	if exist, realHomeDir, err = p.checkUser(user); err != nil || exist {
		if exist {
			cmd := []string{"chage", "-R", p.mountPath, "-E", "-1", "-m", "0", "-M", "99999", "-I", "-1", user}
			_, err = p.term.Run(strings.Join(cmd, " "))
			if err != nil {
				if !strings.Contains(err.Error(), "not found") {
					err = errors.Wrap(err, "chage")
					return
				} else {
					err = nil
				}
			}
			if !p.Exists(realHomeDir, false) {
				err = p.Mkdir(realHomeDir, 0700, false)
				if err != nil {
					err = errors.Wrapf(err, "Mkdir %s", realHomeDir)
				} else {
					cmd := []string{"/usr/sbin/chroot", p.mountPath, "chown", user, realHomeDir}
					_, err = p.term.Run(strings.Join(cmd, " "))
					if err != nil {
						err = errors.Wrap(err, "chown")
					}
				}
			}
		}
		return
	}
	return path.Join(homeDir, user), p.userAdd(user, homeDir, isSys)
}

func (p *SSHPartition) checkUser(user string) (exist bool, homeDir string, err error) {
	cmd := fmt.Sprintf("/usr/sbin/chroot %s /bin/cat /etc/passwd", p.mountPath)
	lines, err := p.term.Run(cmd)
	if err != nil {
		return
	}
	log.Debugf("exec command 'cat /etc/passwd', output: %v", lines)
	for i := len(lines) - 1; i >= 0; i-- {
		userInfos := strings.Split(strings.TrimSpace(lines[i]), ":")
		if len(userInfos) < 6 {
			continue
		}
		if userInfos[0] != user {
			continue
		}
		exist = true
		homeDir = userInfos[5]
		break
	}
	return
}

func (p *SSHPartition) userAdd(user, homeDir string, isSys bool) error {
	cmd := fmt.Sprintf("/usr/sbin/chroot %s /usr/sbin/useradd -m -s /bin/bash %s", p.mountPath, user)
	if isSys {
		cmd += " -r -e '' -f '-1' -K 'PASS_MAX_DAYS=-1'"
	}
	if len(homeDir) > 0 {
		cmd += fmt.Sprintf(" -d %s", path.Join(homeDir, user))
	}
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
			stMode, err := fsdriver.ModeStr2Bin(dat[2])
			if err != nil {
				return nil, err
			}
			stIno, _ := strconv.Atoi(dat[0])
			stUid, _ := strconv.Atoi(dat[4])
			stGid, _ := strconv.Atoi(dat[5])
			stSize, _ := strconv.Atoi(dat[6])
			info := fsdriver.NewFileInfo(
				sPath,
				int64(stSize),
				os.FileMode(stMode),
				dat[2][0] == 'd',
				&syscall.Stat_t{
					Ino:  uint64(stIno),
					Uid:  uint32(stUid),
					Gid:  uint32(stGid),
					Size: int64(stSize),
				},
			)
			return info, nil
		}
	}
	return nil, fmt.Errorf("Can't stat for path %s", sPath)
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

func (p *SSHPartition) Zerofree() {
	log.Warningf("zerofree should not called in ssh partition")
}

func MountSSHRootfs(tool *disktool.SSHPartitionTool, term *ssh.Client, layouts []baremetal.Layout) (*SSHPartition, fsdriver.IRootFsDriver, error) {
	// tool, err := disktool.NewSSHPartitionTool(term, layouts)
	// if err != nil {
	// 	return nil, nil, err
	// }
	// tool.RetrievePartitionInfo()
	rootDisk := tool.GetRootDisk()
	if err := rootDisk.ReInitInfo(); err != nil {
		return nil, nil, errors.Wrapf(err, "Reinit root disk")
	}
	parts := rootDisk.GetPartitions()
	if len(parts) == 0 {
		return nil, nil, fmt.Errorf("Not found root disk partitions")
	}
	for _, part := range parts {
		dev := NewSSHPartition(term, part.GetDev(), false)
		if !dev.Mount() {
			continue
		}
		rootFs, err := guestfs.DetectRootFs(dev)
		if err == nil {
			log.Infof("Use class %#v", rootFs)
			return dev, rootFs, nil
		}
		dev.Umount()
	}
	return nil, nil, fmt.Errorf("Fail to find rootfs")
}

func (p *SSHPartition) GenerateSshHostKeys() error {
	for _, cmd := range []string{
		"touch /dev/null",
		"/usr/bin/ssh-keygen -A",
	} {
		_, err := p.term.Run(fmt.Sprintf("/usr/sbin/chroot %s %s", p.mountPath, cmd))
		if err != nil {
			return errors.Wrapf(err, "GenerateSshHostKeys exec %s", cmd)
		}
	}
	return nil
}
