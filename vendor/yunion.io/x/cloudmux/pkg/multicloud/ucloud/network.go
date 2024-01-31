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

package ucloud

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/rbacscope"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

// https://docs.ucloud.cn/api/vpc2.0-api/describe_subnet
type SNetwork struct {
	multicloud.SNetworkBase
	UcloudTags
	wire *SWire

	CreateTime   int64  `json:"CreateTime"`
	Gateway      string `json:"Gateway"`
	HasNATGW     bool   `json:"HasNATGW"`
	Name         string `json:"Name"`
	Netmask      string `json:"Netmask"`
	Remark       string `json:"Remark"`
	RouteTableID string `json:"RouteTableId"`
	Subnet       string `json:"Subnet"`
	SubnetID     string `json:"SubnetId"`
	SubnetName   string `json:"SubnetName"`
	SubnetType   int    `json:"SubnetType"`
	Tag          string `json:"Tag"`
	VPCID        string `json:"VPCId"`
	VPCName      string `json:"VPCName"`
	VRouterID    string `json:"VRouterId"`
	Zone         string `json:"Zone"`
}

func (self *SNetwork) GetProjectId() string {
	return self.wire.region.client.projectId
}

func (self *SNetwork) GetId() string {
	return self.SubnetID
}

func (self *SNetwork) GetName() string {
	if len(self.SubnetName) > 0 {
		return self.SubnetName
	}

	return self.GetId()
}

func (self *SNetwork) GetGlobalId() string {
	return self.GetId()
}

func (self *SNetwork) GetStatus() string {
	return api.NETWORK_STATUS_AVAILABLE
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
	pref, _ := netutils.NewIPV4Prefix(self.Subnet + "/" + self.Netmask)
	startIp := pref.Address.NetAddr(pref.MaskLen) // 0
	startIp = startIp.StepUp()                    // 1 gateway
	startIp = startIp.StepUp()                    // 2
	return startIp.String()
}

func (self *SNetwork) GetIpEnd() string {
	pref, _ := netutils.NewIPV4Prefix(self.Subnet + "/" + self.Netmask)
	endIp := pref.Address.BroadcastAddr(pref.MaskLen) // 255
	endIp = endIp.StepDown()                          // 254
	return endIp.String()
}

func (self *SNetwork) GetIpMask() int8 {
	pref, _ := netutils.NewIPV4Prefix(self.Subnet + "/" + self.Netmask)
	return pref.MaskLen
}

func (self *SNetwork) GetGateway() string {
	return self.Gateway
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
	return self.wire.region.DeleteNetwork(self.GetId())
}

func (self *SNetwork) GetAllocTimeoutSeconds() int {
	return 120 // 2 minutes
}

// https://docs.ucloud.cn/api/vpc2.0-api/describe_subnet
func (self *SRegion) getNetwork(networkId string) (*SNetwork, error) {
	if len(networkId) == 0 {
		return nil, fmt.Errorf("getNetwork network id should not be empty")
	}
	networks := make([]SNetwork, 0)

	params := NewUcloudParams()
	params.Set("SubnetId", networkId)
	err := self.DoListAll("DescribeSubnet", params, &networks)
	if err != nil {
		return nil, err
	}

	if len(networks) == 1 {
		network := networks[0]
		vpc, err := self.getVpc(network.VPCID)
		if err != nil {
			return nil, err
		}
		network.wire = &SWire{region: self, vpc: vpc, inetworks: []cloudprovider.ICloudNetwork{&network}}
		return &network, nil
	} else if len(networks) == 0 {
		return nil, cloudprovider.ErrNotFound
	} else {
		return nil, fmt.Errorf("getNetwork %s %d found", networkId, len(networks))
	}
}

// https://docs.ucloud.cn/api/vpc2.0-api/delete_subnet
func (self *SRegion) DeleteNetwork(networkId string) error {
	params := NewUcloudParams()
	params.Set("SubnetId", networkId)

	return self.DoAction("DeleteSubnet", params, nil)
}

// https://docs.ucloud.cn/api/vpc2.0-api/create_subnet
func (self *SRegion) CreateNetwork(vpcId string, name string, cidr string, desc string) (*SNetwork, error) {
	ip, mask, err := netutils.ParsePrefix(cidr)
	if err != nil {
		return nil, fmt.Errorf("CreateINetwork invalid cidr %s", cidr)
	}

	params := NewUcloudParams()
	params.Set("VPCId", vpcId)
	params.Set("Subnet", ip.String())
	params.Set("Netmask", int(mask))
	params.Set("SubnetName", name)
	params.Set("Remark", desc)

	type SNet struct {
		SubnetId string
	}

	net := SNet{}
	err = self.DoAction("CreateSubnet", params, &net)
	if err != nil {
		return nil, err
	}

	return self.getNetwork(net.SubnetId)
}
