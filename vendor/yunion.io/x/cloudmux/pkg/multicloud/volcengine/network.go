// Copyright 2023 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package volcengine

import (
	"fmt"
	"strings"
	"time"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"
)

type SNetwork struct {
	multicloud.SNetworkBase
	VolcEngineTags
	wire *SWire

	AccountId               string
	SubnetId                string
	VpcId                   string
	Status                  string
	CidrBlock               string
	Ipv6CidrBlock           string
	ZoneId                  string
	AvailableIpAddressCount int
	Description             string
	SubnetName              string
	CreationTime            time.Time
	UpdateTime              time.Time
	TotalIpv4Count          int
	NetworkAclId            string
	IsDefault               bool
	RouteTable              SRouteTable
	ProjectName             string
}

func (subnet *SNetwork) GetId() string {
	return subnet.SubnetId
}

func (subnet *SNetwork) GetName() string {
	if len(subnet.SubnetName) > 0 {
		return subnet.SubnetName
	}
	return subnet.SubnetId
}

func (subnet *SNetwork) GetGlobalId() string {
	return subnet.SubnetId
}

func (subnet *SNetwork) GetStatus() string {
	return strings.ToLower(subnet.Status)
}

func (subnet *SNetwork) Refresh() error {
	net, err := subnet.wire.zone.region.GetSubnetAttributes(subnet.SubnetId)
	if err != nil {
		return err
	}
	return jsonutils.Update(subnet, net)
}

func (subnet *SNetwork) GetIWire() cloudprovider.ICloudWire {
	return subnet.wire
}

func (subnet *SNetwork) GetProjectId() string {
	return subnet.ProjectName
}

func (net *SNetwork) GetIp6Start() string {
	if len(net.Ipv6CidrBlock) > 0 {
		prefix, err := netutils.NewIPV6Prefix(net.Ipv6CidrBlock)
		if err != nil {
			return ""
		}
		return prefix.Address.NetAddr(prefix.MaskLen).StepUp().StepUp().String()
	}
	return ""
}

func (net *SNetwork) GetIp6End() string {
	if len(net.Ipv6CidrBlock) > 0 {
		prefix, err := netutils.NewIPV6Prefix(net.Ipv6CidrBlock)
		if err != nil {
			return ""
		}
		end := prefix.Address.NetAddr(prefix.MaskLen).BroadcastAddr(prefix.MaskLen)
		return end.StepDown().String()
	}
	return ""
}

func (net *SNetwork) GetIp6Mask() uint8 {
	if len(net.Ipv6CidrBlock) > 0 {
		prefix, err := netutils.NewIPV6Prefix(net.Ipv6CidrBlock)
		if err != nil {
			return 0
		}
		return prefix.MaskLen
	}
	return 0
}

func (net *SNetwork) GetGateway6() string {
	if len(net.Ipv6CidrBlock) > 0 {
		prefix, err := netutils.NewIPV6Prefix(net.Ipv6CidrBlock)
		if err != nil {
			return ""
		}
		return prefix.Address.NetAddr(prefix.MaskLen).StepUp().String()
	}
	return ""
}

func (subnet *SNetwork) GetIpStart() string {
	pref, _ := netutils.NewIPV4Prefix(subnet.CidrBlock)
	startIp := pref.Address.NetAddr(pref.MaskLen)
	startIp = startIp.StepUp()
	startIp = startIp.StepUp()
	return startIp.String()
}

func (subnet *SNetwork) GetIpEnd() string {
	pref, _ := netutils.NewIPV4Prefix(subnet.CidrBlock)
	endIp := pref.Address.BroadcastAddr(pref.MaskLen)
	endIp = endIp.StepDown()
	return endIp.String()
}

func (subnet *SNetwork) GetIpMask() int8 {
	pref, _ := netutils.NewIPV4Prefix(subnet.CidrBlock)
	return pref.MaskLen
}

func (subnet *SNetwork) GetGateway() string {
	pref, _ := netutils.NewIPV4Prefix(subnet.CidrBlock)
	endIp := pref.Address.BroadcastAddr(pref.MaskLen)
	endIp = endIp.StepDown()
	return endIp.String()
}

func (subnet *SNetwork) GetServerType() string {
	return api.NETWORK_TYPE_GUEST
}

func (subnet *SNetwork) GetIsPublic() bool {
	return subnet.IsDefault
}

func (subnet *SNetwork) GetPublicScope() rbacscope.TRbacScope {
	return rbacscope.ScopeDomain
}

func (region *SRegion) CreateSubnet(zoneId string, vpcId string, name string, cidr string, desc string) (string, error) {
	params := make(map[string]string)
	params["ZoneId"] = zoneId
	params["VpcId"] = vpcId
	params["CidrBlock"] = cidr
	params["SubnetName"] = name
	if len(desc) > 0 {
		params["Description"] = desc
	}
	params["ClientToken"] = utils.GenRequestId(20)

	body, err := region.vpcRequest("CreateSubnet", params)
	if err != nil {
		return "", err
	}
	return body.GetString("SubnetId")
}

func (region *SRegion) DeleteSubnet(SubnetId string) error {
	params := make(map[string]string)
	params["SubnetId"] = SubnetId

	_, err := region.vpcRequest("DeleteSubnet", params)
	return err
}

func (subnet *SNetwork) dissociateWithSNAT() error {
	natgatways, err := subnet.wire.vpc.getNatGateways()
	if err != nil {
		return err
	}
	for i := range natgatways {
		err = natgatways[i].dissociateWithSubnet(subnet.SubnetId)
		if err != nil {
			return err
		}
	}
	return nil
}

func (subnet *SNetwork) Delete() error {
	err := subnet.Refresh()
	if err != nil {
		log.Errorf("refresh Subnet fail %s", err)
		return err
	}
	if len(subnet.RouteTable.RouteTableId) > 0 && !subnet.RouteTable.IsSystem() {
		err = subnet.wire.zone.region.UnassociateRouteTable(subnet.RouteTable.RouteTableId, subnet.SubnetId)
		if err != nil {
			log.Errorf("unassociate routetable fail %s", err)
			return err
		}
	}
	err = subnet.dissociateWithSNAT()
	if err != nil {
		log.Errorf("fail to dissociateWithSNAT")
		return err
	}
	err = cloudprovider.Wait(10*time.Second, time.Minute, func() (bool, error) {
		err := subnet.wire.zone.region.DeleteSubnet(subnet.SubnetId)
		if err != nil {
			if isError(err, "DependencyViolation") {
				return false, nil
			}
			return false, err
		} else {
			return true, nil
		}
	})
	return err
}

func (subnet *SNetwork) GetAllocTimeoutSeconds() int {
	return 120
}

func (region *SRegion) GetSubnets(ids []string, zoneId string, vpcId string, pageNumber int, pageSize int) ([]SNetwork, int, error) {
	if pageSize > 100 || pageSize <= 0 {
		pageSize = 100
	}
	params := make(map[string]string)
	params["PageSize"] = fmt.Sprintf("%d", pageSize)
	params["PageNumber"] = fmt.Sprintf("%d", pageNumber)
	for idx, id := range ids {
		params[fmt.Sprintf("SubnetIds.%d", idx)] = id
	}
	if len(zoneId) > 0 {
		params["ZoneId"] = zoneId
	}
	if len(vpcId) > 0 {
		params["VpcId"] = vpcId
	}

	body, err := region.vpcRequest("DescribeSubnets", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "GetSubnets fail")
	}

	subnets := make([]SNetwork, 0)
	err = body.Unmarshal(&subnets, "Subnets")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "Unmarshal subnets fail")
	}
	total, _ := body.Int("TotalCount")
	return subnets, int(total), nil
}

func (region *SRegion) GetSubnetAttributes(SubnetId string) (*SNetwork, error) {
	params := make(map[string]string)
	params["SubnetId"] = SubnetId
	body, err := region.vpcRequest("DescribeSubnetAttributes", params)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeSubnetAttributes fail")
	}
	if region.client.debug {
		log.Debugf("%s", body.PrettyString())
	}
	subnet := SNetwork{}
	err = body.Unmarshal(&subnet)
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal subnet fail")
	}
	return &subnet, nil
}
