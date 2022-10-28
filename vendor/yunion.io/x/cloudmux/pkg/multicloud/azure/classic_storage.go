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

package azure

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

const (
	STORAGE_LRS = "Standard-LRS"
	STORAGE_GRS = "Standard-GRS"
)

type SClassicStorage struct {
	multicloud.SStorageBase
	AzureTags
	region *SRegion

	AccountType string
}

func (self *SClassicStorage) GetId() string {
	zone := self.region.getZone()
	return fmt.Sprintf("%s-%s-classic", zone.GetGlobalId(), self.AccountType)
}

func (self *SClassicStorage) GetName() string {
	return self.AccountType
}

func (self *SClassicStorage) GetGlobalId() string {
	return self.GetId()
}

func (self *SClassicStorage) IsEmulated() bool {
	return true
}

func (self *SClassicStorage) GetIZone() cloudprovider.ICloudZone {
	return self.region.getZone()
}

func (self *SClassicStorage) GetEnabled() bool {
	return false
}

func (self *SClassicStorage) GetCapacityMB() int64 {
	return 0 // unlimited
}

func (self *SClassicStorage) GetCapacityUsedMB() int64 {
	return 0
}

func (self *SClassicStorage) CreateIDisk(conf *cloudprovider.DiskCreateConfig) (cloudprovider.ICloudDisk, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SClassicStorage) GetIDiskById(diskId string) (cloudprovider.ICloudDisk, error) {
	disk, err := self.region.GetClassicDisk(diskId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetDisk(%s)", diskId)
	}
	return disk, nil
}

func (self *SClassicStorage) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	return []cloudprovider.ICloudDisk{}, nil
}

func (self *SClassicStorage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return self.region.getStoragecache()
}

func (self *SClassicStorage) GetMediumType() string {
	if self.AccountType == STORAGE_LRS {
		return api.DISK_TYPE_SSD
	}
	return api.DISK_TYPE_ROTATE
}

func (self *SClassicStorage) GetStorageConf() jsonutils.JSONObject {
	conf := jsonutils.NewDict()
	return conf
}

func (self *SClassicStorage) GetStatus() string {
	return api.STORAGE_ONLINE
}

func (self *SClassicStorage) GetStorageType() string {
	return strings.ToLower(self.AccountType)
}

func (self *SClassicStorage) Refresh() error {
	// do nothing
	return nil
}

func (self *SClassicStorage) GetMountPoint() string {
	return ""
}

func (self *SClassicStorage) IsSysDiskStore() bool {
	return true
}
