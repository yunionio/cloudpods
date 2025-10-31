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
	"time"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SProject struct {
	multicloud.SProjectBase
	GoogleTags
	Name           string
	CreateTime     time.Time
	LifecycleState string
	ProjectId      string
	ProjectNumber  string
}

func (cli *SGoogleClient) GetProject(id string) (*SProject, error) {
	project := &SProject{}
	resp, err := cli.managerGet(id)
	if err != nil {
		return nil, errors.Wrap(err, "managerGet")
	}
	err = resp.Unmarshal(project)
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return project, nil
}

func (cli *SGoogleClient) GetProjects() ([]SProject, error) {
	nextPageToken := ""
	params := map[string]string{}
	result := []SProject{}
	for {
		if len(nextPageToken) > 0 {
			params["pageToken"] = nextPageToken
		}
		resp, err := cli.managerList("projects", params)
		if err != nil {
			return nil, errors.Wrap(err, "managerList")
		}
		part := struct {
			Projects []SProject
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrapf(err, "resp.Unmarshal")
		}
		result = append(result, part.Projects...)
		nextPageToken, _ = resp.GetString("nextPageToken")
		if len(nextPageToken) == 0 || len(part.Projects) == 0 {
			break
		}
	}
	return result, nil
}

func (p *SProject) GetName() string {
	return p.Name
}

func (p *SProject) GetId() string {
	return p.ProjectId
}

func (p *SProject) GetGlobalId() string {
	return p.ProjectId
}

func (p *SProject) GetStatus() string {
	return api.EXTERNAL_PROJECT_STATUS_AVAILABLE
}

func (p *SProject) Refresh() error {
	return nil
}

func (p *SProject) IsEmulated() bool {
	return false
}
