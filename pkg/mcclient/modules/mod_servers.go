package modules

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/seclib2"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/utils"
)

type ServerManager struct {
	ResourceManager
}

func (this *ServerManager) GetLoginInfo(s *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	data, e := this.GetMetadata(s, id, nil)
	if e != nil {
		return nil, e
	}
	ret := jsonutils.NewDict()
	loginKey, e := data.GetString("login_key")
	if e != nil {
		return nil, fmt.Errorf("No login key: %s", e)
	}

	var privateKey string
	if params != nil && !gotypes.IsNil(params) {
		privateKey, _ = params.GetString("private_key")
	}

	var passwd string
	if len(privateKey) > 0 {
		passwd, e = seclib2.DecryptBase64(privateKey, loginKey)
	} else {
		passwd, e = utils.DescryptAESBase64(id, loginKey)
	}
	if e != nil {
		return nil, e
	}
	ret.Add(jsonutils.NewString(passwd), "password")
	v, e := data.Get("login_account")
	if e == nil {
		ret.Add(v, "username")
	}
	v, e = data.Get("login_key_timestamp")
	if e == nil {
		ret.Add(v, "updated")
	}

	return ret, nil
}

func (this *ServerManager) DoChangeSetting(s *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return this.Update(s, id, params)
}

var (
	Servers ServerManager
)

func init() {
	Servers = ServerManager{NewComputeManager("server", "servers",
		[]string{"ID", "Name", "Billing_type",
			"IPs", "EIP", "Disk", "Status",
			"vcpu_count", "vmem_size",
			"ext_bw", "Zone_name",
			"Secgroup", "Secgrp_id",
			"vrouter", "vrouter_id",
			"Created_at", "Group_name",
			"Group_id", "Hypervisor", "os_type",
			"expired_at"},
		[]string{"Host", "Tenant", "is_system", "auto_delete_at"})}

	registerCompute(&Servers)
}
