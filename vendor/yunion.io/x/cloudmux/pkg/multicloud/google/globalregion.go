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

package google

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SGlobalRegion struct {
	cloudprovider.SFakeOnPremiseRegion
	multicloud.SNoObjectStorageRegion
	multicloud.SRegion
	client *SGoogleClient

	Description       string
	ID                string
	Kind              string
	Name              string
	Status            string
	SelfLink          string
	CreationTimestamp time.Time
}

func (region *SGlobalRegion) GetClient() *SGoogleClient {
	return region.client
}

func (region *SGlobalRegion) GetName() string {
	if name, ok := RegionNames[region.Name]; ok {
		return fmt.Sprintf("%s %s", CLOUD_PROVIDER_GOOGLE_CN, name)
	}
	return fmt.Sprintf("%s %s", CLOUD_PROVIDER_GOOGLE_CN, region.Name)
}

func (self *SGlobalRegion) GetI18n() cloudprovider.SModelI18nTable {
	en := fmt.Sprintf("%s %s", CLOUD_PROVIDER_GOOGLE, self.Name)
	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(self.GetName()).CN(self.GetName()).EN(en)
	return table
}

func (region *SGlobalRegion) GetId() string {
	return region.Name
}

func (region *SGlobalRegion) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", CLOUD_PROVIDER_GOOGLE, region.Name)
}

func (region *SGlobalRegion) GetGeographicInfo() cloudprovider.SGeographicInfo {
	if geoInfo, ok := LatitudeAndLongitude[region.Name]; ok {
		return geoInfo
	}
	return cloudprovider.SGeographicInfo{}
}

func (self *SGlobalRegion) GetCreatedAt() time.Time {
	return self.CreationTimestamp
}

func (region *SGlobalRegion) GetProvider() string {
	return CLOUD_PROVIDER_GOOGLE
}

func (region *SGlobalRegion) GetStatus() string {
	return api.CLOUD_REGION_STATUS_INSERVER
}

func (self *SGlobalRegion) GetCapabilities() []string {
	return []string{
		cloudprovider.CLOUD_CAPABILITY_EIP + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_LOADBALANCER + cloudprovider.READ_ONLY_SUFFIX,
	}
}

func (self *SGlobalRegion) GetILoadBalancers() ([]cloudprovider.ICloudLoadbalancer, error) {
	lbs, err := self.GetGlobalLoadbalancers()
	if err != nil {
		return nil, errors.Wrap(err, "GetGlobalLoadbalancers")
	}
	ilbs := []cloudprovider.ICloudLoadbalancer{}
	for i := range lbs {
		lbs[i].region = self
		ilbs = append(ilbs, &lbs[i])
	}
	return ilbs, nil
}

func (self *SGlobalRegion) Delete(id string) error {
	operation := &SOperation{}
	err := self.client.ecsDelete(id, operation)
	if err != nil {
		return errors.Wrap(err, "client.ecsDelete")
	}
	_, err = self.client.WaitOperation(operation.SelfLink, id, "delete")
	if err != nil {
		return errors.Wrapf(err, "region.WaitOperation(%s)", operation.SelfLink)
	}
	return nil
}

func (self *SGlobalRegion) GetProjectId() string {
	return self.client.projectId
}

func (self *SGlobalRegion) GetBySelfId(id string, retval interface{}) error {
	return self.client.GetBySelfId(id, retval)
}

func (region *SGlobalRegion) Do(id string, action string, params map[string]string, body jsonutils.JSONObject) error {
	opId, err := region.client.ecsDo(id, action, params, body)
	if err != nil {
		return err
	}
	if strings.Index(opId, "/operations/") > 0 {
		_, err = region.client.WaitOperation(opId, id, action)
		return err
	}
	return nil
}

func (region *SGlobalRegion) Get(resourceType, id string, retval interface{}) error {
	return region.client.ecsGet(resourceType, id, retval)
}

func (self *SGlobalRegion) getLoadbalancerComponents(resource string, filter string, result interface{}) error {
	url := fmt.Sprintf("%s/%s", self.Name, resource)
	params := map[string]string{}
	if len(filter) > 0 {
		params["filter"] = filter
	}

	err := self.ListAll(url, params, result)
	if err != nil {
		return errors.Wrap(err, "ListAll")
	}

	return nil
}
