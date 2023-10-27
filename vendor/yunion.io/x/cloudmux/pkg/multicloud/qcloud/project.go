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

func (client *SQcloudClient) GetProjects(offset, limit int) ([]SProject, int, error) {
	if limit < 1 || limit > 1000 {
		limit = 1000
	}
	params := map[string]string{"AllList": "0"}
	params["Limit"] = fmt.Sprintf("%d", limit)
	params["Offset"] = fmt.Sprintf("%d", offset)

	resp, err := client.tagRequest("DescribeProjects", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeProjects")
	}
	projects := []SProject{}
	err = resp.Unmarshal(&projects, "Projects")
	if err != nil {
		return nil, 0, errors.Wrap(err, "resp.Unmarshal")
	}
	total, _ := resp.Float("Total")
	return projects, int(total), nil
}

func (client *SQcloudClient) CreateProject(name, desc string) (*SProject, error) {
	params := map[string]string{
		"ProjectName": name,
	}
	if len(desc) > 0 {
		params["Info"] = desc
	}
	body, err := client.tagRequest("AddProject", params)
	if err != nil {
		return nil, errors.Wrap(err, "AddProject")
	}
	projectId, _ := body.GetString("ProjectId")
	if len(projectId) == 0 {
		return nil, fmt.Errorf("empty project reture")
	}
	return &SProject{
		client:      client,
		ProjectName: name,
		ProjectId:   projectId,
		ProjectInfo: desc,
	}, nil
}
