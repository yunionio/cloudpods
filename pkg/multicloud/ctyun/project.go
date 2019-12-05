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

package ctyun

import (
	"strings"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

// GET http://ctyun-api-url/apiproxy/v3/ondemand/queryProjectIds
type SProject struct {
	Enabled     bool   `json:"enabled"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ID          string `json:"id"`
}

func (self *SProject) GetRegionID() string {
	return strings.Split(self.Name, "_")[1]
}

func (self *SProject) GetHealthStatus() string {
	if self.Enabled {
		return api.CLOUD_PROVIDER_HEALTH_NORMAL
	}

	return api.CLOUD_PROVIDER_HEALTH_SUSPENDED
}

func (self *SCtyunClient) FetchProjects() ([]SProject, error) {
	client, err := NewSCtyunClient("", "", "", self.accessKey, self.secret, self.debug)
	if err != nil {
		return nil, errors.Wrap(err, "CtyunClient.FetchProjects")
	}
	projects := make([]SProject, 0)
	resp, err := client.DoGet("/apiproxy/v3/ondemand/queryProjectIds", map[string]string{})
	if err != nil {
		return nil, errors.Wrap(err, "CtyunClient.FetchProjects.DoGet")
	}

	err = resp.Unmarshal(&projects, "returnObj")
	if err != nil {
		return nil, errors.Wrap(err, "CtyunClient.FetchProjects.Unmarshal")
	}

	return projects, err
}

func (self *SRegion) FetchProjects() ([]SProject, error) {
	return self.client.FetchProjects()
}
