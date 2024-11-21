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

package baidu

import (
	"fmt"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SWire struct {
	multicloud.SResourceBase
	SBaiduTag

	zone *SZone
	vpc  *SVpc
}

func (wire *SWire) GetId() string {
	return fmt.Sprintf("%s-%s", wire.vpc.GetId(), wire.zone.GetId())
}

func (wire *SWire) GetName() string {
	return wire.GetId()
}

func (wire *SWire) IsEmulated() bool {
	return true
}

func (wire *SWire) GetStatus() string {
	return api.WIRE_STATUS_AVAILABLE
}

func (wire *SWire) Refresh() error {
	return nil
}

func (wire *SWire) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", wire.vpc.GetGlobalId(), wire.zone.GetGlobalId())
}

func (wire *SWire) GetIVpc() cloudprovider.ICloudVpc {
	return wire.vpc
}

func (wire *SWire) GetIZone() cloudprovider.ICloudZone {
	return wire.zone
}

func (wire *SWire) GetINetworks() ([]cloudprovider.ICloudNetwork, error) {
	networks, err := wire.vpc.region.GetNetworks(wire.vpc.VpcId, wire.zone.ZoneName)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudNetwork{}
	for i := range networks {
		networks[i].wire = wire
		ret = append(ret, &networks[i])
	}
	return ret, nil
}

func (wire *SWire) GetBandwidth() int {
	return 10000
}

func (wire *SWire) CreateINetwork(opts *cloudprovider.SNetworkCreateOptions) (cloudprovider.ICloudNetwork, error) {
	net, err := wire.vpc.region.CreateNetwork(wire.zone.ZoneName, wire.vpc.VpcId, opts)
	if err != nil {
		return nil, err
	}
	net.wire = wire
	return net, nil
}

func (wire *SWire) GetINetworkById(id string) (cloudprovider.ICloudNetwork, error) {
	net, err := wire.vpc.region.GetNetwork(id)
	if err != nil {
		return nil, err
	}
	net.wire = wire
	return net, nil
}
