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
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type AddressSpace struct {
	AddressPrefixes []string `json:"addressPrefixes,omitempty"`
}

type SubnetPropertiesFormat struct {
	AddressPrefix string `json:"addressPrefix,omitempty"`
	//ProvisioningState string
}

type Subnet struct {
	Properties SubnetPropertiesFormat
	Name       string
	ID         string
}

type VirtualNetworkPropertiesFormat struct {
	ProvisioningState      string
	Status                 string
	VirtualNetworkSiteName string
	AddressSpace           AddressSpace `json:"addressSpace,omitempty"`
	Subnets                *[]SNetwork  `json:"subnets,omitempty"`
}

type SVpc struct {
	multicloud.SVpc

	region *SRegion

	iwires    []cloudprovider.ICloudWire
	secgroups []cloudprovider.ICloudSecurityGroup

	isDefault bool

	ID         string
	Name       string
	Etag       string
	Type       string
	Location   string
	Tags       map[string]string
	Properties VirtualNetworkPropertiesFormat `json:"properties,omitempty"`
}

func (self *SVpc) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SVpc) GetId() string {
	return self.ID
}

func (self *SVpc) GetName() string {
	return self.Name
}

func (self *SVpc) GetGlobalId() string {
	return strings.ToLower(self.ID)
}

func (self *SVpc) IsEmulated() bool {
	return false
}

func (self *SVpc) GetIsDefault() bool {
	return self.isDefault
}

func (self *SVpc) GetCidrBlock() string {
	return self.Properties.AddressSpace.AddressPrefixes[0]
}

func (self *SVpc) Delete() error {
	return self.region.DeleteVpc(self.ID)
}

func (self *SRegion) DeleteVpc(vpcId string) error {
	return self.client.Delete(vpcId)
}

func (self *SVpc) getSecurityGroups() ([]SSecurityGroup, error) {
	securityGroups, err := self.region.GetSecurityGroups("")
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(securityGroups); i++ {
		securityGroups[i].vpc = self
	}
	return securityGroups, nil
}

func (self *SVpc) fetchSecurityGroups() error {
	self.secgroups = make([]cloudprovider.ICloudSecurityGroup, 0)
	if secgrps, err := self.getSecurityGroups(); err != nil {
		return err
	} else {
		for i := 0; i < len(secgrps); i++ {
			self.secgroups = append(self.secgroups, &secgrps[i])
		}
		return nil
	}
}

func (self *SVpc) getWire() *SWire {
	if self.iwires == nil {
		self.fetchWires()
	}
	return self.iwires[0].(*SWire)
}

func (self *SVpc) fetchNetworks() error {
	vpc, err := self.region.GetVpc(self.ID)
	if err != nil {
		return err
	}
	if vpc.Properties.Subnets != nil {
		networks := *vpc.Properties.Subnets
		wire := self.getWire()
		for i := 0; i < len(networks); i++ {
			networks[i].wire = wire
			wire.addNetwork(&networks[i])
		}
	}
	return nil
}

func (self *SVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	if self.secgroups == nil {
		err := self.fetchSecurityGroups()
		if err != nil {
			return nil, err
		}
	}
	return self.secgroups, nil
}

func (self *SVpc) GetIRouteTables() ([]cloudprovider.ICloudRouteTable, error) {
	rts := []cloudprovider.ICloudRouteTable{}
	return rts, nil
}

func (self *SVpc) fetchWires() error {
	networks := make([]cloudprovider.ICloudNetwork, len(*self.Properties.Subnets))
	if len(self.region.izones) == 0 {
		self.region.fetchZones()
	}
	wire := SWire{zone: self.region.izones[0].(*SZone), vpc: self, inetworks: networks}
	for i, _network := range *self.Properties.Subnets {
		network := SNetwork{wire: &wire}
		if err := jsonutils.Update(&network, _network); err != nil {
			return err
		}
		networks[i] = &network
	}
	self.iwires = []cloudprovider.ICloudWire{&wire}
	return nil
}

func (self *SVpc) GetIWireById(wireId string) (cloudprovider.ICloudWire, error) {
	if self.iwires == nil {
		if err := self.fetchNetworks(); err != nil {
			return nil, err
		}
	}
	for i := 0; i < len(self.iwires); i++ {
		if self.iwires[i].GetGlobalId() == wireId {
			return self.iwires[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SVpc) GetIWires() ([]cloudprovider.ICloudWire, error) {
	if self.iwires == nil {
		if err := self.fetchWires(); err != nil {
			return nil, err
		}
	}
	return self.iwires, nil
}

func (self *SVpc) GetRegion() cloudprovider.ICloudRegion {
	return self.region
}

func (self *SVpc) GetStatus() string {
	if strings.ToLower(self.Properties.ProvisioningState) == "succeeded" {
		return "available"
	}
	return "disabled"
}

func (region *SRegion) GetVpc(vpcId string) (*SVpc, error) {
	vpc := SVpc{region: region}
	return &vpc, region.client.Get(vpcId, []string{}, &vpc)
}

func (self *SVpc) Refresh() error {
	if vpc, err := self.region.GetVpc(self.ID); err != nil {
		return err
	} else if err := jsonutils.Update(self, vpc); err != nil {
		return err
	}
	return nil
}

func (self *SVpc) addWire(wire *SWire) {
	if self.iwires == nil {
		self.iwires = make([]cloudprovider.ICloudWire, 0)
	}
	self.iwires = append(self.iwires, wire)
}

func (self *SVpc) GetNetworks() []SNetwork {
	return *self.Properties.Subnets
}

func (self *SRegion) GetNetworkDetail(networkId string) (*Subnet, error) {
	subnet := Subnet{}
	return &subnet, self.client.Get(networkId, []string{}, &subnet)
}
