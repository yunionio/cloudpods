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
	"yunion.io/x/log"
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

	AvailableIpAddressCount int
	CidrBlock               string
	CreationTime            time.Time
	Description             string
	IsDefault               bool
	Status                  string
	NetworkId               string
	NetworkName             string
	VpcId                   string
	ZoneId                  string
}

func (self *SNetwork) GetId() string {
	return self.NetworkId
}

func (self *SNetwork) GetName() string {
	if len(self.NetworkName) == 0 {
		return self.NetworkId
	}

	return self.NetworkName
}

func (self *SNetwork) GetGlobalId() string {
	return self.NetworkId
}

func (self *SNetwork) GetStatus() string {
	if self.wire != nil && self.wire.vpc != nil && self.wire.vpc.InstanceTenancy == "dedicated" {
		return api.NETWORK_STATUS_UNAVAILABLE
	}

	return strings.ToLower(self.Status)
}

func (self *SNetwork) Refresh() error {
	new, err := self.wire.zone.region.getNetwork(self.NetworkId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SNetwork) IsEmulated() bool {
	return false
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
	return self.wire.zone.region.deleteNetwork(self.NetworkId)
}

func (self *SNetwork) GetAllocTimeoutSeconds() int {
	return 120 // 2 minutes
}

func (self *SRegion) createNetwork(zoneId string, vpcId string, name string, cidr string, desc string) (string, error) {
	params := &ec2.CreateSubnetInput{}
	params.SetAvailabilityZone(zoneId)
	params.SetVpcId(vpcId)
	params.SetCidrBlock(cidr)

	ec2Client, err := self.getEc2Client()
	if err != nil {
		return "", errors.Wrap(err, "getEc2Client")
	}
	ret, err := ec2Client.CreateSubnet(params)
	if err != nil {
		return "", err
	} else {
		paramsTags := &ec2.CreateTagsInput{}
		tagspec := TagSpec{ResourceType: "subnet"}
		tagspec.SetNameTag(name)
		tagspec.SetDescTag(desc)
		ec2Tag, _ := tagspec.GetTagSpecifications()
		paramsTags.SetResources([]*string{ret.Subnet.SubnetId})
		paramsTags.SetTags(ec2Tag.Tags)
		_, err := ec2Client.CreateTags(paramsTags)
		if err != nil {
			log.Infof("createNetwork write tags failed:%s", err)
		}
		return *ret.Subnet.SubnetId, nil
	}
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
		subnet.NetworkId = *item.SubnetId
		subnet.NetworkName = tagspec.GetNameTag()
		jsonutils.Update(&subnet.AwsTags.TagSet, item.Tags)
		subnets = append(subnets, subnet)
	}
	return subnets, nil
}

func (self *SNetwork) GetProjectId() string {
	return ""
}
