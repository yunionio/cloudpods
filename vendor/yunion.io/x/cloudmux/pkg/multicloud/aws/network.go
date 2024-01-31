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

package aws

import (
	"fmt"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/rbacscope"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SNetwork struct {
	multicloud.SNetworkBase
	AwsTags
	wire *SWire

	AvailableIpAddressCount int    `xml:"availableIpAddressCount"`
	CidrBlock               string `xml:"cidrBlock"`
	DefaultForAz            bool   `xml:"DefaultForAz"`
	State                   string `xml:"state"`
	SubnetId                string `xml:"subnetId"`
	VpcId                   string `xml:"vpcId"`
	AvailabilityZone        string `xml:"availabilityZone"`
}

func (self *SNetwork) GetId() string {
	return self.SubnetId
}

func (self *SNetwork) GetName() string {
	if name := self.AwsTags.GetName(); len(name) > 0 {
		return name
	}
	return self.SubnetId
}

func (self *SNetwork) GetGlobalId() string {
	return self.SubnetId
}

func (self *SNetwork) GetStatus() string {
	if self.wire != nil && self.wire.vpc != nil && self.wire.vpc.InstanceTenancy == "dedicated" {
		return api.NETWORK_STATUS_UNAVAILABLE
	}

	return strings.ToLower(self.State)
}

func (self *SNetwork) Refresh() error {
	net, err := self.wire.zone.region.getNetwork(self.SubnetId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, net)
}

func (self *SNetwork) GetSysTags() map[string]string {
	data := map[string]string{}
	routes, _ := self.wire.vpc.region.GetRouteTables("", self.SubnetId, "", "", false)
	if len(routes) == 0 {
		routes, _ = self.wire.vpc.region.GetRouteTables(self.VpcId, "", "", "", true)
	}
	support_eip := false
	if len(routes) >= 1 {
		for i := range routes[0].Routes {
			route := routes[0].Routes[i]
			if route.GetNextHopType() == api.NEXT_HOP_TYPE_INTERNET {
				support_eip = true
			}
		}
	}
	data["support_eip"] = strconv.FormatBool(support_eip)
	return data
}

func (self *SNetwork) GetIWire() cloudprovider.ICloudWire {
	return self.wire
}

// 每个子网 CIDR 块中的前四个 IP 地址和最后一个 IP 地址无法使用
func (self *SNetwork) GetIpStart() string {
	pref, _ := netutils.NewIPV4Prefix(self.CidrBlock)
	startIp := pref.Address.NetAddr(pref.MaskLen) // 0
	startIp = startIp.StepUp()                    // 1
	startIp = startIp.StepUp()                    // 2
	startIp = startIp.StepUp()                    // 3
	startIp = startIp.StepUp()                    // 4
	return startIp.String()
}

func (self *SNetwork) GetIpEnd() string {
	pref, _ := netutils.NewIPV4Prefix(self.CidrBlock)
	endIp := pref.Address.BroadcastAddr(pref.MaskLen) // 255
	endIp = endIp.StepDown()                          // 254
	return endIp.String()
}

func (self *SNetwork) GetIpMask() int8 {
	pref, _ := netutils.NewIPV4Prefix(self.CidrBlock)
	return pref.MaskLen
}

func (self *SNetwork) GetGateway() string {
	pref, _ := netutils.NewIPV4Prefix(self.CidrBlock)
	endIp := pref.Address.BroadcastAddr(pref.MaskLen) // 255
	endIp = endIp.StepDown()                          // 254
	return endIp.String()
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
	return self.wire.zone.region.DeleteNetwork(self.SubnetId)
}

func (self *SNetwork) GetAllocTimeoutSeconds() int {
	return 120 // 2 minutes
}

func (self *SRegion) ModifySubnetAttribute(subnetId string, assignPublicIp bool) error {
	params := map[string]string{
		"SubnetId":                  subnetId,
		"MapPublicIpOnLaunch.Value": "false",
	}
	if assignPublicIp {
		params["MapPublicIpOnLaunch.Value"] = "true"
	}
	ret := struct {
		Return bool `xml:"return"`
	}{}

	return self.ec2Request("ModifySubnetAttribute", params, &ret)
}

func (self *SRegion) CreateNetwork(zoneId string, vpcId string, name string, cidr, desc string) (*SNetwork, error) {
	params := map[string]string{
		"AvailabilityZone":                zoneId,
		"VpcId":                           vpcId,
		"CidrBlock":                       cidr,
		"TagSpecification.1.ResourceType": "subnet",
		"TagSpecification.1.Tag.1.Key":    "Name",
		"TagSpecification.1.Tag.1.Value":  name,
	}
	if len(desc) > 0 {
		params["TagSpecification.1.Tag.2.Key"] = "Description"
		params["TagSpecification.1.Tag.2.Value"] = desc
	}

	ret := struct {
		Subnet SNetwork `xml:"subnet"`
	}{}

	err := self.ec2Request("CreateSubnet", params, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateSubnet")
	}
	return &ret.Subnet, nil
}

func (self *SRegion) getNetwork(networkId string) (*SNetwork, error) {
	networks, err := self.GetNetwroks([]string{networkId}, "", "")
	if err != nil {
		return nil, errors.Wrap(err, "GetNetwroks")
	}
	for i := range networks {
		if networks[i].GetGlobalId() == networkId {
			return &networks[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, networkId)
}

func (self *SRegion) DeleteNetwork(id string) error {
	params := map[string]string{
		"SubnetId": id,
	}
	return self.ec2Request("DeleteSubnet", params, nil)
}

func (self *SRegion) GetNetwroks(ids []string, zoneId, vpcId string) ([]SNetwork, error) {
	params := map[string]string{}
	idx := 1
	if len(vpcId) > 0 {
		params[fmt.Sprintf("Filter.%d.Name", idx)] = "vpc-id"
		params[fmt.Sprintf("Filter.%d.Value.1", idx)] = vpcId
		idx++
	}
	if len(zoneId) > 0 {
		params[fmt.Sprintf("Filter.%d.Name", idx)] = "availability-zone"
		params[fmt.Sprintf("Filter.%d.Value.1", idx)] = zoneId
		idx++
	}
	for i, id := range ids {
		params[fmt.Sprintf("SubnetId.%d", i+1)] = id
	}

	ret := []SNetwork{}
	for {
		part := struct {
			NextToken string     `xml:"nextToken"`
			SubnetSet []SNetwork `xml:"subnetSet>item"`
		}{}
		err := self.ec2Request("DescribeSubnets", params, &part)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeSubnets")
		}
		ret = append(ret, part.SubnetSet...)
		if len(part.NextToken) == 0 || len(part.SubnetSet) == 0 {
			break
		}
		params["NextToken"] = part.NextToken
	}
	return ret, nil
}

func (self *SNetwork) GetProjectId() string {
	return ""
}

func (self *SNetwork) GetDescription() string {
	return self.AwsTags.GetDescription()
}
