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

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SZone struct {
	multicloud.SResourceBase
	EcloudTags
	region *SRegion

	ZoneId   string `json:"zoneId"`
	ZoneName string `json:"zoneName"`
	ZoneCode string `json:"zoneCode"`
}

func (z *SZone) GetId() string {
	return z.ZoneCode
}

func (z *SZone) GetName() string {
	return fmt.Sprintf("%s %s", z.region.GetName(), z.ZoneName)
}

func (z *SZone) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", z.region.GetGlobalId(), z.ZoneCode)
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
	return table
}

func (z *SZone) GetIRegion() cloudprovider.ICloudRegion {
	return z.region
}

func (z *SZone) GetHost() *SHost {
	return &SHost{zone: z}
}

func (z *SZone) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	return []cloudprovider.ICloudHost{z.GetHost()}, nil
}

func (z *SZone) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	host := z.GetHost()
	if host.GetGlobalId() == id {
		return host, nil
	}
	return nil, cloudprovider.ErrNotFound
}

func (z *SZone) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	// 直接复用区域级 GetStorages 能力，并按当前 Zone 过滤。
	storages, err := z.region.GetStorages(z.ZoneCode)
	if err != nil {
		return nil, err
	}
	istorages := make([]cloudprovider.ICloudStorage, len(storages))
	for i := range storages {
		// 确保 zone 指针为当前 z
		storages[i].zone = z
		istorages[i] = &storages[i]
	}
	return istorages, nil
}

func (z *SZone) GetStorageByType(t string) (*SStorage, error) {
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

func (z *SZone) GetIWires() ([]cloudprovider.ICloudWire, error) {
	vpcs, err := z.region.GetVpcs()
	if err != nil {
		return nil, err
	}
	for i := range vpcs {
		vpcs[i].region = z.region
	}
	wires := []cloudprovider.ICloudWire{}
	for i := range vpcs {
		wires = append(wires, &SWire{zone: z, vpc: &vpcs[i]})
	}
	return wires, nil
}
