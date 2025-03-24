// Copyright 2023 Yunion
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
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
)

type SUser struct {
	multicloud.SBaseClouduser
	client *SVolcEngineClient

	Id                  int
	CreateDate          string
	UpdateDate          string
	Status              string
	AccountId           string
	UserName            string
	Description         string
	DisplayName         string
	Email               string
	EmailIsVerify       bool
	MobilePhone         string
	MobilePhoneIsVerify bool
	Trn                 string
	Source              string
}

func (user *SUser) GetGlobalId() string {
	return user.UserName
}

func (user *SUser) GetName() string {
	return user.UserName
}

func (user *SUser) GetEmailAddr() string {
	return user.Email
}

func (user *SUser) GetInviteUrl() string {
	return ""
}

func (user *SUser) Delete() error {
	return user.client.DeleteUser(user.UserName)
}

func (user *SUser) GetICloudgroups() ([]cloudprovider.ICloudgroup, error) {
	groups, err := user.client.ListGroupsForUser(user.UserName)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudgroup{}
	for i := range groups {
		groups[i].client = user.client
		ret = append(ret, &groups[i])
	}
	return ret, nil
}

func (user *SUser) GetICloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	policies, err := user.client.ListAttachedUserPolicies(user.UserName)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudpolicy{}
	for i := range policies {
		policies[i].client = user.client
		ret = append(ret, &policies[i])
	}
	return ret, nil
}

func (user *SUser) SetDisable() error {
	return user.client.DeleteLoginProfile(user.UserName)
}

func (user *SUser) SetEnable(opts *cloudprovider.SClouduserEnableOptions) error {
	login := true
	return user.client.UpdateLoginProfile(user.UserName, opts.Password, &login, &opts.PasswordResetRequired, &opts.EnableMfa)
}

func (user *SUser) IsConsoleLogin() bool {
	profile, err := user.client.GetLoginProfile(user.UserName)
	if err != nil {
		return false
	}
	return profile.LoginAllowed
}

func (user *SUser) ResetPassword(password string) error {
	return user.client.UpdateLoginProfile(user.UserName, password, nil, nil, nil)
}

func (user *SUser) AttachPolicy(policyName string, policyType api.TPolicyType) error {
	return user.client.AttachUserPolicy(user.UserName, policyName, utils.Capitalize(string(policyType)))
}

func (user *SUser) DetachPolicy(policyName string, policyType api.TPolicyType) error {
	return user.client.DetachUserPolicy(user.UserName, policyName, utils.Capitalize(string(policyType)))
}

func (client *SVolcEngineClient) GetUsers() ([]SUser, error) {
	params := map[string]string{
		"Limit": "50",
	}
	offset := 0
	ret := []SUser{}
	for {
		params["Offset"] = fmt.Sprintf("%d", offset)
		resp, err := client.iamRequest("", "ListUsers", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			UserMetadata []SUser
			Total        int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.UserMetadata...)
		if len(part.UserMetadata) == 0 || len(ret) >= part.Total {
			break
		}
		offset = len(ret)
	}
	return ret, nil
}

func (self *SVolcEngineClient) DeleteUser(name string) error {
	params := map[string]string{
		"UserName": name,
	}
	_, err := self.iamRequest("", "DeleteUser", params)
	return err
}

func (client *SVolcEngineClient) GetICloudusers() ([]cloudprovider.IClouduser, error) {
	users, err := client.GetUsers()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.IClouduser{}
	for i := range users {
		users[i].client = client
		ret = append(ret, &users[i])
	}
	return ret, nil
}

func (client *SVolcEngineClient) CreateIClouduser(opts *cloudprovider.SClouduserCreateConfig) (cloudprovider.IClouduser, error) {
	user, err := client.CreateUser(opts)
	if err != nil {
		return nil, err
	}
	err = client.CreateLoginProfile(user.UserName, opts.Password, &opts.IsConsoleLogin)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateLoginProfile")
	}
	return user, nil
}

func (self *SVolcEngineClient) CreateUser(opts *cloudprovider.SClouduserCreateConfig) (*SUser, error) {
	params := map[string]string{
		"UserName":    opts.Name,
		"Description": opts.Desc,
		"Email":       opts.Email,
		"MobilePhone": opts.MobilePhone,
	}
	resp, err := self.iamRequest("", "CreateUser", params)
	if err != nil {
		return nil, err
	}
	ret := &SUser{client: self}
	err = resp.Unmarshal(ret, "User")
	if err != nil {
		return nil, err
	}
	return ret, nil
}

type LoginProfile struct {
	PasswordResetRequired bool
	LoginAllowed          bool
	LastLoginDate         string
}

func (self *SVolcEngineClient) GetLoginProfile(name string) (*LoginProfile, error) {
	params := map[string]string{
		"UserName": name,
	}
	resp, err := self.iamRequest("", "GetLoginProfile", params)
	if err != nil {
		return nil, err
	}
	ret := &LoginProfile{}
	err = resp.Unmarshal(ret, "LoginProfile")
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SVolcEngineClient) CreateLoginProfile(name, password string, loginAllowd *bool) error {
	params := map[string]string{
		"UserName": name,
		"Password": password,
	}
	if loginAllowd != nil {
		params["LoginAllowed"] = fmt.Sprintf("%v", *loginAllowd)
	}
	_, err := self.iamRequest("", "CreateLoginProfile", params)
	return err
}

func (self *SVolcEngineClient) DeleteLoginProfile(name string) error {
	params := map[string]string{
		"UserName": name,
	}
	_, err := self.iamRequest("", "DeleteLoginProfile", params)
	return err
}

func (self *SVolcEngineClient) UpdateLoginProfile(name, password string, loginAllowd *bool, reset, mfa *bool) error {
	params := map[string]string{
		"UserName": name,
		"Password": password,
	}
	if loginAllowd != nil {
		params["LoginAllowed"] = fmt.Sprintf("%v", *loginAllowd)
	}
	if reset != nil {
		params["PasswordResetRequired"] = "false"
		if *reset {
			params["PasswordResetRequired"] = "true"
		}
	}
	if mfa != nil {
		params["SafeAuthFlag"] = "false"
		if *mfa {
			params["SafeAuthFlag"] = "true"
			params["SafeAuthType"] = "vmfa"
		}
	}
	_, err := self.iamRequest("", "UpdateLoginProfile", params)
	return err
}

func (client *SVolcEngineClient) ListGroupsForUser(name string) ([]SGroup, error) {
	params := map[string]string{
		"Limit":    "50",
		"UserName": name,
	}
	offset := 0
	ret := []SGroup{}
	for {
		params["Offset"] = fmt.Sprintf("%d", offset)
		resp, err := client.iamRequest("", "ListGroupsForUser", params)
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

func (client *SVolcEngineClient) ListAttachedUserPolicies(name string) ([]SPolicy, error) {
	params := map[string]string{
		"UserName": name,
	}
	resp, err := client.iamRequest("", "ListAttachedUserPolicies", params)
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

func (client *SVolcEngineClient) AttachUserPolicy(name, policy, policyType string) error {
	params := map[string]string{
		"UserName":   name,
		"PolicyName": policy,
		"PolicyType": policyType,
	}
	_, err := client.iamRequest("", "AttachUserPolicy", params)
	return err
}

func (client *SVolcEngineClient) DetachUserPolicy(name, policy, policyType string) error {
	params := map[string]string{
		"UserName":   name,
		"PolicyName": policy,
		"PolicyType": policyType,
	}
	_, err := client.iamRequest("", "DetachUserPolicy", params)
	return err
}

func (client *SVolcEngineClient) GetIClouduserByName(name string) (cloudprovider.IClouduser, error) {
	user, err := client.GetUser(name)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (client *SVolcEngineClient) GetUser(name string) (*SUser, error) {
	params := map[string]string{
		"UserName": name,
	}
	resp, err := client.iamRequest("", "GetUser", params)
	if err != nil {
		return nil, err
	}
	ret := &SUser{client: client}
	err = resp.Unmarshal(ret, "User")
	if err != nil {
		return nil, err
	}
	return ret, nil
}
