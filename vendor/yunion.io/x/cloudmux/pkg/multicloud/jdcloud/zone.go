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

package jdcloud

import (
	"fmt"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type sZoneIden struct {
	Id   string
	Name string
}

var ZonesInRegion = map[string][]sZoneIden{
	"cn-north-1": {
		{
			Id:   "cn-north-1a",
			Name: "可用区A",
		},
		{
			Id:   "cn-north-1b",
			Name: "可用区B",
		},
		{
			Id:   "cn-north-1c",
			Name: "可用区C",
		},
	},
	"cn-east-1": {
		{
			Id:   "cn-east-1a",
			Name: "可用区A",
		},
	},
	"cn-east-2": {
		{
			Id:   "cn-east-2a",
			Name: "可用区A",
		},
		{
			Id:   "cn-east-2b",
			Name: "可用区B",
		},
		{
			Id:   "cn-east-2c",
			Name: "可用区C",
		},
	},
	"cn-south-1": {
		{
			Id:   "cn-south-1a",
			Name: "可用区A",
		},
		{
			Id:   "cn-south-1b",
			Name: "可用区B",
		},
		{
			Id:   "cn-south-1c",
			Name: "可用区C",
		},
	},
}

type SZone struct {
	multicloud.SResourceBase
	JdcloudTags
	region *SRegion

	ihost     cloudprovider.ICloudHost
	istorages []cloudprovider.ICloudStorage

	ID   string
	Name string
}

func (z *SZone) GetId() string {
	return z.ID
}

func (z *SZone) GetName() string {
	return z.Name
}

func (z *SZone) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", z.region.GetGlobalId(), z.ID)
}

func (z *SZone) GetStatus() string {
	return api.ZONE_ENABLE
}

func (z *SZone) Refresh() error {
	return nil
}

func (z *SZone) IsEmulated() bool {
	return false
}

func (z *SZone) GetI18n() cloudprovider.SModelI18nTable {
	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(z.GetName()).CN(z.GetName()).EN(fmt.Sprintf("%s %s", CLOUD_PROVIDER_JDCLOUD_EN, z.Name))
	return table
}

func (z *SZone) GetIRegion() cloudprovider.ICloudRegion {
	return z.region
}

func (z *SZone) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	if z.ihost != nil {
		return []cloudprovider.ICloudHost{z.ihost}, nil
	}
	z.ihost = &SHost{
		zone: z,
	}
	return []cloudprovider.ICloudHost{z.ihost}, nil
}

func (z *SZone) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	ihosts, err := z.GetIHosts()
	if err != nil {
		return nil, err
	}
	for i := range ihosts {
		if ihosts[i].GetGlobalId() == id {
			return ihosts[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
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

func (z *SZone) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	if z.istorages == nil {
		err := z.fetchStorages()
		if err != nil {
			return nil, err
		}
	}
	return z.istorages, nil
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
