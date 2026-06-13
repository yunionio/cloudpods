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

package ucloud

import (
	"fmt"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

// 子网在整个region可用
type SWire struct {
	multicloud.SResourceBase
	UcloudTags
	vpc *SVPC
}

func (self *SWire) GetId() string {
	return fmt.Sprintf("%s-%s", self.vpc.GetId(), self.vpc.region.GetId())
}

func (self *SWire) GetName() string {
	return self.GetId()
}

func (self *SWire) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", self.vpc.GetGlobalId(), self.vpc.region.GetGlobalId())
}

func (self *SWire) GetStatus() string {
	return api.WIRE_STATUS_AVAILABLE
}

func (self *SWire) Refresh() error {
	return nil
}

func (self *SWire) IsEmulated() bool {
	return true
}

func (self *SWire) GetIVpc() cloudprovider.ICloudVpc {
	return self.vpc
}

func (self *SWire) GetIZone() cloudprovider.ICloudZone {
	return nil
}

func (self *SWire) GetINetworks() ([]cloudprovider.ICloudNetwork, error) {
	networks, err := self.vpc.region.GetNetworks(self.vpc.GetId())
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudNetwork{}
	for i := range networks {
		networks[i].wire = self
		ret = append(ret, &networks[i])
	}
	return ret, nil
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
		if networks[i].GetGlobalId() == netid {
			return networks[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

// https://docs.ucloud.cn/api/vpc2.0-api/create_subnet
func (self *SWire) CreateINetwork(opts *cloudprovider.SNetworkCreateOptions) (cloudprovider.ICloudNetwork, error) {
	return self.vpc.region.CreateNetwork(self.vpc.GetId(), opts.Name, opts.Cidr, opts.Desc)
}
