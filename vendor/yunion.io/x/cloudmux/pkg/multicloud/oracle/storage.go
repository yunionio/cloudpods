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

package oracle

import (
	"fmt"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
)

type SStorage struct {
	multicloud.SStorageBase
	boot bool

	zone *SZone
}

func (self *SStorage) GetId() string {
	if self.boot {
		return fmt.Sprintf("%s/%s-system", self.zone.GetGlobalId(), self.zone.region.client.cpcfg.Id)
	}
	return fmt.Sprintf("%s/%s", self.zone.GetGlobalId(), self.zone.region.client.cpcfg.Id)
}

func (self *SStorage) GetGlobalId() string {
	return self.GetId()
}

func (self *SStorage) GetName() string {
	res := fmt.Sprintf("%s-%s-%s-%s", self.zone.region.client.cpcfg.Name, self.zone.region.RegionName, self.zone.Name, api.STORAGE_FULL)
	if self.boot {
		res += "-system"
	}
	return res
}

func (self *SStorage) IsEmulated() bool {
	return true
}

func (self *SStorage) GetStatus() string {
	return api.STORAGE_ONLINE
}

func (self *SStorage) CreateIDisk(opts *cloudprovider.DiskCreateConfig) (cloudprovider.ICloudDisk, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SStorage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return self.zone.region.GetStoragecache()
}

func (self *SStorage) GetIZone() cloudprovider.ICloudZone {
	return self.zone
}

func (self *SStorage) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	ret := []cloudprovider.ICloudDisk{}
	if self.boot {
		disks, err := self.zone.region.GetBootDisks(self.zone.Name)
		if err != nil {
			return nil, err
		}
		for i := range disks {
			disks[i].storage = self
			ret = append(ret, &disks[i])
		}
		return ret, nil
	}
	disks, err := self.zone.region.GetDisks(self.zone.Name)
	if err != nil {
		return nil, err
	}
	for i := range disks {
		disks[i].storage = self
		ret = append(ret, &disks[i])
	}
	return ret, nil
}

func (self *SStorage) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	if self.boot {
		disk, err := self.zone.region.GetBootDisk(id)
		if err != nil {
			return nil, err
		}
		disk.storage = self
		return disk, nil
	}
	disk, err := self.zone.region.GetDisk(id)
	if err != nil {
		return nil, err
	}
	disk.storage = self
	return disk, nil
}

func (self *SStorage) GetStorageType() string {
	if self.boot {
		return api.STORAGE_SYSTEM_FULL
	}
	return api.STORAGE_FULL
}

func (self *SStorage) GetMediumType() string {
	return api.DISK_TYPE_SSD
}

func (self *SStorage) GetEnabled() bool {
	return true
}

func (self *SStorage) IsSysDiskStore() bool {
	return self.boot
}

func (self *SStorage) GetCapacityMB() int64 {
	return 0
}

func (self *SStorage) GetCapacityUsedMB() int64 {
	return 0
}

func (self *SStorage) GetMountPoint() string {
	return ""
}

func (self *SStorage) GetStorageConf() jsonutils.JSONObject {
	return jsonutils.NewDict()
}
