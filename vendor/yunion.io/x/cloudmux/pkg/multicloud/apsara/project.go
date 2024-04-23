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

package apsara

import (
	"fmt"
	"time"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type DepartmentInfo struct {
	Department        string
	DepartmentName    string
	ResourceGroup     string
	ResourceGroupName string
	ResourceGroupId   string
}

func (self DepartmentInfo) GetProjectId() string {
	return self.ResourceGroup
}

type SResourceGroup struct {
	multicloud.SProjectBase
	ApsaraTags
	client *SApsaraClient

	Creator           string
	GmtCreated        string
	Id                string
	OrganizationId    string
	OrganizationName  string
	ResourceGroupName string
	RsId              string
}

func (self *SResourceGroup) GetGlobalId() string {
	return self.Id
}

func (self *SResourceGroup) GetId() string {
	return self.Id
}

func (self *SResourceGroup) GetName() string {
	return self.ResourceGroupName
}

func (self *SResourceGroup) Refresh() error {
	return nil
}

func (self *SResourceGroup) GetStatus() string {
	return api.EXTERNAL_PROJECT_STATUS_AVAILABLE
}

func (self *SApsaraClient) GetResourceGroups(pageNumber int, pageSize int) ([]SResourceGroup, int, error) {
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
	resp, err := self.ascmRequest("ListResourceGroup", params)
	if err != nil {
		return nil, 0, errors.Wrap(err, "rmRequest.ListResourceGroup")
	}
	groups := []SResourceGroup{}
	err = resp.Unmarshal(&groups, "data")
	if err != nil {
		return nil, 0, errors.Wrap(err, "resp.Unmarshal")
	}
	total, _ := resp.Int("pageInfo", "total")
	return groups, int(total), nil
}

func (self *SApsaraClient) CreateIProject(name string) (cloudprovider.ICloudProject, error) {
	group, err := self.CreateResourceGroup(name)
	if err != nil {
		return nil, errors.Wrap(err, "CreateProject")
	}
	return group, nil
}

func (self *SApsaraClient) CreateResourceGroup(name string) (*SResourceGroup, error) {
	params := map[string]string{
		"DisplayName": name,
		"Name":        name,
	}
	resp, err := self.ascmRequest("CreateResourceGroup", params)
	if err != nil {
		return nil, errors.Wrap(err, "CreateResourceGroup")
	}
	group := &SResourceGroup{client: self}
	err = resp.Unmarshal(group, "ResourceGroup")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	err = cloudprovider.WaitStatus(group, api.EXTERNAL_PROJECT_STATUS_AVAILABLE, time.Second*5, time.Minute*3)
	if err != nil {
		return nil, errors.Wrap(err, "WaitStatus")
	}
	return group, nil
}

func (self *SApsaraClient) GetResourceGroup(id string) (*SResourceGroup, error) {
	params := map[string]string{
		"ResourceGroupId": id,
	}
	resp, err := self.ascmRequest("GetResourceGroup", params)
	if err != nil {
		return nil, err
	}
	group := &SResourceGroup{client: self}
	err = resp.Unmarshal(group, "ResourceGroup")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return group, nil
}
