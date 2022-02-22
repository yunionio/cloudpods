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
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SVpc struct {
	CidrBlock       string `json:"cidrBlock"`
	IsPublicNetwork string `json:"isPublicNetwork"`
	VpcId           string `json:"vpcId"`
	State           string `json:"state"`
	VpcName         string `json:"vpcName"`
	WanCode         string `json:"wanCode"`
	SubnetPolicy    string `json:"subnetPolicy"`
	InstanceTenancy string `json:"instanceTenancy"`
	IsDefault       string `json:"isDefault"`
	Shared          string `json:"shared"`
	VlanNum         string `json:"vlanNum"`
	GatewayId       string `json:"gatewayId"`
	AsGateway       string `json:"asGateway"`
	DhcpOptionsId   string `json:"dhcpOptionsId"`
	OwnerId         string `json:"ownerId"`
	Provider        string `json:"provider"`
	Description     string `json:"description"`
}

func (self *SRegion) GetVpcs() ([]SVpc, error) {
	resp, err := self.invoke("DescribeVpcs", nil)
	if err != nil {
		return nil, err
	}
	log.Errorf("resp=:%s", resp)
	result := struct {
		VpcSet struct {
			Item []SVpc
		}
	}{}
	err = resp.Unmarshal(&result)
	if err != nil {
		return nil, err
	}
	return result.VpcSet.Item, nil
}

func (self *SRegion) GetVpc(id string) (*SVpc, error) {
	vpcs := &SVpc{}
	// return storage, self.get("storage_containers", id, nil, storage)
	return vpcs, cloudprovider.ErrNotImplemented
}
