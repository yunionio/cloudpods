package guestfs

import (
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon"
	"yunion.io/x/onecloud/pkg/cloudcommon/fstabutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/sshkeys"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/util/seclib2"
	"yunion.io/x/pkg/utils"
)

type SLinuxRootFs struct {
	*SGuestRootFsDriver
}

func NewLinuxRootFs(part *SKVMGuestDiskPartition) IRootFsDriver {
	return &SLinuxRootFs{SGuestRootFsDriver: NewGuestRootFsDriver(part).(*SGuestRootFsDriver)}
}

func (l *SLinuxRootFs) String() string {
	return "LinuxRootFs"
}

func (l *SLinuxRootFs) RootSignatures() []string {
	return []string{"/bin", "/etc", "/boot", "/lib", "/usr"}
}

func (l *SLinuxRootFs) DeployHost(hn, domain string, ips []string) error {
	var oldHostFile string
	if l.rootFs.Exists("/etc/hosts", false) {
		oldhf, err := l.rootFs.FileGetContents("/etc/hosts", false)
		if err != nil {
			return err
		}
		oldHostFile = string(oldhf)
	}
	hf := make(cloudcommon.HostsFile, 0)
	hf.Parse(oldHostFile)
	hf.Add("127.0.0.1", "localhost")
	for _, ip := range ips {
		hf.Add(ip, fmt.Sprintf("%s.%s", hn+domain), hn)
	}
	return nil
}

func (l *SLinuxRootFs) GetLoginAccount() string {
	var selUsr string
	if options.HostOptions.LinuxDefaultRootUser && l.rootFs.Exists("/root", false) {
		selUsr = "root"
	} else {
		usrs := l.rootFs.Listdir("/home", false)
		for _, usr := range usrs {
			if len(selUsr) == 0 || len(selUsr) > len(usr) {
				selUsr = usr
			}
		}
		if len(selUsr) > 0 && l.rootFs.Exists("/root", false) {
			selUsr = "root"
		}
	}
	return selUsr
}

func (l *SLinuxRootFs) ChangeUserPassswd(account, gid, publicKey, password string) string {
	var secret string
	if err := l.rootFs.Passwd(account, password, false); err != nil {
		if len(publicKey) > 0 {
			secret, _ = seclib2.EncryptBase64(publicKey, password)
		} else {
			secret, _ = utils.EncryptAESBase64(gid, password)
		}
	} else {
		log.Errorf("Change uer passwd error: %s", err)
	}
	return secret
}

func (l *SLinuxRootFs) DeployPublicKey(selUsr string, pubkeys *sshkeys.SSHKeys) error {
	var usrDir string
	if selUsr == "root" {
		usrDir = "/root"
	} else {
		usrDir = path.Join("/home", selUsr)
	}
	return l.rootFs.DeployAuthorizedKeys(usrDir, pubkeys, false)
}

func (l *SLinuxRootFs) DeployYunionroot(pubkeys *sshkeys.SSHKeys) error {
	l.DisableSelinux()
	l.DisableCloudinit()
	var yunionroot = "cloudroot"
	if err := l.rootFs.UserAdd(yunionroot, false); err != nil {
		return err
	}
	err := l.rootFs.DeployAuthorizedKeys(path.Join("/home", yunionroot), pubkeys, false)
	if err != nil {
		return err
	}
	return l.EnableUserSudo(yunionroot)
}

func (l *SLinuxRootFs) EnableUserSudo(user string) error {
	var sudoDir = "/etc/sudoers.d"
	var content = fmt.Sprintf("%s ALL=(ALL) NOPASSWD:ALL\n", user)
	if l.rootFs.Exists(sudoDir, false) {
		filepath := path.Join(sudoDir, fmt.Sprintf("90-%s-users", user))
		err := l.rootFs.FilePutContents(filepath, content, false, false)
		if err != nil {
			log.Errorln(err)
			return err
		}
		return l.rootFs.Chmod(filepath, syscall.S_IRUSR|syscall.S_IRGRP, false)
	}
	return nil
}

func (l *SLinuxRootFs) DisableSelinux() {
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
	if l.rootFs.Exists(selinuxConfig, false) {
		l.rootFs.FilePutContents(selinuxConfig, content, false, false)
	}
}

func (l *SLinuxRootFs) DisableCloudinit() {
	cloudDir := "/etc/cloud"
	cloudDisableFile := "/etc/cloud/cloud-init.disabled"
	if l.rootFs.Exists(cloudDir, false) {
		l.rootFs.FilePutContents(cloudDisableFile, "", false, false)
	}
}

func (l *SLinuxRootFs) DeployFstabScripts(disks []jsonutils.JSONObject) error {
	fstabcont, err := l.rootFs.FileGetContents("/etc/fstab", false)
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
			fs, err := disks[i].GetString("fs")
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
	return l.rootFs.FilePutContents("/etc/fstab", cf, false, false)
}

func (l *SLinuxRootFs) DeployNetworkingScripts(nics []jsonutils.JSONObject) error {
	udevPath := "/etc/udev/rules.d/"
	if l.rootFs.Exists(udevPath, false) {
		rules := l.rootFs.Listdir(udevPath, false)
		for _, rule := range rules {
			if strings.Index(rule, "persistent-net.rules") > 0 {
				l.rootFs.Remove(path.Join(udevPath, rule), false)
			} else if strings.Index(rule, "persistent-cd.rules") > 0 {
				if err := l.rootFs.FilePutContents(path.Join(udevPath, rule), "", false, false); err != nil {
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
			nicRules += fmt.Sprintf(`NAME="eth%d"\n`, idx)
		}
		if err := l.rootFs.FilePutContents(path.Join(udevPath, "70-persistent-net.rules"), nicRules, false, false); err != nil {
			return err
		}

		var usbRules string
		usbRules = `SUBSYSTEM=="usb", ATTRS{idVendor}=="1d6b", ATTRS{idProduct}=="0001", `
		usbRules += `RUN+="/bin/sh -c \'echo enabled > /sys$env{DEVPATH}/../power/wakeup\'"\n`
		if err := l.rootFs.FilePutContents(path.Join(udevPath,
			"90-usb-tablet-remote-wakeup.rules"), usbRules, false, false); err != nil {
			return err
		}
	}
	return nil
}

func (l *SLinuxRootFs) DeployStandbyNetworkingScripts(nics, nicsStandby []jsonutils.JSONObject) error {
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
			nicRules += fmt.Sprintf(`NAME="eth%d"\n`, idx)
		}
	}
	if err := l.rootFs.FilePutContents(path.Join(udevPath, "70-persistent-net.rules"), nicRules, false, false); err != nil {
		return err
	}
	return nil
}

func (l *SLinuxRootFs) GetOs() string {
	return "Linux"
}

func (l *SLinuxRootFs) GetArch() string {
	if l.rootFs.Exists("/lib64", false) && l.rootFs.Exists("/usr/lib64", false) {
		return "x86_64"
	} else {
		return "x86"
	}
}

func (l *SLinuxRootFs) PrepareFsForTemplate() error {
	// clean /etc/fstab
	if l.rootFs.Exists("/etc/fstab", false) {
		fstabcont, _ := l.rootFs.FileGetContents("/etc/fstab", false)
		fstab := fstabutils.FSTabFile(string(fstabcont))
		fstab.RemoveDevices(1)
		cf := fstab.ToConf()
		if err := l.rootFs.FilePutContents("/etc/fstab", cf, false, false); err != nil {
			return err
		}
	}
	// rm /etc/ssh/*_key.*
	if l.rootFs.Exists("/etc/ssh", false) {
		for _, f := range l.rootFs.Listdir("/etc/ssh", false) {
			if strings.HasSuffix(f, "_key") || strings.HasSuffix(f, "_key.pub") {
				l.rootFs.Remove("/etc/ssh/"+f, false)
			}
		}
	}
	// clean cloud-init
	if l.rootFs.Exists("/var/lib/cloud", false) {
		if err := l.rootFs.Cleandir("/var/lib/cloud", false, false); err != nil {
			return err
		}
	}
	cloudDisableFile := "/etc/cloud/cloud-init.disabled"
	if l.rootFs.Exists(cloudDisableFile, false) {
		l.rootFs.Remove(cloudDisableFile, false)
	}
	// clean /tmp /var/log /var/cache /var/spool /var/run
	for _, dir := range []string{"/tmp", "/var/tmp"} {
		if l.rootFs.Exists(dir, false) {
			if err := l.rootFs.Cleandir(dir, false, false); err != nil {
				return err
			}
		}
	}
	for _, dir := range []string{"/var/log", "/var/cache", "/usr/local/var/log", "/usr/local/var/cache"} {
		if l.rootFs.Exists(dir, false) {
			if err := l.rootFs.Zerofiles(dir, false); err != nil {
				return err
			}
		}
	}
	for _, dir := range []string{"/var/spool", "/var/run", "/run", "/usr/local/var/spool", "/usr/local/var/run"} {
		if l.rootFs.Exists(dir, false) {
			if err := l.rootFs.Cleandir(dir, true, true); err != nil {
				return err
			}
		}
	}
	return nil
}

func (l *SLinuxRootFs) GetSerialPorts() []string {
	var confpath = "/proc/tty/driver/serial"
	if l.rootFs.SupportOsPathExists() {
		// TODO: with sshpart utils
		return nil
	} else {
		return nil
	}
}

// TODO: _enable_serial_console_init_centos, _enable_serial_console_init,
// _disable_serial_console_init, _enable_serial_console_systemd, _disable_serial_console_systemd

type SDebianLikeRootFs struct {
	*SLinuxRootFs
}

func NewDebianLikeRootFs(part *SKVMGuestDiskPartition) IRootFsDriver {
	return &SDebianLikeRootFs{SLinuxRootFs: NewLinuxRootFs(part).(*SLinuxRootFs)}
}

func (d *SDebianLikeRootFs) RootSignatures() []string {
	sig := d.SLinuxRootFs.RootSignatures()
	return append([]string{"/etc/hostname"}, sig...)
}

func (d *SDebianLikeRootFs) DeployHostname(hn, domain string) error {
	return d.rootFs.FilePutContents("/etc/hostname", hn, false, false)
}

func (d *SDebianLikeRootFs) DeployNetworkingScripts(nics []jsonutils.JSONObject) error {
	if err := d.SLinuxRootFs.DeployNetworkingScripts(nics); err != nil {
		return err
	}
	fn := "/etc/network/interfaces"
	cmds := ""
	cmds += "auto lo\n"
	cmds += "iface lo inet loopback\n\n"
	mainNic, err := cloudcommon.GetMainNic(nics)
	if err != nil {
		return err
	}
	var mainIp string
	if mainNic != nil {
		mainIp, _ := mainNic.GetString("ip")
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
			cmds += fmt.Sprintf("    address %s\n", cloudcommon.PSEUDO_VIP)
			cmds += "    netmask 255.255.255.255\n"
			cmds += "\n"
		} else if jsonutils.QueryBoolean(nic, "manual", false) {
			netmask := cloudcommon.Netlen2Mask(nicDesc.Masklen)
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
			cloudcommon.AddNicRoutes(&routes, nicDesc, mainIp, len(nics), options.HostOptions.PrivatePrefixes)
			for _, r := range routes {
				cmds += fmt.Sprintf("    up route add -net %s gw %s || true\n", r[0], r[1])
				cmds += fmt.Sprintf("    down route del -net %s gw %s || true\n", r[0], r[1])
			}
			dnslist := cloudcommon.GetNicDns(nicDesc)
			if len(dnslist) > 0 {
				cmds += fmt.Sprintf("    dns-nameservers %s\n", strings.Join(dnslist, " "))
				cmds += fmt.Sprintf("    dns-search %s\n", nicDesc.Domain)
			}
			cmds += "\n"
		} else {
			cmds += fmt.Sprintf("iface eth%d inet dhcp\n\n", nicIdx)
		}
	}
	return d.rootFs.FilePutContents(fn, cmds, false, false)
}

type SDebianRootFs struct {
	*SDebianLikeRootFs
}

func NewDebianRootFs(part *SKVMGuestDiskPartition) IRootFsDriver {
	return &SDebianRootFs{SDebianLikeRootFs: NewDebianLikeRootFs(part).(*SDebianLikeRootFs)}
}

type SCirrosRootFs struct {
	*SDebianRootFs
}

func NewCirrosRootFs(part *SKVMGuestDiskPartition) IRootFsDriver {
	return &SCirrosRootFs{SDebianRootFs: NewDebianRootFs(part).(*SDebianRootFs)}
}

type SCirrosNewRootFs struct {
	*SDebianRootFs
}

func NewCirrosNewRootFs(part *SKVMGuestDiskPartition) IRootFsDriver {
	return &SCirrosNewRootFs{SDebianRootFs: NewDebianRootFs(part).(*SDebianRootFs)}
}

type SUbuntuRootFs struct {
	*SDebianRootFs
}

func NewUbuntuRootFs(part *SKVMGuestDiskPartition) IRootFsDriver {
	return &SUbuntuRootFs{SDebianRootFs: NewDebianRootFs(part).(*SDebianRootFs)}
}

type SRedhatLikeRootFs struct {
	*SLinuxRootFs
}

func NewRedhatLikeRootFs(part *SKVMGuestDiskPartition) IRootFsDriver {
	return &SRedhatLikeRootFs{SLinuxRootFs: NewLinuxRootFs(part).(*SLinuxRootFs)}
}

func (r *SRedhatLikeRootFs) RootSignatures() []string {
	sig := r.SLinuxRootFs.RootSignatures()
	return append([]string{"/etc/sysconfig/network", "/etc/redhat-release"}, sig...)
}

func (r *SRedhatLikeRootFs) DeployHostname(hn, domain string) error {
	var sPath = "/etc/sysconfig/network"
	centosHn := ""
	centosHn += "NETWORKING=yes\n"
	centosHn += fmt.Sprintf("HOSTNAME=%s.%s\n", hn, domain)
	if err := r.rootFs.FilePutContents(sPath, centosHn, false, false); err != nil {
		return err
	}
	if r.rootFs.Exists("/etc/hostname", false) {
		return r.rootFs.FilePutContents("/etc/hostname", hn, false, false)
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

func (r *SRedhatLikeRootFs) Centos5DeployNetworkingScripts(nics []jsonutils.JSONObject) error {
	var udevPath = "/etc/udev/rules.d/"
	if r.rootFs.Exists(udevPath, false) {
		var nicRules = ""
		for _, nic := range nics {
			var nicdesc = new(types.ServerNic)
			if err := nic.Unmarshal(nicdesc); err != nil {
				return err
			}
			nicRules += `KERNEL=="eth*", `
			nicRules += fmt.Sprintf(`SYSFS{address}=="%s", `, strings.ToLower(nicdesc.Mac))
			nicRules += fmt.Sprintf(`NAME="eth%d"\`, nicdesc.Index)
		}
		return r.rootFs.FilePutContents(udevPath, nicRules, false, false)
	}
	return nil
}

func (r *SRedhatLikeRootFs) DeployNetworkingScripts(nics []jsonutils.JSONObject) error {
	relinfo := r.GetReleaseInfo()
	ver := strings.Split(relinfo[0], ".")
	iv, err := strconv.ParseInt(ver[0], 10, 0)
	if err != nil {
		return err
	}
	if iv < 6 {
		err = r.Centos5DeployNetworkingScripts(nics)
	} else {
		err = r.SLinuxRootFs.DeployNetworkingScripts(nics)
	}
	if err != nil {
		return err
	}
	mainNic, err := cloudcommon.GetMainNic(nics)
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
			cmds += fmt.Sprintf("IPADDR=%s\n", cloudcommon.PSEUDO_VIP)
			cmds += "USERCTL=no\n"
		} else if nicdesc.Manual {
			netmask := cloudcommon.Netlen2Mask(nicdesc.Masklen)
			cmds += "BOOTPROTO=none\n"
			cmds += fmt.Sprintf("NETMASK=%s\n", netmask)
			cmds += fmt.Sprintf("IPADDR=%s\n", nicdesc.Ip)
			cmds += "USERCTL=no\n"
			if len(nicdesc.Gateway) > 0 && nicdesc.Ip == mainIp {
				cmds += fmt.Sprintf("GATEWAY=%s\n", nicdesc.Gateway)
			}
			var routes = make([][]string, 0)
			var rtbl string
			cloudcommon.AddNicRoutes(&routes, nicdesc, mainIp, len(nics), options.HostOptions.PrivatePrefixes)
			for _, r := range routes {
				rtbl += fmt.Sprintf("%s via %s dev eth%d\n", r[0], r[1], nicdesc.Index)
			}
			if len(rtbl) > 0 {
				var fn = fmt.Sprintf("/etc/sysconfig/network-scripts/route-eth%d", nicdesc.Index)
				if err := r.rootFs.FilePutContents(fn, rtbl, false, false); err != nil {
					return err
				}
			}
			dnslist := cloudcommon.GetNicDns(nicdesc)
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
		if err := r.rootFs.FilePutContents(fn, cmds, false, false); err != nil {
			return err
		}
	}
	return nil
}

func (r *SRedhatLikeRootFs) DeployStandbyNetworkingScripts(nics, nicsStandby []jsonutils.JSONObject) error {
	if err := r.SLinuxRootFs.DeployStandbyNetworkingScripts(nics, nicsStandby); err != nil {
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
			if err := r.rootFs.FilePutContents(fn, cmds, false, false); err != nil {
				return err
			}
		}
	}
	return nil
}

//TODO enable_serial_console
//TODO disable_serial_console

type SCentosRootFs struct {
	*SRedhatLikeRootFs
}

func NewCentosRootFs(part *SKVMGuestDiskPartition) IRootFsDriver {
	return &SCentosRootFs{SRedhatLikeRootFs: NewRedhatLikeRootFs(part).(*SRedhatLikeRootFs)}
}

func (c *SCentosRootFs) RootSignatures() []string {
	sig := c.SRedhatLikeRootFs.RootSignatures()
	return append([]string{"/etc/centos-release"}, sig...)
}

func (c *SCentosRootFs) GetReleaseInfo() []string {
	rel, _ := c.rootFs.FileGetContents("/etc/centos-release", false)
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
	return []string{"CentOS", version, c.GetArch()}
}

func (c *SCentosRootFs) DeployNetworkingScripts(nics []jsonutils.JSONObject) error {
	if err := c.SRedhatLikeRootFs.DeployNetworkingScripts(nics); err != nil {
		return err
	}
	var udevPath = "/etc/udev/rules.d/"
	var files = []string{"60-net.rules", "75-persistent-net-generator.rules"}
	for _, f := range files {
		sPath := path.Join(udevPath, f)
		if !c.rootFs.Exists(sPath, false) {
			if err := c.rootFs.FilePutContents(sPath, "", false, false); err != nil {
				return err
			}
		}
	}
	return nil
}

type SFedoraRootFs struct {
	*SRedhatLikeRootFs
}

func NewFedoraRootFs(part *SKVMGuestDiskPartition) IRootFsDriver {
	return &SFedoraRootFs{SRedhatLikeRootFs: NewRedhatLikeRootFs(part).(*SRedhatLikeRootFs)}
}

type SRhelRootFs struct {
	*SRedhatLikeRootFs
}

func NewRhelRootFs(part *SKVMGuestDiskPartition) IRootFsDriver {
	return &SRhelRootFs{SRedhatLikeRootFs: NewRedhatLikeRootFs(part).(*SRedhatLikeRootFs)}
}

type SGentooRootFs struct {
	*SLinuxRootFs
}

func NewGentooRootFs(part *SKVMGuestDiskPartition) IRootFsDriver {
	return &SGentooRootFs{SLinuxRootFs: NewLinuxRootFs(part).(*SLinuxRootFs)}
}

type SArchLinuxRootFs struct {
	*SLinuxRootFs
}

func NewArchLinuxRootFs(part *SKVMGuestDiskPartition) IRootFsDriver {
	return &SArchLinuxRootFs{SLinuxRootFs: NewLinuxRootFs(part).(*SLinuxRootFs)}
}

type SOpenWrtRootFs struct {
	*SLinuxRootFs
}

func NewOpenWrtRootFs(part *SKVMGuestDiskPartition) IRootFsDriver {
	return &SOpenWrtRootFs{SLinuxRootFs: NewLinuxRootFs(part).(*SLinuxRootFs)}
}

type SCoreOsRootFs struct {
	*SGuestRootFsDriver
}

func NewCoreOsRootFs(part *SKVMGuestDiskPartition) IRootFsDriver {
	return &SCoreOsRootFs{SGuestRootFsDriver: NewGuestRootFsDriver(part).(*SGuestRootFsDriver)}
}

func init() {
	linuxFsDrivers := []newRootFsDriverFunc{
		NewLinuxRootFs, NewDebianLikeRootFs, NewDebianRootFs, NewCirrosRootFs, NewCirrosNewRootFs,
		NewUbuntuRootFs, NewRedhatLikeRootFs, NewCentosRootFs, NewFedoraRootFs, NewRhelRootFs,
		NewGentooRootFs, NewArchLinuxRootFs, NewOpenWrtRootFs, NewCoreOsRootFs,
	}
	rootfsDrivers = append(rootfsDrivers, linuxFsDrivers...)
}
