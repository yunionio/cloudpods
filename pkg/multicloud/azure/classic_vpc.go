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

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
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

	region *SRegion

	iwires    []cloudprovider.ICloudWire
	secgroups []cloudprovider.ICloudSecurityGroup

	ID         string
	Name       string
	Type       string
	Location   string
	Properties ClassicVpcProperties
}

func (self *SClassicVpc) GetMetadata() *jsonutils.JSONDict {
	return nil
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
	// TODO
	return true
}

func (self *SClassicVpc) GetCidrBlock() string {
	if len(self.Properties.AddressSpace.AddressPrefixes) > 0 {
		return self.Properties.AddressSpace.AddressPrefixes[0]
	}
	return ""
}

func (self *SClassicVpc) Delete() error {
	return self.region.client.Delete(self.ID)
}

func (self *SClassicVpc) getWire() *SClassicWire {
	if self.iwires == nil {
		self.fetchWires()
	}
	return self.iwires[0].(*SClassicWire)
}

func (region *SRegion) GetClassicVpc(vpcId string) (*SClassicVpc, error) {
	vpc := SClassicVpc{region: region}
	return &vpc, region.client.Get(vpcId, []string{}, &vpc)
}

func (self *SClassicVpc) fetchNetworks() error {
	vpc, err := self.region.GetClassicVpc(self.ID)
	if err != nil {
		return err
	}
	for i := 0; i < len(vpc.Properties.Subnets); i++ {
		network := vpc.Properties.Subnets[i]
		network.id = fmt.Sprintf("%s/%s", vpc.ID, network.Name)
		wire := self.getWire()
		network.wire = wire
		wire.addNetwork(&network)
	}
	return nil
}

func (self *SClassicVpc) getClassicSecurityGroups() ([]SClassicSecurityGroup, error) {
	securityGroups, err := self.region.GetClassicSecurityGroups("")
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(securityGroups); i++ {
		securityGroups[i].vpc = self
	}
	return securityGroups, nil
}

func (self *SClassicVpc) fetchSecurityGroups() error {
	self.secgroups = make([]cloudprovider.ICloudSecurityGroup, 0)
	secgrps, err := self.getClassicSecurityGroups()
	if err != nil {
		return err
	}
	for i := 0; i < len(secgrps); i++ {
		self.secgroups = append(self.secgroups, &secgrps[i])
	}
	return nil
}

func (self *SClassicVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	if self.secgroups == nil {
		err := self.fetchSecurityGroups()
		if err != nil {
			return nil, err
		}
	}
	return self.secgroups, nil
}

func (self *SClassicVpc) GetIRouteTables() ([]cloudprovider.ICloudRouteTable, error) {
	rts := []cloudprovider.ICloudRouteTable{}
	return rts, nil
}

func (self *SClassicVpc) fetchWires() error {
	networks := make([]cloudprovider.ICloudNetwork, len(self.Properties.Subnets))
	wire := SClassicWire{zone: self.region.izones[0].(*SZone), vpc: self}
	for i := 0; i < len(self.Properties.Subnets); i++ {
		network := self.Properties.Subnets[i]
		network.id = fmt.Sprintf("%s/%s", self.ID, self.Properties.Subnets[i].Name)
		network.wire = &wire
		networks[i] = &network
	}
	wire.inetworks = networks
	self.iwires = []cloudprovider.ICloudWire{&wire}
	return nil
}

func (self *SClassicVpc) GetIWireById(wireId string) (cloudprovider.ICloudWire, error) {
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

func (self *SClassicVpc) GetIWires() ([]cloudprovider.ICloudWire, error) {
	if self.iwires == nil {
		if err := self.fetchWires(); err != nil {
			return nil, err
		}
	}
	return self.iwires, nil
}

func (self *SClassicVpc) GetRegion() cloudprovider.ICloudRegion {
	return self.region
}

func (self *SClassicVpc) GetStatus() string {
	if strings.ToLower(self.Properties.Status) == "created" {
		return "available"
	}
	return "disabled"
}

func (self *SClassicVpc) Refresh() error {
	vpc, err := self.region.GetClassicVpc(self.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, vpc)
}

func (self *SClassicVpc) addWire(wire *SClassicWire) {
	if self.iwires == nil {
		self.iwires = make([]cloudprovider.ICloudWire, 0)
	}
	self.iwires = append(self.iwires, wire)
}

func (self *SClassicVpc) GetNetworks() []SClassicNetwork {
	for i := 0; i < len(self.Properties.Subnets); i++ {
		self.Properties.Subnets[i].id = fmt.Sprintf("%s/%s", self.ID, self.Properties.Subnets[i].Name)
	}
	return self.Properties.Subnets
}
