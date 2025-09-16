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
)

var regions = map[string]string{
	"bj":  "北京",
	"cd":  "成都",
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

	RegionId       string
	RegionName     string
	RegionEndpoint string
}

func (region *SRegion) GetId() string {
	return region.RegionId
}

func (region *SRegion) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", api.CLOUD_PROVIDER_BAIDU, region.RegionId)
}

func (region *SRegion) GetProvider() string {
	return api.CLOUD_PROVIDER_BAIDU
}

func (region *SRegion) GetCloudEnv() string {
	return api.CLOUD_PROVIDER_BAIDU
}

func (region *SRegion) GetGeographicInfo() cloudprovider.SGeographicInfo {
	geo, ok := map[string]cloudprovider.SGeographicInfo{
		"bj":  api.RegionBeijing,
		"gz":  api.RegionGuangzhou,
		"su":  api.RegionSuzhou,
		"hkg": api.RegionHongkong,
		"fwh": api.RegionHangzhou,
		"bd":  api.RegionBaoDing,
		"sin": api.RegionSingapore,
		"fsh": api.RegionShanghai,
		"cd":  api.RegionChengdu,
	}[region.RegionId]
	if ok {
		return geo
	}
	return cloudprovider.SGeographicInfo{}
}

func (region *SRegion) GetName() string {
	if name, ok := regions[region.RegionId]; ok {
		return name
	}
	return region.RegionName
}

func (region *SRegion) GetI18n() cloudprovider.SModelI18nTable {
	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(region.GetName()).CN(region.GetName()).EN(region.RegionName)
	return table
}

func (region *SRegion) GetStatus() string {
	return api.CLOUD_REGION_STATUS_INSERVER
}

func (region *SRegion) GetClient() *SBaiduClient {
	return region.client
}

func (region *SRegion) CreateEIP(opts *cloudprovider.SEip) (cloudprovider.ICloudEIP, error) {
	eip, err := region.CreateEip(opts)
	if err != nil {
		return nil, err
	}
	return eip, nil
}

func (region *SRegion) CreateISecurityGroup(opts *cloudprovider.SecurityGroupCreateInput) (cloudprovider.ICloudSecurityGroup, error) {
	group, err := region.CreateSecurityGroup(opts)
	if err != nil {
		return nil, err
	}
	return group, nil
}

func (region *SRegion) GetISecurityGroupById(id string) (cloudprovider.ICloudSecurityGroup, error) {
	group, err := region.GetSecurityGroup(id)
	if err != nil {
		return nil, err
	}
	return group, nil
}

func (region *SRegion) CreateIVpc(opts *cloudprovider.VpcCreateOptions) (cloudprovider.ICloudVpc, error) {
	vpc, err := region.CreateVpc(opts)
	if err != nil {
		return nil, err
	}
	return vpc, nil
}

func (region *SRegion) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
	vpcs, err := region.GetVpcs()
	if err != nil {
		return nil, errors.Wrapf(err, "GetVpcs")
	}
	ret := []cloudprovider.ICloudVpc{}
	for i := range vpcs {
		vpcs[i].region = region
		ret = append(ret, &vpcs[i])
	}
	return ret, nil
}

func (region *SRegion) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	vpc, err := region.GetVpc(id)
	if err != nil {
		return nil, err
	}
	return vpc, nil
}

func (region *SRegion) GetCapabilities() []string {
	return region.client.GetCapabilities()
}

func (region *SRegion) GetIEipById(id string) (cloudprovider.ICloudEIP, error) {
	eip, err := region.GetEip(id)
	if err != nil {
		return nil, err
	}
	return eip, nil
}

func (region *SRegion) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	eips, err := region.GetEips("")
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudEIP{}
	for i := range eips {
		eips[i].region = region
		ret = append(ret, &eips[i])
	}
	return ret, nil
}

func (region *SRegion) GetIZones() ([]cloudprovider.ICloudZone, error) {
	zones, err := region.GetZones()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudZone{}
	for i := range zones {
		ret = append(ret, &zones[i])
	}
	return ret, nil
}

func (region *SRegion) GetIZoneById(id string) (cloudprovider.ICloudZone, error) {
	zones, err := region.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := range zones {
		if zones[i].GetId() == id || zones[i].GetGlobalId() == id {
			return zones[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (region *SRegion) getStoragecache() *SStoragecache {
	return &SStoragecache{region: region}
}

func (region *SRegion) GetIStoragecaches() ([]cloudprovider.ICloudStoragecache, error) {
	return []cloudprovider.ICloudStoragecache{region.getStoragecache()}, nil
}

func (region *SRegion) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	caches, err := region.GetIStoragecaches()
	if err != nil {
		return nil, err
	}
	for i := range caches {
		if caches[i].GetGlobalId() == id {
			return caches[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (region *SRegion) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	zones, err := region.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(zones); i += 1 {
		ihost, err := zones[i].GetIHostById(id)
		if err == nil {
			return ihost, nil
		} else if errors.Cause(err) != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (region *SRegion) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	zones, err := region.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(zones); i += 1 {
		istore, err := zones[i].GetIStorageById(id)
		if err == nil {
			return istore, nil
		} else if errors.Cause(err) != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (region *SRegion) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	ret := make([]cloudprovider.ICloudHost, 0)

	zones, err := region.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(zones); i += 1 {
		iZoneHost, err := zones[i].GetIHosts()
		if err != nil {
			return nil, err
		}
		ret = append(ret, iZoneHost...)
	}
	return ret, nil
}

func (region *SRegion) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	ret := make([]cloudprovider.ICloudStorage, 0)

	zones, err := region.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(zones); i += 1 {
		iZoneStores, err := zones[i].GetIStorages()
		if err != nil {
			return nil, err
		}
		ret = append(ret, iZoneStores...)
	}
	return ret, nil
}

func (region *SRegion) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	vms, err := region.GetInstances("", nil)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudVM{}
	for i := range vms {
		ret = append(ret, &vms[i])
	}
	return ret, nil
}

func (region *SRegion) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	return region.GetInstance(id)
}

func (region *SRegion) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	return region.GetDisk(id)
}

func (region *SRegion) bccList(resource string, params url.Values) (jsonutils.JSONObject, error) {
	return region.client.bccList(region.RegionId, resource, params)
}

func (region *SRegion) bccPost(resource string, params url.Values, body map[string]interface{}) (jsonutils.JSONObject, error) {
	return region.client.bccPost(region.RegionId, resource, params, body)
}

func (region *SRegion) bccDelete(resource string, params url.Values) (jsonutils.JSONObject, error) {
	return region.client.bccDelete(region.RegionId, resource, params)
}

func (region *SRegion) bccUpdate(resource string, params url.Values, body map[string]interface{}) (jsonutils.JSONObject, error) {
	return region.client.bccUpdate(region.RegionId, resource, params, body)
}

func (region *SRegion) eipList(resource string, params url.Values) (jsonutils.JSONObject, error) {
	return region.client.eipList(region.RegionId, resource, params)
}

func (region *SRegion) eipPost(resource string, params url.Values, body map[string]interface{}) (jsonutils.JSONObject, error) {
	return region.client.eipPost(region.RegionId, resource, params, body)
}

func (region *SRegion) eipDelete(resource string, params url.Values) (jsonutils.JSONObject, error) {
	return region.client.eipDelete(region.RegionId, resource, params)
}

func (region *SRegion) eipUpdate(resource string, params url.Values, body map[string]interface{}) (jsonutils.JSONObject, error) {
	return region.client.eipUpdate(region.RegionId, resource, params, body)
}
