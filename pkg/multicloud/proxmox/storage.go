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
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SStorage struct {
	multicloud.SStorageBase
	multicloud.ProxmoxTags

	zone *SZone

	Id   string
	Node string

	Total        int64   `json:"total"`
	Storage      string  `json:"storage"`
	Shared       int     `json:"shared"`
	Used         int64   `json:"used"`
	Content      string  `json:"content"`
	Active       int     `json:"active"`
	UsedFraction float64 `json:"used_fraction"`
	Avail        int64   `json:"avail"`
	Enabled      int     `json:"enabled"`
	Type         string  `json:"type"`
}

func (self *SStorage) GetName() string {
	return self.Storage
}

func (self *SStorage) GetId() string {
	return self.Id
}

func (self *SStorage) GetGlobalId() string {
	return self.GetId()
}

func (self *SStorage) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks, err := self.zone.region.GetDisks(self.Id)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudDisk{}
	for i := range disks {
		disks[i].region = self.zone.region
		ret = append(ret, &disks[i])
	}
	return ret, nil
}

func (self *SStorage) CreateIDisk(conf *cloudprovider.DiskCreateConfig) (cloudprovider.ICloudDisk, error) {
	return nil, cloudprovider.ErrNotFound
}

func (self *SStorage) GetCapacityMB() int64 {
	return int64(self.Total / 1024 / 1024)
}

func (self *SStorage) GetCapacityUsedMB() int64 {
	return int64(self.Used / 1024 / 1024)
}

func (self *SStorage) GetEnabled() bool {
	return true
}

func (self *SStorage) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	disk, err := self.zone.region.GetDisk(id)
	if err != nil {
		return nil, err
	}

	return disk, nil
}

func (self *SStorage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return nil
}

func (self *SStorage) GetMediumType() string {
	return api.DISK_TYPE_SSD
}

func (self *SStorage) GetMountPoint() string {
	return ""
}

func (self *SStorage) GetStatus() string {
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
	return strings.ToLower(self.Type)
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
	resources, err := self.GetClusterStoragesResources()
	if err != nil {
		return nil, err
	}

	for _, res := range resources {
		storage := &SStorage{}
		status := fmt.Sprintf("%s/status", res.Path)
		err := self.get(status, url.Values{}, storage)
		storage.Id = res.Id
		storage.Node = res.Node

		if err != nil {
			return nil, err
		}

		storages = append(storages, *storage)
	}

	return storages, nil
}

func (self *SRegion) GetStoragesByHost(hostId string) ([]SStorage, error) {
	storages := []SStorage{}
	nodeName := ""
	splited := strings.Split(hostId, "/")
	nodeName = splited[1]

	res := fmt.Sprintf("/nodes/%s/storage", nodeName)
	err := self.get(res, url.Values{}, &storages)

	if err != nil {
		return nil, err
	}

	for _, storage := range storages {
		id := fmt.Sprintf("storage/%s/%s", hostId, storage.Storage)
		storage.Node = hostId
		storage.Id = id
	}

	return storages, nil
}

func (self *SRegion) GetStorage(id string) (*SStorage, error) {
	ret := &SStorage{}

	//"id": "storage/nodeNAME/strogeNAME",
	splited := strings.Split(id, "/")
	nodeName := ""
	storageName := ""

	if len(splited) == 3 {
		nodeName, storageName = splited[1], splited[2]
	}

	status := fmt.Sprintf("/nodes/%s/storage/%s/status", nodeName, storageName)
	err := self.get(status, url.Values{}, ret)
	ret.Id = id
	ret.Node = nodeName

	return ret, err
}
