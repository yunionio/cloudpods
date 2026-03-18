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

package ecloud

import (
	"yunion.io/x/jsonutils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SVpc struct {
	multicloud.SVpc
	EcloudTags

	region *SRegion

	Id       string `json:"id"`
	Name     string `json:"name"`
	Region   string `json:"region"`
	EcStatus string `json:"ecStatus"`
	RouterId string `json:"routerId"`
	Scale    string `json:"scale"`
	UserId   string `json:"userId"`
	UserName string `json:"userName"`
}

func (v *SVpc) GetId() string {
	return v.Id
}

func (v *SVpc) GetName() string {
	return v.Name
}

func (v *SVpc) GetGlobalId() string {
	return v.GetId()
}

func (v *SVpc) GetStatus() string {
	switch v.EcStatus {
	case "ACTIVE":
		return api.VPC_STATUS_AVAILABLE
	case "DOWN", "BUILD", "ERROR":
		return api.VPC_STATUS_UNAVAILABLE
	case "PENDING_DELETE":
		return api.VPC_STATUS_DELETING
	case "PENDING_CREATE", "PENDING_UPDATE":
		return api.VPC_STATUS_PENDING
	default:
		return api.VPC_STATUS_UNKNOWN
	}
}

func (v *SVpc) Refresh() error {
	n, err := v.region.GetVpc(v.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(v, n)
}

func (v *SVpc) GetRegion() cloudprovider.ICloudRegion {
	return v.region
}

func (v *SVpc) GetIsDefault() bool {
	return false
}

func (v *SVpc) GetCidrBlock() string {
	return ""
}

func (v *SVpc) GetIWires() ([]cloudprovider.ICloudWire, error) {
	zones, err := v.region.GetZones()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudWire{}
	for i := range zones {
		ret = append(ret, &SWire{
			vpc:  v,
			zone: &zones[i],
		})
	}
	return ret, nil
}

func (v *SVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	// 移动云安全组为 region 维度，返回本 region 下全部安全组
	return v.region.GetISecurityGroups()
}

func (v *SVpc) GetIRouteTables() ([]cloudprovider.ICloudRouteTable, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (v *SVpc) GetIRouteTableById(routeTableId string) (cloudprovider.ICloudRouteTable, error) {
	return nil, nil
}

func (v *SVpc) Delete() error {
	return v.region.DeleteVpc(v.RouterId)
}

func (v *SVpc) GetIWireById(wireId string) (cloudprovider.ICloudWire, error) {
	iwires, err := v.GetIWires()
	if err != nil {
		return nil, err
	}
	for i := range iwires {
		if iwires[i].GetGlobalId() == wireId {
			return iwires[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}
