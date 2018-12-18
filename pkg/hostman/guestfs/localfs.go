package guestfs

import (
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/sshkeys"
	"yunion.io/x/onecloud/pkg/hostman"
)

type SLocalGuestFS struct {
	mountPath string
	readOnly  bool
}

func (f *SLocalGuestFS) IsReadonly() bool {
	log.Infof("Test if read-only fs ...")
	var filename = fmt.Sprint("./%f", rand.Float32())
	if err := f.FilePutContents(filename, fmt.Sprint("%f", rand.Float32()), false, false); err == nil {
		f.Remove(filename, false)
		return false
	} else {
		log.Errorf("File system is readonly: %s", err)
		f.readOnly = true
		return true
	}
}

func (f *SLocalGuestFS) GetReadonly() bool {
	return f.readOnly
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
						strings.ToLower(seg[:len(seg)]))) {
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

func (f *SLocalGuestFS) Listdir(sPath string, caseInsensitive bool) []string {
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
		return hostman.Cleandir(sPath, keepdir)
	}
	return fmt.Errorf("No such file %s", sPath)
}

// TODO
func (f *SLocalGuestFS) Zerofiles(dir string, caseInsensitive bool) error {
	sPath := f.getLocalPath(dir, caseInsensitive)
	if len(sPath) > 0 {
		//写到这里了。。。
		return hostman.Zerofiles(sPath)
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
	output, err := exec.Command("chroot", f.mountPath, "useradd", user).Output()
	if err != nil {
		log.Errorf("Useradd fail: %s", err)
		return err
	} else {
		log.Infof("Useradd: %s", output)
	}
	return nil
}

func (f *SLocalGuestFS) MergeAuthorizedKeys(oldKeys string, pubkeys *sshkeys.SSHKeys) string {
	var allkeys = make(map[string]string, 0)
	if len(oldKeys) > 0 {
		for _, line := range strings.Split(oldKeys, "\n") {
			line = strings.TrimSpace(line)
			dat := strings.Split(line, " ")
			if len(dat) > 1 {
				if _, ok := allkeys[dat[1]]; !ok {
					allkeys[dat[1]] = line
				}
			}
		}
	}
	if len(pubkeys.DeletePublicKey) > 0 {
		dat := strings.Split(pubkeys.DeletePublicKey, " ")
		if len(dat) > 1 {
			if _, ok := allkeys[dat[1]]; ok {
				delete(allkeys, dat[1])
			}
		}
	}
	for _, k := range []string{pubkeys.PublicKey, pubkeys.AdminPublicKey, pubkeys.ProjectPublicKey} {
		if len(k) > 0 {
			k = strings.TrimSpace(k)
			dat := strings.Split(k, " ")
			if len(dat) > 1 {
				if _, ok := allkeys[dat[1]]; !ok {
					allkeys[dat[1]] = k
				}
			}
		}
	}
	var keys = make([]string, len(allkeys))
	for key, _ := range allkeys {
		keys = append(keys, key)
	}
	return strings.Join(keys, "\n")
}

func (f *SLocalGuestFS) DeployAuthorizedKeys(usrDir string, pubkeys *sshkeys.SSHKeys, replace bool) error {
	usrStat := f.Stat(usrDir, false)
	if usrStat != nil {
		sshDir := path.Join(usrDir, ".ssh")
		authFile := path.Join(sshDir, "authorized_keys")
		modeRwxOwner := syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IXUSR
		modeRwOwner := syscall.S_IRUSR | syscall.S_IWUSR
		fStat := usrStat.Sys().(*syscall.Stat_t)
		if !f.Exists(sshDir, false) {
			err := f.Mkdir(sshDir, modeRwxOwner, false)
			if err != nil {
				return err
			}
			err = f.Chown(sshDir, int(fStat.Uid), int(fStat.Gid), false)
			if err != nil {
				return err
			}
		}
		var oldKeys = ""
		if !replace {
			bOldKeys, _ := f.FileGetContents(authFile, false)
			oldKeys = string(bOldKeys)
		}
		newKeys := f.MergeAuthorizedKeys(oldKeys, pubkeys)
		err := f.FilePutContents(authFile, newKeys, false, false)
		if err != nil {
			return err
		}
		err = f.Chown(authFile, int(fStat.Uid), int(fStat.Gid), false)
		if err != nil {
			return err
		}
		return f.Chmod(authFile, uint32(modeRwOwner), false)
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
		return hostman.FilePutContents(sPath, content, modAppend)
	} else {
		return fmt.Errorf("Cann't put content")
	}
}

func NewLocalGuestFS(mountPath string) *SLocalGuestFS {
	var ret = new(SLocalGuestFS)
	ret.mountPath = mountPath
	return ret
}
