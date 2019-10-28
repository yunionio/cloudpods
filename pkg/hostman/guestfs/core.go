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

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/netutils"

	fsdriver "yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
)

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

func DoDeployGuestFs(rootfs fsdriver.IRootFsDriver, guestDesc *deployapi.GuestDesc, deployInfo *deployapi.DeployInfo,
) (*deployapi.DeployGuestFsResponse, error) {
	var (
		err         error
		ret         = new(deployapi.DeployGuestFsResponse)
		ips         = make([]string, 0)
		hn          = guestDesc.Name
		domain      = guestDesc.Domain
		gid         = guestDesc.Uuid
		nics        = fsdriver.ToServerNics(guestDesc.Nics)
		nicsStandby = fsdriver.ToServerNics(guestDesc.NicsStandby)
		partition   = rootfs.GetPartition()
		releaseInfo = rootfs.GetReleaseInfo(partition)
	)
	for _, n := range nics {
		var addr netutils.IPV4Addr
		if addr, err = netutils.NewIPV4Addr(n.Ip); err != nil {
			return nil, fmt.Errorf("Fail to get ip addr from %#v: %s", n, err)
		}
		if netutils.IsPrivate(addr) {
			ips = append(ips, n.Ip)
		}
	}
	if releaseInfo != nil {
		ret.Distro = releaseInfo.Distro
		if len(releaseInfo.Version) > 0 {
			ret.Version = releaseInfo.Version
		}
		if len(releaseInfo.Arch) > 0 {
			ret.Arch = releaseInfo.Arch
		}
		if len(releaseInfo.Language) > 0 {
			ret.Language = releaseInfo.Language
		}
	}
	ret.Os = rootfs.GetOs()

	if IsPartitionReadonly(partition) {
		return ret, nil
	}

	if len(deployInfo.Deploys) > 0 {
		if err = rootfs.DeployFiles(deployInfo.Deploys); err != nil {
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
	if len(nicsStandby) > 0 {
		if err = rootfs.DeployStandbyNetworkingScripts(partition, nics, nicsStandby); err != nil {
			return nil, fmt.Errorf("DeployStandbyNetworkingScripts: %v", err)
		}
	}
	if err = rootfs.DeployUdevSubsystemScripts(partition); err != nil {
		return nil, fmt.Errorf("DeployUdevSubsystemScripts: %v", err)
	}
	if deployInfo.IsInit {
		if err = rootfs.DeployFstabScripts(partition, guestDesc.Disks); err != nil {
			return nil, fmt.Errorf("DeployFstabScripts: %v", err)
		}
	}

	if len(deployInfo.Password) > 0 {
		if account := rootfs.GetLoginAccount(partition,
			deployInfo.DefaultRootUser, deployInfo.WindowsDefaultAdminUser); len(account) > 0 {
			if err = rootfs.DeployPublicKey(partition, account, deployInfo.PublicKey); err != nil {
				return nil, fmt.Errorf("DeployPublicKey: %v", err)
			}
			var secret string
			if secret, err = rootfs.ChangeUserPasswd(partition, account, gid,
				deployInfo.PublicKey.PublicKey, deployInfo.Password); err != nil {
				return nil, fmt.Errorf("ChangeUserPasswd: %v", err)
			}
			if len(secret) > 0 {
				ret.Key = secret
			}
			ret.Account = account
		}
	}

	if err = rootfs.DeployYunionroot(partition, deployInfo.PublicKey, deployInfo.IsInit, deployInfo.EnableCloudInit); err != nil {
		return nil, fmt.Errorf("DeployYunionroot: %v", err)
	}
	if partition.SupportSerialPorts() {
		if deployInfo.EnableTty {
			if err = rootfs.EnableSerialConsole(partition, nil); err != nil {
				return nil, fmt.Errorf("EnableSerialConsole: %v", err)
			}
		} else {
			if err = rootfs.DisableSerialConsole(partition); err != nil {
				return nil, fmt.Errorf("DisableSerialConsole: %v", err)
			}
		}
	}
	if err = rootfs.CommitChanges(partition); err != nil {
		return nil, fmt.Errorf("CommitChanges: %v", err)
	}

	log.Infof("Deploy finished, return: %s", ret.String())
	return ret, nil
}

func DeployGuestFs(
	rootfs fsdriver.IRootFsDriver,
	guestDesc *jsonutils.JSONDict,
	deployInfo *deployapi.DeployInfo,
) (jsonutils.JSONObject, error) {
	desc, err := deployapi.GuestDescToDeployDesc(guestDesc)
	if err != nil {
		return nil, errors.Wrap(err, "To deploy desc fail")
	}
	ret, err := DoDeployGuestFs(rootfs, desc, deployInfo)
	if err != nil {
		return nil, err
	}
	return jsonutils.Marshal(ret), nil
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
