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

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SWire struct {
	vpc *SVpc
}

func (wire *SWire) GetId() string {
	return wire.vpc.GetGlobalId()
}

func (wire *SWire) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", getGlobalId(wire.GetId()), wire.vpc.region.Name)
}

func (wire *SWire) GetName() string {
	return wire.vpc.GetName()
}

func (wire *SWire) CreateINetwork(name string, cidr string, desc string) (cloudprovider.ICloudNetwork, error) {
	return nil, cloudprovider.ErrNotImplemented
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

func (wire *SWire) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (wire *SWire) GetStatus() string {
	return "available"
}

func (wire *SWire) IsEmulated() bool {
	return true
}

func (wire *SWire) Refresh() error {
	return nil
}
