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

package kvmpart

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

type SLocalGuestFS struct {
	mountPath string
}

func NewLocalGuestFS(mountPath string) *SLocalGuestFS {
	var ret = new(SLocalGuestFS)
	ret.mountPath = mountPath
	return ret
}

func (f *SLocalGuestFS) GetMountPath() string {
	return f.mountPath
}

func (f *SLocalGuestFS) SupportSerialPorts() bool {
	return false
}

func (f *SLocalGuestFS) GetLocalPath(sPath string, caseInsensitive bool) string {
	if sPath == "." {
		sPath = ""
	}
	var fullPath = f.mountPath
	pathSegs := strings.Split(sPath, "/")
	for _, seg := range pathSegs {
		if len(seg) > 0 {
			var realSeg string
			files, _ := ioutil.ReadDir(fullPath)
			for _, file := range files {
				var f = file.Name()
				if f == seg || (caseInsensitive && strings.ToLower(f) == strings.ToLower(seg)) ||
					(seg[len(seg)-1] == '*' && (strings.HasPrefix(f, seg[:len(seg)-1]) ||
						(caseInsensitive && strings.HasPrefix(strings.ToLower(f),
							strings.ToLower(seg[:len(seg)-1]))))) {
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
	}
	return fullPath
}

func (f *SLocalGuestFS) Remove(path string, caseInsensitive bool) {
	path = f.GetLocalPath(path, caseInsensitive)
	if len(path) > 0 {
		os.Remove(path)
	}
}

func (f *SLocalGuestFS) Mkdir(sPath string, mode int, caseInsensitive bool) error {
	segs := strings.Split(sPath, "/")
	sPath = ""
	pPath := f.GetLocalPath("/", caseInsensitive)
	for _, s := range segs {
		if len(s) > 0 {
			sPath = path.Join(sPath, s)
			vPath := f.GetLocalPath(sPath, caseInsensitive)
			if len(vPath) == 0 {
				if err := os.Mkdir(path.Join(pPath, s), os.FileMode(mode)); err != nil {
					return err
				}
				pPath = f.GetLocalPath(sPath, caseInsensitive)
			} else {
				pPath = vPath
			}
		}
	}
	return nil
}

func (f *SLocalGuestFS) ListDir(sPath string, caseInsensitive bool) []string {
	sPath = f.GetLocalPath(sPath, caseInsensitive)
	if len(sPath) > 0 {
		files, err := ioutil.ReadDir(sPath)
		if err != nil {
			log.Errorln(err)
			return nil
		}
		var res = make([]string, 0)
		for _, file := range files {
			res = append(res, file.Name())
		}
		return res
	}
	return nil
}

func (f *SLocalGuestFS) Cleandir(dir string, keepdir, caseInsensitive bool) error {
	sPath := f.GetLocalPath(dir, caseInsensitive)
	if len(sPath) > 0 {
		return fileutils2.Cleandir(sPath, keepdir)
	}
	return fmt.Errorf("No such file %s", sPath)
}

func (f *SLocalGuestFS) Zerofiles(dir string, caseInsensitive bool) error {
	sPath := f.GetLocalPath(dir, caseInsensitive)
	if len(sPath) > 0 {
		return fileutils2.Zerofiles(sPath)
	}
	return fmt.Errorf("No such file %s", sPath)
}

func (f *SLocalGuestFS) Passwd(account, password string, caseInsensitive bool) error {
	var proc = procutils.NewCommand("chroot", f.mountPath, "passwd", account)
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
	io.WriteString(stdin, fmt.Sprintf("%s\n", password))
	io.WriteString(stdin, fmt.Sprintf("%s\n", password))
	stdoutPut, err := ioutil.ReadAll(outb)
	if err != nil {
		return err
	}
	stderrOutPut, err := ioutil.ReadAll(errb)
	if err != nil {
		return err
	}
	log.Infof("Passwd %s %s", stdoutPut, stderrOutPut)
	return proc.Wait()
}

func (f *SLocalGuestFS) Stat(usrDir string, caseInsensitive bool) os.FileInfo {
	sPath := f.GetLocalPath(usrDir, caseInsensitive)
	if len(sPath) > 0 {
		fileInfo, err := os.Stat(sPath)
		if err != nil {
			log.Errorln(err)
		}
		return fileInfo
	}
	return nil
}

func (f *SLocalGuestFS) Symlink(src string, dst string, caseInsensitive bool) error {
	dir := path.Dir(dst)
	if err := f.Mkdir(dir, 0755, caseInsensitive); err != nil {
		return errors.Wrapf(err, "Mkdir %s", dir)
	}
	if f.Exists(dst, caseInsensitive) {
		f.Remove(dst, caseInsensitive)
	}
	dir = f.GetLocalPath(dir, caseInsensitive)
	dst = path.Join(dir, path.Base(dst))
	return os.Symlink(src, dst)
}

func (f *SLocalGuestFS) Exists(sPath string, caseInsensitive bool) bool {
	sPath = f.GetLocalPath(sPath, caseInsensitive)
	if len(sPath) > 0 {
		return fileutils2.Exists(sPath)
	}
	return false
}

func (f *SLocalGuestFS) Chown(sPath string, uid, gid int, caseInsensitive bool) error {
	sPath = f.GetLocalPath(sPath, caseInsensitive)
	if len(sPath) > 0 {
		return os.Chown(sPath, uid, gid)
	}
	return nil
}

func (f *SLocalGuestFS) Chmod(sPath string, mode uint32, caseInsensitive bool) error {
	sPath = f.GetLocalPath(sPath, caseInsensitive)
	if len(sPath) > 0 {
		return os.Chmod(sPath, os.FileMode(mode))
	}
	return nil
}

func (f *SLocalGuestFS) updateUserEtcShadow(username string) error {
	sPath := f.GetLocalPath("/etc/shadow", false)
	if !fileutils2.Exists(sPath) {
		return nil
	}
	content, err := fileutils2.FileGetContents(sPath)
	if err != nil {
		return errors.Wrap(err, "read /etc/shadow")
	}

	var (
		minimumDays = "0"     // -m 0
		maximumDays = "99999" // -M 99999
	)

	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		fields := strings.Split(line, ":")
		if len(fields) >= 7 && fields[0] == username {
			fields[3] = minimumDays
			fields[4] = maximumDays
			fields[5] = "" // password warning period
			fields[6] = "" // password inactivity period
			fields[7] = "" // account expiration date
			line = strings.Join(fields, ":")
			lines[i] = line
			break
		}
	}
	newContent := strings.Join(lines, "\n")
	err = fileutils2.FilePutContents(sPath, newContent, false)
	if err != nil {
		return errors.Wrapf(err, "read %s, put %s to /etc/shadow", content, newContent)
	}

	return nil
}

func (f *SLocalGuestFS) CheckOrAddUser(user, homeDir string, isSys bool) (realHomeDir string, err error) {
	var exist bool
	if exist, realHomeDir, err = f.checkUser(user); err != nil || exist {
		if exist {
			err = f.updateUserEtcShadow(user)
			if err != nil {
				err = errors.Wrap(err, "updateUserEtcShadow")
				return
			}
			if !f.Exists(realHomeDir, false) {
				err = f.Mkdir(realHomeDir, 0700, false)
				if err != nil {
					err = errors.Wrapf(err, "Mkdir %s", realHomeDir)
				} else {
					cmd := []string{"chroot", f.mountPath, "chown", user, realHomeDir}
					err = procutils.NewCommand(cmd[0], cmd[1:]...).Run()
					if err != nil {
						err = errors.Wrap(err, "chown")
					}
				}
			}
		}
		return
	}
	return path.Join(homeDir, user), f.userAdd(user, homeDir, isSys)
}

func (f *SLocalGuestFS) checkUser(user string) (exist bool, homeDir string, err error) {
	cmd := []string{"chroot", f.mountPath, "cat", "/etc/passwd"}
	command := procutils.NewCommand(cmd[0], cmd[1:]...)
	output, err := command.Output()
	if err != nil {
		return
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
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

func (f *SLocalGuestFS) userAdd(user, homeDir string, isSys bool) error {
	if err := f.Mkdir(homeDir, 0755, false); err != nil {
		return errors.Wrap(err, "Mkdir")
	}
	cmd := []string{"chroot", f.mountPath, "useradd", "-m", "-s", "/bin/bash", user}
	if isSys {
		cmd = append(cmd, "-r", "-e", "", "-f", "-1", "-K", "PASS_MAX_DAYS=-1")
	}
	if len(homeDir) > 0 {
		cmd = append(cmd, "-d", path.Join(homeDir, user))
	}
	output, err := procutils.NewCommand(cmd[0], cmd[1:]...).Output()
	if err != nil {
		log.Errorf("Useradd fail: %s, %s", err, output)
		return fmt.Errorf("%s", output)
	} else {
		log.Infof("Useradd: %s", output)
	}
	return nil
}

func (f *SLocalGuestFS) FileGetContents(sPath string, caseInsensitive bool) ([]byte, error) {
	sPath = f.GetLocalPath(sPath, caseInsensitive)
	return f.FileGetContentsByPath(sPath)
}

func (f *SLocalGuestFS) FileGetContentsByPath(sPath string) ([]byte, error) {
	if len(sPath) > 0 {
		return ioutil.ReadFile(sPath)
	}
	return nil, fmt.Errorf("Cann't find local path")
}

func (f *SLocalGuestFS) FilePutContents(sPath, content string, modAppend, caseInsensitive bool) error {
	sFilePath := f.GetLocalPath(sPath, caseInsensitive)
	if len(sFilePath) > 0 {
		sPath = sFilePath
	} else {
		dirPath := f.GetLocalPath(path.Dir(sPath), caseInsensitive)
		if len(dirPath) > 0 {
			sPath = path.Join(dirPath, path.Base(sPath))
		}
	}
	if len(sPath) > 0 {
		return fileutils2.FilePutContents(sPath, content, modAppend)
	} else {
		return fmt.Errorf("Can't put content to empty Path")
	}
}

func (f *SLocalGuestFS) GenerateSshHostKeys() error {
	for _, cmd := range [][]string{
		{f.mountPath, "touch", "/dev/null"},
		{f.mountPath, "/usr/bin/ssh-keygen", "-A"},
	} {
		output, err := procutils.NewCommand("chroot", cmd...).Output()
		if err != nil {
			return errors.Wrapf(err, "GenerateSshHostKeys %s %s", strings.Join(cmd, " "), output)
		}
	}
	return nil
}
