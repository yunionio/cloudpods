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

package qcloud

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SProject struct {
	multicloud.SProjectBase
	QcloudTags
	client *SQcloudClient

	ProjectName string    `json:"projectName"`
	ProjectId   string    `json:"projectId"`
	CreateTime  time.Time `json:"createTime"`
	CreateorUin int       `json:"creatorUin"`
	ProjectInfo string    `json:"projectInfo"`
}

func (p *SProject) GetId() string {
	var pId string
	pos := strings.Index(p.ProjectId, ".")
	if pos >= 0 {
		pId = p.ProjectId[:pos]
	} else {
		pId = p.ProjectId
	}
	return pId
}

func (p *SProject) GetGlobalId() string {
	return p.GetId()
}

func (p *SProject) GetName() string {
	return p.ProjectName
}

func (p *SProject) GetStatus() string {
	return api.EXTERNAL_PROJECT_STATUS_AVAILABLE
}

func (p *SProject) IsEmulated() bool {
	return false
}

func (p *SProject) Refresh() error {
	return nil
}

func (client *SQcloudClient) CreateIProject(name string) (cloudprovider.ICloudProject, error) {
	return client.CreateProject(name, "")
}

func (client *SQcloudClient) GetProjects() ([]SProject, error) {
	projects := []SProject{}
	params := map[string]string{"allList": "1"}
	resp, err := client.accountRequestRequest("DescribeProject", params)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeProject")
	}
	err = resp.Unmarshal(&projects)
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return projects, nil
}

func (client *SQcloudClient) CreateProject(name, desc string) (*SProject, error) {
	params := map[string]string{
		"projectName": name,
	}
	if len(desc) > 0 {
		params["projectDesc"] = desc
	}
	body, err := client.accountRequestRequest("AddProject", params)
	if err != nil {
		return nil, errors.Wrap(err, "AddProject")
	}
	projectId, _ := body.GetString("projectId")
	if len(projectId) == 0 {
		return nil, fmt.Errorf("empty project reture")
	}
	projects, err := client.GetProjects()
	if err != nil {
		return nil, errors.Wrap(err, "GetProjects")
	}
	for i := range projects {
		if projects[i].GetId() == projectId {
			return &projects[i], nil
		}
	}
	return nil, fmt.Errorf("failedd to found created project")
}
