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

package proxmox

import (
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SVpc struct {
	multicloud.SVpc
	ProxmoxTags

	client *SProxmoxClient
}

func (self *SVpc) GetName() string {
	return self.client.GetName()
}

func (self *SVpc) GetId() string {
	return self.client.GetId()
}

func (self *SVpc) GetGlobalId() string {
	return self.GetId()
}

func (self *SVpc) Delete() error {
	return cloudprovider.ErrNotSupported
}

func (self *SVpc) IsEmulated() bool {
	return true
}

func (self *SVpc) GetCidrBlock() string {
	return "0.0.0.0/0"
}

func (self *SVpc) GetIRouteTables() ([]cloudprovider.ICloudRouteTable, error) {
	return []cloudprovider.ICloudRouteTable{}, nil
}

func (self *SVpc) GetIRouteTableById(routeTableId string) (cloudprovider.ICloudRouteTable, error) {
	return nil, cloudprovider.ErrNotFound
}

func (self *SVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	return []cloudprovider.ICloudSecurityGroup{}, nil
}

func (self *SVpc) GetIWires() ([]cloudprovider.ICloudWire, error) {
	nodes, err := self.client.GetHosts()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudWire{}
	for i := range nodes {
		wires, err := self.client.GetWires(nodes[i].Node)
		if err != nil {
			return nil, err
		}
		for j := range wires {
			wires[j].client = self.client
			ret = append(ret, &wires[j])
		}
	}
	return ret, nil
}

func (self *SVpc) GetIWireById(id string) (cloudprovider.ICloudWire, error) {
	wire, err := self.client.GetWire(id)
	if err != nil {
		return nil, err
	}
	wire.client = self.client
	return wire, nil
}

func (self *SVpc) GetIsDefault() bool {
	return true
}

func (self *SVpc) GetRegion() cloudprovider.ICloudRegion {
	return self.client
}

func (self *SVpc) GetStatus() string {
	return api.VPC_STATUS_AVAILABLE
}
