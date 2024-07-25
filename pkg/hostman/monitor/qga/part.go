package qga

import (
	"bytes"
	"encoding/base64"
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

	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
)

var _ fsdriver.IDiskPartition = &QemuGuestAgentPartition{}

type QemuGuestAgentPartition struct {
	agent *QemuGuestAgent
}

func NewQGAPartition(agent *QemuGuestAgent) *QemuGuestAgentPartition {
	return &QemuGuestAgentPartition{
		agent: agent,
	}
}

func (qga *QemuGuestAgent) CommandWithTimeout(
	cmdPath string, args, env []string, inputData string, captureOutput bool,
	timeoutSecond int,
) (int, string, string, error) {
	pid, err := qga.GuestExecCommand(cmdPath, args, env, inputData, captureOutput)
	if err != nil {
		return -1, "", "", errors.Wrap(err, "GuestExecCommand")
	}

	if timeoutSecond <= 0 {
		timeoutSecond = QGA_EXEC_DEFAULT_WAIT_TIMEOUT
	}
	for i := 0; i < timeoutSecond; i++ {
		execStatus, err := qga.GuestExecStatusCommand(pid.Pid)
		if err != nil {
			return -1, "", "", errors.Wrap(err, "GuestExecStatusCommand")
		}
		if !execStatus.Exited {
			time.Sleep(time.Second)
		} else {
			if captureOutput {
				var stdout, stderr string
				if len(execStatus.OutData) > 0 {
					stdoutb, err := base64.StdEncoding.DecodeString(execStatus.OutData)
					if err != nil {
						return -1, "", "", errors.Wrap(err, "base64.StdEncoding.DecodeString")
					}
					stdout = string(stdoutb)
				}
				if len(execStatus.ErrData) > 0 {
					stderrb, err := base64.StdEncoding.DecodeString(execStatus.ErrData)
					if err != nil {
						return -1, "", "", errors.Wrap(err, "base64.StdEncoding.DecodeString")
					}
					stderr = string(stderrb)
				}

				return execStatus.Exitcode, stdout, stderr, nil
			} else {
				return execStatus.Exitcode, "", "", nil
			}
		}
	}
	return -1, "", "", errors.Errorf("QGA guest-exec wait process exit timeout after wait %d second", timeoutSecond)
}

func (qga *QemuGuestAgent) FileGetContents(path string) (string, error) {
	fileno, err := qga.QgaFileOpen(path, "r")
	if err != nil {
		return "", err
	}
	defer func() {
		if e := qga.QgaFileClose(fileno); e != nil {
			log.Errorf("failed close path %s: %s", path, e)
		}
	}()

	var buf bytes.Buffer
	for {
		content, eof, err := qga.QgaFileRead(fileno, -1)
		if err != nil {
			return "", err
		}
		log.Debugf("read %s", content)
		if len(content) > 0 {
			buf.Write(content)
		}
		if eof {
			break
		}
	}
	return buf.String(), nil
}

func (qga *QemuGuestAgent) FilePutContents(path, content string, modAppend bool) error {
	var mode = "w+"
	if modAppend {
		mode = "a+"
	}
	fileno, err := qga.QgaFileOpen(path, mode)
	if err != nil {
		return err
	}
	defer func() {
		if e := qga.QgaFileClose(fileno); e != nil {
			log.Errorf("failed close path %s: %s", path, e)
		}
	}()

	var buf = bytes.NewBufferString(content)
	for {
		log.Debugf("write %s", buf.String())
		count, _, err := qga.QgaFileWrite(fileno, buf.String())
		if err != nil {
			return err
		}
		if count > 0 {
			buf.Next(count)
		} else {
			if buf.Len() > 0 {
				return errors.Errorf("qga file put content remaining %d", buf.Len())
			} else {
				break
			}
		}
	}
	return nil
}

func (p *QemuGuestAgentPartition) osPathExists(path string) (bool, error) {
	retCode, _, _, err := p.agent.CommandWithTimeout("test", []string{"-e", path}, nil, "", false, -1)
	if err != nil {
		return false, errors.Wrap(err, "CommandWithTimeout")
	}
	if retCode == 0 {
		return true, nil
	}
	retCode, _, _, err = p.agent.CommandWithTimeout("test", []string{"-L", path}, nil, "", false, -1)
	if err != nil {
		return false, errors.Wrap(err, "CommandWithTimeout")
	}
	return retCode == 0, nil
}

func (p *QemuGuestAgentPartition) osIsDir(path string) (bool, error) {
	retCode, _, _, err := p.agent.CommandWithTimeout("test", []string{"-d", path}, nil, "", false, -1)
	if err != nil {
		return false, errors.Wrap(err, "CommandWithTimeout")
	}
	return retCode == 0, nil
}

func (p *QemuGuestAgentPartition) osListDir(path string) ([]string, error) {
	if ok, err := p.osIsDir(path); err != nil {
		return nil, err
	} else if !ok {
		return nil, errors.Errorf("Path %s is not dir", path)
	}

	retcode, stdout, stderr, err := p.agent.CommandWithTimeout("ls", []string{"-a", path}, nil, "", true, -1)
	if err != nil {
		return nil, errors.Wrapf(err, "command ls -a %s", path)
	}
	if retcode > 0 {
		return nil, errors.Errorf("failed guest-exec ls -a %s: %s %s %d", path, stdout, stderr, retcode)
	}

	files := []string{}
	strfiles := strings.Split(stdout, "\n")
	for _, f := range strfiles {
		f = strings.TrimSpace(f)
		if !utils.IsInStringArray(f, []string{"", ".", ".."}) {
			files = append(files, f)
		}
	}
	return files, nil
}

func (p *QemuGuestAgentPartition) GetLocalPath(sPath string, caseInsensitive bool) string {
	if sPath == "." {
		sPath = ""
	}
	if !caseInsensitive {
		return sPath
	}

	var fullPath = "/"
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
			if f == seg || (caseInsensitive && (strings.ToLower(f)) == strings.ToLower(seg)) {
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
	log.Debugf("QGA GetLocalPath %s=>%s", sPath, fullPath)
	return fullPath
}

func (p *QemuGuestAgentPartition) FileGetContents(sPath string, caseInsensitive bool) ([]byte, error) {
	sPath = p.GetLocalPath(sPath, caseInsensitive)
	return p.FileGetContentsByPath(sPath)
}

func (p *QemuGuestAgentPartition) FileGetContentsByPath(sPath string) ([]byte, error) {
	res, err := p.agent.FileGetContents(sPath)
	if err != nil {
		return nil, err
	}
	return []byte(res), nil
}

func (p *QemuGuestAgentPartition) FilePutContents(sPath, content string, modAppend, caseInsensitive bool) error {
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
		return p.agent.FilePutContents(sPath, content, modAppend)
	} else {
		return errors.Errorf("Can't put content to empty Path")
	}
}

func (p *QemuGuestAgentPartition) Exists(sPath string, caseInsensitive bool) bool {
	sPath = p.GetLocalPath(sPath, caseInsensitive)
	if len(sPath) > 0 {
		exists, err := p.osPathExists(sPath)
		if err != nil {
			log.Errorf("QGA failed detect path exist %s", err)
		}
		return exists
	}
	return false
}

func (p *QemuGuestAgentPartition) Chown(sPath string, uid, gid int, caseInsensitive bool) error {
	sPath = p.GetLocalPath(sPath, caseInsensitive)
	if len(sPath) == 0 {
		return errors.Errorf("Can't get local path: %s", sPath)
	}
	args := []string{fmt.Sprintf("%d.%d", uid, gid), sPath}
	retCode, stdout, stderr, err := p.agent.CommandWithTimeout("chown", args, nil, "", true, -1)
	if err != nil {
		return errors.Wrap(err, "CommandWithTimeout")
	}
	if retCode != 0 {
		return errors.Errorf("QGA chown failed %s %s, retCode %d", stdout, stderr, retCode)
	}
	return nil
}

func (p *QemuGuestAgentPartition) Chmod(sPath string, mode uint32, caseInsensitive bool) error {
	sPath = p.GetLocalPath(sPath, caseInsensitive)
	if sPath == "" {
		return nil
	}
	modeStr := fmt.Sprintf("%o", mode&0777)
	args := []string{modeStr, sPath}
	retCode, stdout, stderr, err := p.agent.CommandWithTimeout("chmod", args, nil, "", true, -1)
	if err != nil {
		return errors.Wrap(err, "CommandWithTimeout")
	}
	if retCode != 0 {
		return errors.Errorf("QGA chmod failed %s %s, retCode %d", stdout, stderr, retCode)
	}
	return nil
}

func (p *QemuGuestAgentPartition) CheckOrAddUser(user, homeDir string, isSys bool) (string, error) {
	// check user
	retCode, stdout, stderr, err := p.agent.CommandWithTimeout("/bin/cat", []string{"/etc/passwd"}, nil, "", true, -1)
	if err != nil {
		return "", errors.Wrap(err, "CommandWithTimeout")
	}
	if retCode != 0 {
		return "", errors.Errorf("QGA cat passwd failed %s %s, retCode %d", stdout, stderr, retCode)
	}
	var exist = false
	var realHomeDir = ""
	lines := strings.Split(stdout, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		userInfos := strings.Split(strings.TrimSpace(lines[i]), ":")
		if len(userInfos) < 6 {
			continue
		}
		if userInfos[0] != user {
			continue
		}
		exist = true
		realHomeDir = userInfos[5]
		break
	}
	if exist {
		args := []string{"-R", "/", "-E", "-1", "-m", "0", "-M", "99999", "-I", "-1", user}
		retCode, stdout, stderr, err = p.agent.CommandWithTimeout("chage", args, nil, "", true, -1)
		if err != nil {
			return "", errors.Wrap(err, "CommandWithTimeout")
		}
		if retCode != 0 && !strings.Contains(stderr, "not found") {
			return "", errors.Errorf("failed chage %s %s", stdout, stderr)
		}
		if !p.Exists(realHomeDir, false) {
			err = p.Mkdir(realHomeDir, 0700, false)
			if err != nil {
				return "", errors.Wrapf(err, "Mkdir %s", realHomeDir)
			}
			retCode, stdout, stderr, err = p.agent.CommandWithTimeout("chown", []string{user, realHomeDir}, nil, "", true, -1)
			if err != nil {
				return "", errors.Wrap(err, "CommandWithTimeout")
			}
			if retCode != 0 {
				return "", errors.Errorf("failed chown %s %s: %s %s, retcode %d", user, realHomeDir, stdout, stderr, retCode)
			}
		}
		return realHomeDir, nil
	}
	return path.Join(homeDir, user), p.userAdd(user, homeDir, isSys)
}

func (p *QemuGuestAgentPartition) userAdd(user, homeDir string, isSys bool) error {
	args := []string{"-m", "-s", "/bin/bash", user}
	if isSys {
		args = append(args, "-r", "-e", "''", "-f", "'-1'", "-K", "'PASS_MAX_DAYS=-1'")
	}
	if len(homeDir) > 0 {
		args = append(args, "-d", path.Join(homeDir, user))
	}
	retCode, stdout, stderr, err := p.agent.CommandWithTimeout("useradd", args, nil, "", true, -1)
	if err != nil {
		return errors.Wrap(err, "CommandWithTimeout")
	}
	if retCode != 0 {
		return errors.Errorf("failed useradd %s: %s %s, retcode %d", user, stdout, stderr, retCode)
	}
	return nil
}

func (p *QemuGuestAgentPartition) Stat(sPath string, caseInsensitive bool) os.FileInfo {
	sPath = p.GetLocalPath(sPath, caseInsensitive)
	if len(sPath) == 0 {
		return nil
	}
	args := []string{"-a", "-l", "-n", "-i", "-s", "-d", sPath}
	retCode, stdout, stderr, err := p.agent.CommandWithTimeout("ls", args, nil, "", true, -1)
	if err != nil {
		log.Errorf("CommandWithTimeout %s", err)
		return nil
	}
	if retCode != 0 {
		log.Errorf("failed ls %s: %s %s, retcode %d", sPath, stdout, stderr, retCode)
		return nil
	}
	ret := strings.Split(stdout, "\n")
	for _, line := range ret {
		dat := regexp.MustCompile(`\s+`).Split(strings.TrimSpace(line), -1)
		if len(dat) > 7 && ((dat[2][0] != 'l' && dat[len(dat)-1] == sPath) ||
			(dat[2][0] == 'l' && dat[len(dat)-3] == sPath)) {
			stMode, err := fsdriver.ModeStr2Bin(dat[2])
			if err != nil {
				log.Errorf("ModeStr2Bin %s", err)
				return nil
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
			return info
		}
	}
	return nil
}

func (p *QemuGuestAgentPartition) Symlink(src, dst string, caseInsensitive bool) error {
	odstDir := path.Dir(dst)
	if err := p.Mkdir(odstDir, 0755, caseInsensitive); err != nil {
		return errors.Wrapf(err, "Mkdir %s", odstDir)
	}
	if p.Exists(dst, caseInsensitive) {
		p.Remove(dst, caseInsensitive)
	}
	odstDir = p.GetLocalPath(odstDir, caseInsensitive)
	dst = path.Join(odstDir, path.Base(dst))
	retCode, stdout, stderr, err := p.agent.CommandWithTimeout("ln", []string{"-s", src, dst}, nil, "", true, -1)
	if err != nil {
		return errors.Wrap(err, "CommandWithTimeout")
	}
	if retCode != 0 {
		return errors.Errorf("failed ln -s %s %s: %s %s, retcode %d", src, dst, stdout, stderr, retCode)
	}
	return nil
}

func (p *QemuGuestAgentPartition) Passwd(account, password string, caseInsensitive bool) error {
	return p.agent.GuestSetUserPassword(account, password, false)
}

func (p *QemuGuestAgentPartition) Mkdir(sPath string, mode int, caseInsensitive bool) error {
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

func (p *QemuGuestAgentPartition) osMkdirP(dir string, mode uint32) error {
	retCode, stdout, stderr, err := p.agent.CommandWithTimeout("mkdir", []string{"-p", dir}, nil, "", true, -1)
	if err != nil {
		return errors.Wrap(err, "CommandWithTimeout")
	}
	if retCode != 0 {
		return errors.Errorf("mkdir -p %s: %s %s: retCode: %d", dir, stdout, stderr, retCode)
	}
	return p.Chmod(dir, mode, false)
}

func (p *QemuGuestAgentPartition) ListDir(sPath string, caseInsensitive bool) []string {
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

func (p *QemuGuestAgentPartition) Remove(sPath string, caseInsensitive bool) {
	sPath = p.GetLocalPath(sPath, caseInsensitive)
	if len(sPath) > 0 {
		retCode, stdout, stderr, err := p.agent.CommandWithTimeout("rm", []string{sPath}, nil, "", true, -1)
		if err != nil {
			log.Errorf("remove %s: %s", sPath, err)
			return
		}
		if retCode != 0 {
			log.Errorf("remove %s: %s %s: retCode: %d", sPath, stdout, stderr, retCode)
			return
		}
	}
}

func (p *QemuGuestAgentPartition) Cleandir(dir string, keepdir, caseInsensitive bool) error {
	sPath := p.GetLocalPath(dir, caseInsensitive)
	if len(sPath) > 0 {
		retCode, stdout, stderr, err := p.agent.CommandWithTimeout("rm", []string{"-rf", sPath}, nil, "", true, -1)
		if err != nil {
			return errors.Wrapf(err, "remove -rf %s", sPath)
		}
		if retCode != 0 {
			return errors.Wrapf(err, "remove -rf %s: %s %s: retCode: %d", sPath, stdout, stderr, retCode)
		}
	}
	return nil
}

func (*QemuGuestAgentPartition) Zerofiles(dir string, caseInsensitive bool) error {
	return nil
}

func (*QemuGuestAgentPartition) SupportSerialPorts() bool {
	return false
}

func (*QemuGuestAgentPartition) GetPartDev() string {
	return "QGA"
}

func (*QemuGuestAgentPartition) IsMounted() bool {
	return true
}

func (*QemuGuestAgentPartition) Mount() bool {
	return true
}

func (*QemuGuestAgentPartition) MountPartReadOnly() bool {
	return false
}

func (*QemuGuestAgentPartition) Umount() error {
	return nil
}

func (*QemuGuestAgentPartition) GetMountPath() string {
	return ""
}

func (*QemuGuestAgentPartition) IsReadonly() bool {
	return false
}

func (*QemuGuestAgentPartition) GetPhysicalPartitionType() string {
	return ""
}

func (*QemuGuestAgentPartition) Zerofree() {
}

func (*QemuGuestAgentPartition) GenerateSshHostKeys() error {
	return nil
}
