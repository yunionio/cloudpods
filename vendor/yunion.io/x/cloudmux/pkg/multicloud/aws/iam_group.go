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

	api "yunion.io/x/cloudmux/pkg/apis/cloudid"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SGroups struct {
	Groups      []SGroup `xml:"Groups>member"`
	IsTruncated bool     `xml:"IsTruncated"`
	Marker      string   `xml:"Marker"`
}

type SGroup struct {
	client *SAwsClient

	Path       string    `xml:"Path"`
	GroupName  string    `xml:"GroupName"`
	GroupId    string    `xml:"GroupId"`
	Arn        string    `xml:"Arn"`
	CreateDate time.Time `xml:"CreateDate"`
}

func (self *SGroup) GetName() string {
	return self.GroupName
}

func (self *SGroup) GetDescription() string {
	return ""
}

func (self *SGroup) GetGlobalId() string {
	return self.GroupName
}

func (self *SGroup) AddUser(userName string) error {
	return self.client.AddUserToGroup(self.GroupName, userName)
}

func (self *SGroup) RemoveUser(userName string) error {
	return self.client.RemoveUserFromGroup(self.GroupName, userName)
}

func (self *SGroup) GetICloudusers() ([]cloudprovider.IClouduser, error) {
	users, err := self.client.ListGroupUsers(self.GroupName)
	if err != nil {
		return nil, errors.Wrapf(err, "ListGroupUsers")
	}
	ret := []cloudprovider.IClouduser{}
	for i := range users {
		users[i].client = self.client
		ret = append(ret, &users[i])
	}
	return ret, nil
}

func (self *SGroup) AttachPolicy(policyId string, policyType api.TPolicyType) error {
	return self.client.AttachGroupPolicy(self.GroupName, policyId)
}

func (self *SGroup) DetachPolicy(policyId string, policyType api.TPolicyType) error {
	return self.client.DetachGroupPolicy(self.GroupName, policyId)
}

func (self *SGroup) Delete() error {
	return self.client.DeleteGroup(self.GroupName)
}

func (self *SAwsClient) CreateICloudgroup(name, desc string) (cloudprovider.ICloudgroup, error) {
	return self.CreateGroup(name, "")
}

func (self *SGroup) ListPolicies() ([]SAttachedPolicy, error) {
	policies := []SAttachedPolicy{}
	offset := ""
	for {
		part, err := self.client.ListAttachedGroupPolicies(self.GroupName, offset, 1000)
		if err != nil {
			return nil, errors.Wrapf(err, "ListAttachedGroupPolicies")
		}
		for i := range part.AttachedPolicies {
			part.AttachedPolicies[i].client = self.client
			policies = append(policies, part.AttachedPolicies[i])
		}
		offset = part.Marker
		if len(offset) == 0 || !part.IsTruncated {
			break
		}
	}
	return policies, nil
}

func (self *SGroup) ListGroupPolicies() ([]SPolicy, error) {
	policies := []SPolicy{}
	offset := ""
	for {
		part, err := self.client.ListGroupPolicies(self.GroupName, offset, 1000)
		if err != nil {
			return nil, errors.Wrapf(err, "ListGroupPolicies")
		}
		for i := range part.Policies {
			part.Policies[i].client = self.client
			policies = append(policies, part.Policies[i])
		}
		offset = part.Marker
		if len(offset) == 0 || !part.IsTruncated {
			break
		}
	}
	return policies, nil
}

func (self *SGroup) GetICloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	policies, err := self.ListPolicies()
	if err != nil {
		return nil, errors.Wrapf(err, "ListPolicies")
	}
	ret := []cloudprovider.ICloudpolicy{}
	for i := range policies {
		ret = append(ret, &policies[i])
	}
	return ret, nil
}

func (self *SAwsClient) GetICloudgroups() ([]cloudprovider.ICloudgroup, error) {
	ret := []cloudprovider.ICloudgroup{}
	offset := ""
	for {
		part, err := self.ListGroups(offset, 1000, "")
		if err != nil {
			return nil, errors.Wrapf(err, "ListGroups")
		}
		for i := range part.Groups {
			part.Groups[i].client = self
			ret = append(ret, &part.Groups[i])
		}
		offset = part.Marker
		if len(offset) == 0 || !part.IsTruncated {
			break
		}
	}
	return ret, nil
}

func (self *SAwsClient) GetICloudgroupByName(name string) (cloudprovider.ICloudgroup, error) {
	group, err := self.GetGroup(name, "", 1)
	if err != nil {
		return nil, errors.Wrapf(err, "GetGroup(%s)", name)
	}
	group.Group.client = self
	return &group.Group, nil
}

func (self *SAwsClient) ListGroupUsers(groupName string) ([]SUser, error) {
	users := []SUser{}
	offset := ""
	for {
		part, err := self.GetGroup(groupName, offset, 1000)
		if err != nil {
			return nil, errors.Wrap(err, "GetGroup")
		}
		users = append(users, part.Users...)
		offset = part.Marker
		if len(offset) == 0 {
			break
		}
	}
	return users, nil
}

func (self *SAwsClient) CreateGroup(name string, path string) (*SGroup, error) {
	params := map[string]string{
		"Path":      "/",
		"GroupName": name,
	}
	if len(path) > 0 {
		params["Path"] = path
	}
	group := struct {
		Group SGroup `xml:"Group"`
	}{}
	err := self.iamRequest("CreateGroup", params, &group)
	if err != nil {
		return nil, errors.Wrap(err, "iamRequest.CreateGroup")
	}
	group.Group.client = self
	return &group.Group, nil
}

func (self *SAwsClient) ListGroups(offset string, limit int, pathPrefix string) (*SGroups, error) {
	if limit <= 0 || limit > 1000 {
		limit = 1000
	}
	params := map[string]string{
		"MaxItems": fmt.Sprintf("%d", limit),
	}
	if len(offset) > 0 {
		params["Marker"] = offset
	}
	if len(pathPrefix) > 0 {
		params["PathPrefix"] = pathPrefix
	}
	groups := &SGroups{}
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
	return self.iamRequest("DeleteGroup", params, nil)
}

type SGroupDetails struct {
	Group       SGroup  `xml:"Group"`
	Users       []SUser `xml:"Users>member"`
	IsTruncated bool    `xml:"IsTruncated"`
	Marker      string  `xml:"Marker"`
}

func (self *SAwsClient) GetGroup(name string, offset string, limit int) (*SGroupDetails, error) {
	if limit <= 0 || limit > 1000 {
		limit = 1000
	}
	params := map[string]string{
		"GroupName": name,
		"MaxItems":  fmt.Sprintf("%d", limit),
	}
	if len(offset) > 0 {
		params["Marker"] = offset
	}
	group := &SGroupDetails{}
	err := self.iamRequest("GetGroup", params, group)
	if err != nil {
		return nil, errors.Wrap(err, "iamRequest.GetGroup")
	}
	return group, nil
}

func (self *SAwsClient) ListGroupPolicies(name string, offset string, limit int) (*SPolicies, error) {
	if limit <= 0 || limit > 1000 {
		limit = 1000
	}

	params := map[string]string{
		"GroupName": name,
		"MaxItems":  fmt.Sprintf("%d", limit),
	}
	if len(offset) > 0 {
		params["Marker"] = offset
	}
	policies := &SPolicies{}
	err := self.iamRequest("ListGroupPolicies", params, policies)
	if err != nil {
		return nil, errors.Wrap(err, "iamRequest.ListGroupPolicies")
	}
	return policies, nil
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
