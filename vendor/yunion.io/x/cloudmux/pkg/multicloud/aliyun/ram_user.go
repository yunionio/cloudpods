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
	client *SAliyunClient
	multicloud.SBaseClouduser

	Comments          string
	CreateDate        time.Time
	DisplayName       string
	Email             string
	MobilePhone       string
	ProvisionType     string
	Status            string
	UserId            string
	UserPrincipalName string
}

func (user *SUser) GetGlobalId() string {
	if len(user.UserId) > 0 {
		return user.UserId
	}
	u, err := user.client.GetUser(user.UserPrincipalName)
	if err != nil {
		return ""
	}
	return u.UserId
}

func (user *SUser) GetName() string {
	info := strings.Split(user.UserPrincipalName, "@")
	if len(info) == 2 {
		return info[0]
	}
	return user.UserPrincipalName
}

func (user *SUser) GetEmailAddr() string {
	return user.Email
}

func (user *SUser) GetInviteUrl() string {
	return ""
}

func (user *SUser) Delete() error {
	groups, err := user.client.ListGroupsForUser(user.GetName())
	if err != nil {
		return errors.Wrap(err, "ListGroupsForUser")
	}
	for i := range groups {
		err = user.client.RemoveUserFromGroup(groups[i].GroupName, user.GetName())
		if err != nil {
			return errors.Wrapf(err, "RemoveUserFromGroup %s > %s", groups[i].GroupName, user.GetName())
		}
	}
	policies, err := user.client.ListPoliciesForUser(user.GetName())
	if err != nil {
		return errors.Wrap(err, "ListPoliciesForUser")
	}
	for i := range policies {
		err = user.client.DetachPolicyFromUser(policies[i].PolicyName, policies[i].PolicyType, user.GetName())
		if err != nil {
			return errors.Wrapf(err, "DetachPolicyFromUser %s %s %s", policies[i].PolicyName, policies[i].PolicyType, user.GetName())
		}
	}
	return user.client.DeleteClouduser(user.GetName())
}

func (user *SUser) GetICloudgroups() ([]cloudprovider.ICloudgroup, error) {
	groups, err := user.client.ListGroupsForUser(user.GetName())
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
	return user.client.UpdateLoginProfile(user.UserPrincipalName, password, nil, nil, "")
}

func (user *SUser) GetICloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	policies, err := user.client.ListPoliciesForUser(user.GetName())
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
	profile, err := user.client.GetLoginProfile(user.UserPrincipalName)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			return false
		}
		return false
	}
	return profile.Status == "Active"
}

func (user *SUser) SetDisable() error {
	profile, err := user.client.GetLoginProfile(user.UserPrincipalName)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			return nil
		}
		return errors.Wrapf(err, "GetLoginProfile")
	}
	return user.client.UpdateLoginProfile(user.UserPrincipalName, "", &profile.PasswordResetRequired, &profile.MFABindRequired, "Inactive")
}

func (user *SUser) SetEnable(opts *cloudprovider.SClouduserEnableOptions) error {
	_, err := user.client.GetLoginProfile(user.UserPrincipalName)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			_, err := user.client.CreateLoginProfile(user.UserPrincipalName, opts.Password, opts.PasswordResetRequired, opts.EnableMfa)
			return err
		}
		return errors.Wrapf(err, "GetLoginProfile")
	}
	return user.client.UpdateLoginProfile(user.UserPrincipalName, opts.Password, &opts.PasswordResetRequired, &opts.EnableMfa, "Active")
}

func (user *SUser) ResetPassword(password string) error {
	return user.client.ResetClouduserPassword(user.UserPrincipalName, password)
}

func (user *SUser) AttachPolicy(policyName string, policyType api.TPolicyType) error {
	return user.client.AttachPolicyToUser(policyName, utils.Capitalize(string(policyType)), user.GetName())
}

func (user *SUser) DetachPolicy(policyName string, policyType api.TPolicyType) error {
	return user.client.DetachPolicyFromUser(policyName, utils.Capitalize(string(policyType)), user.GetName())
}

func (self *SAliyunClient) DeleteClouduser(name string) error {
	params := map[string]string{
		"UserName": name,
	}
	_, err := self.ramRequest("DeleteUser", params)
	return err
}

func (self *SAliyunClient) GetDefaultDomain() (string, error) {
	resp, err := self.imsRequest("GetDefaultDomain", nil)
	if err != nil {
		return "", errors.Wrapf(err, "GetDefaultDomain")
	}
	return resp.GetString("DefaultDomainName")
}

func (self *SAliyunClient) CreateUser(name, phone, email, comments string) (*SUser, error) {
	domain, err := self.GetDefaultDomain()
	if err != nil {
		return nil, err
	}
	params := map[string]string{
		"UserPrincipalName": fmt.Sprintf("%s@%s", name, domain),
		"DisplayName":       name,
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
	resp, err := self.imsRequest("CreateUser", params)
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

func (self *SAliyunClient) ListUsers(offset string, limit int) (*SUsers, error) {
	params := map[string]string{}
	if len(offset) > 0 {
		params["Marker"] = offset
	}
	if limit > 0 {
		params["MaxItems"] = fmt.Sprintf("%d", limit)
	}
	resp, err := self.imsRequest("ListUsers", params)
	if err != nil {
		return nil, errors.Wrap(err, "ListUsers")
	}
	users := &SUsers{}
	err = resp.Unmarshal(users)
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return users, nil
}

func (self *SAliyunClient) CreateIClouduser(conf *cloudprovider.SClouduserCreateConfig) (cloudprovider.IClouduser, error) {
	user, err := self.CreateUser(conf.Name, conf.MobilePhone, conf.Email, conf.Desc)
	if err != nil {
		return nil, errors.Wrap(err, "CreateUser")
	}
	if len(conf.Password) > 0 {
		_, err := self.CreateLoginProfile(user.UserPrincipalName, conf.Password, false, true)
		if err != nil {
			return nil, errors.Wrap(err, "CreateLoginProfile")
		}
	}
	return user, nil
}

func (self *SAliyunClient) GetICloudusers() ([]cloudprovider.IClouduser, error) {
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

func (self *SAliyunClient) GetUser(name string) (*SUser, error) {
	if !strings.Contains(name, "@") {
		domain, err := self.GetDefaultDomain()
		if err != nil {
			return nil, err
		}
		name = fmt.Sprintf("%s@%s", name, domain)
	}
	params := map[string]string{
		"UserPrincipalName": name,
	}
	resp, err := self.imsRequest("GetUser", params)
	if err != nil {
		return nil, errors.Wrap(err, "GetUser")
	}
	user := &SUser{client: self}
	err = resp.Unmarshal(user, "User")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return user, nil
}

func (self *SAliyunClient) GetIClouduserByName(name string) (cloudprovider.IClouduser, error) {
	return self.GetUser(name)
}

type SLoginProfile struct {
	CreateDate            string
	MFABindRequired       bool
	PasswordResetRequired bool
	UserPrincipalName     string
	Status                string
}

func (self *SAliyunClient) GetLoginProfile(name string) (*SLoginProfile, error) {
	params := map[string]string{
		"UserPrincipalName": name,
	}
	resp, err := self.imsRequest("GetLoginProfile", params)
	if err != nil {
		return nil, errors.Wrap(err, "GetLoginProfile")
	}
	profile := &SLoginProfile{}
	err = resp.Unmarshal(profile, "LoginProfile")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return profile, nil
}

func (self *SAliyunClient) DeleteLoginProfile(name string) error {
	params := map[string]string{
		"UserName": name,
	}
	_, err := self.ramRequest("DeleteLoginProfile", params)
	return err
}

func (self *SAliyunClient) CreateLoginProfile(name, password string, reset bool, mfa bool) (*SLoginProfile, error) {
	params := map[string]string{
		"UserPrincipalName":     name,
		"Password":              password,
		"PasswordResetRequired": "false",
		"MFABindRequired":       "false",
	}
	if reset {
		params["PasswordResetRequired"] = "true"
	}
	if mfa {
		params["MFABindRequired"] = "true"
	}
	resp, err := self.imsRequest("CreateLoginProfile", params)
	if err != nil {
		return nil, errors.Wrap(err, "CreateLoginProfile")
	}
	profile := &SLoginProfile{}
	err = resp.Unmarshal(profile, "LoginProfile")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return profile, nil
}

func (self *SAliyunClient) UpdateLoginProfile(name, password string, reset *bool, mfa *bool, state string) error {
	params := map[string]string{
		"UserPrincipalName": name,
		"Password":          password,
	}
	if len(password) > 0 {
		params["Password"] = password
	}
	if reset != nil {
		params["PasswordResetRequired"] = "false"
		if *reset {
			params["PasswordResetRequired"] = "true"
		}
	}
	if mfa != nil {
		params["MFABindRequired"] = "false"
		if *mfa {
			params["MFABindRequired"] = "true"
		}
	}
	if len(state) > 0 {
		params["Status"] = state
	}
	_, err := self.imsRequest("UpdateLoginProfile", params)
	if err != nil {
		return errors.Wrap(err, "ramRequest.UpdateLoginProfile")
	}
	return nil
}

func (self *SAliyunClient) ResetClouduserPassword(name, password string) error {
	profile, err := self.GetLoginProfile(name)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			_, err = self.CreateLoginProfile(name, password, profile.PasswordResetRequired, profile.MFABindRequired)
			return err
		}
		return errors.Wrap(err, "GetLoginProfile")
	}
	return self.UpdateLoginProfile(name, password, &profile.PasswordResetRequired, &profile.MFABindRequired, "")
}

func (self *SAliyunClient) GetIamLoginUrl() string {
	params := map[string]string{}
	resp, err := self.ramRequest("GetAccountAlias", params)
	if err != nil {
		log.Errorf("GetAccountAlias error: %v", err)
		return ""
	}
	alias, _ := resp.GetString("AccountAlias")
	if len(alias) > 0 {
		return fmt.Sprintf("https://signin.aliyun.com/%s.onaliyun.com/login.htm", alias)
	}
	return ""
}

// https://help.aliyun.com/document_detail/28707.html?spm=a2c4g.11186623.6.752.f4466bbfVy5j0s
func (self *SAliyunClient) ListGroupsForUser(user string) ([]SGroup, error) {
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

// https://help.aliyun.com/document_detail/28732.html?spm=a2c4g.11186623.6.777.580735b2m2xUh8
func (self *SAliyunClient) ListPoliciesForUser(user string) ([]SPolicy, error) {
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
