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
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/cloudid"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type sUsers struct {
	User []SUser
}

type SUsers struct {
	Users       sUsers
	Marker      string
	IsTruncated bool
}

type SUser struct {
	client *SApsaraClient
	multicloud.SBaseClouduser

	Comments    string
	CreateDate  time.Time
	DisplayName string
	Email       string
	MobilePhone string
	UserId      string
	UserName    string
}

func (user *SUser) GetEmailAddr() string {
	return ""
}

func (user *SUser) GetInviteUrl() string {
	return ""
}

func (user *SUser) GetGlobalId() string {
	if len(user.UserId) > 0 {
		return user.UserId
	}
	u, err := user.client.GetUser(user.UserName)
	if err != nil {
		return ""
	}
	return u.UserId
}

func (user *SUser) GetName() string {
	return user.UserName
}

func (user *SUser) Delete() error {
	groups, err := user.client.ListGroupsForUser(user.UserName)
	if err != nil {
		return errors.Wrap(err, "ListGroupsForUser")
	}
	for i := range groups {
		err = user.client.RemoveUserFromGroup(groups[i].GroupName, user.UserName)
		if err != nil {
			return errors.Wrapf(err, "RemoveUserFromGroup %s > %s", groups[i].GroupName, user.UserName)
		}
	}
	policies, err := user.client.ListPoliciesForUser(user.UserName)
	if err != nil {
		return errors.Wrap(err, "ListPoliciesForUser")
	}
	for i := range policies {
		err = user.client.DetachPolicyFromUser(policies[i].PolicyName, policies[i].PolicyType, user.UserName)
		if err != nil {
			return errors.Wrapf(err, "DetachPolicyFromUser %s %s %s", policies[i].PolicyName, policies[i].PolicyType, user.UserName)
		}
	}
	return user.client.DeleteClouduser(user.UserName)
}

func (user *SUser) GetICloudgroups() ([]cloudprovider.ICloudgroup, error) {
	groups, err := user.client.ListGroupsForUser(user.UserName)
	if err != nil {
		return nil, errors.Wrapf(err, "ListGroupsForUser")
	}
	ret := []cloudprovider.ICloudgroup{}
	for i := range groups {
		groups[i].client = user.client
		ret = append(ret, &groups[i])
	}
	return ret, nil
}

func (user *SUser) UpdatePassword(password string) error {
	return user.client.UpdateLoginProfile(user.UserName, password)
}

func (user *SUser) GetICloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	policies, err := user.client.ListPoliciesForUser(user.UserName)
	if err != nil {
		return nil, errors.Wrap(err, "ListPoliciesForUser")
	}
	ret := []cloudprovider.ICloudpolicy{}
	for i := range policies {
		policies[i].client = user.client
		ret = append(ret, &policies[i])
	}
	return ret, nil
}

func (user *SUser) IsConsoleLogin() bool {
	_, err := user.client.GetLoginProfile(user.UserName)
	if errors.Cause(err) == cloudprovider.ErrNotFound {
		return false
	}
	return true
}

func (user *SUser) ResetPassword(password string) error {
	return user.client.ResetClouduserPassword(user.UserName, password)
}

func (user *SUser) AttachPolicy(policyName string, policyType api.TPolicyType) error {
	return user.client.AttachPolicyToUser(policyName, utils.Capitalize(string(policyType)), user.UserName)
}

func (user *SUser) DetachPolicy(policyName string, policyType api.TPolicyType) error {
	return user.client.DetachPolicyFromUser(policyName, utils.Capitalize(string(policyType)), user.UserName)
}

func (self *SApsaraClient) DeleteClouduser(name string) error {
	params := map[string]string{
		"UserName": name,
	}
	_, err := self.ramRequest("DeleteUser", params)
	return err
}

func (self *SApsaraClient) CreateUser(name, phone, email, comments string) (*SUser, error) {
	params := map[string]string{
		"UserName":    name,
		"DisplayName": name,
	}
	if len(phone) > 0 {
		params["MobilePhone"] = phone
	}
	if len(email) > 0 {
		params["Email"] = email
	}
	if len(comments) > 0 {
		params["Comments"] = comments
	}
	resp, err := self.ramRequest("CreateUser", params)
	if err != nil {
		return nil, errors.Wrap(err, "ramRequest.CreateUser")
	}

	user := &SUser{client: self}
	err = resp.Unmarshal(user, "User")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return user, nil
}

func (self *SApsaraClient) ListUsers(offset string, limit int) (*SUsers, error) {
	params := map[string]string{}
	if len(offset) > 0 {
		params["Marker"] = offset
	}
	if limit > 0 {
		params["MaxItems"] = fmt.Sprintf("%d", limit)
	}
	resp, err := self.ramRequest("ListUsers", params)
	if err != nil {
		return nil, errors.Wrap(err, "ramRequest.ListUsers")
	}
	users := &SUsers{}
	err = resp.Unmarshal(users)
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return users, nil
}

func (self *SApsaraClient) CreateIClouduser(conf *cloudprovider.SClouduserCreateConfig) (cloudprovider.IClouduser, error) {
	user, err := self.CreateUser(conf.Name, conf.MobilePhone, conf.Email, conf.Desc)
	if err != nil {
		return nil, errors.Wrap(err, "CreateUser")
	}
	if len(conf.Password) > 0 {
		_, err := self.CreateLoginProfile(conf.Name, conf.Password)
		if err != nil {
			return nil, errors.Wrap(err, "CreateLoginProfile")
		}
	}
	return user, nil
}

func (self *SApsaraClient) GetICloudusers() ([]cloudprovider.IClouduser, error) {
	ret := []cloudprovider.IClouduser{}
	offset := ""
	for {
		part, err := self.ListUsers(offset, 100)
		if err != nil {
			return nil, errors.Wrap(err, "GetCloudusers")
		}
		for i := range part.Users.User {
			part.Users.User[i].client = self
			ret = append(ret, &part.Users.User[i])
		}
		offset = part.Marker
		if len(offset) == 0 || !part.IsTruncated {
			break
		}
	}
	return ret, nil
}

func (self *SApsaraClient) GetUser(name string) (*SUser, error) {
	params := map[string]string{
		"UserName": name,
	}
	resp, err := self.ramRequest("GetUser", params)
	if err != nil {
		return nil, errors.Wrap(err, "ramRequest.CreateUser")
	}
	user := &SUser{client: self}
	err = resp.Unmarshal(user, "User")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return user, nil
}

func (self *SApsaraClient) GetIClouduserByName(name string) (cloudprovider.IClouduser, error) {
	return self.GetUser(name)
}

type SLoginProfile struct {
	CreateDate            string
	MFABindRequired       bool
	PasswordResetRequired bool
	UserName              string
}

func (self *SApsaraClient) GetLoginProfile(name string) (*SLoginProfile, error) {
	params := map[string]string{
		"UserName": name,
	}
	resp, err := self.ramRequest("GetLoginProfile", params)
	if err != nil {
		return nil, errors.Wrap(err, "ramRequest.GetLoginProfile")
	}
	profile := &SLoginProfile{}
	err = resp.Unmarshal(profile, "LoginProfile")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return profile, nil
}

func (self *SApsaraClient) DeleteLoginProfile(name string) error {
	params := map[string]string{
		"UserName": name,
	}
	_, err := self.ramRequest("DeleteLoginProfile", params)
	return err
}

func (self *SApsaraClient) CreateLoginProfile(name, password string) (*SLoginProfile, error) {
	params := map[string]string{
		"UserName": name,
		"Password": password,
	}
	resp, err := self.ramRequest("CreateLoginProfile", params)
	if err != nil {
		return nil, errors.Wrap(err, "ramRequest.CreateLoginProfile")
	}
	profile := &SLoginProfile{}
	err = resp.Unmarshal(profile, "LoginProfile")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return profile, nil
}

func (self *SApsaraClient) UpdateLoginProfile(name, password string) error {
	params := map[string]string{
		"UserName": name,
		"Password": password,
	}
	_, err := self.ramRequest("UpdateLoginProfile", params)
	if err != nil {
		return errors.Wrap(err, "ramRequest.CreateLoginProfile")
	}
	return nil
}

func (self *SApsaraClient) ResetClouduserPassword(name, password string) error {
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

func (self *SApsaraClient) GetIamLoginUrl() string {
	params := map[string]string{}
	resp, err := self.ramRequest("GetAccountAlias", params)
	if err != nil {
		log.Errorf("GetAccountAlias error: %v", err)
		return ""
	}
	alias, _ := resp.GetString("AccountAlias")
	if len(alias) > 0 {
		return fmt.Sprintf("https://signin.apsara.com/%s.onapsara.com/login.htm", alias)
	}
	return ""
}

// https://help.apsara.com/document_detail/28707.html?spm=a2c4g.11186623.6.752.f4466bbfVy5j0s
func (self *SApsaraClient) ListGroupsForUser(user string) ([]SGroup, error) {
	params := map[string]string{
		"UserName": user,
	}
	resp, err := self.ramRequest("ListGroupsForUser", params)
	if err != nil {
		return nil, errors.Wrap(err, "ListGroupsForUser")
	}
	groups := []SGroup{}
	err = resp.Unmarshal(&groups, "Groups", "Group")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return groups, nil
}

// https://help.apsara.com/document_detail/28732.html?spm=a2c4g.11186623.6.777.580735b2m2xUh8
func (self *SApsaraClient) ListPoliciesForUser(user string) ([]SPolicy, error) {
	params := map[string]string{
		"UserName": user,
	}
	resp, err := self.ramRequest("ListPoliciesForUser", params)
	if err != nil {
		return nil, errors.Wrap(err, "ListPoliciesForUser")
	}
	policies := []SPolicy{}
	err = resp.Unmarshal(&policies, "Policies", "Policy")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return policies, nil
}
