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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

type ServerManager struct {
	modulebase.ResourceManager
}

func (this *ServerManager) GetLoginInfo(s *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	data, e := this.Get(s, id, nil)
	if e != nil {
		return nil, e
	}

	ret := jsonutils.NewDict()

	v, e := data.Get("metadata", "login_account")
	if e == nil {
		ret.Add(v, "username")
	}
	v, e = data.Get("metadata", "login_key_timestamp")
	if e == nil {
		ret.Add(v, "updated")
	}

	loginKey, e := data.GetString("metadata", "login_key")
	if e != nil {
		return nil, httperrors.NewNotFoundError("No login key: %s", e)
	}

	if len(loginKey) > 0 {
		ret.Add(jsonutils.NewString(loginKey), "login_key")
		var passwd string
		keypairId, _ := data.GetString("keypair_id")
		if len(keypairId) > 0 && !strings.EqualFold(keypairId, "none") {
			keypair, e := data.Get("keypair")
			if e == nil {
				ret.Add(keypair, "keypair")
			}
			if params != nil && !gotypes.IsNil(params) {
				privateKey, _ := params.GetString("private_key")
				privateKey = strings.TrimSpace(privateKey)
				if len(privateKey) > 0 {
					passwd, e = seclib2.DecryptBase64(privateKey, loginKey)
					if e != nil {
						return nil, e
					}
				}
			}
		} else {
			passwd, e = utils.DescryptAESBase64(id, loginKey)
			if e != nil {
				return nil, e
			}
		}
		if len(passwd) > 0 {
			ret.Add(jsonutils.NewString(passwd), "password")
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
	Servers = ServerManager{modules.NewComputeManager("server", "servers",
		[]string{"ID", "Name", "Billing_type",
			"IPs", "EIP", "Disk", "Status",
			"vcpu_count", "vmem_size",
			"ext_bw", "Zone_name",
			"Secgroup", "Secgrp_id",
			"vrouter", "vrouter_id",
			"Created_at", "Group_name",
			"Group_id", "Hypervisor", "os_type",
			"expired_at"},
		[]string{"Host", "Tenant", "is_system", "auto_delete_at", "backup_host_name"})}

	modules.RegisterCompute(&Servers)
}
