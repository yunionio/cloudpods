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

package google

import (
	"fmt"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SWire struct {
	multicloud.SResourceBase
	multicloud.GoogleTags
	vpc *SVpc
}

func (wire *SWire) GetId() string {
	return wire.vpc.GetGlobalId()
}

func (wire *SWire) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", wire.GetId(), wire.vpc.region.Name)
}

func (wire *SWire) GetName() string {
	return wire.vpc.GetName()
}

func (wire *SWire) CreateINetwork(opts *cloudprovider.SNetworkCreateOptions) (cloudprovider.ICloudNetwork, error) {
	network, err := wire.vpc.region.CreateNetwork(opts.Name, wire.vpc.globalnetwork.SelfLink, opts.Cidr, opts.Desc)
	if err != nil {
		return nil, err
	}
	network.wire = wire
	return network, nil
}

func (wire *SWire) GetIVpc() cloudprovider.ICloudVpc {
	return wire.vpc
}

func (wire *SWire) GetIZone() cloudprovider.ICloudZone {
	return nil
}

func (wire *SWire) GetINetworks() ([]cloudprovider.ICloudNetwork, error) {
	networks, err := wire.vpc.region.GetNetworks(wire.vpc.globalnetwork.SelfLink, 0, "")
	if err != nil {
		return nil, err
	}
	inetworks := []cloudprovider.ICloudNetwork{}
	for i := range networks {
		networks[i].wire = wire
		inetworks = append(inetworks, &networks[i])
	}
	return inetworks, nil
}

func (wire *SWire) GetINetworkById(id string) (cloudprovider.ICloudNetwork, error) {
	network, err := wire.vpc.region.GetNetwork(id)
	if err != nil {
		return nil, err
	}
	if network.Network != wire.vpc.globalnetwork.SelfLink {
		return nil, cloudprovider.ErrNotFound
	}
	network.wire = wire
	return network, nil
}

func (wire *SWire) GetBandwidth() int {
	return 0
}

func (wire *SWire) GetStatus() string {
	return api.WIRE_STATUS_AVAILABLE
}

func (wire *SWire) IsEmulated() bool {
	return true
}

func (wire *SWire) Refresh() error {
	return nil
}
