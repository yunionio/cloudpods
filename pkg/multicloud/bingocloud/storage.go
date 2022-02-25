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
package bingocloud

import (
	"github.com/pkg/errors"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SStorage struct {
	zone          *SZone
	host          *SHost
	SStoragecache *SStoragecache
	region        *SRegion

	Disabled     bool   `json:"disabled"`
	DrCloudId    string `json:"drCloudId"`
	ParameterSet struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"parameterSet"`
	StorageId    string `json:"storageId"`
	ClusterId    string `json:"clusterId"`
	UsedBy       string `json:"usedBy"`
	SpaceMax     string `json:"spaceMax"`
	IsDRStorage  string `json:"isDRStorage"`
	ScheduleTags string `json:"scheduleTags"`
	StorageName  string `json:"storageName"`
	Location     string `json:"location"`
	SpaceUsed    string `json:"spaceUsed"`
	StorageType  string `json:"storageType"`
	FileFormat   string `json:"fileFormat"`
	ResUsage     string `json:"resUsage"`
}

func (self *SRegion) GetStorages() ([]SStorage, error) {
	resp, err := self.invoke("DescribeStorages", nil)
	if err != nil {
		return nil, err
	}
	log.Errorf("resp=:%s", resp)
	result := struct {
		StorageSet struct {
			Item []SStorage
		}
	}{}
	err = resp.Unmarshal(&result)
	if err != nil {
		return nil, err
	}
	return result.StorageSet.Item, nil
}

func (self *SRegion) GetStorage(id string) (*SStorage, error) {
	storage := &SStorage{}
	return storage, cloudprovider.ErrNotImplemented
}

func (self *SStorage) GetIZone() cloudprovider.ICloudZone {
	return self.zone
}

func (self *SStorage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return nil
}

func (self *SStorage) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks, err := self.zone.region.GetDisks(self.GetGlobalId(), "")
	if err != nil {
		return nil, errors.Wrapf(err, "GetDisks")
	}
	ret := []cloudprovider.ICloudDisk{}
	for i := range disks {
		disks[i].storage = self
		ret = append(ret, &disks[i])
	}
	return ret, nil
}

func (self *SStorage) GetStorageType() string {
	return api.STORAGE_LOCAL
}

func (self *SStorage) GetMediumType() string {
	return api.DISK_TYPE_SSD
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
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SStorage) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	disk, err := self.zone.region.GetDisk(id)
	if err != nil {
		return nil, err
	}
	if disk.storage.StorageId != self.GetGlobalId() {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
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

func (self *SStorage) DisableSync() bool {
	return false
}

//
func (self *SStorage) GetId() string {
	return self.StorageId
}

func (self *SStorage) GetName() string {
	return self.StorageName
}

func (self *SStorage) GetGlobalId() string {
	return self.StorageId
}

func (self *SStorage) GetStatus() string {
	return api.STORAGE_ONLINE
}

func (self *SStorage) Refresh() error {
	storage, err := self.zone.region.GetStorage(self.GetGlobalId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, storage)
}

func (self *SStorage) IsEmulated() bool {
	return false
}

func (self *SStorage) GetSysTags() map[string]string {
	return nil
}

func (self *SStorage) GetTags() (map[string]string, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SStorage) SetTags(tags map[string]string, replace bool) error {
	return cloudprovider.ErrNotImplemented
}
