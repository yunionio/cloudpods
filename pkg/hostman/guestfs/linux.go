package guestfs

import (
	"fmt"
	"path"
	"strings"
	"syscall"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/fstabutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/sshkeys"
	"yunion.io/x/onecloud/pkg/hostman"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/util/seclib2"
	"yunion.io/x/pkg/utils"
)

type SLinuxRootFs struct {
	*SGuestRootFsDriver
}

func NewLinuxRootFs(part *SKVMGuestDiskPartition) IRootFsDriver {
	return &SAndroidRootFs{SGuestRootFsDriver: NewGuestRootFsDriver(part).(*SGuestRootFsDriver)}
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
	hf := make(hostman.HostsFile, 0)
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

func (l *SLinuxRootFs) PrepareFsForTemplate() {
	// clean /etc/fstab
	if l.rootFs.Exists("/etc/fstab", false) {
		fstabcont, _ := l.rootFs.FileGetContents("/etc/fstab", false)
		fstab := fstabutils.FSTabFile(string(fstabcont))
		fstab.RemoveDevices(1)
		cf := fstab.ToConf()
		l.rootFs.FilePutContents("/etc/fstab", cf, false, false)
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
		l.rootFs.Cleandir("/var/lib/cloud", false, false)
	}
	cloudDisableFile := "/etc/cloud/cloud-init.disabled"
	if l.rootFs.Exists(cloudDisableFile, false) {
		l.rootFs.Remove(cloudDisableFile, false)
	}
	// clean /tmp /var/log /var/cache /var/spool /var/run
	for _, dir := range []string{"/tmp", "/var/tmp"} {
		if l.rootFs.Exists(dir, false) {
			l.rootFs.Cleandir(dir, false, false)
		}
	}
	for _, dir := range []string{"/var/log", "/var/cache", "/usr/local/var/log", "/usr/local/var/cache"} {
		if l.rootFs.Exists(dir, false) {
			l.rootFs.Zerofiles(dir, false)
		}
	}
	for _, dir := range []string{"/var/spool", "/var/run", "/run", "/usr/local/var/spool", "/usr/local/var/run"} {
		if l.rootFs.Exists(dir, false) {
			l.rootFs.Cleandir(dir, true, true)
		}
	}
}

type SDebianLikeRootFs struct {
	*SLinuxRootFs
}

func NewDebianLikeRootFs(part *SKVMGuestDiskPartition) IRootFsDriver {
	return &SDebianLikeRootFs{SLinuxRootFs: NewLinuxRootFs(part).(*SLinuxRootFs)}
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

type SCentosRootFs struct {
	*SRedhatLikeRootFs
}

func NewCentosRootFs(part *SKVMGuestDiskPartition) IRootFsDriver {
	return &SCentosRootFs{SRedhatLikeRootFs: NewRedhatLikeRootFs(part).(*SRedhatLikeRootFs)}
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
