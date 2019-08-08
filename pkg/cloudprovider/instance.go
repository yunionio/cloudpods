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

package cloudprovider

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/osprofile"

	"yunion.io/x/onecloud/pkg/util/ansible"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/onecloud/pkg/util/cloudinit"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

type SDiskInfo struct {
	StorageExternalId string
	StorageType       string
	SizeGB            int
	Name              string
}

const (
	CLOUD_SHELL  = "cloud-shell"
	CLOUD_CONFIG = "cloud-config"
)

type SManagedVMCreateConfig struct {
	Name                string
	ExternalImageId     string
	ImageType           string
	OsType              string
	OsDistribution      string
	OsVersion           string
	InstanceType        string // InstanceType 不为空时，直接采用InstanceType创建机器。
	Cpu                 int
	MemoryMB            int
	ExternalNetworkId   string
	IpAddr              string
	Description         string
	SysDisk             SDiskInfo
	DataDisks           []SDiskInfo
	PublicKey           string
	ExternalSecgroupId  string
	ExternalSecgroupIds []string
	Account             string
	Password            string
	UserData            string

	BillingCycle *billing.SBillingCycle
}

type SManagedVMChangeConfig struct {
	Cpu          int
	MemoryMB     int
	InstanceType string
}

func (vmConfig *SManagedVMCreateConfig) GetConfig(config *jsonutils.JSONDict) error {
	if err := config.Unmarshal(vmConfig, "desc"); err != nil {
		return err
	}
	if publicKey, _ := config.GetString("public_key"); len(publicKey) > 0 {
		vmConfig.PublicKey = publicKey
	}
	//目前所写的userData格式仅支持Linux
	if vmConfig.OsType == osprofile.OS_TYPE_LINUX {
		adminPublicKey, _ := config.GetString("admin_public_key")
		projectPublicKey, _ := config.GetString("project_public_key")
		oUserData, _ := config.GetString("user_data")

		vmConfig.UserData = generateUserData(adminPublicKey, projectPublicKey, oUserData)
	}

	resetPassword := jsonutils.QueryBoolean(config, "reset_password", false)
	vmConfig.Password, _ = config.GetString("password")
	if resetPassword && len(vmConfig.Password) == 0 {
		vmConfig.Password = seclib2.RandomPassword2(12)
	}
	return nil
}

func generateUserData(adminPublicKey, projectPublicKey, oUserData string) string {
	var oCloudConfig *cloudinit.SCloudConfig

	if len(oUserData) > 0 {
		oCloudConfig, _ = cloudinit.ParseUserData(oUserData)
	}

	ansibleUser := cloudinit.NewUser(ansible.PUBLIC_CLOUD_ANSIBLE_USER)
	ansibleUser.SshKey(adminPublicKey).SshKey(projectPublicKey).SudoPolicy(cloudinit.USER_SUDO_NOPASSWD)

	cloudConfig := cloudinit.SCloudConfig{
		DisableRoot: 0,
		SshPwauth:   1,

		Users: []cloudinit.SUser{
			ansibleUser,
		},
	}

	if oCloudConfig != nil {
		cloudConfig.Merge(oCloudConfig)
	}

	return cloudConfig.UserData()
}

func (vmConfig *SManagedVMCreateConfig) InjectPasswordByCloudInit() error {
	if vmConfig.OsType != osprofile.OS_TYPE_LINUX {
		return fmt.Errorf("Only support inject Linux password, current osType is %s", vmConfig.OsType)
	}
	loginUser := cloudinit.NewUser(vmConfig.Account)
	loginUser.SudoPolicy(cloudinit.USER_SUDO_NOPASSWD)
	if len(vmConfig.PublicKey) > 0 {
		loginUser.SshKey(vmConfig.PublicKey)
	}
	if len(vmConfig.Password) > 0 {
		loginUser.Password(vmConfig.Password)
	}

	cloudconfig := cloudinit.SCloudConfig{
		DisableRoot: 0,
		SshPwauth:   1,
		Users: []cloudinit.SUser{
			loginUser,
		},
	}

	if len(vmConfig.UserData) > 0 {
		oCloudConfig, err := cloudinit.ParseUserData(vmConfig.UserData)
		if err != nil {
			return err
		}
		cloudconfig.Merge(oCloudConfig)
	}
	vmConfig.UserData = cloudconfig.UserData()
	return nil
}
