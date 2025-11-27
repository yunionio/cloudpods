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

package oracle

import (
	"fmt"
	"net/url"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

type SRegion struct {
	multicloud.SRegion
	multicloud.SNoObjectStorageRegion
	multicloud.SNoLbRegion
	client *SOracleClient

	IsHomeRegion bool   `json:"is-home-region"`
	RegionKey    string `json:"region-key"`
	RegionName   string `json:"region-name"`
	Status       string `json:"status"`
}

func (region *SRegion) GetId() string {
	return region.RegionName
}

func (region *SRegion) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", api.CLOUD_PROVIDER_ORACLE, region.RegionName)
}

func (region *SRegion) GetProvider() string {
	return api.CLOUD_PROVIDER_ORACLE
}

func (region *SRegion) GetCloudEnv() string {
	return api.CLOUD_PROVIDER_ORACLE
}

func (region *SRegion) GetGeographicInfo() cloudprovider.SGeographicInfo {
	geo, ok := map[string]cloudprovider.SGeographicInfo{
		"ap-singapore-1": api.RegionSingapore,
	}[region.RegionName]
	if ok {
		return geo
	}
	return cloudprovider.SGeographicInfo{}
}

func (region *SRegion) GetName() string {
	return region.RegionName
}

func (region *SRegion) GetI18n() cloudprovider.SModelI18nTable {
	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(region.GetName()).CN(region.GetName()).EN(region.RegionName)
	return table
}

func (region *SRegion) GetStatus() string {
	if region.Status != "READY" {
		return api.CLOUD_REGION_STATUS_OUTOFSERVICE
	}
	return api.CLOUD_REGION_STATUS_INSERVER
}

func (region *SRegion) GetClient() *SOracleClient {
	return region.client
}

func (region *SRegion) CreateEIP(opts *cloudprovider.SEip) (cloudprovider.ICloudEIP, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) CreateISecurityGroup(opts *cloudprovider.SecurityGroupCreateInput) (cloudprovider.ICloudSecurityGroup, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetISecurityGroupById(id string) (cloudprovider.ICloudSecurityGroup, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) CreateIVpc(opts *cloudprovider.VpcCreateOptions) (cloudprovider.ICloudVpc, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
	vpcs, err := region.GetVpcs()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudVpc{}
	for i := range vpcs {
		vpcs[i].region = region
		ret = append(ret, &vpcs[i])
	}
	return ret, nil
}

func (region *SRegion) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	vpc, err := region.GetVpc(id)
	if err != nil {
		return nil, err
	}
	return vpc, nil
}

func (region *SRegion) GetCapabilities() []string {
	return region.client.GetCapabilities()
}

func (region *SRegion) GetIEipById(id string) (cloudprovider.ICloudEIP, error) {
	eip, err := region.GetEip(id)
	if err != nil {
		return nil, err
	}
	return eip, nil
}

func (region *SRegion) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	eips, err := region.GetEips("RESERVED")
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudEIP{}
	for i := range eips {
		eips[i].region = region
		ret = append(ret, &eips[i])
	}
	return ret, nil
}

func (region *SRegion) GetIZones() ([]cloudprovider.ICloudZone, error) {
	zones, err := region.GetZones()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudZone{}
	for i := range zones {
		zones[i].region = region
		ret = append(ret, &zones[i])
	}
	return ret, nil
}

func (region *SRegion) GetIZoneById(id string) (cloudprovider.ICloudZone, error) {
	zones, err := region.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := range zones {
		if zones[i].GetGlobalId() == id {
			return zones[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (region *SRegion) list(service, resource string, query url.Values) (jsonutils.JSONObject, error) {
	return region.client.list(service, region.RegionName, resource, query)
}

func (region *SRegion) get(service, resource, id string, query url.Values) (jsonutils.JSONObject, error) {
	return region.client.get(service, region.RegionName, resource, id, query)
}

func (region *SRegion) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	caches, err := region.GetIStoragecaches()
	if err != nil {
		return nil, err
	}
	for i := range caches {
		if caches[i].GetGlobalId() == id {
			return caches[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "%s", id)
}

func (region *SRegion) GetIStoragecaches() ([]cloudprovider.ICloudStoragecache, error) {
	return []cloudprovider.ICloudStoragecache{region.GetStoragecache()}, nil
}

func (region *SRegion) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	izones, err := region.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		ihost, err := izones[i].GetIHostById(id)
		if err == nil {
			return ihost, nil
		} else if errors.Cause(err) != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (region *SRegion) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	iHosts := make([]cloudprovider.ICloudHost, 0)

	izones, err := region.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		iZoneHost, err := izones[i].GetIHosts()
		if err != nil {
			return nil, err
		}
		iHosts = append(iHosts, iZoneHost...)
	}
	return iHosts, nil
}

func (region *SRegion) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	iStores := make([]cloudprovider.ICloudStorage, 0)

	izones, err := region.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		iZoneStores, err := izones[i].GetIStorages()
		if err != nil {
			return nil, err
		}
		iStores = append(iStores, iZoneStores...)
	}
	return iStores, nil
}

func (region *SRegion) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	izones, err := region.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		istore, err := izones[i].GetIStorageById(id)
		if err == nil {
			return istore, nil
		} else if errors.Cause(err) != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (region *SRegion) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	return region.GetInstance(id)
}

func (region *SRegion) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	hosts, err := region.GetIHosts()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudVM{}
	for i := range hosts {
		vms, err := hosts[i].GetIVMs()
		if err != nil {
			return nil, err
		}
		ret = append(ret, vms...)
	}
	return ret, nil
}
