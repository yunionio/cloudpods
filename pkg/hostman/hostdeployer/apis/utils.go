package apis

import (
	"yunion.io/x/jsonutils"
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
) *DeployInfo {
	return &DeployInfo{
		PublicKey:               publicKey,
		Deploys:                 deploys,
		Password:                password,
		IsInit:                  isInit,
		EnableTty:               enableTty,
		DefaultRootUser:         defaultRootUser,
		WindowsDefaultAdminUser: windowsDefaultAdminUser,
		EnableCloudInit:         enableCloudInit,
	}
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

func GuestDescToDeployDesc(guestDesc *jsonutils.JSONDict) (*GuestDesc, error) {
	ret := new(GuestDesc)
	ret.Name, _ = guestDesc.GetString("name")
	ret.Domain, _ = guestDesc.GetString("domain")
	ret.Uuid, _ = guestDesc.GetString("uuid")
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
		err := jnicsStandby.Unmarshal(nicsStandby)
		if err != nil {
			return nil, err
		}
		ret.NicsStandby = nicsStandby
	}

	return ret, nil
}
