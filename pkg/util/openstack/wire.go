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

package openstack

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SWire struct {
	zone *SZone
	vpc  *SVpc

	inetworks []cloudprovider.ICloudNetwork
}

func (wire *SWire) GetMetadata() *jsonutils.JSONDict {
	return nil
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
	return "available"
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

func (wire *SWire) GetBandwidth() int {
	return 10000
}

func (wire *SWire) CreateINetwork(name string, cidr string, desc string) (cloudprovider.ICloudNetwork, error) {
	networkId, err := wire.zone.region.CreateNetwork(wire.vpc.ID, name, cidr, desc)
	if err != nil {
		log.Errorf("CreateNetwork error %s", err)
		return nil, err
	}
	wire.inetworks = nil
	return wire.GetINetworkById(networkId)
}

func (wire *SWire) GetINetworkById(netid string) (cloudprovider.ICloudNetwork, error) {
	networks, err := wire.GetINetworks()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(networks); i++ {
		if networks[i].GetGlobalId() == netid {
			return networks[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (wire *SWire) GetINetworks() ([]cloudprovider.ICloudNetwork, error) {
	if wire.inetworks == nil {
		err := wire.vpc.fetchNetworks()
		if err != nil {
			return nil, err
		}
	}
	return wire.inetworks, nil
}

func (wire *SWire) addNetwork(network *SNetwork) {
	if wire.inetworks == nil {
		wire.inetworks = []cloudprovider.ICloudNetwork{}
	}
	find := false
	for i := 0; i < len(wire.inetworks); i++ {
		if wire.inetworks[i].GetGlobalId() == network.GetGlobalId() {
			find = true
			break
		}
	}
	if !find {
		wire.inetworks = append(wire.inetworks, network)
	}
}
