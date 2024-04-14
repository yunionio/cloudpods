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

package ksyun

import (
	"fmt"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"

	"yunion.io/x/pkg/errors"
)

type SWire struct {
	multicloud.SResourceBase
	SKsTag
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
	return fmt.Sprintf("%s-%s", wire.vpc.GetGlobalId(), wire.zone.GetId())
}

func (wire *SWire) GetIVpc() cloudprovider.ICloudVpc {
	return wire.vpc
}

func (wire *SWire) GetIZone() cloudprovider.ICloudZone {
	return wire.zone
}

func (wire *SWire) GetINetworks() ([]cloudprovider.ICloudNetwork, error) {
	networks, err := wire.vpc.region.GetNetworks([]string{wire.vpc.VpcID}, []string{}, wire.zone.GetName())
	if err != nil {
		return nil, errors.Wrap(err, "GetNetworks")
	}
	ret := []cloudprovider.ICloudNetwork{}
	for i := 0; i < len(networks); i++ {
		networks[i].wire = wire
		ret = append(ret, &networks[i])
	}
	return ret, nil
}

func (wire *SWire) GetBandwidth() int {
	return 10000
}

func (wire *SWire) CreateINetwork(opts *cloudprovider.SNetworkCreateOptions) (cloudprovider.ICloudNetwork, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (wire *SWire) GetINetworkById(netid string) (cloudprovider.ICloudNetwork, error) {
	network, err := wire.vpc.region.GetNetwork(netid)
	if err != nil {
		return nil, errors.Wrap(err, "GetNetwork")
	}
	network.wire = wire
	return network, nil
}

func (wire *SWire) GetDescription() string {
	return ""
}
