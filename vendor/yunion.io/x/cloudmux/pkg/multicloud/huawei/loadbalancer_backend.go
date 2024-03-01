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

package huawei

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SElbBackend struct {
	multicloud.SResourceBase
	HuaweiTags
	region         *SRegion
	backendGroupId string

	Name            string `json:"name"`
	Weight          int    `json:"weight"`
	AdminStateUp    bool   `json:"admin_state_up"`
	SubnetID        string `json:"subnet_id"`
	TenantID        string `json:"tenant_id"`
	ProjectID       string `json:"project_id"`
	Address         string `json:"address"`
	ProtocolPort    int    `json:"protocol_port"`
	OperatingStatus string `json:"operating_status"`
	ID              string `json:"id"`
}

func (self *SElbBackend) GetId() string {
	return self.ID
}

func (self *SElbBackend) GetName() string {
	return self.Name
}

func (self *SElbBackend) GetGlobalId() string {
	return self.GetId()
}

func (self *SElbBackend) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (self *SElbBackend) Refresh() error {
	backend, err := self.region.GetElbBackend(self.backendGroupId, self.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, backend)
}

func (self *SElbBackend) GetProjectId() string {
	return ""
}

func (self *SElbBackend) GetWeight() int {
	return self.Weight
}

func (self *SElbBackend) GetPort() int {
	return self.ProtocolPort
}

func (self *SElbBackend) GetBackendType() string {
	return api.LB_BACKEND_GUEST
}

func (self *SElbBackend) GetBackendRole() string {
	return api.LB_BACKEND_ROLE_DEFAULT
}

func (self *SElbBackend) GetBackendId() string {
	vms, err := self.region.GetInstances(self.Address)
	if err != nil {
		return ""
	}
	for _, vm := range vms {
		return vm.GetId()
	}
	return ""
}

func (self *SElbBackend) GetIpAddress() string {
	return ""
}

func (self *SElbBackend) SyncConf(ctx context.Context, port, weight int) error {
	if port > 0 {
		log.Warningf("Elb backend SyncConf unsupport modify port")
	}

	params := map[string]interface{}{
		"weight": weight,
	}
	res := fmt.Sprintf("elb/pools/%s/members/%s", self.backendGroupId, self.ID)
	_, err := self.region.put(SERVICE_ELB, res, map[string]interface{}{"member": params})
	return err
}

func (self *SRegion) GetElbBackend(pool, id string) (*SElbBackend, error) {
	res := fmt.Sprintf("elb/pools/%s/members/%s", pool, id)
	resp, err := self.list(SERVICE_ELB, res, nil)
	if err != nil {
		return nil, err
	}
	ret := &SElbBackend{region: self}
	return ret, resp.Unmarshal(ret, "member")
}
