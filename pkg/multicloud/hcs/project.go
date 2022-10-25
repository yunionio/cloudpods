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
	"strings"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SProject struct {
	Id          string
	Name        string
	DomainId    string
	Description bool
	Enabled     bool
	ParentId    string
	IsDomain    bool
}

func (self *SProject) GetHealthStatus() string {
	if self.Enabled {
		return api.CLOUD_PROVIDER_HEALTH_NORMAL
	}
	return api.CLOUD_PROVIDER_HEALTH_SUSPENDED
}

func (self *SHcsClient) GetProjects() ([]SProject, error) {
	if len(self.projects) > 0 {
		return self.projects, nil
	}
	resp, err := self.iamGet("v3/projects", nil)
	if err != nil {
		return nil, err
	}
	self.projects = []SProject{}
	return self.projects, resp.Unmarshal(&self.projects, "projects")
}

func (self *SHcsClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	projects, err := self.GetProjects()
	if err != nil {
		return nil, err
	}

	ret := make([]cloudprovider.SSubAccount, 0)
	for i := range projects {
		project := projects[i]
		// name 为MOS的project是华为云内部的一个特殊project。不需要同步到本地
		if strings.ToLower(project.Name) == "mos" {
			continue
		}
		s := cloudprovider.SSubAccount{
			Name:             fmt.Sprintf("%s-%s", self.cpcfg.Name, project.Name),
			Account:          fmt.Sprintf("%s/%s", self.accessKey, project.Id),
			HealthStatus:     project.GetHealthStatus(),
			DefaultProjectId: "0",
		}
		for j := range self.regions {
			region := self.regions[j]
			if strings.Contains(project.Name, region.Id) {
				s.Desc = region.Locales.ZhCN
				break
			}
		}
		ret = append(ret, s)
	}
	return ret, nil
}

func (self *SHcsClient) GetProjectById(id string) (*SProject, error) {
	projects, err := self.GetProjects()
	if err != nil {
		return nil, err
	}
	for i := range projects {
		if projects[i].Id == id {
			return &projects[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}
