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

package remotefile

import (
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SVpc struct {
	multicloud.SVpc
	SResourceBase

	region *SRegion

	RegionId  string
	CidrBlock string
	IsDefault bool
}

func (self *SVpc) Delete() error {
	return cloudprovider.ErrNotSupported
}

func (self *SVpc) GetCidrBlock() string {
	return self.CidrBlock
}

func (self *SVpc) GetIRouteTableById(id string) (cloudprovider.ICloudRouteTable, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SVpc) GetIRouteTables() ([]cloudprovider.ICloudRouteTable, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SVpc) GetIWires() ([]cloudprovider.ICloudWire, error) {
	wires, err := self.region.client.GetWires()
	if err != nil {
		return nil, err
	}
	zones, err := self.region.client.GetZones()
	if err != nil {
		return nil, err
	}
	zoneIds := []string{}
	for i := range zones {
		zoneIds = append(zoneIds, zones[i].Id)
	}
	ret := []cloudprovider.ICloudWire{}
	for i := range wires {
		if wires[i].VpcId != self.GetId() || !utils.IsInStringArray(wires[i].ZoneId, zoneIds) {
			continue
		}
		wires[i].region = self.region
		ret = append(ret, &wires[i])
	}
	return ret, nil
}

func (self *SVpc) CreateIWire(opts *cloudprovider.SWireCreateOptions) (cloudprovider.ICloudWire, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	secgroups, err := self.region.client.GetSecgroups()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudSecurityGroup{}
	for i := range secgroups {
		if secgroups[i].VpcId != self.GetId() {
			continue
		}
		ret = append(ret, &secgroups[i])
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SVpc) GetIWireById(id string) (cloudprovider.ICloudWire, error) {
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

func (self *SVpc) GetINatGateways() ([]cloudprovider.ICloudNatGateway, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SVpc) CreateINatGateway(opts *cloudprovider.NatGatewayCreateOptions) (cloudprovider.ICloudNatGateway, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SVpc) GetICloudVpcPeeringConnections() ([]cloudprovider.ICloudVpcPeeringConnection, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SVpc) GetICloudAccepterVpcPeeringConnections() ([]cloudprovider.ICloudVpcPeeringConnection, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SVpc) GetICloudVpcPeeringConnectionById(id string) (cloudprovider.ICloudVpcPeeringConnection, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SVpc) CreateICloudVpcPeeringConnection(opts *cloudprovider.VpcPeeringConnectionCreateOptions) (cloudprovider.ICloudVpcPeeringConnection, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SVpc) AcceptICloudVpcPeeringConnection(id string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SVpc) GetAuthorityOwnerId() string {
	return ""
}

func (self *SVpc) ProposeJoinICloudInterVpcNetwork(opts *cloudprovider.SVpcJointInterVpcNetworkOption) error {
	return cloudprovider.ErrNotSupported
}

func (self *SVpc) GetICloudIPv6Gateways() ([]cloudprovider.ICloudIPv6Gateway, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SVpc) GetIsDefault() bool {
	return self.IsDefault
}

func (self *SVpc) GetRegion() cloudprovider.ICloudRegion {
	return self.region
}
