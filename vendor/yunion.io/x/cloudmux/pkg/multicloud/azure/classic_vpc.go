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

package azure

import (
	"fmt"
	"strings"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type ClassicAddressSpace struct {
	AddressPrefixes []string
}

type ClassicSubnet struct {
	Name          string
	AddressPrefix string
}

type ClassicVpcProperties struct {
	ProvisioningState string
	Status            string
	SiteId            string
	InUse             bool
	AddressSpace      ClassicAddressSpace
	Subnets           []SClassicNetwork
}

type SClassicVpc struct {
	multicloud.SVpc
	AzureTags

	region *SRegion

	ID         string
	Name       string
	Type       string
	Location   string
	Properties ClassicVpcProperties
}

func (self *SClassicVpc) GetId() string {
	return self.ID
}

func (self *SClassicVpc) GetName() string {
	return self.Name
}

func (self *SClassicVpc) GetGlobalId() string {
	return strings.ToLower(self.ID)
}

func (self *SClassicVpc) IsEmulated() bool {
	return false
}

func (self *SClassicVpc) GetIsDefault() bool {
	return false
}

func (self *SClassicVpc) GetCidrBlock() string {
	if len(self.Properties.AddressSpace.AddressPrefixes) > 0 {
		return self.Properties.AddressSpace.AddressPrefixes[0]
	}
	return ""
}

func (self *SClassicVpc) Delete() error {
	return self.region.del(self.ID)
}

func (self *SClassicVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	secgroups, err := self.region.ListSecgroups()
	if err != nil {
		return nil, errors.Wrapf(err, "ListSecgroups")
	}
	ret := []cloudprovider.ICloudSecurityGroup{}
	for i := range secgroups {
		secgroups[i].region = self.region
		ret = append(ret, &secgroups[i])
	}
	return ret, nil
}

func (self *SClassicVpc) GetIRouteTables() ([]cloudprovider.ICloudRouteTable, error) {
	rts := []cloudprovider.ICloudRouteTable{}
	return rts, nil
}

func (self *SClassicVpc) GetIRouteTableById(routeTableId string) (cloudprovider.ICloudRouteTable, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SClassicVpc) getWire() *SClassicWire {
	return &SClassicWire{vpc: self, zone: self.region.getZone()}
}

func (self *SClassicVpc) GetIWireById(wireId string) (cloudprovider.ICloudWire, error) {
	wire := self.getWire()
	if wire.GetGlobalId() != wireId {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, wireId)
	}
	return wire, nil
}

func (self *SClassicVpc) GetIWires() ([]cloudprovider.ICloudWire, error) {
	return []cloudprovider.ICloudWire{self.getWire()}, nil
}

func (self *SClassicVpc) GetRegion() cloudprovider.ICloudRegion {
	return self.region
}

func (self *SClassicVpc) GetStatus() string {
	return api.VPC_STATUS_AVAILABLE
}

func (self *SClassicVpc) GetNetworks() []SClassicNetwork {
	for i := 0; i < len(self.Properties.Subnets); i++ {
		self.Properties.Subnets[i].id = fmt.Sprintf("%s/%s", self.ID, self.Properties.Subnets[i].Name)
	}
	return self.Properties.Subnets
}
