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

package zstack

import (
	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SZone struct {
	region *SRegion

	ZStackBasic
	Type  string
	State string

	iwires    []cloudprovider.ICloudWire
	istorages []cloudprovider.ICloudStorage

	ihosts []cloudprovider.ICloudHost
}

func (zone *SZone) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (zone *SZone) GetId() string {
	return zone.Name
}

func (zone *SZone) GetName() string {
	return zone.Name
}

func (zone *SZone) GetGlobalId() string {
	return zone.GetId()
}

func (zone *SZone) IsEmulated() bool {
	return false
}

func (zone *SZone) GetStatus() string {
	if zone.State == "Enabled" {
		return api.ZONE_ENABLE
	}
	return api.ZONE_DISABLE
}

func (zone *SZone) Refresh() error {
	// do nothing
	return nil
}

func (zone *SZone) GetIRegion() cloudprovider.ICloudRegion {
	return zone.region
}

func (zone *SZone) fetchStorages(clusterId string) error {
	storages, err := zone.region.getIStorages(zone.UUID)
	if err != nil {
		return err
	}
	zone.istorages = storages
	return nil
}

func (zone *SZone) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	if zone.istorages == nil || len(zone.istorages) == 0 {
		return zone.istorages, zone.fetchStorages("")
	}
	return zone.istorages, nil
}

func (zone *SZone) GetIStorageById(storageId string) (cloudprovider.ICloudStorage, error) {
	err := zone.fetchStorages("")
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(zone.istorages); i++ {
		if zone.istorages[i].GetGlobalId() == storageId {
			return zone.istorages[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (zone *SZone) GetIHostById(hostId string) (cloudprovider.ICloudHost, error) {
	host, err := zone.region.GetHost(hostId)
	if err != nil {
		return nil, err
	}
	if host.ZoneUUID != zone.UUID {
		return nil, cloudprovider.ErrNotFound
	}
	host.zone = zone
	return host, nil
}

func (zone *SZone) fetchHosts() error {
	hosts, err := zone.region.GetHosts(zone.UUID, "")
	if err != nil {
		return err
	}
	zone.ihosts = []cloudprovider.ICloudHost{}
	for i := 0; i < len(hosts); i++ {
		hosts[i].zone = zone
		zone.ihosts = append(zone.ihosts, &hosts[i])
	}
	return nil
}

func (zone *SZone) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	if zone.ihosts == nil || len(zone.ihosts) == 0 {
		return zone.ihosts, zone.fetchHosts()
	}
	return zone.ihosts, nil
}

func (zone *SZone) GetIWires() ([]cloudprovider.ICloudWire, error) {
	if zone.iwires == nil || len(zone.iwires) == 0 {
		wires, err := zone.region.GetWires(zone.UUID, "", "")
		if err != nil {
			return nil, err
		}
		zone.iwires = []cloudprovider.ICloudWire{}
		for i := 0; i < len(wires); i++ {
			zone.iwires = append(zone.iwires, &wires[i])
		}
	}
	return zone.iwires, nil
}
