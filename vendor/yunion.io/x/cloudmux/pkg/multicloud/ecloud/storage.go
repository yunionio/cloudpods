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

// storageTypeConstMap 将 API 返回的存储类型 code 映射为 cloudmux 统一常量值。
// 例如 API 返回 "local" 时应使用 api.STORAGE_ECLOUD_LOCAL。
var storageTypeConstMap = map[string]string{
	"local":                     api.STORAGE_ECLOUD_LOCAL,
	"capacity":                  api.STORAGE_ECLOUD_CAPEBS,
	"highPerformance":           api.STORAGE_ECLOUD_SSD,
	"performanceOptimization":   api.STORAGE_ECLOUD_SSDEBS,
	"highPerformanceyc":         api.STORAGE_ECLOUD_SSDYC,
	"performanceOptimizationyc": api.STORAGE_ECLOUD_SSDEBS_YC,
}

type SStorage struct {
	multicloud.SStorageBase
	EcloudTags
	zone *SZone

	// 对齐 /api/ebs/customer/v3/volume/volumeType/list（GetVolumeConfig）
	CinderType        string   `json:"cinderType,omitempty"`
	BackupType        string   `json:"backupType,omitempty"`
	SnapshotType      string   `json:"snapshotType,omitempty"`
	Priority          int      `json:"priority,omitempty"`
	AttachServerTypes []string `json:"attachServerTypes,omitempty"`
	CustomBack        bool     `json:"customBack,omitempty"`
	Iscsi             bool     `json:"iscsi,omitempty"`
	Region            string   `json:"region,omitempty"`
	Encryption        bool     `json:"encryption,omitempty"`
	IsEdge            bool     `json:"isEdge,omitempty"`
	IsStorage         bool     `json:"isStorage,omitempty"`
	StorageType       string   `json:"opType,omitempty"`
}

func (s *SStorage) GetId() string {
	return fmt.Sprintf("%s-%s-%s", s.zone.region.client.cpcfg.Id, s.zone.GetGlobalId(), s.StorageType)
}

func (s *SStorage) GetName() string {
	return fmt.Sprintf("%s-%s-%s", s.zone.region.client.cpcfg.Name, s.zone.GetId(), s.StorageType)
}

func (s *SStorage) GetGlobalId() string {
	return s.GetId()
}

func (s *SStorage) GetStatus() string {
	if !s.IsStorage {
		return api.STORAGE_OFFLINE
	}
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
	disks, err := s.zone.region.GetDisks()
	if err != nil {
		return nil, err
	}

	// 按storage type 过滤出disk
	filtedDisks := make([]SDisk, 0)
	for i := range disks {
		disk := disks[i]
		if disk.Type == s.StorageType && disk.Region == s.zone.ZoneCode {
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
	return s.StorageType
}

func (s *SStorage) GetMediumType() string {
	return api.DISK_TYPE_SSD
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
	return s.StorageType != api.STORAGE_ECLOUD_LOCAL
}

func (s *SStorage) CreateIDisk(conf *cloudprovider.DiskCreateConfig) (cloudprovider.ICloudDisk, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (s *SStorage) GetIDiskById(idStr string) (cloudprovider.ICloudDisk, error) {
	disk, err := s.zone.region.GetDisk(idStr)
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

func (s *SStorage) DisableSync() bool {
	return s.StorageType == api.STORAGE_ECLOUD_LOCAL
}

func (s *SRegion) getStoragecache() *SStoragecache {
	return &SStoragecache{region: s}
}
