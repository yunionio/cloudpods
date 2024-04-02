// Copyright 2023 Yunion
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

package volcengine

import (
	"fmt"
	"time"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

type SProject struct {
	multicloud.SProjectBase
	VolcEngineTags
	client *SVolcEngineClient

	AccountId         int
	ProjectName       string
	ParentProjectName string
	Path              string
	DisplayName       string
	Description       string
	CreateDate        string
	UpdateDate        string
	Status            string
}

func (project *SProject) GetGlobalId() string {
	return project.ProjectName
}

func (project *SProject) GetId() string {
	return project.ProjectName
}

func (project *SProject) GetName() string {
	if len(project.DisplayName) > 0 {
		return project.DisplayName
	}
	return project.ProjectName
}

func (project *SProject) Refresh() error {
	group, err := project.client.GetProject(project.ProjectName)
	if err != nil {
		return errors.Wrap(err, "GetProject")
	}
	return jsonutils.Update(project, group)
}

func (project *SProject) GetStatus() string {
	switch project.Status {
	case "active":
		return api.EXTERNAL_PROJECT_STATUS_AVAILABLE
	default:
		return api.EXTERNAL_PROJECT_STATUS_UNKNOWN
	}
}

func (client *SVolcEngineClient) GetProject(name string) (*SProject, error) {
	params := map[string]string{
		"ProjectName": name,
	}
	body, err := client.iam20210801Request("", "GetProject", params)
	if err != nil {
		return nil, err
	}
	project := &SProject{client: client}
	err = body.Unmarshal(project)
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return project, nil
}

func (client *SVolcEngineClient) ListProjects(limit int, offset int) ([]SProject, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}
	params := map[string]string{
		"Limit":  fmt.Sprintf("%d", limit),
		"Offset": fmt.Sprintf("%d", offset),
	}
	resp, err := client.iam20210801Request("", "ListProjects", params)
	if err != nil {
		return nil, 0, errors.Wrap(err, "iamRequest.ListProjects")
	}
	projects := []SProject{}
	err = resp.Unmarshal(&projects, "Projects")
	if err != nil {
		return nil, 0, errors.Wrap(err, "resp.Unmarshal")
	}
	total, _ := resp.Int("Total")
	return projects, int(total), nil
}

func (client *SVolcEngineClient) CreateIProject(name string) (cloudprovider.ICloudProject, error) {
	group, err := client.CreateProject(name)
	if err != nil {
		return nil, errors.Wrap(err, "CreateProject")
	}
	return group, nil
}

func (client *SVolcEngineClient) CreateProject(name string) (*SProject, error) {
	params := map[string]string{
		"DisplayName": name,
		"ProjectName": name,
	}
	resp, err := client.iam20210801Request("", "CreateProject", params)
	if err != nil {
		return nil, errors.Wrap(err, "CreateProject")
	}
	group := &SProject{client: client}
	err = resp.Unmarshal(group, "Project")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	err = cloudprovider.WaitStatus(group, api.EXTERNAL_PROJECT_STATUS_AVAILABLE, time.Second*5, time.Minute*3)
	if err != nil {
		return nil, errors.Wrap(err, "WaitStatus")
	}
	return group, nil
}
