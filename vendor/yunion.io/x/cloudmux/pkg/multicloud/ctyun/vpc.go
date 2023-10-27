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
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SVpc struct {
	multicloud.SVpc
	CtyunTags

	region *SRegion

	VpcId          string
	Name           string
	Description    string
	CIDR           string
	Ipv6Enabled    bool
	SubnetIDs      []string
	NatGatewayIDs  []string
	ProjectId      string
	Ipv6CIDRS      []string
	EnableIpv6     bool
	SecondaryCIDRS []string
}

func (self *SVpc) GetId() string {
	return self.VpcId
}

func (self *SVpc) GetName() string {
	if len(self.Name) > 0 {
		return self.Name
	}
	return self.VpcId
}

func (self *SVpc) GetGlobalId() string {
	return self.GetId()
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
	rts := []cloudprovider.ICloudRouteTable{}
	return rts, nil
}

func (self *SVpc) GetIRouteTableById(routeTableId string) (cloudprovider.ICloudRouteTable, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SVpc) Delete() error {
	return self.region.DeleteVpc(self.GetId())
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
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SRegion) GetVpc(vpcId string) (*SVpc, error) {
	params := map[string]interface{}{
		"vpcID": vpcId,
	}
	resp, err := self.list(SERVICE_VPC, "/v4/vpc/query", params)
	if err != nil {
		return nil, err
	}
	vpc := &SVpc{region: self}
	err = resp.Unmarshal(vpc, "returnObj")
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	return vpc, nil
}

func (self *SRegion) DeleteVpc(vpcId string) error {
	params := map[string]interface{}{
		"vpcID": vpcId,
	}
	_, err := self.post(SERVICE_VPC, "/v4/vpc/delete", params)
	return err
}

func (self *SRegion) GetVpcs() ([]SVpc, error) {
	pageNo := 1
	params := map[string]interface{}{
		"pageNo":   pageNo,
		"pageSize": 50,
	}
	ret := []SVpc{}
	for {
		resp, err := self.list(SERVICE_VPC, "/v4/vpc/list", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			ReturnObj struct {
				Vpcs []SVpc
			}
			TotalCount int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.ReturnObj.Vpcs...)
		if len(ret) >= part.TotalCount || len(part.ReturnObj.Vpcs) == 0 {
			break
		}
		pageNo++
		params["pageNo"] = pageNo
	}
	return ret, nil
}
