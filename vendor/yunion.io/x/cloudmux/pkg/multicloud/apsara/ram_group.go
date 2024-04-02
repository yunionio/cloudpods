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
	"time"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/cloudid"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SGroup struct {
	client *SApsaraClient

	Comments    string
	CreatedDate time.Time
	GroupName   string
	UpdateDate  time.Time
}

type sGroups struct {
	Group []SGroup
}

type SGroups struct {
	Groups      sGroups
	Marker      string
	IsTruncated bool
}

func (self *SGroup) GetName() string {
	return self.GroupName
}

func (self *SGroup) GetGlobalId() string {
	return self.GroupName
}

func (self *SGroup) GetDescription() string {
	return self.Comments
}

func (self *SGroup) GetICloudusers() ([]cloudprovider.IClouduser, error) {
	ret := []cloudprovider.IClouduser{}
	offset := ""
	for {
		part, err := self.client.ListUsersForGroup(self.GroupName, offset, 1000)
		if err != nil {
			return nil, errors.Wrapf(err, "ListUsersForGroup")
		}
		for i := range part.Users.User {
			part.Users.User[i].client = self.client
			ret = append(ret, &part.Users.User[i])
		}
		offset = part.Marker
		if len(offset) == 0 || !part.IsTruncated {
			break
		}
	}
	return ret, nil
}

func (self *SGroup) GetICloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	policies, err := self.client.ListPoliciesForGroup(self.GroupName)
	if err != nil {
		return nil, errors.Wrapf(err, "ListPoliciesForGroup")
	}
	ret := []cloudprovider.ICloudpolicy{}
	for i := range policies {
		policies[i].client = self.client
		ret = append(ret, &policies[i])
	}
	return ret, nil
}

func (self *SGroup) AddUser(name string) error {
	return self.client.AddUserToGroup(self.GroupName, name)
}

func (self *SGroup) RemoveUser(name string) error {
	return self.client.RemoveUserFromGroup(self.GroupName, name)
}

func (self *SGroup) AttachPolicy(policyName string, policyType api.TPolicyType) error {
	return self.client.AttachPolicyToGroup(utils.Capitalize(string(policyType)), policyName, self.GroupName)
}

func (self *SGroup) DetachPolicy(policyName string, policyType api.TPolicyType) error {
	return self.client.DetachPolicyFromGroup(utils.Capitalize(string(policyType)), policyName, self.GroupName)
}

func (self *SGroup) Delete() error {
	return self.client.DeleteGroup(self.GroupName)
}

func (self *SApsaraClient) GetICloudgroupByName(name string) (cloudprovider.ICloudgroup, error) {
	group, err := self.GetGroup(name)
	if err != nil {
		return nil, errors.Wrapf(err, "GetGroup(%s)", name)
	}
	return group, nil
}

func (self *SApsaraClient) CreateICloudgroup(name string, desc string) (cloudprovider.ICloudgroup, error) {
	group, err := self.CreateGroup(name, desc)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateGroup")
	}
	return group, nil
}

func (self *SApsaraClient) GetICloudgroups() ([]cloudprovider.ICloudgroup, error) {
	ret := []cloudprovider.ICloudgroup{}
	offset := ""
	for {
		part, err := self.ListGroups(offset, 100)
		if err != nil {
			return nil, errors.Wrap(err, "ListGroups")
		}
		for i := range part.Groups.Group {
			part.Groups.Group[i].client = self
			ret = append(ret, &part.Groups.Group[i])
		}
		offset = part.Marker
		if len(offset) == 0 || !part.IsTruncated {
			break
		}
	}
	return ret, nil
}

func (self *SApsaraClient) ListGroups(offset string, limit int) (*SGroups, error) {
	if limit < 1 || limit > 1000 {
		limit = 1000
	}
	params := map[string]string{
		"MaxItems": fmt.Sprintf("%d", limit),
	}
	if len(offset) > 0 {
		params["Marker"] = offset
	}
	groups := SGroups{}
	resp, err := self.ramRequest("ListGroups", params)
	if err != nil {
		return nil, errors.Wrap(err, "ramRequest.ListGroups")
	}
	err = resp.Unmarshal(&groups)
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return &groups, nil
}

// https://help.apsara.com/document_detail/28732.html?spm=a2c4g.11186623.6.777.580735b2m2xUh8
func (self *SApsaraClient) ListPoliciesForGroup(groupName string) ([]SPolicy, error) {
	params := map[string]string{
		"GroupName": groupName,
	}
	resp, err := self.ramRequest("ListPoliciesForGroup", params)
	if err != nil {
		return nil, errors.Wrap(err, "ramRequest.ListPoliciesForGroup")
	}
	policies := []SPolicy{}
	err = resp.Unmarshal(&policies, "Policies", "Policy")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return policies, nil
}

func (self *SApsaraClient) ListUsersForGroup(groupName string, offset string, limit int) (*SUsers, error) {
	if limit < 1 || limit > 1000 {
		limit = 1000
	}
	params := map[string]string{
		"GroupName": groupName,
		"MaxItems":  fmt.Sprintf("%d", limit),
	}
	if len(offset) > 0 {
		params["Marker"] = offset
	}
	resp, err := self.ramRequest("ListUsersForGroup", params)
	if err != nil {
		return nil, errors.Wrap(err, "ramRequest.ListUserForGroup")
	}
	users := &SUsers{}
	err = resp.Unmarshal(users)
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return users, nil
}

func (self *SApsaraClient) DeleteGroup(groupName string) error {
	params := map[string]string{
		"GroupName": groupName,
	}
	_, err := self.ramRequest("DeleteGroup", params)
	return err
}

func (self *SApsaraClient) CreateGroup(groupName, comments string) (*SGroup, error) {
	params := map[string]string{
		"GroupName": groupName,
	}
	if len(comments) > 0 {
		params["Comments"] = comments
	}
	resp, err := self.ramRequest("CreateGroup", params)
	if err != nil {
		return nil, errors.Wrap(err, "ramRequest.CreateGroup")
	}
	group := &SGroup{client: self}
	err = resp.Unmarshal(group, "Group")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return group, nil
}

func (self *SApsaraClient) GetGroup(groupName string) (*SGroup, error) {
	params := map[string]string{
		"GroupName": groupName,
	}
	resp, err := self.ramRequest("GetGroup", params)
	if err != nil {
		return nil, errors.Wrap(err, "GetGroup")
	}
	group := &SGroup{client: self}
	err = resp.Unmarshal(group, "Group")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return group, nil
}

func (self *SApsaraClient) RemoveUserFromGroup(groupName, userName string) error {
	params := map[string]string{
		"GroupName": groupName,
		"UserName":  userName,
	}
	_, err := self.ramRequest("RemoveUserFromGroup", params)
	if err != nil && errors.Cause(err) != cloudprovider.ErrNotFound {
		return errors.Wrap(err, "RemoveUserFromGroup")
	}
	return nil
}

func (self *SApsaraClient) DetachPolicyFromGroup(policyType, policyName, groupName string) error {
	params := map[string]string{
		"GroupName":  groupName,
		"PolicyName": policyName,
		"PolicyType": policyType,
	}
	_, err := self.ramRequest("DetachPolicyFromGroup", params)
	if err != nil && errors.Cause(err) != cloudprovider.ErrNotFound {
		return errors.Wrap(err, "DetachPolicyFromGroup")
	}
	return nil
}

func (self *SApsaraClient) AddUserToGroup(groupName, userName string) error {
	params := map[string]string{
		"GroupName": groupName,
		"UserName":  userName,
	}
	_, err := self.ramRequest("AddUserToGroup", params)
	if err != nil && !strings.Contains(err.Error(), "EntityAlreadyExists.User.Group") {
		return errors.Wrap(err, "AddUserToGroup")
	}
	return nil
}

func (self *SApsaraClient) AttachPolicyToGroup(policyType, policyName, groupName string) error {
	params := map[string]string{
		"GroupName":  groupName,
		"PolicyName": policyName,
		"PolicyType": policyType,
	}
	_, err := self.ramRequest("AttachPolicyToGroup", params)
	if err != nil && !strings.Contains(err.Error(), "EntityAlreadyExists.Group.Policy") {
		return errors.Wrap(err, "AttachPolicyToGroup")
	}
	return nil
}
