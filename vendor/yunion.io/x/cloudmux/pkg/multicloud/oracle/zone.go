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
)

type SZone struct {
	multicloud.SResourceBase
	multicloud.STagBase

	region *SRegion

	Id   string
	Name string
}

func (self *SZone) GetName() string {
	return self.Name
}

func (self *SZone) GetId() string {
	return self.Name
}

func (self *SZone) GetGlobalId() string {
	return fmt.Sprintf("%s/%s/%s", api.CLOUD_PROVIDER_ORACLE, self.region.RegionName, self.Name)
}

func (self *SZone) GetStatus() string {
	return api.ZONE_ENABLE
}

func (self *SZone) GetIRegion() cloudprovider.ICloudRegion {
	return self.region
}

func (self *SZone) GetI18n() cloudprovider.SModelI18nTable {
	table := cloudprovider.SModelI18nTable{}
	return table
}

func (self *SZone) getHost() *SHost {
	return &SHost{zone: self}
}

func (self *SZone) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	return []cloudprovider.ICloudHost{self.getHost()}, nil
}

func (self *SZone) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	hosts, err := self.GetIHosts()
	if err != nil {
		return nil, err
	}
	for i := range hosts {
		if hosts[i].GetGlobalId() == id {
			return hosts[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SZone) getStorage() *SStorage {
	return &SStorage{zone: self}
}

func (self *SZone) getBootStorage() *SStorage {
	return &SStorage{zone: self, boot: true}
}

func (self *SZone) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	return []cloudprovider.ICloudStorage{self.getStorage(), self.getBootStorage()}, nil
}

func (self *SZone) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	storages, err := self.GetIStorages()
	if err != nil {
		return nil, err
	}
	for i := range storages {
		if storages[i].GetGlobalId() == id {
			return storages[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetZones() ([]SZone, error) {
	resp, err := self.list(SERVICE_IDENTITY, "availabilityDomains", nil)
	if err != nil {
		return nil, err
	}
	zones := []SZone{}
	err = resp.Unmarshal(&zones)
	if err != nil {
		return nil, err
	}
	return zones, nil
}
