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

package ecloud

import (
	"fmt"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SZone struct {
	multicloud.SResourceBase
	EcloudTags
	region *SRegion
	host   *SHost

	istorages []cloudprovider.ICloudStorage

	ID      string `json:"id"`
	Name    string `json:"name"`
	Region  string `json:"region"`
	Deleted bool
	Visible bool
}

func (z *SZone) GetId() string {
	return z.ID
}

func (z *SZone) GetName() string {
	//return fmt.Sprintf("%s %s", CLOUD_PROVIDER_ECLOUD_CN, z.Name)
	return z.Name
}

func (z *SZone) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", z.region.GetGlobalId(), z.ID)
}

func (z *SZone) GetStatus() string {
	if z.Deleted || !z.Visible {
		return "disable"
	}
	return "enable"
}

func (z *SZone) Refresh() error {
	return nil
}

func (z *SZone) IsEmulated() bool {
	return false
}

func (z *SZone) GetI18n() cloudprovider.SModelI18nTable {
	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(z.GetName()).CN(z.GetName()).EN(fmt.Sprintf("%s %s", CLOUD_PROVIDER_ECLOUD_EN, z.Name))
	return table
}

func (z *SZone) GetIRegion() cloudprovider.ICloudRegion {
	return z.region
}

func (z *SZone) getHost() cloudprovider.ICloudHost {
	if z.host == nil {
		z.host = &SHost{
			zone: z,
		}
	}
	return z.host
}

func (z *SZone) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	return []cloudprovider.ICloudHost{z.getHost()}, nil
}

func (z *SZone) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	host := z.getHost()
	if host.GetGlobalId() == id {
		return host, nil
	}
	return nil, cloudprovider.ErrNotFound
}

func (z *SZone) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	if z.istorages == nil {
		err := z.fetchStorages()
		if err != nil {
			return nil, err
		}
	}
	return z.istorages, nil
}

func (z *SZone) fetchStorages() error {
	istorages := make([]cloudprovider.ICloudStorage, len(storageTypes))
	for i := range istorages {
		istorages[i] = &SStorage{
			zone:        z,
			storageType: storageTypes[i],
		}
	}
	z.istorages = istorages
	return nil
}

func (z *SZone) getStorageByType(t string) (*SStorage, error) {
	istorages, err := z.GetIStorages()
	if err != nil {
		return nil, err
	}
	for i := range istorages {
		if istorages[i].GetStorageType() == t {
			return istorages[i].(*SStorage), nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (z *SZone) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	istorages, err := z.GetIStorages()
	if err != nil {
		return nil, err
	}
	for i := range istorages {
		if istorages[i].GetGlobalId() == id {
			return istorages[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

type SZoneRegionBase struct {
	Region string
}
