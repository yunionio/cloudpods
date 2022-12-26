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

package cloudpods

import (
	"fmt"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

type SRegion struct {
	multicloud.SRegion
	multicloud.SRegionOssBase
	multicloud.SRegionLbBase
	multicloud.SRegionSecurityGroupBase
	cli *SCloudpodsClient

	api.CloudregionDetails
}

func (self *SRegion) GetClient() *SCloudpodsClient {
	return self.cli
}

func (self *SRegion) GetId() string {
	return self.Id
}

func (self *SRegion) GetName() string {
	return fmt.Sprintf("%s-%s", self.cli.cpcfg.Name, self.Name)
}

func (self *SRegion) GetGlobalId() string {
	return fmt.Sprintf("%s/%s/%s", CLOUD_PROVIDER_CLOUDPODS, self.cli.cpcfg.Id, self.Name)
}

func (self *SRegion) GetStatus() string {
	return self.Status
}

func (self *SRegion) GetI18n() cloudprovider.SModelI18nTable {
	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(self.GetName()).CN(self.GetName())
	return table
}

func (self *SRegion) GetCapabilities() []string {
	return self.cli.GetCapabilities()
}

func (self *SRegion) GetCloudEnv() string {
	return api.CLOUD_ENV_PRIVATE_CLOUD
}

func (self *SRegion) GetProvider() string {
	return api.CLOUD_PROVIDER_CLOUDPODS
}

func (self *SRegion) GetGeographicInfo() cloudprovider.SGeographicInfo {
	return cloudprovider.SGeographicInfo{
		Latitude:    self.Latitude,
		Longitude:   self.Longitude,
		City:        self.City,
		CountryCode: self.CountryCode,
	}
}

func (self *SRegion) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	ins, err := self.GetInstance(id)
	if err != nil {
		return nil, err
	}
	return ins, nil
}

func (self *SRegion) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	disk, err := self.GetDisk(id)
	if err != nil {
		return nil, err
	}
	return disk, nil
}

func (self *SRegion) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	host, err := self.GetHost(id)
	if err != nil {
		return nil, err
	}
	if host.CloudregionId != self.Id {
		return nil, cloudprovider.ErrNotFound
	}
	zone, err := self.GetZone(host.ZoneId)
	if err != nil {
		return nil, err
	}
	host.zone = zone
	return host, nil
}

func (self *SRegion) create(manager ModelManager, params interface{}, retVal interface{}) error {
	return self.cli.create(manager, params, retVal)
}

func (self *SRegion) perform(manager ModelManager, id, action string, params interface{}) (jsonutils.JSONObject, error) {
	return self.cli.perform(manager, id, action, params)
}

func (self *SRegion) list(manager ModelManager, params map[string]interface{}, retVal interface{}) error {
	if params == nil {
		params = map[string]interface{}{}
	}
	params["cloudregion_id"] = self.Id
	return self.cli.list(manager, params, retVal)
}
