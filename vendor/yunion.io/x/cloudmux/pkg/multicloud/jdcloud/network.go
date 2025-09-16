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

package jdcloud

import (
	"fmt"

	commodels "github.com/jdcloud-api/jdcloud-sdk-go/services/common/models"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/vpc/apis"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/vpc/client"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/vpc/models"

	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/rbacscope"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SNetwork struct {
	multicloud.SNetworkBase
	JdcloudTags

	wire *SWire

	models.Subnet
}

func (n *SNetwork) GetId() string {
	return n.SubnetId
}

func (n *SNetwork) GetName() string {
	return n.SubnetName
}

func (n *SNetwork) GetGlobalId() string {
	return n.GetId()
}

func (n *SNetwork) GetStatus() string {
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
	return n.StartIp
}

func (n *SNetwork) GetIpEnd() string {
	return n.EndIp
}

func (n *SNetwork) Cidr() string {
	return n.AddressPrefix
}

func (n *SNetwork) GetIpMask() int8 {
	pref, _ := netutils.NewIPV4Prefix(n.Cidr())
	return pref.MaskLen
}

func (n *SNetwork) GetGateway() string {
	return ""
}

func (n *SNetwork) GetServerType() string {
	return api.NETWORK_TYPE_GUEST
}

func (n *SNetwork) GetIsPublic() bool {
	return true
}

func (n *SNetwork) GetPublicScope() rbacscope.TRbacScope {
	return rbacscope.ScopeDomain
}

func (n *SNetwork) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (n *SNetwork) GetAllocTimeoutSeconds() int {
	return 120
}

func (r *SRegion) GetNetworks(vpcId string, pageNumber int, pageSize int) ([]SNetwork, int, error) {
	filters := []commodels.Filter{}
	if vpcId != "" {
		filters = append(filters, commodels.Filter{
			Name:   "vpcId",
			Values: []string{vpcId},
		})
	}
	req := apis.NewDescribeSubnetsRequestWithAllParams(r.ID, &pageNumber, &pageSize, filters)
	client := client.NewVpcClient(r.getCredential())
	client.Logger = Logger{debug: r.client.debug}
	resp, err := client.DescribeSubnets(req)
	if err != nil {
		return nil, 0, err
	}
	if resp.Error.Code >= 400 {
		return nil, 0, fmt.Errorf(resp.Error.Message)
	}
	nets := make([]SNetwork, len(resp.Result.Subnets))
	for i := range nets {
		nets[i] = SNetwork{
			Subnet: resp.Result.Subnets[i],
		}
	}
	return nets, resp.Result.TotalCount, nil
}

func (r *SRegion) GetNetworkById(id string) (*SNetwork, error) {
	req := apis.NewDescribeSubnetRequest(r.ID, id)
	client := client.NewVpcClient(r.getCredential())
	client.Logger = Logger{debug: r.client.debug}
	resp, err := client.DescribeSubnet(req)
	if err != nil {
		return nil, err
	}
	if resp.Error.Code >= 400 {
		return nil, fmt.Errorf(resp.Error.Message)
	}
	return &SNetwork{
		Subnet: resp.Result.Subnet,
	}, nil
}
