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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

// https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_Subnet.html
type SNetwork struct {
	multicloud.SResourceBase
	multicloud.AwsTags
	wire *SWire

	AssignIpv6AddressOnCreation bool   `xml:"assignIpv6AddressOnCreation"`
	AvailabilityZone            string `xml:"availabilityZone"`
	AvailabilityZoneId          string `xml:"availabilityZoneId"`
	AvailableIpAddressCount     int    `xml:"availableIpAddressCount"`
	CidrBlock                   string `xml:"cidrBlock"`
	CustomerOwnedIpv4Pools      string `xml:"customerOwnedIpv4Pool"`
	DefaultForAz                bool   `xml:"defaultForAz"`
	Ipv6CidrBlockAssociationSet []struct {
		AssociationId      string `xml:"associationId"`
		Ipv6CidrBlock      string `xml:"ipv6CidrBlock"`
		Ipv6CidrBlockState struct {
			State         string `xml:"state"`
			StatusMessage string `xml:"statusMessage"`
		}
	} `xml:"ipv6CidrBlockAssociationSet>item"`
	MapCustomerOwnedIpOnLaunch bool   `xml:"mapCustomerOwnedIpOnLaunch"`
	MapPublicIpOnLaunch        bool   `xml:"mapPublicIpOnLaunch"`
	OutpostArn                 string `xml:"outpostArn"`
	OwnerId                    string `xml:"ownerId"`
	State                      string `xml:"state"`
	SubnetArn                  string `xml:"subnetArn"`
	SubnetId                   string `xml:"subnetId"`
	VpcId                      string `xml:"vpcId"`
}

func (self *SNetwork) GetId() string {
	return self.SubnetId
}

func (self *SNetwork) GetName() string {
	name := self.AwsTags.GetName()
	if len(name) > 0 {
		return name
	}
	return self.SubnetId
}

func (self *SNetwork) GetGlobalId() string {
	return self.SubnetId
}

func (self *SNetwork) GetStatus() string {
	switch self.State {
	case "pending":
		return api.NETWORK_STATUS_PENDING
	case "available":
		return api.NETWORK_STATUS_AVAILABLE
	}
	return self.State
}

func (self *SNetwork) Refresh() error {
	net, err := self.wire.zone.region.GetNetwork(self.SubnetId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, net)
}

func (self *SNetwork) GetSysTags() map[string]string {
	data := map[string]string{}
	routes, _ := self.wire.vpc.region.GetRouteTablesByNetworkId(self.GetId())
	if len(routes) == 0 {
		routes, _ = self.wire.vpc.region.GetRouteTables(self.VpcId, "", "", nil, true)
	}

	support_eip := false
	if len(routes) >= 1 {
		for i := range routes[0].RouteSet {
			route := routes[0].RouteSet[i]
			if route.GetNextHopType() == api.Next_HOP_TYPE_INTERNET {
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

func (self *SNetwork) GetPublicScope() rbacutils.TRbacScope {
	return rbacutils.ScopeDomain
}

func (self *SNetwork) Delete() error {
	return self.wire.zone.region.DeleteNetwork(self.SubnetId)
}

func (self *SNetwork) GetAllocTimeoutSeconds() int {
	return 120 // 2 minutes
}

func (self *SRegion) CreateNetwork(zoneId string, vpcId string, name string, cidr string, desc string) (*SNetwork, error) {
	params := map[string]string{
		"AvailabilityZone":                zoneId,
		"VpcId":                           vpcId,
		"CidrBlock":                       cidr,
		"TagSpecification.1.ResourceType": "subnet",
		"TagSpecification.1.Tags.1.Key":   "Name",
		"TagSpecification.1.Tags.1.Value": name,
	}
	if len(desc) > 0 {
		params["TagSpecification.1.Tags.2.Key"] = "Description"
		params["TagSpecification.1.Tags.2.Value"] = desc
	}

	ret := struct {
		Network SNetwork `xml:"subnet"`
	}{}
	err := self.ec2Request("CreateSubnet", params, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateSubnet")
	}
	return &ret.Network, nil
}

func (self *SRegion) GetNetwork(id string) (*SNetwork, error) {
	networks, err := self.GetNetwroks([]string{id}, "")
	if err != nil {
		return nil, errors.Wrap(err, "GetNetwroks")
	}
	for i := range networks {
		if networks[i].SubnetId == id {
			return &networks[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SRegion) DeleteNetwork(id string) error {
	params := map[string]string{
		"SubnetId": id,
	}
	return self.ec2Request("DeleteSubnet", params, nil)
}

func (self *SRegion) GetNetwroks(ids []string, vpcId string) ([]SNetwork, error) {
	params := map[string]string{}
	for i, id := range ids {
		params[fmt.Sprintf("SubnetId.%d", i+1)] = id
	}
	if len(vpcId) > 0 {
		params["Filter.1.vpc-id"] = vpcId
	}
	ret := []SNetwork{}
	for {
		result := struct {
			NextToken string     `xml:"nextToken"`
			Networks  []SNetwork `xml:"subnetSet>item"`
		}{}
		err := self.ec2Request("DescribeSubnets", params, &result)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeSubnets")
		}
		ret = append(ret, result.Networks...)
		if len(result.NextToken) == 0 || len(result.Networks) == 0 {
			break
		}
		params["NextToken"] = result.NextToken
	}
	return ret, nil
}

func (self *SNetwork) GetProjectId() string {
	return ""
}
