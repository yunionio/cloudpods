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
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type AddressSpace struct {
	AddressPrefixes []string `json:"addressPrefixes,omitempty"`
}

type VirtualNetworkPropertiesFormat struct {
	ProvisioningState      string
	Status                 string
	VirtualNetworkSiteName string
	AddressSpace           AddressSpace `json:"addressSpace,omitempty"`
	Subnets                []SNetwork   `json:"subnets,omitempty"`
}

type SVpc struct {
	multicloud.SVpc
	AzureTags
	region *SRegion

	ID         string
	Name       string
	Etag       string
	Type       string
	Location   string
	Properties VirtualNetworkPropertiesFormat `json:"properties,omitempty"`
}

func (self *SVpc) GetTags() (map[string]string, error) {
	return self.Tags, nil
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
	return true
}

func (self *SVpc) GetCidrBlock() string {
	if len(self.Properties.AddressSpace.AddressPrefixes) > 0 {
		return self.Properties.AddressSpace.AddressPrefixes[0]
	}
	return ""
}

func (self *SVpc) Delete() error {
	return self.region.DeleteVpc(self.ID)
}

func (self *SRegion) DeleteVpc(vpcId string) error {
	return self.del(vpcId)
}

func (self *SVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	return []cloudprovider.ICloudSecurityGroup{}, nil
}

func (self *SVpc) GetIRouteTables() ([]cloudprovider.ICloudRouteTable, error) {
	rts := []cloudprovider.ICloudRouteTable{}
	return rts, nil
}

func (self *SVpc) GetIRouteTableById(routeTableId string) (cloudprovider.ICloudRouteTable, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SVpc) GetIWireById(wireId string) (cloudprovider.ICloudWire, error) {
	wire := self.getWire()
	if wire.GetGlobalId() != wireId {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, wireId)
	}
	return wire, nil
}

func (self *SVpc) getWire() *SWire {
	zone := self.region.getZone()
	return &SWire{zone: zone, vpc: self}
}

func (self *SVpc) GetIWires() ([]cloudprovider.ICloudWire, error) {
	return []cloudprovider.ICloudWire{self.getWire()}, nil
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
	return &vpc, region.get(vpcId, url.Values{}, &vpc)
}

func (self *SVpc) Refresh() error {
	vpc, err := self.region.GetVpc(self.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, vpc)
}

func (self *SVpc) GetNetworks() []SNetwork {
	return self.Properties.Subnets
}

func (self *SRegion) GetNetwork(networkId string) (*SNetwork, error) {
	network := SNetwork{}
	return &network, self.get(networkId, url.Values{}, &network)
}
