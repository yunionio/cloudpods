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

func (self *SRegion) getAccountUser() string {
	quotas, err := self.GetQuotas()
	if err != nil {
		return ""
	}
	ownerId := ""
	if len(quotas) > 0 {
		ownerId = quotas[0].OwnerId
	}
	return ownerId
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
	var ret []SRegion
	return ret, resp.Unmarshal(&ret, "regionInfo")
}

func (self *SRegion) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	storages, err := self.getStorages()
	if err != nil {
		return nil, err
	}
	for i := range storages {
		if storages[i].GetGlobalId() == id {
			storages[i].cluster = &SCluster{region: self}
			return &storages[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SRegion) GetIStoragecaches() ([]cloudprovider.ICloudStoragecache, error) {
	var storages []SStorage
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
	var ret []cloudprovider.ICloudStoragecache
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
	var ret []cloudprovider.ICloudHost
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

func (self *SRegion) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots, err := self.getSnapshots("", "")
	if err != nil {
		return nil, err
	}
	iSnapshots := make([]cloudprovider.ICloudSnapshot, len(snapshots))
	for i := 0; i < len(snapshots); i++ {
		snapshots[i].region = self
		iSnapshots[i] = snapshots[i]
	}
	return iSnapshots, nil
}

func (self *SRegion) GetISnapshotById(id string) (cloudprovider.ICloudSnapshot, error) {
	snapshots, err := self.getSnapshots(id, "")
	if err != nil {
		return nil, err
	}
	for i := range snapshots {
		if snapshots[i].GetGlobalId() == id {
			snapshots[i].region = self
			return &snapshots[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	var groups []SSecurityGroup
	nextToken := ""
	for {
		part, _nextToken, err := self.GetSecurityGroups("", "", nextToken)
		if err != nil {
			return nil, err
		}
		groups = append(groups, part...)
		if len(_nextToken) == 0 || len(part) == 0 {
			break
		}
		nextToken = _nextToken
	}
	var ret []cloudprovider.ICloudSecurityGroup
	for i := range groups {
		groups[i].region = self
		ret = append(ret, &groups[i])
	}
	return ret, nil
}

func (region *SRegion) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	var vms []SInstance
	nextToken := ""
	for {
		part, _nextToken, err := region.GetInstances("", "", MAX_RESULT, nextToken)
		if err != nil {
			return nil, err
		}
		vms = append(vms, part...)
		if len(part) == 0 || len(_nextToken) == 0 {
			break
		}
		nextToken = _nextToken
	}
	var ret []cloudprovider.ICloudVM
	for i := range vms {
		ret = append(ret, &vms[i])
	}
	return ret, nil
}
