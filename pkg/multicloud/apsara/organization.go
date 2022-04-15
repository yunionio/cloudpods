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
	"strings"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
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

type ResourceGroupList []SResourceGroupList

func (rgs ResourceGroupList) ToProjects(tags []string) []SProject {
	ret := []SProject{}
	for _, rg := range rgs {
		name := rg.ResourceGroupName
		if strings.HasPrefix(name, "ResourceSet(") {
			name = strings.TrimPrefix(name, "ResourceSet(")
			name = strings.TrimSuffix(name, ")")
		}
		proj := SProject{
			Id:   rg.Id,
			Name: name,
			Tags: tags,
		}
		ret = append(ret, proj)
	}
	return ret
}

type SOrganizationTree struct {
	Active            bool
	Alias             string
	Id                string
	Name              string
	ParentId          string
	MultiCCloudStatus string
	Children          []SOrganizationTree
	ResourceGroupList ResourceGroupList
	SupportRegions    string
	UUID              string
}

func (self *SOrganizationTree) GetProject(tags []string) []SProject {
	ret := []SProject{}
	if self.Name != "root" {
		tags = append(tags, self.Name)
	}
	ret = append(ret, self.ResourceGroupList.ToProjects(tags)...)
	if len(self.Children) == 0 {
		return ret
	}
	for _, child := range self.Children {
		ret = append(ret, child.GetProject(tags)...)
	}
	return ret
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

type SProject struct {
	multicloud.SProjectBase

	client *SApsaraClient
	Id     string
	Name   string
	Tags   []string
}

func (self *SProject) GetId() string {
	return self.Id
}

func (self *SProject) GetGlobalId() string {
	return self.Id
}

func (self *SProject) GetName() string {
	return self.Name
}

func (self *SProject) GetStatus() string {
	return api.EXTERNAL_PROJECT_STATUS_AVAILABLE
}

func (self *SProject) GetSysTags() map[string]string {
	return nil
}

func (self *SProject) SetTags(tags map[string]string, replace bool) error {
	return cloudprovider.ErrNotSupported
}

func (self *SProject) GetTags() (map[string]string, error) {
	ret := map[string]string{}
	for i, key := range self.Tags {
		ret[fmt.Sprintf("L%d", i+1)] = key
	}
	return ret, nil
}

func (self *SApsaraClient) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	tree, err := self.GetOrganizationTree(1)
	if err != nil {
		return nil, errors.Wrapf(err, "GetOrganizationTree")
	}
	ret := []cloudprovider.ICloudProject{}
	projects := tree.GetProject([]string{})
	for i := range projects {
		ret = append(ret, &projects[i])
	}
	return ret, nil
}
