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

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SZone struct {
	region *SRegion
	host   *SHost

	iwires    []cloudprovider.ICloudWire
	istorages []cloudprovider.ICloudStorage
	/* 支持的磁盘种类集合 */
	storageTypes []string

	RegionID string `json:"regionId"`
	ZoneID   string `json:"zoneId"`
	ZoneName string `json:"zoneName"`
	ZoneType string `json:"zoneType"`
}

func getZoneGlobalId(region *SRegion, zoneId string) string {
	return fmt.Sprintf("%s/%s", region.GetGlobalId(), zoneId)
}

func (self *SZone) addWire(wire *SWire) {
	if self.iwires == nil {
		self.iwires = make([]cloudprovider.ICloudWire, 0)
	}
	self.iwires = append(self.iwires, wire)
}

func (self *SZone) GetId() string {
	return self.ZoneID
}

func (self *SZone) GetName() string {
	return self.ZoneName
}

func (self *SZone) GetGlobalId() string {
	return getZoneGlobalId(self.region, self.GetId())
}

func (self *SZone) GetStatus() string {
	return "enable"
}

func (self *SZone) Refresh() error {
	return nil
}

func (self *SZone) IsEmulated() bool {
	return false
}

func (self *SZone) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SZone) GetIRegion() cloudprovider.ICloudRegion {
	return self.region
}

func (self *SZone) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	return []cloudprovider.ICloudHost{self.getHost()}, nil
}

func (self *SZone) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	host := self.getHost()
	if host.GetGlobalId() == id {
		return host, nil
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SZone) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	if self.istorages == nil {
		self.fetchStorages()
	}
	return self.istorages, nil
}

func (self *SZone) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	if self.istorages == nil {
		self.fetchStorages()
	}
	for i := 0; i < len(self.istorages); i += 1 {
		if self.istorages[i].GetGlobalId() == id {
			return self.istorages[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SZone) getHost() *SHost {
	if self.host == nil {
		self.host = &SHost{zone: self}
	}
	return self.host
}

func (self *SZone) fetchStorages() error {
	self.getStorageType()
	self.istorages = make([]cloudprovider.ICloudStorage, len(self.storageTypes))

	for i, sc := range self.storageTypes {
		storage := SStorage{zone: self, storageType: sc}
		self.istorages[i] = &storage
	}
	return nil
}

func (self *SZone) getStorageType() {
	if len(self.storageTypes) == 0 {
		self.storageTypes = StorageTypes
	}
}

func (self *SZone) GetIWires() ([]cloudprovider.ICloudWire, error) {
	return self.iwires, nil
}
