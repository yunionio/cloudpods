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

package rockbase

import (
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SVPC struct {
	multicloud.SVpc
	RockbaseTags

	region *SRegion

	CreateTime  int64         `json:"CreateTime"`
	Name        string        `json:"Name"`
	Network     []string      `json:"Network"`
	NetworkInfo []NetworkInfo `json:"NetworkInfo"`
	SubnetCount int           `json:"SubnetCount"`
	Tag         string        `json:"Tag"`
	UpdateTime  int64         `json:"UpdateTime"`
	VpcId       string        `json:"VPCId"`
}

type NetworkInfo struct {
	Network     string `json:"Network"`
	SubnetCount int    `json:"SubnetCount"`
}

func (self *SVPC) GetId() string {
	return self.VpcId
}

func (self *SVPC) GetName() string {
	if len(self.Name) > 0 {
		return self.Name
	}
	return self.VpcId
}

func (self *SVPC) GetGlobalId() string {
	return self.GetId()
}

func (self *SVPC) GetStatus() string {
	return api.VPC_STATUS_AVAILABLE
}

func (self *SVPC) Refresh() error {
	new, err := self.region.GetVpc(self.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SVPC) IsEmulated() bool {
	return false
}

func (self *SVPC) GetRegion() cloudprovider.ICloudRegion {
	return self.region
}

func (self *SVPC) GetIsDefault() bool {
	return false
}

func (self *SVPC) GetCidrBlock() string {
	return strings.Join(self.Network, ",")
}

func (self *SVPC) GetIWires() ([]cloudprovider.ICloudWire, error) {
	return []cloudprovider.ICloudWire{&SWire{vpc: self}}, nil
}

// 由于Ucloud 安全组和vpc没有直接关联，这里是返回同一个项目下的防火墙列表，会导致重复同步的问题。
// https://docs.ucloud.cn/api/unet-api/grant_firewall
func (self *SVPC) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	return []cloudprovider.ICloudSecurityGroup{}, nil
}

func (self *SVPC) GetIRouteTables() ([]cloudprovider.ICloudRouteTable, error) {
	rts := []cloudprovider.ICloudRouteTable{}
	return rts, nil
}

func (self *SVPC) GetIRouteTableById(routeTableId string) (cloudprovider.ICloudRouteTable, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SVPC) Delete() error {
	return self.region.DeleteVpc(self.GetId())
}

func (self *SVPC) GetIWireById(wireId string) (cloudprovider.ICloudWire, error) {
	wires, err := self.GetIWires()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(wires); i += 1 {
		if wires[i].GetGlobalId() == wireId {
			return wires[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetVpc(vpcId string) (*SVPC, error) {
	if len(vpcId) == 0 {
		return nil, errors.Wrap(cloudprovider.ErrNotFound, "empty vpc id")
	}
	vpcs, err := self.GetVpcs(vpcId)
	if err != nil {
		return nil, errors.Wrap(err, "GetVpcs")
	}
	for i := range vpcs {
		if vpcs[i].VpcId == vpcId {
			vpcs[i].region = self
			return &vpcs[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "%s", vpcId)
}

// https://docs.ucloud.cn/api/vpc2.0-api/delete_vpc
func (self *SRegion) DeleteVpc(vpcId string) error {
	params := NewRockbaseParams()
	params.Set("VPCId", vpcId)
	return self.DoAction("DeleteVPC", params, nil)
}

// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0020090625.html
func (self *SRegion) GetVpcs(vpcId string) ([]SVPC, error) {
	vpcs := make([]SVPC, 0)
	params := NewRockbaseParams()
	if len(vpcId) > 0 {
		params.Set("VPCIds.0", vpcId)
	}

	err := self.DoListAll("DescribeVPC", params, &vpcs)
	return vpcs, err
}

func (self *SRegion) GetNetworks(vpcId string) ([]SNetwork, error) {
	params := NewRockbaseParams()
	if len(vpcId) > 0 {
		params.Set("VPCId", vpcId)
	}

	networks := make([]SNetwork, 0)
	err := self.DoAction("DescribeSubnet", params, &networks)
	return networks, err
}
