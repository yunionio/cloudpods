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

	"yunion.io/x/pkg/errors"
)

type SResourceGroupList struct {
	tree *SOrganizationTree

	Creator           string
	GmtCreated        int64
	GmtModified       int64
	Id                string
	OrganizationId    int
	OrganizationName  string
	ResourceGroupName string
	ResourceGroupType int
}

type SOrganization struct {
	Active            bool
	Alias             string
	Id                string
	Name              string
	ParentId          string
	MultiCCloudStatus string
	ResourceGroupList []SResourceGroupList
	SupportRegions    string
	UUID              string
}

type SOrganizationTree struct {
	Active            bool
	Alias             string
	Id                string
	Name              string
	ParentId          string
	MultiCCloudStatus string
	Children          []SOrganizationTree
	ResourceGroupList []SResourceGroupList
	SupportRegions    string
	UUID              string
}

func (self *SApsaraClient) GetOrganizationTree(id int) (*SOrganizationTree, error) {
	if id == 0 {
		id = 1
	}
	params := map[string]string{
		"Id": fmt.Sprintf("%d", id),
	}
	resp, err := self.ascmRequest("GetOrganizationTree", params)
	if err != nil {
		return nil, err
	}
	tree := SOrganizationTree{}
	err = resp.Unmarshal(&tree, "data")
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return &tree, nil
}

func (self *SApsaraClient) GetOrganizationList() ([]SOrganization, error) {
	params := map[string]string{"Id": "1"}
	resp, err := self.ascmRequest("GetOrganizationList", params)
	if err != nil {
		return nil, err
	}
	result := []SOrganization{}
	err = resp.Unmarshal(&result, "data")
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return result, nil
}

func (self *SOrganizationTree) ListProjects() []SResourceGroupList {
	ret := []SResourceGroupList{}
	for i := range self.ResourceGroupList {
		self.ResourceGroupList[i].tree = self
		ret = append(ret, self.ResourceGroupList[i])
	}
	for i := range self.Children {
		ret = append(ret, self.Children[i].ListProjects()...)
	}
	return ret
}
