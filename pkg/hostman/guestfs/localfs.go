package guestfs

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

type SLocalGuestFS struct {
	mountPath string
}

func (f *SLocalGuestFS) SupportSerialPorts() bool {
	return false
}

func (f *SLocalGuestFS) getLocalPath(sPath string, caseInsensitive bool) string {
	var fullPath = f.mountPath
	pathSegs := strings.Split(sPath, "/")
	for _, seg := range pathSegs {
		if len(seg) > 0 {
			var realSeg string
			files, _ := ioutil.ReadDir(fullPath)
			for _, file := range files {
				var f = file.Name()
				if f == seg || (caseInsensitive && (strings.ToLower(f)) == strings.ToLower(seg)) ||
					(seg[len(seg)-1] == '*' && strings.HasPrefix(f, seg[:len(seg)-1])) ||
					(caseInsensitive && strings.HasPrefix(strings.ToLower(f),
						strings.ToLower(seg[:]))) {
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
	path = f.getLocalPath(path, caseInsensitive)
	if len(path) > 0 {
		os.Remove(path)
	}
}

func (f *SLocalGuestFS) Mkdir(sPath string, mode int, caseInsensitive bool) error {
	segs := strings.Split(sPath, "/")
	sPath = ""
	pPath := f.getLocalPath("/", caseInsensitive)
	for _, s := range segs {
		if len(s) > 0 {
			sPath = path.Join(sPath, s)
			vPath := f.getLocalPath(sPath, caseInsensitive)
			if len(vPath) > 0 {
				if err := os.Mkdir(path.Join(pPath, s), os.FileMode(mode)); err != nil {
					return err
				}
				pPath = f.getLocalPath(sPath, caseInsensitive)
			} else {
				pPath = vPath
			}
		}
	}
	return nil
}

func (f *SLocalGuestFS) ListDir(sPath string, caseInsensitive bool) []string {
	sPath = f.getLocalPath(sPath, caseInsensitive)
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
	sPath := f.getLocalPath(dir, caseInsensitive)
	if len(sPath) > 0 {
		return fileutils2.Cleandir(sPath, keepdir)
	}
	return fmt.Errorf("No such file %s", sPath)
}

func (f *SLocalGuestFS) Zerofiles(dir string, caseInsensitive bool) error {
	sPath := f.getLocalPath(dir, caseInsensitive)
	if len(sPath) > 0 {
		return fileutils2.Zerofiles(sPath)
	}
	return fmt.Errorf("No such file %s", sPath)
}

// TODO: test
func (f *SLocalGuestFS) Passwd(account, password string, caseInsensitive bool) error {
	var proc = exec.Command("chroot", f.mountPath, "passwd", account)
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
	sPath := f.getLocalPath(usrDir, caseInsensitive)
	if len(sPath) > 0 {
		fileInfo, err := os.Stat(sPath)
		if err != nil {
			log.Errorln(err)
		}
		return fileInfo
	}
	return nil
}

func (f *SLocalGuestFS) Exists(sPath string, caseInsensitive bool) bool {
	sPath = f.getLocalPath(sPath, caseInsensitive)
	if len(sPath) > 0 {
		if _, err := os.Stat(sPath); !os.IsNotExist(err) {
			return true
		}
	}
	return false
}

func (f *SLocalGuestFS) Chown(sPath string, uid, gid int, caseInsensitive bool) error {
	sPath = f.getLocalPath(sPath, caseInsensitive)
	if len(sPath) > 0 {
		return os.Chown(sPath, uid, gid)
	}
	return nil
}

func (f *SLocalGuestFS) Chmod(sPath string, mode uint32, caseInsensitive bool) error {
	sPath = f.getLocalPath(sPath, caseInsensitive)
	if len(sPath) > 0 {
		return os.Chmod(sPath, os.FileMode(mode))
	}
	return nil
}

func (f *SLocalGuestFS) UserAdd(user string, caseInsensitive bool) error {
	output, err := exec.Command("chroot", f.mountPath, "useradd", "-m", "-s", "/bin/bash", user).Output()
	if err != nil {
		log.Errorf("Useradd fail: %s", err)
		return err
	} else {
		log.Infof("Useradd: %s", output)
	}
	return nil
}

func (f *SLocalGuestFS) FileGetContents(sPath string, caseInsensitive bool) ([]byte, error) {
	sPath = f.getLocalPath(sPath, caseInsensitive)
	if len(sPath) > 0 {
		return ioutil.ReadFile(sPath)
	}
	return nil, fmt.Errorf("Cann't find local path")
}

func (f *SLocalGuestFS) FilePutContents(sPath, content string, modAppend, caseInsensitive bool) error {
	sFilePath := f.getLocalPath(sPath, caseInsensitive)
	if len(sFilePath) > 0 {
		sPath = sFilePath
	} else {
		dirPath := f.getLocalPath(path.Dir(sPath), caseInsensitive)
		if len(dirPath) > 0 {
			sPath = path.Join(dirPath, path.Base(sPath))
		}
	}
	if len(sPath) > 0 {
		return fileutils2.FilePutContents(sPath, content, modAppend)
	} else {
		return fmt.Errorf("Cann't put content")
	}
}

func NewLocalGuestFS(mountPath string) *SLocalGuestFS {
	var ret = new(SLocalGuestFS)
	ret.mountPath = mountPath
	return ret
}
