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

package huawei

import (
	"fmt"
	"strings"

	"yunion.io/x/pkg/errors"
)

type SProject struct {
	client *SHuaweiClient

	IsDomain    bool
	Description string
	Enabled     bool
	Id          string
	ParentId    string
	DomainId    string
	Name        string
}

func (self *SProject) GetRegionId() string {
	return strings.Split(self.Name, "_")[0]
}

func (self *SHuaweiClient) GetProjects() ([]SProject, error) {
	if len(self.projects) > 0 {
		ret := []SProject{}
		for _, project := range self.projects {
			ret = append(ret, project)
		}
		return ret, nil
	}
	self.projects = map[string]SProject{}
	projects := []SProject{}
	resp, err := self.list(SERVICE_IAM_V3, "", "auth/projects", nil)
	if err != nil {
		return nil, errors.Wrapf(err, "list projects")
	}
	err = resp.Unmarshal(&projects, "projects")
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	for _, project := range projects {
		self.projects[project.Name] = project
	}
	return projects, nil
}

// obs 权限必须赋予到mos project之上
func (self *SHuaweiClient) GetMosProjectId() string {
	projects, err := self.GetProjects()
	if err != nil {
		return ""
	}
	for i := range projects {
		if strings.ToLower(projects[i].Name) == "mos" {
			return projects[i].Id
		}
	}
	return ""
}

func (self *SHuaweiClient) GetProjectById(projectId string) (SProject, error) {
	projects, err := self.GetProjects()
	if err != nil {
		return SProject{}, err
	}

	for _, project := range projects {
		if project.Id == projectId {
			return project, nil
		}
	}
	return SProject{}, fmt.Errorf("project %s not found", projectId)
}
