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

package winutils

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/regutils2"
)

var _CHNTPW_PATH string

func SetChntpwPath(spath string) {
	_CHNTPW_PATH = spath
}

func GetChntpwPath() string {
	if len(_CHNTPW_PATH) == 0 {
		_CHNTPW_PATH = "/usr/local/bin/chntpw.static"
	}
	return _CHNTPW_PATH
}

const (
	SYSTEM   = "system"
	SOFTWARE = "software"
	SECURITY = "security"
	SAM      = "sam"
	CONFIRM  = "y"
)

func NewWinRegTool(spath string) *SWinRegTool {
	return &SWinRegTool{ConfigPath: spath}
}

func CheckTool(spath string) bool {
	return procutils.NewCommand(spath, "-h").Run() == nil
}

type SWinRegTool struct {
	ConfigPath   string
	SystemPath   string
	SoftwarePath string
	SamPath      string
	SecurityPath string
}

func (w *SWinRegTool) CheckPath() bool {
	files, err := ioutil.ReadDir(w.ConfigPath)
	if err != nil {
		log.Errorln(err)
		return false
	}

	for _, file := range files {
		switch strings.ToLower(file.Name()) {
		case SYSTEM:
			w.SystemPath = path.Join(w.ConfigPath, file.Name())
		case SOFTWARE:
			w.SoftwarePath = path.Join(w.ConfigPath, file.Name())
		case SAM:
			w.SamPath = path.Join(w.ConfigPath, file.Name())
		case SECURITY:
			w.SecurityPath = path.Join(w.ConfigPath, file.Name())
		}
	}
	if len(w.SystemPath) > 0 && len(w.SoftwarePath) > 0 &&
		len(w.SamPath) > 0 && len(w.SecurityPath) > 0 {
		return true
	}
	return false
}

func (w *SWinRegTool) GetUsers() map[string]bool {
	output, err := procutils.NewCommand(GetChntpwPath(), "-l", w.SamPath).Output()
	if err != nil {
		log.Errorln(err)
		return nil
	}

	users := map[string]bool{}
	re := regexp.MustCompile(
		`\|\s*\w+\s*\|\s*(?P<user>\w+)\s*\|\s*(ADMIN)?\s*\|\s*(?P<lock>(dis/lock|\*BLANK\*)?)`)
	for _, line := range strings.Split(string(output), "\n") {
		m := regutils2.GetParams(re, line)
		if len(m) > 0 {
			user, _ := m["user"]
			if strings.ToLower(user) != "guest" {
				lock, _ := m["lock"]
				users[user] = lock != "dis/lock"
			}
		}
	}
	return users
}

func (w *SWinRegTool) samChange(user string, seq ...string) error {
	proc := procutils.NewCommand(GetChntpwPath(), "-u", user, w.SamPath, w.SystemPath, w.SecurityPath)
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

	io.WriteString(stdin, "n\n")
	for _, s := range seq {
		io.WriteString(stdin, s+"\n")
	}
	io.WriteString(stdin, CONFIRM+"\n")
	stdoutPut, err := ioutil.ReadAll(outb)
	if err != nil {
		return err
	}
	stderrOutPut, err := ioutil.ReadAll(errb)
	if err != nil {
		return err
	}
	log.Debugf("Sam change %s %s", stdoutPut, stderrOutPut)

	done := make(chan error, 1)
	go func() {
		done <- proc.Wait()
	}()
	select {
	case <-time.After(time.Millisecond * 100):
		proc.Kill()
		return fmt.Errorf("Failed to change SAM password, not exit cleanly")
	case err := <-done:
		if err != nil {
			if exitStatus, ok := procutils.GetExitStatus(err); ok {
				if exitStatus == 2 {
					return nil
				}
			}
			log.Errorf("Failed to change SAM password")
			return err
		} else {
			return nil
		}
	}
}

func (w *SWinRegTool) ChangePassword(user, password string) error {
	return w.samChange(user, "2", password)
}

func (w *SWinRegTool) RemovePassword(user string) error {
	return w.samChange(user, "2")
}

func (w *SWinRegTool) UnlockUser(user string) error {
	return w.samChange(user, "4")
}

func (w *SWinRegTool) GetRegFile(regPath string) (string, []string) {
	re := regexp.MustCompile(`\\`)
	vals := re.Split(regPath, -1)
	regSeg := []string{}
	for _, val := range vals {
		if len(val) > 0 {
			regSeg = append(regSeg, val)
		}
	}

	if regSeg[0] == "HKLM" {
		regSeg = regSeg[1:]
	}
	if strings.ToLower(regSeg[0]) == SOFTWARE {
		regSeg = regSeg[1:]
		return w.SoftwarePath, regSeg
	} else if strings.ToLower(regSeg[0]) == SYSTEM {
		regSeg = regSeg[1:]
		return w.SystemPath, regSeg
	} else {
		return "", nil
	}
}

func (w *SWinRegTool) showRegistry(spath string, keySeg []string, verb string) ([]string, error) {
	proc := procutils.NewCommand(GetChntpwPath(), spath)
	stdin, err := proc.StdinPipe()
	if err != nil {
		return nil, err
	}
	defer stdin.Close()

	outb, err := proc.StdoutPipe()
	if err != nil {
		return nil, err
	}
	defer outb.Close()

	// errb, err := proc.StderrPipe()
	// if err != nil {
	// 	return nil, err
	// }
	// defer errb.Close()

	if err := proc.Start(); err != nil {
		return nil, err
	}
	keypath := strings.Join(keySeg, "\\")
	io.WriteString(stdin, fmt.Sprintf("%s %s\n", verb, keypath))
	io.WriteString(stdin, "q\n")
	stdoutPut, err := ioutil.ReadAll(outb)
	if err != nil {
		return nil, err
	}
	time.Sleep(time.Millisecond * 100)

	done := make(chan error, 1)
	go func() {
		done <- proc.Wait()
	}()
	select {
	case <-time.After(time.Millisecond * 100):
		proc.Kill()
	case err := <-done:
		if err != nil {
			return nil, err
		}
	}
	return strings.Split(string(stdoutPut), "\n"), nil
}

func (w *SWinRegTool) getRegistry(spath string, keySeg []string) string {
	keyPath := strings.Join(keySeg, "\\")
	lines, err := w.showRegistry(spath, keySeg, "cat")
	if err != nil {
		log.Errorln(err)
		return ""
	}
	for i, line := range lines {
		if len(keyPath) > 85 {
			keyPath = keyPath[:85]
		}
		if strings.Contains(line, fmt.Sprintf("> Value <%s> of type REG_", keyPath)) {
			return lines[i+1]
		}
	}
	return ""
}

type sRegistry struct {
	Key  string
	Type string
	Size string
}

func (w *SWinRegTool) listRegistry(spath string, keySeg []string) ([]string, []sRegistry, error) {
	lines, err := w.showRegistry(spath, keySeg, "ls")
	if err != nil {
		return nil, nil, err
	}
	keys := []string{}
	values := []sRegistry{}
	keyPattern := regexp.MustCompile("^<(?P<key>[^>]+)>$")
	valPattern := regexp.MustCompile(`^(?P<size>\d+)\s+(?P<type>REG\_\w+)\s+<(?P<key>[^>]+)>\s*`)
	for _, line := range lines {
		m := regutils2.GetParams(keyPattern, line)
		if len(m) > 0 {
			keys = append(keys, m["key"])
		}
		m = regutils2.GetParams(valPattern, line)
		if len(m) > 0 {
			values = append(values, sRegistry{m["key"], m["type"], m["size"]})
		}
	}

	return keys, values, nil
}

func (w *SWinRegTool) cmdRegistry(spath string, ops []string, retcode int) bool {
	proc := procutils.NewCommand(GetChntpwPath(), "-e", spath)
	stdin, err := proc.StdinPipe()
	if err != nil {
		log.Errorln(err)
		return false
	}
	defer stdin.Close()

	outb, err := proc.StdoutPipe()
	if err != nil {
		log.Errorln(err)
		return false
	}
	defer outb.Close()

	errb, err := proc.StderrPipe()
	if err != nil {
		log.Errorln(err)
		return false
	}
	defer errb.Close()

	if err := proc.Start(); err != nil {
		log.Errorln(err)
		return false
	}

	for _, op := range ops {
		io.WriteString(stdin, op+"\n")
	}
	io.WriteString(stdin, "q\n")
	io.WriteString(stdin, CONFIRM+"\n")
	stdoutPut, err := ioutil.ReadAll(outb)
	if err != nil {
		log.Errorln(err)
		return false
	}
	stderrOutPut, err := ioutil.ReadAll(errb)
	if err != nil {
		log.Errorln(err)
		return false
	}
	log.Debugf("Cmd registry %s %s", stdoutPut, stderrOutPut)

	done := make(chan error, 1)
	go func() {
		done <- proc.Wait()
	}()
	select {
	case <-time.After(time.Millisecond * 100):
		proc.Kill()
	case err := <-done:
		if err != nil {
			if exitStatus, ok := procutils.GetExitStatus(err); ok {
				if exitStatus == retcode {
					return true
				}
			}
		} else {
			return retcode == 0
		}
	}
	return false
}

func (w *SWinRegTool) setRegistry(spath string, keySeg []string, value string) bool {
	keyPath := strings.Join(keySeg, "\\")
	return w.cmdRegistry(spath, []string{fmt.Sprintf("ed %s", keyPath), value}, 0)
}

func (w *SWinRegTool) mkdir(spath string, keySeg []string) bool {
	return w.cmdRegistry(spath,
		[]string{
			fmt.Sprintf("cd %s", strings.Join(keySeg[:len(keySeg)-1], "\\")),
			fmt.Sprintf("nk %s", keySeg[len(keySeg)-1]),
		}, 2)
}

func (w *SWinRegTool) keyExists(spath string, keySeg []string) bool {
	keys, _, err := w.listRegistry(spath, keySeg[:len(keySeg)-1])
	if err != nil {
		log.Errorln(err)
		return false
	}
	if utils.IsInStringArray(keySeg[len(keySeg)-1], keys) {
		return true
	}
	return false
}

func (w *SWinRegTool) valExists(spath string, keySeg []string) bool {
	_, vals, err := w.listRegistry(spath, keySeg[:len(keySeg)-1])
	if err != nil {
		log.Errorln(err)
		return false
	}
	for _, val := range vals {
		if val.Key == keySeg[len(keySeg)-1] {
			return true
		}
	}
	return false
}

func (w *SWinRegTool) mkdir_P(spath string, keySeg []string) bool {
	seg := []string{}
	for _, k := range keySeg {
		seg = append(seg, k)
		if !w.keyExists(spath, seg) {
			if !w.mkdir(spath, seg) {
				return false
			}
		}
	}
	return true
}

func (w *SWinRegTool) newValue(spath string, keySeg []string, regtype, val string) bool {
	REG_TYPE_TBL := []string{
		"REG_NONE",
		"REG_SZ",
		"REG_EXPAND_SZ",
		"REG_BINARY",
		"REG_DWORD",
		"REG_DWORD_BIG_ENDIAN",
		"REG_LINK",
		"REG_MULTI_SZ",
		"REG_RESOUCE_LIST",
		"REG_FULL_RES_DESC",
		"REG_RES_REQ",
		"REG_QWORD",
	}

	ok, idx := utils.InStringArray(regtype, REG_TYPE_TBL)
	if !ok {
		return false
	}

	cmds := []string{
		fmt.Sprintf("cd %s", strings.Join(keySeg[:len(keySeg)-1], "\\")),
		fmt.Sprintf("nv %x %s", idx, keySeg[len(keySeg)-1]),
		fmt.Sprintf("ed %s", keySeg[len(keySeg)-1]),
	}

	if regtype == "REG_QWORD" {
		cmds = append(cmds, "16", ": 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0", "s")
	} else {
		cmds = append(cmds, val)
	}
	return w.cmdRegistry(spath, cmds, 0)
}

func (w *SWinRegTool) GetRegistry(keyPath string) string {
	p1, p2s := w.GetRegFile(keyPath)
	if len(p1) == 0 && len(p2s) == 0 {
		return ""
	} else {
		return w.getRegistry(p1, p2s)
	}
}

func (w *SWinRegTool) ListRegistry(keyPath string) ([]string, []sRegistry) {
	p1, p2s := w.GetRegFile(keyPath)
	if len(p1) == 0 && len(p2s) == 0 {
		return nil, nil
	} else {
		v1, v2, err := w.listRegistry(p1, p2s)
		if err != nil {
			log.Errorln(err)
			return nil, nil
		}
		return v1, v2
	}
}

func (w *SWinRegTool) SetRegistry(keyPath, value, regtype string) bool {
	p1, p2s := w.GetRegFile(keyPath)
	if len(p1) == 0 && len(p2s) == 0 {
		return false
	} else {
		if w.valExists(p1, p2s) {
			return w.setRegistry(p1, p2s, value)
		} else {
			if !w.keyExists(p1, p2s[:len(p2s)-1]) {
				if !w.mkdir_P(p1, p2s[:len(p2s)-1]) {
					return false
				}
			}
			return w.newValue(p1, p2s, regtype, value)
		}
	}
}

func (w *SWinRegTool) KeyExists(keyPath string) bool {
	p1, p2s := w.GetRegFile(keyPath)
	if len(p1) == 0 && len(p2s) == 0 {
		return false
	} else {
		return w.keyExists(p1, p2s)
	}
}

func (w *SWinRegTool) MkdirP(keyPath string) bool {
	p1, p2s := w.GetRegFile(keyPath)
	if len(p1) == 0 && len(p2s) == 0 {
		return false
	} else {
		return w.mkdir_P(p1, p2s)
	}
}

func (w *SWinRegTool) GetCcsKey() string {
	ver := w.GetRegistry(`HKLM\SYSTEM\Select\Current`)
	iv, _ := strconv.ParseInt(ver, 16, 0)
	return fmt.Sprintf("ControlSet%03d", iv)
}

func (w *SWinRegTool) GetCcsKeyPath() string {
	return fmt.Sprintf(`HKLM\SYSTEM\%s`, w.GetCcsKey())
}

func (w *SWinRegTool) getComputerNameKeyPath() string {
	key := w.GetCcsKey()
	return key + `\Control\ComputerName\ComputerName\ComputerName`
}

func (w *SWinRegTool) GetComputerName() string {
	key := w.getComputerNameKeyPath()
	return w.GetRegistry(key)
}

func (w *SWinRegTool) setComputerName(cn string) {
	MAX_COMPUTER_NAME_LEN := 15
	COMMON_PREFIX_LEN := 10
	if len(cn) > MAX_COMPUTER_NAME_LEN {
		suffix := cn[COMMON_PREFIX_LEN:]
		suffixlen := MAX_COMPUTER_NAME_LEN - COMMON_PREFIX_LEN
		md5sum := md5.Sum([]byte(suffix))
		cn = cn[:COMMON_PREFIX_LEN] + string(md5sum[:])[:suffixlen]
	}
	key := w.getComputerNameKeyPath()
	w.SetRegistry(key, cn, "")
}

func (w *SWinRegTool) SetHostname(hostname, domain string) {
	tcpipKey := w.GetCcsKeyPath() + `\Services\Tcpip\Parameters`
	hnKey := tcpipKey + `\Hostname`
	dmKey := tcpipKey + `\Domain`
	nvHnKey := tcpipKey + `\NV Hostname`
	nvDmKey := tcpipKey + `\NV Domain`
	w.SetRegistry(hnKey, hostname, "REG_SZ")
	w.SetRegistry(dmKey, domain, "REG_SZ")
	w.SetRegistry(nvHnKey, hostname, "REG_SZ")
	w.SetRegistry(nvDmKey, domain, "REG_SZ")
}

func (w *SWinRegTool) SetDnsServer(nameserver, searchlist string) {
	tcpipKey := w.GetCcsKeyPath() + `\Services\Tcpip\Parameters`
	ns_key := tcpipKey + `\NameServer`
	search_key := tcpipKey + `\SearchList`
	w.SetRegistry(ns_key, nameserver, "REG_SZ")
	w.SetRegistry(search_key, searchlist, "REG_SZ")
}

func (w *SWinRegTool) GetProductName() string {
	prodKey := `HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\ProductName`
	return w.GetRegistry(prodKey)
}

func (w *SWinRegTool) GetVersion() string {
	prodKey := `HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\CurrentVersion`
	return w.GetRegistry(prodKey)
}

func (w *SWinRegTool) GetInstallLanguage() string {
	nlsTbl := map[string]string{"0804": "zh_CN", "0404": "zh_TW", "0c04": "zh_HK",
		"1004": "zh_SG", "0409": "en_US", "0809": "en_UK"}
	key := w.GetCcsKeyPath()
	key += `\Control\Nls\Language\InstallLanguage`
	val := w.GetRegistry(key)
	if xval, ok := nlsTbl[key]; ok {
		return xval
	} else {
		return val
	}
}

func (w *SWinRegTool) GetArch() string {
	prodKey := `HKLM\SOFTWARE\Wow6432Node\Microsoft\Windows NT\CurrentVersion\CurrentVersion`
	ver := w.GetRegistry(prodKey)
	if len(ver) > 0 {
		return "x86_64"
	} else {
		return "x86"
	}
}

func (w *SWinRegTool) LogontypePath() string {
	return `HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon\LogonType`
}

func (w *SWinRegTool) GetLogontype() string {
	return w.GetRegistry(w.LogontypePath())
}

func (w *SWinRegTool) SetLogontype(val string) {
	w.SetRegistry(w.LogontypePath(), val, "")
}

func (w *SWinRegTool) DefaultAccountPath() string {
	return `HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon\DefaultUserName`
}

func (w *SWinRegTool) GetDefaultAccount() string {
	return w.GetRegistry(w.DefaultAccountPath())
}

func (w *SWinRegTool) SetDefaultAccount(user string) {
	w.SetRegistry(w.DefaultAccountPath(), user, "")
}

func (w *SWinRegTool) GpeditScriptPath() string {
	return `HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Group Policy\Scripts`
}

func (w *SWinRegTool) GpeditScriptStatePath() string {
	return `HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Group Policy\State\Machine\Scripts`
}

func (w *SWinRegTool) GetGpeditStartScripts() []string {
	scriptKey := w.GpeditScriptPath() + `\Startup\0`
	keys, _ := w.ListRegistry(scriptKey)
	ret := []string{}
	for _, k := range keys {
		spath := scriptKey + (fmt.Sprintf(`\%s\Script`, k))
		val := w.GetRegistry(spath)
		ret = append(ret, val)
	}
	return ret
}

func (w *SWinRegTool) IsGpeditStartScriptInstalled(script string) bool {
	scripts := w.GetGpeditStartScripts()
	return utils.IsInStringArray(script, scripts)
}

func (w *SWinRegTool) InstallGpeditStartScript(script string) {
	if w.IsGpeditStartScriptInstalled(script) {
		return
	}
	w.installGpeditStartScript(script, w.GpeditScriptPath())
	w.installGpeditStartScript(script, w.GpeditScriptStatePath())
}

func (w *SWinRegTool) GetGpoDisplayname() string {
	spath := `HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Group Policy\State\Machine\GPO-List\0\DisplayName`
	return w.GetRegistry(spath)
}

func (w *SWinRegTool) installGpeditStartScript(script, scriptPath string) {
	idx := 0
	if !w.KeyExists(scriptPath + `\Startup`) {
		w.MkdirP(scriptPath + `\Startup`)
		w.MkdirP(scriptPath + `\Shutdown`)
		dsname := "Local Group Policy"
		kvts := [][3]string{
			{"GPO-ID", "LocalGPO", "REG_SZ"},
			{"SOM-ID", "Local", "REG_SZ"},
			{"FileSysPath", `C:\Windows\System32\GroupPolicy\Machine`, "REG_SZ"},
			{"DisplayName", dsname, "REG_SZ"},
			{"GPOName", dsname, "REG_SZ"},
			{"PSScriptOrder", "1", "REG_DWORD"},
		}
		for _, kvt := range kvts {
			w.SetRegistry(fmt.Sprintf(`%s\Startup\0\%s`, scriptPath, kvt[0]), kvt[1], kvt[2])
		}

	} else {
		for w.KeyExists(scriptPath + (fmt.Sprintf(`\Startup\0\%d`, idx))) {
			idx += 1
		}
	}

	kvts := [][3]string{
		{"Script", script, "REG_SZ"},
		{"Parameters", "", "REG_SZ"},
		{"ExecTime", "", "REG_QWORD"},
		{"IsPowershell", "0", "REG_DWORD"},
	}
	for _, kvt := range kvts {
		w.SetRegistry(fmt.Sprintf(`%s\Startup\0\%d\%s`, scriptPath, idx, kvt[0]), kvt[1], kvt[2])
	}
}

func (w *SWinRegTool) EnableRdp() {
	key := w.GetCcsKeyPath() + `\Control\Terminal Server\fDenyTSConnections`
	w.SetRegistry(key, "0", `REG_DWORD`)
	key = w.GetCcsKeyPath() + `\Services\MpsSvc\Start`
	w.SetRegistry(key, "3", `REG_DWORD`)
}
