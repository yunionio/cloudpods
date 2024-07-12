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
	"path"
	"strings"

	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/util/macutils"
	"yunion.io/x/onecloud/pkg/util/seclib2"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SMacOSRootFs struct {
	*sGuestRootFsDriver
	scripts []string
}

func NewMacOSRootFs(part IDiskPartition) IRootFsDriver {
	return &SMacOSRootFs{sGuestRootFsDriver: newGuestRootFsDriver(part)}
}

func (m *SMacOSRootFs) IsFsCaseInsensitive() bool {
	return false
}

func (m *SMacOSRootFs) GetName() string {
	return "macOS"
}

func (m *SMacOSRootFs) String() string {
	return "MacOSRootFs"
}

func (m *SMacOSRootFs) GetLoginAccount(rootFs IDiskPartition, user string, defaultRootUser bool, windowsDefaultAdminUser bool) (string, error) {
	selUsr := ""
	usrs := m.rootFs.ListDir("/Users", false)
	if len(usrs) > 0 {
		for _, usr := range usrs {
			if usr != "Shared" && usr[0] != '.' {
				if len(selUsr) < len(usr) {
					selUsr = usr
				}
			}
		}
	}
	return selUsr, nil
}

func (m *SMacOSRootFs) RootSignatures() []string {
	return []string{
		"/Applications", "/Library", "/Network",
		"/System", "/System/Library", "/Volumes", "/Users", "/usr",
		"/private/etc", "/private/var", "/bin", "/sbin", "/dev",
	}
}

func (m *SMacOSRootFs) DeployPublicKey(rootfs IDiskPartition, uname string, pubkeys *deployapi.SSHKeys) error {
	usrDir := fmt.Sprintf("/Users/%s", uname)
	return DeployAuthorizedKeys(m.rootFs, usrDir, pubkeys, false, false)
}

func (m *SMacOSRootFs) addScripts(lines []string) {
	if m.scripts == nil {
		m.scripts = []string{}
	}
	m.scripts = append(m.scripts, lines...)
	m.scripts = append(m.scripts, "")
}

func (m *SMacOSRootFs) ChangeUserPasswd(part IDiskPartition, account, gid, publicKey, password string) (string, error) {
	lines := []string{
		fmt.Sprintf("dscl . -passwd /Users/%s %s", account, password),
		fmt.Sprintf("rm -fr /Users/%s/Library/Keychains/*", account),
	}
	m.addScripts(lines)
	if len(publicKey) > 0 {
		return seclib2.EncryptBase64(publicKey, password)
	} else {
		return utils.EncryptAESBase64(gid, password)
	}
}

func (m *SMacOSRootFs) DeployHostname(part IDiskPartition, hostname, domain string) error {
	lines := []string{
		fmt.Sprintf("scutil --set HostName '%s'",
			stringutils2.EscapeString(hostname, nil)),
		fmt.Sprintf("scutil --set ComputerName '%s'",
			stringutils2.EscapeString(hostname, nil)),
		fmt.Sprintf("scutil --set LocalHostName '%s'",
			stringutils2.EscapeString(hostname, nil)),
	}
	m.addScripts(lines)
	return nil
}

func (m *SMacOSRootFs) DeployHosts(part IDiskPartition, hn, domain string, ips []string) error {
	return nil
}

func (m *SMacOSRootFs) GetReleaseInfo(IDiskPartition) *deployapi.ReleaseInfo {
	spath := "/System/Library/CoreServices/SystemVersion.plist"
	sInfo, _ := m.rootFs.FileGetContents(spath, false)
	info := macutils.ParsePlist(sInfo)
	distro, _ := info["ProductName"]
	version, _ := info["ProductUserVisibleVersion"]
	return &deployapi.ReleaseInfo{
		Distro:  distro,
		Version: version,
		Arch:    apis.OS_ARCH_X86_64,
	}
}

func (m *SMacOSRootFs) GetOs() string {
	return "macOs"
}

func (m *SMacOSRootFs) DeployNetworkingScripts(rootfs IDiskPartition, nics []*types.SServerNic) error {
	return nil
}

func (m *SMacOSRootFs) PrepareFsForTemplate(IDiskPartition) error {
	sshDir := "/private/etc/ssh"
	if m.rootFs.Exists(sshDir, false) {
		for _, f := range m.rootFs.ListDir(sshDir, false) {
			if strings.HasSuffix(f, "_key") || strings.HasSuffix(f, "_key.pub") {
				m.rootFs.Remove(path.Join(sshDir, f), false)
			}
		}
	}
	tmpDirs := []string{"/private/tmp", "/private/var/tmp",
		"/private/var/vm",
		"/System/Library/Caches",
		"/Library/Caches"}
	logDirs := []string{"/var/log", "/Library/Logs"}
	users := m.rootFs.ListDir("/Users", false)
	if len(users) > 0 {
		for _, usr := range users {
			if usr != "Shared" && usr[0] != '.' {
				tmpDirs = append(tmpDirs, fmt.Sprintf("/Users/%s/Library", usr))
			}
		}
	}

	for _, dir := range tmpDirs {
		if m.rootFs.Exists(dir, false) {
			if err := m.rootFs.Cleandir(dir, false, false); err != nil {
				return err
			}
		}
	}

	for _, dir := range logDirs {
		if m.rootFs.Exists(dir, false) {
			if err := m.rootFs.Zerofiles(dir, false); err != nil {
				return err
			}
		}
	}
	for _, dir := range []string{"/private/var/spool", "/private/var/run"} {
		if m.rootFs.Exists(dir, false) {
			if err := m.rootFs.Cleandir(dir, true, false); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *SMacOSRootFs) CommitChanges(part IDiskPartition) error {
	var (
		label = "com.meituan.cloud_init"
		spath = "/private/var/cloud_init.sh"
	)
	cont := macutils.LaunchdRun(label, spath)
	if err := m.rootFs.FilePutContents(fmt.Sprintf("/Library/LaunchDaemons/%s.plist", label), cont, false, false); err != nil {
		return err
	}
	lines := []string{"systemsetup -setcomputersleep Never",
		"systemsetup -setdisplaysleep Never",
		"systemsetup -setharddisksleep Never",
		"systemsetup -settimezone Asia/Shanghai",
		"networksetup -detectnewhardware",
		// # "diskutil disableJournal /",
	}
	m.addScripts(lines)
	m.addScripts([]string{fmt.Sprintf("echo > %s", spath)})
	cont = strings.Join(m.scripts, "\n") + "\n"
	return m.rootFs.FilePutContents(spath, cont, false, false)
}
