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
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SZone struct {
	multicloud.SResourceBase
	AzureTags
	region *SRegion

	Name string
}

func (self *SZone) GetId() string {
	return self.region.client.cpcfg.Id
}

func (self *SZone) GetName() string {
	return self.region.GetName()
}

func (self *SZone) GetI18n() cloudprovider.SModelI18nTable {
	return self.region.GetI18n()
}

func (self *SZone) GetGlobalId() string {
	return self.region.GetGlobalId()
}

func (self *SZone) IsEmulated() bool {
	return true
}

func (self *SZone) GetStatus() string {
	return api.ZONE_ENABLE
}

func (self *SZone) Refresh() error {
	// do nothing
	return nil
}

func (self *SZone) getHost() *SHost {
	return &SHost{zone: self}
}

func (self *SZone) GetIRegion() cloudprovider.ICloudRegion {
	return self.region
}

func (self *SZone) ListStorageTypes() []SStorage {
	storages := []SStorage{}
	for _, storageType := range STORAGETYPES {
		storage := SStorage{zone: self, storageType: storageType}
		storages = append(storages, storage)
	}
	return storages
}

func (self *SZone) getIStorages() []cloudprovider.ICloudStorage {
	ret := []cloudprovider.ICloudStorage{}
	storages := self.ListStorageTypes()
	for i := range storages {
		storages[i].zone = self
		ret = append(ret, &storages[i])
	}
	return ret
}

func (self *SZone) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	return self.getIStorages(), nil
}

func (self *SZone) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	storages, err := self.GetIStorages()
	if err != nil {
		return nil, errors.Wrapf(err, "GetIStorages")
	}
	for i := range storages {
		if storages[i].GetGlobalId() == id {
			return storages[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SZone) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	hosts, err := self.GetIHosts()
	if err != nil {
		return nil, errors.Wrapf(err, "GetIHosts")
	}
	for i := range hosts {
		if hosts[i].GetGlobalId() == id {
			return hosts[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SZone) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	return []cloudprovider.ICloudHost{self.getHost()}, nil
}

func (self *SZone) GetIWires() ([]cloudprovider.ICloudWire, error) {
	vpcs, err := self.region.ListVpcs()
	if err != nil {
		return nil, errors.Wrapf(err, "ListVpcs")
	}
	wires := []cloudprovider.ICloudWire{}
	for i := range vpcs {
		vpcs[i].region = self.region
		wire := &SWire{vpc: &vpcs[i], zone: self}
		wires = append(wires, wire)
	}
	return wires, nil
}
