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

package cloudpods

import (
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

type SStorage struct {
	multicloud.SStorageBase
	region *SRegion

	api.StorageDetails
}

func (self *SStorage) GetId() string {
	return self.Id
}

func (self *SStorage) GetGlobalId() string {
	return self.Id
}

func (self *SStorage) GetName() string {
	return self.Name
}

func (self *SStorage) GetStatus() string {
	return self.Status
}

func (self *SStorage) GetIZone() cloudprovider.ICloudZone {
	zone, err := self.region.GetZone(self.ZoneId)
	if err != nil {
		return nil
	}
	return zone
}

func (self *SStorage) GetMediumType() string {
	return self.MediumType
}

func (self *SStorage) GetCapacityMB() int64 {
	return self.Capacity
}

func (self *SStorage) GetStorageType() string {
	return self.StorageType
}

func (self *SStorage) GetCapacityUsedMB() int64 {
	return self.ActualCapacityUsed
}

func (self *SStorage) GetStorageConf() jsonutils.JSONObject {
	return self.StorageConf
}

func (self *SStorage) GetEnabled() bool {
	return self.Enabled != nil && *self.Enabled
}

func (self *SStorage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	if len(self.StoragecacheId) == 0 {
		return nil
	}
	cache, _ := self.region.GetStoragecache(self.StoragecacheId)
	return cache
}

func (self *SStorage) GetMountPoint() string {
	return ""
}

func (self *SStorage) IsSysDiskStore() bool {
	return self.SStorage.IsSysDiskStore != nil && *self.SStorage.IsSysDiskStore
}

func (self *SZone) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	storages, err := self.region.GetStorages(self.Id, "")
	if err != nil {
		return nil, errors.Wrapf(err, "GetStorages")
	}
	ret := []cloudprovider.ICloudStorage{}
	for i := range storages {
		storages[i].region = self.region
		ret = append(ret, &storages[i])
	}
	return ret, nil
}

func (self *SZone) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return self.region.GetIStorageById(id)
}

func (self *SRegion) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	storage, err := self.GetStorage(id)
	if err != nil {
		return nil, err
	}
	storage.region = self
	return storage, nil
}

func (self *SRegion) GetStorage(id string) (*SStorage, error) {
	storage := &SStorage{region: self}
	return storage, self.cli.get(&modules.Storages, id, nil, storage)
}

func (self *SRegion) GetStorages(zoneId, hostId string) ([]SStorage, error) {
	params := map[string]interface{}{}
	if len(zoneId) > 0 {
		params["zone_id"] = zoneId
	}
	if len(hostId) > 0 {
		params["host_id"] = hostId
	}
	storages := []SStorage{}
	return storages, self.list(&modules.Storages, params, &storages)
}
