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
	"path/filepath"
	"strings"
	"syscall"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
)

type sGuestRootFsDriver struct {
	rootFs IDiskPartition
}

func newGuestRootFsDriver(rootFs IDiskPartition) *sGuestRootFsDriver {
	return &sGuestRootFsDriver{
		rootFs: rootFs,
	}
}

func (d *sGuestRootFsDriver) DeployFiles(deploys []*deployapi.DeployContent) error {
	caseInsensitive := d.IsFsCaseInsensitive()
	for _, deploy := range deploys {
		var modAppend = false
		if deploy.Action == "append" {
			modAppend = true
		}
		if len(deploy.Path) == 0 {
			return fmt.Errorf("Deploy file missing param path")
		}
		dirname := filepath.Dir(deploy.Path)
		if !d.GetPartition().Exists(dirname, caseInsensitive) {
			modeRWXOwner := syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IXUSR
			err := d.GetPartition().Mkdir(dirname, modeRWXOwner, caseInsensitive)
			if err != nil {
				log.Errorln(err)
				return err
			}
		}
		if len(deploy.Content) > 0 {
			err := d.GetPartition().FilePutContents(deploy.Path, deploy.Content, modAppend, caseInsensitive)
			if err != nil {
				log.Errorln(err)
				return err
			}
		}
	}
	return nil
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

func (d *sGuestRootFsDriver) DeployYunionroot(rootfs IDiskPartition, pubkeys *deployapi.SSHKeys, isInit, enableCloudInit bool) error {
	return nil
}

func (d *sGuestRootFsDriver) DeployUdevSubsystemScripts(rootfs IDiskPartition) error {
	return nil
}

func (d *sGuestRootFsDriver) DeployStandbyNetworkingScripts(part IDiskPartition, nics, nicsStandby []*types.SServerNic) error {
	return nil
}

func (d *sGuestRootFsDriver) DeployFstabScripts(_ IDiskPartition, _ []*deployapi.Disk) error {
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

func (d *sGuestRootFsDriver) DetectIsUEFISupport(IDiskPartition) bool {
	return false
}

func (l *sGuestRootFsDriver) IsCloudinitInstall() bool {
	return false
}

func (l *sGuestRootFsDriver) IsResizeFsPartitionSupport() bool {
	return true
}

func DeployAuthorizedKeys(rootFs IDiskPartition, usrDir string, pubkeys *deployapi.SSHKeys, replace bool) error {
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
		if err := rootFs.Chown(authFile, int(fStat.Uid), int(fStat.Gid), false); err != nil {
			return fmt.Errorf("Chown %s to uid: %d, gid: %d: %v", authFile, fStat.Uid, fStat.Gid, err)
		}
		if err := rootFs.Chmod(authFile, uint32(modeRwOwner), false); err != nil {
			return fmt.Errorf("Chmod %s to %d error: %v", authFile, uint32(modeRwOwner), err)
		}
	}
	return nil
}

func MergeAuthorizedKeys(oldKeys string, pubkeys *deployapi.SSHKeys) string {
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
	var keys = make([]string, 0)
	for _, val := range allkeys {
		keys = append(keys, val)
	}
	return strings.Join(keys, "\n") + "\n"
}
