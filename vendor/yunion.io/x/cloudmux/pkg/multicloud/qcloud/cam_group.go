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
	"fmt"
	"time"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/cloudid"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SGroup struct {
	client *SQcloudClient

	GroupId    int
	GroupName  string
	CreateTime time.Time
}

func (self *SGroup) GetName() string {
	return self.GroupName
}

func (self *SGroup) GetGlobalId() string {
	return self.GroupName
}

func (self *SGroup) GetDescription() string {
	return ""
}

func (self *SGroup) GetICloudusers() ([]cloudprovider.IClouduser, error) {
	users := []SUser{}
	offset := 1
	for {
		part, total, err := self.client.ListGroupUsers(self.GroupId, offset, 100)
		if err != nil {
			return nil, errors.Wrapf(err, "ListGroupUsers")
		}
		users = append(users, part...)
		if len(users) >= total {
			break
		}
		offset += 1
	}
	ret := []cloudprovider.IClouduser{}
	for i := range users {
		users[i].client = self.client
		ret = append(ret, &users[i])
	}
	return ret, nil
}

func (self *SGroup) AddUser(name string) error {
	user, err := self.client.GetUser(name)
	if err != nil {
		return errors.Wrapf(err, "GetUser(%s)", name)
	}
	return self.client.AddUserToGroup(self.GroupId, int(user.Uid))
}

func (self *SGroup) RemoveUser(name string) error {
	user, err := self.client.GetUser(name)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			return nil
		}
		return errors.Wrapf(err, "GetUser(%s)", name)
	}
	return self.client.RemoveUserFromGroup(self.GroupId, int(user.Uid))
}

func (self *SGroup) AttachPolicy(policyId string, policyType api.TPolicyType) error {
	return self.client.AttachGroupPolicy(self.GroupId, policyId)
}

func (self *SGroup) DetachPolicy(policyId string, policyType api.TPolicyType) error {
	return self.client.DetachGroupPolicy(self.GroupId, policyId)
}

func (self *SGroup) ListGroupPolicies() ([]SPolicy, error) {
	policies := []SPolicy{}
	offset := 1
	for {
		part, total, err := self.client.ListAttachedGroupPolicies(self.GroupId, offset, 200)
		if err != nil {
			return nil, errors.Wrapf(err, "ListAttachedGroupPolicies")
		}
		policies = append(policies, part...)
		if len(policies) >= total {
			break
		}
		offset += 1
	}
	return policies, nil
}

func (self *SGroup) GetICloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	policies, err := self.ListGroupPolicies()
	if err != nil {
		return nil, errors.Wrapf(err, "ListGroupPolicies")
	}
	ret := []cloudprovider.ICloudpolicy{}
	for i := range policies {
		policies[i].client = self.client
		ret = append(ret, &policies[i])
	}
	return ret, nil
}

func (self *SGroup) Delete() error {
	return self.client.DeleteGroup(self.GroupId)
}

func (self *SQcloudClient) GetICloudgroups() ([]cloudprovider.ICloudgroup, error) {
	ret := []cloudprovider.ICloudgroup{}
	offset := 1
	for {
		part, total, err := self.ListGroups("", offset, 50)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeRoleList")
		}
		for i := range part {
			part[i].client = self
			ret = append(ret, &part[i])
		}
		if total >= len(ret) {
			break
		}
		offset += 1
	}
	return ret, nil
}

func (self *SQcloudClient) CreateICloudgroup(name, desc string) (cloudprovider.ICloudgroup, error) {
	group, err := self.CreateGroup(name, desc)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateGroup")
	}
	return group, nil
}

func (self *SQcloudClient) GetICloudgroupByName(name string) (cloudprovider.ICloudgroup, error) {
	groups := []SGroup{}
	offset := 1
	for {
		part, total, err := self.ListGroups(name, offset, 50)
		if err != nil {
			return nil, errors.Wrapf(err, "ListGroups")
		}
		groups = append(groups, part...)
		if len(groups) >= total {
			break
		}
		offset += 1
	}
	for i := range groups {
		if groups[i].GroupName == name {
			groups[i].client = self
			return &groups[i], nil
		}
	}
	return nil, errors.Wrap(cloudprovider.ErrNotFound, name)
}

func (self *SQcloudClient) ListGroups(keyword string, offset int, limit int) ([]SGroup, int, error) {
	if offset < 1 {
		offset = 1
	}
	if limit <= 0 || limit > 50 {
		limit = 50
	}
	params := map[string]string{
		"Page": fmt.Sprintf("%d", offset),
		"Rp":   fmt.Sprintf("%d", limit),
	}
	if len(keyword) > 0 {
		params["Keyword"] = keyword
	}

	resp, err := self.camRequest("ListGroups", params)
	if err != nil {
		return nil, 0, errors.Wrap(err, "camRequest.ListGroups")
	}
	groups := []SGroup{}
	err = resp.Unmarshal(&groups, "GroupInfo")
	if err != nil {
		return nil, 0, errors.Wrap(err, "resp.Unmarshal")
	}
	total, _ := resp.Float("TotalNum")
	return groups, int(total), nil
}

func (self *SQcloudClient) CreateGroup(name string, remark string) (*SGroup, error) {
	params := map[string]string{
		"GroupName": name,
	}
	if len(remark) > 0 {
		params["Remark"] = remark
	}
	resp, err := self.camRequest("CreateGroup", params)
	if err != nil {
		return nil, errors.Wrap(err, "camRequest.CreateGroup")
	}
	groupId, _ := resp.Float("GroupId")
	return self.GetGroup(int(groupId))
}

func (self *SQcloudClient) GetGroup(groupId int) (*SGroup, error) {
	params := map[string]string{
		"GroupId": fmt.Sprintf("%d", groupId),
	}
	resp, err := self.camRequest("GetGroup", params)
	if err != nil {
		return nil, errors.Wrap(err, "camRequest.GetGroup")
	}
	group := &SGroup{client: self}
	err = resp.Unmarshal(group)
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return group, nil
}

func (self *SQcloudClient) DeleteGroup(id int) error {
	params := map[string]string{
		"GroupId": fmt.Sprintf("%d", id),
	}
	_, err := self.camRequest("DeleteGroup", params)
	return err
}

func (self *SQcloudClient) ListAttachedGroupPolicies(groupId int, offset int, limit int) ([]SPolicy, int, error) {
	if offset < 1 {
		offset = 1
	}
	if limit <= 0 || limit > 50 {
		limit = 50
	}
	params := map[string]string{
		"TargetGroupId": fmt.Sprintf("%d", groupId),
		"Page":          fmt.Sprintf("%d", offset),
		"Rp":            fmt.Sprintf("%d", limit),
	}
	resp, err := self.camRequest("ListAttachedGroupPolicies", params)
	if err != nil {
		return nil, 0, errors.Wrap(err, "iamRequest.ListAttachedGroupPolicies")
	}
	policies := []SPolicy{}
	err = resp.Unmarshal(&policies, "List")
	if err != nil {
		return nil, 0, errors.Wrap(err, "resp.Unmarshal")
	}
	total, _ := resp.Float("TotalNum")
	return policies, int(total), nil
}

func (self *SQcloudClient) ListGroupUsers(groupId int, offset int, limit int) ([]SUser, int, error) {
	if offset < 1 {
		offset = 1
	}
	if limit <= 0 || limit > 50 {
		limit = 50
	}
	params := map[string]string{
		"GroupId": fmt.Sprintf("%d", groupId),
		"Page":    fmt.Sprintf("%d", offset),
		"Rp":      fmt.Sprintf("%d", limit),
	}
	resp, err := self.camRequest("ListUsersForGroup", params)
	if err != nil {
		return nil, 0, errors.Wrap(err, "iamRequest.ListUserForGroup")
	}
	users := []SUser{}
	err = resp.Unmarshal(&users, "UserInfo")
	if err != nil {
		return nil, 0, errors.Wrap(err, "resp.Unmarshal")
	}
	total, _ := resp.Float("TotalNum")
	return users, int(total), nil
}

func (self *SQcloudClient) RemoveUserFromGroup(groupId, userId int) error {
	params := map[string]string{
		"Info.0.Uid":     fmt.Sprintf("%d", userId),
		"Info.0.GroupId": fmt.Sprintf("%d", groupId),
	}
	_, err := self.camRequest("RemoveUserFromGroup", params)
	return err
}

func (self *SQcloudClient) AddUserToGroup(groupId, userId int) error {
	params := map[string]string{
		"Info.0.Uid":     fmt.Sprintf("%d", userId),
		"Info.0.GroupId": fmt.Sprintf("%d", groupId),
	}
	_, err := self.camRequest("AddUserToGroup", params)
	return err
}

func (self *SQcloudClient) AttachGroupPolicy(groupId int, policyId string) error {
	params := map[string]string{
		"PolicyId":      policyId,
		"AttachGroupId": fmt.Sprintf("%d", groupId),
	}
	_, err := self.camRequest("AttachGroupPolicy", params)
	return err
}

func (self *SQcloudClient) DetachGroupPolicy(groupId int, policyId string) error {
	params := map[string]string{
		"PolicyId":      policyId,
		"DetachGroupId": fmt.Sprintf("%d", groupId),
	}
	_, err := self.camRequest("DetachGroupPolicy", params)
	return err
}
