// Copyright 2023 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package volcengine

import (
	"fmt"
	"time"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

type SWire struct {
	multicloud.SResourceBase
	VolcEngineTags

	zone      *SZone
	vpc       *SVpc
	inetworks []cloudprovider.ICloudNetwork
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

func (wire *SWire) GetBandwidth() int {
	return 10000
}

func (wire *SWire) GetINetworks() ([]cloudprovider.ICloudNetwork, error) {
	networks, err := wire.vpc.region.GetSubnets(nil, wire.zone.ZoneId, wire.vpc.VpcId)
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

func (wire *SWire) getNetworkById(SubnetId string) (*SNetwork, error) {
	networks, err := wire.GetINetworks()
	if err != nil {
		return nil, err
	}
	log.Debugf("search for networks %d", len(networks))
	for i := 0; i < len(networks); i += 1 {
		log.Debugf("search %s", networks[i].GetName())
		network := networks[i].(*SNetwork)
		if network.SubnetId == SubnetId {
			return network, nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (wire *SWire) CreateINetwork(opts *cloudprovider.SNetworkCreateOptions) (cloudprovider.ICloudNetwork, error) {
	subnetId, err := wire.zone.region.CreateSubnet(wire.zone.ZoneId, wire.vpc.VpcId, opts.Name, opts.Cidr, opts.Desc)
	if err != nil {
		log.Errorf("createSubnet error %s", err)
		return nil, err
	}
	err = cloudprovider.Wait(5*time.Second, time.Minute, func() (bool, error) {
		_, err = wire.getNetworkById(subnetId)
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			return false, nil
		} else {
			return true, err
		}
	})
	if err != nil {
		return nil, errors.Wrapf(err, "cannot find subnet after create")
	}
	subnet, err := wire.getNetworkById(subnetId)
	if err != nil {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "%s not found", subnetId)
	}
	subnet.wire = wire
	if wire.inetworks == nil {
		wire.inetworks = []cloudprovider.ICloudNetwork{}
	}
	wire.inetworks = append(wire.inetworks, subnet)
	return subnet, nil
}

func (wire *SWire) GetINetworkById(netid string) (cloudprovider.ICloudNetwork, error) {
	networks, err := wire.GetINetworks()
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
