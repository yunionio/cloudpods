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

package hcs

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SElbBackend struct {
	multicloud.SResourceBase
	multicloud.HuaweiTags
	region       *SRegion
	lb           *SLoadbalancer
	backendGroup *SElbBackendGroup

	Name            string `json:"name"`
	Weight          int    `json:"weight"`
	AdminStateUp    bool   `json:"admin_state_up"`
	SubnetId        string `json:"subnet_id"`
	TenantId        string `json:"tenant_id"`
	ProjectId       string `json:"project_id"`
	Address         string `json:"address"`
	ProtocolPort    int    `json:"protocol_port"`
	OperatingStatus string `json:"operating_status"`
	Id              string `json:"id"`
}

func (self *SElbBackend) GetId() string {
	return self.Id
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
	backend, err := self.region.GetElbBackend(self.backendGroup.GetId(), self.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, backend)
}

func (self *SElbBackend) IsEmulated() bool {
	return false
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
	i, err := self.lb.region.getInstanceByIP(self.Address)
	if err != nil {
		log.Errorf("ElbBackend GetBackendId %s", err)
	}

	if i != nil {
		return i.GetId()
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
	res := fmt.Sprintf("lbaas/pools/%s/members/%s", self.backendGroup.GetId(), self.Id)
	return self.region.lbUpdate(res, map[string]interface{}{"member": params})
}

func (self *SRegion) getInstanceByIP(ip string) (*SInstance, error) {
	instances, err := self.GetInstances(ip)
	if err != nil {
		return nil, errors.Wrapf(err, "GetInstance(%s)", ip)
	}
	for i := range instances {
		return &instances[i], nil
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetElbBackend(pool, id string) (*SElbBackend, error) {
	res := fmt.Sprintf("lbaas/pools/%s/members/%s", pool, id)
	ret := &SElbBackend{region: self}
	return ret, self.lbGet(res, ret)
}
