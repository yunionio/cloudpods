package guestfs

import (
	"fmt"
	"path/filepath"
	"syscall"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/sshkeys"
	"yunion.io/x/pkg/util/netutils"
)

type SDeployInfo struct {
	publicKey *sshkeys.SSHKeys
	deploys   []jsonutils.JSONObject
	password  string
	isInit    bool
	enableTty bool
}

type newRootFsDriverFunc func(part *SKVMGuestDiskPartition) IRootFsDriver

var rootfsDrivers = make([]newRootFsDriverFunc, 0)

func DetectRootFs(part *SKVMGuestDiskPartition) IRootFsDriver {
	for _, newDriverFunc := range rootfsDrivers {
		d := newDriverFunc(part)
		if testRootfs(d) {
			return d
		}
	}
	return nil
}

func testRootfs(d IRootFsDriver) bool {
	caseInsensitive := d.IsFsCaseInsensitive()
	for _, rd := range d.RootSignatures() {
		if !d.GetPartition().Exists(rd, caseInsensitive) {
			log.Infof("[%s] test root fs: %s not exists", d, rd)
			return false
		}
	}
	for _, rd := range d.RootExcludeSignatures() {
		if d.GetPartition().Exists(rd, caseInsensitive) {
			log.Infof("[%s] test root fs: %s exists, test failed", d, rd)
			return false
		}
	}
	return true
}

type IRootFsDriver interface {
	GetPartition() *SKVMGuestDiskPartition
	String() string

	IsFsCaseInsensitive() bool
	RootSignatures() []string
	RootExcludeSignatures() []string
	GetReleaseInfo() []string
	GetOs() string
	DeployFiles([]jsonutils.JSONObject) error
	DeployHostname(hn, domain string) error
	DeployHost(hn, domain string, ips []string) error
	DeployNetworkingScripts([]jsonutils.JSONObject) error
	DeployStandbyNetworkingScripts(nics, nicsStandby []jsonutils.JSONObject) error
	DeployUdevSubsystemScripts() error
	DeployFstabScripts([]jsonutils.JSONObject) error
	GetLoginAccount() string
	DeployPublicKey(string, *sshkeys.SSHKeys) error
	ChangeUserPasswd(account, gid, publicKey, password string) string
	DeployYunionroot(*sshkeys.SSHKeys) error
	EnableSerialConsole(*jsonutils.JSONDict) error
	DisableSerialConsole() error
	CommitChanges() error

	DeployGuestFs(IRootFsDriver, *jsonutils.JSONDict, *SDeployInfo) (jsonutils.JSONObject, error)

	// PrepareFsForTemplate() error
}

type SGuestRootFsDriver struct {
	rootFs *SKVMGuestDiskPartition
}

func NewGuestRootFsDriver(rootFs *SKVMGuestDiskPartition) IRootFsDriver {
	return &SGuestRootFsDriver{rootFs}
}

func (d *SGuestRootFsDriver) GetPartition() *SKVMGuestDiskPartition {
	return d.rootFs
}

func (d *SGuestRootFsDriver) String() string {
	return ""
}

func (d *SGuestRootFsDriver) IsFsCaseInsensitive() bool {
	return false
}

func (d *SGuestRootFsDriver) RootSignatures() []string {
	return nil
}

func (d *SGuestRootFsDriver) RootExcludeSignatures() []string {
	return nil
}

func (d *SGuestRootFsDriver) GetReleaseInfo() []string {
	return nil
}

func (d *SGuestRootFsDriver) GetOs() string {
	return ""
}

func (d *SGuestRootFsDriver) DeployFiles(deploys []jsonutils.JSONObject) error {
	caseInsensitive := d.IsFsCaseInsensitive()
	for _, deploy := range deploys {
		var modAppend = false
		if action, _ := deploy.GetString("action"); action == "append" {
			modAppend = true
		}
		sPath, err := deploy.GetString("path")
		if err != nil {
			return err
		}
		dirname := filepath.Dir(sPath)
		if !d.rootFs.Exists(sPath, caseInsensitive) {
			modeRWXOwner := syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IXUSR
			err := d.rootFs.Mkdir(dirname, modeRWXOwner, caseInsensitive)
			if err != nil {
				return err
			}
		}
		if content, err := deploy.GetString("content"); err != nil {
			err := d.rootFs.FilePutContents(sPath, content, modAppend, caseInsensitive)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (d *SGuestRootFsDriver) DeployHostname(hn, domain string) error {
	return fmt.Errorf("Not Implemented")
}

func (d *SGuestRootFsDriver) DeployHost(hn, domain string, ips []string) error {
	return fmt.Errorf("Not Implemented")
}

func (d *SGuestRootFsDriver) DeployNetworkingScripts([]jsonutils.JSONObject) error {
	return fmt.Errorf("Not Implemented")
}

func (d *SGuestRootFsDriver) DeployStandbyNetworkingScripts(nics, nicsStandby []jsonutils.JSONObject) error {
	return fmt.Errorf("Not Implemented")
}

func (d *SGuestRootFsDriver) DeployUdevSubsystemScripts() error {
	return fmt.Errorf("Not Implemented")
}

func (d *SGuestRootFsDriver) DeployFstabScripts([]jsonutils.JSONObject) error {
	return fmt.Errorf("Not Implemented")
}

func (d *SGuestRootFsDriver) GetLoginAccount() string {
	return ""
}

func (d *SGuestRootFsDriver) DeployPublicKey(string, *sshkeys.SSHKeys) error {
	return fmt.Errorf("Not Implemented")
}

func (d *SGuestRootFsDriver) ChangeUserPasswd(account, gid, publicKey, password string) string {
	return ""
}

func (d *SGuestRootFsDriver) DeployYunionroot(*sshkeys.SSHKeys) error {
	return fmt.Errorf("Not Implemented")
}

func (d *SGuestRootFsDriver) EnableSerialConsole(*jsonutils.JSONDict) error {
	return fmt.Errorf("Not Implemented")
}

func (d *SGuestRootFsDriver) DisableSerialConsole() error {
	return fmt.Errorf("Not Implemented")
}

func (d *SGuestRootFsDriver) CommitChanges() error {
	return fmt.Errorf("Not Implemented")
}

func (d *SGuestRootFsDriver) DeployGuestFs(rootfs IRootFsDriver, guestDesc *jsonutils.JSONDict,
	deployInfo *SDeployInfo) (jsonutils.JSONObject, error) {
	var ret = jsonutils.NewDict()
	var ips = make([]string, 0)
	var releaseInfo = rootfs.GetReleaseInfo()
	hn, _ := guestDesc.GetString("name")
	domain, _ := guestDesc.GetString("domain")
	gid, _ := guestDesc.GetString("uuid")
	nics, _ := guestDesc.GetArray("nisc")

	var err error
	for _, n := range nics {
		ip, _ := n.GetString("ip")
		var addr netutils.IPV4Addr
		if addr, err = netutils.NewIPV4Addr(ip); err != nil {
			log.Errorln(err)
			return nil, err
		}
		if netutils.IsPrivate(addr) {
			ips = append(ips, ip)
		}
	}
	if releaseInfo != nil {
		ret.Set("distro", jsonutils.NewString(releaseInfo[0]))
		if len(releaseInfo) > 1 && len(releaseInfo[1]) > 0 {
			ret.Set("version", jsonutils.NewString(releaseInfo[1]))
		}
		if len(releaseInfo) > 2 && len(releaseInfo[2]) > 0 {
			ret.Set("arch", jsonutils.NewString(releaseInfo[2]))
		}
		if len(releaseInfo) > 3 && len(releaseInfo[3]) > 0 {
			ret.Set("language", jsonutils.NewString(releaseInfo[3]))
		}
	}
	ret.Set("os", jsonutils.NewString(rootfs.GetOs()))
	if d.rootFs.GetReadonly() {
		if len(deployInfo.deploys) > 0 {
			if err = rootfs.DeployFiles(deployInfo.deploys); err != nil {
				log.Errorln(err)
				return nil, err
			}
		}
		if err = rootfs.DeployHostname(hn, domain); err != nil {
			log.Errorln(err)
			return nil, err
		}
		if err = rootfs.DeployHost(hn, domain, ips); err != nil {
			log.Errorln(err)
			return nil, err
		}
		if err = rootfs.DeployNetworkingScripts(nics); err != nil {
			log.Errorln(err)
			return nil, err
		}
		if nicsStandby, e := guestDesc.GetArray("nics_standby"); e == nil {
			rootfs.DeployStandbyNetworkingScripts(nics, nicsStandby)
		}
		if err = rootfs.DeployUdevSubsystemScripts(); err != nil {
			log.Errorln(err)
			return nil, err
		}
		if deployInfo.isInit {
			disks, _ := guestDesc.GetArray("disks")
			if err = rootfs.DeployFstabScripts(disks); err != nil {
				log.Errorln(err)
				return nil, err
			}
		}
		if len(deployInfo.password) > 0 {
			if account := rootfs.GetLoginAccount(); len(account) > 0 {
				if err = rootfs.DeployPublicKey(account, deployInfo.publicKey); err != nil {
					log.Errorln(err)
					return nil, err
				}
				secret := rootfs.ChangeUserPasswd(account, gid,
					deployInfo.publicKey.PublicKey, deployInfo.password)
				if len(secret) > 0 {
					ret.Set("key", jsonutils.NewString(secret))
				}
			}
		}
		if err = rootfs.DeployYunionroot(deployInfo.publicKey); err != nil {
			log.Errorln(err)
			return nil, err
		}
		if deployInfo.enableTty {
			if err = rootfs.EnableSerialConsole(ret); err != nil {
				log.Errorln(err)
				return nil, err
			}
		} else {
			if err = rootfs.DisableSerialConsole(); err != nil {
				log.Errorln(err)
				return nil, err
			}
		}
		if err = rootfs.CommitChanges(); err != nil {
			log.Errorln(err)
			return nil, err
		}
	}
	return ret, nil
}
