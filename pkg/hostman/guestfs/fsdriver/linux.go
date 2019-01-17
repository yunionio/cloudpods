package fsdriver

import (
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudcommon/sshkeys"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/fstabutils"
	"yunion.io/x/onecloud/pkg/util/netutils2"
	"yunion.io/x/onecloud/pkg/util/seclib2"
	"yunion.io/x/onecloud/pkg/util/sysutils"
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

func (l *sLinuxRootFs) GetLoginAccount(rootFs IDiskPartition) string {
	var selUsr string
	if options.HostOptions.LinuxDefaultRootUser && rootFs.Exists("/root", false) {
		selUsr = "root"
	} else {
		usrs := rootFs.ListDir("/home", false)
		for _, usr := range usrs {
			if len(selUsr) == 0 || len(selUsr) > len(usr) {
				selUsr = usr
			}
		}
		if len(selUsr) > 0 && rootFs.Exists("/root", false) {
			selUsr = "root"
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

func (l *sLinuxRootFs) DeployYunionroot(rootFs IDiskPartition, pubkeys *sshkeys.SSHKeys) error {
	l.DisableSelinux(rootFs)
	l.DisableCloudinit(rootFs)
	var yunionroot = "cloudroot"
	if err := rootFs.UserAdd(yunionroot, false); err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("UserAdd %s: %v", yunionroot, err)
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

	for i := 1; i < len(disks); i++ {
		diskId, err := disks[i].GetString("disk_id")
		if err != nil {
			diskId = "None"
		}
		dev := fmt.Sprintf("UUID=%s", diskId)
		if !fstab.IsExists(dev) {
			fs, _ := disks[i].GetString("fs")
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
			nicRules += `KERNEL=="eth*", SUBSYSTEM=="net", ACTION=="add", `
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
			nicRules += `KERNEL=="eth*", SUBSYSTEM=="net", ACTION=="add", `
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

func (d *sDebianLikeRootFs) DeployNetworkingScripts(rootFs IDiskPartition, nics []jsonutils.JSONObject) error {
	if err := d.sLinuxRootFs.DeployNetworkingScripts(rootFs, nics); err != nil {
		return err
	}
	fn := "/etc/network/interfaces"
	cmds := ""
	cmds += "auto lo\n"
	cmds += "iface lo inet loopback\n\n"
	mainNic, err := netutils2.GetMainNic(nics)
	if err != nil {
		return err
	}
	var mainIp string
	if mainNic != nil {
		mainIp, _ = mainNic.GetString("ip")
	}
	for _, nic := range nics {
		var nicDesc = new(types.ServerNic)
		err := nic.Unmarshal(nicDesc)
		if err != nil {
			return err
		}
		nicIdx, err := nic.Int("index")
		if err != nil {
			return err
		}
		cmds += fmt.Sprintf("auto eth%d\n", nicIdx)
		if jsonutils.QueryBoolean(nic, "virtual", false) {
			cmds += fmt.Sprintf("iface eth%d inet static\n", nicIdx)
			cmds += fmt.Sprintf("    address %s\n", netutils2.PSEUDO_VIP)
			cmds += "    netmask 255.255.255.255\n"
			cmds += "\n"
		} else if jsonutils.QueryBoolean(nic, "manual", false) {
			netmask := netutils2.Netlen2Mask(nicDesc.Masklen)
			ip, err := nic.GetString("ip")
			if err != nil {
				return err
			}
			cmds += fmt.Sprintf("iface eth%d inet static\n", nicIdx)
			cmds += fmt.Sprintf("    address %s\n", ip)
			cmds += fmt.Sprintf("    netmask %s\n", netmask)
			if len(nicDesc.Gateway) > 0 && ip == mainIp {
				cmds += fmt.Sprintf("    gateway %s\n", nicDesc.Gateway)
			}
			var routes = make([][]string, 0)
			netutils2.AddNicRoutes(&routes, nicDesc, mainIp, len(nics), options.HostOptions.PrivatePrefixes)
			for _, r := range routes {
				cmds += fmt.Sprintf("    up route add -net %s gw %s || true\n", r[0], r[1])
				cmds += fmt.Sprintf("    down route del -net %s gw %s || true\n", r[0], r[1])
			}
			dnslist := netutils2.GetNicDns(nicDesc)
			if len(dnslist) > 0 {
				cmds += fmt.Sprintf("    dns-nameservers %s\n", strings.Join(dnslist, " "))
				cmds += fmt.Sprintf("    dns-search %s\n", nicDesc.Domain)
			}
			cmds += "\n"
		} else {
			cmds += fmt.Sprintf("iface eth%d inet dhcp\n\n", nicIdx)
		}
	}
	return rootFs.FilePutContents(fn, cmds, false, false)
}

type SDebianRootFs struct {
	*sDebianLikeRootFs
}

func NewDebianRootFs(part IDiskPartition) IRootFsDriver {
	driver := new(SDebianRootFs)
	driver.sDebianLikeRootFs = newDebianLikeRootFs(part)
	return driver
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

/*
   udev_path = '/etc/udev/rules.d/'
   if self.root_fs.exists(udev_path):
       nic_rules = ''
       for nic in nics:
           nic_rules += 'KERNEL=="eth*", '
           nic_rules += 'SYSFS{address}=="%s", ' % (nic['mac'].lower())
           nic_rules += 'NAME="eth%d"\n' % (nic['index'])
       print nic_rules
       self.root_fs.file_put_contents(os.path.join(udev_path, '60-net.rules'), nic_rules)
*/

func (r *sRedhatLikeRootFs) Centos5DeployNetworkingScripts(rootFs IDiskPartition, nics []jsonutils.JSONObject) error {
	var udevPath = "/etc/udev/rules.d/"
	if rootFs.Exists(udevPath, false) {
		var nicRules = ""
		for _, nic := range nics {
			var nicdesc = new(types.ServerNic)
			if err := nic.Unmarshal(nicdesc); err != nil {
				return err
			}
			nicRules += `KERNEL=="eth*", `
			nicRules += fmt.Sprintf(`SYSFS{address}=="%s", `, strings.ToLower(nicdesc.Mac))
			nicRules += fmt.Sprintf("NAME=\"eth%d\"\n", nicdesc.Index)
		}
		return rootFs.FilePutContents(path.Join(udevPath, "60-net.rules"),
			nicRules, false, false)
	}
	return nil
}

func (r *sRedhatLikeRootFs) deployNetworkingScripts(rootFs IDiskPartition, nics []jsonutils.JSONObject, relInfo *SReleaseInfo) error {
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
	mainNic, err := netutils2.GetMainNic(nics)
	if err != nil {
		return err
	}
	var mainIp string
	if mainNic != nil {
		mainIp, _ = mainNic.GetString("ip")
	}
	for _, nic := range nics {
		var cmds string
		var nicdesc = new(types.ServerNic)
		if err := nic.Unmarshal(nicdesc); err != nil {
			return err
		}
		cmds += fmt.Sprintf("DEVICE=eth%d\n", nicdesc.Index)
		cmds += fmt.Sprintf("NAME=eth%d\n", nicdesc.Index)
		cmds += fmt.Sprintf("HWADDR=%s\n", nicdesc.Mac)
		cmds += fmt.Sprintf("MACADDR=%s\n", nicdesc.Mac)
		if nicdesc.Virtual {
			cmds += "BOOTPROTO=none\n"
			cmds += "NETMASK=255.255.255.255\n"
			cmds += fmt.Sprintf("IPADDR=%s\n", netutils2.PSEUDO_VIP)
			cmds += "USERCTL=no\n"
		} else if nicdesc.Manual {
			netmask := netutils2.Netlen2Mask(nicdesc.Masklen)
			cmds += "BOOTPROTO=none\n"
			cmds += fmt.Sprintf("NETMASK=%s\n", netmask)
			cmds += fmt.Sprintf("IPADDR=%s\n", nicdesc.Ip)
			cmds += "USERCTL=no\n"
			if len(nicdesc.Gateway) > 0 && nicdesc.Ip == mainIp {
				cmds += fmt.Sprintf("GATEWAY=%s\n", nicdesc.Gateway)
			}
			var routes = make([][]string, 0)
			var rtbl string
			netutils2.AddNicRoutes(&routes, nicdesc, mainIp, len(nics), options.HostOptions.PrivatePrefixes)
			for _, r := range routes {
				rtbl += fmt.Sprintf("%s via %s dev eth%d\n", r[0], r[1], nicdesc.Index)
			}
			if len(rtbl) > 0 {
				var fn = fmt.Sprintf("/etc/sysconfig/network-scripts/route-eth%d", nicdesc.Index)
				if err := rootFs.FilePutContents(fn, rtbl, false, false); err != nil {
					return err
				}
			}
			dnslist := netutils2.GetNicDns(nicdesc)
			if len(dnslist) > 0 {
				cmds += "PEERDNS=yes\n"
				for i := 0; i < len(dnslist); i++ {
					cmds += fmt.Sprintf("DNS%d=%s\n", i+1, dnslist[i])
				}
				cmds += fmt.Sprintf("DOMAIN=%s\n", nicdesc.Domain)
			}
		} else {
			cmds += "BOOTPROTO=dhcp\n"
		}
		var fn = fmt.Sprintf("/etc/sysconfig/network-scripts/ifcfg-eth%d", nicdesc.Index)
		if err := rootFs.FilePutContents(fn, cmds, false, false); err != nil {
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
		var nicdesc = new(types.ServerNic)
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

type SRhelRootFs struct {
	*sRedhatLikeRootFs
}

func NewRhelRootFs(part IDiskPartition) IRootFsDriver {
	return &SRhelRootFs{sRedhatLikeRootFs: newRedhatLikeRootFs(part)}
}

func (d *SRhelRootFs) GetName() string {
	return "RHEL"
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

/*type SGentooRootFs struct {
	*sLinuxRootFs
}

func NewGentooRootFs(part IDiskPartition) IRootFsDriver {
	return &SGentooRootFs{sLinuxRootFs: newLinuxRootFs(part)}
}

type SArchLinuxRootFs struct {
	*sLinuxRootFs
}

func NewArchLinuxRootFs(part IDiskPartition) IRootFsDriver {
	return &SArchLinuxRootFs{sLinuxRootFs: newLinuxRootFs(part)}
}

type SOpenWrtRootFs struct {
	*sLinuxRootFs
}

func NewOpenWrtRootFs(part IDiskPartition) IRootFsDriver {
	return &SOpenWrtRootFs{sLinuxRootFs: newLinuxRootFs(part)}
}

type SCoreOsRootFs struct {
	*SGuestRootFsDriver
}

func NewCoreOsRootFs(part IDiskPartition) IRootFsDriver {
	return &SCoreOsRootFs{SGuestRootFsDriver: newGuestRootFsDriver(part)}
}*/
