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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SStorage struct {
	multicloud.SStorageBase
	CtyunTags
	zone        *SZone
	Name        string
	storageType string
}

func (self *SStorage) GetId() string {
	return fmt.Sprintf("%s-%s-%s", self.zone.region.client.cpcfg.Id, self.zone.GetGlobalId(), self.storageType)
}

func (self *SStorage) GetName() string {
	return fmt.Sprintf("%s-%s-%s", self.zone.region.client.cpcfg.Name, self.zone.GetId(), self.Name)
}

func (self *SStorage) GetGlobalId() string {
	return self.GetId()
}

func (self *SStorage) GetStatus() string {
	return api.STORAGE_ONLINE
}

func (self *SStorage) IsEmulated() bool {
	return true
}

func (self *SStorage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return self.zone.region.getStoragecache()
}

func (self *SStorage) GetIZone() cloudprovider.ICloudZone {
	return self.zone
}

func (self *SStorage) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks, err := self.zone.region.GetDisks()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudDisk{}
	for i := range disks {
		if disks[i].DiskType == self.storageType && (len(disks[i].AzName) == 0 || disks[i].AzName == self.zone.AzDisplayName) {
			disks[i].storage = self
			ret = append(ret, &disks[i])
		}
	}
	return ret, nil
}

func (self *SStorage) GetStorageType() string {
	return self.storageType
}

func (self *SStorage) GetMediumType() string {
	if strings.Contains(self.storageType, "SSD") {
		return api.DISK_TYPE_SSD
	}
	return api.DISK_TYPE_ROTATE
}

func (self *SStorage) GetCapacityMB() int64 {
	return 0
}

func (self *SStorage) GetCapacityUsedMB() int64 {
	return 0
}

func (self *SStorage) GetStorageConf() jsonutils.JSONObject {
	return jsonutils.NewDict()
}

func (self *SStorage) GetEnabled() bool {
	return true
}

func (self *SStorage) CreateIDisk(conf *cloudprovider.DiskCreateConfig) (cloudprovider.ICloudDisk, error) {
	disk, err := self.zone.region.CreateDisk(self.zone.GetId(), conf.Name, self.GetStorageType(), conf.SizeGb)
	if err != nil {
		return nil, errors.Wrap(err, "CreateDisk")
	}
	disk.storage = self
	return disk, nil
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

func (self *SRegion) getStoragecache() *SStoragecache {
	return &SStoragecache{region: self}
}
