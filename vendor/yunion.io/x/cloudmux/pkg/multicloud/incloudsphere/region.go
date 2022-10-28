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

package incloudsphere

import (
	"fmt"
	"net/url"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SRegion struct {
	multicloud.SRegion
	multicloud.SRegionEipBase
	multicloud.SNoObjectStorageRegion
	multicloud.SRegionLbBase

	client *SphereClient
}

func (self *SRegion) GetClient() *SphereClient {
	return self.client
}

func (self *SRegion) GetName() string {
	return self.client.cpcfg.Name
}

func (self *SRegion) GetI18n() cloudprovider.SModelI18nTable {
	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(self.GetName()).CN(self.GetName())
	return table
}

func (self *SRegion) GetId() string {
	return fmt.Sprintf("%s/%s", CLOUD_PROVIDER_INCLOUD_SPHERE, self.client.cpcfg.Id)
}

func (self *SRegion) GetGlobalId() string {
	return self.GetId()
}

func (self *SRegion) GetProvider() string {
	return CLOUD_PROVIDER_INCLOUD_SPHERE
}

func (self *SRegion) GetCloudEnv() string {
	return ""
}

func (self *SRegion) GetGeographicInfo() cloudprovider.SGeographicInfo {
	return cloudprovider.SGeographicInfo{}
}

func (self *SRegion) GetStatus() string {
	return api.CLOUD_REGION_STATUS_INSERVER
}

func (self *SRegion) Refresh() error {
	return nil
}

func (self *SRegion) GetISecurityGroupById(secgroupId string) (cloudprovider.ICloudSecurityGroup, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SRegion) GetISecurityGroupByName(opts *cloudprovider.SecurityGroupFilterOptions) (cloudprovider.ICloudSecurityGroup, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SRegion) CreateISecurityGroup(conf *cloudprovider.SecurityGroupCreateInput) (cloudprovider.ICloudSecurityGroup, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SRegion) CreateIVpc(conf *cloudprovider.VpcCreateOptions) (cloudprovider.ICloudVpc, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SRegion) GetCapabilities() []string {
	return self.client.GetCapabilities()
}

func (self *SRegion) getVpc() *SVpc {
	return &SVpc{region: self}
}

func (self *SRegion) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
	vpc := self.getVpc()
	return []cloudprovider.ICloudVpc{vpc}, nil
}

func (self *SRegion) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	vpcs, err := self.GetIVpcs()
	if err != nil {
		return nil, err
	}
	for i := range vpcs {
		if vpcs[i].GetGlobalId() == id {
			return vpcs[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetIZoneById(id string) (cloudprovider.ICloudZone, error) {
	zone, err := self.GetZone(id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetZone(%s)", id)
	}
	return zone, nil
}

func (self *SRegion) GetIZones() ([]cloudprovider.ICloudZone, error) {
	zones, err := self.GetZones()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudZone{}
	for i := range zones {
		zones[i].region = self
		ret = append(ret, &zones[i])
	}
	return ret, nil
}

func (self *SRegion) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	vm, err := self.GetInstance(id)
	if err != nil {
		return nil, err
	}
	return vm, nil
}

func (self *SRegion) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	host, err := self.GetHost(id)
	if err != nil {
		return nil, err
	}
	zone, err := self.GetZone(host.DataCenterId)
	if err != nil {
		return nil, err
	}
	host.zone = zone
	return host, nil
}

func (self *SRegion) GetIStoragecaches() ([]cloudprovider.ICloudStoragecache, error) {
	zones, err := self.GetZones()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudStoragecache{}
	for i := range zones {
		cache := &SStoragecache{zone: &zones[i]}
		ret = append(ret, cache)
	}
	return ret, nil
}

func (self *SRegion) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	caches, err := self.GetIStoragecaches()
	if err != nil {
		return nil, err
	}
	for i := range caches {
		if caches[i].GetGlobalId() == id {
			return caches[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) _list(res string, params url.Values) (jsonutils.JSONObject, error) {
	return self.client._list(res, params)
}

func (self *SRegion) list(res string, params url.Values, retVal interface{}) error {
	return self.client.list(res, params, retVal)
}

func (self *SRegion) get(res string, params url.Values, retVal interface{}) error {
	return self.client.get(res, params, retVal)
}

func (self *SRegion) post(res string, params interface{}) (jsonutils.JSONObject, error) {
	return self.client.post(res, params)
}

func (self *SRegion) put(res string, params url.Values, body jsonutils.JSONObject, retVal interface{}) error {
	return self.client.put(res, params, body, retVal)
}

func (self *SRegion) del(res string, params url.Values, retVal interface{}) error {
	return self.client.del(res, params, retVal)
}
