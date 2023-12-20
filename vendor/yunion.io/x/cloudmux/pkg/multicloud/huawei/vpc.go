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
	"net/url"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0020090625.html
type SVpc struct {
	multicloud.SVpc
	HuaweiTags

	region *SRegion

	ID                  string `json:"id"`
	Name                string `json:"name"`
	CIDR                string `json:"cidr"`
	Status              string `json:"status"`
	EnterpriseProjectID string `json:"enterprise_project_id"`
}

func (self *SVpc) GetId() string {
	return self.ID
}

func (self *SVpc) GetName() string {
	if len(self.Name) > 0 {
		return self.Name
	}
	return self.ID
}

func (self *SVpc) GetGlobalId() string {
	return self.ID
}

func (self *SVpc) GetStatus() string {
	return api.VPC_STATUS_AVAILABLE
}

func (self *SVpc) Refresh() error {
	vpc, err := self.region.GetVpc(self.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, vpc)
}

func (self *SVpc) GetRegion() cloudprovider.ICloudRegion {
	return self.region
}

func (self *SVpc) GetIsDefault() bool {
	return false
}

func (self *SVpc) GetCidrBlock() string {
	return self.CIDR
}

func (self *SVpc) GetIWires() ([]cloudprovider.ICloudWire, error) {
	return []cloudprovider.ICloudWire{&SWire{vpc: self}}, nil
}

func (self *SVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	return []cloudprovider.ICloudSecurityGroup{}, nil
}

func (self *SVpc) GetIRouteTables() ([]cloudprovider.ICloudRouteTable, error) {
	rtbs, err := self.region.GetRouteTables(self.ID)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudRouteTable{}
	for i := range rtbs {
		rtbs[i].vpc = self
		ret = append(ret, &rtbs[i])
	}
	return ret, nil
}

func (self *SVpc) GetIRouteTableById(routeTableId string) (cloudprovider.ICloudRouteTable, error) {
	rtb, err := self.region.GetRouteTable(routeTableId)
	if err != nil {
		return nil, err
	}
	rtb.vpc = self
	return rtb, nil
}

func (self *SVpc) Delete() error {
	return self.region.DeleteVpc(self.GetId())
}

func (self *SVpc) GetIWireById(wireId string) (cloudprovider.ICloudWire, error) {
	wires, err := self.GetIWires()
	if err != nil {
		return nil, err
	}
	for i := range wires {
		if wires[i].GetGlobalId() == wireId {
			return wires[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, wireId)
}

func (self *SVpc) GetINatGateways() ([]cloudprovider.ICloudNatGateway, error) {
	nats, err := self.region.GetNatGateways(self.GetId(), "")
	if err != nil {
		return nil, err
	}
	ret := make([]cloudprovider.ICloudNatGateway, len(nats))
	for i := 0; i < len(nats); i++ {
		nats[i].region = self.region
		ret[i] = &nats[i]
	}
	return ret, nil
}

func (self *SVpc) GetICloudVpcPeeringConnections() ([]cloudprovider.ICloudVpcPeeringConnection, error) {
	svpcPCs, err := self.getVpcPeeringConnections()
	if err != nil {
		return nil, errors.Wrap(err, "self.getVpcPeeringConnections()")
	}
	ivpcPCs := []cloudprovider.ICloudVpcPeeringConnection{}
	for i := range svpcPCs {
		ivpcPCs = append(ivpcPCs, &svpcPCs[i])
	}
	return ivpcPCs, nil
}

func (self *SVpc) GetICloudAccepterVpcPeeringConnections() ([]cloudprovider.ICloudVpcPeeringConnection, error) {
	svpcPCs, err := self.getAccepterVpcPeeringConnections()
	if err != nil {
		return nil, errors.Wrap(err, "self.getAccepterVpcPeeringConnections()")
	}
	ivpcPCs := []cloudprovider.ICloudVpcPeeringConnection{}
	for i := range svpcPCs {
		ivpcPCs = append(ivpcPCs, &svpcPCs[i])
	}
	return ivpcPCs, nil
}

func (self *SVpc) GetICloudVpcPeeringConnectionById(id string) (cloudprovider.ICloudVpcPeeringConnection, error) {
	svpcPC, err := self.getVpcPeeringConnectionById(id)
	if err != nil {
		return nil, errors.Wrapf(err, "self.getVpcPeeringConnectionById(%s)", id)
	}
	return svpcPC, nil
}

func (self *SVpc) CreateICloudVpcPeeringConnection(opts *cloudprovider.VpcPeeringConnectionCreateOptions) (cloudprovider.ICloudVpcPeeringConnection, error) {
	svpcPC, err := self.region.CreateVpcPeering(self.GetId(), opts)
	if err != nil {
		return nil, errors.Wrapf(err, "self.region.CreateVpcPeering(%s,%s)", self.GetId(), jsonutils.Marshal(opts).String())
	}
	svpcPC.vpc = self
	return svpcPC, nil
}
func (self *SVpc) AcceptICloudVpcPeeringConnection(id string) error {
	vpcPC, err := self.getVpcPeeringConnectionById(id)
	if err != nil {
		return errors.Wrapf(err, "self.getVpcPeeringConnectionById(%s)", id)
	}
	if vpcPC.GetStatus() == api.VPC_PEERING_CONNECTION_STATUS_ACTIVE {
		return nil
	}
	if vpcPC.GetStatus() == api.VPC_PEERING_CONNECTION_STATUS_UNKNOWN {
		return errors.Wrapf(cloudprovider.ErrInvalidStatus, "vpcPC: %s", jsonutils.Marshal(vpcPC).String())
	}
	err = self.region.AcceptVpcPeering(id)
	if err != nil {
		return errors.Wrapf(err, "self.region.AcceptVpcPeering(%s)", id)
	}
	return nil
}

func (self *SVpc) GetAuthorityOwnerId() string {
	return self.region.client.projectId
}

func (self *SVpc) getVpcPeeringConnections() ([]SVpcPeering, error) {
	svpcPeerings, err := self.region.GetVpcPeerings(self.GetId())
	if err != nil {
		return nil, errors.Wrapf(err, "self.region.GetVpcPeerings(%s)", self.GetId())
	}
	vpcPCs := []SVpcPeering{}
	for i := range svpcPeerings {
		if svpcPeerings[i].GetVpcId() == self.GetId() {
			svpcPeerings[i].vpc = self
			vpcPCs = append(vpcPCs, svpcPeerings[i])
		}
	}
	return vpcPCs, nil
}

func (self *SVpc) getAccepterVpcPeeringConnections() ([]SVpcPeering, error) {
	svpcPeerings, err := self.region.GetVpcPeerings(self.GetId())
	if err != nil {
		return nil, errors.Wrapf(err, "self.region.GetVpcPeerings(%s)", self.GetId())
	}
	vpcPCs := []SVpcPeering{}
	for i := range svpcPeerings {
		if svpcPeerings[i].GetPeerVpcId() == self.GetId() {
			svpcPeerings[i].vpc = self
			vpcPCs = append(vpcPCs, svpcPeerings[i])
		}
	}
	return vpcPCs, nil
}

func (self *SVpc) getVpcPeeringConnectionById(id string) (*SVpcPeering, error) {
	svpcPC, err := self.region.GetVpcPeering(id)
	if err != nil {
		return nil, errors.Wrapf(err, "self.region.GetVpcPeering(%s)", id)
	}
	svpcPC.vpc = self
	return svpcPC, nil
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/VPC/doc?version=v3&api=ShowVpc
func (self *SRegion) GetVpc(vpcId string) (*SVpc, error) {
	resp, err := self.list(SERVICE_VPC_V3, "vpc/vpcs/"+vpcId, nil)
	if err != nil {
		return nil, err
	}
	ret := &SVpc{region: self}
	err = resp.Unmarshal(ret, "vpc")
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	return ret, nil
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/VPC/doc?version=v2&api=DeleteVpc
func (self *SRegion) DeleteVpc(vpcId string) error {
	_, err := self.delete(SERVICE_VPC, "vpcs/"+vpcId)
	return err
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/VPC/doc?version=v3&api=ListVpcs
func (self *SRegion) GetVpcs() ([]SVpc, error) {
	ret := make([]SVpc, 0)
	query := url.Values{}
	for {
		resp, err := self.list(SERVICE_VPC_V3, "vpc/vpcs", query)
		if err != nil {
			return nil, err
		}
		part := struct {
			Vpcs     []SVpc
			PageInfo sPageInfo
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.Vpcs...)
		if len(part.Vpcs) == 0 || len(part.PageInfo.NextMarker) == 0 {
			break
		}
		query.Set("marker", part.PageInfo.NextMarker)
	}
	return ret, nil
}
