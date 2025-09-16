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

package azure

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SWire struct {
	multicloud.SResourceBase
	AzureTags

	zone      *SZone
	vpc       *SVpc
	inetworks []cloudprovider.ICloudNetwork
}

func (self *SWire) GetId() string {
	return fmt.Sprintf("%s/%s/%s", self.zone.region.GetGlobalId(), self.zone.region.client.subscriptionId, self.vpc.GetName())
}

func (self *SWire) GetGlobalId() string {
	return strings.ToLower(self.GetId())
}

func (self *SWire) GetName() string {
	return fmt.Sprintf("%s-%s", self.zone.region.client.cpcfg.Name, self.vpc.GetName())
}

func (self *SWire) IsEmulated() bool {
	return true
}

func (self *SWire) GetStatus() string {
	return api.WIRE_STATUS_AVAILABLE
}

func (self *SRegion) CreateNetwork(vpcId, name string, cidr string, desc string) (*SNetwork, error) {
	params := map[string]interface{}{
		"Name": name,
		"Properties": map[string]interface{}{
			"AddressPrefix": cidr,
		},
	}
	resource := fmt.Sprintf("%s/subnets/%s", vpcId, name)
	network := &SNetwork{}
	resp, err := self.put(resource, jsonutils.Marshal(params))
	if err != nil {
		return nil, errors.Wrapf(err, "put(%s)", resource)
	}
	err = resp.Unmarshal(network)
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return network, nil
}

func (self *SWire) CreateINetwork(opts *cloudprovider.SNetworkCreateOptions) (cloudprovider.ICloudNetwork, error) {
	network, err := self.zone.region.CreateNetwork(self.vpc.ID, opts.Name, opts.Cidr, opts.Desc)
	if err != nil {
		return nil, err
	}
	network.wire = self
	return network, nil
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
		if networks[i].GetGlobalId() == strings.ToLower(netid) {
			return networks[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, netid)
}

func (self *SWire) GetINetworks() ([]cloudprovider.ICloudNetwork, error) {
	ret := []cloudprovider.ICloudNetwork{}
	networks := self.vpc.GetNetworks()
	for i := range networks {
		networks[i].wire = self
		ret = append(ret, &networks[i])
	}
	return ret, nil
}

func (self *SWire) GetIVpc() cloudprovider.ICloudVpc {
	return self.vpc
}

func (self *SWire) GetIZone() cloudprovider.ICloudZone {
	return self.zone
}
