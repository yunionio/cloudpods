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

package cucloud

import (
	"fmt"
	"net/url"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

type SRegion struct {
	multicloud.SRegion
	multicloud.SNoObjectStorageRegion
	multicloud.SNoLbRegion
	client *SChinaUnionClient

	Id              string
	CloudRegionId   string
	CloudRegionName string
	CloudRegionCode string
	Status          string
	ProvinceName    string
	Area            string
}

func (self *SRegion) GetId() string {
	return self.CloudRegionCode
}

func (self *SRegion) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", api.CLOUD_PROVIDER_CUCLOUD, self.CloudRegionCode)
}

func (self *SRegion) GetProvider() string {
	return api.CLOUD_PROVIDER_CUCLOUD
}

func (self *SRegion) GetCloudEnv() string {
	return api.CLOUD_PROVIDER_CUCLOUD
}

func (self *SRegion) GetGeographicInfo() cloudprovider.SGeographicInfo {
	geo, ok := map[string]cloudprovider.SGeographicInfo{
		"cn-chongqing-1": api.RegionChongqing,
		"cn-wuhan-2":     api.RegionWuhan,
		"cn-guangzhou-1": api.RegionGuangzhou,
		"cn-beijing-1":   api.RegionBeijing,
	}[self.CloudRegionCode]
	if ok {
		return geo
	}
	return cloudprovider.SGeographicInfo{}
}

func (self *SRegion) GetName() string {
	return self.CloudRegionName
}

func (self *SRegion) GetI18n() cloudprovider.SModelI18nTable {
	table := cloudprovider.SModelI18nTable{}
	return table
}

func (self *SRegion) GetStatus() string {
	return api.CLOUD_REGION_STATUS_INSERVER
}

func (self *SRegion) GetClient() *SChinaUnionClient {
	return self.client
}

func (self *SRegion) CreateEIP(opts *cloudprovider.SEip) (cloudprovider.ICloudEIP, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) CreateISecurityGroup(conf *cloudprovider.SecurityGroupCreateInput) (cloudprovider.ICloudSecurityGroup, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetISecurityGroupById(id string) (cloudprovider.ICloudSecurityGroup, error) {
	group, err := region.GetSecurityGroup(id)
	if err != nil {
		return nil, err
	}
	group.region = region
	return group, nil
}

func (region *SRegion) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	secgroups, err := region.GetSecurityGroups("")
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudSecurityGroup{}
	for i := range secgroups {
		secgroups[i].region = region
		ret = append(ret, &secgroups[i])
	}
	return ret, nil
}

func (self *SRegion) CreateIVpc(opts *cloudprovider.VpcCreateOptions) (cloudprovider.ICloudVpc, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
	vpc, err := self.GetVpcs("")
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudVpc{}
	for i := range vpc {
		vpc[i].region = self
		ret = append(ret, &vpc[i])
	}
	return ret, nil
}

func (self *SRegion) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	vpc, err := self.GetVpc(id)
	if err != nil {
		return nil, err
	}
	return vpc, nil
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
	zones, err := self.GetZones()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudZone{}
	for i := range zones {
		zones[i].region = self
		ret = append(ret, &zones[i])
	}
	return ret, nil
}

func (self *SRegion) GetIZoneById(id string) (cloudprovider.ICloudZone, error) {
	zones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := range zones {
		if zones[i].GetGlobalId() == id {
			return zones[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (region *SRegion) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	zones, err := region.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := range zones {
		hosts, err := zones[i].GetIHosts()
		if err != nil {
			return nil, err
		}
		for j := range hosts {
			if hosts[j].GetGlobalId() == id {
				return hosts[j], nil
			}
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (region *SRegion) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	ret := []cloudprovider.ICloudHost{}
	zones, err := region.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := range zones {
		hosts, err := zones[i].GetIHosts()
		if err != nil {
			return nil, err
		}
		ret = append(ret, hosts...)
	}
	return ret, nil
}

func (region *SRegion) list(resource string, params url.Values) (jsonutils.JSONObject, error) {
	return region.client.list(resource, params)
}

func (region *SRegion) get(resource string) (jsonutils.JSONObject, error) {
	return region.client.get(resource)
}

func (region *SRegion) post(resource string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	return region.client.post(resource, params)
}
