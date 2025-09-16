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

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SZone struct {
	multicloud.SResourceBase
	CtyunTags
	region *SRegion

	AzDisplayName string
	Name          string
}

func (self *SZone) GetId() string {
	return self.Name
}

func (self *SZone) GetName() string {
	return self.AzDisplayName
}

func (self *SZone) GetI18n() cloudprovider.SModelI18nTable {
	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(self.GetName()).CN(self.GetName()).EN(self.AzDisplayName)
	return table
}

func (self *SZone) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", self.region.GetGlobalId(), self.Name)
}

func (self *SZone) GetStatus() string {
	return api.ZONE_ENABLE
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

func (self *SZone) GetStorages() ([]SStorage, error) {
	product, err := self.region.getProduct()
	if err != nil {
		return nil, errors.Wrapf(err, "getProduct")
	}
	ret := []SStorage{}
	for i := range product.Ebs.StorageType {
		storage := SStorage{
			zone:        self,
			storageType: product.Ebs.StorageType[i].Type,
			Name:        product.Ebs.StorageType[i].Name,
		}
		ret = append(ret, storage)
	}
	return ret, nil
}

func (self *SZone) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	storages, err := self.GetStorages()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudStorage{}
	for i := range storages {
		ret = append(ret, &storages[i])
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
	return nil, cloudprovider.ErrNotFound
}

func (self *SZone) getHost() *SHost {
	return &SHost{zone: self}
}

func (self *SZone) GetIWires() ([]cloudprovider.ICloudWire, error) {
	vpcs, err := self.region.GetVpcs()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudWire{}
	for i := range vpcs {
		vpcs[i].region = self.region
		wires, err := vpcs[i].GetIWires()
		if err != nil {
			return nil, err
		}
		ret = append(ret, wires...)
	}
	return ret, nil
}

func (self *SRegion) GetZones() ([]SZone, error) {
	resp, err := self.list(SERVICE_ECS, "/v4/region/get-zones", nil)
	if err != nil {
		return nil, err
	}
	ret := struct {
		ReturnObj struct {
			ZoneList []SZone
		}
	}{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, err
	}
	return ret.ReturnObj.ZoneList, nil
}
