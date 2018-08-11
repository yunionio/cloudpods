package modules

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/utils"
)

type HostManager struct {
	ResourceManager
}

func (this *HostManager) GetLoginInfo(s *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	data, e := this.GetMetadata(s, id, nil)
	if e != nil {
		return nil, e
	}
	ret := jsonutils.NewDict()
	login_key, e := data.GetString("password")
	if e != nil {
		return nil, fmt.Errorf("No ssh password: %s", e)
	}
	passwd, e := utils.DescryptAESBase64(id, login_key)
	if e != nil {
		return nil, e
	}
	passwd, e = utils.DescryptAESBase64(id, passwd)
	if e != nil {
		return nil, e
	}
	ret.Add(jsonutils.NewString(passwd), "password")
	v, e := data.Get("username")
	if e == nil {
		ret.Add(v, "username")
	}
	v, e = data.Get("ip")
	if e == nil {
		ret.Add(v, "ip")
	}
	return ret, nil
}

func (this *HostManager) GetIpmiInfo(s *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	data, e := this.GetSpecific(s, id, "ipmi", nil)
	if e != nil {
		return nil, e
	}

	ret := jsonutils.NewDict()
	passwd, e := data.GetString("password")
	if e != nil {
		return nil, fmt.Errorf("No IPMI password: %s", e)
	}

	ret.Add(jsonutils.NewString(passwd), "password")
	v, e := data.Get("username")
	if e == nil {
		ret.Add(v, "username")
	}
	v, e = data.Get("ip_addr")
	if e == nil {
		ret.Add(v, "ip")
	}
	return ret, nil
}

var (
	Hosts HostManager
)

func init() {
	Hosts = HostManager{NewComputeManager("host", "hosts",
		[]string{"ID", "Name", "Access_mac", "Access_ip",
			"Manager_URI",
			"Status", "enabled", "host_status",
			"Guests", "Running_guests",
			"storage", "storage_used",
			"storage_virtual", "disk_used",
			"storage_free", "storage_commit_rate",
			"mem_size", "mem_used", "mem_free",
			"mem_commit", "cpu_count", "cpu_used",
			"cpu_commit", "cpu_commit_rate",
			"mem_commit_rate", "cpu_commit_bound",
			"mem_commit_bound", "node_count", "sn", "storage_type",
			"host_type", "version", "schedtags",
			"storage_size"},
		[]string{})}
	registerCompute(&Hosts)
}
