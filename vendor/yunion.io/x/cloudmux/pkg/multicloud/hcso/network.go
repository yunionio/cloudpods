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

package hcso

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/rbacscope"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/cloudmux/pkg/multicloud/hcso/client/modules"
	"yunion.io/x/cloudmux/pkg/multicloud/huawei"
)

/*
Subnets
*/

// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0020090590.html
type SNetwork struct {
	multicloud.SNetworkBase
	huawei.HuaweiTags
	wire *SWire

	AvailabilityZone string   `json:"availability_zone"`
	CIDR             string   `json:"cidr"`
	DHCPEnable       bool     `json:"dhcp_enable"`
	DNSList          []string `json:"dnsList"`
	GatewayIP        string   `json:"gateway_ip"`
	ID               string   `json:"id"`
	Ipv6Enable       bool     `json:"ipv6_enable"`
	Name             string   `json:"name"`
	NeutronNetworkID string   `json:"neutron_network_id"`
	NeutronSubnetID  string   `json:"neutron_subnet_id"`
	PrimaryDNS       string   `json:"primary_dns"`
	SecondaryDNS     string   `json:"secondary_dns"`
	Status           string   `json:"status"`
	VpcID            string   `json:"vpc_id"`
}

func (self *SNetwork) GetId() string {
	return self.ID
}

func (self *SNetwork) GetName() string {
	if len(self.Name) == 0 {
		return self.ID
	}

	return self.Name
}

func (self *SNetwork) GetGlobalId() string {
	return self.ID
}

// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0020090591.html
func (self *SNetwork) GetStatus() string {
	switch self.Status {
	case "ACTIVE", "UNKNOWN":
		return api.NETWORK_STATUS_AVAILABLE // ? todo: // UNKNOWN
	case "ERROR":
		return api.NETWORK_STATUS_UNKNOWN
	default:
		return api.NETWORK_STATUS_UNKNOWN
	}
}

func (self *SNetwork) Refresh() error {
	log.Debugf("network refresh %s", self.GetId())
	new, err := self.wire.region.getNetwork(self.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SNetwork) IsEmulated() bool {
	return false
}

func (self *SNetwork) GetIWire() cloudprovider.ICloudWire {
	return self.wire
}

func (self *SNetwork) GetIpStart() string {
	pref, _ := netutils.NewIPV4Prefix(self.CIDR)
	startIp := pref.Address.NetAddr(pref.MaskLen) // 0
	startIp = startIp.StepUp()                    // 1
	startIp = startIp.StepUp()                    // 2
	return startIp.String()
}

func (self *SNetwork) GetIpEnd() string {
	pref, _ := netutils.NewIPV4Prefix(self.CIDR)
	endIp := pref.Address.BroadcastAddr(pref.MaskLen) // 255
	endIp = endIp.StepDown()                          // 254
	endIp = endIp.StepDown()                          // 253
	endIp = endIp.StepDown()                          // 252
	return endIp.String()
}

func (self *SNetwork) GetIpMask() int8 {
	pref, _ := netutils.NewIPV4Prefix(self.CIDR)
	return pref.MaskLen
}

func (self *SNetwork) GetGateway() string {
	pref, _ := netutils.NewIPV4Prefix(self.CIDR)
	startIp := pref.Address.NetAddr(pref.MaskLen) // 0
	startIp = startIp.StepUp()                    // 1
	return startIp.String()
}

func (self *SNetwork) GetServerType() string {
	return api.NETWORK_TYPE_GUEST
}

func (self *SNetwork) GetIsPublic() bool {
	return true
}

func (self *SNetwork) GetPublicScope() rbacscope.TRbacScope {
	return rbacscope.ScopeDomain
}

func (self *SNetwork) Delete() error {
	return self.wire.region.deleteNetwork(self.VpcID, self.GetId())
}

func (self *SNetwork) GetAllocTimeoutSeconds() int {
	return 120 // 2 minutes
}

func (self *SRegion) getNetwork(networkId string) (*SNetwork, error) {
	network := SNetwork{}
	err := DoGet(self.ecsClient.Subnets.Get, networkId, nil, &network)
	return &network, err
}

// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0020090592.html
func (self *SRegion) GetNetwroks(vpcId string) ([]SNetwork, error) {
	querys := map[string]string{}
	if len(vpcId) > 0 {
		querys["vpc_id"] = vpcId
	}

	networks := make([]SNetwork, 0)
	err := doListAllWithMarker(self.ecsClient.Subnets.List, querys, &networks)
	return networks, err
}

func (self *SRegion) deleteNetwork(vpcId string, networkId string) error {
	ctx := &modules.SManagerContext{InstanceId: vpcId, InstanceManager: self.ecsClient.Vpcs}
	return DoDeleteWithSpec(self.ecsClient.Subnets.DeleteInContextWithSpec, ctx, networkId, "", nil, nil)
}

func (self *SNetwork) GetProjectId() string {
	return self.wire.vpc.EnterpriseProjectID
}
