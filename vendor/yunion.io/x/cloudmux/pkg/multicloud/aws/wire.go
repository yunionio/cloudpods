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

package aws

import (
	"fmt"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SWire struct {
	multicloud.SResourceBase
	AwsTags
	zone *SZone
	vpc  *SVpc
}

func (self *SWire) GetId() string {
	return fmt.Sprintf("%s-%s", self.vpc.GetId(), self.zone.GetId())
}

func (self *SWire) GetName() string {
	return self.GetId()
}

func (self *SWire) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", self.vpc.GetGlobalId(), self.zone.GetGlobalId())
}

func (self *SWire) GetStatus() string {
	return api.WIRE_STATUS_AVAILABLE
}

func (self *SWire) IsEmulated() bool {
	return true
}

func (self *SWire) GetIVpc() cloudprovider.ICloudVpc {
	return self.vpc
}

func (self *SWire) GetIZone() cloudprovider.ICloudZone {
	return self.zone
}

func (self *SWire) GetINetworks() ([]cloudprovider.ICloudNetwork, error) {
	networks, err := self.vpc.region.GetNetwroks(nil, self.zone.ZoneName, self.vpc.VpcId)
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
	networks, err := self.vpc.region.GetNetwroks([]string{netid}, self.zone.ZoneName, self.vpc.VpcId)
	if err != nil {
		return nil, err
	}
	for i := range networks {
		networks[i].wire = self
		if networks[i].GetGlobalId() == netid || networks[i].GetId() == netid {
			return &networks[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "GetINetworkById %s", netid)
}

func (self *SWire) CreateINetwork(opts *cloudprovider.SNetworkCreateOptions) (cloudprovider.ICloudNetwork, error) {
	network, err := self.zone.region.CreateNetwork(self.zone.ZoneName, self.vpc.VpcId, opts.Name, opts.Cidr, opts.Desc)
	if err != nil {
		return nil, errors.Wrap(err, "CreateNetwork")
	}
	network.wire = self
	if opts.AssignPublicIp {
		self.zone.region.ModifySubnetAttribute(network.SubnetId, opts.AssignPublicIp)
	}
	return network, nil
}

func (self *SWire) GetDescription() string {
	return self.AwsTags.GetDescription()
}
