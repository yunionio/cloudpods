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
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/ec2"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/netutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SNetwork struct {
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
	log.Debugf("network refresh %s", self.NetworkId)
	new, err := self.wire.zone.region.getNetwork(self.NetworkId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SNetwork) IsEmulated() bool {
	return false
}

func (self *SNetwork) GetMetadata() *jsonutils.JSONDict {
	meta := jsonutils.NewDict()
	routes, _ := self.wire.vpc.region.GetRouteTablesByNetworkId(self.GetId())
	if len(routes) == 0 {
		routes, _ = self.wire.vpc.region.GetRouteTables(self.VpcId, true)
	}

	support_eip := false
	if len(routes) >= 1 {
		for i := range routes[0].Routes {
			route := routes[0].Routes[i]
			if route.GetNextHopType() == api.Next_HOP_TYPE_INTERNET {
				support_eip = true
			}
		}
	}

	meta.Set("support_eip", jsonutils.NewBool(support_eip))
	return meta
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

	ret, err := self.ec2Client.CreateSubnet(params)
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
		_, err := self.ec2Client.CreateTags(paramsTags)
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
	networks, total, err := self.GetNetwroks([]string{networkId}, "", 0, 0)
	if err != nil {
		return nil, err
	}
	if total != 1 {
		return nil, ErrorNotFound()
	}
	return &networks[0], nil
}

func (self *SRegion) deleteNetwork(networkId string) error {
	params := &ec2.DeleteSubnetInput{}
	params.SetSubnetId(networkId)
	_, err := self.ec2Client.DeleteSubnet(params)
	return err
}

func (self *SRegion) GetNetwroks(ids []string, vpcId string, limit int, offset int) ([]SNetwork, int, error) {
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

	ret, err := self.ec2Client.DescribeSubnets(params)
	err = parseNotFoundError(err)
	if err != nil {
		return nil, 0, err
	}

	subnets := []SNetwork{}
	for i := range ret.Subnets {
		item := ret.Subnets[i]
		if err := FillZero(item); err != nil {
			return nil, 0, err
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
		subnets = append(subnets, subnet)
	}
	return subnets, len(subnets), nil
}

func (self *SNetwork) GetProjectId() string {
	return ""
}
