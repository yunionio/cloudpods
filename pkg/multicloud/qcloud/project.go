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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
)

type SProject struct {
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

func (p *SProject) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (p *SProject) GetName() string {
	return p.ProjectName
}

func (p *SProject) GetStatus() string {
	return ""
}

func (p *SProject) IsEmulated() bool {
	return false
}

func (p *SProject) Refresh() error {
	return nil
}
