package fsdriver

import (
	"fmt"
	"path"
	"strings"
	"syscall"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/sshkeys"
)

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

func DeployAuthorizedKeys(rootFs IDiskPartition, usrDir string, pubkeys *sshkeys.SSHKeys, replace bool) error {
	usrStat := rootFs.Stat(usrDir, false)
	if usrStat != nil {
		sshDir := path.Join(usrDir, ".ssh")
		authFile := path.Join(sshDir, "authorized_keys")
		modeRwxOwner := syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IXUSR
		modeRwOwner := syscall.S_IRUSR | syscall.S_IWUSR
		fStat, _ := usrStat.Sys().(*syscall.Stat_t)
		if !rootFs.Exists(sshDir, false) {
			err := rootFs.Mkdir(sshDir, modeRwxOwner, false)
			if err != nil {
				log.Errorln(err)
				return err
			}
			err = rootFs.Chown(sshDir, int(fStat.Uid), int(fStat.Gid), false)
			if err != nil {
				log.Errorln(err)
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
