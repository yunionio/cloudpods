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

package google

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SStorage struct {
	zone *SZone
	SResourceBase
	multicloud.GoogleTags

	CreationTimestamp time.Time
	Description       string
	ValidDiskSize     string
	Zone              string
	DefaultDiskSizeGb string
	Kind              string
}

func (region *SRegion) GetStorages(zone string, maxResults int, pageToken string) ([]SStorage, error) {
	storages := []SStorage{}
	if len(zone) == 0 {
		return nil, fmt.Errorf("zone params can not be empty")
	}
	resource := fmt.Sprintf("zones/%s/diskTypes", zone)
	params := map[string]string{}
	return storages, region.List(resource, params, maxResults, pageToken, &storages)
}

func (self *SStorage) GetGlobalId() string {
	return strings.TrimPrefix(self.SelfLink, fmt.Sprintf("%s/%s/", GOOGLE_COMPUTE_DOMAIN, GOOGLE_API_VERSION))
}

func (region *SRegion) GetStorage(id string) (*SStorage, error) {
	storage := &SStorage{}
	return storage, region.GetBySelfId(id, storage)
}

func (storage *SStorage) GetName() string {
	return storage.Description
}

func (storage *SStorage) GetStatus() string {
	return api.STORAGE_ONLINE
}

func (storage *SStorage) IsEmulated() bool {
	return true
}

func (self *SStorage) GetCreatedAt() time.Time {
	return time.Time{}
}

func (storage *SStorage) Refresh() error {
	_storage, err := storage.zone.region.GetStorage(storage.SelfLink)
	if err != nil {
		return err
	}
	return jsonutils.Update(storage, _storage)
}

func (storage *SStorage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return storage.zone.region.getStoragecache()
}

func (storage *SStorage) GetIZone() cloudprovider.ICloudZone {
	return storage.zone
}

func (storage *SStorage) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks, err := storage.zone.region.GetDisks(storage.zone.Name, storage.Name, 0, "")
	if err != nil {
		return nil, err
	}
	idisks := []cloudprovider.ICloudDisk{}
	for i := range disks {
		disks[i].storage = storage
		idisks = append(idisks, &disks[i])
	}
	return idisks, nil
}

func (storage *SStorage) GetStorageType() string {
	return storage.Name
}

func (storage *SStorage) GetMediumType() string {
	return api.DISK_TYPE_SSD
}

func (storage *SStorage) GetCapacityMB() int64 {
	return 0
}

func (self *SStorage) GetCapacityUsedMB() int64 {
	return 0
}

func (storage *SStorage) GetStorageConf() jsonutils.JSONObject {
	return jsonutils.Marshal(map[string]string{
		"ValidDiskSize":     storage.ValidDiskSize,
		"DefaultDiskSizeGb": storage.DefaultDiskSizeGb,
	})
}

func (storage *SStorage) GetEnabled() bool {
	return true
}

func (storage *SStorage) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	disk, err := storage.zone.region.GetDisk(id)
	if err != nil {
		return nil, err
	}
	disk.storage = storage
	return disk, nil
}

func (storage *SStorage) CreateIDisk(conf *cloudprovider.DiskCreateConfig) (cloudprovider.ICloudDisk, error) {
	disk, err := storage.zone.region.CreateDisk(conf.Name, conf.SizeGb, storage.zone.Name, storage.Name, "", conf.Desc)
	if err != nil {
		return nil, err
	}
	disk.storage = storage
	return disk, nil
}

func (storage *SStorage) GetMountPoint() string {
	return ""
}

func (storage *SStorage) IsSysDiskStore() bool {
	return storage.Name != api.STORAGE_GOOGLE_LOCAL_SSD
}

func (storage *SStorage) DisableSync() bool {
	return false
}
