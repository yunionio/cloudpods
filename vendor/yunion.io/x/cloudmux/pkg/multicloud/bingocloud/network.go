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

package bingocloud

import (
	"fmt"
	"strings"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/rbacscope"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SNetwork struct {
	multicloud.SNetworkBase
	multicloud.STagBase
	wire *SWire

	CidrBlock               string `json:"cidrBlock"`
	MapPublicIPOnLaunch     string `json:"mapPublicIpOnLaunch"`
	RouterId                string `json:"routerId"`
	NetworkFrom             string `json:"networkFrom"`
	NetworkNode             string `json:"networkNode"`
	IsManagedNetwork        string `json:"isManagedNetwork"`
	SubnetName              string `json:"subnetName"`
	NetworkMask             string `json:"networkMask"`
	NetName                 string `json:"netName"`
	Index                   string `json:"index"`
	VpcIsPublic             string `json:"vpcIsPublic"`
	Description             string `json:"description"`
	MicroSegmentation       string `json:"microSegmentation"`
	State                   string `json:"state"`
	DefaultForAz            string `json:"defaultForAz"`
	Router                  string `json:"router"`
	Active                  string `json:"active"`
	VpcShared               string `json:"vpcShared"`
	DvswitchName            string `json:"dvswitchName"`
	AvailableIPAddressCount string `json:"availableIpAddressCount"`
	VlanNum                 string `json:"vlanNum"`
	RouterCc2               string `json:"router_cc2"`
	ResStatus               string `json:"res_status"`
	IPAddressCount          string `json:"ipAddressCount"`
	RouterCc                string `json:"router_cc"`
	VpcId                   string `json:"vpcId"`
	NetworkTo               string `json:"networkTo"`
	Broadcast               string `json:"broadcast"`
	UserName                string `json:"userName"`
	IPVersion               string `json:"ipVersion"`
	AvailabilityZone        string `json:"availabilityZone"`
	Network                 string `json:"network"`
	RouterMac               string `json:"router_mac"`
	CheckIP                 string `json:"checkIp"`
	FloatingPool            string `json:"floatingPool"`
	StaticPool              string `json:"staticPool"`
	DvsPortGroup            string `json:"dvsPortGroup"`
	SubnetId                string `json:"subnetId"`
}

func (self *SNetwork) GetId() string {
	return self.SubnetId
}

func (self *SNetwork) GetGlobalId() string {
	return self.SubnetId
}

func (self *SNetwork) GetName() string {
	if len(self.SubnetName) > 0 {
		return self.SubnetName
	}
	return self.SubnetId
}

func (self *SNetwork) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (self *SNetwork) GetAllocTimeoutSeconds() int {
	return 300
}

func (self *SNetwork) GetGateway() string {
	prefix, err := netutils.NewIPV4Prefix(self.CidrBlock)
	if err == nil {
		return prefix.Address.StepUp().String()
	}
	return ""
}

func (self *SNetwork) GetIpStart() string {
	pref, _ := netutils.NewIPV4Prefix(self.CidrBlock)
	startIp := pref.Address.NetAddr(pref.MaskLen) // 0
	startIp = startIp.StepUp()                    // 1
	return startIp.String()
}

func (self *SNetwork) GetIpEnd() string {
	pref, _ := netutils.NewIPV4Prefix(self.CidrBlock)
	endIp := pref.Address.BroadcastAddr(pref.MaskLen) // 255
	endIp = endIp.StepDown()                          // 254
	endIp = endIp.StepDown()                          // 253
	endIp = endIp.StepDown()                          // 252
	return endIp.String()
}

func (self *SNetwork) GetIpMask() int8 {
	pref, _ := netutils.NewIPV4Prefix(self.CidrBlock)
	return pref.MaskLen
}

func (self *SNetwork) GetIWire() cloudprovider.ICloudWire {
	return self.wire
}

func (self *SNetwork) GetProjectId() string {
	return ""
}

func (self *SNetwork) GetPublicScope() rbacscope.TRbacScope {
	return rbacscope.ScopeDomain
}

func (self *SNetwork) GetServerType() string {
	return api.NETWORK_TYPE_GUEST
}

func (self *SNetwork) GetStatus() string {
	return strings.ToLower(self.State)
}

func (self *SWire) CreateINetwork(opts *cloudprovider.SNetworkCreateOptions) (cloudprovider.ICloudNetwork, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SWire) GetINetworks() ([]cloudprovider.ICloudNetwork, error) {
	networks, err := self.vpc.region.GetNetworks("", self.cluster.ClusterId, self.vpc.VpcId)
	if err != nil {
		return nil, err
	}
	var ret []cloudprovider.ICloudNetwork
	for i := range networks {
		networks[i].wire = self
		ret = append(ret, &networks[i])
	}
	return ret, nil
}

func (self *SWire) GetINetworkById(id string) (cloudprovider.ICloudNetwork, error) {
	networks, err := self.GetINetworks()
	if err != nil {
		return nil, err
	}
	for i := range networks {
		if networks[i].GetGlobalId() == id {
			return networks[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SRegion) GetNetworks(id, clusterId, vpcId string) ([]SNetwork, error) {
	params := map[string]string{}
	if len(id) > 0 {
		params["SubnetId"] = id
	}
	idx := 1
	if len(clusterId) > 0 {
		params[fmt.Sprintf("Filter.%d.Name", idx)] = "availability-zone"
		params[fmt.Sprintf("Filter.%d.Value.1", idx)] = clusterId
		idx++
	}
	if len(vpcId) > 0 {
		params[fmt.Sprintf("Filter.%d.Name", idx)] = "vpc-id"
		params[fmt.Sprintf("Filter.%d.Value.1", idx)] = vpcId
		idx++
	}
	resp, err := self.invoke("DescribeSubnets", params)
	if err != nil {
		return nil, err
	}
	var networks []SNetwork
	return networks, resp.Unmarshal(&networks, "subnetSet")
}
