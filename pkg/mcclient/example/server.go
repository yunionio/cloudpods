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

package example

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

const (
	AuthURL      = "https://10.168.222.209:5000/v3"
	DomainName   = "Default"
	TenantDomain = ""
	TenantName   = "system"
	Username     = "a"
	Password     = "a"
	Region       = "Yunion"
	EndpointType = "publicURL"
	Debug        = false
)

func getSession() (*mcclient.ClientSession, error) {
	client := mcclient.NewClient(AuthURL, 10, Debug, true, "", "")
	token, err := client.Authenticate(Username, Password, DomainName, TenantName, TenantDomain)
	if err != nil {
		return nil, err
	}
	return client.NewSession(context.Background(), Region, "", EndpointType, token, "v2"), nil
}

func CreateServerExample() error {
	session, err := getSession()
	if err != nil {
		log.Errorf("get session error: %v", err)
		return err
	}
	// params := options.ServerCreateOptions{}
	// params.NAME = "test-create"
	// params.VcpuCount = 1
	// params.MEMSPEC = "2G"
	// params.Disk = []string{"a4171f87-7e70-43bf-852e-4a3e1bf7deab:local"}

	// 参数参考: docs/schemas/instance.yaml#InstanceCreate
	params := map[string]interface{}{
		"name":       "test-create",
		"vcpu_count": 1,
		"vmem_size":  2048,
		"disks": []map[string]string{
			{
				"image_id":  "a4171f87-7e70-43bf-852e-4a3e1bf7deab",
				"disk_type": "sys",
				"backend":   "local",
			},
		},
	}
	resp, err := modules.Servers.Create(session, jsonutils.Marshal(params))
	if err != nil {
		log.Errorf("create server error: %v", err)
		return err
	}
	log.Infof("result: %s", resp.PrettyString())
	return nil
}

func GetServerExample() error {
	session, err := getSession()
	if err != nil {
		log.Errorf("get session error: %v", err)
		return err
	}

	pendingDelete := true //回收站实例
	opts := options.ServerListOptions{}
	opts.PendingDelete = &pendingDelete

	params, err := options.StructToParams(opts)
	if err != nil {
		log.Errorf("params error: %v", err)
		return err
	}

	server, err := modules.Servers.Get(session, "test-create", params)
	if err != nil {
		log.Errorf("get server info error")
		return err
	}
	log.Infof("server info: %s", server.PrettyString())
	return nil
}
