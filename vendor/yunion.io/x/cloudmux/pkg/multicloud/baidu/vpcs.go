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

package baidu

import (
	"fmt"
	"net/url"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
)

type SVpc struct {
	multicloud.SVpc
	SBaiduTag

	region *SRegion

	IsDefault bool   `json:"IsDefault"`
	VpcId     string `json:"vpcId"`
	Name      string `json:"name"`

	CreateTime string `json:"CreateTime"`
	Cidr       string `json:"Cidr"`
}

func (region *SRegion) GetVpcs() ([]SVpc, error) {
	params := url.Values{}
	vpcs := []SVpc{}
	for {
		resp, err := region.bccList("v1/vpc", params)
		if err != nil {
			return nil, errors.Wrap(err, "list vpcs")
		}
		part := struct {
			Vpcs       []SVpc
			NextMarker string
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrap(err, "Unmarshal")
		}
		for i := range part.Vpcs {
			part.Vpcs[i].region = region
			vpcs = append(vpcs, part.Vpcs[i])
		}
		if len(part.NextMarker) == 0 {
			break
		}
		params.Set("marker", part.NextMarker)
	}
	return vpcs, nil
}

func (region *SRegion) GetVpc(vpcId string) (*SVpc, error) {
	resp, err := region.bccList(fmt.Sprintf("v1/vpc/%s", vpcId), nil)
	if err != nil {
		return nil, errors.Wrap(err, "list vpcs")
	}
	vpc := SVpc{region: region}
	err = resp.Unmarshal(&vpc, "vpc")
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal vpc")
	}
	return &vpc, nil
}

func (vpc *SVpc) GetId() string {
	return vpc.VpcId
}

func (vpc *SVpc) GetName() string {
	if len(vpc.Name) > 0 {
		return vpc.Name
	}
	return vpc.VpcId
}

func (vpc *SVpc) GetGlobalId() string {
	return vpc.VpcId
}

func (vpc *SVpc) GetStatus() string {
	return api.VPC_STATUS_AVAILABLE
}

func (vpc *SVpc) Refresh() error {
	res, err := vpc.region.GetVpc(vpc.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(vpc, res)
}

func (vpc *SVpc) GetRegion() cloudprovider.ICloudRegion {
	return vpc.region
}

func (vpc *SVpc) GetIsDefault() bool {
	return vpc.IsDefault
}

func (vpc *SVpc) GetCidrBlock() string {
	return vpc.Cidr
}

func (vpc *SVpc) GetIWires() ([]cloudprovider.ICloudWire, error) {
	zones, err := vpc.region.GetZones()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudWire{}
	for i := range zones {
		wire := &SWire{vpc: vpc, zone: &zones[i]}
		ret = append(ret, wire)
	}
	return ret, nil
}

func (vpc *SVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	groups, err := vpc.region.GetSecurityGroups(vpc.VpcId)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudSecurityGroup{}
	for i := range groups {
		groups[i].region = vpc.region
		ret = append(ret, &groups[i])
	}
	return ret, nil
}

func (vpc *SVpc) GetIRouteTables() ([]cloudprovider.ICloudRouteTable, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (vpc *SVpc) GetIRouteTableById(routeTableId string) (cloudprovider.ICloudRouteTable, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (vpc *SVpc) Delete() error {
	return vpc.region.DeleteVpc(vpc.VpcId)
}

func (vpc *SVpc) GetIWireById(id string) (cloudprovider.ICloudWire, error) {
	wires, err := vpc.GetIWires()
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

func (region *SRegion) DeleteVpc(vpcId string) error {
	params := url.Values{}
	params.Set("clientToken", utils.GenRequestId(20))
	_, err := region.bccDelete(fmt.Sprintf("v1/vpc/%s", vpcId), params)
	return err
}

func (region *SRegion) CreateVpc(opts *cloudprovider.VpcCreateOptions) (*SVpc, error) {
	params := url.Values{}
	params.Set("clientToken", utils.GenRequestId(20))
	body := map[string]interface{}{
		"name":        opts.NAME,
		"description": opts.Desc,
		"cidr":        opts.CIDR,
	}
	resp, err := region.bccPost("v1/vpc", params, body)
	if err != nil {
		return nil, err
	}
	vpcId, err := resp.GetString("vpcId")
	if err != nil {
		return nil, err
	}
	return region.GetVpc(vpcId)
}
