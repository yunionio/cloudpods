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

package huawei

import (
	"fmt"
	"strings"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type ZoneState struct {
	Available bool `json:"available"`
}

type SZone struct {
	multicloud.SResourceBase
	HuaweiTags
	region *SRegion
	host   *SHost

	ZoneState ZoneState `json:"zoneState"`
	ZoneName  string    `json:"zoneName"`
}

func (self *SZone) getHost() *SHost {
	if self.host == nil {
		self.host = &SHost{zone: self}
	}
	return self.host
}

func (self *SZone) GetId() string {
	return self.ZoneName
}

func (self *SZone) GetName() string {
	return fmt.Sprintf("%s %s", CLOUD_PROVIDER_HUAWEI_CN, self.ZoneName)
}

func (self *SZone) GetI18n() cloudprovider.SModelI18nTable {
	en := fmt.Sprintf("%s %s", CLOUD_PROVIDER_HUAWEI_EN, self.ZoneName)
	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(self.GetName()).CN(self.GetName()).EN(en)
	return table
}

func (self *SZone) GetGlobalId() string {
	return fmt.Sprintf("%s/%s/%s", api.CLOUD_PROVIDER_HUAWEI, self.region.getId(), self.ZoneName)
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
	storageTypes, err := self.region.GetDiskTypes()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudStorage{}
	for _, storageType := range storageTypes {
		if strings.Contains(storageType.ExtraSpecs.RESKEYAvailabilityZones, self.ZoneName) {
			ret = append(ret, &SStorage{
				zone:         self,
				storageType:  storageType.Name,
				volumeTypeId: storageType.Id,
			})
		}
	}
	return ret, nil
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
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SZone) GetIWires() ([]cloudprovider.ICloudWire, error) {
	vpcs, err := self.region.GetVpcs()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudWire{}
	for i := range vpcs {
		vpcs[i].region = self.region
		ret = append(ret, &SWire{vpc: &vpcs[i]})
	}
	return ret, nil
}

func (self *SZone) getStorageByCategory(category string) (*SStorage, error) {
	storages, err := self.GetIStorages()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(storages); i += 1 {
		storage := storages[i].(*SStorage)
		if storage.storageType == category {
			return storage, nil
		}
	}
	return nil, fmt.Errorf("No such storage %s", category)
}

func (self *SRegion) getZoneById(id string) (*SZone, error) {
	zones, err := self.GetZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(zones); i += 1 {
		zones[i].region = self
		zone := zones[i]
		if zone.GetId() == id {
			return &zone, nil
		}
	}
	return nil, fmt.Errorf("no such zone %s", id)
}
