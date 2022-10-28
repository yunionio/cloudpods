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

package hcs

import (
	"fmt"
	"net/url"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/cloudmux/pkg/multicloud/huawei"
)

type RequestVpcInfo struct {
	VpcId    string `json:"vpc_id"`
	TenantId string `json:"tenant_id"`
}

type AcceptVpcInfo struct {
	VpcId    string `json:"vpc_id"`
	TenantId string `json:"tenant_id"`
}

type SVpcPeering struct {
	multicloud.SResourceBase
	huawei.HuaweiTags
	vpc *SVpc

	RequestVpcInfo RequestVpcInfo `json:"request_vpc_info"`
	AcceptVpcInfo  AcceptVpcInfo  `json:"accept_vpc_info"`
	Name           string         `json:"name"`
	Id             string         `json:"id"`
	Status         string         `json:"status"`
}

func (self *SRegion) GetVpcPeerings(vpcId string) ([]SVpcPeering, error) {
	params := url.Values{}
	if len(vpcId) > 0 {
		params.Set("vpc_id", vpcId)
	}
	ret := []SVpcPeering{}
	return ret, self.list("vpc", "v2.0", "vpc/peerings", params, &ret)
}

func (self *SRegion) GetVpcPeering(id string) (*SVpcPeering, error) {
	ret := SVpcPeering{}
	resource := fmt.Sprintf("vpc/peerings/%s", id)
	return &ret, self.get("vpc", "v2.0", resource, &ret)
}

func (self *SRegion) CreateVpcPeering(vpcId string, opts *cloudprovider.VpcPeeringConnectionCreateOptions) (*SVpcPeering, error) {
	params := map[string]interface{}{
		"peering": map[string]interface{}{
			"name": opts.Name,
			"request_vpc_info": map[string]interface{}{
				"vpc_id":    vpcId,
				"tenant_id": self.client.projectId,
			},
			"accept_vpc_info": map[string]interface{}{
				"vpc_id":    opts.PeerVpcId,
				"tenant_id": opts.PeerAccountId,
			},
		},
	}
	ret := &SVpcPeering{}
	return ret, self.create("vpc", "v2.0", "vpc/peerings", params, ret)
}

func (self *SRegion) AcceptVpcPeering(id string) error {
	res := fmt.Sprintf("vpc/peerings/%s/accept", id)
	return self.update("vpc", "v2.0", res, nil)
}

func (self *SRegion) DeleteVpcPeering(id string) error {
	res := fmt.Sprintf("vpc/peerings/%s", id)
	return self.delete("vpc", "v2.0", res)
}

func (self *SVpcPeering) GetId() string {
	return self.Id
}

func (self *SVpcPeering) GetName() string {
	return self.Name
}

func (self *SVpcPeering) GetGlobalId() string {
	return self.GetId()
}

func (self *SVpcPeering) GetStatus() string {
	switch self.Status {
	case "PENDING_ACCEPTANCE":
		return api.VPC_PEERING_CONNECTION_STATUS_PENDING_ACCEPT
	case "ACTIVE":
		return api.VPC_PEERING_CONNECTION_STATUS_ACTIVE
	default:
		return api.VPC_PEERING_CONNECTION_STATUS_UNKNOWN
	}
}

func (self *SVpcPeering) Refresh() error {
	peer, err := self.vpc.region.GetVpcPeering(self.Id)
	if err != nil {
		return errors.Wrapf(err, "self.region.GetVpcPeering(%s)", self.Id)
	}
	return jsonutils.Update(self, peer)
}

func (self *SVpcPeering) GetVpcId() string {
	return self.RequestVpcInfo.VpcId
}

func (self *SVpcPeering) GetPeerVpcId() string {
	return self.AcceptVpcInfo.VpcId
}

func (self *SVpcPeering) GetPeerAccountId() string {
	return self.AcceptVpcInfo.TenantId
}

func (self *SVpcPeering) GetEnabled() bool {
	return true
}

func (self *SVpcPeering) Delete() error {
	return self.vpc.region.DeleteVpcPeering(self.Id)
}
