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

package hcs

import (
	"fmt"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

// 华为云的子网有点特殊。子网在整个region可用。
type SExternalWire struct {
	multicloud.SResourceBase
	HcsTags
	region *SRegion
	vpc    *SExternalVpc
}

func (self *SExternalWire) GetId() string {
	return fmt.Sprintf("%s-%s", self.vpc.GetId(), self.region.GetId())
}

func (self *SExternalWire) GetName() string {
	return self.GetId()
}

func (self *SExternalWire) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", self.vpc.GetGlobalId(), self.region.GetGlobalId())
}

func (self *SExternalWire) IsEmulated() bool {
	return true
}

func (self *SExternalWire) GetStatus() string {
	return api.WIRE_STATUS_AVAILABLE
}

func (self *SExternalWire) GetIVpc() cloudprovider.ICloudVpc {
	return self.vpc
}

func (self *SExternalWire) GetIZone() cloudprovider.ICloudZone {
	return nil
}

func (self *SExternalWire) GetINetworks() ([]cloudprovider.ICloudNetwork, error) {
	nets, err := self.region.GetExternalNetworks(self.vpc.Id)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudNetwork{}
	for i := range nets {
		nets[i].wire = self
		ret = append(ret, &nets[i])
	}
	return ret, nil
}

func (self *SExternalWire) GetBandwidth() int {
	return 10000
}

func (self *SExternalWire) GetINetworkById(id string) (cloudprovider.ICloudNetwork, error) {
	net, err := self.region.GetExternalNetwork(id)
	if err != nil {
		return nil, err
	}
	net.wire = self
	return net, nil
}

func (self *SExternalWire) CreateINetwork(opts *cloudprovider.SNetworkCreateOptions) (cloudprovider.ICloudNetwork, error) {
	return nil, cloudprovider.ErrNotSupported
}
