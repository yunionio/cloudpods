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

package incloudsphere

import (
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SVpc struct {
	multicloud.SVpc
	multicloud.InCloudSphereTags

	region *SRegion
}

func (self *SVpc) GetName() string {
	return "Default"
}

func (self *SVpc) GetId() string {
	return self.region.GetId()
}

func (self *SVpc) GetGlobalId() string {
	return self.GetId()
}

func (self *SVpc) Delete() error {
	return cloudprovider.ErrNotImplemented
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
	wires, err := self.region.GetWires()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudWire{}
	for i := range wires {
		wires[i].region = self.region
		ret = append(ret, &wires[i])
	}
	return ret, nil
}

func (self *SVpc) GetIWireById(id string) (cloudprovider.ICloudWire, error) {
	wire, err := self.region.GetWire(id)
	if err != nil {
		return nil, err
	}
	wire.region = self.region
	return wire, nil
}

func (self *SVpc) GetIsDefault() bool {
	return true
}

func (self *SVpc) GetRegion() cloudprovider.ICloudRegion {
	return self.region
}

func (self *SVpc) GetStatus() string {
	return api.VPC_STATUS_AVAILABLE
}
