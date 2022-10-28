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

package hcso

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/cloudmux/pkg/multicloud/huawei"
)

type SElbBackend struct {
	multicloud.SResourceBase
	huawei.HuaweiTags
	region       *SRegion
	lb           *SLoadbalancer
	backendGroup *SElbBackendGroup

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
	m := self.lb.region.ecsClient.ElbBackend
	err := m.SetBackendGroupId(self.backendGroup.GetId())
	if err != nil {
		return err
	}

	backend := SElbBackend{}
	err = DoGet(m.Get, self.GetId(), nil, &backend)
	if err != nil {
		return err
	}

	backend.lb = self.lb
	backend.backendGroup = self.backendGroup
	err = jsonutils.Update(self, backend)
	if err != nil {
		return err
	}

	return nil
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

	params := jsonutils.NewDict()
	memberObj := jsonutils.NewDict()
	memberObj.Set("weight", jsonutils.NewInt(int64(weight)))
	params.Set("member", memberObj)
	err := self.lb.region.ecsClient.ElbBackend.SetBackendGroupId(self.backendGroup.GetId())
	if err != nil {
		return err
	}
	return DoUpdate(self.lb.region.ecsClient.ElbBackend.Update, self.GetId(), params, nil)
}

func (self *SRegion) getInstanceByIP(privateIP string) (*SInstance, error) {
	queries := make(map[string]string)

	if len(self.client.projectId) > 0 {
		queries["project_id"] = self.client.projectId
	}

	if len(privateIP) > 0 {
		queries["ip"] = privateIP
	}

	instances := make([]SInstance, 0)
	err := doListAllWithOffset(self.ecsClient.Servers.List, queries, &instances)
	if err != nil {
		return nil, err
	}

	if len(instances) == 1 {
		return &instances[0], nil
	} else if len(instances) > 1 {
		log.Warningf("SRegion.getInstanceByIP %s result: multiple server find", privateIP)
		return &instances[0], nil
	}

	return nil, cloudprovider.ErrNotFound
}
