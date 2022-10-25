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

package hcs

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SRegion struct {
	multicloud.HcsTags
	multicloud.SRegion
	multicloud.SRegionEipBase
	multicloud.SRegionOssBase
	multicloud.SRegionLbBase
	multicloud.SRegionVpcBase
	multicloud.SRegionZoneBase
	multicloud.SRegionSecurityGroupBase

	obsClient *obs.ObsClient // 对象存储client.请勿直接引用。

	client *SHcsClient

	Id          string
	CloudInfras []string
	Description string
	Locales     struct {
		EnUS string `json:"en-us"`
		ZhCN string `json:"zh-cn"`
	}
	ParentRegionId string
}

func (self *SRegion) GetClient() *SHcsClient {
	return self.client
}

func (self *SRegion) GetId() string {
	return self.Id
}

func (self *SRegion) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", CLOUD_PROVIDER_HCS, self.Id)
}

func (self *SRegion) GetName() string {
	return fmt.Sprintf("%s %s", CLOUD_PROVIDER_HCS_CN, self.Locales.ZhCN)
}

func (self *SRegion) GetStatus() string {
	return api.CLOUD_REGION_STATUS_INSERVER
}

func (self *SRegion) GetILoadBalancerBackendGroups() ([]cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetI18n() cloudprovider.SModelI18nTable {
	en := fmt.Sprintf("%s %s", CLOUD_PROVIDER_HCS_EN, self.Locales.EnUS)
	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(self.GetName()).CN(self.GetName()).EN(en)
	return table
}

func (self *SRegion) GetCapabilities() []string {
	return self.client.GetCapabilities()
}

func (self *SRegion) GetCloudEnv() string {
	return ""
}

func (self *SRegion) GetProvider() string {
	return CLOUD_PROVIDER_HCS
}

func (self *SRegion) GetGeographicInfo() cloudprovider.SGeographicInfo {
	return cloudprovider.SGeographicInfo{}
}

func (self *SRegion) ecsList(resource string, query url.Values, retVal interface{}) error {
	return self.client.ecsList(self.Id, resource, query, retVal)
}

func (self *SRegion) ecsGet(resource string, retVal interface{}) error {
	return self.client.ecsGet(self.Id, resource, retVal)
}

func (self *SRegion) ecsDelete(resource string) error {
	return self.client.ecsDelete(self.Id, resource)
}

func (self *SRegion) ecsCreate(resource string, body map[string]interface{}, retVal interface{}) error {
	return self.client.ecsCreate(self.Id, resource, body, retVal)
}

func (self *SRegion) ecsPerform(resource, action string, params map[string]interface{}, retVal interface{}) error {
	return self.client.ecsPerform(self.Id, resource, action, params, retVal)
}

func (region *SRegion) rdsList(resource string, query url.Values, retVal interface{}) error {
	return region.client.rdsList(region.Id, resource, query, retVal)
}

func (region *SRegion) rdsGet(resource string, retVal interface{}) error {
	return region.client.rdsGet(region.Id, resource, retVal)
}

func (region *SRegion) rdsDelete(resource string) error {
	return region.client.rdsDelete(region.Id, resource)
}

func (region *SRegion) rdsCreate(resource string, body map[string]interface{}, retVal interface{}) error {
	return region.client.rdsCreate(region.Id, resource, body, retVal)
}

func (self *SRegion) rdsPerform(resource, action string, params map[string]interface{}, retVal interface{}) error {
	return self.client.rdsPerform(self.Id, resource, action, params, retVal)
}

func (region *SRegion) rdsJobGet(resource string, query url.Values, retVal interface{}) error {
	return region.client.rdsJobGet(region.Id, resource, query, retVal)
}

func (region *SRegion) rdsDBPrivilegesDelete(resource string, params map[string]interface{}) error {
	return region.client.rdsDBPrivvilegesDelete(region.Id, resource, params)
}

func (region *SRegion) rdsDBPrivilegesGrant(resource string, params map[string]interface{}, retVal interface{}) error {
	return region.client.rdsDBPrivilegesGrant(region.Id, resource, params, retVal)
}

func (self *SRegion) perform(product, version, resource, action string, params map[string]interface{}, retVal interface{}) error {
	return self.client.perform(product, version, self.Id, resource, action, params, retVal)
}

func (self *SRegion) create(product, version, resource string, body map[string]interface{}, retVal interface{}) error {
	return self.client.create(product, version, self.Id, resource, body, retVal)
}

func (self *SRegion) delete(product, version, resource string) error {
	return self.client.delete(product, version, self.Id, resource)
}

func (self *SRegion) list(product, version, resource string, query url.Values, retVal interface{}) error {
	return self.client.list(product, version, self.Id, resource, query, retVal)
}

func (self *SRegion) get(product, version, resource string, retVal interface{}) error {
	return self.client.get(product, version, self.Id, resource, retVal)
}
func (self *SRegion) update(product, version, resource string, params map[string]interface{}) error {
	return self.client.update(product, version, self.Id, resource, params)
}

func (self *SRegion) evsList(resource string, query url.Values, retVal interface{}) error {
	return self.client.evsList(self.Id, resource, query, retVal)
}

func (self *SRegion) evsGet(resource string, retVal interface{}) error {
	return self.client.evsGet(self.Id, resource, retVal)
}

func (self *SRegion) evsDelete(resource string) error {
	return self.client.evsDelete(self.Id, resource)
}

func (self *SRegion) evsPerform(resource, action string, params map[string]interface{}) error {
	return self.client.evsPerform(self.Id, resource, action, params)
}

func (self *SRegion) evsCreate(resource string, body map[string]interface{}, retVal interface{}) error {
	return self.client.evsCreate(self.Id, resource, body, retVal)
}

func (self *SRegion) vpcList(resource string, query url.Values, retVal interface{}) error {
	return self.client.vpcList(self.Id, resource, query, retVal)
}

func (self *SRegion) vpcCreate(resource string, body map[string]interface{}, retVal interface{}) error {
	return self.client.vpcCreate(self.Id, resource, body, retVal)
}

func (self *SRegion) vpcGet(resource string, retVal interface{}) error {
	return self.client.vpcGet(self.Id, resource, retVal)
}

func (self *SRegion) vpcUpdate(resource string, params map[string]interface{}) error {
	return self.client.vpcUpdate(self.Id, resource, params)
}

func (self *SRegion) vpcDelete(resource string) error {
	return self.client.vpcDelete(self.Id, resource)
}

func (self *SRegion) imsList(resource string, query url.Values, retVal interface{}) error {
	return self.client.imsList(self.Id, resource, query, retVal)
}

func (self *SRegion) imsCreate(resource string, body map[string]interface{}, retVal interface{}) error {
	return self.client.imsCreate(self.Id, resource, body, retVal)
}

func (self *SRegion) imsGet(resource string, retVal interface{}) error {
	return self.client.imsGet(self.Id, resource, retVal)
}

func (self *SRegion) imsUpdate(resource string, params map[string]interface{}) error {
	return self.client.imsUpdate(self.Id, resource, params)
}

func (self *SRegion) imsDelete(resource string) error {
	return self.client.imsDelete(self.Id, resource)
}

func (self *SRegion) imsPerform(resource, action string, params map[string]interface{}, retVal interface{}) error {
	return self.client.imsPerform(self.Id, resource, action, params, retVal)
}

func (self *SRegion) redisList(resource string, query url.Values, retVal interface{}) error {
	return self.client.dcsList(self.Id, resource, query, retVal)
}

func (self *SRegion) redisCreate(resource string, body map[string]interface{}, retVal interface{}) error {
	return self.client.dcsCreate(self.Id, resource, body, retVal)
}

func (self *SRegion) redisGet(resource string, retVal interface{}) error {
	return self.client.dcsGet(self.Id, resource, retVal)
}

func (self *SRegion) redisUpdate(resource string, params map[string]interface{}) error {
	return self.client.dcsUpdate(self.Id, resource, params)
}

func (self *SRegion) redisDelete(resource string) error {
	return self.client.dcsDelete(self.Id, resource)
}

func (self *SRegion) redisPerform(resource, action string, params map[string]interface{}, retVal interface{}) error {
	return self.client.dcsPerform(self.Id, resource, action, params, retVal)
}

func (self *SRegion) GetICloudSku(skuId string) (cloudprovider.ICloudSku, error) {
	sku, err := self.GetchInstanceType(skuId)
	if err != nil {
		return nil, err
	}
	return sku, nil
}

func (self *SRegion) getOBSClient() (*obs.ObsClient, error) {
	if self.obsClient == nil {
		obsClient, err := self.client.getOBSClient(self.GetId())
		if err != nil {
			return nil, err
		}
		self.obsClient = obsClient
	}
	return self.obsClient, nil
}

func (self *SRegion) HeadBucket(name string) (*obs.BaseModel, error) {
	obsClient, err := self.getOBSClient()
	if err != nil {
		return nil, errors.Wrap(err, "region.getOBSClient")
	}
	return obsClient.HeadBucket(name)
}

func (self *SRegion) getOBSEndpoint() string {
	return self.client.getOBSEndpoint(self.GetId())
}

func str2StorageClass(storageClassStr string) (obs.StorageClassType, error) {
	if strings.EqualFold(storageClassStr, string(obs.StorageClassStandard)) {
		return obs.StorageClassStandard, nil
	} else if strings.EqualFold(storageClassStr, string(obs.StorageClassWarm)) {
		return obs.StorageClassWarm, nil
	} else if strings.EqualFold(storageClassStr, string(obs.StorageClassCold)) {
		return obs.StorageClassCold, nil
	} else {
		return obs.StorageClassStandard, errors.Error("unsupported storageClass")
	}
}

func (self *SRegion) GetBuckets() ([]SBucket, error) {
	buckets, err := self.client.GetBuckets()
	if err != nil {
		return nil, err
	}
	ret := []SBucket{}
	for i := range buckets {
		if buckets[i].region.GetGlobalId() == self.GetGlobalId() {
			ret = append(ret, buckets[i])
		}
	}
	return ret, nil
}

func (self *SRegion) GetIBuckets() ([]cloudprovider.ICloudBucket, error) {
	buckets, err := self.GetBuckets()
	if err != nil {
		return nil, errors.Wrap(err, "GetBuckets")
	}
	ret := make([]cloudprovider.ICloudBucket, 0)
	for i := range buckets {
		ret = append(ret, &buckets[i])
	}
	return ret, nil
}
