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

import "yunion.io/x/cloudmux/pkg/cloudprovider"

type SWire struct {
	SResourceBase

	region *SRegion

	WireId    string
	VpcId     string
	ZoneId    string
	Bandwidth int
}

func (self *SWire) GetIVpc() cloudprovider.ICloudVpc {
	vpcs, err := self.region.client.GetVpcs()
	if err != nil {
		return nil
	}
	for i := range vpcs {
		if vpcs[i].GetId() == self.VpcId {
			vpcs[i].region = self.region
			return &vpcs[i]
		}
	}
	return nil
}

func (self *SWire) GetIZone() cloudprovider.ICloudZone {
	zones, err := self.region.client.GetZones()
	if err != nil {
		return nil
	}
	for i := range zones {
		if zones[i].GetId() == self.ZoneId {
			zones[i].region = self.region
			return &zones[i]
		}
	}
	return nil
}

func (self *SWire) GetINetworks() ([]cloudprovider.ICloudNetwork, error) {
	networks, err := self.region.client.GetNetworks()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudNetwork{}
	for i := range networks {
		if networks[i].WireId != self.GetId() {
			continue
		}
		networks[i].wire = self
		ret = append(ret, &networks[i])
	}
	return ret, nil
}

func (self *SWire) GetBandwidth() int {
	return self.Bandwidth
}

func (self *SWire) GetINetworkById(id string) (cloudprovider.ICloudNetwork, error) {
	networks, err := self.GetINetworks()
	if err != nil {
		return nil, err
	}
	for i := range networks {
		if networks[i].GetGlobalId() == id {
			return networks[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SWire) CreateINetwork(opts *cloudprovider.SNetworkCreateOptions) (cloudprovider.ICloudNetwork, error) {
	return nil, cloudprovider.ErrNotSupported
}
