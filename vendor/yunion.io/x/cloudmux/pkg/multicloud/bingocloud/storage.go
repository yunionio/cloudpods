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
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SStorage struct {
	multicloud.STagBase
	multicloud.SStorageBase
	cluster *SCluster

	StorageId    string `json:"storageId"`
	Location     string `json:"location"`
	UsedBy       string `json:"usedBy"`
	SpaceUsed    int64  `json:"spaceUsed"`
	StorageName  string `json:"storageName"`
	FileFormat   string `json:"fileFormat"`
	Disabled     string `json:"disabled"`
	SpaceMax     int64  `json:"spaceMax"`
	StorageType  string `json:"storageType"`
	DrCloudId    string `json:"drCloudId"`
	ParameterSet []struct {
		Name  string
		Value string
	} `json:"parameterSet"`
	ClusterId    string `json:"clusterId"`
	IsDRStorage  string `json:"isDRStorage"`
	ResUsage     string `json:"resUsage"`
	ScheduleTags string `json:"scheduleTags"`
}

func (self *SStorage) GetName() string {
	return self.StorageName
}

func (self *SStorage) GetId() string {
	return self.StorageId
}

func (self *SStorage) GetGlobalId() string {
	return self.GetId()
}

func (self *SStorage) CreateIDisk(conf *cloudprovider.DiskCreateConfig) (cloudprovider.ICloudDisk, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SStorage) DisableSync() bool {
	return false
}

func (self *SStorage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return &SStoragecache{storageId: self.StorageId, storageName: self.StorageName, region: self.cluster.region}
}

func (self *SStorage) GetIZone() cloudprovider.ICloudZone {
	return self.cluster
}

func (self *SStorage) GetStorageType() string {
	return self.StorageType
}

func (self *SStorage) GetMediumType() string {
	return api.DISK_TYPE_SSD
}

func (self *SStorage) GetCapacityMB() int64 {
	return self.SpaceMax * 1024
}

func (self *SStorage) GetCapacityUsedMB() int64 {
	return self.SpaceUsed * 1024
}

func (self *SStorage) GetMountPoint() string {
	return ""
}

func (self *SStorage) GetStatus() string {
	return api.STORAGE_ONLINE
}

func (self *SStorage) GetStorageConf() jsonutils.JSONObject {
	return jsonutils.Marshal(self.ParameterSet)
}

func (self *SStorage) GetEnabled() bool {
	return self.Disabled == "false"
}

func (self *SStorage) IsSysDiskStore() bool {
	return true
}

func (self *SRegion) GetStorages(nextToken string) ([]SStorage, string, error) {
	params := map[string]string{}
	if len(nextToken) > 0 {
		params["nextToken"] = nextToken
	}
	resp, err := self.invoke("DescribeStorages", params)
	if err != nil {
		return nil, "", err
	}
	ret := struct {
		NextToken  string
		StorageSet []SStorage
	}{}
	resp.Unmarshal(&ret)
	return ret.StorageSet, ret.NextToken, nil
}

func (self *SCluster) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	storages, err := self.GetIStorages()
	if err != nil {
		return nil, err
	}
	for i := range storages {
		if storages[i].GetGlobalId() == id {
			return storages[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SCluster) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	storages, err := self.region.getStorages()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudStorage{}
	for i := range storages {
		storages[i].cluster = self
		ret = append(ret, &storages[i])
	}
	return ret, nil
}

func (self *SRegion) getStorages() ([]SStorage, error) {
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
	return storages, nil
}
