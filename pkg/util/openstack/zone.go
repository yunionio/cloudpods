package openstack

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/version"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type ZoneState struct {
	Available bool
}

type SZone struct {
	region *SRegion

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

func (zone *SZone) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	// if self.istorages == nil {
	// 	self.fetchStorages()
	// }
	// return self.istorages, nil
	return nil, cloudprovider.ErrNotImplemented
}

func (zone *SZone) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (zone *SZone) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	_, maxVersion, err := zone.region.GetVersion("compute")
	if err != nil {
		return nil, err
	}
	if version.GT(maxVersion, "2.43") {
		return zone.GetIHostsV3()
	}
	return zone.GetIHostsV2()
}

func (zone *SZone) GetIHostsV2() ([]cloudprovider.ICloudHost, error) {
	_, resp, err := zone.region.Get("/os-hosts", "", nil)
	if err != nil {
		return nil, err
	}
	ihosts := []cloudprovider.ICloudHost{}
	hosts := []SHostV2{}
	if err := resp.Unmarshal(&hosts, "hosts"); err != nil {
		return nil, err
	}
	for i := 0; i < len(hosts); i++ {
		if hosts[i].Zone == zone.ZoneName {
			hosts[i].zone = zone
			ihosts = append(ihosts, &hosts[i])
		}
	}
	return ihosts, nil
}

func (zone *SZone) GetIHostsV3() ([]cloudprovider.ICloudHost, error) {
	_, resp, err := zone.region.Get("/os-hypervisors/detail", "2.28", nil)
	if err != nil {
		return nil, err
	}
	ihosts := []cloudprovider.ICloudHost{}
	hosts := []SHostV3{}
	if err := resp.Unmarshal(&hosts, "hypervisors"); err != nil {
		return nil, err
	}
	for i := 0; i < len(hosts); i++ {
		hosts[i].zone = zone
		ihosts = append(ihosts, &hosts[i])
	}
	return ihosts, nil
}

func (zone *SZone) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	_, maxVersion, err := zone.region.GetVersion("compute")
	if err != nil {
		return nil, err
	}
	if version.GT(maxVersion, "2.43") {
		return zone.GetIHostByIdV3(id)
	}
	return zone.GetIHostByIdV2(id)
}

func (zone *SZone) GetIHostByIdV3(id string) (cloudprovider.ICloudHost, error) {

	return nil, cloudprovider.ErrNotImplemented
}

func (zone *SZone) GetIHostByIdV2(id string) (*SHostV2, error) {
	_, resp, err := zone.region.Get("/os-hosts/"+id, "", nil)
	if err != nil {
		return nil, err
	}
	host := SHostV2{zone: zone, HostName: id, Resource: []map[string]SResource{}}
	if err := resp.Unmarshal(&host.Resource, "host"); err != nil {
		return nil, err
	}
	host.zone = zone
	return &host, nil
}
