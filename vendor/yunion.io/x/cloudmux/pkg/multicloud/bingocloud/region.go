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
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SRegion struct {
	multicloud.SRegion
	multicloud.SRegionEipBase
	multicloud.SRegionLbBase
	multicloud.SRegionOssBase
	multicloud.SRegionSecurityGroupBase
	multicloud.SRegionVpcBase
	multicloud.SRegionZoneBase

	client *SBingoCloudClient

	RegionId       string
	RegionName     string
	Hypervisor     string
	NetworkMode    string
	RegionEndpoint string
}

func (self *SRegion) GetId() string {
	return self.RegionId
}

func (self *SRegion) GetName() string {
	return self.RegionName
}

func (self *SRegion) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", CLOUD_PROVIDER_BINGO_CLOUD, self.RegionId)
}

func (self *SRegion) GetClient() *SBingoCloudClient {
	return self.client
}

func (self *SRegion) GetCapabilities() []string {
	return self.client.GetCapabilities()
}

func (self *SRegion) GetI18n() cloudprovider.SModelI18nTable {
	return cloudprovider.SModelI18nTable{}
}

func (self *SRegion) GetProvider() string {
	return api.CLOUD_PROVIDER_BINGO_CLOUD
}

func (self *SRegion) GetStatus() string {
	return api.CLOUD_REGION_STATUS_INSERVER
}

func (self *SRegion) GetCloudEnv() string {
	return ""
}

func (self *SRegion) GetGeographicInfo() cloudprovider.SGeographicInfo {
	return cloudprovider.SGeographicInfo{}
}

func (self *SRegion) invoke(action string, params map[string]string) (jsonutils.JSONObject, error) {
	return self.client.invoke(action, params)
}

func (self *SBingoCloudClient) GetRegions() ([]SRegion, error) {
	resp, err := self.invoke("DescribeRegions", nil)
	if err != nil {
		return nil, err
	}
	ret := []SRegion{}
	err = resp.Unmarshal(&ret, "regionInfo")
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SRegion) GetIStoragecaches() ([]cloudprovider.ICloudStoragecache, error) {
	storages := []SStorage{}
	part, nextToken, err := self.GetStorages("")
	if err != nil {
		return nil, err
	}
	storages = append(storages, part...)
	for len(nextToken) > 0 {
		part, nextToken, err = self.GetStorages(nextToken)
		if err != nil {
			return nil, err
		}
		storages = append(storages, part...)
	}
	ret := []cloudprovider.ICloudStoragecache{}
	for i := range storages {
		cache := SStoragecache{
			region:      self,
			storageName: storages[i].StorageName,
			storageId:   storages[i].StorageId,
		}
		ret = append(ret, &cache)
	}
	return ret, nil
}

func (self *SRegion) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	caches, err := self.GetIStoragecaches()
	if err != nil {
		return nil, err
	}
	for i := range caches {
		if caches[i].GetGlobalId() == id {
			return caches[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SRegion) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	zones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudHost{}
	for i := range zones {
		hosts, err := zones[i].GetIHosts()
		if err != nil {
			return nil, err
		}
		ret = append(ret, hosts...)
	}
	return ret, nil
}

func (self *SRegion) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	zones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := range zones {
		hosts, err := zones[i].GetIHosts()
		if err != nil {
			return nil, err
		}
		for i := range hosts {
			if hosts[i].GetGlobalId() == id {
				return hosts[i], nil
			}
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}
