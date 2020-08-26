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
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SWire struct {
	zone      *SZone
	vpc       *SVpc
	inetworks []cloudprovider.ICloudNetwork
}

func (self *SWire) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SWire) GetId() string {
	return fmt.Sprintf("%s/%s/%s", self.zone.region.GetGlobalId(), self.zone.region.client.subscriptionId, self.vpc.GetName())
}

func (self *SWire) GetGlobalId() string {
	return strings.ToLower(self.GetId())
}

func (self *SWire) GetName() string {
	return fmt.Sprintf("%s-%s", self.zone.region.client.cpcfg.Name, self.vpc.GetName())
}

func (self *SWire) IsEmulated() bool {
	return true
}

func (self *SWire) GetStatus() string {
	return "available"
}

func (self *SWire) Refresh() error {
	return nil
}

func (self *SWire) addNetwork(network *SNetwork) {
	if self.inetworks == nil {
		self.inetworks = make([]cloudprovider.ICloudNetwork, 0)
	}
	find := false
	for i := 0; i < len(self.inetworks); i += 1 {
		if self.inetworks[i].GetGlobalId() == strings.ToLower(network.ID) {
			find = true
			break
		}
	}
	if !find {
		self.inetworks = append(self.inetworks, network)
	}
}

func (self *SRegion) createNetwork(vpc *SVpc, subnetName string, cidr string, desc string) (*SNetwork, error) {
	subnet := SNetwork{
		Name: subnetName,
		Properties: SubnetPropertiesFormat{
			AddressPrefix: cidr,
		},
	}
	if vpc.Properties.Subnets == nil {
		subnets := []SNetwork{subnet}
		vpc.Properties.Subnets = &subnets
	} else {
		*vpc.Properties.Subnets = append(*vpc.Properties.Subnets, subnet)
	}
	vpc.Properties.ProvisioningState = ""
	err := self.client.Update(jsonutils.Marshal(vpc), vpc)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(*vpc.Properties.Subnets); i++ {
		if (*vpc.Properties.Subnets)[i].Name == subnetName {
			subnet.ID = (*vpc.Properties.Subnets)[i].ID
		}
	}
	return &subnet, nil
}

func (self *SWire) CreateINetwork(opts *cloudprovider.SNetworkCreateOptions) (cloudprovider.ICloudNetwork, error) {
	network, err := self.zone.region.createNetwork(self.vpc, opts.Name, opts.Cidr, opts.Desc)
	if err != nil {
		return nil, err
	}
	network.wire = self
	return network, nil
}

func (self *SWire) GetBandwidth() int {
	return 10000
}

func (self *SWire) GetINetworkById(netid string) (cloudprovider.ICloudNetwork, error) {
	networks, err := self.GetINetworks()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(networks); i += 1 {
		if networks[i].GetGlobalId() == strings.ToLower(netid) {
			return networks[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SWire) GetINetworks() ([]cloudprovider.ICloudNetwork, error) {
	if err := self.vpc.fetchNetworks(); err != nil {
		return nil, err
	}
	return self.inetworks, nil
}

func (self *SWire) GetIVpc() cloudprovider.ICloudVpc {
	return self.vpc
}

func (self *SWire) GetIZone() cloudprovider.ICloudZone {
	return self.zone
}

func (self *SWire) getNetworkById(networkId string) *SNetwork {
	if networks, err := self.GetINetworks(); err != nil {
		log.Errorf("getNetworkById error: %v", err)
		return nil
	} else {
		log.Debugf("search for networks %d", len(networks))
		for i := 0; i < len(networks); i++ {
			network := networks[i].(*SNetwork)
			if strings.ToLower(networkId) == strings.ToLower(network.ID) {
				return network
			}
		}
	}
	return nil
}
