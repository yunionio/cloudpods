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

package proxmox

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SStorage struct {
	multicloud.SStorageBase
	ProxmoxTags

	zone *SZone

	Storage string `json:"storage"`
	Status  string
	Id      string
	Node    string

	Shared     int    `json:"shared"`
	Content    string `json:"content"`
	MaxDisk    int64  `json:"maxdisk"`
	Disk       int64  `json:"disk"`
	PluginType string `json:"plugintype"`
}

func (self *SStorage) GetName() string {
	if self.Shared == 0 {
		return fmt.Sprintf("%s-%s", self.Node, self.Storage)
	}
	return self.Storage
}

func (self *SStorage) GetId() string {
	if self.Shared == 0 {
		return self.Id
	}
	return self.Storage
}

func (self *SStorage) GetGlobalId() string {
	return self.GetId()
}

func (self *SStorage) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks, err := self.zone.region.GetDisks(self.Node, self.Storage)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudDisk{}
	for i := range disks {
		disks[i].storage = self
		ret = append(ret, &disks[i])
	}
	return ret, nil
}

func (self *SStorage) CreateIDisk(conf *cloudprovider.DiskCreateConfig) (cloudprovider.ICloudDisk, error) {
	return nil, cloudprovider.ErrNotFound
}

func (self *SStorage) GetCapacityMB() int64 {
	return int64(self.MaxDisk / 1024 / 1024)
}

func (self *SStorage) GetCapacityUsedMB() int64 {
	return int64(self.Disk / 1024 / 1024)
}

func (self *SStorage) GetEnabled() bool {
	if strings.Contains(self.Content, "images") {
		return true
	}
	return false
}

func (self *SStorage) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	disks, err := self.GetIDisks()
	if err != nil {
		return nil, err
	}
	for i := range disks {
		if disks[i].GetGlobalId() == id {
			return disks[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SStorage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return &SStoragecache{
		region:  self.zone.region,
		Node:    self.Node,
		isShare: self.Shared == 1,
	}
}

func (self *SStorage) GetMediumType() string {
	return api.DISK_TYPE_SSD
}

func (self *SStorage) GetMountPoint() string {
	return ""
}

func (self *SStorage) GetStatus() string {
	if self.Status != "available" {
		return api.STORAGE_OFFLINE
	}
	return api.STORAGE_ONLINE
}

func (self *SStorage) Refresh() error {
	ret, err := self.zone.region.GetStorage(self.GetGlobalId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, ret)
}

func (self *SStorage) GetIZone() cloudprovider.ICloudZone {
	return self.zone
}

func (self *SStorage) GetStorageConf() jsonutils.JSONObject {
	return jsonutils.NewDict()
}

func (self *SStorage) GetStorageType() string {
	return strings.ToLower(self.PluginType)
}

func (self *SStorage) IsSysDiskStore() bool {
	return true
}

func (self *SRegion) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	storage, err := self.GetStorage(id)
	if err != nil {
		return nil, err
	}
	zone, err := self.GetZone()
	if err != nil {
		return nil, err
	}
	storage.zone = zone
	return storage, nil
}

func (self *SRegion) GetStorages() ([]SStorage, error) {
	storages := []SStorage{}
	resources, err := self.GetClusterResources("storage")
	if err != nil {
		return nil, err
	}

	err = jsonutils.Update(&storages, resources)
	if err != nil {
		return nil, errors.Wrapf(err, "jsonutils.Update")
	}

	storageMap := map[string]bool{}
	ret := []SStorage{}
	for i := range storages {
		if storages[i].Shared == 0 {
			ret = append(ret, storages[i])
			continue
		}
		if _, ok := storageMap[storages[i].Storage]; !ok {
			ret = append(ret, storages[i])
			storageMap[storages[i].Storage] = true
		}
	}

	return ret, nil
}

func (self *SRegion) GetStoragesByHost(node string) ([]SStorage, error) {
	storages, err := self.GetStorages()
	if err != nil {
		return nil, err
	}
	ret := []SStorage{}
	for i := range storages {
		if storages[i].Shared == 1 || storages[i].Node == node {
			ret = append(ret, storages[i])
		}
	}
	return ret, nil
}

func (self *SRegion) GetStorage(id string) (*SStorage, error) {
	storages, err := self.GetStorages()
	if err != nil {
		return nil, err
	}
	for i := range storages {
		if storages[i].GetGlobalId() == id {
			return &storages[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}
