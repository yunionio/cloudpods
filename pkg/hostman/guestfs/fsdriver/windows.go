package fsdriver

import (
	"fmt"
	"math/rand"
	"path"
	"regexp"
	"strings"
	"syscall"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudcommon/sshkeys"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/netutils2"
	"yunion.io/x/onecloud/pkg/util/seclib2"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/onecloud/pkg/util/version"
	"yunion.io/x/onecloud/pkg/util/winutils"
)

const (
	TCPIP_PARAM_KEY      = `HKLM\SYSTEM\CurrentControlSet\Services\Tcpip\Parameters`
	BOOT_SCRIPT_PATH     = "/Windows/System32/GroupPolicy/Machine/Scripts/Startup/cloudboot.bat"
	WIN_BOOT_SCRIPT_PATH = "cloudboot.bat"
)

type SWindowsRootFs struct {
	*sGuestRootFsDriver

	guestDebugLogPath string
	bootScripts       string
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

func (w *SWindowsRootFs) DeployPublicKey(IDiskPartition, string, *sshkeys.SSHKeys) error {
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

func (w *SWindowsRootFs) GetReleaseInfo(IDiskPartition) *SReleaseInfo {
	confPath := w.rootFs.GetLocalPath("/windows/system32/config", true)
	tool := winutils.NewWinRegTool(confPath)
	if tool.CheckPath() {
		distro := tool.GetProductName()
		version := tool.GetVersion()
		arch := tool.GetArch()
		lan := tool.GetInstallLanguage()
		return &SReleaseInfo{distro, version, arch, lan}
	} else {
		return nil
	}
}

func (w *SWindowsRootFs) GetLoginAccount(rootFs IDiskPartition, defaultRootUser bool, windowsDefaultAdminUser bool) string {
	confPath := w.rootFs.GetLocalPath("/windows/system32/config", true)
	tool := winutils.NewWinRegTool(confPath)
	tool.CheckPath()
	users := tool.GetUsers()
	admin := "Administrator"
	selUsr := ""
	if w.IsWindows10() {
		delete(users, admin)
	}
	if _, ok := users[admin]; ok && windowsDefaultAdminUser {
		selUsr = admin
	} else {
		for user := range users {
			if user != admin && (len(selUsr) == 0 || len(selUsr) > len(user)) {
				selUsr = user
			}
		}
		if _, ok := users[admin]; ok && len(selUsr) == 0 {
			selUsr = admin
		}
	}
	if len(selUsr) > 0 {
		if _, ok := users[selUsr]; !ok {
			tool.UnlockUser(selUsr)
		}
	}
	return selUsr
}

func (w *SWindowsRootFs) IsWindows10() bool {
	info := w.GetReleaseInfo(nil)
	if info != nil && strings.HasPrefix(info.Distro, "Windows 10 ") {
		return true
	}
	return false
}

func (w *SWindowsRootFs) GetOs() string {
	return "Windows"
}

func (w *SWindowsRootFs) appendGuestBootScript(content string) string {
	w.bootScripts += "\r\n" + content
	return w.bootScripts
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
	w.appendGuestBootScript(bootScript)

	lines := []string{}
	for k, v := range map[string]string{
		"Hostname":    hostname,
		"Domain":      domain,
		"NV Hostname": hostname,
		"NV Domain":   domain,
	} {
		lines = append(lines, w.regAdd(TCPIP_PARAM_KEY, k, v, "REG_SZ"))
	}
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
		hf.Add(ip, fmt.Sprintf("%s.%s", hn, domain), hn)
	}
	return w.rootFs.FilePutContents(ETC_HOSTS, hf.String(), false, true)
}

func (w *SWindowsRootFs) DeployNetworkingScripts(rootfs IDiskPartition, nics []jsonutils.JSONObject) error {
	mainNic, err := netutils2.GetMainNic(nics)
	if err != nil {
		return err
	}
	mainIp := ""
	if mainNic != nil {
		mainIp, _ = mainNic.GetString("ip")
	}
	bootScript := strings.Join([]string{
		`set NETCFG_SCRIPT=%SystemRoot%\netcfg.bat`,
		`if exist %NETCFG_SCRIPT% (`,
		`    call %NETCFG_SCRIPT%`,
		`    del %NETCFG_SCRIPT%`,
		`)`,
	}, "\r\n")
	w.appendGuestBootScript(bootScript)
	lines := []string{
		"@echo off",
		w.MakeGuestDebugCmd("netcfg step 1"),
		"setlocal enableDelayedExpansion",
		`for /f "delims=" %%a in ('getmac /fo csv /nh /v') do (`,
		`  set line=%%a&set line=!line:"=,!`,
		`  for /f "delims=,,, tokens=1,3" %%b in ("!line!") do (`,
	}

	for _, nic := range nics {
		snic := &types.SServerNic{}
		if err := nic.Unmarshal(snic); err != nil {
			log.Errorln(err)
			return err
		}

		mac := snic.Mac
		mac = strings.Replace(strings.ToUpper(mac), ":", "-", -1)
		lines = append(lines, fmt.Sprintf(`    if "%%%%c" == "%s" (`, mac))
		if jsonutils.QueryBoolean(nic, "manual", false) {
			netmask := netutils2.Netlen2Mask(snic.Masklen)
			cfg := fmt.Sprintf(`      netsh interface ip set address "%%%%b" static %s %s`, snic.Ip, netmask)
			if len(snic.Gateway) > 0 && snic.Ip == mainIp {
				cfg += fmt.Sprintf(" %s", snic.Gateway)
			}
			lines = append(lines, cfg)
			routes := [][]string{}
			netutils2.AddNicRoutes(&routes, snic, mainIp, len(nics), options.HostOptions.PrivatePrefixes)
			for _, r := range routes {
				lines = append(lines, fmt.Sprintf(`      netsh interface ip add route %s "%%%%b" %s`, r[0], r[1]))
			}
			dnslist := netutils2.GetNicDns(snic)
			if len(dnslist) > 0 {
				lines = append(lines, fmt.Sprintf(
					`      netsh interface ip set dns name="%%%%b" source=static addr=%s ddns=disabled suffix=interface`, dnslist[0]))
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
	lines = append(lines, `netsh advfirewall firewall set rule group=\"remote desktop\" new enable=yes`)
	netScript := strings.Join(lines, "\r\n")
	return w.putGuestScriptContents("/windows/netcfg.bat", netScript)
}

func (w *SWindowsRootFs) MakeGuestDebugCmd(content string) string {
	mark := "============="
	content = regexp.MustCompile(`(["^&<>|])`).ReplaceAllString(content, "^$1")
	return fmt.Sprintf("echo %s %s %s >> %s", mark, content, mark, w.guestDebugLogPath)
}

func (w *SWindowsRootFs) prependGuestBootScript(content string) {
	w.bootScripts = content + "\r\n" + w.bootScripts
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
	tool.InstallGpeditStartScript(WIN_BOOT_SCRIPT_PATH)
	if err := w.rootFs.Mkdir(path.Dir(BOOT_SCRIPT_PATH), syscall.S_IRUSR|syscall.S_IWUSR|syscall.S_IXUSR, true); err != nil {
		return err
	}
	return w.rootFs.FilePutContents(BOOT_SCRIPT_PATH, w.bootScripts, false, false)
}

func (w *SWindowsRootFs) ChangeUserPasswd(part IDiskPartition, account, gid, publicKey, password string) (string, error) {
	rinfo := w.GetReleaseInfo(part)
	confPath := part.GetLocalPath("/windows/system32/config", true)
	tool := winutils.NewWinRegTool(confPath)
	tool.CheckPath()
	success := false
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
		log.Errorf("Filaed Password %s", account)
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
	w.prependGuestBootScript(bootScript)
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
	cmds = append(cmds, "Net start wuauserv")
	cmds = append(cmds, "wuauclt /detectnow")
	cmds = append(cmds, `del %SystemRoot%\chgpwd_setup.ps1`)
	cmds = append(cmds, `del %SystemRoot%\Setup\Scripts\SetupComplete.cmd`)
	if w.putGuestScriptContents(SETUP_SCRIPT_PATH, strings.Join(cmds, "\r\n")) != nil {
		return false
	}
	return true
}

func (w *SWindowsRootFs) DeployFstabScripts(rootFs IDiskPartition, disks []jsonutils.JSONObject) error {
	if len(disks) == 1 {
		return nil
	}

	bootScript := strings.Join([]string{
		`set MOUNT_DISK_SCRIPT=%SystemRoot%\mountdisk.bat`,
		`if exist %MOUNT_DISK_SCRIPT% (`,
		`    call %MOUNT_DISK_SCRIPT%`,
		`    del %MOUNT_DISK_SCRIPT%`,
		`)`,
	}, "\r\n")
	w.appendGuestBootScript(bootScript)
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
