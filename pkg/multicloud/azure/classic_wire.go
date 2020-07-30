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

type SClassicWire struct {
	zone      *SZone
	vpc       *SClassicVpc
	inetworks []cloudprovider.ICloudNetwork
}

func (self *SClassicWire) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SClassicWire) GetId() string {
	return fmt.Sprintf("%s/%s/%s-classic", self.zone.region.GetGlobalId(), self.zone.region.client.subscriptionId, self.vpc.GetName())
}

func (self *SClassicWire) GetGlobalId() string {
	return strings.ToLower(self.GetId())
}

func (self *SClassicWire) GetName() string {
	return fmt.Sprintf("%s-%s-classic", self.zone.region.client.cpcfg.Name, self.vpc.GetName())
}

func (self *SClassicWire) IsEmulated() bool {
	return true
}

func (self *SClassicWire) GetStatus() string {
	return "available"
}

func (self *SClassicWire) Refresh() error {
	return nil
}

func (self *SClassicWire) addNetwork(network *SClassicNetwork) {
	if self.inetworks == nil {
		self.inetworks = make([]cloudprovider.ICloudNetwork, 0)
	}
	find := false
	for i := 0; i < len(self.inetworks); i += 1 {
		if self.inetworks[i].GetName() == network.Name {
			find = true
			break
		}
	}
	if !find {
		self.inetworks = append(self.inetworks, network)
	}
}

func (self *SClassicWire) CreateINetwork(opts *cloudprovider.SNetworkCreateOptions) (cloudprovider.ICloudNetwork, error) {
	return nil, cloudprovider.ErrNotImplemented
	// if network, err := self.zone.region.createNetwork(self.vpc, name, cidr, desc); err != nil {
	// 	return nil, err
	// } else {
	// 	network.wire = self
	// 	return network, nil
	// }
}

func (self *SClassicWire) GetBandwidth() int {
	return 10000
}

func (self *SClassicWire) GetINetworkById(netid string) (cloudprovider.ICloudNetwork, error) {
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

func (self *SClassicWire) GetINetworks() ([]cloudprovider.ICloudNetwork, error) {
	if err := self.vpc.fetchNetworks(); err != nil {
		return nil, err
	}
	return self.inetworks, nil
}

func (self *SClassicWire) GetIVpc() cloudprovider.ICloudVpc {
	return self.vpc
}

func (self *SClassicWire) GetIZone() cloudprovider.ICloudZone {
	return self.zone
}

func (self *SClassicWire) getNetworkById(networkId string) *SClassicNetwork {
	networks, err := self.GetINetworks()
	if err != nil {
		log.Errorf("getNetworkById error: %v", err)
		return nil
	}
	log.Debugf("search for networks %d networkId: %s", len(networks), networkId)
	for i := 0; i < len(networks); i++ {
		network := networks[i].(*SClassicNetwork)
		if network.GetId() == strings.ToLower(networkId) {
			return network
		}
	}
	return nil
}
