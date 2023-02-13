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

func GuestDescToDeployDesc(guestDesc *jsonutils.JSONDict) (*GuestDesc, error) {
	ret := new(GuestDesc)
	ret.Name, _ = guestDesc.GetString("name")
	ret.Domain, _ = guestDesc.GetString("domain")
	ret.Uuid, _ = guestDesc.GetString("uuid")
	ret.Hostname, _ = guestDesc.GetString("hostname")
	jnics, _ := guestDesc.Get("nics")
	jdisks, _ := guestDesc.Get("disks")
	jnicsStandby, _ := guestDesc.Get("nics_standby")

	if jnics != nil {
		nics := make([]*Nic, 0)
		err := jnics.Unmarshal(&nics)
		if err != nil {
			return nil, err
		}
		ret.Nics = nics
	}

	if jdisks != nil {
		disks := make([]*Disk, 0)
		err := jdisks.Unmarshal(&disks)
		if err != nil {
			return nil, err
		}
		ret.Disks = disks
	}

	if jnicsStandby != nil {
		nicsStandby := make([]*Nic, 0)
		err := jnicsStandby.Unmarshal(&nicsStandby)
		if err != nil {
			return nil, err
		}
		ret.NicsStandby = nicsStandby
	}

	return ret, nil
}

func NewReleaseInfo(distro, version, arch string) *ReleaseInfo {
	return &ReleaseInfo{
		Distro:  distro,
		Version: version,
		Arch:    arch,
	}
}
