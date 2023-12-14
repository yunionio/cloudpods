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

package aliyun

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SStorage struct {
	multicloud.SStorageBase
	AliyunTags
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

func (self *SStorage) IsEmulated() bool {
	return true
}

func (self *SStorage) GetIZone() cloudprovider.ICloudZone {
	return self.zone
}

func (self *SStorage) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks := make([]SDisk, 0)
	storageType := self.storageType
	if self.storageType == api.STORAGE_CLOUD_ESSD_PL2 || self.storageType == api.STORAGE_CLOUD_ESSD_PL3 || self.storageType == api.STORAGE_CLOUD_ESSD_PL0 {
		storageType = api.STORAGE_CLOUD_ESSD
	}
	parts, err := self.zone.region.GetDisks("", self.zone.GetId(), storageType, nil, "")
	if err != nil {
		return nil, errors.Wrapf(err, "GetDisks")
	}
	performanceLevel := ""
	switch self.storageType {
	case api.STORAGE_CLOUD_ESSD_PL0:
		performanceLevel = "PL0"
	case api.STORAGE_CLOUD_ESSD:
		performanceLevel = "PL1"
	case api.STORAGE_CLOUD_ESSD_PL2:
		performanceLevel = "PL2"
	case api.STORAGE_CLOUD_ESSD_PL3:
		performanceLevel = "PL3"
	}
	for _, disk := range parts {
		if disk.PerformanceLevel == performanceLevel {
			disks = append(disks, disk)
		}
	}

	idisks := make([]cloudprovider.ICloudDisk, len(disks))
	for i := 0; i < len(disks); i += 1 {
		disks[i].storage = self
		idisks[i] = &disks[i]
	}
	return idisks, nil
}

func (self *SStorage) GetStorageType() string {
	//return models.STORAGE_PUBLIC_CLOUD
	return self.storageType
}

func (self *SStorage) GetMediumType() string {
	if strings.Contains(self.storageType, "_ssd") {
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

func (self *SStorage) GetStatus() string {
	return api.STORAGE_ONLINE
}

func (self *SStorage) Refresh() error {
	// do nothing
	return nil
}

func (self *SStorage) GetEnabled() bool {
	if utils.IsInStringArray(self.storageType, LOCAL_STORAGES) {
		return false
	}
	return true
}

func (self *SStorage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return self.zone.region.getStoragecache()
}

func (self *SStorage) CreateIDisk(conf *cloudprovider.DiskCreateConfig) (cloudprovider.ICloudDisk, error) {
	diskId, err := self.zone.region.CreateDisk(self.zone.ZoneId, self.storageType, conf.Name, conf.SizeGb, conf.Desc, conf.ProjectId)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateDisk")
	}
	disk, err := self.zone.region.getDisk(diskId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetDisk")
	}
	disk.storage = self
	return disk, nil
}

func (self *SStorage) GetIDiskById(idStr string) (cloudprovider.ICloudDisk, error) {
	disk, err := self.zone.region.getDisk(idStr)
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
	if utils.IsInStringArray(self.storageType, self.zone.getSysDiskCategories()) {
		return true
	}
	return false
}
