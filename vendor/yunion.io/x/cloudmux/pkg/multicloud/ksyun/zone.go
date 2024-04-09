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

package ksyun

import (
	"fmt"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
)

type SZone struct {
	multicloud.SResourceBase
	region *SRegion
	host   *SHost
	SKsTag

	AvailabilityZone string
}

func (region *SRegion) GetZones() ([]SZone, error) {
	params := map[string]string{}
	if len(region.Region) > 0 {
		params = map[string]string{"Region": region.Region}
	}
	resp, err := region.ecsRequest("DescribeAvailabilityZones", params)
	if err != nil {
		return nil, errors.Wrap(err, "request zone")
	}
	zones := []SZone{}
	err = resp.Unmarshal(&zones, "AvailabilityZoneSet")
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal zones")
	}
	return zones, nil
}

func (zone *SZone) GetId() string {
	return zone.AvailabilityZone
}

func (zone *SZone) GetName() string {
	return zone.AvailabilityZone
}

func (zone *SZone) GetI18n() cloudprovider.SModelI18nTable {
	return nil
}

func (zone *SZone) GetGlobalId() string {
	return fmt.Sprintf("%s/%s/%s", api.CLOUD_PROVIDER_KSYUN, zone.region.GetId(), zone.AvailabilityZone)
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

func (zone *SZone) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	host := zone.getHost()
	if host.GetGlobalId() == id {
		return host, nil
	}
	return nil, cloudprovider.ErrNotFound
}

func (zone *SZone) getHost() *SHost {
	if zone.host == nil {
		zone.host = &SHost{zone: zone}
	}
	return zone.host
}

func (zone *SZone) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	storages, err := zone.GetStorages()
	if err != nil {
		return nil, errors.Wrap(err, "GetStorages")
	}
	istorages := []cloudprovider.ICloudStorage{}
	for i := range storages {
		storages[i].zone = zone
		istorages = append(istorages, &storages[i])
	}
	return istorages, nil
}

func (zone *SZone) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	istorages, err := zone.GetIStorages()
	if err != nil {
		return nil, errors.Wrap(err, "GetIStorages")
	}
	for _, istorage := range istorages {
		if istorage.GetGlobalId() == id {
			return istorage, nil
		}
	}
	return nil, errors.Wrapf(errors.ErrNotFound, "storage id:%s", id)
}

func (zone *SZone) GetIWires() ([]cloudprovider.ICloudWire, error) {
	vpcs, err := zone.region.GetVpcs([]string{})
	if err != nil {
		return nil, errors.Wrap(err, "GetVpcs")
	}
	for i := range vpcs {
		vpcs[i].region = zone.region
	}
	iwires := []cloudprovider.ICloudWire{}
	for _, vpc := range vpcs {
		iwires = append(iwires, &SWire{
			vpc:  &vpc,
			zone: zone,
		})
	}
	return iwires, nil
}

func (zone *SZone) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	return []cloudprovider.ICloudHost{zone.getHost()}, nil
}

func (zone *SZone) GetDescription() string {
	return ""
}

func (zone *SZone) GetStorages() ([]SStorage, error) {
	zoneDiskType := []string{}
	for i := range ksDiskTypes {
		params := map[string]string{
			"VolumeType": ksDiskTypes[i],
		}
		resp, err := zone.region.ebsRequest("DescribeAvailabilityZones", params)
		if err != nil {
			return nil, errors.Wrapf(err, "%s:ValidateAttachInstance", ksDiskTypes[i])
		}
		zoneList := []string{}
		err = resp.Unmarshal(&zoneList, "AvailabilityZones")
		if err != nil {
			return nil, errors.Wrap(err, "unmarshal zoneList")
		}
		if utils.IsInStringArray(zone.GetName(), zoneList) {
			zoneDiskType = append(zoneDiskType, ksDiskTypes[i])
		}
	}
	storages := []SStorage{}
	for i := range zoneDiskType {
		storages = append(storages, SStorage{zone: zone, StorageType: zoneDiskType[i]})
	}
	return storages, nil
}
