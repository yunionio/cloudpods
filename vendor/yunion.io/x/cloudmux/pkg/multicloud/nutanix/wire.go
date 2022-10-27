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

package nutanix

import (
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SWire struct {
	multicloud.SResourceBase
	multicloud.STagBase

	vpc *SVpc
}

func (self *SWire) GetName() string {
	return self.vpc.GetName()
}

func (self *SWire) GetId() string {
	return self.vpc.GetId()
}

func (self *SWire) GetGlobalId() string {
	return self.vpc.GetGlobalId()
}

func (self *SWire) IsEmulated() bool {
	return true
}

func (self *SWire) CreateINetwork(opts *cloudprovider.SNetworkCreateOptions) (cloudprovider.ICloudNetwork, error) {
	network, err := self.vpc.region.CreateNetwork(self.vpc.UUID, opts)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateNetwork")
	}
	return network, nil
}

func (self *SWire) GetBandwidth() int {
	return 10000
}

func (self *SWire) GetINetworks() ([]cloudprovider.ICloudNetwork, error) {
	ret := []cloudprovider.ICloudNetwork{}
	if len(self.vpc.IPConfig.Pool) == 0 {
		network := &SNetwork{wire: self}
		ret = append(ret, network)
	}
	for _, pool := range self.vpc.IPConfig.Pool {
		network := &SNetwork{wire: self}
		network.Range = pool.Range
		ret = append(ret, network)
	}
	return ret, nil
}

func (self *SWire) GetINetworkById(netid string) (cloudprovider.ICloudNetwork, error) {
	networks, err := self.GetINetworks()
	if err != nil {
		return nil, err
	}

	for i := range networks {
		if networks[i].GetGlobalId() == netid {
			return networks[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SWire) GetIVpc() cloudprovider.ICloudVpc {
	return self.vpc
}

func (self *SWire) GetIZone() cloudprovider.ICloudZone {
	cluster := SCluster{}
	err := self.vpc.region.get("cluster", "", nil, &cluster)
	if err != nil {
		return nil
	}
	return &SZone{SCluster: cluster, region: self.vpc.region}
}

func (self *SWire) GetStatus() string {
	return api.WIRE_STATUS_AVAILABLE
}
