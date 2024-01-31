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

package ctyun

import (
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SNetwork struct {
	multicloud.SNetworkBase
	CtyunTags
	wire *SWire

	SubnetId          string
	Name              string
	Description       string
	VpcId             string
	CIDR              string
	AvailableIPCount  int
	GatewayIP         string
	AvailabilityZones []string
	RouteTableId      string
	NetworkAclId      string
	Start             string
	End               string
	Ipv6Enabled       int
	Ipv6CIDR          string
	Ipv6Start         string
	Ipv6End           string
	Ipv6GatewayIP     string
	DnsList           []string
	NtpList           []string
	Type              int
	CreateAt          time.Time
}

func (self *SNetwork) GetId() string {
	return self.SubnetId
}

func (self *SNetwork) GetName() string {
	return self.Name
}

func (self *SNetwork) GetGlobalId() string {
	return self.GetId()
}

func (self *SNetwork) GetStatus() string {
	return api.NETWORK_STATUS_AVAILABLE
}

func (self *SNetwork) Refresh() error {
	net, err := self.wire.vpc.region.GetNetwork(self.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, net)
}

func (self *SNetwork) GetProjectId() string {
	return self.wire.vpc.ProjectId
}

func (self *SNetwork) GetIWire() cloudprovider.ICloudWire {
	return self.wire
}

func (self *SNetwork) GetIpStart() string {
	return self.Start
}

func (self *SNetwork) GetIpEnd() string {
	return self.End
}

func (self *SNetwork) GetIpMask() int8 {
	pref, _ := netutils.NewIPV4Prefix(self.CIDR)
	return pref.MaskLen
}

func (self *SNetwork) GetGateway() string {
	return self.GatewayIP
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
	return self.wire.vpc.region.DeleteNetwork(self.GetId())
}

func (self *SNetwork) GetAllocTimeoutSeconds() int {
	return 120
}

func (self *SRegion) GetNetwroks(vpcId string) ([]SNetwork, error) {
	pageNo := 1
	params := map[string]interface{}{
		"vpcID":    vpcId,
		"pageNo":   pageNo,
		"pageSize": 50,
	}
	ret := []SNetwork{}
	for {
		resp, err := self.list(SERVICE_VPC, "/v4/vpc/new-list-subnet", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			ReturnObj struct {
				Subnets []SNetwork
			}
			TotalCount int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrapf(err, "Unmarshal")
		}
		ret = append(ret, part.ReturnObj.Subnets...)
		if len(part.ReturnObj.Subnets) == 0 || len(ret) >= part.TotalCount {
			break
		}
		pageNo++
		params["pageNo"] = pageNo
	}
	return ret, nil
}

func (self *SRegion) GetNetwork(subnetId string) (*SNetwork, error) {
	params := map[string]interface{}{
		"subnetID": subnetId,
	}
	resp, err := self.list(SERVICE_VPC, "/v4/vpc/query-subnet", params)
	if err != nil {
		return nil, err
	}
	network := &SNetwork{}
	return network, resp.Unmarshal(network, "returnObj")
}

func (self *SRegion) CreateNetwork(vpcId string, opts *cloudprovider.SNetworkCreateOptions) (*SNetwork, error) {
	params := map[string]interface{}{
		"clientToken": utils.GenRequestId(20),
		"vpcID":       vpcId,
		"name":        opts.Name,
		"description": opts.Desc,
		"CIDR":        opts.Cidr,
	}
	resp, err := self.post(SERVICE_VPC, "/v4/vpc/create-subnet", params)
	if err != nil {
		return nil, err
	}
	netId, err := resp.GetString("returnObj", "subnetID")
	if err != nil {
		return nil, errors.Wrapf(err, "get subnetID")
	}
	return self.GetNetwork(netId)
}

func (self *SRegion) DeleteNetwork(subnetId string) error {
	params := map[string]interface{}{
		"subnetID": subnetId,
	}
	_, err := self.post(SERVICE_VPC, "/v4/vpc/delete-subnet", params)
	return err
}
