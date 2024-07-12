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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"

	comapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
)

func DetectRootFs(part fsdriver.IDiskPartition) (fsdriver.IRootFsDriver, error) {
	for _, newDriverFunc := range fsdriver.GetRootfsDrivers() {
		d := newDriverFunc(part)
		d.SetVirtualObject(d)
		if testRootfs(d) {
			return d, nil
		}
	}
	return nil, fmt.Errorf("DetectRootFs with partition %s no root fs found", part.GetPartDev())
}

func testRootfs(d fsdriver.IRootFsDriver) bool {
	caseInsensitive := d.IsFsCaseInsensitive()
	for _, rd := range d.RootSignatures() {
		if !d.GetPartition().Exists(rd, caseInsensitive) {
			log.Debugf("[%s] test root fs: %s not exists", d, rd)
			return false
		}
	}
	for _, rd := range d.RootExcludeSignatures() {
		if d.GetPartition().Exists(rd, caseInsensitive) {
			log.Debugf("[%s] test root fs: %s exists, test failed", d, rd)
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
	if len(guestDesc.Hostname) > 0 {
		hn = guestDesc.Hostname
	}
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

	if deployInfo.IsInit {
		if err := rootfs.CleanNetworkScripts(partition); err != nil {
			return nil, errors.Wrap(err, "Clean network scripts")
		}
		if len(deployInfo.Deploys) > 0 {
			if err := rootfs.DeployFiles(deployInfo.Deploys); err != nil {
				return nil, errors.Wrap(err, "DeployFiles")
			}
		}
		if len(deployInfo.UserData) > 0 {
			if err := rootfs.DeployUserData(deployInfo.UserData); err != nil {
				return nil, errors.Wrap(err, "DeployUserData")
			}
		}
	}

	if deployInfo.Telegraf != nil {
		if deployed, err := rootfs.DeployTelegraf(deployInfo.Telegraf.TelegrafConf); err != nil {
			return nil, errors.Wrap(err, "deploy telegraf")
		} else {
			ret.TelegrafDeployed = deployed
		}
	}

	if err := rootfs.DeployHostname(partition, hn, domain); err != nil {
		return nil, errors.Wrap(err, "DeployHostname")
	}
	if err := rootfs.DeployHosts(partition, hn, domain, ips); err != nil {
		return nil, errors.Wrap(err, "DeployHosts")
	}

	if guestDesc.Hypervisor == comapi.HYPERVISOR_KVM {
		if err := rootfs.DeployQgaService(partition); err != nil {
			return nil, errors.Wrap(err, "DeployQgaService")
		}
		if err := rootfs.DeployQgaBlackList(partition); err != nil {
			return nil, errors.Wrap(err, "DeployQgaBlackList")
		}
	}

	if err := rootfs.DeployNetworkingScripts(partition, nics); err != nil {
		return nil, errors.Wrap(err, "DeployNetworkingScripts")
	}
	if len(nicsStandby) > 0 {
		if err := rootfs.DeployStandbyNetworkingScripts(partition, nics, nicsStandby); err != nil {
			return nil, errors.Wrap(err, "DeployStandbyNetworkingScripts")
		}
	}
	if err := rootfs.DeployUdevSubsystemScripts(partition); err != nil {
		return nil, errors.Wrap(err, "DeployUdevSubsystemScripts")
	}
	if deployInfo.IsInit {
		if err := rootfs.DeployFstabScripts(partition, guestDesc.Disks); err != nil {
			return nil, errors.Wrap(err, "DeployFstabScripts")
		}
	}

	if len(deployInfo.Password) > 0 {
		account, err := rootfs.GetLoginAccount(partition, deployInfo.LoginAccount,
			deployInfo.DefaultRootUser, deployInfo.WindowsDefaultAdminUser)
		if err != nil {
			return nil, errors.Wrap(err, "get login account")
		}
		if len(account) > 0 {
			if err = rootfs.DeployPublicKey(partition, account, deployInfo.PublicKey); err != nil {
				return nil, errors.Wrap(err, "DeployPublicKey")
			}
			var secret string
			if secret, err = rootfs.ChangeUserPasswd(partition, account, gid,
				deployInfo.PublicKey.PublicKey, deployInfo.Password); err != nil {
				return nil, errors.Wrap(err, "ChangeUserPasswd")
			}
			if len(secret) > 0 {
				ret.Key = secret
			}
			ret.Account = account
		}
	}

	if err = rootfs.DeployYunionroot(partition, deployInfo.PublicKey, deployInfo.IsInit, deployInfo.EnableCloudInit); err != nil {
		return nil, errors.Wrap(err, "DeployYunionroot")
	}
	if partition.SupportSerialPorts() {
		if deployInfo.EnableTty {
			if err = rootfs.EnableSerialConsole(partition, nil); err != nil {
				// return nil, fmt.Errorf("EnableSerialConsole: %v", err)
				log.Warningf("EnableSerialConsole error: %v", err)
			}
		} else {
			if err = rootfs.DisableSerialConsole(partition); err != nil {
				// return nil, fmt.Errorf("DisableSerialConsole: %v", err)
				log.Warningf("DisableSerialConsole error: %v", err)
			}
		}
	}
	if err = rootfs.CommitChanges(partition); err != nil {
		return nil, errors.Wrap(err, "CommitChanges")
	}

	log.Infof("Deploy finished, return: %s", ret.String())
	return ret, nil
}

func DeployGuestFs(
	rootfs fsdriver.IRootFsDriver,
	desc *deployapi.GuestDesc,
	deployInfo *deployapi.DeployInfo,
) (jsonutils.JSONObject, error) {
	ret, err := DoDeployGuestFs(rootfs, desc, deployInfo)
	if err != nil {
		return nil, err
	}
	return jsonutils.Marshal(ret), nil
}

func IsPartitionReadonly(rootfs fsdriver.IDiskPartition) bool {
	var filename = fmt.Sprintf("/.%f", rand.Float32())
	if err := rootfs.FilePutContents(filename, fmt.Sprintf("%f", rand.Float32()), false, false); err == nil {
		rootfs.Remove(filename, false)
		log.Infof("File system %s is not readonly", rootfs.GetMountPath())
		return false
	} else {
		log.Errorf("File system %s is readonly: %s", rootfs.GetMountPath(), err)
		return true
	}
}
