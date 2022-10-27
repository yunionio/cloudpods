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

package aliyun

import (
	"fmt"

	"yunion.io/x/log"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SWire struct {
	multicloud.SResourceBase
	AliyunTags
	zone *SZone
	vpc  *SVpc

	inetworks []cloudprovider.ICloudNetwork
}

func (self *SWire) GetId() string {
	return fmt.Sprintf("%s-%s", self.vpc.GetId(), self.zone.GetId())
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
	return fmt.Sprintf("%s-%s", self.vpc.GetGlobalId(), self.zone.GetGlobalId())
}

func (self *SWire) GetIVpc() cloudprovider.ICloudVpc {
	return self.vpc
}

func (self *SWire) GetIZone() cloudprovider.ICloudZone {
	return self.zone
}

func (self *SWire) addNetwork(vswitch *SVSwitch) {
	if self.inetworks == nil {
		self.inetworks = make([]cloudprovider.ICloudNetwork, 0)
	}
	find := false
	for i := 0; i < len(self.inetworks); i += 1 {
		if self.inetworks[i].GetId() == vswitch.VSwitchId {
			find = true
			break
		}
	}
	if !find {
		self.inetworks = append(self.inetworks, vswitch)
	}
}

func (self *SWire) GetINetworks() ([]cloudprovider.ICloudNetwork, error) {
	if self.inetworks == nil {
		err := self.vpc.fetchVSwitches()
		if err != nil {
			return nil, err
		}
	}
	return self.inetworks, nil
}

func (self *SWire) getNetworkById(vswitchId string) *SVSwitch {
	networks, err := self.GetINetworks()
	if err != nil {
		return nil
	}
	log.Debugf("search for networks %d", len(networks))
	for i := 0; i < len(networks); i += 1 {
		log.Debugf("search %s", networks[i].GetName())
		network := networks[i].(*SVSwitch)
		if network.VSwitchId == vswitchId {
			return network
		}
	}
	return nil
}

func (self *SWire) GetBandwidth() int {
	return 10000
}

func (self *SWire) CreateINetwork(opts *cloudprovider.SNetworkCreateOptions) (cloudprovider.ICloudNetwork, error) {
	vswitchId, err := self.zone.region.createVSwitch(self.zone.ZoneId, self.vpc.VpcId, opts.Name, opts.Cidr, opts.Desc)
	if err != nil {
		log.Errorf("createVSwitch error %s", err)
		return nil, err
	}
	self.inetworks = nil
	vswitch := self.getNetworkById(vswitchId)
	if vswitch == nil {
		log.Errorf("cannot find vswitch after create????")
		return nil, cloudprovider.ErrNotFound
	}
	return vswitch, nil
}

func (self *SWire) GetINetworkById(netid string) (cloudprovider.ICloudNetwork, error) {
	networks, err := self.GetINetworks()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(networks); i += 1 {
		if networks[i].GetGlobalId() == netid {
			return networks[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}
