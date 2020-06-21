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

package aws

import (
	"fmt"
	"time"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SCloudgroup struct {
	client *SAwsClient

	Path       string    `xml:"Path"`
	GroupName  string    `xml:"GroupName"`
	GroupId    string    `xml:"GroupId"`
	Arn        string    `xml:"Arn"`
	CreateDate time.Time `xml:"CreateDate"`
}

func (self *SCloudgroup) GetGlobalId() string {
	return self.GroupId
}

func (self *SCloudgroup) GetName() string {
	return self.GroupName
}

func (self *SCloudgroup) GetDescription() string {
	return self.Path
}

func (self *SCloudgroup) Delete() error {
	return self.client.DeleteGroup(self.GroupName)
}

func (self *SCloudgroup) AddUser(name string) error {
	return self.client.AddUserToGroup(self.GroupName, name)
}

func (self *SCloudgroup) RemoveUser(name string) error {
	return self.client.RemoveUserFromGroup(self.GroupName, name)
}

func (self *SCloudgroup) AttachSystemPolicy(policyArn string) error {
	return self.client.AttachGroupPolicy(self.GroupName, policyArn)
}

func (self *SCloudgroup) DetachSystemPolicy(policyArn string) error {
	return self.client.DetachGroupPolicy(self.GroupName, policyArn)
}

func (self *SCloudgroup) GetICloudusers() ([]cloudprovider.IClouduser, error) {
	users := []SClouduser{}
	marker := ""
	for {
		part, err := self.client.GetGroup(self.GroupName, marker, 1000)
		if err != nil {
			return nil, errors.Wrap(err, "GetGroup")
		}
		users = append(users, part.Users...)
		marker = part.Marker
		if len(marker) == 0 {
			break
		}
	}
	ret := []cloudprovider.IClouduser{}
	for i := range users {
		users[i].client = self.client
		ret = append(ret, &users[i])
	}
	return ret, nil
}

func (self *SCloudgroup) GetISystemCloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	policies := []SPolicy{}
	marker := ""
	for {
		part, err := self.client.ListGroupPolicies(self.GroupName, marker, 1000)
		if err != nil {
			return nil, errors.Wrap(err, "ListGroupPolicies")
		}
		policies = append(policies, part.Policies...)
		marker = part.Marker
		if len(marker) == 0 {
			break
		}
	}
	ret := []cloudprovider.ICloudpolicy{}
	for i := range policies {
		ret = append(ret, &policies[i])
	}
	return ret, nil
}

type SCloudgroups struct {
	Groups      []SCloudgroup `xml:"Groups>member"`
	IsTruncated bool          `xml:"IsTruncated"`
	Marker      string        `xml:"Marker"`
}

func (self *SAwsClient) CreateGroup(name string, path string) (*SCloudgroup, error) {
	params := map[string]string{
		"Path":      "/",
		"GroupName": name,
	}
	if len(path) > 0 {
		params["Path"] = path
	}
	group := struct {
		Group SCloudgroup `xml:"Group"`
	}{}
	err := self.iamRequest("CreateGroup", params, &group)
	if err != nil {
		return nil, errors.Wrap(err, "iamRequest.CreateGroup")
	}
	group.Group.client = self
	return &group.Group, nil
}

func (self *SAwsClient) ListGroups(marker string, maxItems int, pathPrefix string) (*SCloudgroups, error) {
	if maxItems <= 0 || maxItems > 1000 {
		maxItems = 1000
	}
	params := map[string]string{
		"MaxItems": fmt.Sprintf("%d", maxItems),
	}
	if len(marker) > 0 {
		params["Marker"] = marker
	}
	if len(pathPrefix) > 0 {
		params["PathPrefix"] = pathPrefix
	}
	groups := &SCloudgroups{}
	err := self.iamRequest("ListGroups", params, groups)
	if err != nil {
		return nil, errors.Wrap(err, "iamRequest.ListGroups")
	}
	return groups, nil
}

func (self *SAwsClient) DeleteGroup(name string) error {
	params := map[string]string{
		"GroupName": name,
	}
	return self.iamRequest("ListGroups", params, nil)
}

func (self *SAwsClient) GetICloudgroups() ([]cloudprovider.ICloudgroup, error) {
	groups := []SCloudgroup{}
	marker := ""
	for {
		part, err := self.ListGroups(marker, 1000, "")
		if err != nil {
			return nil, errors.Wrap(err, "ListGroups")
		}
		groups = append(groups, part.Groups...)
		marker = part.Marker
		if len(marker) == 0 {
			break
		}
	}
	ret := []cloudprovider.ICloudgroup{}
	for i := range groups {
		groups[i].client = self
		ret = append(ret, &groups[i])
	}
	return ret, nil
}

type SCloudgroupDetails struct {
	Group       SCloudgroup  `xml:"Group"`
	Users       []SClouduser `xml:"Users>member"`
	IsTruncated bool         `xml:"IsTruncated"`
	Marker      string       `xml:"Marker"`
}

func (self *SAwsClient) GetGroup(name string, marker string, maxItems int) (*SCloudgroupDetails, error) {
	if maxItems <= 0 || maxItems > 1000 {
		maxItems = 1000
	}
	params := map[string]string{
		"GroupName": name,
		"MaxItems":  fmt.Sprintf("%d", maxItems),
	}
	if len(marker) > 0 {
		params["Marker"] = marker
	}
	group := &SCloudgroupDetails{}
	err := self.iamRequest("GetGroup", params, group)
	if err != nil {
		return nil, errors.Wrap(err, "iamRequest.GetGroup")
	}
	return group, nil
}

func (self *SAwsClient) GetICloudgroupByName(name string) (cloudprovider.ICloudgroup, error) {
	group, err := self.GetGroup(name, "", 1)
	if err != nil {
		return nil, err
	}
	group.Group.client = self
	return &group.Group, nil
}

func (self *SAwsClient) ListGroupPolicies(name string, marker string, maxItems int) (*SPolicies, error) {
	if maxItems <= 0 || maxItems > 1000 {
		maxItems = 1000
	}

	params := map[string]string{
		"GroupName": name,
		"MaxItems":  fmt.Sprintf("%d", maxItems),
	}
	if len(marker) > 0 {
		params["Marker"] = marker
	}
	policies := &SPolicies{}
	err := self.iamRequest("ListGroupPolicies", params, policies)
	if err != nil {
		return nil, errors.Wrap(err, "iamRequest.ListGroupPolicies")
	}
	return policies, nil
}

func (self *SAwsClient) CreateICloudgroup(name, desc string) (cloudprovider.ICloudgroup, error) {
	group, err := self.CreateGroup(name, "")
	if err != nil {
		return nil, errors.Wrap(err, "CreateGroup")
	}
	return group, nil
}

func (self *SAwsClient) AddUserToGroup(groupName, userName string) error {
	params := map[string]string{
		"GroupName": groupName,
		"UserName":  userName,
	}
	return self.iamRequest("AddUserToGroup", params, nil)
}

func (self *SAwsClient) RemoveUserFromGroup(groupName, userName string) error {
	params := map[string]string{
		"GroupName": groupName,
		"UserName":  userName,
	}
	return self.iamRequest("RemoveUserFromGroup", params, nil)
}

func (self *SAwsClient) AttachGroupPolicy(groupName, policyArn string) error {
	params := map[string]string{
		"GroupName": groupName,
		"PolicyArn": policyArn,
	}
	return self.iamRequest("AttachGroupPolicy", params, nil)
}

func (self *SAwsClient) DetachGroupPolicy(groupName, policyArn string) error {
	params := map[string]string{
		"GroupName": groupName,
		"PolicyArn": policyArn,
	}
	err := self.iamRequest("DetachGroupPolicy", params, nil)
	if err != nil && errors.Cause(err) != cloudprovider.ErrNotFound {
		return errors.Wrap(err, "DetachGroupPolicy")
	}
	return nil
}
