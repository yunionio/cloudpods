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

package ksyun

import (
	"time"

	api "yunion.io/x/cloudmux/pkg/apis/cloudid"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/pkg/errors"
)

type SUser struct {
	multicloud.SBaseClouduser
	client *SKsyunClient

	UserId                string
	Path                  string
	UserName              string
	RealName              string
	CreateDate            time.Time
	Phone                 string
	CountryMobileCode     string
	Email                 string
	PhoneVerified         string
	EmailVerified         string
	Remark                string
	Krn                   string
	PasswordResetRequired bool
	EnableMFA             int
	UpdateDate            time.Time
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

func (user *SUser) IsConsoleLogin() bool {
	profile, err := user.client.GetLoginProfile(user.UserName)
	if err != nil {
		return false
	}
	return profile.ConsoleLogin
}

func (user *SUser) ResetPassword(password string) error {
	return user.client.UpdateLoginProfile(user.UserName, password)
}

func (client *SKsyunClient) UpdateLoginProfile(name, password string) error {
	params := map[string]string{
		"UserName":       name,
		"Password":       password,
		"ViewAllProject": "true",
	}
	_, err := client.iamRequest("", "UpdateLoginProfile", params)
	return err
}

type LoginProfile struct {
	PasswordResetRequired bool
	ConsoleLogin          bool
	LastLoginDate         time.Time
}

func (client *SKsyunClient) GetLoginProfile(name string) (*LoginProfile, error) {
	params := map[string]string{
		"UserName": name,
	}
	resp, err := client.iamRequest("", "GetLoginProfile", params)
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

func (user *SUser) AttachPolicy(policyName string, policyType api.TPolicyType) error {
	return user.client.AttachUserPolicy(user.UserName, policyName)
}

func (user *SUser) DetachPolicy(policyName string, policyType api.TPolicyType) error {
	return user.client.DetachUserPolicy(user.UserName, policyName)
}

func (client *SKsyunClient) GetUsers() ([]SUser, error) {
	params := map[string]string{
		"MaxItems": "100",
	}
	ret := []SUser{}
	for {
		resp, err := client.iamRequest("", "ListUsers", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			Users struct {
				Member []SUser
			}
			Marker string
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.Users.Member...)
		if len(part.Users.Member) == 0 || len(part.Marker) == 0 {
			break
		}
		params["Marker"] = part.Marker
	}
	return ret, nil
}

func (client *SKsyunClient) DeleteUser(name string) error {
	params := map[string]string{
		"UserName": name,
	}
	_, err := client.iamRequest("", "DeleteUser", params)
	return err
}

func (client *SKsyunClient) GetICloudusers() ([]cloudprovider.IClouduser, error) {
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

func (client *SKsyunClient) CreateIClouduser(opts *cloudprovider.SClouduserCreateConfig) (cloudprovider.IClouduser, error) {
	user, err := client.CreateUser(opts)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (client *SKsyunClient) CreateUser(opts *cloudprovider.SClouduserCreateConfig) (*SUser, error) {
	params := map[string]string{
		"UserName": opts.Name,
		"Remark":   opts.Desc,
		"Email":    opts.Email,
		"Phone":    opts.MobilePhone,
		"Password": opts.Password,
	}
	resp, err := client.iamRequest("", "CreateUser", params)
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

func (client *SKsyunClient) ListGroupsForUser(name string) ([]SGroup, error) {
	params := map[string]string{
		"UserName": name,
		"MaxItems": "100",
	}
	ret := []SGroup{}
	for {
		resp, err := client.iamRequest("", "ListGroupsForUser", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			Groups struct {
				Memeber []SGroup
			}
			Marker string
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.Groups.Memeber...)
		if len(part.Marker) == 0 || len(part.Groups.Memeber) == 0 {
			break
		}
		params["Marker"] = part.Marker
	}
	return ret, nil
}

func (client *SKsyunClient) ListAttachedUserPolicies(name string) ([]SPolicy, error) {
	params := map[string]string{
		"UserName": name,
		"MaxItems": "100",
	}
	ret := []SPolicy{}
	for {
		resp, err := client.iamRequest("", "ListAttachedUserPolicies", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			AttachedPolicies struct {
				Member []SPolicy
			}
			Marker string
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrapf(err, "Unmarshal")
		}
		ret = append(ret, part.AttachedPolicies.Member...)
		if len(part.Marker) == 0 || len(part.AttachedPolicies.Member) == 0 {
			break
		}
		params["Marker"] = part.Marker
	}
	return ret, nil
}

func (client *SKsyunClient) AttachUserPolicy(name, policy string) error {
	params := map[string]string{
		"UserName":  name,
		"PolicyKrn": policy,
	}
	_, err := client.iamRequest("", "AttachUserPolicy", params)
	return err
}

func (client *SKsyunClient) DetachUserPolicy(name, policy string) error {
	params := map[string]string{
		"UserName":  name,
		"PolicyKrn": policy,
	}
	_, err := client.iamRequest("", "DetachUserPolicy", params)
	return err
}

func (client *SKsyunClient) GetIClouduserByName(name string) (cloudprovider.IClouduser, error) {
	user, err := client.GetUser(name)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (client *SKsyunClient) GetUser(name string) (*SUser, error) {
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
