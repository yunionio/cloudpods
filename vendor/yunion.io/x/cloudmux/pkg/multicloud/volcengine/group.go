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

package volcengine

import (
	"fmt"

	api "yunion.io/x/cloudmux/pkg/apis/cloudid"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/utils"
)

type SGroup struct {
	client *SVolcEngineClient

	Description   string
	CreatedDate   string
	UserGroupName string
	UpdateDate    string
	AccountId     string
	UserGroupId   string
}

func (self *SGroup) GetName() string {
	return self.UserGroupName
}

func (self *SGroup) GetGlobalId() string {
	return self.UserGroupName
}

func (self *SGroup) GetDescription() string {
	return self.Description
}

func (self *SGroup) GetICloudusers() ([]cloudprovider.IClouduser, error) {
	users, err := self.client.ListUsersForGroup(self.UserGroupName)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.IClouduser{}
	for i := range users {
		users[i].client = self.client
		ret = append(ret, &users[i])
	}
	return ret, nil
}

func (self *SGroup) GetICloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	policies, err := self.client.ListAttachedUserGroupPolicies(self.UserGroupName)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudpolicy{}
	for i := range policies {
		policies[i].client = self.client
		ret = append(ret, &policies[i])
	}
	return ret, nil
}

func (self *SGroup) AddUser(name string) error {
	return self.client.AddUserToGroup(name, self.UserGroupName)
}

func (self *SGroup) RemoveUser(name string) error {
	return self.client.RemoveUserFromGroup(name, self.UserGroupName)
}

func (self *SGroup) AttachPolicy(policyName string, policyType api.TPolicyType) error {
	return self.client.AttachUserGroupPolicy(self.UserGroupName, policyName, utils.Capitalize(string(policyType)))
}

func (self *SGroup) DetachPolicy(policyName string, policyType api.TPolicyType) error {
	return self.client.DetachUserGroupPolicy(self.UserGroupName, policyName, utils.Capitalize(string(policyType)))
}

func (self *SGroup) Delete() error {
	return self.client.DeleteGroup(self.UserGroupName)
}

func (self *SVolcEngineClient) CreateICloudgroup(name string, desc string) (cloudprovider.ICloudgroup, error) {
	group, err := self.CreateGroup(name, desc)
	if err != nil {
		return nil, err
	}
	return group, nil
}

func (self *SVolcEngineClient) GetICloudgroups() ([]cloudprovider.ICloudgroup, error) {
	groups, err := self.ListGroups()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudgroup{}
	for i := range groups {
		groups[i].client = self
		ret = append(ret, &groups[i])
	}
	return ret, nil
}

func (client *SVolcEngineClient) ListGroups() ([]SGroup, error) {
	params := map[string]string{
		"Limit": "50",
	}
	offset := 0
	ret := []SGroup{}
	for {
		params["Offset"] = fmt.Sprintf("%d", offset)
		resp, err := client.iamRequest("", "ListGroups", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			UserGroups []SGroup
			Total      int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.UserGroups...)
		if len(part.UserGroups) == 0 || len(ret) >= part.Total {
			break
		}
		offset = len(ret)
	}
	return ret, nil
}

func (client *SVolcEngineClient) ListUsersForGroup(name string) ([]SUser, error) {
	params := map[string]string{
		"Limit":         "50",
		"UserGroupName": name,
	}
	offset := 0
	ret := []SUser{}
	for {
		params["Offset"] = fmt.Sprintf("%d", offset)
		resp, err := client.iamRequest("", "ListUsersForGroup", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			Users []SUser
			Total int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.Users...)
		if len(part.Users) == 0 || len(ret) >= part.Total {
			break
		}
		offset = len(ret)
	}
	return ret, nil
}

func (client *SVolcEngineClient) ListAttachedUserGroupPolicies(name string) ([]SPolicy, error) {
	params := map[string]string{
		"UserGroupName": name,
	}
	resp, err := client.iamRequest("", "ListAttachedUserGroupPolicies", params)
	if err != nil {
		return nil, err
	}
	ret := []SPolicy{}
	err = resp.Unmarshal(&ret, "AttachedPolicyMetadata")
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (client *SVolcEngineClient) AttachUserGroupPolicy(name, policy, policyType string) error {
	params := map[string]string{
		"UserGroupName": name,
		"PolicyName":    policy,
		"PolicyType":    policyType,
	}
	_, err := client.iamRequest("", "AttachUserGroupPolicy", params)
	return err
}

func (client *SVolcEngineClient) DetachUserGroupPolicy(name, policy, policyType string) error {
	params := map[string]string{
		"UserGroupName": name,
		"PolicyName":    policy,
		"PolicyType":    policyType,
	}
	_, err := client.iamRequest("", "DetachUserGroupPolicy", params)
	return err
}

func (client *SVolcEngineClient) DeleteGroup(name string) error {
	params := map[string]string{
		"UserGroupName": name,
	}
	_, err := client.iamRequest("", "DeleteGroup", params)
	return err
}

func (client *SVolcEngineClient) AddUserToGroup(user, group string) error {
	params := map[string]string{
		"UserGroupName": group,
		"UserName":      user,
	}
	_, err := client.iamRequest("", "AddUserToGroup", params)
	return err
}

func (client *SVolcEngineClient) RemoveUserFromGroup(user, group string) error {
	params := map[string]string{
		"UserGroupName": group,
		"UserName":      user,
	}
	_, err := client.iamRequest("", "RemoveUserFromGroup", params)
	return err
}

func (client *SVolcEngineClient) CreateGroup(name, desc string) (*SGroup, error) {
	params := map[string]string{
		"UserGroupName": name,
		"Description":   desc,
	}
	resp, err := client.iamRequest("", "CreateGroup", params)
	if err != nil {
		return nil, err
	}
	ret := &SGroup{client: client}
	err = resp.Unmarshal(ret, "UserGroup")
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (client *SVolcEngineClient) GetICloudgroupByName(name string) (cloudprovider.ICloudgroup, error) {
	group, err := client.GetGroup(name)
	if err != nil {
		return nil, err
	}
	return group, nil
}

func (client *SVolcEngineClient) GetGroup(name string) (*SGroup, error) {
	params := map[string]string{
		"UserGroupName": name,
	}
	resp, err := client.iamRequest("", "GetGroup", params)
	if err != nil {
		return nil, err
	}
	ret := &SGroup{client: client}
	err = resp.Unmarshal(ret, "UserGroup")
	if err != nil {
		return nil, err
	}
	return ret, nil
}
