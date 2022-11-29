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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SVpc struct {
	multicloud.SVpc
	HcsTags

	region *SRegion

	Id                  string
	Name                string
	CIDR                string
	Status              string
	EnterpriseProjectId string
}

func (self *SRegion) GetVpcs() ([]SVpc, error) {
	ret := []SVpc{}
	return ret, self.vpcList("vpcs", nil, &ret)
}

func (self *SVpc) GetId() string {
	return self.Id
}

func (self *SVpc) GetName() string {
	if len(self.Name) > 0 {
		return self.Name
	}
	return self.Id
}

func (self *SVpc) GetGlobalId() string {
	return self.Id
}

func (self *SVpc) GetStatus() string {
	return api.VPC_STATUS_AVAILABLE
}

func (self *SVpc) Refresh() error {
	ret, err := self.region.GetVpc(self.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, ret)
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

func (self *SVpc) GetExternalAccessMode() string {
	return api.VPC_EXTERNAL_ACCESS_MODE_NONE
}

func (self *SVpc) GetIWires() ([]cloudprovider.ICloudWire, error) {
	wire := &SWire{region: self.region, vpc: self}
	return []cloudprovider.ICloudWire{wire}, nil
}

func (self *SVpc) GetIWireById(id string) (cloudprovider.ICloudWire, error) {
	wires, err := self.GetIWires()
	if err != nil {
		return nil, err
	}
	for i := range wires {
		if wires[i].GetGlobalId() == id {
			return wires[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	secgroups, err := self.region.GetSecurityGroups(self.Id)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudSecurityGroup{}
	for i := range secgroups {
		secgroups[i].region = self.region
		ret = append(ret, &secgroups[i])
	}
	return ret, nil
}

func (self *SVpc) GetIRouteTables() ([]cloudprovider.ICloudRouteTable, error) {
	rtbs, err := self.region.GetRouteTables(self.Id)
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

func (self *SVpc) GetINatGateways() ([]cloudprovider.ICloudNatGateway, error) {
	nats, err := self.region.GetNatGateways(self.GetId())
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

func (self *SVpc) GetAuthorityOwnerId() string {
	return self.region.client.projectId
}

func (self *SRegion) GetVpc(id string) (*SVpc, error) {
	vpc := &SVpc{region: self}
	res := fmt.Sprintf("vpcs/%s", id)
	return vpc, self.vpcGet(res, vpc)
}

func (self *SRegion) DeleteVpc(id string) error {
	if id != "default" {
		secgroups, err := self.GetSecurityGroups(id)
		if err != nil {
			return errors.Wrap(err, "GetSecurityGroups")
		}
		for _, secgroup := range secgroups {
			err = self.DeleteSecurityGroup(secgroup.Id)
			if err != nil {
				return errors.Wrapf(err, "DeleteSecurityGroup(%s)", secgroup.Id)
			}
		}
	}
	res := fmt.Sprintf("vpcs/%s", id)
	return self.vpcDelete(res)
}

func (self *SRegion) CreateVpc(name, cidr, desc string) (*SVpc, error) {
	params := map[string]interface{}{
		"vpc": map[string]string{
			"name":        name,
			"cidr":        cidr,
			"description": desc,
		},
	}
	vpc := &SVpc{region: self}
	return vpc, self.vpcCreate("vpcs", params, vpc)
}

func (self *SRegion) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
	vpcs, err := self.GetVpcs()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudVpc{}
	for i := range vpcs {
		vpcs[i].region = self
		ret = append(ret, &vpcs[i])
	}
	extVpcs, err := self.GetExternalVpcs()
	if err != nil {
		return nil, err
	}
	for i := range extVpcs {
		extVpcs[i].region = self
		ret = append(ret, &extVpcs[i])
	}
	return ret, nil
}

func (self *SRegion) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	vpc, err := self.GetVpc(id)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			extVpcs, err := self.GetExternalVpcs()
			if err == nil {
				for i := range extVpcs {
					if extVpcs[i].Id == id {
						extVpcs[i].region = self
						return &extVpcs[i], nil
					}
				}
			}
		}
		return nil, err
	}
	vpc.region = self
	return vpc, nil
}

func (self *SRegion) CreateIVpc(opts *cloudprovider.VpcCreateOptions) (cloudprovider.ICloudVpc, error) {
	return self.CreateVpc(opts.NAME, opts.CIDR, opts.Desc)
}
