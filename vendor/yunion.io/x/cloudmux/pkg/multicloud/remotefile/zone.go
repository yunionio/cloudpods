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

package remotefile

import (
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SZone struct {
	SResourceBase
	region *SRegion

	RegionId string
}

func (self *SZone) GetIRegion() cloudprovider.ICloudRegion {
	return self.region
}

func (self *SZone) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	hosts, err := self.region.client.GetHosts()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudHost{}
	for i := range hosts {
		if hosts[i].ZoneId != self.GetId() {
			continue
		}
		hosts[i].zone = self
		ret = append(ret, &hosts[i])
	}
	return ret, nil
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
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SZone) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	storages, err := self.region.client.GetStorages()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudStorage{}
	for i := range storages {
		if storages[i].ZoneId != self.GetId() {
			continue
		}
		storages[i].zone = self
		ret = append(ret, &storages[i])
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
	return nil, cloudprovider.ErrNotFound
}
