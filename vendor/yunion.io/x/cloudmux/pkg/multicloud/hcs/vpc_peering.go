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

	RequestVpcInfo RequestVpcInfo `json:"requesterVpcInfo"`
	AcceptVpcInfo  AcceptVpcInfo  `json:"accepterVpcInfo"`
	Name           string         `json:"name"`
	Id             string         `json:"id"`
	Status         string         `json:"status"`
}

func (self *SRegion) GetVpcPeerings() ([]SVpcPeering, error) {
	ret := []SVpcPeering{}
	return ret, self.list("vpc", "v1", "vpcpeering", nil, &ret)
}

func (self *SRegion) GetVpcPeering(id string) (*SVpcPeering, error) {
	ret := []SVpcPeering{}
	params := url.Values{}
	params.Set("peering_id", id)
	err := self.list("vpc", "v1", "vpcpeering", params, &ret)
	if err != nil {
		return nil, err
	}
	for i := range ret {
		if ret[i].Id == id {
			return &ret[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SRegion) CreateVpcPeering(vpcId string, opts *cloudprovider.VpcPeeringConnectionCreateOptions) (*SVpcPeering, error) {
	params := map[string]interface{}{
		"name":         opts.Name,
		"local_vpc_id": vpcId,
		"peer_vpc_id":  opts.PeerVpcId,
	}
	ret := &SVpcPeering{}
	return ret, self.create("vpc", "v1", "vpcpeering", params, ret)
}

func (self *SRegion) AcceptVpcPeering(id string) error {
	res := fmt.Sprintf("vpcpeering/%s/accept", id)
	return self.update("vpc", "v1", res, nil)
}

func (self *SRegion) DeleteVpcPeering(id string) error {
	res := fmt.Sprintf("vpcpeering/%s", id)
	return self.delete("vpc", "v1", res)
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
	case "pending_acceptance":
		return api.VPC_PEERING_CONNECTION_STATUS_PENDING_ACCEPT
	case "active":
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

func (self *SVpc) GetICloudVpcPeeringConnections() ([]cloudprovider.ICloudVpcPeeringConnection, error) {
	peers, err := self.region.GetVpcPeeringConnections()
	if err != nil {
		return nil, errors.Wrap(err, "GetVpcPeeringConnections")
	}
	ret := []cloudprovider.ICloudVpcPeeringConnection{}
	for i := range peers {
		if peers[i].RequestVpcInfo.VpcId != self.Id {
			continue
		}
		peers[i].vpc = self
		ret = append(ret, &peers[i])
	}
	return ret, nil
}

func (self *SVpc) GetICloudAccepterVpcPeeringConnections() ([]cloudprovider.ICloudVpcPeeringConnection, error) {
	peers, err := self.region.GetVpcPeeringConnections()
	if err != nil {
		return nil, errors.Wrap(err, "GetVpcPeeringConnections")
	}
	ret := []cloudprovider.ICloudVpcPeeringConnection{}
	for i := range peers {
		if peers[i].AcceptVpcInfo.VpcId != self.Id {
			continue
		}
		peers[i].vpc = self
		ret = append(ret, &peers[i])
	}
	return ret, nil
}

func (self *SVpc) GetICloudVpcPeeringConnectionById(id string) (cloudprovider.ICloudVpcPeeringConnection, error) {
	ret, err := self.region.GetVpcPeering(id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetVpcPeering(%s)", id)
	}
	ret.vpc = self
	return ret, nil
}

func (self *SVpc) CreateICloudVpcPeeringConnection(opts *cloudprovider.VpcPeeringConnectionCreateOptions) (cloudprovider.ICloudVpcPeeringConnection, error) {
	ret, err := self.region.CreateVpcPeering(self.GetId(), opts)
	if err != nil {
		return nil, errors.Wrapf(err, "self.region.CreateVpcPeering(%s,%s)", self.GetId(), jsonutils.Marshal(opts).String())
	}
	ret.vpc = self
	return ret, nil
}

func (self *SVpc) AcceptICloudVpcPeeringConnection(id string) error {
	peer, err := self.region.GetVpcPeering(id)
	if err != nil {
		return errors.Wrapf(err, "GetVpcPeering%s)", id)
	}
	if peer.GetStatus() == api.VPC_PEERING_CONNECTION_STATUS_ACTIVE {
		return nil
	}
	if peer.GetStatus() == api.VPC_PEERING_CONNECTION_STATUS_UNKNOWN {
		return errors.Wrapf(cloudprovider.ErrInvalidStatus, "vpcPC: %s", jsonutils.Marshal(peer).String())
	}
	return self.region.AcceptVpcPeering(id)
}

func (self *SRegion) GetVpcPeeringConnections() ([]SVpcPeering, error) {
	if len(self.peers) > 0 {
		return self.peers, nil
	}
	var err error
	self.peers, err = self.GetVpcPeerings()
	if err != nil {
		return nil, errors.Wrapf(err, "GetVpcPeerings()")
	}
	return self.peers, nil
}
