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

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

var regions = map[string]string{
	"bj":  "北京",
	"gz":  "广州",
	"su":  "苏州",
	"hkg": "香港",
	"fwh": "武汉",
	"bd":  "保定",
	"sin": "新加坡",
	"fsh": "上海",
}

type SRegion struct {
	multicloud.SRegion
	multicloud.SNoObjectStorageRegion
	multicloud.SNoLbRegion
	client *SBaiduClient

	Region     string
	RegionName string
}

func (self *SRegion) GetId() string {
	return self.Region
}

func (self *SRegion) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", api.CLOUD_PROVIDER_BAIDU, self.Region)
}

func (self *SRegion) GetProvider() string {
	return api.CLOUD_PROVIDER_BAIDU
}

func (self *SRegion) GetCloudEnv() string {
	return api.CLOUD_PROVIDER_BAIDU
}

func (self *SRegion) GetGeographicInfo() cloudprovider.SGeographicInfo {
	geo, ok := map[string]cloudprovider.SGeographicInfo{
		"bj":  api.RegionBeijing,
		"gz":  api.RegionGuangzhou,
		"su":  api.RegionSuzhou,
		"hkg": api.RegionHongkong,
		"fwh": api.RegionHangzhou,
		"bd":  api.RegionBaoDing,
		"sin": api.RegionSingapore,
		"fsh": api.RegionShanghai,
	}[self.Region]
	if ok {
		return geo
	}
	return cloudprovider.SGeographicInfo{}
}

func (self *SRegion) GetName() string {
	return self.RegionName
}

func (self *SRegion) GetI18n() cloudprovider.SModelI18nTable {
	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(self.GetName()).CN(self.GetName()).EN(self.Region)
	return table
}

func (self *SRegion) GetStatus() string {
	return api.CLOUD_REGION_STATUS_INSERVER
}

func (self *SRegion) GetClient() *SBaiduClient {
	return self.client
}

func (self *SRegion) CreateEIP(opts *cloudprovider.SEip) (cloudprovider.ICloudEIP, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) CreateISecurityGroup(conf *cloudprovider.SecurityGroupCreateInput) (cloudprovider.ICloudSecurityGroup, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetISecurityGroupById(secgroupId string) (cloudprovider.ICloudSecurityGroup, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetISecurityGroupByName(opts *cloudprovider.SecurityGroupFilterOptions) (cloudprovider.ICloudSecurityGroup, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) CreateIVpc(opts *cloudprovider.VpcCreateOptions) (cloudprovider.ICloudVpc, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetCapabilities() []string {
	return region.client.GetCapabilities()
}

func (self *SRegion) GetIEipById(eipId string) (cloudprovider.ICloudEIP, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetIZones() ([]cloudprovider.ICloudZone, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetIZoneById(id string) (cloudprovider.ICloudZone, error) {
	return nil, cloudprovider.ErrNotImplemented
}
