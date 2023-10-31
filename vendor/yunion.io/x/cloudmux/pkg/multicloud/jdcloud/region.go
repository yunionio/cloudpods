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

package jdcloud

import (
	"fmt"

	"github.com/jdcloud-api/jdcloud-sdk-go/core"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

const (
	CLOUD_PROVIDER_JDCLOUD    = api.CLOUD_PROVIDER_JDCLOUD
	CLOUD_PROVIDER_JDCLOUD_CN = "京东云"
	CLOUD_PROVIDER_JDCLOUD_EN = "JDcloud"

	JDCLOUD_DEFAULT_REGION = "cn-north-1"
)

// https://docs.jdcloud.com/cn/common-declaration/api/introduction
var regionList = map[string]string{
	"cn-north-1": "华北-北京",
	"cn-east-1":  "华东-宿迁",
	"cn-east-2":  "华东-上海",
	"cn-south-1": "华南-广州",
}

type SRegion struct {
	cloudprovider.SFakeOnPremiseRegion
	multicloud.SRegion
	multicloud.SNoObjectStorageRegion

	client *SJDCloudClient

	cpcfg        cloudprovider.ProviderConfig
	storageCache *SStoragecache

	ID   string `json:"id"`
	Name string `json:"Name"`

	izones []cloudprovider.ICloudZone
	ivpcs  []cloudprovider.ICloudVpc
}

func (self *SRegion) GetClient() *SJDCloudClient {
	return self.client
}

func (self *SRegion) getCredential() *core.Credential {
	return self.client.getCredential()
}

func (r *SRegion) GetId() string {
	return r.ID
}

func (r *SRegion) GetName() string {
	return r.Name
}

func (r *SRegion) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", api.CLOUD_ACCESS_ENV_JDCLOUD_CHINA, r.GetId())
}

func (r *SRegion) GetStatus() string {
	return api.CLOUD_REGION_STATUS_INSERVER
}

func (r *SRegion) Refresh() error {
	return nil
}

func (r *SRegion) IsEmulated() bool {
	return false
}

func (r *SRegion) GetI18n() cloudprovider.SModelI18nTable {
	en := fmt.Sprintf("%s %s", CLOUD_PROVIDER_JDCLOUD_EN, r.Name)
	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(r.GetName()).CN(r.GetName()).EN(en)
	return table
}

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
	zoneIdens := ZonesInRegion[r.ID]
	r.izones = make([]cloudprovider.ICloudZone, 0, len(zoneIdens))
	for _, iden := range zoneIdens {
		zone := SZone{
			region: r,
			ID:     iden.Id,
			Name:   iden.Name,
		}
		r.izones = append(r.izones, &zone)
	}
	return nil
}

func (r *SRegion) fetchVpcs() error {
	vpcs := make([]SVpc, 0)
	n := 1
	for {
		part, total, err := r.GetVpcs(n, 100)
		if err != nil {
			return err
		}
		vpcs = append(vpcs, part...)
		if len(vpcs) >= total {
			break
		}
		n++
	}
	r.ivpcs = make([]cloudprovider.ICloudVpc, len(vpcs))
	for i := 0; i < len(vpcs); i++ {
		r.ivpcs[i] = &vpcs[i]
	}
	return nil
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
	return nil, nil
}

func (r *SRegion) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	vpc, err := r.GetVpcById(id)
	if err != nil {
		return nil, err
	}
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

func (r *SRegion) getIZoneByRealId(id string) (cloudprovider.ICloudZone, error) {
	izones, err := r.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		if izones[i].GetId() == id {
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
	izone, err := r.getIZoneByRealId(vm.Az)
	if err != nil {
		return nil, err
	}
	zone := izone.(*SZone)
	vm.host = &SHost{
		zone: zone,
	}
	return vm, nil
}

func (r *SRegion) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	return r.GetDiskById(id)
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

func (s *SRegion) getStoragecache() *SStoragecache {
	if s.storageCache == nil {
		s.storageCache = &SStoragecache{region: s}
	}
	return s.storageCache
}

func (r *SRegion) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	storageCache := r.getStoragecache()
	if storageCache.GetGlobalId() == id {
		return storageCache, nil
	}
	return nil, cloudprovider.ErrNotFound
}

func (r *SRegion) GetProvider() string {
	return api.CLOUD_PROVIDER_JDCLOUD
}

func (r *SRegion) GetCapabilities() []string {
	return []string{
		cloudprovider.CLOUD_CAPABILITY_COMPUTE + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_NETWORK + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_SECURITY_GROUP + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_EIP + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_RDS + cloudprovider.READ_ONLY_SUFFIX,
	}
}

func (region *SRegion) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	vms := make([]SInstance, 0)
	n := 1
	for {
		parts, total, err := region.GetInstances("", nil, n, 100)
		if err != nil {
			return nil, err
		}
		vms = append(vms, parts...)
		if len(vms) >= total {
			break
		}
		n++
	}
	ivms := make([]cloudprovider.ICloudVM, len(vms))
	for i := range vms {
		ivms[i] = &vms[i]
	}
	return ivms, nil
}
