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
	"strings"

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

	iwires    []cloudprovider.ICloudWire
	secgroups []cloudprovider.ICloudSecurityGroup

	ID                  string `json:"id"`
	Name                string `json:"name"`
	CIDR                string `json:"cidr"`
	Status              string `json:"status"`
	EnterpriseProjectID string `json:"enterprise_project_id"`
}

func (self *SVpc) addWire(wire *SWire) {
	if self.iwires == nil {
		self.iwires = make([]cloudprovider.ICloudWire, 0)
	}
	self.iwires = append(self.iwires, wire)
}

func (self *SVpc) getWireByRegionId(regionId string) *SWire {
	if len(regionId) == 0 {
		return nil
	}

	for i := 0; i < len(self.iwires); i++ {
		wire := self.iwires[i].(*SWire)

		if wire.region.GetId() == regionId {
			return wire
		}
	}

	return nil
}

func (self *SVpc) fetchNetworks() error {
	networks, err := self.region.GetNetwroks(self.ID)
	if err != nil {
		return err
	}

	// ???????
	if len(networks) == 0 {
		self.iwires = append(self.iwires, &SWire{region: self.region, vpc: self})
		return nil
	}

	for i := 0; i < len(networks); i += 1 {
		wire := self.getWireByRegionId(self.region.GetId())
		networks[i].wire = wire
		wire.addNetwork(&networks[i])
	}
	return nil
}

// 华为云安全组可以被同region的VPC使用
func (self *SVpc) fetchSecurityGroups() error {
	secgroups, err := self.region.GetSecurityGroups("", "")
	if err != nil {
		return err
	}

	self.secgroups = make([]cloudprovider.ICloudSecurityGroup, len(secgroups))
	for i := 0; i < len(secgroups); i++ {
		self.secgroups[i] = &secgroups[i]
	}
	return nil
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
	new, err := self.region.getVpc(self.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SVpc) IsEmulated() bool {
	return false
}

func (self *SVpc) IsPublic() bool {
	return false
}

func (self *SVpc) GetRegion() cloudprovider.ICloudRegion {
	return self.region
}

func (self *SVpc) GetIsDefault() bool {
	// 华为云没有default vpc.
	return false
}

func (self *SVpc) GetCidrBlock() string {
	return self.CIDR
}

func (self *SVpc) GetIWires() ([]cloudprovider.ICloudWire, error) {
	if self.iwires == nil {
		err := self.fetchNetworks()
		if err != nil {
			return nil, err
		}
	}
	return self.iwires, nil
}

func (self *SVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	if self.secgroups == nil {
		err := self.fetchSecurityGroups()
		if err != nil {
			return nil, err
		}
	}
	return self.secgroups, nil
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
	// todo: 确定删除VPC的逻辑
	return self.region.DeleteVpc(self.GetId())
}

func (self *SVpc) GetIWireById(wireId string) (cloudprovider.ICloudWire, error) {
	if self.iwires == nil {
		err := self.fetchNetworks()
		if err != nil {
			return nil, err
		}
	}
	for i := 0; i < len(self.iwires); i += 1 {
		if self.iwires[i].GetGlobalId() == wireId {
			return self.iwires[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SVpc) GetINatGateways() ([]cloudprovider.ICloudNatGateway, error) {
	nats, err := self.region.GetNatGateways(self.GetId(), "")
	if err != nil {
		return nil, err
	}
	ret := make([]cloudprovider.ICloudNatGateway, len(nats))
	for i := 0; i < len(nats); i++ {
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

func (self *SRegion) getVpc(vpcId string) (*SVpc, error) {
	vpc := SVpc{}
	err := DoGet(self.ecsClient.Vpcs.Get, vpcId, nil, &vpc)
	if err != nil && strings.Contains(err.Error(), "RouterNotFound") {
		return nil, cloudprovider.ErrNotFound
	}
	vpc.region = self
	return &vpc, err
}

func (self *SRegion) DeleteVpc(vpcId string) error {
	if vpcId != "default" {
		secgroups, err := self.GetSecurityGroups(vpcId, "")
		if err != nil {
			return errors.Wrap(err, "GetSecurityGroups")
		}
		for _, secgroup := range secgroups {
			err = self.DeleteSecurityGroup(secgroup.ID)
			if err != nil {
				return errors.Wrapf(err, "DeleteSecurityGroup(%s)", secgroup.ID)
			}
		}
	}
	return DoDelete(self.ecsClient.Vpcs.Delete, vpcId, nil, nil)
}

// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0020090625.html
func (self *SRegion) GetVpcs() ([]SVpc, error) {
	querys := make(map[string]string)

	vpcs := make([]SVpc, 0)
	err := doListAllWithMarker(self.ecsClient.Vpcs.List, querys, &vpcs)
	if err != nil {
		return nil, err
	}

	for i := range vpcs {
		vpcs[i].region = self
	}
	return vpcs, err
}
