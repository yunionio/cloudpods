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

package azure

import (
	"fmt"
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SAppEnvironment struct {
	multicloud.SResourceBase
	AzureTags
	region *SRegion

	Id   string
	Name string
}

func (ae *SAppEnvironment) GetId() string {
	return ae.Id
}

func (ae *SAppEnvironment) GetGlobalId() string {
	return strings.ToLower(ae.Id)
}

func (ae *SAppEnvironment) GetName() string {
	return ae.Name
}

func (ae *SAppEnvironment) GetProjectId() string {
	return getResourceGroup(ae.Id)
}

func (ae *SAppEnvironment) GetStatus() string {
	return "ready"
}

func (a *SAppSite) GetEnvironments() ([]cloudprovider.ICloudAppEnvironment, error) {
	sites, err := a.region.GetSlots(a.Id)
	if err != nil {
		return nil, err
	}
	aes := []cloudprovider.ICloudAppEnvironment{}
	for i := range sites {
		sites[i].region = a.region
		aes = append(aes, &sites[i])
	}
	return aes, nil
}

func (self *SRegion) GetSlots(appId string) ([]SAppEnvironment, error) {
	resource := fmt.Sprintf("%s/slots", appId)
	resp, err := self.list_v2(resource, "2023-12-01", nil)
	if err != nil {
		return nil, err
	}
	ret := []SAppEnvironment{}
	err = resp.Unmarshal(&ret, "value")
	if err != nil {
		return nil, err
	}
	return ret, nil
}
