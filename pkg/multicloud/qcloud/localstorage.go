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

package qcloud

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SLocalStorage struct {
	zone        *SZone
	storageType string
	available   bool
}

func (self *SLocalStorage) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SLocalStorage) GetId() string {
	return fmt.Sprintf("%s-%s-%s", self.zone.region.client.providerId, self.zone.GetId(), self.storageType)
}

func (self *SLocalStorage) GetName() string {
	return fmt.Sprintf("%s-%s-%s", self.zone.region.client.providerName, self.zone.GetId(), self.storageType)
}

func (self *SLocalStorage) GetGlobalId() string {
	return fmt.Sprintf("%s-%s-%s", self.zone.region.client.providerId, self.zone.GetGlobalId(), self.storageType)
}

func (self *SLocalStorage) IsEmulated() bool {
	return true
}

func (self *SLocalStorage) GetIZone() cloudprovider.ICloudZone {
	return self.zone
}

func (self *SLocalStorage) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks := []SLocalDisk{}
	idisks := make([]cloudprovider.ICloudDisk, len(disks))
	for i := 0; i < len(disks); i++ {
		disks[i].storage = self
		idisks[i] = &disks[i]
	}
	return idisks, nil
}

func (self *SLocalStorage) GetStorageType() string {
	return strings.ToLower(self.storageType)
}

func (self *SLocalStorage) GetMediumType() string {
	if strings.HasSuffix(self.storageType, "_BASIC") {
		return api.DISK_TYPE_ROTATE
	}
	return api.DISK_TYPE_SSD
}

func (self *SLocalStorage) GetCapacityMB() int64 {
	return 0 // unlimited
}

func (self *SLocalStorage) GetStorageConf() jsonutils.JSONObject {
	conf := jsonutils.NewDict()
	return conf
}

func (self *SLocalStorage) GetStatus() string {
	if !self.available {
		return api.STORAGE_OFFLINE
	}
	return api.STORAGE_ONLINE
}

func (self *SLocalStorage) Refresh() error {
	// do nothing
	return nil
}

func (self *SLocalStorage) GetEnabled() bool {
	return self.available == true
}

func (self *SLocalStorage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return self.zone.region.getStoragecache()
}

func (self *SLocalStorage) CreateIDisk(name string, sizeGb int, desc string) (cloudprovider.ICloudDisk, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SLocalStorage) GetIDiskById(idStr string) (cloudprovider.ICloudDisk, error) {
	return &SLocalDisk{storage: self, DiskId: idStr}, nil
}

func (self *SLocalStorage) GetMountPoint() string {
	return ""
}

func (self *SLocalStorage) IsSysDiskStore() bool {
	return true
}
