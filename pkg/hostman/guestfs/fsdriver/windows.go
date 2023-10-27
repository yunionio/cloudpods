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

package fsdriver

import (
	"fmt"
	"math/rand"
	"path"
	"regexp"
	"strings"
	"syscall"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/version"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/netutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/seclib2"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/onecloud/pkg/util/winutils"
)

const (
	ACTIVE_COMPUTER_NAME_KEY = `HKLM\SYSTEM\CurrentControlSet\Control\ComputerName\ActiveComputerName`
	COMPUTER_NAME_KEY        = `HKLM\SYSTEM\CurrentControlSet\Control\ComputerName\ComputerName`

	TCPIP_PARAM_KEY      = `HKLM\SYSTEM\CurrentControlSet\Services\Tcpip\Parameters`
	BOOT_SCRIPT_PATH     = "/Windows/System32/GroupPolicy/Machine/Scripts/Startup/cloudboot.bat"
	WIN_BOOT_SCRIPT_PATH = "cloudboot"

	WIN_TELEGRAF_BINARY_PATH = "/opt/yunion/bin/telegraf.exe"
	WIN_TELEGRAF_PATH        = "/Program Files/Telegraf"
)

type SWindowsRootFs struct {
	*sGuestRootFsDriver

	guestDebugLogPath string

	bootScript  string
	bootScripts map[string]string
}

func NewWindowsRootFs(part IDiskPartition) IRootFsDriver {
	seq := []byte{'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l',
		'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z'}
	suffix := make([]byte, 16)
	lenSeq := len(seq)
	for i := 0; i < 16; i++ {
		suffix[i] = seq[rand.Intn(lenSeq)]
	}
	return &SWindowsRootFs{
		sGuestRootFsDriver: newGuestRootFsDriver(part),
		guestDebugLogPath:  `%SystemRoot%\mdbg_` + string(suffix),
		bootScripts:        make(map[string]string),
	}
}

func (w *SWindowsRootFs) IsFsCaseInsensitive() bool {
	return true
}

func (w *SWindowsRootFs) GetName() string {
	return "Windows"
}

func (w *SWindowsRootFs) String() string {
	return "WindowsRootFs"
}

func (w *SWindowsRootFs) DeployPublicKey(IDiskPartition, string, *deployapi.SSHKeys) error {
	return nil
}

func (w *SWindowsRootFs) RootSignatures() []string {
	return []string{
		"/program files", "/windows",
		"/windows/system32/drivers/etc", "/windows/system32/config",
		"/windows/system32/config/sam",
		"/windows/system32/config/software",
		"/windows/system32/config/system",
	}
}

func (w *SWindowsRootFs) GetReleaseInfo(IDiskPartition) *deployapi.ReleaseInfo {
	confPath := w.rootFs.GetLocalPath("/windows/system32/config", true)
	tool := winutils.NewWinRegTool(confPath)
	if tool.CheckPath() {
		distro := tool.GetProductName()
		version := tool.GetVersion()
		arch := w.GetArch(hostCpuArch)
		lan := tool.GetInstallLanguage()
		return &deployapi.ReleaseInfo{
			Distro:   distro,
			Version:  version,
			Arch:     arch,
			Language: lan,
		}
	} else {
		return nil
	}
}

func (w *SWindowsRootFs) GetLoginAccount(rootFs IDiskPartition, sUser string, defaultRootUser bool, windowsDefaultAdminUser bool) (string, error) {
	confPath := w.rootFs.GetLocalPath("/windows/system32/config", true)
	tool := winutils.NewWinRegTool(confPath)
	tool.CheckPath()
	users := tool.GetUsers()
	admin := "Administrator"
	selUsr := ""
	isWin10NonPro := w.IsWindows10NonPro()
	// Win10 try not to use Administrator users // Win 10 professional can use Adminsitrator
	if _, ok := users[admin]; ok && windowsDefaultAdminUser && !isWin10NonPro && w.GetIRootFsDriver().AllowAdminLogin() {
		selUsr = admin
	} else {
		// Looking for an unlocked user who is not an Administrator
		for user, ok := range users {
			if !ok {
				continue
			}
			if user != admin {
				selUsr = user
			}
		}
		if _, ok := users[admin]; ok && len(selUsr) == 0 {
			selUsr = admin
		}
	}
	if selUsr == "" {
		return "", fmt.Errorf("no unlocked user")
	}
	if !users[selUsr] {
		// user is locked
		tool.UnlockUser(selUsr)
	}
	return selUsr, nil
}

func (w *SWindowsRootFs) GetArch(hostCpuArch string) string {
	if w.rootFs.Exists("/program files (x86)", true) {
		return apis.OS_ARCH_X86_64
	} else if w.rootFs.Exists("/program files (arm)", true) {
		return apis.OS_ARCH_AARCH64
	}
	if hostCpuArch == apis.OS_ARCH_AARCH32 {
		return apis.OS_ARCH_AARCH32
	} else {
		return apis.OS_ARCH_X86_32
	}
}

func (w *SWindowsRootFs) IsWindows10NonPro() bool {
	info := w.GetReleaseInfo(nil)
	if info != nil && strings.HasPrefix(info.Distro, "Windows 10 ") && !strings.HasPrefix(info.Distro, "Windows 10 Pro") {
		return true
	}
	return false
}

func (w *SWindowsRootFs) IsOldWindows() bool {
	info := w.GetReleaseInfo(nil)
	if info != nil && strings.HasPrefix(info.Version, "5.") {
		return true
	}
	return false
}

func (w *SWindowsRootFs) GetOs() string {
	return "Windows"
}

func (w *SWindowsRootFs) appendGuestBootScript(name, content string) {
	w.bootScript += "\r\n" + fmt.Sprintf("start %s", name)
	w.bootScripts[name] = content
}

func (w *SWindowsRootFs) regAdd(path, key, val, regType string) string {
	return fmt.Sprintf(`REG ADD %s /V "%s" /D "%s" /T %s /F`, path, key, val, regType)
}

func (w *SWindowsRootFs) putGuestScriptContents(spath, content string) error {
	contentArr := []string{}
	contentLen := len(content)

	var j = 0
	if content[0] == '\n' {
		contentArr = append(contentArr, "")
		j += 1
	}

	for i := 1; i < contentLen; i++ {
		if content[i] == '\n' && content[i-1] != '\r' {
			contentArr = append(contentArr, content[j:i])
			j = i + 1
		}
	}
	if j < contentLen {
		contentArr = append(contentArr, content[j:])
	} else {
		contentArr = append(contentArr, "")
	}

	content = strings.Join(contentArr, "\r\n")
	return w.rootFs.FilePutContents(spath, content, false, true)
}

func (w *SWindowsRootFs) DeployHostname(part IDiskPartition, hostname, domain string) error {
	bootScript := strings.Join([]string{
		`set HOSTNAME_SCRIPT=%SystemRoot%\hostnamecfg.bat`,
		`if exist %HOSTNAME_SCRIPT% (`,
		`    call %HOSTNAME_SCRIPT%`,
		`    del %HOSTNAME_SCRIPT%`,
		`)`,
	}, "\r\n")
	w.appendGuestBootScript("hostnamecfg", bootScript)

	lines := []string{}
	for k, v := range map[string]string{
		"Hostname":    hostname,
		"Domain":      domain,
		"NV Hostname": hostname,
		"NV Domain":   domain,
	} {
		lines = append(lines, w.regAdd(TCPIP_PARAM_KEY, k, v, "REG_SZ"))
	}
	// windows allow a maximal length of 15
	// http://support.microsoft.com/kb/909264
	if len(hostname) > api.MAX_WINDOWS_COMPUTER_NAME_LENGTH {
		hostname = hostname[:api.MAX_WINDOWS_COMPUTER_NAME_LENGTH]
	}
	lines = append(lines, w.regAdd(ACTIVE_COMPUTER_NAME_KEY, "ComputerName", hostname, "REG_SZ"))
	lines = append(lines, w.regAdd(COMPUTER_NAME_KEY, "ComputerName", hostname, "REG_SZ"))
	hostScripts := strings.Join(lines, "\r\n")
	return w.putGuestScriptContents("/windows/hostnamecfg.bat", hostScripts)
}

func (w *SWindowsRootFs) DeployHosts(part IDiskPartition, hn, domain string, ips []string) error {
	var (
		ETC_HOSTS = "/windows/system32/drivers/etc/hosts"
		oldHf     = ""
	)

	if w.rootFs.Exists(ETC_HOSTS, true) {
		oldHfBytes, err := w.rootFs.FileGetContents(ETC_HOSTS, true)
		if err != nil {
			log.Errorln(err)
			return err
		}
		oldHf = string(oldHfBytes)
	}

	hf := fileutils2.HostsFile{}
	hf.Parse(oldHf)
	hf.Add("127.0.0.1", "localhost")
	for _, ip := range ips {
		hf.Add(ip, getHostname(hn, domain), hn)
	}
	return w.rootFs.FilePutContents(ETC_HOSTS, hf.String(), false, true)
}

func (w *SWindowsRootFs) DeployQgaBlackList(part IDiskPartition) error {
	return nil
}

func (w *SWindowsRootFs) DeployNetworkingScripts(rootfs IDiskPartition, nics []*types.SServerNic) error {
	mainNic, err := netutils2.GetMainNicFromDeployApi(nics)
	if err != nil {
		return err
	}
	mainIp := ""
	if mainNic != nil {
		mainIp = mainNic.Ip
	}
	bootScript := strings.Join([]string{
		`set NETCFG_SCRIPT=%SystemRoot%\netcfg.bat`,
		`if exist %NETCFG_SCRIPT% (`,
		`    call %NETCFG_SCRIPT%`,
		`    del %NETCFG_SCRIPT%`,
		`)`,
	}, "\r\n")
	w.appendGuestBootScript("netcfg", bootScript)
	lines := []string{
		"@echo off",
		w.MakeGuestDebugCmd("netcfg step 1"),
		"setlocal enableDelayedExpansion",
		`for /f "delims=" %%a in ('getmac /fo csv /nh /v') do (`,
		`  set line=%%a&set line=!line:"=,!`,
		`  for /f "delims=,,, tokens=1,3" %%b in ("!line!") do (`,
	}

	for _, snic := range nics {
		mac := snic.Mac
		mac = strings.Replace(strings.ToUpper(mac), ":", "-", -1)
		lines = append(lines, fmt.Sprintf(`    if "%%%%c" == "%s" (`, mac))
		if snic.Mtu > 0 {
			lines = append(lines, fmt.Sprintf(`      netsh interface ipv4 set subinterface "%%%%b" mtu=%d`, snic.Mtu))
		}
		if snic.Manual {
			netmask := netutils2.Netlen2Mask(int(snic.Masklen))
			cfg := fmt.Sprintf(`      netsh interface ip set address "%%%%b" static %s %s`, snic.Ip, netmask)
			if len(snic.Gateway) > 0 && snic.Ip == mainIp {
				cfg += fmt.Sprintf(" %s", snic.Gateway)
			}
			lines = append(lines, cfg)
			routes := [][]string{}
			netutils2.AddNicRoutes(&routes, snic, mainIp, len(nics), privatePrefixes)
			for _, r := range routes {
				lines = append(lines, fmt.Sprintf(`      netsh interface ip add route %s "%%%%b" %s`, r[0], r[1]))
			}
			dnslist := netutils2.GetNicDns(snic)
			if len(dnslist) > 0 {
				lines = append(lines, fmt.Sprintf(
					`      netsh interface ip set dns name="%%%%b" source=static addr=%s`, dnslist[0]))
				if len(dnslist) > 1 {
					for i := 1; i < len(dnslist); i++ {
						lines = append(lines, fmt.Sprintf(`      netsh interface ip add dns "%%%%b" %s index=%d`, dnslist[i], i+1))
					}
				}
			}

			if len(snic.Domain) > 0 && snic.Ip == mainIp {
				lines = append(lines, w.regAdd(TCPIP_PARAM_KEY, "SearchList", snic.Domain, "REG_SZ"))
			}
		} else {
			lines = append(lines, `      netsh interface ip set address "%%b" dhcp`)
			lines = append(lines, `      netsh interface ip set dns "%%b" dhcp`)
		}
		lines = append(lines, `    )`)
	}
	lines = append(lines, `  )`)
	lines = append(lines, `)`)
	lines = append(lines, w.MakeGuestDebugCmd("netcfg step 2"))
	// lines = append(lines, `netsh advfirewall firewall set rule group=\"remote desktop\" new enable=yes`)
	netScript := strings.Join(lines, "\r\n")
	return w.putGuestScriptContents("/windows/netcfg.bat", netScript)
}

func (w *SWindowsRootFs) MakeGuestDebugCmd(content string) string {
	mark := "============="
	content = regexp.MustCompile(`(["^&<>|])`).ReplaceAllString(content, "^$1")
	return fmt.Sprintf("echo %s %s %s >> %s", mark, content, mark, w.guestDebugLogPath)
}

func (w *SWindowsRootFs) prependGuestBootScript(content string) {
	w.bootScript = content + "\r\n" + w.bootScript
}

func (w *SWindowsRootFs) PrepareFsForTemplate(IDiskPartition) error {
	for _, f := range []string{"/Pagefile.sys", "/Hiberfil.sys", "/Swapfile.sys"} {
		if w.rootFs.Exists(f, true) {
			w.rootFs.Remove(f, true)
		}
	}
	return nil
}

func (w *SWindowsRootFs) CommitChanges(part IDiskPartition) error {
	confPath := part.GetLocalPath("/windows/system32/config", true)
	tool := winutils.NewWinRegTool(confPath)
	tool.CheckPath()
	tool.EnableRdp()
	tool.ResetUSBProfile()

	if w.IsOldWindows() {
		// windows prior to windows 2003 should not try to commit changes
		return nil
	}

	tool.InstallGpeditStartScript(WIN_BOOT_SCRIPT_PATH)

	bootDir := path.Dir(BOOT_SCRIPT_PATH)
	if err := w.rootFs.Mkdir(bootDir, syscall.S_IRUSR|syscall.S_IWUSR|syscall.S_IXUSR, true); err != nil {
		return err
	}
	if err := w.rootFs.FilePutContents(BOOT_SCRIPT_PATH, w.bootScript, false, false); err != nil {
		return errors.Wrap(err, "write boot script")
	}
	for k, v := range w.bootScripts {
		if err := w.rootFs.FilePutContents(path.Join(bootDir, fmt.Sprintf("%s.bat", k)), v, false, false); err != nil {
			return errors.Wrap(err, "write boot scripts")
		}
	}
	return nil
}

func (w *SWindowsRootFs) ChangeUserPasswd(part IDiskPartition, account, gid, publicKey, password string) (string, error) {
	rinfo := w.GetReleaseInfo(part)
	confPath := part.GetLocalPath("/windows/system32/config", true)
	tool := winutils.NewWinRegTool(confPath)
	tool.CheckPath()
	success := false

	// symbol ^ is escape character is batch file.
	password = strings.ReplaceAll(password, "^", "")
	if rinfo != nil && version.GE(rinfo.Version, "6.1") {
		success = w.deployPublicKeyByGuest(account, password)
	} else {
		success = tool.ChangePassword(account, password) == nil
	}

	var (
		secret string
		err    error
	)
	if success {
		if len(publicKey) > 0 {
			secret, err = seclib2.EncryptBase64(publicKey, password)
			if err != nil {
				return "", err
			}
		} else {
			secret, err = utils.EncryptAESBase64(gid, password)
			if err != nil {
				return "", err
			}
		}
		if rinfo != nil && strings.Contains(rinfo.Distro, "Windows XP") {
			if len(tool.GetLogontype()) > 0 {
				tool.SetLogontype("0x0")
			}
		}
	} else {
		log.Errorf("Failed Password %s", account)
	}
	defUanme := tool.GetDefaultAccount()
	if len(defUanme) > 0 && defUanme != account {
		tool.SetDefaultAccount(account)
	}
	return secret, nil
}

func (w *SWindowsRootFs) deployPublicKeyByGuest(uname, passwd string) bool {
	if !w.deploySetupCompleteScripts(uname, passwd) {
		return false
	}
	bootScript := strings.Join([]string{
		`set CHANGE_PASSWD_SCRIPT=%SystemRoot%\chgpwd.bat`,
		`if exist %CHANGE_PASSWD_SCRIPT% (`,
		`    call %CHANGE_PASSWD_SCRIPT%`,
		`    del %CHANGE_PASSWD_SCRIPT%`,
		`)`,
	}, "\r\n")
	w.appendGuestBootScript("chgpwd", bootScript)
	logPath := w.guestDebugLogPath
	chksum := stringutils2.GetMD5Hash(passwd + logPath[(len(logPath)-10):])

	chgpwdScript := strings.Join([]string{
		w.MakeGuestDebugCmd("change password step 1"),
		strings.Join([]string{
			`%SystemRoot%\System32\WindowsPowerShell\v1.0\powershell.exe`,
			` -executionpolicy bypass %SystemRoot%\chgpwd.ps1`,
			fmt.Sprintf(" %s %s %s %s", uname, passwd, chksum, logPath),
		}, ""),
		`del %SystemRoot%\chgpwd.ps1`,
		w.MakeGuestDebugCmd("change password step 2"),
	}, "\r\n")
	if w.putGuestScriptContents("/windows/chgpwd.bat", chgpwdScript) != nil {
		return false
	}
	if w.putGuestScriptContents("/windows/chgpwd.ps1", WinScriptChangePassword) != nil {
		return false
	}
	return true
}

func (w *SWindowsRootFs) deploySetupCompleteScripts(uname, passwd string) bool {
	SETUP_SCRIPT_PATH := "/Windows/Setup/Scripts/SetupComplete.cmd"
	if !w.rootFs.Exists(path.Dir(SETUP_SCRIPT_PATH), true) {
		w.rootFs.Mkdir(path.Dir(SETUP_SCRIPT_PATH),
			syscall.S_IRUSR|syscall.S_IWUSR|syscall.S_IXUSR, true)
	}
	if w.putGuestScriptContents("/windows/chgpwd_setup.ps1", WinScriptChangePassword) != nil {
		return false
	}
	cmds := []string{
		`%SystemRoot%\System32\WindowsPowerShell\v1.0\powershell.exe -executionpolicy bypass %SystemRoot%\chgpwd_setup.ps1 ` +
			fmt.Sprintf("%s %s", uname, passwd),
		"Net stop wuauserv",
	}
	for _, v := range [][3]string{
		{"AUOptions", "REG_DWORD", "3"},
		{"NoAutoUpdate", "REG_DWORD", "0"},
		{"ScheduledInstallDay", "REG_DWORD", "0"},
		{"ScheduledInstallTime", "REG_DWORD", "4"},
		{"AutoInstallMinorUpdates", "REG_DWORD", "1"},
		{"NoAutoRebootWithLoggedOnUsers", "REG_DWORD", "1"},
		{"IncludeRecommendedUpdates", "REG_DWORD", "0"},
		{"EnableFeaturedSoftware", "REG_DWORD", "1"},
	} {
		cmds = append(cmds, fmt.Sprintf(`REG ADD "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update" /v %s /t %s /d %s /f`,
			v[0], v[1], v[2]))
	}
	cmds = append(cmds, `REG ADD "HKLM\SYSTEM\CurrentControlSet\Control\TimeZoneInformation" /v RealTimeIsUniversal /t REG_DWORD /d 1 /f`)
	cmds = append(cmds, "Net start wuauserv")
	cmds = append(cmds, "wuauclt /detectnow")
	cmds = append(cmds, `del %SystemRoot%\chgpwd_setup.ps1`)
	cmds = append(cmds, `del %SystemRoot%\Setup\Scripts\SetupComplete.cmd`)
	if w.putGuestScriptContents(SETUP_SCRIPT_PATH, strings.Join(cmds, "\r\n")) != nil {
		return false
	}
	return true
}

func (w *SWindowsRootFs) DeployFstabScripts(rootFs IDiskPartition, disks []*deployapi.Disk) error {
	bootScript := strings.Join([]string{
		`set MOUNT_DISK_SCRIPT=%SystemRoot%\mountdisk.bat`,
		`if exist %MOUNT_DISK_SCRIPT% (`,
		`    call %MOUNT_DISK_SCRIPT%`,
		`    del %MOUNT_DISK_SCRIPT%`,
		`)`,
	}, "\r\n")
	w.appendGuestBootScript("mountdisk", bootScript)
	logPath := w.guestDebugLogPath
	mountScript := strings.Join([]string{
		w.MakeGuestDebugCmd("mount disk step 1"),
		`cscript %SystemRoot%\mountdisk.js --debug ` + logPath,
		`del %SystemRoot%\mountdisk.js`,
		w.MakeGuestDebugCmd("mount disk step 2"),
	}, "\r\n")

	if w.putGuestScriptContents("/windows/mountdisk.bat", mountScript) != nil {
		return nil
	}
	w.putGuestScriptContents("/windows/mountdisk.js", WinScriptMountDisk)
	return nil
}

func (w *SWindowsRootFs) DetectIsUEFISupport(part IDiskPartition) bool {
	content, err := w.rootFs.FileGetContents("/windows/panther/setupact.log", true)
	if err != nil {
		log.Errorln(err)
		return false
	}
	contentStr := string(content)
	sep := "Detected boot environment: "
	idx := strings.Index(contentStr, sep)
	if idx < 0 {
		return false
	}
	if strings.HasPrefix(contentStr[idx+len(sep):], "EFI") ||
		strings.HasPrefix(contentStr[idx+len(sep):], "UEFI") {
		return true
	}
	return false
}

func (l *SWindowsRootFs) IsResizeFsPartitionSupport() bool {
	return true
}

func (w *SWindowsRootFs) DeployTelegraf(config string) (bool, error) {
	if err := w.rootFs.Mkdir(WIN_TELEGRAF_PATH, syscall.S_IRUSR|syscall.S_IWUSR|syscall.S_IXUSR, true); err != nil {
		return false, errors.Wrap(err, "mkdir telegraf path")
	}

	telegrafConfPath := path.Join(w.rootFs.GetMountPath(), WIN_TELEGRAF_PATH, "telegraf.conf")
	if err := w.rootFs.FilePutContents(telegrafConfPath, config, false, true); err != nil {
		return false, errors.Wrap(err, "write boot script")
	}
	telegrafConfPath = strings.ReplaceAll(path.Join("%PROGRAMFILES%", "Telegraf", "telegraf.conf"), "/", "\\")

	telegrafBinaryPath := path.Join(w.rootFs.GetMountPath(), WIN_TELEGRAF_PATH, "telegraf.exe")
	output, err := procutils.NewCommand("cp", "-f", WIN_TELEGRAF_BINARY_PATH, telegrafBinaryPath).Output()
	if err != nil {
		return false, errors.Wrapf(err, "cp telegraf failed %s", output)
	}
	telegrafBinaryPath = strings.ReplaceAll(path.Join("%PROGRAMFILES%", "Telegraf", "telegraf.exe"), "/", "\\")
	bootScript := strings.Join([]string{
		`set SETUP_TELEGRAF_SCRIPT=%SystemRoot%\telegraf.bat`,
		`if exist %SETUP_TELEGRAF_SCRIPT% (`,
		`    call %SETUP_TELEGRAF_SCRIPT%`,
		`    del %SETUP_TELEGRAF_SCRIPT%`,
		`)`,
	}, "\r\n")
	w.appendGuestBootScript("telegraf", bootScript)

	setupTelegrafBatScript := strings.Join([]string{
		w.MakeGuestDebugCmd("setup telegraf step 1"),
		strings.Join([]string{
			`%SystemRoot%\System32\WindowsPowerShell\v1.0\powershell.exe`,
			` -executionpolicy bypass %SystemRoot%\telegraf.ps1`,
			fmt.Sprintf(" '%s' '%s' >> %s_telegraf", telegrafBinaryPath, telegrafConfPath, w.guestDebugLogPath),
		}, ""),
		`del %SystemRoot%\telegraf.ps1`,
		w.MakeGuestDebugCmd("setup telegraf step 2"),
	}, "\r\n")
	if err := w.putGuestScriptContents("/windows/telegraf.bat", setupTelegrafBatScript); err != nil {
		return false, errors.Wrap(err, "put setup bat scirpt")
	}
	if err := w.putGuestScriptContents("/windows/telegraf.ps1", winTelegrafSetupPowerShellScript); err != nil {
		return false, errors.Wrap(err, "put setup ps1 script")
	}
	return true, nil
}
