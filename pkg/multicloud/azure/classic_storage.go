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
	"strings"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type ClassicStorageProperties struct {
	ProvisioningState       string
	Status                  string
	Endpoints               []string
	AccountType             string `json:"accountType"`
	GeoPrimaryRegion        string
	StatusOfPrimaryRegion   string
	GeoSecondaryRegion      string
	StatusOfSecondaryRegion string
	//CreationTime            time.Time
}

type SClassicStorage struct {
	zone *SZone

	Properties ClassicStorageProperties
	Name       string
	ID         string
	Type       string
	Location   string
}

func (self *SClassicStorage) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SClassicStorage) GetId() string {
	return self.ID
}

func (self *SClassicStorage) GetName() string {
	return self.Name
}

func (self *SClassicStorage) GetGlobalId() string {
	return strings.ToLower(self.ID)
}

func (self *SClassicStorage) IsEmulated() bool {
	return false
}

func (self *SClassicStorage) GetIZone() cloudprovider.ICloudZone {
	return self.zone
}

func (self *SClassicStorage) GetEnabled() bool {
	return false
}

func (self *SClassicStorage) GetCapacityMB() int64 {
	return 0 // unlimited
}

func (self *SClassicStorage) CreateIDisk(name string, sizeGb int, desc string) (cloudprovider.ICloudDisk, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SClassicStorage) GetIDiskById(diskId string) (cloudprovider.ICloudDisk, error) {
	disks, err := self.GetIDisks()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(disks); i++ {
		if disks[i].GetId() == diskId {
			return disks[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SClassicStorage) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	storageaccount, err := self.zone.region.GetStorageAccountDetail(self.ID)
	disks, _, err := self.zone.region.GetStorageAccountDisksWithSnapshots(storageaccount)
	if err != nil {
		return nil, err
	}
	idisks := make([]cloudprovider.ICloudDisk, len(disks))
	for i := 0; i < len(disks); i++ {
		disks[i].storage = self
		idisks[i] = &disks[i]
	}
	return idisks, nil
}

func (self *SClassicStorage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return self.zone.region.getStoragecache()
}

func (self *SClassicStorage) GetMediumType() string {
	if strings.Contains(self.Properties.AccountType, "Premium") {
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
	return strings.ToLower(self.Properties.AccountType)
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
