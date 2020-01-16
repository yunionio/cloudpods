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

package ctyun

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

var StorageTypes = []string{
	api.STORAGE_CTYUN_SAS,
	api.STORAGE_CTYUN_SATA,
	api.STORAGE_CTYUN_SSD,
}

type SStorage struct {
	zone        *SZone
	storageType string
}

func (self *SStorage) GetId() string {
	return fmt.Sprintf("%s-%s-%s", self.zone.region.client.providerId, self.zone.GetGlobalId(), self.storageType)
}

func (self *SStorage) GetName() string {
	return fmt.Sprintf("%s-%s-%s", self.zone.region.client.providerName, self.zone.GetId(), self.storageType)
}

func (self *SStorage) GetGlobalId() string {
	return self.GetId()
}

func (self *SStorage) GetStatus() string {
	return api.STORAGE_ONLINE
}

func (self *SStorage) Refresh() error {
	return nil
}

func (self *SStorage) IsEmulated() bool {
	return true
}

func (self *SStorage) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SStorage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return self.zone.region.getStoragecache()
}

func (self *SStorage) GetIZone() cloudprovider.ICloudZone {
	return self.zone
}

// http://ctyun-api-url/apiproxy/v3/ondemand/queryVolumes
//  http://ctyun-api-url/apiproxy/v3/queryDataDisks
func (self *SStorage) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks, err := self.zone.region.GetDisks()
	if err != nil {
		return nil, err
	}

	// 按storage type 过滤出disk
	filtedDisks := make([]SDisk, 0)
	for i := range disks {
		disk := disks[i]
		if disk.VolumeType == self.storageType && disk.AvailabilityZone == self.zone.GetId() {
			filtedDisks = append(filtedDisks, disk)
		}
	}

	idisks := make([]cloudprovider.ICloudDisk, len(filtedDisks))
	for i := 0; i < len(filtedDisks); i += 1 {
		filtedDisks[i].storage = self
		idisks[i] = &filtedDisks[i]
	}
	return idisks, nil
}

func (self *SStorage) GetStorageType() string {
	return self.storageType
}

func (self *SStorage) GetMediumType() string {
	if self.storageType == api.STORAGE_CTYUN_SSD {
		return api.DISK_TYPE_SSD
	} else {
		return api.DISK_TYPE_ROTATE
	}
}

func (self *SStorage) GetCapacityMB() int64 {
	return 0
}

func (self *SStorage) GetStorageConf() jsonutils.JSONObject {
	return jsonutils.NewDict()
}

func (self *SStorage) GetEnabled() bool {
	return true
}

func (self *SStorage) CreateIDisk(name string, sizeGb int, desc string) (cloudprovider.ICloudDisk, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SStorage) GetIDiskById(idStr string) (cloudprovider.ICloudDisk, error) {
	if len(idStr) == 0 {
		log.Debugf("GetIDiskById disk id should not be empty")
		return nil, cloudprovider.ErrNotFound
	}

	if disk, err := self.zone.region.GetDisk(idStr); err != nil {
		return nil, err
	} else {
		disk.storage = self
		return disk, nil
	}
}

func (self *SStorage) GetMountPoint() string {
	return ""
}

func (self *SStorage) IsSysDiskStore() bool {
	return true
}

func (self *SRegion) getStoragecache() *SStoragecache {
	if self.storageCache == nil {
		self.storageCache = &SStoragecache{region: self}
	}
	return self.storageCache
}
