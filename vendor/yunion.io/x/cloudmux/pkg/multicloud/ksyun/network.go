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

package ksyun

import (
	"fmt"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/rbacscope"
)

type SNetworkResp struct {
	SubnetSet []SNetwork `json:"SubnetSet"`
	RequestID string     `json:"RequestId"`
	NextToken string     `json:"NextToken"`
}

type SNetwork struct {
	multicloud.SNetworkBase
	SKsTag
	wire *SWire

	RouteTableID          string `json:"RouteTableId"`
	NetworkACLID          string `json:"NetworkAclId"`
	NatID                 string `json:"NatId"`
	CreateTime            string `json:"CreateTime"`
	DhcpIPTo              string `json:"DhcpIpTo"`
	DNS1                  string `json:"Dns1"`
	CidrBlock             string `json:"CidrBlock"`
	DNS2                  string `json:"Dns2"`
	ProvidedIpv6CidrBlock bool   `json:"ProvidedIpv6CidrBlock"`
	SubnetID              string `json:"SubnetId"`
	SubnetType            string `json:"SubnetType"`
	SubnetName            string `json:"SubnetName"`
	VpcID                 string `json:"VpcId"`
	GatewayIP             string `json:"GatewayIp"`
	AvailabilityZoneName  string `json:"AvailabilityZoneName"`
	DhcpIPFrom            string `json:"DhcpIpFrom"`
	AvailableIPNumber     int    `json:"AvailableIpNumber"`
	SecondaryCidrID       string `json:"SecondaryCidrId"`
}

func (region *SRegion) GetNetworks(vpcIds, networkIds []string, zoneName string) ([]SNetwork, error) {
	networks := []SNetwork{}
	param := map[string]string{}
	searchIndex := 1
	if len(vpcIds) > 0 {
		param[fmt.Sprintf("Filter.%d.Name", searchIndex)] = "vpc-id"
		for i, vpcId := range vpcIds {
			param[fmt.Sprintf("Filter.%d.Value.%d", searchIndex, i+1)] = vpcId
		}
		searchIndex++
	}

	if len(zoneName) > 0 {
		param[fmt.Sprintf("Filter.%d.Name", searchIndex)] = "availability-zone-name"
		param[fmt.Sprintf("Filter.%d.Value.%d", searchIndex, 1)] = zoneName
		searchIndex++
	}

	for i, networkId := range networkIds {
		param[fmt.Sprintf("SubnetId.%d", i+1)] = networkId
	}
	for {
		resp, err := region.vpcRequest("DescribeSubnets", param)
		if err != nil {
			return nil, errors.Wrap(err, "list networks")
		}
		res := SNetworkResp{}
		err = resp.Unmarshal(&res)
		if err != nil {
			return nil, errors.Wrap(err, "unmarshal instances")
		}
		networks = append(networks, res.SubnetSet...)
		if len(res.NextToken) == 0 {
			break
		}
		param["NextToken"] = res.NextToken
	}

	return networks, nil
}

func (region *SRegion) GetNetwork(networkId string) (*SNetwork, error) {
	extNetworks, err := region.GetNetworks(nil, []string{networkId}, "")
	if err != nil {
		return nil, errors.Wrap(err, "GetNetworks")
	}
	for _, extNetwork := range extNetworks {
		if extNetwork.GetGlobalId() == networkId {
			return &extNetwork, nil
		}
	}
	return nil, errors.Wrapf(errors.ErrNotFound, "network id:%s", networkId)
}

func (net *SNetwork) GetId() string {
	return net.SubnetID
}

func (net *SNetwork) GetName() string {
	if len(net.SubnetName) == 0 {
		return net.SubnetID
	}

	return net.SubnetName
}

func (net *SNetwork) GetGlobalId() string {
	return net.SubnetID
}

func (net *SNetwork) GetStatus() string {
	return api.NETWORK_STATUS_AVAILABLE
}

func (net *SNetwork) Refresh() error {
	extNet, err := net.wire.zone.region.GetNetwork(net.GetGlobalId())
	if err != nil {
		return errors.Wrap(err, "GetNetwork")
	}
	return jsonutils.Update(net, extNet)
}

func (net *SNetwork) GetTags() (map[string]string, error) {
	tags, err := net.wire.zone.region.ListTags("subnet", net.SubnetID)
	if err != nil {
		return nil, err
	}
	return tags.GetTags(), nil
}

func (net *SNetwork) GetIWire() cloudprovider.ICloudWire {
	return net.wire
}

func (net *SNetwork) GetIpStart() string {
	pref, _ := netutils.NewIPV4Prefix(net.CidrBlock)
	startIp := pref.Address.NetAddr(pref.MaskLen) // 0
	startIp = startIp.StepUp()                    // 1
	return startIp.String()
}

func (net *SNetwork) GetIpEnd() string {
	pref, _ := netutils.NewIPV4Prefix(net.CidrBlock)
	endIp := pref.Address.BroadcastAddr(pref.MaskLen) // 255
	endIp = endIp.StepDown()                          // 254
	return endIp.String()
}

func (net *SNetwork) GetIpMask() int8 {
	pref, _ := netutils.NewIPV4Prefix(net.CidrBlock)
	return pref.MaskLen
}

func (net *SNetwork) GetGateway() string {
	return net.GatewayIP
}

func (net *SNetwork) GetServerType() string {
	return api.NETWORK_TYPE_GUEST
}

func (net *SNetwork) GetIsPublic() bool {
	return true
}

func (net *SNetwork) GetPublicScope() rbacscope.TRbacScope {
	return rbacscope.ScopeDomain
}

func (net *SNetwork) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (net *SNetwork) GetAllocTimeoutSeconds() int {
	return 120 // 2 minutes
}

func (net *SNetwork) GetProjectId() string {
	return ""
}

func (net *SNetwork) GetDescription() string {
	return ""
}
