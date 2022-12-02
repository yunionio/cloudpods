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
	"net/url"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SExternalVpc struct {
	multicloud.SVpc
	HcsTags

	region *SRegion

	Id      string
	Name    string
	CIDR    string
	Status  string
	Subnets []string
}

func (self *SExternalVpc) GetId() string {
	return self.Id
}

func (self *SExternalVpc) GetName() string {
	if len(self.Name) > 0 {
		return self.Name
	}
	return self.Id
}

func (self *SExternalVpc) GetGlobalId() string {
	return self.Id
}

func (self *SExternalVpc) GetStatus() string {
	return api.VPC_STATUS_AVAILABLE
}

func (self *SExternalVpc) Refresh() error {
	return nil
}

func (self *SExternalVpc) IsEmulated() bool {
	return true
}

func (self *SExternalVpc) GetRegion() cloudprovider.ICloudRegion {
	return self.region
}

func (self *SExternalVpc) GetIsDefault() bool {
	return false
}

func (self *SExternalVpc) GetCidrBlock() string {
	return ""
}

func (self *SExternalVpc) GetIWires() ([]cloudprovider.ICloudWire, error) {
	wire := &SExternalWire{region: self.region, vpc: self}
	return []cloudprovider.ICloudWire{wire}, nil
}

func (self *SExternalVpc) GetIWireById(id string) (cloudprovider.ICloudWire, error) {
	wires, err := self.GetIWires()
	if err != nil {
		return nil, err
	}
	for i := range wires {
		if wires[i].GetGlobalId() == id {
			return wires[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SExternalVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SExternalVpc) GetIRouteTables() ([]cloudprovider.ICloudRouteTable, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SExternalVpc) GetIRouteTableById(routeTableId string) (cloudprovider.ICloudRouteTable, error) {
	return nil, cloudprovider.ErrNotFound
}

func (self *SExternalVpc) Delete() error {
	return cloudprovider.ErrNotFound
}

func (self *SRegion) GetExternalVpcs() ([]SExternalVpc, error) {
	ret := []SExternalVpc{}
	params := url.Values{}
	params.Set("router:external", "True")
	params.Set("service_type", "Intranet")
	return ret, self.list("vpc", "v2.0", "networks", params, &ret)
}
