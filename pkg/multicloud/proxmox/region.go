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

package proxmox

import (
	"fmt"
	"net/url"

	"yunion.io/x/jsonutils"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SRegion struct {
	multicloud.SRegion
	multicloud.SRegionEipBase
	multicloud.SNoObjectStorageRegion
	multicloud.SRegionLbBase

	client *SProxmoxClient
}

func (self *SRegion) GetClient() *SProxmoxClient {
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
	return fmt.Sprintf("%s/%s", CLOUD_PROVIDER_PROXMOX, self.client.cpcfg.Id)
}

func (self *SRegion) GetGlobalId() string {
	return self.GetId()
}

func (self *SRegion) GetProvider() string {
	return CLOUD_PROVIDER_PROXMOX
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

//func (self *SRegion) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
//vpc := self.getVpc()
//return []cloudprovider.ICloudVpc{vpc}, nil
//}

func (self *SRegion) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SRegion) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SRegion) GetIZoneById(id string) (cloudprovider.ICloudZone, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SRegion) GetIZones() ([]cloudprovider.ICloudZone, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SRegion) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SRegion) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SRegion) GetIStoragecaches() ([]cloudprovider.ICloudStoragecache, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SRegion) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	return nil, cloudprovider.ErrNotSupported
}

// func (self *SRegion) _list(res string, params url.Values) (jsonutils.JSONObject, error) {
// 	return self.client._list(res, params)
// }

// func (self *SRegion) list(res string, params url.Values, retVal interface{}) error {
// 	return self.client.list(res, params, retVal)
// }

func (self *SRegion) get(res string, params url.Values, retVal interface{}) error {
	return self.client.get(res, params, retVal)
}

func (self *SRegion) post(res string, params interface{}) (jsonutils.JSONObject, error) {
	return self.client.post(res, params)
}

// func (self *SRegion) put(res string, params url.Values, body jsonutils.JSONObject, retVal interface{}) error {
// 	return self.client.put(res, params, body, retVal)
// }

func (self *SRegion) del(res string, params url.Values, retVal interface{}) error {
	return self.client.del(res, params, retVal)
}
