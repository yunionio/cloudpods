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

package huawei

import (
	"fmt"
	"net/url"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type RequestVpcInfo struct {
	VpcID    string `json:"vpc_id"`
	TenantID string `json:"tenant_id"`
}
type AcceptVpcInfo struct {
	VpcID    string `json:"vpc_id"`
	TenantID string `json:"tenant_id"`
}
type SVpcPeering struct {
	multicloud.SResourceBase
	HuaweiTags
	vpc *SVpc

	RequestVpcInfo RequestVpcInfo `json:"request_vpc_info"`
	AcceptVpcInfo  AcceptVpcInfo  `json:"accept_vpc_info"`
	Name           string         `json:"name"`
	ID             string         `json:"id"`
	Status         string         `json:"status"`
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/VPC/doc?version=v2&api=ListVpcPeerings
func (self *SRegion) GetVpcPeerings(vpcId string) ([]SVpcPeering, error) {
	query := url.Values{}
	if len(vpcId) > 0 {
		query.Set("vpc_id", vpcId)
	}
	ret := make([]SVpcPeering, 0)
	for {
		resp, err := self.list(SERVICE_VPC_V2_0, "vpc/peerings", query)
		if err != nil {
			return nil, err
		}
		part := []SVpcPeering{}
		err = resp.Unmarshal(&part, "peerings")
		if err != nil {
			return nil, err
		}
		ret = append(ret, part...)
		if len(part) == 0 {
			break
		}
		query.Set("marker", part[len(part)-1].ID)
	}
	return ret, nil
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/VPC/doc?version=v2&api=ListVpcPeerings
func (self *SRegion) GetVpcPeering(id string) (*SVpcPeering, error) {
	ret := &SVpcPeering{}
	resp, err := self.list(SERVICE_VPC_V2_0, "vpc/peerings/"+id, nil)
	if err != nil {
		return nil, err
	}
	err = resp.Unmarshal(ret, "peering")
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	return ret, nil
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/VPC/doc?version=v2&api=CreateVpcPeering
func (self *SRegion) CreateVpcPeering(vpcId string, opts *cloudprovider.VpcPeeringConnectionCreateOptions) (*SVpcPeering, error) {
	params := map[string]interface{}{
		"peering": map[string]interface{}{
			"name":        opts.Name,
			"description": opts.Desc,
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
	resp, err := self.post(SERVICE_VPC_V2_0, "vpc/peerings", params)
	if err != nil {
		return nil, err
	}
	ret := &SVpcPeering{}
	err = resp.Unmarshal(ret, "peering")
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/VPC/doc?version=v2&api=AcceptVpcPeering
func (self *SRegion) AcceptVpcPeering(id string) error {
	res := fmt.Sprintf("vpc/peerings/%s/accept", id)
	_, err := self.put(SERVICE_VPC_V2_0, res, nil)
	return err
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/VPC/doc?version=v2&api=DeleteVpcPeering
func (self *SRegion) DeleteVpcPeering(id string) error {
	_, err := self.delete(SERVICE_VPC_V2_0, "vpc/peerings/"+id)
	return err
}

func (self *SVpcPeering) GetId() string {
	return self.ID
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
	peer, err := self.vpc.region.GetVpcPeering(self.ID)
	if err != nil {
		return errors.Wrapf(err, "self.region.GetVpcPeering(%s)", self.ID)
	}
	return jsonutils.Update(self, peer)
}

func (self *SVpcPeering) GetVpcId() string {
	return self.RequestVpcInfo.VpcID
}

func (self *SVpcPeering) GetPeerVpcId() string {
	return self.AcceptVpcInfo.VpcID
}

func (self *SVpcPeering) GetPeerAccountId() string {
	return self.AcceptVpcInfo.TenantID
}

func (self *SVpcPeering) GetEnabled() bool {
	return true
}

func (self *SVpcPeering) Delete() error {
	err := self.vpc.region.DeleteVpcPeering(self.ID)
	if err != nil {
		return errors.Wrapf(err, "self.region.DeleteVpcPeering(%s)", self.ID)
	}
	return nil
}
