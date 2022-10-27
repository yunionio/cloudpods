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

	"yunion.io/x/log"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SClassicWire struct {
	multicloud.SResourceBase
	AzureTags

	zone      *SZone
	vpc       *SClassicVpc
	inetworks []cloudprovider.ICloudNetwork
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
	return api.WIRE_STATUS_AVAILABLE
}

func (self *SClassicWire) CreateINetwork(opts *cloudprovider.SNetworkCreateOptions) (cloudprovider.ICloudNetwork, error) {
	return nil, cloudprovider.ErrNotImplemented
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
	networks := self.vpc.GetNetworks()
	ret := []cloudprovider.ICloudNetwork{}
	for i := range networks {
		networks[i].wire = self
		ret = append(ret, &networks[i])
	}
	return ret, nil
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
