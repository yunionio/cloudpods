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
	"time"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
)

type SProject struct {
	Name           string
	CreateTime     time.Time
	LifecycleState string
	ProjectId      string
	ProjectNumber  string
}

func (cli *SGoogleClient) GetProject(id string) (*SProject, error) {
	project := &SProject{}
	return project, cli.get(id, project)
}

func (cli *SGoogleClient) GetProjects() ([]SProject, error) {
	baseUrl := "https://cloudresourcemanager.googleapis.com/v1/projects"
	nextPageToken := ""
	result := []SProject{}
	for {
		url := baseUrl
		if len(nextPageToken) > 0 {
			url = fmt.Sprintf("%s?pageToken=%s", baseUrl, nextPageToken)
		}
		data, err := jsonRequest(cli.client, "GET", url, nil, cli.Debug)
		if err != nil {
			return nil, errors.Wrap(err, "JSONRequest")
		}
		_result := []SProject{}
		if data.Contains("projects") {
			err = data.Unmarshal(&_result, "projects")
			if err != nil {
				return nil, errors.Wrap(err, "data.Unmarshal")
			}
		}
		result = append(result, _result...)
		nextPageToken, _ = data.GetString("nextPageToken")
		if len(nextPageToken) == 0 || len(_result) == 0 {
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
	return ""
}

func (p *SProject) Refresh() error {
	return nil
}

func (p *SProject) IsEmulated() bool {
	return false
}

func (p *SProject) GetMetadata() *jsonutils.JSONDict {
	return nil
}
