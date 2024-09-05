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

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/cloudid"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SUsers struct {
	Users       []SUser `xml:"Users>member"`
	IsTruncated bool    `xml:"IsTruncated"`
	Marker      string  `xml:"Marker"`
}

type SUser struct {
	client *SAwsClient
	multicloud.SBaseClouduser

	UserId           string    `xml:"UserId"`
	Path             string    `xml:"Path"`
	UserName         string    `xml:"UserName"`
	Arn              string    `xml:"Arn"`
	CreateDate       time.Time `xml:"CreateDate"`
	PasswordLastUsed time.Time `xml:"PasswordLastUsed"`
}

func (user *SUser) GetEmailAddr() string {
	return ""
}

func (user *SUser) GetInviteUrl() string {
	return ""
}

func (user *SUser) AttachPolicy(policyArn string, policyType api.TPolicyType) error {
	return user.client.AttachUserPolicy(user.UserName, policyArn)
}

func (user *SUser) DetachPolicy(policyArn string, policyType api.TPolicyType) error {
	return user.client.DetachUserPolicy(user.UserName, policyArn)
}

func (user *SUser) GetGlobalId() string {
	return user.UserId
}

func (user *SUser) GetName() string {
	return user.UserName
}

func (user *SUser) ResetPassword(password string) error {
	return user.client.ResetUserPassword(user.UserName, password)
}

func (user *SUser) IsConsoleLogin() bool {
	_, err := user.client.GetLoginProfile(user.UserName)
	if errors.Cause(err) == cloudprovider.ErrNotFound {
		return false
	}
	return true
}

func (user *SUser) GetICloudgroups() ([]cloudprovider.ICloudgroup, error) {
	groups, err := user.ListGroups()
	if err != nil {
		return nil, errors.Wrapf(err, "ListGroups")
	}
	ret := []cloudprovider.ICloudgroup{}
	for i := range groups {
		groups[i].client = user.client
		ret = append(ret, &groups[i])
	}
	return ret, nil
}

func (self *SUser) ListPolicies() ([]SAttachedPolicy, error) {
	policies := []SAttachedPolicy{}
	offset := ""
	for {
		part, err := self.client.ListAttachedUserPolicies(self.UserName, offset, 1000, "")
		if err != nil {
			return nil, errors.Wrap(err, "ListAttachedUserPolicies")
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

func (self *SUser) GetICloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
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

func (user *SUser) ListGroups() ([]SGroup, error) {
	groups := []SGroup{}
	offset := ""
	for {
		part, err := user.client.ListGroupsForUser(user.UserName, offset, 1000)
		if err != nil {
			return nil, errors.Wrap(err, "ListGroupsForUser")
		}
		groups = append(groups, part.Groups...)
		offset = part.Marker
		if len(offset) == 0 || !part.IsTruncated {
			break
		}
	}
	return groups, nil
}

func (user *SUser) Delete() error {
	groups, err := user.ListGroups()
	if err != nil {
		return errors.Wrapf(err, "ListGroups")
	}
	for _, group := range groups {
		err = user.client.RemoveUserFromGroup(group.GroupName, user.UserName)
		if err != nil {
			return errors.Wrap(err, "RemoveUserFromGroup")
		}
	}
	policies, err := user.ListPolicies()
	if err != nil {
		return errors.Wrapf(err, "ListPolicies")
	}
	for _, policy := range policies {
		err = user.client.DetachUserPolicy(user.UserName, policy.PolicyArn)
		if err != nil {
			return errors.Wrap(err, "DetachPolicy")
		}
	}
	return user.client.DeleteUser(user.UserName)
}

func (self *SAwsClient) ListUsers(offset string, limit int, pathPrefix string) (*SUsers, error) {
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
	users := &SUsers{}
	err := self.iamRequest("ListUsers", params, users)
	if err != nil {
		return nil, errors.Wrap(err, "iamRequest.ListUsers")
	}
	return users, nil
}

func (self *SAwsClient) CreateUser(path string, username string) (*SUser, error) {
	params := map[string]string{
		"UserName": username,
	}
	if len(path) > 0 {
		params["Path"] = path
	}
	user := struct {
		User SUser `xml:"User"`
	}{}
	err := self.iamRequest("CreateUser", params, &user)
	if err != nil {
		return nil, errors.Wrap(err, "iamRequest.CreateUser")
	}
	user.User.client = self
	return &user.User, nil
}

func (self *SAwsClient) DeleteUser(name string) error {
	self.DeleteLoginProfile(name)
	params := map[string]string{
		"UserName": name,
	}
	return self.iamRequest("DeleteUser", params, nil)
}

func (self *SAwsClient) AttachUserPolicy(userName string, policyArn string) error {
	params := map[string]string{
		"PolicyArn": policyArn,
		"UserName":  userName,
	}
	return self.iamRequest("AttachUserPolicy", params, nil)
}

func (self *SAwsClient) DetachUserPolicy(userName string, policyArn string) error {
	params := map[string]string{
		"PolicyArn": policyArn,
		"UserName":  userName,
	}
	err := self.iamRequest("DetachUserPolicy", params, nil)
	if err != nil && errors.Cause(err) != cloudprovider.ErrNotFound {
		return errors.Wrap(err, "DetachUserPolicy")
	}
	return nil
}

func (self *SAwsClient) CreateIClouduser(conf *cloudprovider.SClouduserCreateConfig) (cloudprovider.IClouduser, error) {
	user, err := self.CreateUser("", conf.Name)
	if err != nil {
		return nil, errors.Wrap(err, "CreateUser")
	}
	if len(conf.Password) > 0 {
		_, err := self.CreateLoginProfile(conf.Name, conf.Password)
		if err != nil {
			log.Errorf("failed to create loginProfile for user %s error: %v", conf.Name, err)
		}
	}
	return user, nil
}

func (self *SAwsClient) GetIClouduserByName(name string) (cloudprovider.IClouduser, error) {
	return self.GetUser(name)
}

func (self *SAwsClient) GetUser(name string) (*SUser, error) {
	user := struct {
		User SUser `xml:"User"`
	}{}
	params := map[string]string{
		"UserName": name,
	}
	err := self.iamRequest("GetUser", params, &user)
	if err != nil {
		return nil, errors.Wrap(err, "iamRequest.GetUser")
	}
	user.User.client = self
	return &user.User, nil
}

func (self *SAwsClient) GetICloudusers() ([]cloudprovider.IClouduser, error) {
	ret := []cloudprovider.IClouduser{}
	offset := ""
	for {
		part, err := self.ListUsers(offset, 1000, "")
		if err != nil {
			return nil, errors.Wrap(err, "ListUsers")
		}
		for i := range part.Users {
			part.Users[i].client = self
			ret = append(ret, &part.Users[i])
		}
		offset = part.Marker
		if len(offset) == 0 || !part.IsTruncated {
			break
		}
	}
	return ret, nil
}

type LoginProfile struct {
	UserName   string    `xml:"UserName"`
	CreateDate time.Time `xml:"CreateDate"`
}
type SLoginProfile struct {
	LoginProfile LoginProfile `xml:"LoginProfile"`
}

func (self *SAwsClient) GetLoginProfile(name string) (*SLoginProfile, error) {
	params := map[string]string{
		"UserName": name,
	}
	loginProfix := &SLoginProfile{}
	err := self.iamRequest("GetLoginProfile", params, loginProfix)
	if err != nil {
		return nil, errors.Wrap(err, "iamRequest.GetLoginProfie")
	}
	return loginProfix, nil
}

func (self *SAwsClient) DeleteLoginProfile(name string) error {
	params := map[string]string{
		"UserName": name,
	}
	return self.iamRequest("DeleteLoginProfile", params, nil)
}

func (self *SAwsClient) CreateLoginProfile(name, password string) (*SLoginProfile, error) {
	params := map[string]string{
		"UserName": name,
		"Password": password,
	}
	loginProfile := &SLoginProfile{}
	err := self.iamRequest("CreateLoginProfile", params, loginProfile)
	if err != nil {
		return nil, errors.Wrap(err, "iamRequest.GetLoginProfie")
	}
	return loginProfile, nil
}

func (self *SAwsClient) UpdateLoginProfile(name, password string) error {
	params := map[string]string{
		"UserName": name,
		"Password": password,
	}
	return self.iamRequest("UpdateLoginProfile", params, nil)
}

func (self *SAwsClient) ResetUserPassword(name, password string) error {
	_, err := self.GetLoginProfile(name)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			_, err = self.CreateLoginProfile(name, password)
			return err
		}
		return errors.Wrap(err, "GetLoginProfile")
	}
	return self.UpdateLoginProfile(name, password)
}

func (self *SAwsClient) ListGroupsForUser(name string, offset string, limit int) (*SGroups, error) {
	if limit < 1 || limit > 1000 {
		limit = 1000
	}
	params := map[string]string{
		"UserName": name,
		"MaxItems": fmt.Sprintf("%d", limit),
	}
	if len(offset) > 0 {
		params["Marker"] = offset
	}
	groups := &SGroups{}
	err := self.iamRequest("ListGroupsForUser", params, groups)
	if err != nil {
		return nil, errors.Wrap(err, "ListGroupsForUser")
	}
	return groups, nil
}
