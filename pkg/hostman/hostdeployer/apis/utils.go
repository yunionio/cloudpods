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

package apis

import (
	"encoding/json"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
)

func NewDeployInfo(
	publicKey *SSHKeys,
	deploys []*DeployContent,
	password string,
	isInit bool,
	enableTty bool,
	defaultRootUser bool,
	windowsDefaultAdminUser bool,
	enableCloudInit bool,
	loginAccount string,
	enableTelegraf bool,
	telegrafConf string,
) *DeployInfo {
	depInfo := &DeployInfo{
		PublicKey:               publicKey,
		Deploys:                 deploys,
		Password:                password,
		IsInit:                  isInit,
		EnableTty:               enableTty,
		DefaultRootUser:         defaultRootUser,
		WindowsDefaultAdminUser: windowsDefaultAdminUser,
		EnableCloudInit:         enableCloudInit,
		LoginAccount:            loginAccount,
	}
	if enableTelegraf {
		depInfo.Telegraf = &Telegraf{
			TelegrafConf: telegrafConf,
		}
	}
	return depInfo
}

func JsonDeploysToStructs(jdeploys []jsonutils.JSONObject) []*DeployContent {
	ret := []*DeployContent{}
	for i := 0; i < len(jdeploys); i++ {
		d := new(DeployContent)
		path, err := jdeploys[i].GetString("path")
		if err == nil {
			d.Path = path
		}
		content, err := jdeploys[i].GetString("content")
		if err == nil {
			d.Content = content
		}
		ret = append(ret, d)
	}
	return ret
}

func GetKeys(data jsonutils.JSONObject) *SSHKeys {
	var ret = new(SSHKeys)
	ret.PublicKey, _ = data.GetString("public_key")
	ret.DeletePublicKey, _ = data.GetString("delete_public_key")
	ret.AdminPublicKey, _ = data.GetString("admin_public_key")
	ret.ProjectPublicKey, _ = data.GetString("project_public_key")
	return ret
}

func ConvertRoutes(routes string) []types.SRoute {
	if len(routes) == 0 {
		return nil
	}
	ret := make([]types.SRoute, 0)
	err := json.Unmarshal([]byte(routes), &ret)
	if err != nil {
		log.Errorf("Can't convert %s to types.SRoute", routes)
		return nil
	}
	return ret
}

func GuestdisksDescToDeployDesc(guestDisks []*desc.SGuestDisk) []*Disk {
	if len(guestDisks) == 0 {
		return nil
	}

	disks := make([]*Disk, len(guestDisks))
	for i, disk := range guestDisks {
		disks[i] = new(Disk)
		disks[i].DiskId = disk.DiskId
		disks[i].Driver = disk.Driver
		disks[i].CacheMode = disk.CacheMode
		disks[i].AioMode = disk.AioMode
		disks[i].Size = int64(disk.Size)
		disks[i].TemplateId = disk.TemplateId
		disks[i].ImagePath = disk.ImagePath
		disks[i].StorageId = disk.StorageId
		disks[i].Migrating = disk.Migrating
		disks[i].TargetStorageId = disk.TargetStorageId
		disks[i].Path = disk.Path
		disks[i].Format = disk.Format
		disks[i].Index = int32(disk.Index)
		disks[i].MergeSnapshot = disk.MergeSnapshot
		disks[i].Fs = disk.Fs
		disks[i].Mountpoint = disk.Mountpoint
		disks[i].Dev = disk.Dev
	}
	return disks
}

func GuestnetworksDescToDeployDesc(guestnetworks []*desc.SGuestNetwork) []*Nic {
	if len(guestnetworks) == 0 {
		return nil
	}

	nics := make([]*Nic, len(guestnetworks))
	for i, nic := range guestnetworks {
		nics[i] = new(Nic)
		nics[i].Mac = nic.Mac
		nics[i].Ip = nic.Ip
		nics[i].Net = nic.Net
		nics[i].NetId = nic.NetId
		nics[i].Virtual = nic.Virtual
		nics[i].Gateway = nic.Gateway
		nics[i].Dns = nic.Dns
		nics[i].Domain = nic.Domain
		if nic.Routes != nil {
			nics[i].Routes = nic.Routes.String()
		}
		nics[i].Ifname = nic.Ifname
		nics[i].Masklen = int32(nic.Masklen)
		nics[i].Driver = nic.Driver
		nics[i].Bridge = nic.Bridge
		nics[i].WireId = nic.WireId
		nics[i].Vlan = int32(nic.Vlan)
		nics[i].Interface = nic.Interface
		nics[i].Bw = int32(nic.Bw)
		nics[i].Index = int32(nic.Index)
		nics[i].VirtualIps = nic.VirtualIps
		//nics[i].ExternelId = nic.ExternelId
		nics[i].TeamWith = nic.TeamWith
		if nic.Manual != nil {
			nics[i].Manual = *nic.Manual
		}
		nics[i].NicType = nic.NicType
		nics[i].LinkUp = nic.LinkUp
		nics[i].Mtu = int64(nic.Mtu)
		//nics[i].Name = nic.Name
	}

	return nics
}

func GuestDescToDeployDesc(guestDesc *desc.SGuestDesc) *GuestDesc {
	ret := new(GuestDesc)

	ret.Name = guestDesc.Name
	ret.Domain = guestDesc.Domain
	ret.Uuid = guestDesc.Uuid
	ret.Hostname = guestDesc.Hostname
	ret.Nics = GuestnetworksDescToDeployDesc(guestDesc.Nics)
	ret.Disks = GuestdisksDescToDeployDesc(guestDesc.Disks)
	ret.NicsStandby = GuestnetworksDescToDeployDesc(guestDesc.NicsStandby)

	return ret
}

func NewReleaseInfo(distro, version, arch string) *ReleaseInfo {
	return &ReleaseInfo{
		Distro:  distro,
		Version: version,
		Arch:    arch,
	}
}
