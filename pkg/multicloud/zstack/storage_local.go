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

package zstack

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SLocalStorage struct {
	region *SRegion

	primaryStorageID          string
	HostUUID                  string `json:"hostUuid"`
	TotalCapacity             int64  `json:"totalCapacity"`
	AvailableCapacity         int64  `json:"availableCapacity"`
	TotalPhysicalCapacity     int64  `json:"totalPhysicalCapacity"`
	AvailablePhysicalCapacity int64  `json:"availablePhysicalCapacity"`
}

func (region *SRegion) GetLocalStorage(storageId string, hostId string) (*SLocalStorage, error) {
	storages, err := region.GetLocalStorages(storageId, hostId)
	if err != nil {
		return nil, err
	}
	if len(storages) == 1 {
		if storages[0].HostUUID == hostId {
			return &storages[0], nil
		}
		return nil, cloudprovider.ErrNotFound
	}
	if len(storages) == 0 || len(storageId) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return nil, cloudprovider.ErrDuplicateId
}

func (region *SRegion) GetLocalStorages(storageId string, hostId string) ([]SLocalStorage, error) {
	localStorage := []SLocalStorage{}
	params := []string{}
	if len(hostId) > 0 {
		params = append(params, "hostUuid="+hostId)
	}
	err := region.client.listAll(fmt.Sprintf("primary-storage/local-storage/%s/capacities", storageId), params, &localStorage)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(localStorage); i++ {
		localStorage[i].region = region
		localStorage[i].primaryStorageID = storageId
	}
	return localStorage, nil
}

func (region *SRegion) getILocalStorages(storageId, hostId string) ([]cloudprovider.ICloudStorage, error) {
	storages, err := region.GetLocalStorages(storageId, hostId)
	if err != nil {
		return nil, err
	}
	istorage := []cloudprovider.ICloudStorage{}
	for i := 0; i < len(storages); i++ {
		istorage = append(istorage, &storages[i])
	}
	return istorage, nil
}

func (storage *SLocalStorage) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (storage *SLocalStorage) GetId() string {
	return storage.primaryStorageID
}

func (storage *SLocalStorage) GetName() string {
	primaryStorage, err := storage.region.GetStorage(storage.primaryStorageID)
	if err != nil {
		return "Unknown"
	}
	host, err := storage.region.GetHost(storage.HostUUID)
	if err != nil {
		return "Unknown"
	}
	return fmt.Sprintf("%s/%s", primaryStorage.Name, host.Name)
}

func (storage *SLocalStorage) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", storage.primaryStorageID, storage.HostUUID)
}

func (storage *SLocalStorage) IsEmulated() bool {
	return false
}

func (storage *SLocalStorage) GetIZone() cloudprovider.ICloudZone {
	host, err := storage.region.GetHost(storage.HostUUID)
	if err != nil {
		log.Errorf("failed get host info %s error: %v", storage.HostUUID, err)
		return nil
	}
	zone, err := storage.region.GetZone(host.ZoneUUID)
	if err != nil {
		log.Errorf("failed get zone info %s error: %v", host.ZoneUUID, err)
	}
	return zone
}

func (storage *SLocalStorage) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	tags, err := storage.region.GetSysTags("", "VolumeVO", "", "localStorage::hostUuid::"+storage.HostUUID)
	if err != nil {
		return nil, err
	}
	diskIds := []string{}
	for i := 0; i < len(tags); i++ {
		diskIds = append(diskIds, tags[i].ResourceUUID)
	}
	idisks := []cloudprovider.ICloudDisk{}
	if len(diskIds) == 0 {
		return idisks, nil
	}
	disks, err := storage.region.GetDisks(storage.primaryStorageID, diskIds, "")
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(disks); i++ {
		disks[i].localStorage = storage
		disks[i].region = storage.region
		idisks = append(idisks, &disks[i])
	}
	return idisks, nil
}

func (storage *SLocalStorage) GetStorageType() string {
	return strings.ToLower(string(StorageTypeLocal))
}

func (storage *SLocalStorage) GetMediumType() string {
	return api.DISK_TYPE_ROTATE
}

func (storage *SLocalStorage) GetCapacityMB() int64 {
	return storage.TotalCapacity / 1024 / 1024
}

func (storage *SLocalStorage) GetStorageConf() jsonutils.JSONObject {
	conf := jsonutils.NewDict()
	return conf
}

func (storage *SLocalStorage) GetStatus() string {
	primaryStorage, err := storage.region.GetStorage(storage.primaryStorageID)
	if err != nil {
		return api.STORAGE_OFFLINE
	}
	return primaryStorage.GetStatus()
}

func (storage *SLocalStorage) Refresh() error {
	// do nothing
	return nil
}

func (storage *SLocalStorage) GetEnabled() bool {
	return true
}

func (storage *SLocalStorage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	cache := &SStoragecache{region: storage.region}
	host, _ := storage.region.GetHost(storage.HostUUID)
	if host != nil {
		cache.ZoneId = host.ZoneUUID
	} else {
		_storage, _ := storage.region.GetStorage(storage.primaryStorageID)
		if _storage != nil {
			cache.ZoneId = _storage.ZoneUUID
		}
	}
	return cache
}

func (storage *SLocalStorage) CreateIDisk(name string, sizeGb int, desc string) (cloudprovider.ICloudDisk, error) {
	disk, err := storage.region.CreateDisk(name, storage.primaryStorageID, storage.HostUUID, "", sizeGb, desc)
	if err != nil {
		return nil, err
	}
	disk.localStorage = storage
	return disk, nil
}

func (storage *SLocalStorage) GetIDiskById(diskId string) (cloudprovider.ICloudDisk, error) {
	disk, err := storage.region.GetDisk(diskId)
	if err != nil {
		return nil, err
	}
	if disk.PrimaryStorageUUID != storage.primaryStorageID {
		return nil, cloudprovider.ErrNotFound
	}
	disk.localStorage = storage
	disk.region = storage.region
	return disk, nil
}

func (storage *SLocalStorage) GetMountPoint() string {
	return ""
}

func (storage *SLocalStorage) IsSysDiskStore() bool {
	return true
}
