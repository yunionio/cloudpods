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

package proxmox

import (
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SStorage struct {
	multicloud.SStorageBase
	multicloud.InCloudSphereTags

	zone *SZone

	Total        int64   `json:"total"`
	Storage      string  `json:"storage"`
	Shared       int     `json:"shared"`
	Used         int64   `json:"used"`
	Content      string  `json:"content"`
	Active       int     `json:"active"`
	UsedFraction float64 `json:"used_fraction"`
	Avail        int64   `json:"avail"`
	Enabled      int     `json:"enabled"`
	Type         string  `json:"type"`
}

func (self *SStorage) GetName() string {
	return self.Storage
}

func (self *SStorage) GetId() string {
	return self.Storage
}

func (self *SStorage) GetGlobalId() string {
	return self.GetId()
}

func (self *SStorage) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks, err := self.zone.region.GetDisks(self.Id)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudDisk{}
	for i := range disks {
		disks[i].region = self.zone.region
		ret = append(ret, &disks[i])
	}
	return ret, nil
}

func (self *SStorage) CreateIDisk(conf *cloudprovider.DiskCreateConfig) (cloudprovider.ICloudDisk, error) {
	disk, err := self.zone.region.CreateDisk(conf.Name, self.Id, conf.SizeGb)
	if err != nil {
		return nil, err
	}
	return disk, nil
}

func (self *SStorage) GetCapacityMB() int64 {
	return int64(self.Total / 1024 / 1024)
}

func (self *SStorage) GetCapacityUsedMB() int64 {
	return int64(self.Used / 1024 / 1024)
}

func (self *SStorage) GetEnabled() bool {
	return true
}

func (self *SStorage) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	disk, err := self.zone.region.GetDisk(id)
	if err != nil {
		return nil, err
	}
	if disk.DataStoreId != self.Id {
		return nil, cloudprovider.ErrNotFound
	}
	return disk, nil
}

func (self *SStorage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	cache := &SStoragecache{zone: self.zone}
	return cache
}
