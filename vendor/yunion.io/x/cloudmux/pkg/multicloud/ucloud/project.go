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

package ucloud

import (
	"yunion.io/x/cloudmux/pkg/multicloud"
)

// https://docs.ucloud.cn/api/summary/get_project_list
type SProject struct {
	multicloud.SProjectBase
	UcloudTags
	ProjectID     string `json:"ProjectId"`
	ProjectName   string `json:"ProjectName"`
	ParentID      string `json:"ParentId"`
	ParentName    string `json:"ParentName"`
	CreateTime    int64  `json:"CreateTime"`
	IsDefault     bool   `json:"IsDefault"`
	MemberCount   int64  `json:"MemberCount"`
	ResourceCount int64  `json:"ResourceCount"`
}

func (self *SProject) GetId() string {
	return self.ProjectID
}

func (self *SProject) GetName() string {
	return self.ProjectName
}

func (self *SProject) GetGlobalId() string {
	return self.GetId()
}

func (self *SProject) GetStatus() string {
	return ""
}

func (self *SProject) Refresh() error {
	return nil
}

func (self *SProject) IsEmulated() bool {
	return false
}

func (self *SUcloudClient) FetchProjects() ([]SProject, error) {
	params := NewUcloudParams()
	projects := make([]SProject, 0)
	err := self.DoListAll("GetProjectList", params, &projects)
	return projects, err
}
