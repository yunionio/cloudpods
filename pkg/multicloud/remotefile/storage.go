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

package remotefile

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SStorage struct {
	SResourceBase

	zone           *SZone
	ZoneId         string
	StorageType    string
	MediumType     string
	CapacityMb     int64
	CapacityUsedMb int64
	Enabled        bool
	SkipSync       bool
}

func (self *SStorage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return nil
}

func (self *SStorage) GetIZone() cloudprovider.ICloudZone {
	return self.zone
}

func (self *SStorage) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks, err := self.zone.region.client.GetDisks()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudDisk{}
	for i := range disks {
		if disks[i].StorageId != self.GetId() {
			continue
		}
		disks[i].storage = self
		ret = append(ret, &disks[i])
	}
	return ret, nil
}

func (self *SStorage) GetStorageType() string {
	return self.StorageType
}

func (self *SStorage) GetMediumType() string {
	return self.MediumType
}

func (self *SStorage) GetCapacityMB() int64 {
	return self.CapacityMb
}

func (self *SStorage) GetCapacityUsedMB() int64 {
	return self.CapacityUsedMb
}

func (self *SStorage) GetStorageConf() jsonutils.JSONObject {
	return jsonutils.NewDict()
}

func (self *SStorage) GetEnabled() bool {
	return self.Enabled
}

func (self *SStorage) CreateIDisk(conf *cloudprovider.DiskCreateConfig) (cloudprovider.ICloudDisk, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SStorage) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	disks, err := self.GetIDisks()
	if err != nil {
		return nil, err
	}
	for i := range disks {
		if disks[i].GetGlobalId() == id {
			return disks[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SStorage) GetMountPoint() string {
	return ""
}

func (self *SStorage) IsSysDiskStore() bool {
	return true
}

func (self *SStorage) DisableSync() bool {
	return self.SkipSync
}
