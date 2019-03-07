package cloudprovider

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/util/ansible"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/onecloud/pkg/util/cloudinit"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

type SDiskInfo struct {
	StorageType string
	SizeGB      int
	Name        string
}

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
	Password            string
	UserData            string

	BillingCycle *billing.SBillingCycle
}

func (vmConfig *SManagedVMCreateConfig) GetConfig(config *jsonutils.JSONDict) error {
	if err := config.Unmarshal(vmConfig, "desc"); err != nil {
		return err
	}
	if publicKey, _ := config.GetString("public_key"); len(publicKey) > 0 {
		vmConfig.PublicKey = publicKey
	}

	adminPublicKey, _ := config.GetString("admin_public_key")
	projectPublicKey, _ := config.GetString("project_public_key")
	oUserData, _ := config.GetString("user_data")

	vmConfig.UserData = generateUserData(adminPublicKey, projectPublicKey, oUserData)

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
		oCloudConfig, _ = cloudinit.ParseUserDataBase64(oUserData)
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

	return cloudConfig.UserDataBase64()
}
