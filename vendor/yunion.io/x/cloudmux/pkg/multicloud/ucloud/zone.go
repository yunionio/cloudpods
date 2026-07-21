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

package ucloud

import (
	"fmt"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SZone struct {
	multicloud.SResourceBase
	UcloudTags
	region *SRegion
	host   *SHost

	istorages    []cloudprovider.ICloudStorage
	storageTypes []string

	RegionId   string `json:"Region"`
	ZoneId     string `json:"Zone"`
	LocalName  string `json:"LocalName"`
	Default    bool   `json:"Default"`
	Permission string `json:"Permission"`
}

func (self *SZone) getHost() *SHost {
	if self.host == nil {
		self.host = &SHost{zone: self, projectId: self.region.client.projectId}
	}
	return self.host
}

func (self *SZone) fetchStorages() error {
	storageTypes, err := self.region.GetZoneStorageTypes(self.GetId())
	if err != nil {
		return err
	}
	for _, storageType := range api.UCLOUD_LOCAL_STORAGES {
		if !utils.IsInStringArray(storageType, storageTypes) {
			storageTypes = append(storageTypes, storageType)
		}
	}
	self.storageTypes = storageTypes
	self.istorages = make([]cloudprovider.ICloudStorage, len(self.storageTypes))
	for i, sc := range self.storageTypes {
		self.istorages[i] = &SStorage{zone: self, storageType: sc}
	}
	return nil
}

func (self *SZone) GetId() string {
	return self.ZoneId
}

func (self *SZone) GetName() string {
	return fmt.Sprintf("%s %s", CLOUD_PROVIDER_UCLOUD_CN, self.LocalName)
}

func (self *SZone) GetI18n() cloudprovider.SModelI18nTable {
	name := self.GetName()
	en := fmt.Sprintf("%s %s", CLOUD_PROVIDER_UCLOUD_EN, self.LocalName)
	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(name).CN(name).EN(en)
	return table
}

func (self *SZone) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", self.region.GetGlobalId(), self.GetId())
}

func (self *SZone) GetStatus() string {
	return api.ZONE_ENABLE
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
	if self.istorages == nil {
		err := self.fetchStorages()
		if err != nil {
			return nil, errors.Wrapf(err, "fetchStorages")
		}
	}
	return self.istorages, nil
}

func (self *SZone) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	if self.istorages == nil {
		err := self.fetchStorages()
		if err != nil {
			return nil, errors.Wrapf(err, "fetchStorages")
		}
	}
	for i := range self.istorages {
		if self.istorages[i].GetGlobalId() == id {
			return self.istorages[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

// https://docs.ucloud.cn/api/uhost-api/describe_uhost_instance
func (self *SZone) GetInstances() ([]SInstance, error) {
	return self.region.GetInstances(self.GetId(), "")
}

func (self *SZone) GetIWires() ([]cloudprovider.ICloudWire, error) {
	vpcs, err := self.region.GetVpcs("")
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

// https://docs.ucloud.cn/api/uhost-api/describe_uhost_instance
func (self *SRegion) GetInstances(zoneId string, instanceId string) ([]SInstance, error) {
	instances := make([]SInstance, 0)

	params := NewUcloudParams()
	if len(zoneId) > 0 {
		params.Set("Zone", zoneId)
	}

	if len(instanceId) > 0 {
		params.Set("UHostIds.0", instanceId)
	}

	err := self.DoListAll("DescribeUHostInstance", params, &instances)
	return instances, err
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
