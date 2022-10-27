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

package ecloud

import (
	"fmt"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

var storageTypes = []string{
	api.STORAGE_ECLOUD_CAPEBS,
	api.STORAGE_ECLOUD_EBS,
	api.STORAGE_ECLOUD_SSD,
	api.STORAGE_ECLOUD_SSDEBS,
	// special storage
	api.STORAGE_ECLOUD_SYSTEM,
}

type SStorage struct {
	multicloud.SStorageBase
	EcloudTags
	zone        *SZone
	storageType string
}

func (s *SStorage) GetId() string {
	return fmt.Sprintf("%s-%s-%s", s.zone.region.client.cpcfg.Id, s.zone.GetGlobalId(), s.storageType)
}

func (s *SStorage) GetName() string {
	return fmt.Sprintf("%s-%s-%s", s.zone.region.client.cpcfg.Name, s.zone.GetId(), s.storageType)
}

func (s *SStorage) GetGlobalId() string {
	return s.GetId()
}

func (s *SStorage) GetStatus() string {
	return api.STORAGE_ONLINE
}

func (s *SStorage) Refresh() error {
	return nil
}

func (s *SStorage) IsEmulated() bool {
	return true
}

func (s *SStorage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return s.zone.region.getStoragecache()
}

func (s *SStorage) GetIZone() cloudprovider.ICloudZone {
	return s.zone
}

func (s *SStorage) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	if s.storageType == api.STORAGE_ECLOUD_SYSTEM {
		return nil, nil
	}
	disks, err := s.zone.region.GetDisks()
	if err != nil {
		return nil, err
	}

	// 按storage type 过滤出disk
	filtedDisks := make([]SDisk, 0)
	for i := range disks {
		disk := disks[i]
		if disk.Type == s.storageType && disk.Region == s.zone.Region {
			filtedDisks = append(filtedDisks, disk)
		}
	}

	idisks := make([]cloudprovider.ICloudDisk, len(filtedDisks))
	for i := 0; i < len(filtedDisks); i += 1 {
		filtedDisks[i].storage = s
		idisks[i] = &filtedDisks[i]
	}
	return idisks, nil
}

func (s *SStorage) GetStorageType() string {
	return s.storageType
}

func (s *SStorage) GetMediumType() string {
	if s.storageType == api.STORAGE_ECLOUD_SSD {
		return api.DISK_TYPE_SSD
	} else {
		return api.DISK_TYPE_ROTATE
	}
}

func (s *SStorage) GetCapacityMB() int64 {
	return 0
}

func (s *SStorage) GetCapacityUsedMB() int64 {
	return 0
}

func (s *SStorage) GetStorageConf() jsonutils.JSONObject {
	return jsonutils.NewDict()
}

func (s *SStorage) GetEnabled() bool {
	return true
}

func (s *SStorage) CreateIDisk(conf *cloudprovider.DiskCreateConfig) (cloudprovider.ICloudDisk, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (s *SStorage) GetIDiskById(idStr string) (cloudprovider.ICloudDisk, error) {
	if len(idStr) == 0 {
		return nil, cloudprovider.ErrNotFound
	}

	if disk, err := s.zone.region.GetDisk(idStr); err != nil {
		return nil, err
	} else {
		disk.storage = s
		return disk, nil
	}
}

func (s *SStorage) GetMountPoint() string {
	return ""
}

func (s *SStorage) IsSysDiskStore() bool {
	return true
}

func (s *SStorage) DisableSync() bool {
	if s.storageType == api.STORAGE_ECLOUD_SYSTEM {
		return true
	}
	return s.SStorageBase.DisableSync()
}

func (s *SRegion) getStoragecache() *SStoragecache {
	if s.storageCache == nil {
		s.storageCache = &SStoragecache{region: s}
	}
	return s.storageCache
}
