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

package aws

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SStorage struct {
	multicloud.SStorageBase
	multicloud.STagBase
	zone        *SZone
	storageType string
}

func (self *SStorage) GetId() string {
	return fmt.Sprintf("%s-%s-%s", self.zone.region.client.cpcfg.Id, self.zone.GetId(), self.storageType)
}

func (self *SStorage) GetName() string {
	return fmt.Sprintf("%s-%s-%s", self.zone.region.client.cpcfg.Name, self.zone.GetId(), self.storageType)
}

func (self *SStorage) GetGlobalId() string {
	return fmt.Sprintf("%s-%s-%s", self.zone.region.client.cpcfg.Id, self.zone.GetGlobalId(), self.storageType)
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

func (self *SStorage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return self.zone.region.getStoragecache()
}

func (self *SStorage) GetIZone() cloudprovider.ICloudZone {
	return self.zone
}

func (self *SStorage) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks, err := self.zone.region.GetDisks("", self.zone.GetId(), self.storageType, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "GetDisks")
	}
	ret := []cloudprovider.ICloudDisk{}
	for i := 0; i < len(disks); i += 1 {
		disks[i].storage = self
		ret = append(ret, &disks[i])
	}
	return ret, nil
}

func (self *SStorage) GetStorageType() string {
	return self.storageType
}

func (self *SStorage) GetMediumType() string {
	if self.storageType == api.STORAGE_GP2_SSD || self.storageType == api.STORAGE_IO1_SSD || self.storageType == api.STORAGE_IO2_SSD {
		return api.DISK_TYPE_SSD
	} else {
		return api.DISK_TYPE_ROTATE
	}
}

func (self *SStorage) GetCapacityMB() int64 {
	return 0 // unlimited
}

func (self *SStorage) GetCapacityUsedMB() int64 {
	return 0
}

func (self *SStorage) GetStorageConf() jsonutils.JSONObject {
	conf := jsonutils.NewDict()
	return conf
}

func (self *SStorage) GetEnabled() bool {
	return true
}

func (self *SStorage) CreateIDisk(conf *cloudprovider.DiskCreateConfig) (cloudprovider.ICloudDisk, error) {
	disk, err := self.zone.region.CreateDisk(self.zone.ZoneId, self.storageType, conf.Name, conf.SizeGb, "", conf.Desc)
	if err != nil {
		return nil, errors.Wrap(err, "CreateDisk")
	}
	disk.storage = self
	return disk, nil
}

func (self *SStorage) GetIDiskById(idStr string) (cloudprovider.ICloudDisk, error) {
	disk, err := self.zone.region.GetDisk(idStr)
	if err != nil {
		return nil, errors.Wrap(err, "GetDisk")
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
