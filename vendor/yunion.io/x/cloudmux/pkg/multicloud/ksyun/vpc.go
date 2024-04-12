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

package ksyun

import (
	"fmt"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"

	"yunion.io/x/pkg/errors"
)

type SVpc struct {
	multicloud.SVpc
	SKsTag

	region *SRegion

	IsDefault             bool   `json:"IsDefault"`
	VpcID                 string `json:"VpcId"`
	CreateTime            string `json:"CreateTime"`
	CidrBlock             string `json:"CidrBlock"`
	VpcName               string `json:"VpcName"`
	ProvidedIpv6CidrBlock bool   `json:"ProvidedIpv6CidrBlock"`
}

func (region *SRegion) GetVpcs(ids []string) ([]SVpc, error) {
	param := map[string]string{
		"MaxResults": "1000",
	}
	for i, vpcId := range ids {
		param[fmt.Sprintf("VpcId.%d", i+1)] = vpcId
	}
	vpcs := []SVpc{}
	for {
		resp, err := region.vpcRequest("DescribeVpcs", param)
		if err != nil {
			return nil, errors.Wrap(err, "list instance")
		}
		part := []SVpc{}
		err = resp.Unmarshal(&part, "VpcSet")
		if err != nil {
			return nil, errors.Wrap(err, "unmarshal instances")
		}
		vpcs = append(vpcs, part...)
		nextToken, err := resp.GetString("NextToken")
		if err != nil {
			break
		}
		param["NextToken"] = nextToken
	}
	return vpcs, nil
}

func (region *SRegion) GetVpc(id string) (*SVpc, error) {
	vpcs, err := region.GetVpcs([]string{id})
	if err != nil {
		return nil, errors.Wrap(err, "GetVpcs")
	}
	for _, vpc := range vpcs {
		if vpc.GetGlobalId() == id {
			return &vpc, nil
		}
	}
	return nil, errors.Wrapf(err, "vpc id:%s", id)
}

func (vpc *SVpc) GetId() string {
	return vpc.VpcID
}

func (vpc *SVpc) GetName() string {
	if len(vpc.VpcName) > 0 {
		return vpc.VpcName
	}
	return vpc.VpcID
}

func (vpc *SVpc) GetGlobalId() string {
	return vpc.VpcID
}

func (vpc *SVpc) GetStatus() string {
	return api.VPC_STATUS_AVAILABLE
}

func (vpc *SVpc) Refresh() error {
	extVpc, err := vpc.region.GetVpc(vpc.GetGlobalId())
	if err != nil {
		return errors.Wrap(err, "GetVpc")
	}
	return jsonutils.Update(vpc, extVpc)
}

func (vpc *SVpc) GetRegion() cloudprovider.ICloudRegion {
	return vpc.region
}

func (vpc *SVpc) GetIsDefault() bool {
	return vpc.IsDefault
}

func (vpc *SVpc) GetCidrBlock() string {
	return vpc.CidrBlock
}

func (vpc *SVpc) GetIWires() ([]cloudprovider.ICloudWire, error) {
	zones, err := vpc.region.GetZones()
	if err != nil {
		return nil, errors.Wrap(err, "GetZones")
	}
	for i := range zones {
		zones[i].region = vpc.region
	}
	wires := []cloudprovider.ICloudWire{}
	for i := 0; i < len(zones); i++ {
		wire := SWire{
			vpc:  vpc,
			zone: &zones[i],
		}
		wires = append(wires, &wire)
	}
	return wires, nil
}

func (vpc *SVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	secgroups, err := vpc.region.GetSecurityGroups(vpc.VpcID, nil)
	if err != nil {
		return nil, errors.Wrap(err, "GetSecurityGroups")
	}
	isecgroups := []cloudprovider.ICloudSecurityGroup{}
	for i := range secgroups {
		secgroups[i].region = vpc.region
		isecgroups = append(isecgroups, &secgroups[i])
	}
	return isecgroups, nil
}

func (vpc *SVpc) GetIRouteTables() ([]cloudprovider.ICloudRouteTable, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (vpc *SVpc) GetIRouteTableById(routeTableId string) (cloudprovider.ICloudRouteTable, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (vpc *SVpc) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (vpc *SVpc) GetTags() (map[string]string, error) {
	tags, err := vpc.region.ListTags("vpc", vpc.VpcID)
	if err != nil {
		return nil, err
	}
	return tags.GetTags(), nil
}

func (vpc *SVpc) GetIWireById(wireId string) (cloudprovider.ICloudWire, error) {
	wires, err := vpc.GetIWires()
	if err != nil {
		return nil, errors.Wrap(err, "vpc.GetIWires")
	}
	for _, wire := range wires {
		if wire.GetGlobalId() == wireId {
			return wire, nil
		}
	}
	return nil, errors.Wrapf(errors.ErrNotFound, "wire id:%s", wireId)
}

func (vpc *SVpc) GetINatGateways() ([]cloudprovider.ICloudNatGateway, error) {
	return nil, cloudprovider.ErrNotImplemented

}

func (vpc *SVpc) GetICloudVpcPeeringConnections() ([]cloudprovider.ICloudVpcPeeringConnection, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (vpc *SVpc) GetICloudAccepterVpcPeeringConnections() ([]cloudprovider.ICloudVpcPeeringConnection, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (vpc *SVpc) GetICloudVpcPeeringConnectionById(id string) (cloudprovider.ICloudVpcPeeringConnection, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (vpc *SVpc) CreateICloudVpcPeeringConnection(opts *cloudprovider.VpcPeeringConnectionCreateOptions) (cloudprovider.ICloudVpcPeeringConnection, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (vpc *SVpc) AcceptICloudVpcPeeringConnection(id string) error {
	return cloudprovider.ErrNotImplemented
}

func (vpc *SVpc) GetAuthorityOwnerId() string {
	return ""
}

func (vpc *SRegion) DeleteVpc(vpcId string) error {
	return cloudprovider.ErrNotImplemented
}
