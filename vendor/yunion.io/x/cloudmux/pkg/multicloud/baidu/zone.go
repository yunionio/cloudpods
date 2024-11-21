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

package baidu

import (
	"fmt"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/pkg/errors"
)

type SZone struct {
	multicloud.SResourceBase
	SBaiduTag
	region *SRegion

	ZoneName string `json:"zoneName"`
}

func (zone *SZone) GetId() string {
	return zone.ZoneName
}

func (zone *SZone) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", zone.region.GetGlobalId(), zone.ZoneName)
}

func (zone *SZone) GetName() string {
	return zone.ZoneName
}

func (zone *SZone) GetStatus() string {
	return api.ZONE_ENABLE
}

func (zone *SZone) IsEmulated() bool {
	return false
}

func (zone *SZone) Refresh() error {
	return nil
}

func (zone *SZone) GetIRegion() cloudprovider.ICloudRegion {
	return zone.region
}

func (zone *SZone) GetI18n() cloudprovider.SModelI18nTable {
	table := cloudprovider.SModelI18nTable{}
	return table
}

func (region *SRegion) GetZones() ([]SZone, error) {
	resp, err := region.bccList("v2/zone", nil)
	if err != nil {
		return nil, err
	}
	ret := []SZone{}
	err = resp.Unmarshal(&ret, "zones")
	if err != nil {
		return nil, err
	}
	for i := range ret {
		ret[i].region = region
	}
	return ret, nil
}

func (zone *SZone) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	storages, err := zone.region.GetStorageTypes(zone.ZoneName)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudStorage{}
	for i := range storages {
		storages[i].zone = zone
		ret = append(ret, &storages[i])
	}
	return ret, nil
}

func (zone *SZone) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	storages, err := zone.GetIStorages()
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

func (zone *SZone) getHost() *SHost {
	return &SHost{zone: zone}
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
	return nil, cloudprovider.ErrNotFound
}

func (zone *SZone) GetIWires() ([]cloudprovider.ICloudWire, error) {
	vpcs, err := zone.region.GetVpcs()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudWire{}
	for i := range vpcs {
		wire := &SWire{zone: zone, vpc: &vpcs[i]}
		ret = append(ret, wire)
	}
	return ret, nil
}
