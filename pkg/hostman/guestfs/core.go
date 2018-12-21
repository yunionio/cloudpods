package guestfs

import (
	"fmt"
	"math/rand"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/netutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/sshkeys"
)

type SDeployInfo struct {
	publicKey *sshkeys.SSHKeys
	deploys   []jsonutils.JSONObject
	password  string
	isInit    bool
	enableTty bool
}

func NewDeployInfo(
	publicKey *sshkeys.SSHKeys,
	deploys []jsonutils.JSONObject,
	password string,
	isInit bool,
	enableTty bool,
) *SDeployInfo {
	return &SDeployInfo{
		publicKey: publicKey,
		deploys:   deploys,
		password:  password,
		isInit:    isInit,
		enableTty: enableTty,
	}
}

func DetectRootFs(part IDiskPartition) IRootFsDriver {
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

type sGuestRootFsDriver struct {
	rootFs IDiskPartition
}

func newGuestRootFsDriver(rootFs IDiskPartition) *sGuestRootFsDriver {
	return &sGuestRootFsDriver{
		rootFs: rootFs,
	}
}

func (d *sGuestRootFsDriver) GetPartition() IDiskPartition {
	return d.rootFs
}

func (d *sGuestRootFsDriver) RootExcludeSignatures() []string {
	return []string{}
}

func (d *sGuestRootFsDriver) IsFsCaseInsensitive() bool {
	return false
}

func (d *sGuestRootFsDriver) DeployYunionroot(rootfs IDiskPartition, pubkeys *sshkeys.SSHKeys) error {
	return nil
}

func (d *sGuestRootFsDriver) DeployUdevSubsystemScripts(rootfs IDiskPartition) error {
	return nil
}

func (d *sGuestRootFsDriver) DeployStandbyNetworkingScripts(part IDiskPartition, nics, nicsStandby []jsonutils.JSONObject) error {
	return nil
}

func (d *sGuestRootFsDriver) DeployFstabScripts(_ IDiskPartition, _ []jsonutils.JSONObject) error {
	return nil
}

func (d *sGuestRootFsDriver) EnableSerialConsole(rootfs IDiskPartition, sysInfo *jsonutils.JSONDict) error {
	return nil
}

func (d *sGuestRootFsDriver) DisableSerialConsole(rootfs IDiskPartition) error {
	return nil
}

func (d *sGuestRootFsDriver) CommitChanges(rootfs IDiskPartition) error {
	return nil
}

type SReleaseInfo struct {
	Distro   string
	Version  string
	Arch     string
	Language string
}

func newReleaseInfo(distro, version, arch string) *SReleaseInfo {
	return &SReleaseInfo{
		Distro:  distro,
		Version: version,
		Arch:    arch,
	}
}

func DeployFiles(rootfs IRootFsDriver, deploys []jsonutils.JSONObject) error {
	caseInsensitive := rootfs.IsFsCaseInsensitive()
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
		if !rootfs.GetPartition().Exists(sPath, caseInsensitive) {
			modeRWXOwner := syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IXUSR
			err := rootfs.GetPartition().Mkdir(dirname, modeRWXOwner, caseInsensitive)
			if err != nil {
				return err
			}
		}
		if content, err := deploy.GetString("content"); err != nil {
			err := rootfs.GetPartition().FilePutContents(sPath, content, modAppend, caseInsensitive)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func DeployGuestFs(
	rootfs IRootFsDriver,
	guestDesc *jsonutils.JSONDict,
	deployInfo *SDeployInfo,
) (jsonutils.JSONObject, error) {
	var ret = jsonutils.NewDict()
	var ips = make([]string, 0)
	var err error

	hn, _ := guestDesc.GetString("name")
	domain, _ := guestDesc.GetString("domain")
	gid, _ := guestDesc.GetString("uuid")
	nics, _ := guestDesc.GetArray("nics")

	partition := rootfs.GetPartition()
	releaseInfo := rootfs.GetReleaseInfo(partition)

	for _, n := range nics {
		ip, _ := n.GetString("ip")
		var addr netutils.IPV4Addr
		if addr, err = netutils.NewIPV4Addr(ip); err != nil {
			return nil, fmt.Errorf("Fail to get ip addr from %s: %v", n.String(), err)
		}
		if netutils.IsPrivate(addr) {
			ips = append(ips, ip)
		}
	}
	if releaseInfo != nil {
		ret.Set("distro", jsonutils.NewString(releaseInfo.Distro))
		if len(releaseInfo.Version) > 0 {
			ret.Set("version", jsonutils.NewString(releaseInfo.Version))
		}
		if len(releaseInfo.Arch) > 0 {
			ret.Set("arch", jsonutils.NewString(releaseInfo.Arch))
		}
		if len(releaseInfo.Language) > 0 {
			ret.Set("language", jsonutils.NewString(releaseInfo.Language))
		}
	}
	ret.Set("os", jsonutils.NewString(rootfs.GetOs()))
	if !IsPartitionReadonly(partition) {
		if len(deployInfo.deploys) > 0 {
			if err = DeployFiles(rootfs, deployInfo.deploys); err != nil {
				return nil, fmt.Errorf("DeployFiles: %v", err)
			}
		}
		if err = rootfs.DeployHostname(partition, hn, domain); err != nil {
			return nil, fmt.Errorf("DeployHostname: %v", err)
		}
		if err = rootfs.DeployHosts(partition, hn, domain, ips); err != nil {
			return nil, fmt.Errorf("DeployHosts: %v", err)
		}
		if err = rootfs.DeployNetworkingScripts(partition, nics); err != nil {
			return nil, fmt.Errorf("DeployNetworkingScripts: %v", err)
		}
		if nicsStandby, e := guestDesc.GetArray("nics_standby"); e == nil {
			if err = rootfs.DeployStandbyNetworkingScripts(partition, nics, nicsStandby); err != nil {
				return nil, fmt.Errorf("DeployStandbyNetworkingScripts: %v", err)
			}
		}
		if err = rootfs.DeployUdevSubsystemScripts(partition); err != nil {
			return nil, fmt.Errorf("DeployUdevSubsystemScripts: %v", err)
		}
		if deployInfo.isInit {
			disks, _ := guestDesc.GetArray("disks")
			if err = rootfs.DeployFstabScripts(partition, disks); err != nil {
				return nil, fmt.Errorf("DeployFstabScripts: %v", err)
			}
		}
		if len(deployInfo.password) > 0 {
			if account := rootfs.GetLoginAccount(partition); len(account) > 0 {
				ret.Set("account", jsonutils.NewString(account))
				if err = rootfs.DeployPublicKey(partition, account, deployInfo.publicKey); err != nil {
					return nil, fmt.Errorf("DeployPublicKey: %v", err)
				}
				var secret string
				if secret, err = rootfs.ChangeUserPasswd(partition, account, gid,
					deployInfo.publicKey.PublicKey, deployInfo.password); err != nil {
					return nil, fmt.Errorf("ChangeUserPasswd: %v", err)
				}
				if len(secret) > 0 {
					ret.Set("key", jsonutils.NewString(secret))
				}
			}
		}
		if err = rootfs.DeployYunionroot(partition, deployInfo.publicKey); err != nil {
			return nil, fmt.Errorf("DeployYunionroot: %v", err)
		}
		if partition.SupportSerialPorts() {
			if deployInfo.enableTty {
				if err = rootfs.EnableSerialConsole(partition, ret); err != nil {
					return nil, fmt.Errorf("EnableSerialConsole: %v", err)
				}
			} else {
				if err = rootfs.DisableSerialConsole(partition); err != nil {
					return nil, fmt.Errorf("DisableSerialConsole: %v", err)
				}
			}
			if err = rootfs.CommitChanges(partition); err != nil {
				return nil, fmt.Errorf("CommitChanges: %v", err)
			}
		}
	}
	return ret, nil
}

func IsPartitionReadonly(rootfs IDiskPartition) bool {
	log.Infof("Test if read-only fs ...")
	var filename = fmt.Sprint("./%f", rand.Float32())
	if err := rootfs.FilePutContents(filename, fmt.Sprint("%f", rand.Float32()), false, false); err == nil {
		rootfs.Remove(filename, false)
		return false
	} else {
		log.Errorf("File system is readonly: %s", err)
		return true
	}
}

func DeployAuthorizedKeys(rootFs IDiskPartition, usrDir string, pubkeys *sshkeys.SSHKeys, replace bool) error {
	usrStat := rootFs.Stat(usrDir, false)
	if usrStat != nil {
		sshDir := path.Join(usrDir, ".ssh")
		authFile := path.Join(sshDir, "authorized_keys")
		modeRwxOwner := syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IXUSR
		modeRwOwner := syscall.S_IRUSR | syscall.S_IWUSR
		fStat := usrStat.Sys().(*syscall.Stat_t)
		if !rootFs.Exists(sshDir, false) {
			err := rootFs.Mkdir(sshDir, modeRwxOwner, false)
			if err != nil {
				return err
			}
			err = rootFs.Chown(sshDir, int(fStat.Uid), int(fStat.Gid), false)
			if err != nil {
				return err
			}
		}
		var oldKeys = ""
		if !replace {
			bOldKeys, _ := rootFs.FileGetContents(authFile, false)
			oldKeys = string(bOldKeys)
		}
		newKeys := MergeAuthorizedKeys(oldKeys, pubkeys)
		if err := rootFs.FilePutContents(authFile, newKeys, false, false); err != nil {
			return fmt.Errorf("Put keys to %s: %v", authFile, err)
		}
		log.Infof("after merge keys=====%s, put to: %s", newKeys, authFile)
		if err := rootFs.Chown(authFile, int(fStat.Uid), int(fStat.Gid), false); err != nil {
			return fmt.Errorf("Chown %s to uid: %d, gid: %d: %v", authFile, fStat.Uid, fStat.Gid, err)
		}
		if err := rootFs.Chmod(authFile, uint32(modeRwOwner), false); err != nil {
			return fmt.Errorf("Chmod %s to %d error: %v", authFile, uint32(modeRwOwner), err)
		}
	}
	return nil
}

func MergeAuthorizedKeys(oldKeys string, pubkeys *sshkeys.SSHKeys) string {
	var allkeys = make(map[string]string)
	if len(oldKeys) > 0 {
		for _, line := range strings.Split(oldKeys, "\n") {
			line = strings.TrimSpace(line)
			dat := strings.Split(line, " ")
			if len(dat) > 1 {
				if _, ok := allkeys[dat[1]]; !ok {
					allkeys[dat[1]] = line
				}
			}
		}
	}
	if len(pubkeys.DeletePublicKey) > 0 {
		dat := strings.Split(pubkeys.DeletePublicKey, " ")
		if len(dat) > 1 {
			if _, ok := allkeys[dat[1]]; ok {
				delete(allkeys, dat[1])
			}
		}
	}
	for _, k := range []string{pubkeys.PublicKey, pubkeys.AdminPublicKey, pubkeys.ProjectPublicKey} {
		if len(k) > 0 {
			k = strings.TrimSpace(k)
			dat := strings.Split(k, " ")
			if len(dat) > 1 {
				if _, ok := allkeys[dat[1]]; !ok {
					allkeys[dat[1]] = k
				}
			}
		}
	}
	var keys = make([]string, len(allkeys))
	for key := range allkeys {
		keys = append(keys, key)
	}
	return strings.Join(keys, "\n")
}
