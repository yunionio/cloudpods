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

package hcso

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/cloudmux/pkg/multicloud/huawei"
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
	huawei.HuaweiTags
	vpc *SVpc

	RequestVpcInfo RequestVpcInfo `json:"request_vpc_info"`
	AcceptVpcInfo  AcceptVpcInfo  `json:"accept_vpc_info"`
	Name           string         `json:"name"`
	ID             string         `json:"id"`
	Status         string         `json:"status"`
}

func (self *SRegion) GetVpcPeerings(vpcId string) ([]SVpcPeering, error) {
	querys := make(map[string]string)
	querys["vpc_id"] = vpcId
	vpcPeerings := make([]SVpcPeering, 0)
	err := doListAllWithMarker(self.ecsClient.VpcPeerings.List, querys, &vpcPeerings)
	if err != nil {
		return nil, errors.Wrapf(err, "oListAllWithMarker(self.ecsClient.VpcPeerings.List, %s, &vpcPeerings)", jsonutils.Marshal(querys).String())
	}
	return vpcPeerings, nil
}

func (self *SRegion) GetVpcPeering(vpcPeeringId string) (*SVpcPeering, error) {
	if len(vpcPeeringId) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	vpcPeering := SVpcPeering{}
	err := DoGet(self.ecsClient.VpcPeerings.Get, vpcPeeringId, nil, &vpcPeering)
	if err != nil {
		return nil, errors.Wrapf(err, "DoGet(self.ecsClient.VpcPeerings.Get, %s, nil, &vpcPeering)", vpcPeeringId)
	}
	return &vpcPeering, nil
}

func (self *SRegion) CreateVpcPeering(vpcId string, opts *cloudprovider.VpcPeeringConnectionCreateOptions) (*SVpcPeering, error) {
	params := jsonutils.NewDict()
	vpcPeeringObj := jsonutils.NewDict()
	requestVpcObj := jsonutils.NewDict()
	acceptVpcObj := jsonutils.NewDict()
	vpcPeeringObj.Set("name", jsonutils.NewString(opts.Name))
	requestVpcObj.Set("vpc_id", jsonutils.NewString(vpcId))
	requestVpcObj.Set("tenant_id", jsonutils.NewString(self.client.projectId))
	vpcPeeringObj.Set("request_vpc_info", requestVpcObj)
	acceptVpcObj.Set("vpc_id", jsonutils.NewString(opts.PeerVpcId))
	acceptVpcObj.Set("tenant_id", jsonutils.NewString(opts.PeerAccountId))
	vpcPeeringObj.Set("accept_vpc_info", acceptVpcObj)
	params.Set("peering", vpcPeeringObj)
	ret := SVpcPeering{}
	err := DoCreate(self.ecsClient.VpcPeerings.Create, params, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "DoCreate(self.ecsClient.VpcPeerings.Create, %s, &ret)", jsonutils.Marshal(params).String())
	}
	return &ret, nil
}

func (self *SRegion) AcceptVpcPeering(vpcPeeringId string) error {
	err := DoUpdateWithSpec(self.ecsClient.VpcPeerings.UpdateInContextWithSpec, vpcPeeringId, "accept", nil)
	if err != nil {
		return errors.Wrapf(err, "DoUpdateWithSpec(self.ecsClient.VpcPeerings.UpdateInContextWithSpec, %s, accept, nil)", vpcPeeringId)
	}
	return nil
}

func (self *SRegion) DeleteVpcPeering(vpcPeeringId string) error {
	err := DoDelete(self.ecsClient.VpcPeerings.Delete, vpcPeeringId, nil, nil)
	if err != nil {
		return errors.Wrapf(err, "DoDelete(self.ecsClient.VpcPeerings.Delete,%s,nil)", vpcPeeringId)
	}
	return nil
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
