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

package ksyun

import (
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SProject struct {
	multicloud.SProjectBase
	client *SKsyunClient

	ProjectId   string
	AccountId   string
	ProjectName string
	ProjectDesc string
	Status      string
	Krn         string
}

func (self *SProject) GetGlobalId() string {
	return self.ProjectId
}

func (self *SProject) GetId() string {
	return self.ProjectId
}

func (self *SProject) GetName() string {
	return self.ProjectName
}

func (self *SProject) GetStatus() string {
	return api.EXTERNAL_PROJECT_STATUS_AVAILABLE
}

func (self *SKsyunClient) GetProjects() ([]SProject, error) {
	resp, err := self.iamRequest("", "GetAccountAllProjectList", nil)
	if err != nil {
		return nil, err
	}
	ret := []SProject{}
	err = resp.Unmarshal(&ret, "ProjectList")
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SKsyunClient) GetSplitProjectIds() ([][]string, error) {
	projects, err := self.GetProjects()
	if err != nil {
		return nil, err
	}
	length := 100
	var result [][]string
	for i := 0; i < len(projects); i += length {
		end := i + length
		if end > len(projects) {
			end = len(projects)
		}
		part := []string{}
		for j := i; j < end; j++ {
			part = append(part, projects[j].ProjectId)
		}
		result = append(result, part)
	}
	return result, nil
}

func (self *SKsyunClient) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	projects, err := self.GetProjects()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudProject{}
	for i := range projects {
		projects[i].client = self
		ret = append(ret, &projects[i])
	}
	return ret, nil
}
