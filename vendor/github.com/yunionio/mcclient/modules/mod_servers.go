package modules

import (
	"fmt"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/mcclient"
	"github.com/yunionio/pkg/utils"
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
	login_key, e := data.GetString("login_key")
	if e != nil {
		return nil, fmt.Errorf("No login key: %s", e)
	} else {
		passwd, e := utils.DescryptAESBase64(id, login_key)
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
			"IPs", "Disk", "Status",
			"vcpu_count", "vmem_size",
			"ext_bw", "Zone_name",
			"Secgroup", "Secgrp_id",
			"vrouter", "vrouter_id",
			"Created_at", "Group_name",
			"Group_id", "Hypervisor", "os_type"},
		[]string{"Host", "Tenant", "is_system", "auto_delete_at"})}

	registerCompute(&Servers)
}
