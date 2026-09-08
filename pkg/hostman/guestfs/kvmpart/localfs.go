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
	"os"
	"os/exec"
	"path"
	"path/filepath"
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
	p, err := f.resolveLocalPath(sPath, caseInsensitive, false)
	if err != nil {
		return ""
	}
	return p
}

func (f *SLocalGuestFS) resolveLocalPath(sPath string, caseInsensitive bool, allowMissingLast bool) (string, error) {
	if sPath == "." {
		sPath = ""
	}
	mount := filepath.Clean(f.mountPath)
	fullPath := mount
	segs := strings.Split(sPath, "/")
	for i, seg := range segs {
		if len(seg) == 0 || seg == "." {
			continue
		}
		if seg == ".." {
			parent := filepath.Dir(fullPath)
			if !fileutils2.IsPathInside(mount, parent) {
				return "", errors.Errorf("path %q is outside mount", sPath)
			}
			fullPath = parent
			continue
		}
		isLast := i == len(segs)-1
		realSeg, fi, err := f.lookupSeg(fullPath, seg, caseInsensitive)
		if err != nil {
			return "", err
		}
		if realSeg == "" {
			if allowMissingLast && isLast {
				joined := filepath.Join(fullPath, seg)
				if !fileutils2.IsPathInside(mount, joined) {
					return "", errors.Errorf("path %q is outside mount", sPath)
				}
				return joined, nil
			}
			return "", errors.Errorf("path %q not found", sPath)
		}
		next := filepath.Join(fullPath, realSeg)
		if !fileutils2.IsPathInside(mount, next) {
			return "", errors.Errorf("path %q is outside mount", sPath)
		}
		if fi != nil && !isLast {
			if fi.Mode()&os.ModeSymlink != 0 {
				return "", errors.Errorf("path %q traverses a symlink", sPath)
			}
			if !fi.IsDir() {
				return "", errors.Errorf("path %q traverses a non-directory", sPath)
			}
		}
		fullPath = next
	}
	if !fileutils2.IsPathInside(mount, fullPath) {
		return "", errors.Errorf("path %q is outside mount", sPath)
	}
	return fullPath, nil
}

func (f *SLocalGuestFS) lookupSeg(dir, seg string, caseInsensitive bool) (string, os.FileInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", nil, err
	}
	var realSeg string
	for _, entry := range entries {
		name := entry.Name()
		match := name == seg
		if !match && caseInsensitive && strings.EqualFold(name, seg) {
			match = true
		}
		if !match && len(seg) > 0 && seg[len(seg)-1] == '*' {
			prefix := seg[:len(seg)-1]
			match = strings.HasPrefix(name, prefix) ||
				(caseInsensitive && strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix)))
		}
		if match {
			realSeg = name
			break
		}
	}
	if realSeg == "" {
		return "", nil, nil
	}
	fi, err := os.Lstat(filepath.Join(dir, realSeg))
	if err != nil {
		return "", nil, err
	}
	return realSeg, fi, nil
}

func (f *SLocalGuestFS) mustRegularDir(p string) error {
	fi, err := os.Lstat(p)
	if err != nil {
		return err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return errors.Errorf("%s is a symlink", p)
	}
	if !fi.IsDir() {
		return errors.Errorf("%s is not a directory", p)
	}
	return nil
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
	if err := f.mustRegularDir(pPath); err != nil {
		return err
	}
	for _, s := range segs {
		if len(s) > 0 {
			sPath = path.Join(sPath, s)
			vPath := f.GetLocalPath(sPath, caseInsensitive)
			if len(vPath) == 0 {
				if err := f.mustRegularDir(pPath); err != nil {
					return err
				}
				if err := os.Mkdir(path.Join(pPath, s), os.FileMode(mode)); err != nil {
					return err
				}
				pPath = f.GetLocalPath(sPath, caseInsensitive)
			} else {
				if err := f.mustRegularDir(vPath); err != nil {
					return err
				}
				pPath = vPath
			}
		}
	}
	return nil
}

func (f *SLocalGuestFS) ListDir(sPath string, caseInsensitive bool) []string {
	sPath = f.GetLocalPath(sPath, caseInsensitive)
	if len(sPath) > 0 {
		if err := f.mustRegularDir(sPath); err != nil {
			log.Errorln(err)
			return nil
		}
		files, err := os.ReadDir(sPath)
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
	var proc = exec.Command("chroot", f.mountPath, "passwd", account)

	passwordInput := fmt.Sprintf("%s\n%s\n", password, password)
	proc.Stdin = strings.NewReader(passwordInput)

	out, err := proc.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "failed change passwd %s", out)
	}
	log.Infof("Passwd %s", out)
	return nil
}

func (f *SLocalGuestFS) Stat(usrDir string, caseInsensitive bool) os.FileInfo {
	sPath := f.GetLocalPath(usrDir, caseInsensitive)
	if len(sPath) > 0 {
		fileInfo, err := os.Lstat(sPath)
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
	if err := f.mustRegularDir(dir); err != nil {
		return err
	}
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
		if fileutils2.IsSymlink(sPath) {
			return errors.Errorf("cannot chown symlink %s", sPath)
		}
		return os.Chown(sPath, uid, gid)
	}
	return nil
}

func (f *SLocalGuestFS) Chmod(sPath string, mode uint32, caseInsensitive bool) error {
	sPath = f.GetLocalPath(sPath, caseInsensitive)
	if len(sPath) > 0 {
		if fileutils2.IsSymlink(sPath) {
			return errors.Errorf("cannot chmod symlink %s", sPath)
		}
		return os.Chmod(sPath, os.FileMode(mode))
	}
	return nil
}

func (f *SLocalGuestFS) updateUserEtcShadow(username string) error {
	sPath := f.GetLocalPath("/etc/shadow", false)
	if !fileutils2.Exists(sPath) {
		return nil
	}
	if fileutils2.IsSymlink(sPath) {
		return errors.Errorf("cannot update symlink %s", sPath)
	}
	contentBytes, err := fileutils2.FileGetContentsNoFollow(sPath)
	if err != nil {
		return errors.Wrap(err, "read /etc/shadow")
	}
	content := string(contentBytes)

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
	err = fileutils2.FilePutContentsNoFollow(sPath, newContent, false)
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
	if len(sPath) == 0 {
		return nil, fmt.Errorf("Cann't find local path")
	}
	if fileutils2.IsSymlink(sPath) {
		return nil, errors.Errorf("cannot read symlink %s", sPath)
	}
	return fileutils2.FileGetContentsNoFollow(sPath)
}

func (f *SLocalGuestFS) FilePutContents(sPath, content string, modAppend, caseInsensitive bool) error {
	target, err := f.resolveLocalPath(sPath, caseInsensitive, true)
	if err != nil {
		return err
	}
	if fileutils2.IsSymlink(target) {
		return errors.Errorf("cannot write through symlink %s", sPath)
	}
	parent := filepath.Dir(target)
	if err := f.mustRegularDir(parent); err != nil {
		return err
	}
	return fileutils2.FilePutContentsNoFollow(target, content, modAppend)
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
