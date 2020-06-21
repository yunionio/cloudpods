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

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SCloudgroup struct {
	client *SQcloudClient

	GroupId    int
	GroupName  string
	CreateTime time.Time
}

func (self *SCloudgroup) GetGlobalId() string {
	return fmt.Sprintf("%d", self.GroupId)
}

func (self *SCloudgroup) GetName() string {
	return self.GroupName
}

func (self *SCloudgroup) GetDescription() string {
	return ""
}

func (self *SCloudgroup) Delete() error {
	return self.client.DeleteGroup(self.GroupId)
}

func (self *SCloudgroup) AddUser(name string) error {
	user, err := self.client.GetIClouduserByName(name)
	if err != nil {
		return errors.Wrapf(err, "GetIClouduserByName(%s)", name)
	}
	_user := user.(*SClouduser)
	return self.client.AddUserToGroup(self.GroupId, int(_user.Uid))
}

func (self *SCloudgroup) RemoveUser(name string) error {
	user, err := self.client.GetIClouduserByName(name)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			return nil
		}
		return errors.Wrapf(err, "GetIClouduserByName(%s)", name)
	}
	_user := user.(*SClouduser)
	return self.client.RemoveUserFromGroup(self.GroupId, int(_user.Uid))
}

func (self *SCloudgroup) AttachSystemPolicy(policyId string) error {
	return self.client.AttachGroupPolicy(self.GroupId, policyId)
}

func (self *SCloudgroup) DetachSystemPolicy(policyId string) error {
	return self.client.DetachGroupPolicy(self.GroupId, policyId)
}

func (self *SCloudgroup) GetISystemCloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	policies := []SClouduserPolicy{}
	page := 1
	for {
		part, total, err := self.client.ListAttachedGroupPolicies(self.GroupId, page, 50)
		if err != nil {
			return nil, errors.Wrap(err, "ListAttachedGroupPolicies")
		}
		policies = append(policies, part...)
		if len(policies) >= total {
			break
		}
		page += 1
	}
	ret := []cloudprovider.ICloudpolicy{}
	for i := range policies {
		if policies[i].PolicyType == "QCS" {
			policies[i].client = self.client
			ret = append(ret, &policies[i])
		}
	}
	return ret, nil
}

func (self *SCloudgroup) GetICloudusers() ([]cloudprovider.IClouduser, error) {
	users := []SClouduser{}
	page := 1
	for {
		part, total, err := self.client.ListGroupUsers(self.GroupId, page, 50)
		if err != nil {
			return nil, errors.Wrap(err, "ListGroupUsers")
		}
		users = append(users, part...)
		if len(users) >= total {
			break
		}
		page += 1
	}
	ret := []cloudprovider.IClouduser{}
	for i := range users {
		users[i].client = self.client
		ret = append(ret, &users[i])
	}
	return ret, nil
}

func (self *SQcloudClient) GetICloudgroups() ([]cloudprovider.ICloudgroup, error) {
	groups := []SCloudgroup{}
	page := 1
	for {
		part, total, err := self.ListGroups("", page, 50)
		if err != nil {
			return nil, errors.Wrap(err, "ListGroups")
		}
		groups = append(groups, part...)
		if len(groups) >= total {
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

func (self *SQcloudClient) ListGroups(keyword string, page int, rp int) ([]SCloudgroup, int, error) {
	if page < 1 {
		page = 1
	}
	if rp <= 0 || rp > 50 {
		rp = 50
	}
	params := map[string]string{
		"Page": fmt.Sprintf("%d", page),
		"Rp":   fmt.Sprintf("%d", rp),
	}
	if len(keyword) > 0 {
		params["keyword"] = keyword
	}

	resp, err := self.camRequest("ListGroups", params)
	if err != nil {
		return nil, 0, errors.Wrap(err, "camRequest.ListGroups")
	}
	groups := []SCloudgroup{}
	err = resp.Unmarshal(&groups, "GroupInfo")
	if err != nil {
		return nil, 0, errors.Wrap(err, "resp.Unmarshal")
	}
	total, _ := resp.Int("TotalNum")
	return groups, int(total), nil
}

func (self *SQcloudClient) CreateGroup(name string, remark string) (*SCloudgroup, error) {
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

func (self *SQcloudClient) GetGroup(groupId int) (*SCloudgroup, error) {
	params := map[string]string{
		"GroupId": fmt.Sprintf("%d", groupId),
	}
	resp, err := self.camRequest("GetGroup", params)
	if err != nil {
		return nil, errors.Wrap(err, "camRequest.GetGroup")
	}
	group := &SCloudgroup{client: self}
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

func (self *SQcloudClient) GetICloudgroupByName(name string) (cloudprovider.ICloudgroup, error) {
	groups, err := self.GetICloudgroups()
	if err != nil {
		return nil, errors.Wrap(err, "GetICloudgroups")
	}
	for i := range groups {
		if groups[i].GetName() == name {
			return groups[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SQcloudClient) ListAttachedGroupPolicies(groupId int, page int, rp int) ([]SClouduserPolicy, int, error) {
	if page < 1 {
		page = 1
	}
	if rp <= 0 || rp > 50 {
		rp = 50
	}
	params := map[string]string{
		"TargetGroupId": fmt.Sprintf("%d", groupId),
		"Page":          fmt.Sprintf("%d", page),
		"Rp":            fmt.Sprintf("%d", rp),
	}
	resp, err := self.camRequest("ListAttachedGroupPolicies", params)
	if err != nil {
		return nil, 0, errors.Wrap(err, "iamRequest.ListAttachedGroupPolicies")
	}
	policies := []SClouduserPolicy{}
	err = resp.Unmarshal(&policies, "List")
	if err != nil {
		return nil, 0, errors.Wrap(err, "resp.Unmarshal")
	}
	total, _ := resp.Int("TotalNum")
	return policies, int(total), nil
}

func (self *SQcloudClient) ListGroupUsers(groupId int, page int, rp int) ([]SClouduser, int, error) {
	if page < 1 {
		page = 1
	}
	if rp <= 0 || rp > 50 {
		rp = 50
	}
	params := map[string]string{
		"GroupId": fmt.Sprintf("%d", groupId),
		"Page":    fmt.Sprintf("%d", page),
		"Rp":      fmt.Sprintf("%d", rp),
	}
	resp, err := self.camRequest("ListUsersForGroup", params)
	if err != nil {
		return nil, 0, errors.Wrap(err, "iamRequest.ListUserForGroup")
	}
	users := []SClouduser{}
	err = resp.Unmarshal(&users, "UserInfo")
	if err != nil {
		return nil, 0, errors.Wrap(err, "resp.Unmarshal")
	}
	total, _ := resp.Int("TotalNum")
	return users, int(total), nil
}

func (self *SQcloudClient) CreateICloudgroup(name, desc string) (cloudprovider.ICloudgroup, error) {
	group, err := self.CreateGroup(name, desc)
	if err != nil {
		return nil, errors.Wrap(err, "CreateGroup")
	}
	return group, nil
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
