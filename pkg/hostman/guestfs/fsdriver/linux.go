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
	"regexp"
	"strconv"
	"strings"
	"syscall"

	yaml "gopkg.in/yaml.v2"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudcommon/sshkeys"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/util/coreosutils"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/fstabutils"
	"yunion.io/x/onecloud/pkg/util/netutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/seclib2"
	"yunion.io/x/onecloud/pkg/util/sysutils"
)

const (
	ROOT_USER       = "root"
	YUNIONROOT_USER = "cloudroot"
)

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
		hf.Add(ip, fmt.Sprintf("%s.%s", hostname, domain), hostname)
	}
	return rootFs.FilePutContents(etcHosts, hf.String(), false, false)
}

func (l *sLinuxRootFs) GetLoginAccount(rootFs IDiskPartition, defaultRootUser bool, windowsDefaultAdminUser bool) string {
	var selUsr string
	if defaultRootUser && rootFs.Exists("/root", false) {
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
	return selUsr
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
	} else {
		return "", fmt.Errorf("ChangeUserPasswd error: %v", err)
	}
	return secret, err
}

func (l *sLinuxRootFs) DeployPublicKey(rootFs IDiskPartition, selUsr string, pubkeys *sshkeys.SSHKeys) error {
	var usrDir string
	if selUsr == "root" {
		usrDir = "/root"
	} else {
		usrDir = path.Join("/home", selUsr)
	}
	return DeployAuthorizedKeys(rootFs, usrDir, pubkeys, false)
}

func (l *sLinuxRootFs) DeployYunionroot(rootFs IDiskPartition, pubkeys *sshkeys.SSHKeys, isInit, enableCloudInit bool) error {
	l.DisableSelinux(rootFs)
	if !enableCloudInit && isInit {
		l.DisableCloudinit(rootFs)
	}
	var yunionroot = YUNIONROOT_USER
	if err := rootFs.UserAdd(yunionroot, false); err != nil && !strings.Contains(err.Error(), "already exists") {
		log.Errorf("UserAdd %s: %v", yunionroot, err)
	}
	err := DeployAuthorizedKeys(rootFs, path.Join("/home", yunionroot), pubkeys, true)
	if err != nil {
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

func (l *sLinuxRootFs) DeployFstabScripts(rootFs IDiskPartition, disks []jsonutils.JSONObject) error {
	fstabcont, err := rootFs.FileGetContents("/etc/fstab", false)
	if err != nil {
		return err
	}
	var dataDiskIdx = 0
	var rec string
	var modeRwxOwner = syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IXUSR
	var fstab = fstabutils.FSTabFile(string(fstabcont))
	fstab.RemoveDevices(len(disks))

	for i := 1; i < len(disks); i++ {
		diskId, err := disks[i].GetString("disk_id")
		if err != nil {
			diskId = "None"
		}
		dev := fmt.Sprintf("UUID=%s", diskId)
		if !fstab.IsExists(dev) {
			fs, _ := disks[i].GetString("fs")
			if len(fs) > 0 {
				if fs == "swap" {
					rec = fmt.Sprintf("%s none %s sw 0 0", dev, fs)
				} else {
					mtPath, _ := disks[i].GetString("mountpoint")
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

func (l *sLinuxRootFs) DeployNetworkingScripts(rootFs IDiskPartition, nics []jsonutils.JSONObject) error {
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
			mac, _ := nic.GetString("mac")
			nicRules += fmt.Sprintf(`ATTR{address}=="%s", ATTR{type}=="1", `, strings.ToLower(mac))
			idx, _ := nic.Int("index")
			nicRules += fmt.Sprintf("NAME=\"eth%d\"\n", idx)
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
	return nil
}

func (l *sLinuxRootFs) DeployStandbyNetworkingScripts(rootFs IDiskPartition, nics, nicsStandby []jsonutils.JSONObject) error {
	var udevPath = "/etc/udev/rules.d/"
	var nicRules string
	for _, nic := range nicsStandby {
		nicType, _ := nic.GetString("nic_type")
		if !nic.Contains("nic_type") || nicType != "impi" {
			nicRules += `KERNEL=="*", SUBSYSTEM=="net", ACTION=="add", `
			nicRules += `DRIVERS=="?*", `
			mac, _ := nic.GetString("mac")
			nicRules += fmt.Sprintf(`ATTR{address}=="%s", ATTR{type}=="1", `, strings.ToLower(mac))
			idx, _ := nic.Int("index")
			nicRules += fmt.Sprintf("NAME=\"eth%d\"\n", idx)
		}
	}
	if err := rootFs.FilePutContents(path.Join(udevPath, "70-persistent-net.rules"), nicRules, false, false); err != nil {
		return err
	}
	return nil
}

func (l *sLinuxRootFs) GetOs() string {
	return "Linux"
}

func (l *sLinuxRootFs) GetArch(rootFs IDiskPartition) string {
	if rootFs.Exists("/lib64", false) && rootFs.Exists("/usr/lib64", false) {
		return "x86_64"
	} else {
		return "x86"
	}
}

func (l *sLinuxRootFs) PrepareFsForTemplate(rootFs IDiskPartition) error {
	// clean /etc/fstab
	if rootFs.Exists("/etc/fstab", false) {
		fstabcont, _ := rootFs.FileGetContents("/etc/fstab", false)
		fstab := fstabutils.FSTabFile(string(fstabcont))
		fstab.RemoveDevices(1)
		cf := fstab.ToConf()
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
	for _, dir := range []string{"/var/log", "/var/cache", "/usr/local/var/log", "/usr/local/var/cache"} {
		if rootFs.Exists(dir, false) {
			if err := l.rootFs.Zerofiles(dir, false); err != nil {
				return err
			}
		}
	}
	for _, dir := range []string{"/var/spool", "/var/run", "/run", "/usr/local/var/spool", "/usr/local/var/run"} {
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
	content, err := rootFs.FileGetContents(confpath, false)
	if err != nil {
		log.Errorf("Get %s error: %v", confpath, err)
		return nil
	}
	return sysutils.GetSerialPorts(strings.Split(string(content), "\n"))
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

func (l *sLinuxRootFs) enableSerialConsoleInit(rootFs IDiskPartition) error {
	// https://help.ubuntu.com/community/SerialConsoleHowto
	var err error
	for _, tty := range l.getSerialPorts(rootFs) {
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
		sPath := fmt.Sprintf("/etc/systemd/system/getty.target.wants/getty@%s.service", tty)
		if rootFs.Exists(sPath, false) {
			//rootFs.Symlink("/usr/lib/systemd/system/getty@.service", sPath)
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

type sDebianLikeRootFs struct {
	*sLinuxRootFs
}

func newDebianLikeRootFs(part IDiskPartition) *sDebianLikeRootFs {
	return &sDebianLikeRootFs{
		sLinuxRootFs: newLinuxRootFs(part),
	}
}

func (d *sDebianLikeRootFs) GetReleaseInfo(rootFs IDiskPartition, driver IDebianRootFsDriver) *SReleaseInfo {
	version, err := rootFs.FileGetContents(driver.VersionFilePath(), false)
	if err != nil {
		log.Errorf("Get %s error: %v", driver.VersionFilePath(), err)
		return nil
	}
	versionStr := strings.TrimSpace(string(version))
	return &SReleaseInfo{
		Distro:  driver.DistroName(),
		Version: versionStr,
		Arch:    d.GetArch(rootFs),
	}
}

func (d *sDebianLikeRootFs) RootSignatures() []string {
	sig := d.sLinuxRootFs.RootSignatures()
	return append([]string{"/etc/hostname"}, sig...)
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

func (d *sDebianLikeRootFs) DeployNetworkingScripts(rootFs IDiskPartition, nics []jsonutils.JSONObject) error {
	if err := d.sLinuxRootFs.DeployNetworkingScripts(rootFs, nics); err != nil {
		return err
	}
	fn := "/etc/network/interfaces"
	var cmds strings.Builder
	cmds.WriteString("auto lo\n")
	cmds.WriteString("iface lo inet loopback\n\n")

	allNics, _ := convertNicConfigs(nics)
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
			netmask := netutils2.Netlen2Mask(nicDesc.Masklen)
			cmds.WriteString(fmt.Sprintf("iface %s inet static\n", nicDesc.Name))
			cmds.WriteString(fmt.Sprintf("    address %s\n", nicDesc.Ip))
			cmds.WriteString(fmt.Sprintf("    netmask %s\n", netmask))
			if len(nicDesc.Gateway) > 0 && nicDesc.Ip == mainIp {
				cmds.WriteString(fmt.Sprintf("    gateway %s\n", nicDesc.Gateway))
			}
			var routes = make([][]string, 0)
			netutils2.AddNicRoutes(&routes, nicDesc, mainIp, len(nics), options.HostOptions.PrivatePrefixes)
			for _, r := range routes {
				cmds.WriteString(fmt.Sprintf("    up route add -net %s gw %s || true\n", r[0], r[1]))
				cmds.WriteString(fmt.Sprintf("    down route del -net %s gw %s || true\n", r[0], r[1]))
			}
			dnslist := netutils2.GetNicDns(nicDesc)
			if len(dnslist) > 0 {
				cmds.WriteString(fmt.Sprintf("    dns-nameservers %s\n", strings.Join(dnslist, " ")))
				cmds.WriteString(fmt.Sprintf("    dns-search %s\n", nicDesc.Domain))
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

func (d *SDebianRootFs) GetReleaseInfo(rootFs IDiskPartition) *SReleaseInfo {
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

func (d *SCirrosRootFs) GetReleaseInfo(rootFs IDiskPartition) *SReleaseInfo {
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

func (d *SCirrosNewRootFs) GetReleaseInfo(rootFs IDiskPartition) *SReleaseInfo {
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

func (d *SUbuntuRootFs) GetReleaseInfo(rootFs IDiskPartition) *SReleaseInfo {
	distroKey := "DISTRIB_RELEASE="
	rel, err := rootFs.FileGetContents("/etc/lsb-release", false)
	if err != nil {
		log.Errorf("Get ubuntu release info error: %v", err)
		return nil
	}
	var version string
	lines := strings.Split(string(rel), "\n")
	for _, l := range lines {
		if strings.HasPrefix(l, distroKey) {
			version = strings.TrimSpace(l[len(distroKey) : len(l)-1])
		}
	}
	return newReleaseInfo(d.GetName(), version, d.GetArch(rootFs))
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

type sRedhatLikeRootFs struct {
	*sLinuxRootFs
}

func newRedhatLikeRootFs(part IDiskPartition) *sRedhatLikeRootFs {
	return &sRedhatLikeRootFs{
		sLinuxRootFs: newLinuxRootFs(part),
	}
}

func (r *sRedhatLikeRootFs) RootSignatures() []string {
	sig := r.sLinuxRootFs.RootSignatures()
	return append([]string{"/etc/sysconfig/network", "/etc/redhat-release"}, sig...)
}

func (r *sRedhatLikeRootFs) DeployHostname(rootFs IDiskPartition, hn, domain string) error {
	var sPath = "/etc/sysconfig/network"
	centosHn := ""
	centosHn += "NETWORKING=yes\n"
	centosHn += fmt.Sprintf("HOSTNAME=%s.%s\n", hn, domain)
	if err := rootFs.FilePutContents(sPath, centosHn, false, false); err != nil {
		return err
	}
	if rootFs.Exists("/etc/hostname", false) {
		return rootFs.FilePutContents("/etc/hostname", hn, false, false)
	}
	return nil
}

func (r *sRedhatLikeRootFs) Centos5DeployNetworkingScripts(rootFs IDiskPartition, nics []jsonutils.JSONObject) error {
	var udevPath = "/etc/udev/rules.d/"
	if rootFs.Exists(udevPath, false) {
		var nicRules = ""
		for _, nic := range nics {
			var nicdesc = new(types.SServerNic)
			if err := nic.Unmarshal(nicdesc); err != nil {
				return err
			}
			nicRules += `KERNEL=="*", `
			nicRules += fmt.Sprintf(`SYSFS{address}=="%s", `, strings.ToLower(nicdesc.Mac))
			nicRules += fmt.Sprintf("NAME=\"eth%d\"\n", nicdesc.Index)
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

func (r *sRedhatLikeRootFs) deployNetworkingScripts(rootFs IDiskPartition, nics []jsonutils.JSONObject, relInfo *SReleaseInfo) error {
	if err := r.sLinuxRootFs.DeployNetworkingScripts(rootFs, nics); err != nil {
		return err
	}

	ver := strings.Split(relInfo.Version, ".")
	iv, err := strconv.ParseInt(ver[0], 10, 0)
	if err != nil {
		return fmt.Errorf("Failed to get release version: %v", err)
	}
	if iv < 6 {
		err = r.Centos5DeployNetworkingScripts(rootFs, nics)
	} else {
		err = r.sLinuxRootFs.DeployNetworkingScripts(rootFs, nics)
	}
	if err != nil {
		return err
	}
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
		cmds.WriteString("NM_CONTROLLED=no\n")
		cmds.WriteString("USERCTL=no\n")
		if len(nicDesc.Mac) > 0 {
			cmds.WriteString("HWADDR=")
			cmds.WriteString(nicDesc.Mac)
			cmds.WriteString("\n")
			cmds.WriteString("MACADDR=")
			cmds.WriteString(nicDesc.Mac)
			cmds.WriteString("\n")
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
			netmask := netutils2.Netlen2Mask(nicDesc.Masklen)
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
			netutils2.AddNicRoutes(&routes, nicDesc, mainIp, len(nics), options.HostOptions.PrivatePrefixes)
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
				cmds.WriteString(fmt.Sprintf("DOMAIN=%s\n", nicDesc.Domain))
			}
		} else {
			cmds.WriteString("BOOTPROTO=dhcp\n")
		}
		var fn = fmt.Sprintf("/etc/sysconfig/network-scripts/ifcfg-%s", nicDesc.Name)
		if err := rootFs.FilePutContents(fn, cmds.String(), false, false); err != nil {
			return err
		}
	}
	return nil
}

func (r *sRedhatLikeRootFs) DeployStandbyNetworkingScripts(rootFs IDiskPartition, nics, nicsStandby []jsonutils.JSONObject) error {
	if err := r.sLinuxRootFs.DeployStandbyNetworkingScripts(rootFs, nics, nicsStandby); err != nil {
		return err
	}
	for _, nic := range nicsStandby {
		var cmds string
		var nicdesc = new(types.SServerNic)
		if err := nic.Unmarshal(nicdesc); err != nil {
			return err
		}
		if nicType, err := nic.GetString("nic_type"); err != nil && nicType != "ipmi" {
			cmds += fmt.Sprintf("DEVICE=eth%d\n", nicdesc.Index)
			cmds += fmt.Sprintf("NAME=eth%d\n", nicdesc.Index)
			cmds += fmt.Sprintf("HWADDR=%s\n", nicdesc.Mac)
			cmds += fmt.Sprintf("MACADDR=%s\n", nicdesc.Mac)
			cmds += "ONBOOT=no\n"
			var fn = fmt.Sprintf("/etc/sysconfig/network-scripts/ifcfg-eth%d", nicdesc.Index)
			if err := rootFs.FilePutContents(fn, cmds, false, false); err != nil {
				return err
			}
		}
	}
	return nil
}

//TODO enable_serial_console
//TODO disable_serial_console

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

func (c *SCentosRootFs) GetReleaseInfo(rootFs IDiskPartition) *SReleaseInfo {
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
	return newReleaseInfo(c.GetName(), version, c.GetArch(rootFs))
}

func (c *SCentosRootFs) DeployNetworkingScripts(rootFs IDiskPartition, nics []jsonutils.JSONObject) error {
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

func (c *SFedoraRootFs) GetReleaseInfo(rootFs IDiskPartition) *SReleaseInfo {
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
	return newReleaseInfo(c.GetName(), version, c.GetArch(rootFs))
}

func (c *SFedoraRootFs) DeployNetworkingScripts(rootFs IDiskPartition, nics []jsonutils.JSONObject) error {
	relInfo := c.GetReleaseInfo(rootFs)
	if err := c.sRedhatLikeRootFs.deployNetworkingScripts(rootFs, nics, relInfo); err != nil {
		return err
	}
	return nil
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

func (d *SRhelRootFs) GetReleaseInfo(rootFs IDiskPartition) *SReleaseInfo {
	rel, _ := rootFs.FileGetContents("/etc/redhat-release", false)
	var version string
	if len(rel) > 0 {
		dat := strings.Split(string(rel), " ")
		if len(dat) > 6 {
			version = dat[6]
		}
	}
	return newReleaseInfo(d.GetName(), version, d.GetArch(rootFs))
}

func (d *SRhelRootFs) DeployNetworkingScripts(rootFs IDiskPartition, nics []jsonutils.JSONObject) error {
	relInfo := d.GetReleaseInfo(rootFs)
	if err := d.sRedhatLikeRootFs.deployNetworkingScripts(rootFs, nics, relInfo); err != nil {
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

func (l *SGentooRootFs) DeployNetworkingScripts(rootFs IDiskPartition, nics []jsonutils.JSONObject) error {
	if err := l.sLinuxRootFs.DeployNetworkingScripts(rootFs, nics); err != nil {
		return err
	}

	var (
		fn   = "/etc/conf.d/net"
		cmds = ""
	)

	for _, nic := range nics {
		nicIndex, _ := nic.Int("index")
		if jsonutils.QueryBoolean(nic, "virtual", false) {
			cmds += fmt.Sprintf(`config_eth%d="`, nicIndex)
			cmds += fmt.Sprintf("%s netmask 255.255.255.255", netutils2.PSEUDO_VIP)
			cmds += `"\n`
		} else {
			cmds += fmt.Sprintf(`config_eth%d="dhcp"\n`, nicIndex)
		}
	}
	if err := rootFs.FilePutContents(fn, cmds, false, false); err != nil {
		return err
	}
	for _, nic := range nics {
		nicIndex, _ := nic.Int("index")
		netname := fmt.Sprintf("net.eth%d", nicIndex)
		procutils.NewCommand("ln", "-s", "net.lo",
			fmt.Sprintf("%s/etc/init.d/%s", rootFs.GetMountPath(), netname)).Run()
		procutils.NewCommand("chroot",
			rootFs.GetMountPath(), "rc-update", "add", netname, "default").Run()
	}
	return nil
}

func (d *SGentooRootFs) GetReleaseInfo(rootFs IDiskPartition) *SReleaseInfo {
	return &SReleaseInfo{
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

func (d *SArchLinuxRootFs) GetReleaseInfo(rootFs IDiskPartition) *SReleaseInfo {
	return &SReleaseInfo{
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
	return "OpenWRT"
}

func (d *SOpenWrtRootFs) String() string {
	return "OpenWrtRootFs"
}

func (d *SOpenWrtRootFs) RootSignatures() []string {
	return []string{"/bin", "/etc/", "/lib", "/sbin", "/overlay", "/etc/openwrt_release", "/etc/openwrt_version"}
}

func (d *SOpenWrtRootFs) DeployHostname(rootFs IDiskPartition, hn, domain string) error {
	spath := "/etc/config/system"
	bcont, err := rootFs.FileGetContents(spath, false)
	if err != nil {
		return err
	}
	cont := string(bcont)
	re := regexp.MustCompile("option hostname [^\n]+")
	cont = re.ReplaceAllString(cont, fmt.Sprintf("option hostname %s", hn))
	return rootFs.FilePutContents(spath, cont, false, false)
}

func (d *SOpenWrtRootFs) GetReleaseInfo(rootFs IDiskPartition) *SReleaseInfo {
	ver, _ := rootFs.FileGetContents("/etc/openwrt_version", false)
	return &SReleaseInfo{
		Distro:  "OpenWRT",
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

func (d *SCoreOsRootFs) GetReleaseInfo(rootFs IDiskPartition) *SReleaseInfo {
	return &SReleaseInfo{
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

func (d *SCoreOsRootFs) DeployPublicKey(rootFs IDiskPartition, selUsr string, pubkeys *sshkeys.SSHKeys) error {
	return nil
}

func (d *SCoreOsRootFs) DeployHosts(rootFs IDiskPartition, hostname, domain string, ips []string) error {
	d.GetConfig().SetEtcHosts("localhost")
	return nil
}

func (d *SCoreOsRootFs) DeployNetworkingScripts(rootFs IDiskPartition, nics []jsonutils.JSONObject) error {
	cont := "[Match]\n"
	cont += "Name=eth*\n\n"
	cont += "[Network]\n"
	cont += "DHCP=yes\n"
	runtime := true
	d.GetConfig().AddUnits("00-dhcp.network", nil, nil, &runtime, cont, "", nil)
	return nil
}

func (d *SCoreOsRootFs) DeployFstabScripts(rootFs IDiskPartition, disks []jsonutils.JSONObject) error {
	dataDiskIdx := 0
	for i := 1; i < len(disks); i++ {
		diskId, _ := disks[i].GetString("disk_id")
		dev := fmt.Sprintf("UUID=%s", diskId)
		fs, _ := disks[i].GetString("fs")
		if len(fs) > 0 {
			if fs == "swap" {
				d.GetConfig().AddSwap(dev)
			} else {
				mtPath, _ := disks[i].GetString("mountpoint")
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

func (d *SCoreOsRootFs) GetLoginAccount(rootFs IDiskPartition, defaultRootUser bool, windowsDefaultAdminUser bool) string {
	return "core"
}

func (d *SCoreOsRootFs) DeployFiles(deploys []jsonutils.JSONObject) error {
	for _, deploy := range deploys {
		spath, _ := deploy.GetString("path")
		content, _ := deploy.GetString("content")
		d.GetConfig().AddWriteFile(spath, content, "", "", false)
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
