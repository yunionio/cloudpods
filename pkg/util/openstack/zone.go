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
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type ZoneState struct {
	Available bool
}

const (
	HYPERVISORS_VERSION = "2.28"
)

type HostState struct {
	Available bool
	Active    bool
	UpdatedAt time.Time
}

type SZone struct {
	region *SRegion

	iwires    []cloudprovider.ICloudWire
	istorages []cloudprovider.ICloudStorage

	ZoneName string

	cachedHosts map[string][]string

	Hosts map[string]map[string]HostState
}

func (zone *SZone) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (zone *SZone) GetId() string {
	return zone.ZoneName
}

func (zone *SZone) GetName() string {
	return fmt.Sprintf("%s %s", CLOUD_PROVIDER_OPENSTACK, zone.ZoneName)
}

func (zone *SZone) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", zone.region.GetGlobalId(), zone.ZoneName)
}

func (zone *SZone) IsEmulated() bool {
	return true
}

func (zone *SZone) GetStatus() string {
	return api.ZONE_ENABLE
}

func (zone *SZone) Refresh() error {
	// do nothing
	return nil
}

func (zone *SZone) GetIRegion() cloudprovider.ICloudRegion {
	return zone.region
}

func (zone *SZone) GetIWires() ([]cloudprovider.ICloudWire, error) {
	return zone.iwires, nil
}

func (zone *SZone) getStorageByCategory(category string) (*SStorage, error) {
	storages, err := zone.GetIStorages()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(storages); i++ {
		storage := storages[i].(*SStorage)
		if strings.ToLower(storage.Name) == strings.ToLower(category) {
			return storage, nil
		}
	}
	return nil, fmt.Errorf("No such storage %s", category)
}

func (zone *SZone) addWire(wire *SWire) {
	if zone.iwires == nil {
		zone.iwires = []cloudprovider.ICloudWire{}
	}
	zone.iwires = append(zone.iwires, wire)
}

func (zone *SZone) fetchStorages() error {
	zone.istorages = []cloudprovider.ICloudStorage{}

	for _, service := range []string{"volumev3", "volumev2", "volume"} {
		_, resp, err := zone.region.List(service, "/types", "", nil)
		if err == nil {
			storages := []SStorage{}
			if err := resp.Unmarshal(&storages, "volume_types"); err != nil {
				return err
			}
			for i := 0; i < len(storages); i++ {
				storages[i].zone = zone
				zone.istorages = append(zone.istorages, &storages[i])
			}
			return nil
		}
		log.Debugf("failed to get volume types by service %s error: %v, try another", service, err)
	}
	return fmt.Errorf("failed to find storage types by cinder service")
}

func (zone *SZone) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	if zone.istorages == nil {
		zone.fetchStorages()
	}
	return zone.istorages, nil
}

func (zone *SZone) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	if zone.istorages == nil {
		zone.fetchStorages()
	}
	for i := 0; i < len(zone.istorages); i++ {
		if zone.istorages[i].GetGlobalId() == id {
			return zone.istorages[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

type SOsHost struct {
	Zone     string
	HostName string
	Service  string
}

func (zone *SZone) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	ihosts := []cloudprovider.ICloudHost{}
	hosts := []SHost{}

	// 尽可能的优先使用 os-hypervisor, 里面的信息更全些, 实在不行再使用 os-host
	_, resp, err := zone.region.List("compute", "/os-hypervisors/detail", "", nil)
	if err == nil {
		if err := resp.Unmarshal(&hosts, "hypervisors"); err != nil {
			return nil, err
		}
		for i := 0; i < len(hosts); i++ {
			// 过滤vmware的机器
			hypervisor := strings.ToLower(hosts[i].HypervisorType)
			if strings.Index(hypervisor, "vmware") != -1 {
				continue
			}
			hosts[i].zone = zone
			ihosts = append(ihosts, &hosts[i])
		}
		return ihosts, nil
	}
	_, resp, err = zone.region.List("compute", "/os-hosts", "", nil)
	if err != nil {
		return nil, err
	}

	_hosts := []SOsHost{}

	if err := resp.Unmarshal(&_hosts, "hosts"); err != nil {
		return nil, err
	}
	for i := 0; i < len(_hosts); i++ {
		if _hosts[i].Service == "compute" {
			host := SHost{HostName: _hosts[i].HostName, Zone: _hosts[i].Zone, zone: zone}
			ihosts = append(ihosts, &host)
		}
	}
	return ihosts, nil
}

func (zone *SZone) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	host := &SHost{zone: zone}
	_, resp, err := zone.region.Get("compute", "/os-hypervisors/"+id, "", nil)
	if err == nil {
		return host, resp.Unmarshal(&host, "hypervisor")
	}

	host.HostName = id
	host.Resource = []map[string]SResource{}
	_, resp, err = zone.region.Get("compute", "/os-hosts/"+id, "", nil)
	if err != nil {
		return nil, err
	}
	return host, resp.Unmarshal(&(host.Resource), "host")
}
