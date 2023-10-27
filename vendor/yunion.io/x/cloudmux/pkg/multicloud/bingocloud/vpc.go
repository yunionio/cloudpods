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

package bingocloud

import (
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SVpc struct {
	multicloud.SVpc
	multicloud.STagBase

	region *SRegion

	RequestId       string `json:"requestId"`
	VlanNum         string `json:"vlanNum"`
	AsGateway       string `json:"asGateway"`
	DhcpOptionsId   string `json:"dhcpOptionsId"`
	VpcName         string `json:"vpcName"`
	OwnerId         string `json:"ownerId"`
	WanCode         string `json:"wanCode"`
	Shared          string `json:"shared"`
	SubnetPolicy    string `json:"subnetPolicy"`
	Description     string `json:"description"`
	VpcId           string `json:"vpcId"`
	IsPublicNetwork string `json:"isPublicNetwork"`
	GatewayId       string `json:"gatewayId"`
	IsDefault       string `json:"isDefault"`
	Provider        string `json:"provider"`
	State           string `json:"state"`
	CidrBlock       string `json:"cidrBlock"`
	InstanceTenancy string `json:"instanceTenancy"`
}

func (self *SVpc) GetId() string {
	return self.VpcId
}

func (self *SVpc) GetGlobalId() string {
	return self.VpcId
}

func (self *SVpc) GetName() string {
	return self.VpcName
}

func (self *SVpc) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (self *SVpc) GetCidrBlock() string {
	return self.CidrBlock
}

func (self *SVpc) GetIRouteTableById(id string) (cloudprovider.ICloudRouteTable, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SVpc) GetIRouteTables() ([]cloudprovider.ICloudRouteTable, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SVpc) GetIsDefault() bool {
	return self.IsDefault == "true"
}

func (self *SVpc) GetRegion() cloudprovider.ICloudRegion {
	return self.region
}

func (self *SVpc) GetStatus() string {
	switch self.State {
	case "available":
		return api.VPC_STATUS_AVAILABLE
	default:
		return self.State
	}
}

func (self *SVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	return []cloudprovider.ICloudSecurityGroup{}, nil
}

func (self *SRegion) GetVpcs(id string) ([]SVpc, error) {
	params := map[string]string{}
	if len(id) > 0 {
		params["VpcId"] = id
	}

	resp, err := self.invoke("DescribeVpcs", params)
	if err != nil {
		return nil, err
	}
	var vpcs []SVpc

	return vpcs, resp.Unmarshal(&vpcs, "vpcSet")
}

func (self *SRegion) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
	vpcs, err := self.GetVpcs("")
	if err != nil {
		return nil, errors.Wrapf(err, "GetVpcs")
	}
	var ret []cloudprovider.ICloudVpc
	for i := range vpcs {
		vpcs[i].region = self
		ret = append(ret, &vpcs[i])
	}
	return ret, nil
}
