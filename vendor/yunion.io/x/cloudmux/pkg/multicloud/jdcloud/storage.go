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

package jdcloud

import (
	"fmt"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

var storageTypes = []string{
	api.STORAGE_JDCLOUD_STD,
	api.STORAGE_JDCLOUD_GP1,
	api.STORAGE_JDCLOUD_IO1,
	api.STORAGE_JDCLOUD_SSD,
	api.STORAGE_JDCLOUD_PHD,
}

type SStorage struct {
	multicloud.SStorageBase
	JdcloudTags
	zone        *SZone
	storageType string
}

func (s *SStorage) GetId() string {
	return fmt.Sprintf("%s-%s-%s", s.zone.region.cpcfg.Id, s.zone.GetGlobalId(), s.storageType)
}

func (s *SStorage) GetName() string {
	return fmt.Sprintf("%s-%s-%s", s.zone.region.cpcfg.Name, s.zone.GetName(), s.storageType)
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

var ss cloudprovider.ICloudStorage = &SStorage{}

func (s *SStorage) GetIZone() cloudprovider.ICloudZone {
	return s.zone
}

func (s *SStorage) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks := make([]SDisk, 0)
	n := 1
	for {
		parts, total, err := s.zone.region.GetDisks("", s.zone.GetId(), s.storageType, []string{}, n, 100)
		if err != nil {
			return nil, err
		}
		disks = append(disks, parts...)
		if len(disks) >= total {
			break
		}
		n++
	}
	idisk := make([]cloudprovider.ICloudDisk, len(disks))
	for i := range disks {
		disks[i].storage = s
		idisk[i] = &disks[i]
	}
	return idisk, nil
}

func (s *SStorage) GetStorageType() string {
	return s.storageType
}

func (s *SStorage) GetMediumType() string {
	if s.storageType == api.STORAGE_JDCLOUD_STD {
		return api.DISK_TYPE_ROTATE
	} else {
		return api.DISK_TYPE_SSD
	}
}

func (s *SStorage) GetCapacityMB() int64 {
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
	disk, err := s.zone.region.GetDiskById(idStr)
	if err != nil {
		return nil, err
	}
	disk.storage = s
	return disk, nil
}

func (s *SStorage) GetMountPoint() string {
	return ""
}

func (s *SStorage) IsSysDiskStore() bool {
	return true
}
func (s *SStorage) GetCapacityUsedMB() int64 {
	return 0
}

func (s *SStorage) DisableSync() bool {
	return false
}
