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

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/cloudid"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SUser struct {
	multicloud.SBaseClouduser
	client *SQcloudClient

	Uin          int64
	Name         string
	Uid          int64
	Remark       string
	ConsoleLogin int
	CountryCode  string
	Email        string
}

func (user *SUser) GetGlobalId() string {
	return fmt.Sprintf("%d", user.Uin)
}

func (self *SUser) GetEmailAddr() string {
	return self.Email
}

func (self *SUser) GetInviteUrl() string {
	return ""
}

func (user *SUser) GetICloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	policies := []SPolicy{}
	offset := 1
	for {
		part, total, err := user.client.ListAttachedUserPolicies(user.GetGlobalId(), offset, 50)
		if err != nil {
			return nil, errors.Wrap(err, "GetClouduserPolicy")
		}
		policies = append(policies, part...)
		if len(policies) >= total {
			break
		}
		offset += 1
	}
	ret := []cloudprovider.ICloudpolicy{}
	for i := range policies {
		policies[i].client = user.client
		ret = append(ret, &policies[i])
	}
	return ret, nil
}

func (user *SUser) AttachPolicy(policyId string, policyType api.TPolicyType) error {
	return user.client.AttachUserPolicy(user.GetGlobalId(), policyId)
}

func (user *SUser) DetachPolicy(policyId string, policyType api.TPolicyType) error {
	return user.client.DetachUserPolicy(user.GetGlobalId(), policyId)
}

func (user *SUser) IsConsoleLogin() bool {
	return user.ConsoleLogin == 1
}

func (user *SUser) Delete() error {
	return user.client.DeleteUser(user.Name)
}

func (user *SUser) ResetPassword(password string) error {
	return user.client.UpdateUser(user.Name, password)
}

func (user *SUser) GetICloudgroups() ([]cloudprovider.ICloudgroup, error) {
	ret := []cloudprovider.ICloudgroup{}
	offset := 1
	for {
		part, total, err := user.client.ListGroupsForUser(int(user.Uin), offset, 100)
		if err != nil {
			return nil, errors.Wrapf(err, "ListGroupsForUser")
		}
		for i := range part {
			part[i].client = user.client
			ret = append(ret, &part[i])
		}
		if len(ret) >= total {
			break
		}
		offset += 1
	}
	return ret, nil
}

func (user *SUser) GetName() string {
	return user.Name
}

func (self *SQcloudClient) DeleteUser(name string) error {
	params := map[string]string{
		"Name":  name,
		"Force": "1",
	}
	_, err := self.camRequest("DeleteUser", params)
	return err
}

func (self *SQcloudClient) ListUsers() ([]SUser, error) {
	resp, err := self.camRequest("ListUsers", nil)
	if err != nil {
		return nil, errors.Wrap(err, "camRequest.ListUsers")
	}
	users := []SUser{}
	err = resp.Unmarshal(&users, "Data")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return users, nil
}

func (self *SQcloudClient) ListGroupsForUser(uin int, offset, limit int) ([]SGroup, int, error) {
	if offset < 1 {
		offset = 1
	}
	if limit <= 0 || limit > 50 {
		limit = 50
	}
	params := map[string]string{
		"SubUin": fmt.Sprintf("%d", uin),
		"Page":   fmt.Sprintf("%d", offset),
		"Rp":     fmt.Sprintf("%d", limit),
	}
	resp, err := self.camRequest("ListGroupsForUser", params)
	if err != nil {
		return nil, 0, errors.Wrap(err, "camRequest.ListGroupsForUser")
	}
	groups := []SGroup{}
	err = resp.Unmarshal(&groups, "GroupInfo")
	if err != nil {
		return nil, 0, errors.Wrap(err, "resp.Unmarshal")
	}
	total, _ := resp.Float("TotalNum")
	return groups, int(total), nil
}

func (self *SQcloudClient) GetICloudusers() ([]cloudprovider.IClouduser, error) {
	users, err := self.ListUsers()
	if err != nil {
		return nil, errors.Wrap(err, "ListUsers")
	}

	ret := []cloudprovider.IClouduser{}
	for i := range users {
		users[i].client = self
		ret = append(ret, &users[i])
	}
	collaborators := []SUser{}
	for {
		part, total, err := self.ListCollaborators(len(collaborators), 50)
		if err != nil {
			return nil, errors.Wrapf(err, "ListCollaborators")
		}
		collaborators = append(collaborators, part...)
		if len(collaborators) >= total {
			break
		}
	}
	for i := range collaborators {
		collaborators[i].client = self
		ret = append(ret, &collaborators[i])
	}
	return ret, nil
}

func (self *SQcloudClient) GetIClouduserByName(name string) (cloudprovider.IClouduser, error) {
	return self.GetUser(name)
}

func (self *SQcloudClient) GetUser(name string) (*SUser, error) {
	params := map[string]string{
		"Name": name,
	}
	resp, err := self.camRequest("GetUser", params)
	if err != nil {
		return nil, errors.Wrap(err, "camRequest.GetUser")
	}
	user := &SUser{client: self}
	err = resp.Unmarshal(user)
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return user, nil
}

func (self *SQcloudClient) CreateIClouduser(conf *cloudprovider.SClouduserCreateConfig) (cloudprovider.IClouduser, error) {
	user, err := self.AddUser(conf.Name, conf.Password, conf.Desc, conf.IsConsoleLogin)
	if err != nil {
		return nil, errors.Wrap(err, "CreateClouduser")
	}
	return user, nil
}

func (self *SQcloudClient) AddUser(name, password, desc string, consoleLogin bool) (*SUser, error) {
	params := map[string]string{
		"Name":         name,
		"Remark":       desc,
		"ConsoleLogin": "0",
	}
	if len(password) > 0 {
		params["Password"] = password
	}
	if consoleLogin {
		params["ConsoleLogin"] = "1"
	}
	resp, err := self.camRequest("AddUser", params)
	if err != nil {
		return nil, errors.Wrap(err, "camRequest.AddUser")
	}
	user := &SUser{client: self}
	err = resp.Unmarshal(user)
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return user, nil
}

func (self *SQcloudClient) UpdateUser(name, password string) error {
	params := map[string]string{
		"Name":         name,
		"ConsoleLogin": "1",
		"Password":     password,
	}
	_, err := self.camRequest("UpdateUser", params)
	if err != nil {
		return errors.Wrap(err, "UpdateUser")
	}
	return nil
}
