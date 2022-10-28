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

package openstack

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type ZoneState struct {
	Available bool
}

type HostState struct {
	Available bool
	Active    bool
	UpdatedAt time.Time
}

type SZone struct {
	multicloud.SResourceBase
	OpenStackTags
	region *SRegion

	ZoneName  string
	ZoneState ZoneState

	Hosts map[string]jsonutils.JSONObject

	hosts []SHypervisor
}

func (zone *SZone) GetId() string {
	return zone.ZoneName
}

func (zone *SZone) GetName() string {
	return zone.ZoneName
}

func (zone *SZone) GetI18n() cloudprovider.SModelI18nTable {
	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(zone.GetName()).CN(zone.GetName())
	return table
}

func (zone *SZone) GetGlobalId() string {
	return fmt.Sprintf("%s/%s/%s", CLOUD_PROVIDER_OPENSTACK, zone.region.Name, zone.ZoneName)
}

func (zone *SZone) IsEmulated() bool {
	return false
}

func (zone *SZone) GetStatus() string {
	if zone.ZoneState.Available {
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

func (zone *SZone) getStorageByCategory(category, host string) (*SStorage, error) {
	storages, err := zone.region.GetStorageTypes()
	if err != nil {
		return nil, errors.Wrap(err, "GetStorageTypes")
	}
	for i := range storages {
		if storages[i].Name == category || storages[i].ExtraSpecs.VolumeBackendName == category {
			storages[i].zone = zone
			return &storages[i], nil
		}
	}
	for i := range storages {
		if strings.HasSuffix(host, "#"+storages[i].Name) || strings.HasSuffix(host, "#"+storages[i].ExtraSpecs.VolumeBackendName) {
			storages[i].zone = zone
			return &storages[i], nil
		}
	}
	return nil, fmt.Errorf("No such storage [%s]", category)
}

func (zone *SZone) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	storages, err := zone.region.GetStorageTypes()
	if err != nil && errors.Cause(err) != ErrNoEndpoint {
		return nil, errors.Wrap(err, "GetStorageTypes")
	}
	istorages := []cloudprovider.ICloudStorage{}
	for i := range storages {
		storages[i].zone = zone
		istorages = append(istorages, &storages[i])
	}
	err = zone.fetchHosts()
	if err != nil {
		return nil, errors.Wrap(err, "fetchHosts")
	}
	for i := range zone.hosts {
		nova := &SNovaStorage{host: &zone.hosts[i], zone: zone}
		istorages = append(istorages, nova)
	}
	return istorages, nil
}

func (zone *SZone) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	istorages, err := zone.GetIStorages()
	if err != nil {
		return nil, errors.Wrap(err, "GetIStorages")
	}
	for i := 0; i < len(istorages); i++ {
		if istorages[i].GetGlobalId() == id {
			return istorages[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (zone *SZone) fetchHosts() error {
	if len(zone.hosts) > 0 {
		return nil
	}

	zone.hosts = []SHypervisor{}
	hypervisors, err := zone.region.GetHypervisors()
	if err != nil {
		return errors.Wrap(err, "GetHypervisors")
	}
	for i := range hypervisors {
		hypervisor := strings.ToLower(hypervisors[i].HypervisorType)
		// 过滤vmware的机器
		if strings.Index(hypervisor, "vmware") != -1 {
			continue
		}

		_, ok1 := zone.Hosts[hypervisors[i].HypervisorHostname]
		_, ok2 := zone.Hosts[hypervisors[i].Service.Host]
		if !ok1 && !ok2 {
			continue
		}
		zone.hosts = append(zone.hosts, hypervisors[i])
	}
	return nil

}

func (zone *SZone) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	err := zone.fetchHosts()
	if err != nil {
		return nil, errors.Wrap(err, "fetchHosts")
	}
	ihosts := []cloudprovider.ICloudHost{}
	for i := range zone.hosts {
		zone.hosts[i].zone = zone
		ihosts = append(ihosts, &zone.hosts[i])
	}
	return ihosts, nil
}

func (zone *SZone) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	ihosts, err := zone.GetIHosts()
	if err != nil {
		return nil, errors.Wrap(err, "GetIHosts")
	}
	for i := range ihosts {
		if ihosts[i].GetGlobalId() == id {
			return ihosts[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (region *SRegion) getZones() ([]SZone, error) {
	zones := []SZone{}
	resp, err := region.ecsList("os-availability-zone/detail", nil)
	if err != nil {
		return nil, errors.Wrap(err, "ecsList.os-availability-zone")
	}
	err = resp.Unmarshal(&zones, "availabilityZoneInfo")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return zones, nil
}
