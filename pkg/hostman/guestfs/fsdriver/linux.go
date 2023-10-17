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
	"debug/elf"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"gopkg.in/yaml.v2"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/util/coreosutils"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/fstabutils"
	"yunion.io/x/onecloud/pkg/util/netutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/seclib2"
	"yunion.io/x/onecloud/pkg/util/sysutils"
)

const (
	ROOT_USER             = "root"
	YUNIONROOT_USER       = "cloudroot"
	TELEGRAF_BINARY_PATH  = "/opt/yunion/bin/telegraf"
	SUPERVISE_BINARY_PATH = "/opt/yunion/bin/supervise"
)

var (
	NetDevPrefix   = "eth"
	NetDevPrefixEN = "en"
)

func GetNetDevPrefix(nics []*types.SServerNic) string {
	if NicsHasDifferentDriver(nics) {
		return NetDevPrefixEN
	} else {
		return NetDevPrefix
	}
}

func NicsHasDifferentDriver(nics []*types.SServerNic) bool {
	m := make(map[string]int)
	for i := 0; i < len(nics); i++ {
		if _, ok := m[nics[i].Driver]; !ok {
			m[nics[i].Driver] = 1
		}
	}
	if len(m) > 1 {
		return true
	}
	return false
}

type sLinuxRootFs struct {
	*sGuestRootFsDriver
}

func newLinuxRootFs(part IDiskPartition) *sLinuxRootFs {
	return &sLinuxRootFs{
		sGuestRootFsDriver: newGuestRootFsDriver(part),
	}
}

func (l *sLinuxRootFs) RootSignatures() []string {
	return []string{"/bin", "/etc", "/boot", "/lib", "/usr"}
}

func getHostname(hostname, domain string) string {
	if len(domain) > 0 {
		return fmt.Sprintf("%s.%s", hostname, domain)
	} else {
		return hostname
	}
}

func (l *sLinuxRootFs) DeployQgaBlackList(rootFs IDiskPartition) error {
	var modeRwxOwner = syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IXUSR
	var qgaConfDir = "/etc/sysconfig"
	var etcSysconfigQemuga = path.Join(qgaConfDir, "qemu-ga")

	if err := rootFs.Mkdir(qgaConfDir, modeRwxOwner, false); err != nil {
		return errors.Wrap(err, "mkdir qga conf dir")
	}
	blackListContent := `# This is a systemd environment file, not a shell script.
# It provides settings for \"/lib/systemd/system/qemu-guest-agent.service\".

# Comma-separated blacklist of RPCs to disable, or empty list to enable all.
#
# You can get the list of RPC commands using \"qemu-ga --blacklist='?'\".
# There should be no spaces between commas and commands in the blacklist.
# BLACKLIST_RPC=guest-file-open,guest-file-close,guest-file-read,guest-file-write,guest-file-seek,guest-file-flush,guest-exec,guest-exec-status

# Fsfreeze hook script specification.
#
# FSFREEZE_HOOK_PATHNAME=/dev/null           : disables the feature.
#
# FSFREEZE_HOOK_PATHNAME=/path/to/executable : enables the feature with the
# specified binary or shell script.
#
# FSFREEZE_HOOK_PATHNAME=                    : enables the feature with the
# default value (invoke \"qemu-ga --help\" to interrogate).
FSFREEZE_HOOK_PATHNAME=/etc/qemu-ga/fsfreeze-hook"
`

	if err := rootFs.FilePutContents(etcSysconfigQemuga, blackListContent, false, false); err != nil {
		return errors.Wrap(err, "etcSysconfigQemuga error")
	}
	return nil
}

func (l *sLinuxRootFs) DeployHosts(rootFs IDiskPartition, hostname, domain string, ips []string) error {
	var etcHosts = "/etc/hosts"
	var oldHostFile string
	if rootFs.Exists(etcHosts, false) {
		oldhf, err := rootFs.FileGetContents(etcHosts, false)
		if err != nil {
			return err
		}
		oldHostFile = string(oldhf)
	}
	hf := make(fileutils2.HostsFile, 0)
	hf.Parse(oldHostFile)
	hf.Add("127.0.0.1", "localhost")
	for _, ip := range ips {
		hf.Add(ip, getHostname(hostname, domain), hostname)
	}
	return rootFs.FilePutContents(etcHosts, hf.String(), false, false)
}

func (l *sLinuxRootFs) GetLoginAccount(rootFs IDiskPartition, sUser string, defaultRootUser bool, windowsDefaultAdminUser bool) (string, error) {
	if len(sUser) > 0 {
		if _, err := rootFs.CheckOrAddUser(sUser, "", false); err != nil && !strings.Contains(err.Error(), "already exists") {
			return "", fmt.Errorf("UserAdd %s: %v", sUser, err)
		}
		if err := l.EnableUserSudo(rootFs, sUser); err != nil {
			return "", fmt.Errorf("EnableUserSudo: %s", err)
		}
		return sUser, nil
	}
	var selUsr string
	if defaultRootUser && rootFs.Exists("/root", false) && l.GetIRootFsDriver().AllowAdminLogin() {
		selUsr = ROOT_USER
	} else {
		usrs := rootFs.ListDir("/home", false)
		for _, usr := range usrs {
			if usr == YUNIONROOT_USER {
				continue
			}
			if len(selUsr) == 0 || len(selUsr) > len(usr) {
				selUsr = usr
			}
		}
		if len(selUsr) == 0 && rootFs.Exists("/root", false) {
			selUsr = ROOT_USER
		}
	}
	return selUsr, nil
}

func (l *sLinuxRootFs) ChangeUserPasswd(rootFs IDiskPartition, account, gid, publicKey, password string) (string, error) {
	var secret string
	var err error
	err = rootFs.Passwd(account, password, false)
	if err == nil {
		if len(publicKey) > 0 {
			secret, err = seclib2.EncryptBase64(publicKey, password)
		} else {
			secret, err = utils.EncryptAESBase64(gid, password)
		}
		if err != nil {
			return "", errors.Wrap(err, "Encryption")
		}
		// put /.autorelabel if selinux enabled
		err = rootFs.FilePutContents("/.autorelabel", "", false, false)
		if err != nil {
			return "", errors.Wrap(err, "fail to put .autorelabel")
		}
	} else {
		return "", fmt.Errorf("ChangeUserPasswd error: %v", err)
	}
	return secret, err
}

func (l *sLinuxRootFs) DeployPublicKey(rootFs IDiskPartition, selUsr string, pubkeys *deployapi.SSHKeys) error {
	var usrDir string
	if selUsr == "root" {
		usrDir = "/root"
	} else {
		usrDir = path.Join("/home", selUsr)
	}
	return DeployAuthorizedKeys(rootFs, usrDir, pubkeys, false)
}

func (d *SCoreOsRootFs) DeployQgaBlackList(rootFs IDiskPartition) error {
	var modeRwxOwner = syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IXUSR
	var qgaConfDir = "/etc/sysconfig"
	var etcSysconfigQemuga = path.Join(qgaConfDir, "qemu-ga")

	if err := rootFs.Mkdir(qgaConfDir, modeRwxOwner, false); err != nil {
		return errors.Wrap(err, "mkdir qga conf dir")
	}
	blackListContent := `# This is a systemd environment file, not a shell script.
# It provides settings for \"/lib/systemd/system/qemu-guest-agent.service\".

# Comma-separated blacklist of RPCs to disable, or empty list to enable all.
#
# You can get the list of RPC commands using \"qemu-ga --blacklist='?'\".
# There should be no spaces between commas and commands in the blacklist.
# BLACKLIST_RPC=guest-file-open,guest-file-close,guest-file-read,guest-file-write,guest-file-seek,guest-file-flush,guest-exec,guest-exec-status

# Fsfreeze hook script specification.
#
# FSFREEZE_HOOK_PATHNAME=/dev/null           : disables the feature.
#
# FSFREEZE_HOOK_PATHNAME=/path/to/executable : enables the feature with the
# specified binary or shell script.
#
# FSFREEZE_HOOK_PATHNAME=                    : enables the feature with the
# default value (invoke \"qemu-ga --help\" to interrogate).
FSFREEZE_HOOK_PATHNAME=/etc/qemu-ga/fsfreeze-hook"
`

	if rootFs.Exists(etcSysconfigQemuga, false) {
		if err := rootFs.FilePutContents(etcSysconfigQemuga, blackListContent, false, false); err != nil {
			return errors.Wrap(err, "etcSysconfigQemuga error")
		}
	}
	return nil
}

func (l *sLinuxRootFs) DeployYunionroot(rootFs IDiskPartition, pubkeys *deployapi.SSHKeys, isInit, enableCloudInit bool) error {
	if !consts.AllowVmSELinux() {
		l.DisableSelinux(rootFs)
	}
	if !enableCloudInit && isInit {
		l.DisableCloudinit(rootFs)
	}
	var yunionroot = YUNIONROOT_USER
	var rootdir string // := path.Join(cloudrootDirectory, yunionroot)
	var err error
	if rootdir, err = rootFs.CheckOrAddUser(yunionroot, cloudrootDirectory, true); err != nil {
		return errors.Wrap(err, "unable to CheckOrAddUser")
	}
	log.Infof("DeployYunionroot %s home %s", yunionroot, rootdir)
	err = DeployAuthorizedKeys(rootFs, rootdir, pubkeys, true)
	if err != nil {
		log.Infof("DeployAuthorizedKeys error: %s", err.Error())
		return fmt.Errorf("DeployAuthorizedKeys: %v", err)
	}
	if err := l.EnableUserSudo(rootFs, yunionroot); err != nil {
		return fmt.Errorf("EnableUserSudo: %v", err)
	}
	return nil
}

func (l *sLinuxRootFs) EnableUserSudo(rootFs IDiskPartition, user string) error {
	var sudoDir = "/etc/sudoers.d"
	var content = fmt.Sprintf("%s ALL=(ALL) NOPASSWD:ALL\n", user)
	if rootFs.Exists(sudoDir, false) {
		filepath := path.Join(sudoDir, fmt.Sprintf("90-%s-users", user))
		err := rootFs.FilePutContents(filepath, content, false, false)
		if err != nil {
			return fmt.Errorf("Write contents to %s: %v", filepath, err)
		}
		return rootFs.Chmod(filepath, syscall.S_IRUSR|syscall.S_IRGRP, false)
	}
	return nil
}

func (l *sLinuxRootFs) DisableSelinux(rootFs IDiskPartition) {
	selinuxConfig := "/etc/selinux/config"
	content := `# This file controls the state of SELinux on the system.
# SELINUX= can take one of these three values:
#     enforcing - SELinux security policy is enforced.
#     permissive - SELinux prints warnings instead of enforcing.
#     disabled - No SELinux policy is loaded.
SELINUX=disabled
# SELINUXTYPE= can take one of three two values:
#     targeted - Targeted processes are protected,
#     minimum - Modification of targeted policy. Only selected processes are protected.
#     mls - Multi Level Security protection.
SELINUXTYPE=targeted
`
	if rootFs.Exists(selinuxConfig, false) {
		if err := rootFs.FilePutContents(selinuxConfig, content, false, false); err != nil {
			log.Errorf("DisableSelinux error: %v", err)
		}
	}
}

func (l *sLinuxRootFs) DisableCloudinit(rootFs IDiskPartition) {
	cloudDir := "/etc/cloud"
	cloudDisableFile := "/etc/cloud/cloud-init.disabled"
	if rootFs.Exists(cloudDir, false) {
		if err := rootFs.FilePutContents(cloudDisableFile, "", false, false); err != nil {
			log.Errorf("DisableCloudinit error: %v", err)
		}
	}
}

func (l *sLinuxRootFs) DeployFstabScripts(rootFs IDiskPartition, disks []*deployapi.Disk) error {
	fstabcont, err := rootFs.FileGetContents("/etc/fstab", false)
	if err != nil {
		return err
	}
	var dataDiskIdx = 0
	var rec string
	var modeRwxOwner = syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IXUSR
	var fstab = fstabutils.FSTabFile(string(fstabcont))
	if fstab != nil {
		fstab = fstab.RemoveDevices(len(disks))
	} else {
		_fstab := make(fstabutils.FsTab, 0)
		fstab = &_fstab
	}

	for i := 1; i < len(disks); i++ {
		diskId := disks[i].DiskId
		if len(diskId) == 0 {
			diskId = "None"
		}
		dev := fmt.Sprintf("UUID=%s", diskId)
		if !fstab.IsExists(dev) {
			fs := disks[i].Fs
			if len(fs) > 0 {
				if fs == "swap" {
					rec = fmt.Sprintf("%s none %s sw 0 0", dev, fs)
				} else {
					mtPath := disks[i].Mountpoint
					if len(mtPath) == 0 {
						mtPath = "/data"
						if dataDiskIdx > 0 {
							mtPath += fmt.Sprintf("%d", dataDiskIdx)
						}
						dataDiskIdx += 1
					}
					rec = fmt.Sprintf("%s %s %s defaults 2 2", dev, mtPath, fs)
					if !l.rootFs.Exists(mtPath, false) {
						if err := l.rootFs.Mkdir(mtPath, modeRwxOwner, false); err != nil {
							return err
						}
					}
				}
				fstab.AddFsrec(rec)
			}
		}
	}
	cf := fstab.ToConf()
	return rootFs.FilePutContents("/etc/fstab", cf, false, false)
}

func (l *sLinuxRootFs) DeployNetworkingScripts(rootFs IDiskPartition, nics []*types.SServerNic) error {
	netDevPrefix := GetNetDevPrefix(nics)
	log.Infof("netdev prefix: %s", netDevPrefix)

	udevPath := "/etc/udev/rules.d/"
	if rootFs.Exists(udevPath, false) {
		rules := rootFs.ListDir(udevPath, false)
		for _, rule := range rules {
			if strings.Index(rule, "persistent-net.rules") > 0 {
				rootFs.Remove(path.Join(udevPath, rule), false)
			} else if strings.Index(rule, "persistent-cd.rules") > 0 {
				if err := rootFs.FilePutContents(path.Join(udevPath, rule), "", false, false); err != nil {
					return err
				}
			}
		}
		var nicRules string
		for _, nic := range nics {
			nicRules += `KERNEL=="*", SUBSYSTEM=="net", ACTION=="add", `
			nicRules += `DRIVERS=="?*", `
			mac := nic.Mac
			nicRules += fmt.Sprintf(`ATTR{address}=="%s", ATTR{type}=="1", `, strings.ToLower(mac))
			idx := nic.Index
			nicRules += fmt.Sprintf("NAME=\"%s%d\"\n", netDevPrefix, idx)
		}
		if err := rootFs.FilePutContents(path.Join(udevPath, "70-persistent-net.rules"), nicRules, false, false); err != nil {
			return err
		}

		var usbRules string
		usbRules = `SUBSYSTEM=="usb", ATTRS{idVendor}=="1d6b", ATTRS{idProduct}=="0001", `
		usbRules += "RUN+=" + `"/bin/sh -c \'echo enabled > /sys$env{DEVPATH}/../power/wakeup\'"` + "\n"
		if err := rootFs.FilePutContents(path.Join(udevPath,
			"90-usb-tablet-remote-wakeup.rules"), usbRules, false, false); err != nil {
			return err
		}
	}
	// deploy docker mtu
	{
		minMtu := int16(-1)
		for _, nic := range nics {
			if nic.Mtu > 0 && (minMtu > nic.Mtu || minMtu < 0) {
				minMtu = nic.Mtu
			}
		}
		const dockerDaemonConfPath = "/etc/docker/daemon.json"
		var daemonConfJson jsonutils.JSONObject
		if rootFs.Exists(dockerDaemonConfPath, false) {
			content, _ := rootFs.FileGetContents(dockerDaemonConfPath, false)
			if len(content) > 0 {
				daemonConfJson, _ = jsonutils.Parse(content)
			}
		} else {
			const modeRwxOwner = syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IXUSR
			if err := rootFs.Mkdir("/etc/docker", modeRwxOwner, false); err != nil {
				return errors.Wrap(err, "mkdir /etc/docker fail")
			}
		}
		if daemonConfJson == nil {
			daemonConfJson = jsonutils.NewDict()
		}
		daemonConfJson.(*jsonutils.JSONDict).Set("mtu", jsonutils.NewInt(int64(minMtu)))
		if err := rootFs.FilePutContents(dockerDaemonConfPath, daemonConfJson.PrettyString(), false, false); err != nil {
			return errors.Wrapf(err, "write %s fail", dockerDaemonConfPath)
		}
	}
	// deploy ssh host key
	{
		err := rootFs.GenerateSshHostKeys()
		if err != nil {
			// ignore error
			log.Errorf("rootFs.GenerateSshHostKeys fail %s", err)
		}
	}
	return nil
}

func (l *sLinuxRootFs) DeployStandbyNetworkingScripts(rootFs IDiskPartition, nics, nicsStandby []*types.SServerNic) error {
	var netDevPrefix = GetNetDevPrefix(nicsStandby)
	var udevPath = "/etc/udev/rules.d/"
	var nicRules string
	for _, nic := range nicsStandby {
		if len(nic.NicType) == 0 || nic.NicType != api.NIC_TYPE_IPMI {
			nicRules += `KERNEL=="*", SUBSYSTEM=="net", ACTION=="add", `
			nicRules += `DRIVERS=="?*", `
			mac := nic.Mac
			nicRules += fmt.Sprintf(`ATTR{address}=="%s", ATTR{type}=="1", `, strings.ToLower(mac))
			idx := nic.Index
			nicRules += fmt.Sprintf(`NAME="%s%d"\n`, netDevPrefix, idx)
		}
	}
	if err := rootFs.FilePutContents(path.Join(udevPath, "70-persistent-net.rules"), nicRules, true, false); err != nil {
		return err
	}
	return nil
}

func (l *sLinuxRootFs) GetOs() string {
	return "Linux"
}

func (l *sLinuxRootFs) GetArch(rootFs IDiskPartition) string {
	// search lib64 first
	for _, dir := range []string{"/usr/lib64", "/lib64", "/usr/lib", "/lib"} {
		if !rootFs.Exists(dir, false) {
			continue
		}
		files := rootFs.ListDir(dir, false)
		for i := 0; i < len(files); i++ {
			if strings.HasPrefix(files[i], "ld-") {
				p := rootFs.GetLocalPath(path.Join(dir, files[i]), false)
				fileInfo, err := os.Stat(p)
				if err != nil {
					log.Errorf("stat file %s: %s", p, err)
					continue
				}
				if fileInfo.IsDir() {
					continue
				}
				rp, err := filepath.EvalSymlinks(p)
				if err != nil {
					log.Errorf("readlink of %s: %s", p, err)
					continue
				}
				elfHeader, err := elf.Open(rp)
				if err != nil {
					log.Errorf("failed read file elf %s: %s", rp, err)
					continue
				}
				// https://en.wikipedia.org/wiki/Executable_and_Linkable_Format#File_header
				switch elfHeader.Machine {
				case elf.EM_X86_64:
					return apis.OS_ARCH_X86_64
				case elf.EM_386:
					return apis.OS_ARCH_X86_32
				case elf.EM_AARCH64:
					return apis.OS_ARCH_AARCH64
				case elf.EM_ARM:
					return apis.OS_ARCH_AARCH32
				}
			}
		}
	}
	return apis.OS_ARCH_X86_64
}

func (l *sLinuxRootFs) PrepareFsForTemplate(rootFs IDiskPartition) error {
	// clean /etc/fstab
	if rootFs.Exists("/etc/fstab", false) {
		fstabcont, _ := rootFs.FileGetContents("/etc/fstab", false)
		fstab := fstabutils.FSTabFile(string(fstabcont))
		var cf string
		if fstab != nil {
			fstab = fstab.RemoveDevices(1)
			cf = fstab.ToConf()
		}
		if err := rootFs.FilePutContents("/etc/fstab", cf, false, false); err != nil {
			return err
		}
	}
	// rm /etc/ssh/*_key.*
	if rootFs.Exists("/etc/ssh", false) {
		for _, f := range l.rootFs.ListDir("/etc/ssh", false) {
			if strings.HasSuffix(f, "_key") || strings.HasSuffix(f, "_key.pub") {
				rootFs.Remove("/etc/ssh/"+f, false)
			}
		}
	}
	// clean cloud-init
	if rootFs.Exists("/var/lib/cloud", false) {
		if err := rootFs.Cleandir("/var/lib/cloud", false, false); err != nil {
			return err
		}
	}
	cloudDisableFile := "/etc/cloud/cloud-init.disabled"
	if rootFs.Exists(cloudDisableFile, false) {
		rootFs.Remove(cloudDisableFile, false)
	}
	// clean /tmp /var/log /var/cache /var/spool /var/run
	for _, dir := range []string{"/tmp", "/var/tmp"} {
		if rootFs.Exists(dir, false) {
			if err := rootFs.Cleandir(dir, false, false); err != nil {
				return err
			}
		}
	}
	for _, dir := range []string{
		"/var/log",
		"/var/cache",
		"/usr/local/var/log",
		"/usr/local/var/cache",
	} {
		if rootFs.Exists(dir, false) {
			if err := l.rootFs.Zerofiles(dir, false); err != nil {
				return err
			}
		}
	}
	for _, dir := range []string{
		"/var/spool",
		"/var/run",
		"/run",
		"/usr/local/var/spool",
		"/usr/local/var/run",
		"/etc/openvswitch",
	} {
		if rootFs.Exists(dir, false) {
			if err := rootFs.Cleandir(dir, true, true); err != nil {
				return err
			}
		}
	}
	return nil
}

func (l *sLinuxRootFs) getSerialPorts(rootFs IDiskPartition) []string {
	if !rootFs.SupportSerialPorts() {
		return nil
	}
	// XXX HACK, only sshpart.SSHPartition support this
	var confpath = "/proc/tty/driver/serial"
	content, err := rootFs.FileGetContentsByPath(confpath)
	if err != nil {
		log.Errorf("Get %s error: %v", confpath, err)
		return nil
	}
	ttys := sysutils.GetSerialPorts(strings.Split(string(content), "\n"))
	log.Infof("Get serial ports content:\n%s, find serial ttys: %#v", string(content), ttys)
	return ttys
}

func (l *sLinuxRootFs) enableSerialConsoleInitCentos(rootFs IDiskPartition) error {
	// http://www.jonno.org/drupal/node/10
	var err error
	for _, tty := range l.getSerialPorts(rootFs) {
		content := fmt.Sprintf(
			`stop on runlevel [016]
start on runlevel [345]
instance %s
respawn
pre-start exec /sbin/securetty %s
exec /sbin/agetty /dev/%s 115200 vt100`, tty, tty, tty)
		err = rootFs.FilePutContents(fmt.Sprintf("/etc/init/%s.conf", tty), content, false, false)
	}
	return err
}

func (l *sLinuxRootFs) enableSerialConsoleRootLogin(rootFs IDiskPartition, tty string) error {
	secureTTYFile := "/etc/securetty"
	content, err := rootFs.FileGetContents(secureTTYFile, false)
	if err != nil {
		return errors.Wrapf(err, "get contents of %s", secureTTYFile)
	}
	secureTTYs := sysutils.GetSecureTTYs(strings.Split(string(content), "\n"))
	if utils.IsInStringArray(tty, secureTTYs) {
		return nil
	}
	return rootFs.FilePutContents(secureTTYFile, fmt.Sprintf("\n%s", tty), true, false)
}

func (l *sLinuxRootFs) enableSerialConsoleInit(rootFs IDiskPartition) error {
	// https://help.ubuntu.com/community/SerialConsoleHowto
	var err error
	for _, tty := range l.getSerialPorts(rootFs) {
		if err := l.enableSerialConsoleRootLogin(rootFs, tty); err != nil {
			log.Errorf("Enable %s root login: %v", tty, err)
		}
		content := fmt.Sprintf(
			`start on stopped rc or RUNLEVEL=[12345]
stop on runlevel [!12345]
respawn
exec /sbin/getty -L 115200 %s vt102`, tty)
		err = rootFs.FilePutContents(fmt.Sprintf("/etc/init/%s.conf", tty), content, false, false)
	}
	return err
}

func (l *sLinuxRootFs) disableSerialConsoleInit(rootFs IDiskPartition) {
	for _, tty := range l.getSerialPorts(rootFs) {
		path := fmt.Sprintf("/etc/init/%s.conf", tty)
		if rootFs.Exists(path, false) {
			rootFs.Remove(path, false)
		}
	}
}

func (l *sLinuxRootFs) enableSerialConsoleSystemd(rootFs IDiskPartition) error {
	for _, tty := range l.getSerialPorts(rootFs) {
		if err := l.enableSerialConsoleRootLogin(rootFs, tty); err != nil {
			log.Errorf("Enable %s root login: %v", tty, err)
		}
		sPath := fmt.Sprintf("/etc/systemd/system/getty.target.wants/getty@%s.service", tty)
		if err := rootFs.Symlink("/usr/lib/systemd/system/getty@.service", sPath, false); err != nil {
			return errors.Wrapf(err, "Symbol link tty %s", tty)
		}
	}
	return nil
}

func (l *sLinuxRootFs) disableSerialConsoleSystemd(rootFs IDiskPartition) {
	for _, tty := range l.getSerialPorts(rootFs) {
		sPath := fmt.Sprintf("/etc/systemd/system/getty.target.wants/getty@%s.service", tty)
		if rootFs.Exists(sPath, false) {
			rootFs.Remove(sPath, false)
		}
	}
}

func (l *sLinuxRootFs) dirWalk(part IDiskPartition, sPath string, wF func(path string, isDir bool) bool) bool {
	stat := part.Stat(sPath, false)
	if !stat.IsDir() {
		if wF(sPath, false) {
			return true
		}
		return false
	}
	if wF(sPath, true) {
		return true
	}
	for _, subPath := range part.ListDir(sPath, false) {
		if l.dirWalk(part, path.Join(sPath, subPath), wF) {
			return true
		}
	}
	return false
}

func (l *sLinuxRootFs) DetectIsUEFISupport(part IDiskPartition) bool {
	// ref: https://wiki.archlinux.org/title/EFI_system_partition#Check_for_an_existing_partition
	// To confirm this is the ESP, mount it and check whether it contains a directory named EFI,
	// if it does this is definitely the ESP.
	efiDir := "/EFI"
	exits := part.Exists(efiDir, false)
	if !exits {
		return false
	}

	hasEFIFirmware := false

	l.dirWalk(part, efiDir, func(path string, isDir bool) bool {
		if isDir {
			return false
		}
		// check file is UEFI firmware
		if strings.HasSuffix(path, ".efi") {
			log.Infof("EFI firmware %s found", path)
			hasEFIFirmware = true
			return true
		}
		// continue walk
		return false
	})

	return hasEFIFirmware
}

func (l *sLinuxRootFs) IsCloudinitInstall() bool {
	return l.GetPartition().Exists("/usr/bin/cloud-init", false)
}

func (d *sLinuxRootFs) DeployTelegraf(config string) (bool, error) {
	var (
		part             = d.GetPartition()
		modeRwxOwner     = syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IXUSR
		cloudMonitorPath = "/opt/.cloud-monitor"
		telegrafPath     = path.Join(cloudMonitorPath, "run")
	)

	err := part.Mkdir(cloudMonitorPath, modeRwxOwner, false)
	if err != nil {
		return false, errors.Wrap(err, "mkdir cloud-monitor")
	}
	err = part.Mkdir(telegrafPath, modeRwxOwner, false)
	if err != nil {
		return false, errors.Wrap(err, "mkdir telegraf")
	}
	// telegraf files
	err = part.FilePutContents(path.Join(cloudMonitorPath, "telegraf.conf"), config, false, false)
	if err != nil {
		return false, errors.Wrap(err, "write telegraf config")
	}
	output, err := procutils.NewCommand("cp", "-f", TELEGRAF_BINARY_PATH, path.Join(part.GetMountPath(), cloudMonitorPath)).Output()
	if err != nil {
		return false, errors.Wrapf(err, "cp telegraf failed %s", output)
	}
	// supervise
	output, err = procutils.NewCommand("cp", "-f", SUPERVISE_BINARY_PATH, path.Join(part.GetMountPath(), cloudMonitorPath)).Output()
	if err != nil {
		return false, errors.Wrapf(err, "cp supervise failed %s", output)
	}
	err = part.FilePutContents(
		path.Join(telegrafPath, "run"),
		fmt.Sprintf("#!/bin/sh\n%s/telegraf -config %s/telegraf.conf", cloudMonitorPath, cloudMonitorPath),
		false, false,
	)
	if err != nil {
		return false, errors.Wrap(err, "write supervise run script")
	}
	err = part.Chmod(path.Join(telegrafPath, "run"), 0755, false)
	if err != nil {
		return false, errors.Wrap(err, "chmod supervise run script")
	}
	initCmd := fmt.Sprintf("%s/supervise %s", cloudMonitorPath, telegrafPath)
	err = d.installInitScript("telegraf", initCmd)
	if err != nil {
		return false, errors.Wrap(err, "installInitScript")
	}
	/* // add crontab: start telegraf on guest boot
	cronJob := fmt.Sprintf("@reboot %s/supervise %s", cloudMonitorPath, telegrafPath)
	if procutils.NewCommand("chroot", part.GetMountPath(), "crontab", "-l", "|", "grep", cronJob).Run() == nil {
		// if cronjob exist, return success
		return true, nil
	}
	output, err = procutils.NewCommand("chroot", part.GetMountPath(), "sh", "-c",
		fmt.Sprintf("(crontab -l 2>/dev/null; echo '%s') |crontab -", cronJob),
	).Output()
	if err != nil {
		return false, errors.Wrapf(err, "add crontab %s", output)
	}*/
	return true, nil
}

type sDebianLikeRootFs struct {
	*sLinuxRootFs
}

func newDebianLikeRootFs(part IDiskPartition) *sDebianLikeRootFs {
	return &sDebianLikeRootFs{
		sLinuxRootFs: newLinuxRootFs(part),
	}
}

func (d *sDebianLikeRootFs) PrepareFsForTemplate(rootFs IDiskPartition) error {
	if err := d.sLinuxRootFs.PrepareFsForTemplate(rootFs); err != nil {
		return err
	}
	// clean /etc/network/interface
	if rootFs.Exists("/etc/network/interface", false) {
		fn := "/etc/network/interfaces"
		var cmds strings.Builder
		cmds.WriteString("auto lo\n")
		cmds.WriteString("iface lo inet loopback\n\n")
		if err := rootFs.FilePutContents(fn, cmds.String(), false, false); err != nil {
			return errors.Wrap(err, "file put content /etc/network/interface")
		}
	}

	// clean /etc/netplan/*
	netplanDir := "/etc/netplan/"
	if rootFs.Exists(netplanDir, false) {
		for _, f := range rootFs.ListDir(netplanDir, false) {
			rootFs.Remove(netplanDir+f, false)
		}
	}
	return nil
}

func (d *sDebianLikeRootFs) GetReleaseInfo(rootFs IDiskPartition, driver IDebianRootFsDriver) *deployapi.ReleaseInfo {
	version, err := rootFs.FileGetContents(driver.VersionFilePath(), false)
	if err != nil {
		log.Errorf("Get %s error: %v", driver.VersionFilePath(), err)
		return nil
	}
	versionStr := strings.TrimSpace(string(version))
	return &deployapi.ReleaseInfo{
		Distro:  driver.DistroName(),
		Version: versionStr,
		Arch:    d.GetArch(rootFs),
	}
}

func (d *sDebianLikeRootFs) RootSignatures() []string {
	sig := d.sLinuxRootFs.RootSignatures()
	return append([]string{"/etc/issue"}, sig...)
}

func (d *sDebianLikeRootFs) DeployHostname(rootFs IDiskPartition, hn, domain string) error {
	return rootFs.FilePutContents("/etc/hostname", hn, false, false)
}

func getNicTeamingConfigCmds(slaves []*types.SServerNic) string {
	var cmds strings.Builder
	cmds.WriteString("    bond-mode 4\n")
	cmds.WriteString("    bond-miimon 100\n")
	cmds.WriteString("    bond-lacp-rate 1\n")
	cmds.WriteString("    bond-xmit_hash_policy 1\n")
	cmds.WriteString("    bond-slaves")
	for i := range slaves {
		cmds.WriteString(" ")
		cmds.WriteString(slaves[i].Name)
	}
	cmds.WriteString("\n")
	return cmds.String()
}

func (d *sDebianLikeRootFs) deployNetplanConfigFile(rootFs IDiskPartition, nics []*types.SServerNic) error {
	netplanDir := "/etc/netplan/"
	dirExists := rootFs.Exists(netplanDir, false)
	if !dirExists {
		return nil
	}

	return nil
}

func (d *sDebianLikeRootFs) DeployNetworkingScripts(rootFs IDiskPartition, nics []*types.SServerNic) error {
	if err := d.sLinuxRootFs.DeployNetworkingScripts(rootFs, nics); err != nil {
		return err
	}
	if !rootFs.Exists("/etc/network", false) {
		if err := rootFs.Mkdir("/etc/network",
			syscall.S_IRUSR|syscall.S_IWUSR|syscall.S_IXUSR, false); err != nil {
			return errors.Wrap(err, "mkdir /etc/network")
		}
	}

	fn := "/etc/network/interfaces"
	var cmds strings.Builder
	cmds.WriteString("auto lo\n")
	cmds.WriteString("iface lo inet loopback\n\n")

	// ToServerNics(nics)
	allNics, bondNics := convertNicConfigs(nics)

	netplanDir := "/etc/netplan"
	if rootFs.Exists(netplanDir, false) {
		for _, f := range rootFs.ListDir(netplanDir, false) {
			rootFs.Remove(netplanDir+f, false)
		}
		netplanConfig := NewNetplanConfig(allNics, bondNics)
		if err := rootFs.FilePutContents(path.Join(netplanDir, "config.yaml"), netplanConfig.YAMLString(), false, false); err != nil {
			return errors.Wrap(err, "Put netplan config")
		}
	}

	mainNic, err := getMainNic(allNics)
	if err != nil {
		return err
	}
	var mainIp string
	if mainNic != nil {
		mainIp = mainNic.Ip
	}

	var systemdResolveConfig strings.Builder
	dnss := []string{}
	domains := []string{}
	for i := range allNics {
		nicDesc := allNics[i]
		cmds.WriteString(fmt.Sprintf("auto %s\n", nicDesc.Name))
		if nicDesc.TeamingMaster != nil {
			cmds.WriteString(fmt.Sprintf("iface %s inet manual\n", nicDesc.Name))
			cmds.WriteString(fmt.Sprintf("    bond-master %s\n", nicDesc.TeamingMaster.Name))
			cmds.WriteString("\n")
		} else if nicDesc.Virtual {
			cmds.WriteString(fmt.Sprintf("iface %s inet static\n", nicDesc.Name))
			cmds.WriteString(fmt.Sprintf("    address %s\n", netutils2.PSEUDO_VIP))
			cmds.WriteString("    netmask 255.255.255.255\n")
			cmds.WriteString("\n")
		} else if nicDesc.Manual {
			netmask := netutils2.Netlen2Mask(int(nicDesc.Masklen))
			cmds.WriteString(fmt.Sprintf("iface %s inet static\n", nicDesc.Name))
			cmds.WriteString(fmt.Sprintf("    address %s\n", nicDesc.Ip))
			cmds.WriteString(fmt.Sprintf("    netmask %s\n", netmask))
			if len(nicDesc.Gateway) > 0 && nicDesc.Ip == mainIp {
				cmds.WriteString(fmt.Sprintf("    gateway %s\n", nicDesc.Gateway))
			}
			if nicDesc.Mtu > 0 {
				cmds.WriteString(fmt.Sprintf("    mtu %d\n", nicDesc.Mtu))
			}
			var routes = make([][]string, 0)
			netutils2.AddNicRoutes(&routes, nicDesc, mainIp, len(nics), privatePrefixes)
			for _, r := range routes {
				cmds.WriteString(fmt.Sprintf("    up route add -net %s gw %s || true\n", r[0], r[1]))
				cmds.WriteString(fmt.Sprintf("    down route del -net %s gw %s || true\n", r[0], r[1]))
			}
			dnslist := netutils2.GetNicDns(nicDesc)
			if len(dnslist) > 0 {
				cmds.WriteString(fmt.Sprintf("    dns-nameservers %s\n", strings.Join(dnslist, " ")))
				dnss = append(dnss, dnslist...)
				if len(nicDesc.Domain) > 0 {
					cmds.WriteString(fmt.Sprintf("    dns-search %s\n", nicDesc.Domain))
					domains = append(domains, nicDesc.Domain)
				}
			}
			if len(nicDesc.TeamingSlaves) > 0 {
				cmds.WriteString(getNicTeamingConfigCmds(nicDesc.TeamingSlaves))
			}
			cmds.WriteString("\n")
		} else {
			cmds.WriteString(fmt.Sprintf("iface %s inet dhcp\n", nicDesc.Name))
			if len(nicDesc.TeamingSlaves) > 0 {
				cmds.WriteString(getNicTeamingConfigCmds(nicDesc.TeamingSlaves))
			}
			cmds.WriteString("\n")
		}
	}

	if len(dnss) != 0 {
		systemdResolveConfig.WriteString("[Resolve]\n")
		systemdResolveConfig.WriteString(fmt.Sprintf("DNS=%s\n", strings.Join(dnss, " ")))
		if len(domains) != 0 {
			systemdResolveConfig.WriteString(fmt.Sprintf("Domains=%s\n", strings.Join(domains, " ")))
		}
		systemdResolveFn := "/etc/systemd/resolved.conf"
		content := systemdResolveConfig.String()
		if err := rootFs.FilePutContents(systemdResolveFn, content, false, false); err != nil {
			log.Warningf("Put %s to %s error: %v", content, systemdResolveFn, err)
		}
	}
	log.Debugf("%s", cmds.String())
	return rootFs.FilePutContents(fn, cmds.String(), false, false)
}

type SDebianRootFs struct {
	*sDebianLikeRootFs
}

func NewDebianRootFs(part IDiskPartition) IRootFsDriver {
	driver := new(SDebianRootFs)
	driver.sDebianLikeRootFs = newDebianLikeRootFs(part)
	return driver
}

func (d *SDebianRootFs) String() string {
	return "DebianRootFs"
}

func (d *SDebianRootFs) GetName() string {
	return "Debian"
}

func (d *SDebianRootFs) DistroName() string {
	return d.GetName()
}

func (d *SDebianRootFs) VersionFilePath() string {
	return "/etc/debian_version"
}

func (d *SDebianRootFs) rootSignatures(driver IDebianRootFsDriver) []string {
	sig := d.sDebianLikeRootFs.RootSignatures()
	return append([]string{driver.VersionFilePath()}, sig...)
}

func (d *SDebianRootFs) RootSignatures() []string {
	return d.rootSignatures(d)
}

func (d *SDebianRootFs) RootExcludeSignatures() []string {
	return []string{"/etc/lsb-release"}
}

func (d *SDebianRootFs) GetReleaseInfo(rootFs IDiskPartition) *deployapi.ReleaseInfo {
	return d.sDebianLikeRootFs.GetReleaseInfo(rootFs, d)
}

type SCirrosRootFs struct {
	*SDebianRootFs
}

func NewCirrosRootFs(part IDiskPartition) IRootFsDriver {
	driver := new(SCirrosRootFs)
	driver.SDebianRootFs = NewDebianRootFs(part).(*SDebianRootFs)
	return driver
}

func (d *SCirrosRootFs) GetName() string {
	return "Cirros"
}

func (d *SCirrosRootFs) String() string {
	return "CirrosRootFs"
}

func (d *SCirrosRootFs) DistroName() string {
	return d.GetName()
}

func (d *SCirrosRootFs) VersionFilePath() string {
	return "/etc/br-version"
}

func (d *SCirrosRootFs) GetReleaseInfo(rootFs IDiskPartition) *deployapi.ReleaseInfo {
	return d.SDebianRootFs.sDebianLikeRootFs.GetReleaseInfo(rootFs, d)
}

func (d *SCirrosRootFs) RootSignatures() []string {
	return d.rootSignatures(d)
}

type SCirrosNewRootFs struct {
	*SDebianRootFs
}

func NewCirrosNewRootFs(part IDiskPartition) IRootFsDriver {
	driver := new(SCirrosNewRootFs)
	driver.SDebianRootFs = NewDebianRootFs(part).(*SDebianRootFs)
	return driver
}

func (d *SCirrosNewRootFs) GetName() string {
	return "Cirros"
}

func (d *SCirrosNewRootFs) String() string {
	return "CirrosNewRootFs"
}

func (d *SCirrosNewRootFs) DistroName() string {
	return d.GetName()
}

func (d *SCirrosNewRootFs) VersionFilePath() string {
	return "/etc/cirros/version"
}

func (d *SCirrosNewRootFs) RootSignatures() []string {
	return d.rootSignatures(d)
}

func (d *SCirrosNewRootFs) GetReleaseInfo(rootFs IDiskPartition) *deployapi.ReleaseInfo {
	return d.SDebianRootFs.sDebianLikeRootFs.GetReleaseInfo(rootFs, d)
}

type SUbuntuRootFs struct {
	*sDebianLikeRootFs
}

func NewUbuntuRootFs(part IDiskPartition) IRootFsDriver {
	driver := new(SUbuntuRootFs)
	driver.sDebianLikeRootFs = newDebianLikeRootFs(part)
	return driver
}

func (d *SUbuntuRootFs) RootSignatures() []string {
	sig := d.sDebianLikeRootFs.RootSignatures()
	return append([]string{"/etc/lsb-release"}, sig...)
}

func (d *SUbuntuRootFs) GetName() string {
	return "Ubuntu"
}

func (d *SUbuntuRootFs) String() string {
	return "UbuntuRootFs"
}

func (d *SUbuntuRootFs) GetReleaseInfo(rootFs IDiskPartition) *deployapi.ReleaseInfo {
	distroKey := "DISTRIB_RELEASE="
	distroId := "DISTRIB_ID="
	rel, err := rootFs.FileGetContents("/etc/lsb-release", false)
	if err != nil {
		log.Errorf("Get ubuntu release info error: %v", err)
		return nil
	}
	var version, distro string
	lines := strings.Split(string(rel), "\n")
	for _, l := range lines {
		if strings.HasPrefix(l, distroKey) {
			version = strings.TrimSpace(l[len(distroKey):])
		} else if strings.HasPrefix(l, distroId) {
			distro = strings.TrimSpace(l[len(distroId):])
		}
	}
	if distro == "" {
		distro = d.GetName()
	}
	return deployapi.NewReleaseInfo(distro, version, d.GetArch(rootFs))
}

func (d *SUbuntuRootFs) EnableSerialConsole(rootFs IDiskPartition, sysInfo *jsonutils.JSONDict) error {
	relInfo := d.GetReleaseInfo(rootFs)
	ver := strings.Split(relInfo.Version, ".")
	verInt, _ := strconv.Atoi(ver[0])
	if verInt < 16 {
		return d.enableSerialConsoleInit(rootFs)
	}
	return d.enableSerialConsoleSystemd(rootFs)
}

func (d *SUbuntuRootFs) DisableSerialConcole(rootFs IDiskPartition) error {
	relInfo := d.GetReleaseInfo(rootFs)
	ver := strings.Split(relInfo.Version, ".")
	verInt, _ := strconv.Atoi(ver[0])
	if verInt < 16 {
		d.disableSerialConsoleInit(rootFs)
		return nil
	}
	d.disableSerialConsoleSystemd(rootFs)
	return nil
}

type SUKylinRootfs struct {
	*SUbuntuRootFs
}

func NewUKylinRootfs(part IDiskPartition) IRootFsDriver {
	return &SUKylinRootfs{SUbuntuRootFs: NewUbuntuRootFs(part).(*SUbuntuRootFs)}
}

func (d *SUKylinRootfs) GetName() string {
	return "UbuntuKylin"
}

func (d *SUKylinRootfs) String() string {
	return "UKylinuRootFs"
}

func (d *SUKylinRootfs) RootSignatures() []string {
	sig := d.sDebianLikeRootFs.RootSignatures()
	return append([]string{"/etc/lsb-release", "/etc/kylin-build"}, sig...)
}

func (d *SUKylinRootfs) GetReleaseInfo(rootFs IDiskPartition) *deployapi.ReleaseInfo {
	info := d.SUbuntuRootFs.GetReleaseInfo(rootFs)
	info.Distro = d.GetName()
	return info
}

func (d *SUKylinRootfs) AllowAdminLogin() bool {
	return false
}

type sRedhatLikeRootFs struct {
	*sLinuxRootFs
}

func newRedhatLikeRootFs(part IDiskPartition) *sRedhatLikeRootFs {
	return &sRedhatLikeRootFs{
		sLinuxRootFs: newLinuxRootFs(part),
	}
}

func (r *sRedhatLikeRootFs) PrepareFsForTemplate(rootFs IDiskPartition) error {
	if err := r.sLinuxRootFs.PrepareFsForTemplate(rootFs); err != nil {
		return err
	}
	return r.CleanNetworkScripts(rootFs)
}

func (r *sRedhatLikeRootFs) CleanNetworkScripts(rootFs IDiskPartition) error {
	networkPath := "/etc/sysconfig/network-scripts"
	files := rootFs.ListDir(networkPath, false)
	for i := 0; i < len(files); i++ {
		if strings.HasPrefix(files[i], "ifcfg-") && files[i] != "ifcfg-lo" {
			rootFs.Remove(path.Join(networkPath, files[i]), false)
			continue
		}
		if strings.HasPrefix(files[i], "route-") {
			rootFs.Remove(path.Join(networkPath, files[i]), false)
		}
	}
	return nil
}

func (r *sRedhatLikeRootFs) RootSignatures() []string {
	sig := r.sLinuxRootFs.RootSignatures()
	return append([]string{"/etc/sysconfig/network", "/etc/redhat-release"}, sig...)
}

func (r *sRedhatLikeRootFs) DeployHostname(rootFs IDiskPartition, hn, domain string) error {
	var sPath = "/etc/sysconfig/network"
	centosHn := ""
	centosHn += "NETWORKING=yes\n"
	centosHn += fmt.Sprintf("HOSTNAME=%s\n", getHostname(hn, domain))
	if err := rootFs.FilePutContents(sPath, centosHn, false, false); err != nil {
		return err
	}
	if rootFs.Exists("/etc/hostname", false) {
		return rootFs.FilePutContents("/etc/hostname", hn, false, false)
	}
	return nil
}

func (r *sRedhatLikeRootFs) Centos5DeployNetworkingScripts(rootFs IDiskPartition, nics []*types.SServerNic) error {
	var netDevPrefix = GetNetDevPrefix(nics)
	var udevPath = "/etc/udev/rules.d/"
	if rootFs.Exists(udevPath, false) {
		var nicRules = ""
		for _, nic := range nics {
			nicRules += `KERNEL=="*", `
			nicRules += fmt.Sprintf(`SYSFS{address}=="%s", `, strings.ToLower(nic.Mac))
			nicRules += fmt.Sprintf("NAME=\"%s%d\"\n", netDevPrefix, nic.Index)
		}
		return rootFs.FilePutContents(path.Join(udevPath, "60-net.rules"),
			nicRules, false, false)
	}
	return nil
}

func getMainNic(nics []*types.SServerNic) (*types.SServerNic, error) {
	var mainIp netutils.IPV4Addr
	var mainNic *types.SServerNic
	for i := range nics {
		if len(nics[i].Gateway) > 0 {
			ipInt, err := netutils.NewIPV4Addr(nics[i].Ip)
			if err != nil {
				return nil, err
			}
			if mainIp == 0 {
				mainIp = ipInt
				mainNic = nics[i]
			} else if !netutils.IsPrivate(ipInt) && netutils.IsPrivate(mainIp) {
				mainIp = ipInt
				mainNic = nics[i]
			}
		}
	}
	return mainNic, nil
}

func (r *sRedhatLikeRootFs) enableBondingModule(rootFs IDiskPartition, bondNics []*types.SServerNic) error {
	var content strings.Builder
	for i := range bondNics {
		content.WriteString("alias ")
		content.WriteString(bondNics[i].Name)
		content.WriteString(" bonding\n options ")
		content.WriteString(bondNics[i].Name)
		content.WriteString(" miimon=100 mode=4 lacp_rate=1 xmit_hash_policy=1\n")
	}
	return rootFs.FilePutContents("/etc/modprobe.d/bonding.conf", content.String(), false, false)
}

// check if NetworkManager is enabled
func (r *sRedhatLikeRootFs) isNetworkManagerEnabled(rootFs IDiskPartition) bool {
	return rootFs.Exists("/etc/systemd/system/multi-user.target.wants/NetworkManager.service", false)
}

func (r *sRedhatLikeRootFs) deployNetworkingScripts(rootFs IDiskPartition, nics []*types.SServerNic, relInfo *deployapi.ReleaseInfo) error {
	if err := r.sLinuxRootFs.DeployNetworkingScripts(rootFs, nics); err != nil {
		return err
	}

	ver := strings.Split(relInfo.Version, ".")
	iv, err := strconv.ParseInt(ver[0], 10, 0)
	if err == nil && iv < 6 {
		err = r.Centos5DeployNetworkingScripts(rootFs, nics)
	} else {
		err = r.sLinuxRootFs.DeployNetworkingScripts(rootFs, nics)
	}
	if err != nil {
		return err
	}
	// ToServerNics(nics)
	allNics, bondNics := convertNicConfigs(nics)
	if len(bondNics) > 0 {
		err = r.enableBondingModule(rootFs, bondNics)
		if err != nil {
			return err
		}
	}
	mainNic, err := getMainNic(allNics)
	if err != nil {
		return err
	}
	var mainIp string
	if mainNic != nil {
		mainIp = mainNic.Ip
	}
	for i := range allNics {
		nicDesc := allNics[i]
		var cmds strings.Builder
		cmds.WriteString("DEVICE=")
		cmds.WriteString(nicDesc.Name)
		cmds.WriteString("\n")
		cmds.WriteString("NAME=")
		cmds.WriteString(nicDesc.Name)
		cmds.WriteString("\n")
		cmds.WriteString("ONBOOT=yes\n")
		if r.isNetworkManagerEnabled(rootFs) {
			cmds.WriteString("NM_CONTROLLED=yes\n")
		} else {
			cmds.WriteString("NM_CONTROLLED=no\n")
		}
		cmds.WriteString("USERCTL=no\n")
		if nicDesc.Mtu > 0 {
			cmds.WriteString(fmt.Sprintf("MTU=%d\n", nicDesc.Mtu))
		}
		if len(nicDesc.Mac) > 0 {
			cmds.WriteString("HWADDR=")
			cmds.WriteString(nicDesc.Mac)
			cmds.WriteString("\n")
			cmds.WriteString("MACADDR=")
			cmds.WriteString(nicDesc.Mac)
			cmds.WriteString("\n")
		}
		if len(nicDesc.TeamingSlaves) != 0 {
			cmds.WriteString(`BONDING_OPTS="mode=4 miimon=100"\n`)
		}
		if nicDesc.TeamingMaster != nil {
			cmds.WriteString("BOOTPROTO=none\n")
			cmds.WriteString("MASTER=")
			cmds.WriteString(nicDesc.TeamingMaster.Name)
			cmds.WriteString("\n")
			cmds.WriteString("SLAVE=yes\n")
		} else if nicDesc.Virtual {
			cmds.WriteString("BOOTPROTO=none\n")
			cmds.WriteString("NETMASK=255.255.255.255\n")
			cmds.WriteString("IPADDR=")
			cmds.WriteString(netutils2.PSEUDO_VIP)
			cmds.WriteString("\n")
		} else if nicDesc.Manual {
			netmask := netutils2.Netlen2Mask(int(nicDesc.Masklen))
			cmds.WriteString("BOOTPROTO=none\n")
			cmds.WriteString("NETMASK=")
			cmds.WriteString(netmask)
			cmds.WriteString("\n")
			cmds.WriteString("IPADDR=")
			cmds.WriteString(nicDesc.Ip)
			cmds.WriteString("\n")
			if len(nicDesc.Gateway) > 0 && nicDesc.Ip == mainIp {
				cmds.WriteString("GATEWAY=")
				cmds.WriteString(nicDesc.Gateway)
				cmds.WriteString("\n")
			}
			var routes = make([][]string, 0)
			netutils2.AddNicRoutes(&routes, nicDesc, mainIp, len(nics), privatePrefixes)
			var rtbl strings.Builder
			for _, r := range routes {
				rtbl.WriteString(r[0])
				rtbl.WriteString(" via ")
				rtbl.WriteString(r[1])
				rtbl.WriteString(" dev ")
				rtbl.WriteString(nicDesc.Name)
				rtbl.WriteString("\n")
			}
			rtblStr := rtbl.String()
			if len(rtblStr) > 0 {
				var fn = fmt.Sprintf("/etc/sysconfig/network-scripts/route-%s", nicDesc.Name)
				if err := rootFs.FilePutContents(fn, rtblStr, false, false); err != nil {
					return err
				}
			}
			dnslist := netutils2.GetNicDns(nicDesc)
			if len(dnslist) > 0 {
				cmds.WriteString("PEERDNS=yes\n")
				for i := 0; i < len(dnslist); i++ {
					cmds.WriteString(fmt.Sprintf("DNS%d=%s\n", i+1, dnslist[i]))
				}
				if len(nicDesc.Domain) > 0 {
					cmds.WriteString(fmt.Sprintf("DOMAIN=%s\n", nicDesc.Domain))
				}
			}
		} else {
			cmds.WriteString("BOOTPROTO=dhcp\n")
		}
		var fn = fmt.Sprintf("/etc/sysconfig/network-scripts/ifcfg-%s", nicDesc.Name)
		log.Debugf("%s: %s", fn, cmds.String())
		if err := rootFs.FilePutContents(fn, cmds.String(), false, false); err != nil {
			return err
		}
	}
	return nil
}

func (r *sRedhatLikeRootFs) DeployStandbyNetworkingScripts(rootFs IDiskPartition, nics, nicsStandby []*types.SServerNic) error {
	if err := r.sLinuxRootFs.DeployStandbyNetworkingScripts(rootFs, nics, nicsStandby); err != nil {
		return err
	}
	var netDevPrefix = GetNetDevPrefix(nics)
	for _, nic := range nicsStandby {
		var cmds string
		if len(nic.NicType) == 0 || nic.NicType != "ipmi" {
			cmds += fmt.Sprintf("DEVICE=%s%d\n", netDevPrefix, nic.Index)
			cmds += fmt.Sprintf("NAME=%s%d\n", netDevPrefix, nic.Index)
			cmds += fmt.Sprintf("HWADDR=%s\n", nic.Mac)
			cmds += fmt.Sprintf("MACADDR=%s\n", nic.Mac)
			cmds += "ONBOOT=no\n"
			var fn = fmt.Sprintf("/etc/sysconfig/network-scripts/ifcfg-%s%d", netDevPrefix, nic.Index)
			if err := rootFs.FilePutContents(fn, cmds, false, false); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *sRedhatLikeRootFs) getReleaseMajorVersion(drv IRootFsDriver, rootFs IDiskPartition) (int, error) {
	relInfo := drv.GetReleaseInfo(rootFs)
	if len(relInfo.Version) == 0 {
		return 0, fmt.Errorf("release info version is empty")
	}
	log.Infof("Get release info: %#v", relInfo)
	ver, err := strconv.Atoi(string(relInfo.Version[0]))
	if err != nil {
		return 0, fmt.Errorf("Release version %s not start with digit", relInfo.Version)
	}
	return ver, nil
}

func (r *sRedhatLikeRootFs) enableSerialConsole(drv IRootFsDriver, rootFs IDiskPartition, sysInfo *jsonutils.JSONDict) error {
	ver, err := r.getReleaseMajorVersion(drv, rootFs)
	if err != nil {
		return errors.Wrap(err, "Get release major version")
	}
	if ver <= 6 {
		return r.enableSerialConsoleInitCentos(rootFs)
	}
	return r.enableSerialConsoleSystemd(rootFs)
}

func (r *sRedhatLikeRootFs) disableSerialConcole(drv IRootFsDriver, rootFs IDiskPartition) error {
	ver, err := r.getReleaseMajorVersion(drv, rootFs)
	if err != nil {
		return errors.Wrap(err, "Get release major version")
	}
	if ver <= 6 {
		r.disableSerialConsoleInit(rootFs)
		return nil
	}
	r.disableSerialConsoleSystemd(rootFs)
	return nil
}

type SCentosRootFs struct {
	*sRedhatLikeRootFs
}

func NewCentosRootFs(part IDiskPartition) IRootFsDriver {
	return &SCentosRootFs{sRedhatLikeRootFs: newRedhatLikeRootFs(part)}
}

func (c *SCentosRootFs) String() string {
	return "CentosRootFs"
}

func (c *SCentosRootFs) GetName() string {
	return "CentOS"
}

func (c *SCentosRootFs) RootSignatures() []string {
	sig := c.sRedhatLikeRootFs.RootSignatures()
	return append([]string{"/etc/centos-release"}, sig...)
}

func (c *SCentosRootFs) GetReleaseInfo(rootFs IDiskPartition) *deployapi.ReleaseInfo {
	rel, _ := rootFs.FileGetContents("/etc/centos-release", false)
	var version string
	if len(rel) > 0 {
		re := regexp.MustCompile(`^\d+\.\d+`)
		dat := strings.Split(string(rel), " ")
		for _, v := range dat {
			if re.Match([]byte(v)) {
				version = v
				break
			}
		}
	}
	return deployapi.NewReleaseInfo(c.GetName(), version, c.GetArch(rootFs))
}

func (c *SCentosRootFs) DeployNetworkingScripts(rootFs IDiskPartition, nics []*types.SServerNic) error {
	relInfo := c.GetReleaseInfo(rootFs)
	if err := c.sRedhatLikeRootFs.deployNetworkingScripts(rootFs, nics, relInfo); err != nil {
		return err
	}
	var udevPath = "/etc/udev/rules.d/"
	var files = []string{"60-net.rules", "75-persistent-net-generator.rules"}
	for _, f := range files {
		sPath := path.Join(udevPath, f)
		if !rootFs.Exists(sPath, false) {
			if err := rootFs.FilePutContents(sPath, "", false, false); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *SCentosRootFs) EnableSerialConsole(rootFs IDiskPartition, sysInfo *jsonutils.JSONDict) error {
	return c.enableSerialConsole(c, rootFs, sysInfo)
}

func (c *SCentosRootFs) DisableSerialConsole(rootFs IDiskPartition) error {
	return c.disableSerialConcole(c, rootFs)
}

type SFedoraRootFs struct {
	*sRedhatLikeRootFs
}

func NewFedoraRootFs(part IDiskPartition) IRootFsDriver {
	return &SFedoraRootFs{sRedhatLikeRootFs: newRedhatLikeRootFs(part)}
}

func (c *SFedoraRootFs) String() string {
	return "FedoraRootFs"
}

func (c *SFedoraRootFs) GetName() string {
	return "Fedora"
}

func (c *SFedoraRootFs) RootSignatures() []string {
	sig := c.sRedhatLikeRootFs.RootSignatures()
	return append([]string{"/etc/fedora-release"}, sig...)
}

func (c *SFedoraRootFs) GetReleaseInfo(rootFs IDiskPartition) *deployapi.ReleaseInfo {
	rel, _ := rootFs.FileGetContents("/etc/fedora-release", false)
	var version string
	if len(rel) > 0 {
		re := regexp.MustCompile(`^\d+`)
		dat := strings.Split(string(rel), " ")
		for _, v := range dat {
			if re.Match([]byte(v)) {
				version = v
				break
			}
		}
	}
	return deployapi.NewReleaseInfo(c.GetName(), version, c.GetArch(rootFs))
}

func (c *SFedoraRootFs) DeployNetworkingScripts(rootFs IDiskPartition, nics []*types.SServerNic) error {
	relInfo := c.GetReleaseInfo(rootFs)
	if err := c.sRedhatLikeRootFs.deployNetworkingScripts(rootFs, nics, relInfo); err != nil {
		return err
	}
	return nil
}

func (c *SFedoraRootFs) EnableSerialConsole(rootFs IDiskPartition, sysInfo *jsonutils.JSONDict) error {
	return c.enableSerialConsole(c, rootFs, sysInfo)
}

func (c *SFedoraRootFs) DisableSerialConsole(rootFs IDiskPartition) error {
	return c.disableSerialConcole(c, rootFs)
}

type SRhelRootFs struct {
	*sRedhatLikeRootFs
}

func NewRhelRootFs(part IDiskPartition) IRootFsDriver {
	return &SRhelRootFs{sRedhatLikeRootFs: newRedhatLikeRootFs(part)}
}

func (d *SRhelRootFs) GetName() string {
	return "RHEL"
}

func (d *SRhelRootFs) String() string {
	return "RhelRootFs"
}

func (d *SRhelRootFs) GetReleaseInfo(rootFs IDiskPartition) *deployapi.ReleaseInfo {
	rel, _ := rootFs.FileGetContents("/etc/redhat-release", false)
	var version string
	if len(rel) > 0 {
		dat := strings.Split(string(rel), " ")
		if len(dat) > 6 {
			version = dat[6]
		}
	}
	return deployapi.NewReleaseInfo(d.GetName(), version, d.GetArch(rootFs))
}

func (d *SRhelRootFs) DeployNetworkingScripts(rootFs IDiskPartition, nics []*types.SServerNic) error {
	relInfo := d.GetReleaseInfo(rootFs)
	if err := d.sRedhatLikeRootFs.deployNetworkingScripts(rootFs, nics, relInfo); err != nil {
		return err
	}
	return nil
}

func (c *SRhelRootFs) EnableSerialConsole(rootFs IDiskPartition, sysInfo *jsonutils.JSONDict) error {
	return c.enableSerialConsole(c, rootFs, sysInfo)
}

func (c *SRhelRootFs) DisableSerialConsole(rootFs IDiskPartition) error {
	return c.disableSerialConcole(c, rootFs)
}

type SOpenEulerRootFs struct {
	*SCentosRootFs
}

func NewOpenEulerRootFs(part IDiskPartition) IRootFsDriver {
	return &SOpenEulerRootFs{
		SCentosRootFs: NewCentosRootFs(part).(*SCentosRootFs),
	}
}

func (c *SOpenEulerRootFs) String() string {
	return "OpenEulerRootFs"
}

func (c *SOpenEulerRootFs) GetName() string {
	return "OpenEuler"
}

func (c *SOpenEulerRootFs) RootSignatures() []string {
	sig := c.sLinuxRootFs.RootSignatures()
	return append([]string{"/etc/sysconfig/network", "/etc/openEuler-release"}, sig...)
}

func (c *SOpenEulerRootFs) GetReleaseInfo(rootFs IDiskPartition) *deployapi.ReleaseInfo {
	rel, _ := rootFs.FileGetContents("/etc/openEuler-release", false)
	var version string
	if len(rel) > 0 {
		re := regexp.MustCompile(`^\d+\.\d+`)
		dat := strings.Split(string(rel), " ")
		for _, v := range dat {
			if re.Match([]byte(v)) {
				version = v
				break
			}
		}
	}
	version = strings.TrimSpace(version)
	return deployapi.NewReleaseInfo(c.GetName(), version, c.GetArch(rootFs))
}

func (c *SOpenEulerRootFs) DeployNetworkingScripts(rootFs IDiskPartition, nics []*types.SServerNic) error {
	relInfo := c.GetReleaseInfo(rootFs)
	if err := c.sRedhatLikeRootFs.deployNetworkingScripts(rootFs, nics, relInfo); err != nil {
		return err
	}
	return nil
}

type SGentooRootFs struct {
	*sLinuxRootFs
}

func NewGentooRootFs(part IDiskPartition) IRootFsDriver {
	return &SGentooRootFs{sLinuxRootFs: newLinuxRootFs(part)}
}

func (d *SGentooRootFs) GetName() string {
	return "Gentoo"
}

func (d *SGentooRootFs) String() string {
	return "GentooRootFs"
}

func (d *SGentooRootFs) RootSignatures() []string {
	sig := d.sLinuxRootFs.RootSignatures()
	if sig != nil {
		return append(sig, "/etc/gentoo-release")
	} else {
		return []string{"/etc/gentoo-release"}
	}
}

func (d *SGentooRootFs) DeployHostname(rootFs IDiskPartition, hn, domain string) error {
	spath := "/etc/conf.d/hostname"
	content := fmt.Sprintf(`hostname="%s"\n`, hn)
	return rootFs.FilePutContents(spath, content, false, false)
}

func (l *SGentooRootFs) DeployNetworkingScripts(rootFs IDiskPartition, nics []*types.SServerNic) error {
	if err := l.sLinuxRootFs.DeployNetworkingScripts(rootFs, nics); err != nil {
		return err
	}

	var (
		fn   = "/etc/conf.d/net"
		cmds = ""
	)

	var netDevPrefix = GetNetDevPrefix(nics)
	// Ref https://wiki.gentoo.org/wiki/Netifrc
	for _, nic := range nics {
		nicIndex := nic.Index
		if nic.Virtual {
			cmds += fmt.Sprintf(`config_%s%d="`, netDevPrefix, nicIndex)
			cmds += fmt.Sprintf("%s netmask 255.255.255.255", netutils2.PSEUDO_VIP)
			cmds += `"\n`
		} else {
			cmds += fmt.Sprintf(`config_%s%d="dhcp"\n`, netDevPrefix, nicIndex)
		}
		if nic.Mtu > 0 {
			cmds += fmt.Sprintf(`mtu_%s%d="%d"\n`, netDevPrefix, nicIndex, nic.Mtu)
		}
	}
	if err := rootFs.FilePutContents(fn, cmds, false, false); err != nil {
		return err
	}
	for _, nic := range nics {
		nicIndex := nic.Index
		netname := fmt.Sprintf("net.%s%d", netDevPrefix, nicIndex)
		procutils.NewCommand("ln", "-s", "net.lo",
			fmt.Sprintf("%s/etc/init.d/%s", rootFs.GetMountPath(), netname)).Run()
		procutils.NewCommand("chroot",
			rootFs.GetMountPath(), "rc-update", "add", netname, "default").Run()
	}
	return nil
}

func (d *SGentooRootFs) GetReleaseInfo(rootFs IDiskPartition) *deployapi.ReleaseInfo {
	return &deployapi.ReleaseInfo{
		Distro: "Gentoo",
		Arch:   d.GetArch(rootFs),
	}
}

type SArchLinuxRootFs struct {
	*sLinuxRootFs
}

func NewArchLinuxRootFs(part IDiskPartition) IRootFsDriver {
	return &SArchLinuxRootFs{sLinuxRootFs: newLinuxRootFs(part)}
}

func (d *SArchLinuxRootFs) GetName() string {
	return "ArchLinux"
}

func (d *SArchLinuxRootFs) String() string {
	return "ArchLinuxRootFs"
}

func (d *SArchLinuxRootFs) RootSignatures() []string {
	sig := d.sLinuxRootFs.RootSignatures()
	if sig != nil {
		return append(sig, "/etc/arch-release")
	} else {
		return []string{"/etc/arch-release"}
	}
}

func (d *SArchLinuxRootFs) DeployHostname(rootFs IDiskPartition, hn, domain string) error {
	return rootFs.FilePutContents("/etc/hostname", hn, false, false)
}

func (d *SArchLinuxRootFs) GetReleaseInfo(rootFs IDiskPartition) *deployapi.ReleaseInfo {
	return &deployapi.ReleaseInfo{
		Distro: "ArchLinux",
		Arch:   d.GetArch(rootFs),
	}
}

type SOpenWrtRootFs struct {
	*sLinuxRootFs
}

func NewOpenWrtRootFs(part IDiskPartition) IRootFsDriver {
	return &SOpenWrtRootFs{sLinuxRootFs: newLinuxRootFs(part)}
}

func (d *SOpenWrtRootFs) GetName() string {
	return "OpenWrt"
}

func (d *SOpenWrtRootFs) String() string {
	return "OpenWrtRootFs"
}

func (d *SOpenWrtRootFs) RootSignatures() []string {
	return []string{"/bin", "/etc/", "/lib", "/sbin", "/overlay", "/etc/openwrt_release", "/etc/openwrt_version"}
}
func (d *SOpenWrtRootFs) featureBoardConfig(rootFs IDiskPartition) bool {
	if rootFs.Exists("/etc/board.d", false) {
		return true
	}
	return false
}

func (d *SOpenWrtRootFs) putBoardConfig(rootFs IDiskPartition, f, c string) error {
	if err := rootFs.FilePutContents(f, c, false, false); err != nil {
		return err
	}
	if err := rootFs.Chmod(f, 0755, false); err != nil {
		return err
	}
	return nil
}

func (d *SOpenWrtRootFs) DeployPublicKey(rootFs IDiskPartition, selUsr string, pubkeys *deployapi.SSHKeys) error {
	if selUsr == "root" && rootFs.Exists("/etc/dropbear", false) {
		var (
			authFile = "/etc/dropbear/authorized_keys"
			uid      = 0
			gid      = 0
			replace  = false
		)
		return deployAuthorizedKeys(rootFs, authFile, uid, gid, pubkeys, replace)
	}
	return d.sLinuxRootFs.DeployPublicKey(rootFs, selUsr, pubkeys)
}

func (d *SOpenWrtRootFs) DeployHostname(rootFs IDiskPartition, hn, domain string) error {
	if d.featureBoardConfig(rootFs) {
		f := "/etc/board.d/00-00-onecloud-hostname"
		c := fmt.Sprintf(`. /lib/functions/uci-defaults.sh
board_config_update
ucidef_set_hostname '%s'
board_config_flush
exit 0
`, hn)
		return d.putBoardConfig(rootFs, f, c)
	}

	spath := "/etc/config/system"
	if !rootFs.Exists(spath, false) {
		return nil
	}
	bcont, err := rootFs.FileGetContents(spath, false)
	if err != nil {
		return err
	}
	cont := string(bcont)
	re := regexp.MustCompile("option hostname [^\n]+")
	cont = re.ReplaceAllString(cont, fmt.Sprintf("option hostname %s", hn))
	return rootFs.FilePutContents(spath, cont, false, false)
}

func (d *SOpenWrtRootFs) DeployNetworkingScripts(rootFs IDiskPartition, nics []*types.SServerNic) error {
	if d.featureBoardConfig(rootFs) {
		macs := ""
		for _, nic := range nics {
			macs = "," + nic.Mac
		}
		f := "/etc/board.d/00-01-onecloud-network"
		c := fmt.Sprintf(`. /lib/functions/uci-defaults.sh
[ -d /sys/class/net ] || exit 0

board_config_update
macs='%s'
i=0

oc_set_ifname() {
	local net="$1"; shift
	local ifname="$1"; shift

	if type ucidef_set_interface &>/dev/null; then
		ucidef_set_interface "$net" ifname "$ifname" protocol dhcp
	elif type ucidef_set_interface_raw &>/dev/null; then
		ucidef_set_interface_raw "$net" "$ifname" "dhcp"
	else
		echo "no ucidef function to do network ifname config" >&2
		exit 0
	fi
}

for ifname in $(ls /sys/class/net/); do
	p="/sys/class/net/$ifname"
	mac="$(cat "$p/address")"
	if [ "${macs#*,$mac}" != "$macs" ]; then
		oc_set_ifname "lan$i" "$ifname"
		i="$(($i + 1))"
	fi
done

board_config_flush
exit 0
`, macs)
		return d.putBoardConfig(rootFs, f, c)
	}
	return nil
}

func (d *SOpenWrtRootFs) GetReleaseInfo(rootFs IDiskPartition) *deployapi.ReleaseInfo {
	ver, _ := rootFs.FileGetContents("/etc/openwrt_version", false)
	return &deployapi.ReleaseInfo{
		Distro:  "OpenWrt",
		Version: string(ver),
		Arch:    d.GetArch(rootFs),
	}
}

type SCoreOsRootFs struct {
	*sGuestRootFsDriver
	config *coreosutils.SCloudConfig
}

func NewCoreOsRootFs(part IDiskPartition) IRootFsDriver {
	return &SCoreOsRootFs{sGuestRootFsDriver: newGuestRootFsDriver(part)}
}

func (d *SCoreOsRootFs) GetName() string {
	return "CoreOs"
}

func (d *SCoreOsRootFs) String() string {
	return "CoreOsRootFs"
}

func (d *SCoreOsRootFs) GetOs() string {
	return "Linux"
}

func (d *SCoreOsRootFs) GetReleaseInfo(rootFs IDiskPartition) *deployapi.ReleaseInfo {
	return &deployapi.ReleaseInfo{
		Distro:  "CoreOS",
		Version: "stable",
	}
}

func (d *SCoreOsRootFs) RootSignatures() []string {
	return []string{"cloud-config.yml"}
}

func (d *SCoreOsRootFs) PrepareFsForTemplate(rootFs IDiskPartition) error {
	return nil
}

func (d *SCoreOsRootFs) GetConfig() *coreosutils.SCloudConfig {
	if d.config == nil {
		d.config = coreosutils.NewCloudConfig()
		d.config.YunionInit()
		d.config.SetTimezone("Asia/Shanghai")
	}
	return d.config
}

func (d *SCoreOsRootFs) DeployHostname(rootFs IDiskPartition, hn, domain string) error {
	d.GetConfig().SetHostname(hn)
	return nil
}

func (d *SCoreOsRootFs) DeployPublicKey(rootFs IDiskPartition, selUsr string, pubkeys *deployapi.SSHKeys) error {
	return nil
}

func (d *SCoreOsRootFs) DeployHosts(rootFs IDiskPartition, hostname, domain string, ips []string) error {
	d.GetConfig().SetEtcHosts("localhost")
	return nil
}

func (d *SCoreOsRootFs) DeployNetworkingScripts(rootFs IDiskPartition, nics []*types.SServerNic) error {
	var netDevPrefix = GetNetDevPrefix(nics)
	for _, nic := range nics {
		name := fmt.Sprintf("%s%d", netDevPrefix, nic.Index)
		cont := "[Match]\n"
		cont += "Name=" + name + "\n"
		cont += "\n[Network]\n"
		cont += "DHCP=yes\n"
		if nic.Mtu > 0 {
			cont += "\n[Link]\n"
			cont += fmt.Sprintf("MTUBytes=%d\n", nic.Mtu)
		}
		runtime := true
		d.GetConfig().AddUnits("00-dhcp-"+name+".network", nil, nil, &runtime, cont, "", nil)
	}
	return nil
}

func (d *SCoreOsRootFs) DeployFstabScripts(rootFs IDiskPartition, disks []*deployapi.Disk) error {
	dataDiskIdx := 0
	for i := 1; i < len(disks); i++ {
		dev := fmt.Sprintf("UUID=%s", disks[i].DiskId)
		fs := disks[i].Fs
		if len(fs) > 0 {
			if fs == "swap" {
				d.GetConfig().AddSwap(dev)
			} else {
				mtPath := disks[i].Mountpoint
				if len(mtPath) == 0 {
					mtPath = "/data"
					if dataDiskIdx > 0 {
						mtPath += fmt.Sprintf("%d", dataDiskIdx)
					}
					dataDiskIdx += 1
				}
				d.GetConfig().AddPartition(dev, mtPath, fs)
			}
		}
	}
	return nil
}

func (d *SCoreOsRootFs) ChangeUserPasswd(rootFs IDiskPartition, account, gid, publicKey, password string) (string, error) {
	keys := []string{}
	if len(publicKey) > 0 {
		keys = append(keys, publicKey)
	}
	d.GetConfig().AddUser("core", password, keys, false)
	if len(publicKey) > 0 {
		return seclib2.EncryptBase64(publicKey, password)
	} else {
		return utils.EncryptAESBase64(gid, password)
	}
}

func (d *SCoreOsRootFs) GetLoginAccount(rootFs IDiskPartition, user string, defaultRootUser bool, windowsDefaultAdminUser bool) (string, error) {
	return "core", nil
}

func (d *SCoreOsRootFs) DeployFiles(deploys []*deployapi.DeployContent) error {
	for _, deploy := range deploys {
		d.GetConfig().AddWriteFile(deploy.Path, deploy.Content, "", "", false)
	}
	return nil
}

func (d *SCoreOsRootFs) CommitChanges(IDiskPartition) error {
	ocont, err := d.rootFs.FileGetContents("/cloud-config.yml", false)
	if err != nil {
		return err
	}
	ocfg := coreosutils.NewCloudConfig()
	err = yaml.Unmarshal(ocont, ocfg)
	if err != nil {
		log.Errorln(err)
	}
	conf := d.GetConfig()
	if len(ocfg.Users) > 0 {
		for _, u := range ocfg.Users {
			if conf.HasUser(u.Name) {
				conf.AddUser(u.Name, u.Passwd, u.SshAuthorizedKeys, true)
			}
		}
	}
	if len(ocfg.WriteFiles) > 0 {
		for _, f := range ocfg.WriteFiles {
			if !conf.HasWriteFile(f.Path) {
				conf.AddWriteFile(f.Path, f.Content, f.Permissions, f.Owner, f.Encoding == "base64")
			}
		}
	}
	return d.rootFs.FilePutContents("/cloud-config.yml", conf.String(), false, false)
}
