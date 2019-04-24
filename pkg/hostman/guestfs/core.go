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

package guestfs

import (
	"fmt"
	"math/rand"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/netutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/sshkeys"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
)

type SDeployInfo struct {
	publicKey               *sshkeys.SSHKeys
	deploys                 []jsonutils.JSONObject
	password                string
	isInit                  bool
	enableTty               bool
	defaultRootUser         bool
	windowsDefaultAdminUser bool
}

func NewDeployInfo(
	publicKey *sshkeys.SSHKeys,
	deploys []jsonutils.JSONObject,
	password string,
	isInit bool,
	enableTty bool,
	defaultRootUser bool,
	windowsDefaultAdminUser bool,
) *SDeployInfo {
	return &SDeployInfo{
		publicKey:               publicKey,
		deploys:                 deploys,
		password:                password,
		isInit:                  isInit,
		enableTty:               enableTty,
		defaultRootUser:         defaultRootUser,
		windowsDefaultAdminUser: windowsDefaultAdminUser,
	}
}

func (d *SDeployInfo) String() string {
	return fmt.Sprintf("deplys: %s, password %s, isInit: %v, enableTty: %v, defaultRootUser: %v",
		d.deploys, d.password, d.isInit, d.enableTty, d.defaultRootUser)
}

func DetectRootFs(part fsdriver.IDiskPartition) fsdriver.IRootFsDriver {
	for _, newDriverFunc := range fsdriver.GetRootfsDrivers() {
		d := newDriverFunc(part)
		if testRootfs(d) {
			return d
		}
	}
	return nil
}

func testRootfs(d fsdriver.IRootFsDriver) bool {
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

func DeployGuestFs(
	rootfs fsdriver.IRootFsDriver,
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

	if IsPartitionReadonly(partition) {
		return ret, nil
	}

	if len(deployInfo.deploys) > 0 {
		if err = rootfs.DeployFiles(deployInfo.deploys); err != nil {
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
		if account := rootfs.GetLoginAccount(partition,
			deployInfo.defaultRootUser, deployInfo.windowsDefaultAdminUser); len(account) > 0 {
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

	log.Debugf("Deploy finished, return: %s", ret.String())
	return ret, nil
}

func IsPartitionReadonly(rootfs fsdriver.IDiskPartition) bool {
	log.Infof("Test if read-only fs ...")
	var filename = fmt.Sprintf("/.%f", rand.Float32())
	if err := rootfs.FilePutContents(filename, fmt.Sprintf("%f", rand.Float32()), false, false); err == nil {
		rootfs.Remove(filename, false)
		return false
	} else {
		log.Errorf("File system is readonly: %s", err)
		return true
	}
}
