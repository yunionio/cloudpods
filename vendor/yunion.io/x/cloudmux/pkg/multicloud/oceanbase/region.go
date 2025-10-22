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

package oceanbase

import (
	"fmt"
	"net/url"

	api "yunion.io/x/cloudmux/pkg/apis/compute"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
)

type SRegion struct {
	multicloud.SRegion
	multicloud.SNoObjectStorageRegion
	multicloud.SNoLbRegion
	multicloud.SNoEipRegion
	multicloud.SNoSecurityGroupRegion
	multicloud.SNoVpcRegion
	multicloud.SRegionZoneBase

	client *SOceanBaseClient
}

func (region *SRegion) GetId() string {
	return fmt.Sprintf("%s/Default", api.CLOUD_PROVIDER_OCEANBASE)
}

func (region *SRegion) GetName() string {
	return OB_DEFAULT_REGION_NAME
}

func (region *SRegion) GetGlobalId() string {
	return region.GetId()
}

func (region *SRegion) GetProvider() string {
	return api.CLOUD_PROVIDER_OCEANBASE
}

func (region *SRegion) GetClient() *SOceanBaseClient {
	return region.client
}

func (region *SRegion) GetStatus() string {
	return api.CLOUD_REGION_STATUS_INSERVER
}

func (region *SRegion) list(resource string, params url.Values) (jsonutils.JSONObject, error) {
	return region.client.list(resource, params)
}

func (region *SRegion) delete(resource string, body map[string]interface{}) (jsonutils.JSONObject, error) {
	return region.client.delete(resource, body)
}

func (region *SRegion) put(resource string, body map[string]interface{}) (jsonutils.JSONObject, error) {
	return region.client.put(resource, body)
}

func (region *SRegion) GetCloudEnv() string {
	return api.CLOUD_PROVIDER_OCEANBASE
}

func (region *SRegion) GetI18n() cloudprovider.SModelI18nTable {
	table := cloudprovider.SModelI18nTable{}
	return table
}

func (region *SRegion) GetGeographicInfo() cloudprovider.SGeographicInfo {
	return cloudprovider.SGeographicInfo{}
}

func (region *SRegion) GetCapabilities() []string {
	return region.client.GetCapabilities()
}

func (region *SRegion) GetIDBInstances() ([]cloudprovider.ICloudDBInstance, error) {
	dbinstances, err := region.GetDBInstances()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudDBInstance{}
	for i := range dbinstances {
		dbinstances[i].region = region
		ret = append(ret, &dbinstances[i])
	}
	return ret, nil
}

func (region *SRegion) GetIDBInstanceById(id string) (cloudprovider.ICloudDBInstance, error) {
	ret, err := region.GetDBInstance(id)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
