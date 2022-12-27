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

package compute

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/printutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

const MACAddressPattern = `(([a-fA-F0-9]{2}[:-]){5}([a-fA-F0-9]{2}))`

type HostManager struct {
	modulebase.ResourceManager
}

func (this *HostManager) GetLoginInfo(s *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	data, e := this.GetMetadata(s, id, nil)
	if e != nil {
		return nil, e
	}
	ret := jsonutils.NewDict()
	login_key, e := data.GetString("password")
	if e != nil {
		return nil, httperrors.NewNotFoundError("No ssh password: %s", e)
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

func parseHosts(titles []string, data string) []*jsonutils.JSONDict {
	hosts := strings.Split(data, "\n")
	ret := []*jsonutils.JSONDict{}
	for i, host := range hosts {
		host = strings.TrimSpace(host)
		if len(host) == 0 {
			log.Debugf(fmt.Sprintf("DoBatchRegister 第%d行： 空白行（已忽略）\n", i))
			continue
		}

		fields := strings.Split(host, ",")

		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString("baremetal"), "host_type")

		for i := range fields {
			field := fields[i]
			if len(field) > 0 {
				params.Add(jsonutils.NewString(field), titles[i])
			}
		}

		ret = append(ret, params)
	}

	return ret
}

func (this *HostManager) BatchRegister(s *mcclient.ClientSession, titles []string, params jsonutils.JSONObject) ([]printutils.SubmitResult, error) {
	data, err := params.GetString("hosts")
	if err != nil {
		return nil, err
	}
	input := params.(*jsonutils.JSONDict)
	input.Remove("hosts")

	hosts := parseHosts(titles, data)

	results := make(chan printutils.SubmitResult, len(hosts))
	for _, host := range hosts {
		host.Update(input)
		go func(data jsonutils.JSONObject) {
			ret, e := this.Create(s, data)
			id, _ := data.GetString("access_mac")
			if e != nil {
				ecls, ok := e.(*httputils.JSONClientError)
				if ok {
					results <- printutils.SubmitResult{Status: ecls.Code, Id: id, Data: jsonutils.Marshal(ecls)}
				} else {
					results <- printutils.SubmitResult{Status: 400, Id: id, Data: jsonutils.NewString(e.Error())}
				}
			} else {
				results <- printutils.SubmitResult{Status: 200, Id: id, Data: ret}
			}
		}(host)
	}

	ret := make([]printutils.SubmitResult, len(hosts))
	for i := 0; i < len(hosts); i++ {
		ret[i] = <-results
	}
	close(results)

	return ret, nil
}

func (this *HostManager) DoBatchRegister(s *mcclient.ClientSession, params jsonutils.JSONObject) ([]printutils.SubmitResult, error) {
	titles := []string{"access_mac", "name", "ipmi_ip_addr", "ipmi_username", "ipmi_password"}
	return this.BatchRegister(s, titles, params)
}

var (
	Hosts HostManager
)

func init() {
	Hosts = HostManager{modules.NewComputeManager("host", "hosts",
		[]string{"ID", "Name", "Access_mac", "Access_ip", "Ipmi_Ip", "Ovn_Mapped_Ip_Addr",
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
			"storage_size",
			"expired_at",
			"domain_id", "project_domain",
			"public_scope",
		},
		[]string{})}
	modules.RegisterCompute(&Hosts)
}
