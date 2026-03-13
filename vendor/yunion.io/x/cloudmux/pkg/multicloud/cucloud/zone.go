package cucloud

import (
	"fmt"
	"net/url"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SZone struct {
	multicloud.SResourceBase
	multicloud.STagBase

	region *SRegion

	Id         string
	ZoneId     string
	ZoneCode   string
	ZoneName   string
	Status     string
	RegionCode string
}

func (zone *SZone) GetId() string {
	return zone.ZoneCode
}

func (zone *SZone) GetName() string {
	return fmt.Sprintf("%s %s", zone.region.GetName(), zone.ZoneName)
}

func (zone *SZone) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", zone.region.GetGlobalId(), zone.ZoneCode)
}

func (zone *SZone) GetI18n() cloudprovider.SModelI18nTable {
	table := cloudprovider.SModelI18nTable{}
	return table
}

func (zone *SZone) IsEmulated() bool {
	return false
}

func (zone *SZone) GetStatus() string {
	if zone.Status == "up" {
		return api.ZONE_ENABLE
	}
	return api.ZONE_DISABLE
}

func (zone *SZone) Refresh() error {
	return nil
}

func (zone *SZone) GetIRegion() cloudprovider.ICloudRegion {
	return zone.region
}

func (zone *SZone) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	return []cloudprovider.ICloudStorage{}, nil
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
	return nil, cloudprovider.ErrNotFound
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
	vpcs, err := zone.region.GetVpcs("")
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudWire{}
	for i := range vpcs {
		vpcs[i].region = zone.region
		wire := &SWire{zone: zone, vpc: &vpcs[i]}
		ret = append(ret, wire)
	}
	return ret, nil
}

func (region *SRegion) GetZones() ([]SZone, error) {
	params := url.Values{}
	params.Set("cloudRegionCode", region.CloudRegionCode)
	resp, err := region.list("/instance/v1/product/zones", params)
	if err != nil {
		return nil, err
	}
	ret := []SZone{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, err
	}
	result := []SZone{}
	for i := range ret {
		if ret[i].RegionCode == region.CloudRegionCode {
			result = append(result, ret[i])
		}
	}
	return result, nil
}
