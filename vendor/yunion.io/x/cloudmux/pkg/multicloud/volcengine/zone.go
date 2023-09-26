// Copyright 2023 Yunion
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

package volcengine

import (
	"fmt"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/pkg/errors"
)

var StorageTypes = []string{
	api.STORAGE_VOLCENGINE_FlexPL,
	api.STORAGE_VOLCENGINE_PL0,
	api.STORAGE_VOLCENGINE_PTSSD,
}

type SSupportedResource struct {
	Status string
	Value  string
}

type SAvailableResource struct {
	Type               string
	SupportedResources []SSupportedResource
}

type SZone struct {
	multicloud.SResourceBase
	VolcEngineTags
	region *SRegion

	host *SHost

	istorages          []cloudprovider.ICloudStorage
	Status             string
	AvailableResources []SAvailableResource

	storageTypes []string

	ZoneId    string
	RegionId  string
	LocalName string
}

func (zone *SZone) GetIWires() ([]cloudprovider.ICloudWire, error) {
	vpcs, err := zone.region.GetAllVpcs()
	if err != nil {
		return nil, errors.Wrapf(err, "GetVpcs")
	}
	ret := []cloudprovider.ICloudWire{}
	for i := range vpcs {
		vpcs[i].region = zone.region
		ret = append(ret, &SWire{zone: zone, vpc: &vpcs[i]})
	}
	return ret, nil
}

func (zone *SZone) GetId() string {
	return zone.ZoneId
}

func (zone *SZone) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", zone.region.GetGlobalId(), zone.ZoneId)
}

func (zone *SZone) GetName() string {
	return fmt.Sprintf("%s", zone.ZoneId)
}

func (zone *SZone) GetI18n() cloudprovider.SModelI18nTable {
	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(zone.GetName()).CN(zone.GetName())
	return table
}

func (zone *SZone) GetStatus() string {
	return api.ZONE_ENABLE
}

func (zone *SZone) Refresh() error {
	return nil
}

func (zone *SZone) GetIRegion() cloudprovider.ICloudRegion {
	return zone.region
}

// Host
func (zone *SZone) getHost() *SHost {
	if zone.host == nil {
		zone.host = &SHost{zone: zone}
	}
	return zone.host
}

func (zone *SZone) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	return []cloudprovider.ICloudHost{zone.getHost()}, nil
}

func (zone *SZone) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	hosts, err := zone.GetIHosts()
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

// Storage
func (zone *SZone) getStorageType() {
	if len(zone.storageTypes) == 0 {
		zone.storageTypes = StorageTypes
	}
}

func (zone *SZone) fetchStorages() error {
	zone.getStorageType()
	zone.istorages = make([]cloudprovider.ICloudStorage, len(zone.storageTypes))

	for i, sc := range zone.storageTypes {
		storage := SStorage{zone: zone, storageType: sc}
		zone.istorages[i] = &storage
	}
	return nil
}

func (zone *SZone) getStorageByCategory(category string) (*SStorage, error) {
	storages, err := zone.GetIStorages()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(storages); i += 1 {
		storage := storages[i].(*SStorage)
		if storage.storageType == category {
			return storage, nil
		}
	}
	return nil, fmt.Errorf("no such storage %s", category)
}

func (zone *SZone) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	if zone.istorages == nil {
		err := zone.fetchStorages()
		if err != nil {
			return nil, errors.Wrapf(err, "fetchStorages")
		}
	}
	return zone.istorages, nil
}

func (zone *SZone) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	if zone.istorages == nil {
		err := zone.fetchStorages()
		if err != nil {
			return nil, errors.Wrapf(err, "fetchStorages")
		}
	}
	for i := 0; i < len(zone.istorages); i += 1 {
		if zone.istorages[i].GetGlobalId() == id {
			return zone.istorages[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}
