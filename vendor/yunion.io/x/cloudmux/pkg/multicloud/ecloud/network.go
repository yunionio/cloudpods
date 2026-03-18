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

package ecloud

import (
	"context"
	"fmt"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/rbacscope"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SNetwork struct {
	multicloud.SNetworkBase
	EcloudTags
	SZoneRegionBase

	wire *SWire

	// 与 ecloudsdkvpc ListSubnetsResponseContent 对齐
	VpoolId     string `json:"vpoolId,omitempty"`
	GatewayIp   string `json:"gatewayIp,omitempty"`
	Vaz         string `json:"vaz,omitempty"`
	Edge        bool   `json:"edge,omitempty"`
	Deleted     bool   `json:"deleted,omitempty"`
	IpVersion   int    `json:"ipVersion,omitempty"`
	Name        string `json:"name,omitempty"`
	CreatedTime string `json:"createdTime,omitempty"`
	CidrBlock   string `json:"cidr,omitempty"`
	NetworkId   string `json:"networkId,omitempty"`
	Id          string `json:"id,omitempty"`
	NetworkType string `json:"networkType,omitempty"`
	Region      string `json:"region,omitempty"`
}

func (n *SNetwork) GetId() string {
	return n.Id
}

func (n *SNetwork) GetName() string {
	return n.Name
}

func (n *SNetwork) GetGlobalId() string {
	return n.Id
}

func (n *SNetwork) GetStatus() string {
	// 子网接口未直接返回状态，这里视未删除的子网为可用
	if n.Deleted {
		return api.NETWORK_STATUS_DELETING
	}
	return api.NETWORK_STATUS_AVAILABLE
}

func (n *SNetwork) Refresh() error {
	return nil
}

func (n *SNetwork) IsEmulated() bool {
	return false
}

func (n *SNetwork) GetProjectId() string {
	return ""
}

func (n *SNetwork) GetIWire() cloudprovider.ICloudWire {
	return n.wire
}

func (n *SNetwork) GetIpStart() string {
	cidr := n.Cidr()
	if cidr == "" {
		return ""
	}
	pref, _ := netutils.NewIPV4Prefix(cidr)
	startIp := pref.Address.NetAddr(pref.MaskLen) // 0
	startIp = startIp.StepUp()                    // 1
	startIp = startIp.StepUp()                    // 2
	return startIp.String()
}

func (n *SNetwork) GetIpEnd() string {
	cidr := n.Cidr()
	if cidr == "" {
		return ""
	}
	pref, _ := netutils.NewIPV4Prefix(cidr)
	endIp := pref.Address.BroadcastAddr(pref.MaskLen) // 255
	endIp = endIp.StepDown()                          // 254
	return endIp.String()
}

func (n *SNetwork) Cidr() string {
	return n.CidrBlock
}

func (n *SNetwork) GetIpMask() int8 {
	cidr := n.Cidr()
	if cidr == "" {
		return 0
	}
	pref, _ := netutils.NewIPV4Prefix(cidr)
	return pref.MaskLen
}

func (n *SNetwork) GetGateway() string {
	return n.GatewayIp
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
	return self.wire.vpc.region.DeleteNetwork(self.Id)
}

func (self *SNetwork) GetAllocTimeoutSeconds() int {
	return 120
}

// GetNetworks 使用 OpenAPI VPC 子网列表：GET /api/openapi-vpc/customer/v3/subnet（替代旧的 network/NetworkResps 接口）
func (r *SRegion) GetNetworks(vpcId, zoneCode string) ([]SNetwork, error) {
	query := map[string]string{"networkTypeEnum": "VM"}
	if vpcId != "" {
		query["vpcId"] = vpcId
	}
	if zoneCode != "" {
		query["region"] = zoneCode
	}
	req := NewOpenApiVpcRequest(r.RegionId, "/api/openapi-vpc/customer/v3/subnet", query, nil)
	networks := make([]SNetwork, 0)
	err := r.client.doList(context.Background(), req.Base(), &networks)
	if err != nil {
		return nil, err
	}
	return networks, nil
}

// GetNetwork 使用 OpenAPI VPC 子网详情：GET /api/openapi-vpc/customer/v3/subnet/subnetId/{subnetId}/SubnetDetailResp
func (r *SRegion) GetNetwork(netId string) (*SNetwork, error) {
	base := NewOpenApiVpcRequest(r.RegionId,
		fmt.Sprintf("/api/openapi-vpc/customer/v3/subnet/subnetId/%s/SubnetDetailResp", netId),
		nil, nil).Base()
	base.SetMethod("GET")
	data, err := r.client.request(context.Background(), base)
	if err != nil {
		return nil, err
	}
	net := SNetwork{}
	if err := data.Unmarshal(&net); err != nil {
		return nil, errors.Wrap(err, "Unmarshal subnet detail")
	}
	return &net, nil
}
