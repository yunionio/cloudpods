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

package oracle

import (
	"fmt"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SWire struct {
	multicloud.SResourceBase
	SOracleTag

	vpc  *SVpc
	zone *SZone
}

func (self *SWire) GetId() string {
	if self.zone != nil {
		return fmt.Sprintf("%s-%s", self.vpc.GetId(), self.zone.Name)
	}
	return self.vpc.GetId()
}

func (self *SWire) GetName() string {
	return self.GetId()
}

func (self *SWire) IsEmulated() bool {
	return true
}

func (self *SWire) GetStatus() string {
	return api.WIRE_STATUS_AVAILABLE
}

func (self *SWire) Refresh() error {
	return nil
}

func (self *SWire) GetGlobalId() string {
	return self.GetId()
}

func (self *SWire) GetIVpc() cloudprovider.ICloudVpc {
	return self.vpc
}

func (self *SWire) GetIZone() cloudprovider.ICloudZone {
	if self.zone == nil {
		return nil
	}
	return self.zone
}

func (self *SWire) GetBandwidth() int {
	return 10000
}

func (self *SWire) CreateINetwork(opts *cloudprovider.SNetworkCreateOptions) (cloudprovider.ICloudNetwork, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SWire) GetINetworkById(id string) (cloudprovider.ICloudNetwork, error) {
	network, err := self.vpc.region.GetNetwork(id)
	if err != nil {
		return nil, err
	}
	network.wire = self
	return network, nil
}

func (self *SWire) GetINetworks() ([]cloudprovider.ICloudNetwork, error) {
	networks, err := self.vpc.region.GetNetworks(self.vpc.Id)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudNetwork{}
	for i := range networks {
		if self.zone == nil && len(networks[i].AvailabilityDomain) > 0 || (self.zone != nil && networks[i].AvailabilityDomain != self.zone.Name) {
			continue
		}
		networks[i].wire = self
		ret = append(ret, &networks[i])
	}
	return ret, nil
}
