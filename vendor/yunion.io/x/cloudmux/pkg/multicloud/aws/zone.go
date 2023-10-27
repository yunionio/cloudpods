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

package aws

import (
	"fmt"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

var StorageTypes = []string{
	api.STORAGE_GP2_SSD,
	api.STORAGE_GP3_SSD,
	api.STORAGE_IO1_SSD,
	api.STORAGE_IO2_SSD,
	api.STORAGE_ST1_HDD,
	api.STORAGE_SC1_HDD,
	api.STORAGE_STANDARD_HDD,
}

type SZone struct {
	multicloud.SResourceBase
	AwsTags
	region *SRegion

	ZoneName           string `xml:"zoneName"`
	ZoneId             string `xml:"zoneId"`
	ZoneState          string `xml:"zoneState"`
	RegionName         string `xml:"regionName"`
	GroupName          string `xml:"groupName"`
	OptInStatus        string `xml:"optInStatus"`
	NetworkBorderGroup string `xml:"networkBorderGroup"`
	ZoneType           string `xml:"zoneType"`
}

func (self *SZone) GetIWires() ([]cloudprovider.ICloudWire, error) {
	vpcs, err := self.region.GetVpcs(nil)
	if err != nil {
		return nil, errors.Wrapf(err, "GetVpcs")
	}
	ret := []cloudprovider.ICloudWire{}
	for i := range vpcs {
		vpcs[i].region = self.region
		ret = append(ret, &SWire{zone: self, vpc: &vpcs[i]})
	}
	return ret, nil
}

func (self *SZone) GetId() string {
	return self.ZoneName
}

func (self *SZone) GetName() string {
	return fmt.Sprintf("%s %s", CLOUD_PROVIDER_AWS_CN, self.ZoneName)
}

func (self *SZone) GetI18n() cloudprovider.SModelI18nTable {
	en := fmt.Sprintf("%s %s", CLOUD_PROVIDER_AWS_EN, self.ZoneName)
	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(self.GetName()).CN(self.GetName()).EN(en)
	return table
}

func (self *SZone) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", self.region.GetGlobalId(), self.ZoneName)
}

func (self *SZone) GetStatus() string {
	if self.ZoneState == "available" {
		return api.ZONE_ENABLE
	}
	return api.ZONE_SOLDOUT
}

func (self *SZone) GetIRegion() cloudprovider.ICloudRegion {
	return self.region
}

func (self *SZone) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	host := &SHost{zone: self}
	return []cloudprovider.ICloudHost{host}, nil
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
	return nil, errors.Wrap(cloudprovider.ErrNotFound, "GetIHostById")
}

func (self *SZone) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	ret := []cloudprovider.ICloudStorage{}
	for i := range StorageTypes {
		storage := &SStorage{zone: self, storageType: StorageTypes[i]}
		ret = append(ret, storage)
	}
	return ret, nil
}

func (self *SZone) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	storages, err := self.GetIStorages()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(storages); i += 1 {
		if storages[i].GetGlobalId() == id {
			return storages[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "not found %s", id)
}

func (self *SZone) getStorageByCategory(category string) (*SStorage, error) {
	storages, err := self.GetIStorages()
	if err != nil {
		return nil, errors.Wrap(err, "GetIStorages")
	}
	for i := 0; i < len(storages); i += 1 {
		storage := storages[i].(*SStorage)
		if storage.storageType == category {
			return storage, nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, category)
}

func (self *SZone) GetDescription() string {
	return self.AwsTags.GetDescription()
}
