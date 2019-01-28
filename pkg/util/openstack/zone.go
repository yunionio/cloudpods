package openstack

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/version"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type ZoneState struct {
	Available bool
}

const (
	HYPERVISORS_VERSION = "2.28"
)

type SZone struct {
	region *SRegion

	iwires    []cloudprovider.ICloudWire
	istorages []cloudprovider.ICloudStorage

	ZoneName  string
	ZoneState ZoneState
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
	return false
}

func (zone *SZone) GetStatus() string {
	if zone.ZoneState.Available {
		return models.ZONE_ENABLE
	}
	return models.ZONE_SOLDOUT
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
		if storage.Name == category {
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
		_, resp, err := zone.region.Get(service, "/types", "", nil)
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
	// 2.28 Hypervisor CPU字段是字符串，会解析失败
	ihosts := []cloudprovider.ICloudHost{}
	hosts := []SHost{}
	_, maxVersion, err := zone.region.GetVersion("compute")
	if err == nil && version.GE(maxVersion, HYPERVISORS_VERSION) {
		_, resp, err := zone.region.Get("compute", "/os-hypervisors/detail", maxVersion, nil)
		if err != nil {
			return nil, err
		}
		if err := resp.Unmarshal(&hosts, "hypervisors"); err != nil {
			return nil, err
		}
		for i := 0; i < len(hosts); i++ {
			hosts[i].zone = zone
			ihosts = append(ihosts, &hosts[i])
		}
		return ihosts, nil
	}

	_, resp, err := zone.region.Get("compute", "/os-hosts", "", nil)
	if err != nil {
		return nil, err
	}

	_hosts := []SOsHost{}

	if err := resp.Unmarshal(&_hosts, "hosts"); err != nil {
		return nil, err
	}
	for i := 0; i < len(_hosts); i++ {
		if _hosts[i].Zone == zone.ZoneName && _hosts[i].Service == "compute" {
			host := SHost{HostName: _hosts[i].HostName, Zone: _hosts[i].Zone, zone: zone}
			ihosts = append(ihosts, &host)
		}
	}
	return ihosts, nil
}

func (zone *SZone) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	_, maxVersion, err := zone.region.GetVersion("compute")
	host := &SHost{zone: zone}
	if err == nil && version.GE(maxVersion, HYPERVISORS_VERSION) {
		_, resp, err := zone.region.Get("compute", "/os-hypervisors/"+id, maxVersion, nil)
		if err != nil {
			return nil, err
		}
		return host, resp.Unmarshal(&host, "hypervisor")
	}
	host.HostName = id
	host.Resource = []map[string]SResource{}
	_, resp, err := zone.region.Get("compute", "/os-hosts/"+id, maxVersion, nil)
	if err != nil {
		return nil, err
	}
	return host, resp.Unmarshal(&(host.Resource), "host")
}
