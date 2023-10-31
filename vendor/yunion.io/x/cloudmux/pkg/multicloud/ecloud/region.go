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

package ecloud

import (
	"context"
	"fmt"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

var regionList = map[string]string{
	"guangzhou-2": "华南-广州2",
	"beijing-1":   "华北-北京1",
	"hunan-1":     "华中-长沙1",
	"wuxi-1":      "华东-苏州",
	"dongguan-1":  "华南-广州3",
	"yaan-1":      "西南-成都",
	"zhengzhou-1": "华中-郑州",
	"beijing-2":   "华北-北京3",
	"zhuzhou-1":   "华中-长沙2",
	"jinan-1":     "华东-济南",
	"xian-1":      "西北-西安",
	"shanghai-1":  "华东-上海1",
	"chongqing-1": "西南-重庆",
	"ningbo-1":    "华东-杭州",
	"tianjin-1":   "天津-天津",
	"jilin-1":     "吉林-长春",
	"hubei-1":     "湖北-襄阳",
	"jiangxi-1":   "江西-南昌",
	"gansu-1":     "甘肃-兰州",
	"shanxi-1":    "山西-太原",
	"liaoning-1":  "辽宁-沈阳",
	"yunnan-2":    "云南-昆明2",
	"hebei-1":     "河北-石家庄",
	"fujian-1":    "福建-厦门",
	"guangxi-1":   "广西-南宁",
	"anhui-1":     "安徽-淮南",
	"huhehaote-1": "华北-呼和浩特",
	"guiyang-1":   "西南-贵阳",
}

type SRegion struct {
	cloudprovider.SFakeOnPremiseRegion
	multicloud.SRegion
	multicloud.SNoObjectStorageRegion

	client       *SEcloudClient
	storageCache *SStoragecache

	ID   string `json:"id"`
	Name string `json:"Name"`

	izones []cloudprovider.ICloudZone
	ivpcs  []cloudprovider.ICloudVpc
}

func (r *SRegion) GetId() string {
	return r.ID
}

func (r *SRegion) GetName() string {
	return r.Name
}

func (r *SRegion) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", r.client.GetAccessEnv(), r.ID)
}

func (r *SRegion) GetStatus() string {
	return api.CLOUD_REGION_STATUS_INSERVER
}

func (r *SRegion) Refresh() error {
	// err := r.fetchZones()
	// if err != nil {
	// 	return err
	// }
	// return r.fetchVpcs()
	return nil
}

func (r *SRegion) IsEmulated() bool {
	return false
}

func (r *SRegion) GetI18n() cloudprovider.SModelI18nTable {
	en := fmt.Sprintf("%s %s", CLOUD_PROVIDER_ECLOUD_EN, r.Name)
	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(r.GetName()).CN(r.GetName()).EN(en)
	return table
}

// GetLatitude() float32
// GetLongitude() float32
func (r *SRegion) GetGeographicInfo() cloudprovider.SGeographicInfo {
	if info, ok := LatitudeAndLongitude[r.ID]; ok {
		return info
	}
	return cloudprovider.SGeographicInfo{}
}

func (r *SRegion) GetIZones() ([]cloudprovider.ICloudZone, error) {
	if r.izones == nil {
		err := r.fetchZones()
		if err != nil {
			return nil, err
		}
	}
	return r.izones, nil
}

func (r *SRegion) fetchZones() error {
	request := NewNovaRequest(NewApiRequest(r.ID, "/api/v2/region",
		map[string]string{"component": "NOVA"}, nil))
	zones := make([]SZone, 0)
	err := r.client.doList(context.Background(), request, &zones)
	if err != nil {
		return err
	}
	izones := make([]cloudprovider.ICloudZone, len(zones))
	for i := range zones {
		zones[i].region = r
		zones[i].host = &SHost{
			zone: &zones[i],
		}
		izones[i] = &zones[i]
	}
	r.izones = izones
	return nil
}

func (r *SRegion) fetchVpcs() error {
	vpcs, err := r.getVpcs()
	if err != nil {
		return err
	}
	ivpcs := make([]cloudprovider.ICloudVpc, len(vpcs))
	for i := range vpcs {
		ivpcs[i] = &vpcs[i]
	}
	r.ivpcs = ivpcs
	return nil
}

func (r *SRegion) getVpcs() ([]SVpc, error) {
	request := NewConsoleRequest(r.ID, "/api/v2/netcenter/vpc", nil, nil)
	vpcs := make([]SVpc, 0)
	err := r.client.doList(context.Background(), request, &vpcs)
	if err != nil {
		return nil, err
	}
	for i := range vpcs {
		vpcs[i].region = r
	}
	return vpcs, err
}

func (r *SRegion) getVpcById(id string) (*SVpc, error) {
	request := NewConsoleRequest(r.ID, fmt.Sprintf("/api/v2/netcenter/vpc/%s", id), nil, nil)
	vpc := SVpc{}
	err := r.client.doGet(context.Background(), request, &vpc)
	if err != nil {
		return nil, err
	}
	vpc.region = r
	return &vpc, err
}

func (r *SRegion) getVpcByRouterId(id string) (*SVpc, error) {
	request := NewConsoleRequest(r.ID, fmt.Sprintf("/api/v2/netcenter/vpc/router/%s", id), nil, nil)
	vpc := SVpc{}
	err := r.client.doGet(context.Background(), request, &vpc)
	if err != nil {
		return nil, err
	}
	vpc.region = r
	return &vpc, err
}

func (r *SRegion) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
	if r.ivpcs == nil {
		err := r.fetchVpcs()
		if err != nil {
			return nil, err
		}
	}
	return r.ivpcs, nil
}

func (r *SRegion) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (r *SRegion) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	vpc, err := r.getVpcById(id)
	if err != nil {
		return nil, err
	}
	vpc.region = r
	return vpc, nil
}

func (r *SRegion) GetIZoneById(id string) (cloudprovider.ICloudZone, error) {
	izones, err := r.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		if izones[i].GetGlobalId() == id {
			return izones[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (r *SRegion) GetIEipById(id string) (cloudprovider.ICloudEIP, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (r *SRegion) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	vm, err := r.GetInstanceById(id)
	if err != nil {
		return nil, err
	}
	zone, err := r.FindZone(vm.Region)
	if err != nil {
		return nil, err
	}
	vm.host = &SHost{
		zone: zone,
	}
	return vm, nil
}

func (r *SRegion) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	return r.GetDisk(id)
}

func (r *SRegion) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	izones, err := r.GetIZones()
	if err != nil {
		return nil, err
	}
	iHosts := make([]cloudprovider.ICloudHost, 0, len(izones))
	for i := range izones {
		hosts, err := izones[i].GetIHosts()
		if err != nil {
			return nil, err
		}
		iHosts = append(iHosts, hosts...)
	}
	return iHosts, nil
}

func (r *SRegion) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	hosts, err := r.GetIHosts()
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

func (r *SRegion) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	iStores := make([]cloudprovider.ICloudStorage, 0)

	izones, err := r.GetIZones()
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

func (r *SRegion) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	istores, err := r.GetIStorages()
	if err != nil {
		return nil, err
	}
	for i := range istores {
		if istores[i].GetGlobalId() == id {
			return istores[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (r *SRegion) GetIStoragecaches() ([]cloudprovider.ICloudStoragecache, error) {
	sc := r.getStoragecache()
	return []cloudprovider.ICloudStoragecache{sc}, nil
}

func (r *SRegion) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	storageCache := r.getStoragecache()
	if storageCache.GetGlobalId() == id {
		return storageCache, nil
	}
	return nil, cloudprovider.ErrNotFound
}

func (r *SRegion) GetProvider() string {
	return api.CLOUD_PROVIDER_ECLOUD
}

func (r *SRegion) GetCapabilities() []string {
	return r.client.GetCapabilities()
}

func (r *SRegion) GetClient() *SEcloudClient {
	return r.client
}

func (r *SRegion) FindZone(zoneRegion string) (*SZone, error) {
	izones, err := r.GetIZones()
	if err != nil {
		return nil, errors.Wrap(err, "unable to GetZones")
	}
	findZone := func(zoneRegion string) *SZone {
		for i := range izones {
			zone := izones[i].(*SZone)
			if zone.Region == zoneRegion {
				return zone
			}
		}
		return nil
	}
	return findZone(zoneRegion), nil
}

func (region *SRegion) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	vms, err := region.GetInstances(region.ID)
	if err != nil {
		return nil, errors.Wrap(err, "GetVMs")
	}
	ivms := make([]cloudprovider.ICloudVM, len(vms))
	for i := range vms {
		ivms[i] = &vms[i]
	}
	return ivms, nil
}
