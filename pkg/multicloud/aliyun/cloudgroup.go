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
	"strings"
	"time"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SCloudgroup struct {
	client *SAliyunClient

	Comments    string
	CreatedDate time.Time
	GroupName   string
	UpdateDate  time.Time
}

func (group *SCloudgroup) GetName() string {
	return group.GroupName
}

func (group *SCloudgroup) GetDescription() string {
	return group.Comments
}

func (group *SCloudgroup) GetGlobalId() string {
	return group.GroupName
}

func (group *SCloudgroup) AddUser(name string) error {
	return group.client.AddUserToGroup(group.GroupName, name)
}

func (group *SCloudgroup) RemoveUser(name string) error {
	return group.client.RemoveUserFromGroup(group.GroupName, name)
}

func (group *SCloudgroup) AttachSystemPolicy(policyName string) error {
	return group.client.AttachGroupPolicy(group.GroupName, policyName, "System")
}

func (group *SCloudgroup) DetachSystemPolicy(policyName string) error {
	return group.client.DetachPolicyFromGroup(group.GroupName, policyName, "System")
}

func (group *SCloudgroup) Delete() error {
	users, err := group.GetICloudusers()
	if err != nil {
		return errors.Wrap(err, "GetICloudusers")
	}
	for i := range users {
		err = group.client.RemoveUserFromGroup(group.GroupName, users[i].GetName())
		if err != nil {
			return errors.Wrapf(err, "RemoveUserFromGroup(%s)", users[i].GetName())
		}
	}
	policies, err := group.client.ListGroupPolicies(group.GroupName)
	if err != nil {
		return errors.Wrap(err, "GetICloudpolicies")
	}
	for i := range policies {
		err = group.client.DetachPolicyFromGroup(group.GroupName, policies[i].GetName(), policies[i].PolicyType)
		if err != nil {
			return errors.Wrapf(err, "DetachPolicyFromGroup(%s)", policies[i].GetName())
		}
	}
	return group.client.DeleteGroup(group.GroupName)
}

func (group *SCloudgroup) GetICloudusers() ([]cloudprovider.IClouduser, error) {
	users := []SClouduser{}
	marker := ""
	for {
		part, err := group.client.ListGroupUsers(group.GroupName, marker, 100)
		if err != nil {
			return nil, errors.Wrap(err, "ListGroupUsers")
		}
		users = append(users, part.Users.User...)
		marker = part.Marker
		if len(marker) == 0 {
			break
		}
	}
	ret := []cloudprovider.IClouduser{}
	for i := range users {
		users[i].client = group.client
		ret = append(ret, &users[i])
	}
	return ret, nil
}

func (group *SCloudgroup) GetISystemCloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	policies, err := group.client.ListGroupPolicies(group.GroupName)
	if err != nil {
		return nil, errors.Wrap(err, "ListGroupPolicie")
	}
	ret := []cloudprovider.ICloudpolicy{}
	for i := range policies {
		if policies[i].PolicyType == "System" {
			ret = append(ret, &policies[i])
		}
	}
	return ret, nil
}

type Cloudgroups struct {
	Group []SCloudgroup
}

type SCloudgroups struct {
	Groups      Cloudgroups
	Marker      string
	IsTruncated bool
}

func (self *SAliyunClient) GetICloudgroups() ([]cloudprovider.ICloudgroup, error) {
	groups := []SCloudgroup{}
	marker := ""
	for {
		part, err := self.GetCloudgroups(marker, 100)
		if err != nil {
			return nil, errors.Wrap(err, "GetCloudgroups")
		}
		groups = append(groups, part.Groups.Group...)
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

func (self *SAliyunClient) GetCloudgroups(marker string, maxItems int) (*SCloudgroups, error) {
	params := map[string]string{}
	if len(marker) > 0 {
		params["Marker"] = marker
	}
	if maxItems > 0 {
		params["MaxItems"] = fmt.Sprintf("%d", maxItems)
	}
	groups := SCloudgroups{}
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

// https://help.aliyun.com/document_detail/28732.html?spm=a2c4g.11186623.6.777.580735b2m2xUh8
func (self *SAliyunClient) ListGroupPolicies(groupName string) ([]SPolicy, error) {
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

type Cloudusers struct {
	User []SClouduser
}

type SCloudusers struct {
	Users       Cloudusers
	Marker      string
	IsTruncated bool
}

func (self *SAliyunClient) ListGroupUsers(groupName string, marker string, maxItems int) (*SCloudusers, error) {
	params := map[string]string{
		"GroupName": groupName,
	}
	if len(marker) > 0 {
		params["Marker"] = marker
	}
	if maxItems > 0 {
		params["MaxItems"] = fmt.Sprintf("%d", maxItems)
	}
	resp, err := self.ramRequest("ListUsersForGroup", params)
	if err != nil {
		return nil, errors.Wrap(err, "ramRequest.ListUserForGroup")
	}
	users := &SCloudusers{}
	err = resp.Unmarshal(users)
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return users, nil
}

func (self *SAliyunClient) DeleteGroup(groupName string) error {
	params := map[string]string{
		"GroupName": groupName,
	}
	_, err := self.ramRequest("DeleteGroup", params)
	return err
}

func (self *SAliyunClient) CreateGroup(groupName, comments string) (*SCloudgroup, error) {
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
	group := &SCloudgroup{client: self}
	err = resp.Unmarshal(group, "Group")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return group, nil
}

func (self *SAliyunClient) GetGroup(groupName string) (*SCloudgroup, error) {
	params := map[string]string{
		"GroupName": groupName,
	}
	resp, err := self.ramRequest("GetGroup", params)
	if err != nil {
		return nil, errors.Wrap(err, "GetGroup")
	}
	group := &SCloudgroup{client: self}
	err = resp.Unmarshal(group, "Group")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return group, nil
}

func (self *SAliyunClient) GetICloudgroupByName(name string) (cloudprovider.ICloudgroup, error) {
	return self.GetGroup(name)
}

func (self *SAliyunClient) RemoveUserFromGroup(groupName, userName string) error {
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

func (self *SAliyunClient) DetachPolicyFromGroup(groupName, policyName, policyType string) error {
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

func (self *SAliyunClient) CreateICloudgroup(name string, desc string) (cloudprovider.ICloudgroup, error) {
	return self.CreateGroup(name, desc)
}

func (self *SAliyunClient) AddUserToGroup(groupName, userName string) error {
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

func (self *SAliyunClient) AttachGroupPolicy(groupName, policyName, policyType string) error {
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
