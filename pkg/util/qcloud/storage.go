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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SStorage struct {
	zone        *SZone
	storageType string
	available   bool
}

func (self *SStorage) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SStorage) GetId() string {
	return fmt.Sprintf("%s-%s-%s", self.zone.region.client.providerId, self.zone.GetId(), strings.ToLower(self.storageType))
}

func (self *SStorage) GetName() string {
	return fmt.Sprintf("%s-%s-%s", self.zone.region.client.providerName, self.zone.GetId(), strings.ToLower(self.storageType))
}

func (self *SStorage) GetGlobalId() string {
	return fmt.Sprintf("%s-%s-%s", self.zone.region.client.providerId, self.zone.GetGlobalId(), strings.ToLower(self.storageType))
}

func (self *SStorage) IsEmulated() bool {
	return true
}

func (self *SStorage) GetIZone() cloudprovider.ICloudZone {
	return self.zone
}

func (self *SStorage) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks := make([]SDisk, 0)
	for {
		parts, total, err := self.zone.region.GetDisks("", self.zone.GetId(), self.storageType, nil, len(disks), 50)
		if err != nil {
			log.Errorf("GetDisks fail %s", err)
			return nil, err
		}
		disks = append(disks, parts...)
		if len(disks) >= total {
			break
		}
	}
	idisks := make([]cloudprovider.ICloudDisk, len(disks))
	for i := 0; i < len(disks); i++ {
		disks[i].storage = self
		idisks[i] = &disks[i]
	}
	return idisks, nil
}

func (self *SStorage) GetStorageType() string {
	return strings.ToLower(self.storageType)
}

func (self *SStorage) GetMediumType() string {
	if strings.HasSuffix(self.storageType, "_BASIC") {
		return api.DISK_TYPE_ROTATE
	}
	return api.DISK_TYPE_SSD
}

func (self *SStorage) GetCapacityMB() int {
	return 0 // unlimited
}

func (self *SStorage) GetStorageConf() jsonutils.JSONObject {
	conf := jsonutils.NewDict()
	return conf
}

func (self *SStorage) GetStatus() string {
	if !self.available {
		return api.STORAGE_OFFLINE
	}
	return api.STORAGE_ONLINE
}

func (self *SStorage) Refresh() error {
	// do nothing
	return nil
}

func (self *SStorage) GetEnabled() bool {
	return self.available == true
}

func (self *SStorage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return self.zone.region.getStoragecache()
}

func (self *SStorage) CreateIDisk(name string, sizeGb int, desc string) (cloudprovider.ICloudDisk, error) {
	diskId, err := self.zone.region.CreateDisk(self.zone.Zone, self.storageType, name, sizeGb, desc)
	if err != nil {
		log.Errorf("createDisk fail %s", err)
		return nil, err
	}
	//腾讯刚创建完成的磁盘，需要稍微等待才能查询
	for i := 0; i < 3; i++ {
		disk, err := self.zone.region.GetDisk(diskId)
		if err == nil {
			disk.storage = self
			return disk, nil
		}
		time.Sleep(time.Second * 3)
	}
	log.Errorf("getDisk fail %s id %s", err, diskId)
	return nil, cloudprovider.ErrNotFound
}

func (self *SStorage) GetIDiskById(idStr string) (cloudprovider.ICloudDisk, error) {
	disk, err := self.zone.region.GetDisk(idStr)
	if err != nil {
		return nil, err
	}
	disk.storage = self
	return disk, nil
}

func (self *SStorage) GetMountPoint() string {
	return ""
}

func (self *SStorage) IsSysDiskStore() bool {
	return true
}
