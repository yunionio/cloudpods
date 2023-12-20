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

package huawei

import (
	"fmt"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

// 华为云的子网有点特殊。子网在整个region可用。
type SWire struct {
	multicloud.SResourceBase
	HuaweiTags

	vpc *SVpc
}

func (self *SWire) GetId() string {
	return fmt.Sprintf("%s-%s", self.vpc.GetId(), self.vpc.region.GetId())
}

func (self *SWire) GetName() string {
	return self.GetId()
}

func (self *SWire) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", self.vpc.GetGlobalId(), self.vpc.region.GetGlobalId())
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

func (self *SWire) GetIVpc() cloudprovider.ICloudVpc {
	return self.vpc
}

func (self *SWire) GetIZone() cloudprovider.ICloudZone {
	return nil
}

func (self *SWire) GetINetworks() ([]cloudprovider.ICloudNetwork, error) {
	networks, err := self.vpc.region.GetNetworks(self.vpc.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "GetNetwroks")
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

func (self *SWire) GetINetworkById(id string) (cloudprovider.ICloudNetwork, error) {
	network, err := self.vpc.region.GetNetwork(id)
	if err != nil {
		return nil, err
	}
	network.wire = self
	return network, nil
}

func (self *SWire) CreateINetwork(opts *cloudprovider.SNetworkCreateOptions) (cloudprovider.ICloudNetwork, error) {
	network, err := self.vpc.region.CreateNetwork(self.vpc.ID, opts)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateNetwork")
	}

	network.wire = self
	return network, nil
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

// https://console.huaweicloud.com/apiexplorer/#/openapi/VPC/doc?version=v2&api=ListSubnets
func (self *SRegion) CreateNetwork(vpcId string, opts *cloudprovider.SNetworkCreateOptions) (*SNetwork, error) {
	gateway, err := getDefaultGateWay(opts.Cidr)
	if err != nil {
		return nil, err
	}

	params := map[string]interface{}{
		"name":        opts.Name,
		"description": opts.Desc,
		"vpc_id":      vpcId,
		"cidr":        opts.Cidr,
		"gateway_ip":  gateway,
	}
	resp, err := self.post(SERVICE_VPC, "subnets", map[string]interface{}{"subnet": params})
	if err != nil {
		return nil, errors.Wrapf(err, "create subnet")
	}
	subnet := &SNetwork{}
	err = resp.Unmarshal(subnet, "subnet")
	if err != nil {
		return nil, err
	}
	return subnet, nil
}
