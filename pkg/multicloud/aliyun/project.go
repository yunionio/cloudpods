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

package aliyun

import (
	"fmt"
	"time"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SResourceGroup struct {
	multicloud.SResourceBase

	Status      string
	AccountId   string
	DisplayName string
	Id          string
	CreateDate  time.Time
	Name        string
}

func (self *SResourceGroup) GetGlobalId() string {
	return self.Id
}

func (self *SResourceGroup) GetId() string {
	return self.Id
}

func (self *SResourceGroup) GetName() string {
	if len(self.DisplayName) > 0 {
		return self.DisplayName
	}
	return self.Name
}

func (self *SResourceGroup) GetStatus() string {
	return ""
}

func (self *SAliyunClient) GetResourceGroups(pageNumber int, pageSize int) ([]SResourceGroup, int, error) {
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
	}
	if pageNumber <= 0 {
		pageNumber = 1
	}
	params := map[string]string{
		"PageNumber": fmt.Sprintf("%d", pageNumber),
		"PageSize":   fmt.Sprintf("%d", pageSize),
	}
	resp, err := self.rmRequest("ListResourceGroups", params)
	if err != nil {
		return nil, 0, errors.Wrap(err, "rmRequest.ListResourceGroups")
	}
	groups := []SResourceGroup{}
	err = resp.Unmarshal(&groups, "ResourceGroups", "ResourceGroup")
	if err != nil {
		return nil, 0, errors.Wrap(err, "resp.Unmarshal")
	}
	total, _ := resp.Int("TotalCount")
	return groups, int(total), nil
}

func (self *SAliyunClient) CreateIProject(name string) (cloudprovider.ICloudProject, error) {
	project, err := self.CreateProject(name)
	if err != nil {
		return nil, errors.Wrap(err, "CreateProject")
	}
	return project, nil
}

func (self *SAliyunClient) CreateProject(name string) (*SResourceGroup, error) {
	params := map[string]string{
		"DisplayName": name,
		"Name":        name,
	}
	resp, err := self.rmRequest("CreateResourceGroup", params)
	if err != nil {
		return nil, errors.Wrap(err, "CreateResourceGroup")
	}
	group := SResourceGroup{}
	err = resp.Unmarshal(&group, "ResourceGroup")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return &group, nil
}

func (self *SAliyunClient) SetProjectId(id string) {
	self.projectId = id
}
