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

package ctyun

import (
	"fmt"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SWire struct {
	multicloud.SResourceBase
	CtyunTags
	region *SRegion
	vpc    *SVpc

	inetworks []cloudprovider.ICloudNetwork
}

func (self *SWire) GetId() string {
	return fmt.Sprintf("%s-%s", self.vpc.GetId(), self.region.GetId())
}

func (self *SWire) GetName() string {
	return self.GetId()
}

func (self *SWire) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", self.vpc.GetGlobalId(), self.region.GetGlobalId())
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

//  http://ctyun-api-url/apiproxy/v3/queryVPCDetail
func (self *SWire) GetIVpc() cloudprovider.ICloudVpc {
	return self.vpc
}

func (self *SWire) GetIZone() cloudprovider.ICloudZone {
	return nil
}

// http://ctyun-api-url/apiproxy/v3/getSubnets
func (self *SWire) GetINetworks() ([]cloudprovider.ICloudNetwork, error) {
	if self.inetworks == nil {
		err := self.vpc.fetchNetworks()
		if err != nil {
			return nil, err
		}
	}
	return self.inetworks, nil
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

func getDefaultGateWay(cidr string) (string, error) {
	pref, err := netutils.NewIPV4Prefix(cidr)
	if err != nil {
		return "", errors.Wrap(err, "getDefaultGateWay.NewIPV4Prefix")
	}
	startIp := pref.Address.NetAddr(pref.MaskLen) // 0
	startIp = startIp.StepUp()                    // 1
	return startIp.String(), nil
}

func (self *SWire) CreateINetwork(opts *cloudprovider.SNetworkCreateOptions) (cloudprovider.ICloudNetwork, error) {
	var ZoneId string
	if len(self.inetworks) > 0 {
		ZoneId = self.inetworks[0].(*SNetwork).ZoneID
	} else {
		if len(self.region.izones) > 0 {
			ZoneId = self.region.izones[0].GetId()
		} else {
			return nil, fmt.Errorf("SWire.CreateINetwork region %s zone is empty", self.region.GetName())
		}
	}
	network, err := self.region.CreateNetwork(self.vpc.GetId(), ZoneId, opts.Name, opts.Cidr, "true")
	if err != nil {
		return nil, errors.Wrap(err, "SWire.CreateINetwork.CreateNetwork")
	}

	return network, nil
}

func (self *SWire) addNetwork(network *SNetwork) {
	if self.inetworks == nil {
		self.inetworks = make([]cloudprovider.ICloudNetwork, 0)
	}
	find := false
	for i := 0; i < len(self.inetworks); i += 1 {
		if self.inetworks[i].GetId() == network.GetId() {
			find = true
			break
		}
	}
	if !find {
		self.inetworks = append(self.inetworks, network)
	}
}
