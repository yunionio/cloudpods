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
	"time"

	"github.com/aws/aws-sdk-go/service/ec2"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/rbacscope"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SNetwork struct {
	multicloud.SResourceBase
	AwsTags
	wire *SWire

	AvailableIpAddressCount int    `xml:"availableIpAddressCount"`
	CidrBlock               string `xml:"cidrBlock"`
	CreationTime            time.Time
	Description             string
	IsDefault               bool
	Status                  string
	SubnetId                string `xml:"subnetId"`
	NetworkName             string
	VpcId                   string `xml:"vpcId"`
	ZoneId                  string
}

func (self *SNetwork) GetId() string {
	return self.SubnetId
}

func (self *SNetwork) GetName() string {
	if len(self.NetworkName) == 0 {
		return self.SubnetId
	}
	return self.NetworkName
}

func (self *SNetwork) GetGlobalId() string {
	return self.SubnetId
}

func (self *SNetwork) GetStatus() string {
	if self.wire != nil && self.wire.vpc != nil && self.wire.vpc.InstanceTenancy == "dedicated" {
		return api.NETWORK_STATUS_UNAVAILABLE
	}

	return strings.ToLower(self.Status)
}

func (self *SNetwork) Refresh() error {
	new, err := self.wire.zone.region.getNetwork(self.SubnetId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SNetwork) GetSysTags() map[string]string {
	data := map[string]string{}
	routes, _ := self.wire.vpc.region.GetRouteTablesByNetworkId(self.GetId())
	if len(routes) == 0 {
		routes, _ = self.wire.vpc.region.GetRouteTables(self.VpcId, true)
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
	return self.wire.zone.region.deleteNetwork(self.SubnetId)
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

func (self *SRegion) createNetwork(zoneId string, vpcId string, name string, cidr, desc string) (string, error) {
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
		return "", errors.Wrapf(err, "CreateSubnet")
	}
	return ret.Subnet.SubnetId, nil
}

func (self *SRegion) getNetwork(networkId string) (*SNetwork, error) {
	if len(networkId) == 0 {
		return nil, fmt.Errorf("GetNetwork networkId should not be empty.")
	}
	networks, err := self.GetNetwroks([]string{networkId}, "")
	if err != nil {
		return nil, errors.Wrap(err, "GetNetwroks")
	}
	if len(networks) != 1 {
		return nil, errors.Wrap(cloudprovider.ErrNotFound, "getNetwork")
	}
	return &networks[0], nil
}

func (self *SRegion) deleteNetwork(networkId string) error {
	params := &ec2.DeleteSubnetInput{}
	params.SetSubnetId(networkId)
	ec2Client, err := self.getEc2Client()
	if err != nil {
		return errors.Wrap(err, "getEc2Client")
	}
	_, err = ec2Client.DeleteSubnet(params)
	return errors.Wrap(err, "DeleteSubnet")
}

func (self *SRegion) GetNetwroks(ids []string, vpcId string) ([]SNetwork, error) {
	params := &ec2.DescribeSubnetsInput{}
	if len(ids) > 0 {
		_ids := make([]*string, len(ids))
		for _, id := range ids {
			_ids = append(_ids, &id)
		}
		params.SetSubnetIds(_ids)
	}

	if len(vpcId) > 0 {
		filters := make([]*ec2.Filter, 1)
		vpcFilter := &ec2.Filter{}
		vpcFilter.SetName("vpc-id")
		vpcFilter.SetValues([]*string{&vpcId})
		filters = append(filters, vpcFilter)
		params.SetFilters(filters)
	}

	ec2Client, err := self.getEc2Client()
	if err != nil {
		return nil, errors.Wrap(err, "getEc2Client")
	}

	ret, err := ec2Client.DescribeSubnets(params)
	err = parseNotFoundError(err)
	if err != nil {
		return nil, err
	}

	subnets := []SNetwork{}
	for i := range ret.Subnets {
		item := ret.Subnets[i]
		if err := FillZero(item); err != nil {
			return nil, err
		}

		tagspec := TagSpec{ResourceType: "subnet"}
		tagspec.LoadingEc2Tags(item.Tags)

		subnet := SNetwork{}
		subnet.CidrBlock = *item.CidrBlock
		subnet.VpcId = *item.VpcId
		subnet.Status = *item.State
		subnet.ZoneId = *item.AvailabilityZone
		subnet.IsDefault = *item.DefaultForAz
		subnet.SubnetId = *item.SubnetId
		subnet.NetworkName = tagspec.GetNameTag()
		jsonutils.Update(&subnet.AwsTags.TagSet, item.Tags)
		subnets = append(subnets, subnet)
	}
	return subnets, nil
}

func (self *SNetwork) GetProjectId() string {
	return ""
}
