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
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SClouduser struct {
	client *SAliyunClient

	Comments    string
	CreateDate  time.Time
	DisplayName string
	Email       string
	MobilePhone string
	UserId      string
	UserName    string
}

func (user *SClouduser) GetGlobalId() string {
	if len(user.UserId) > 0 {
		return user.UserId
	}
	u, err := user.client.GetClouduser(user.UserName)
	if err != nil {
		return ""
	}
	return u.UserId
}

func (user *SClouduser) GetName() string {
	return user.UserName
}

func (user *SClouduser) Delete() error {
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

func (user *SClouduser) GetICloudgroups() ([]cloudprovider.ICloudgroup, error) {
	groups, err := user.client.ListGroupsForUser(user.UserName)
	if err != nil {
		return nil, errors.Wrap(err, "ListGroupsForUser")
	}
	iGroups := []cloudprovider.ICloudgroup{}
	for i := range groups {
		groups[i].client = user.client
		iGroups = append(iGroups, &groups[i])
	}
	return iGroups, nil
}

func (user *SClouduser) UpdatePassword(password string) error {
	return user.client.UpdateLoginProfile(user.UserName, password)
}

func (user *SClouduser) GetISystemCloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	policies, err := user.client.ListPoliciesForUser(user.UserName)
	if err != nil {
		return nil, errors.Wrap(err, "ListPoliciesForUser")
	}
	ret := []cloudprovider.ICloudpolicy{}
	for i := range policies {
		if policies[i].PolicyType == "System" {
			ret = append(ret, &policies[i])
		}
	}
	return ret, nil
}

func (user *SClouduser) IsConsoleLogin() bool {
	_, err := user.client.GetLoginProfile(user.UserName)
	if errors.Cause(err) == cloudprovider.ErrNotFound {
		return false
	}
	return true
}

func (user *SClouduser) ResetPassword(password string) error {
	return user.client.ResetClouduserPassword(user.UserName, password)
}

func (user *SClouduser) AttachSystemPolicy(policyName string) error {
	return user.client.AttachPolicyToUser(policyName, "System", user.UserName)
}

func (user *SClouduser) DetachSystemPolicy(policyName string) error {
	return user.client.DetachPolicyFromUser(policyName, "System", user.UserName)
}

func (self *SAliyunClient) DeleteClouduser(name string) error {
	params := map[string]string{
		"UserName": name,
	}
	_, err := self.ramRequest("DeleteUser", params)
	return err
}

func (self *SAliyunClient) CreateClouduser(name, phone, email, comments string) (*SClouduser, error) {
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

	user := &SClouduser{client: self}
	err = resp.Unmarshal(user, "User")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return user, nil
}

func (self *SAliyunClient) GetCloudusers(marker string, maxItems int) ([]SClouduser, string, error) {
	params := map[string]string{}
	if len(marker) > 0 {
		params["Marker"] = marker
	}
	if maxItems > 0 {
		params["MaxItems"] = fmt.Sprintf("%d", maxItems)
	}
	resp, err := self.ramRequest("ListUsers", params)
	if err != nil {
		return nil, "", errors.Wrap(err, "ramRequest.ListUsers")
	}
	users := []SClouduser{}
	err = resp.Unmarshal(&users, "Users", "User")
	if err != nil {
		return nil, "", errors.Wrap(err, "resp.Unmarshal")
	}
	marker, _ = resp.GetString("Marker")
	return users, marker, nil
}

func (self *SAliyunClient) CreateIClouduser(conf *cloudprovider.SClouduserCreateConfig) (cloudprovider.IClouduser, error) {
	user, err := self.CreateClouduser(conf.Name, conf.MobilePhone, conf.Email, conf.Desc)
	if err != nil {
		return nil, errors.Wrap(err, "CreateClouduser")
	}
	if len(conf.Password) > 0 {
		_, err := self.CreateLoginProfile(conf.Name, conf.Password)
		if err != nil {
			return nil, errors.Wrap(err, "CreateLoginProfile")
		}
	}
	for _, policyId := range conf.ExternalPolicyIds {
		err := user.AttachSystemPolicy(policyId)
		if err != nil {
			log.Errorf("attach policy %s for user %s error: %v", policyId, conf.Name, err)
		}
	}
	return user, nil
}

func (self *SAliyunClient) GetICloudusers() ([]cloudprovider.IClouduser, error) {
	var (
		ret    []cloudprovider.IClouduser
		users  []SClouduser
		marker string
		err    error
	)
	for {
		users, marker, err = self.GetCloudusers(marker, 100)
		if err != nil {
			return nil, errors.Wrap(err, "GetCloudusers")
		}
		for i := range users {
			users[i].client = self
			ret = append(ret, &users[i])
		}
		if len(marker) == 0 || len(users) == 0 {
			break
		}
	}
	return ret, nil
}

func (self *SAliyunClient) GetClouduser(name string) (*SClouduser, error) {
	params := map[string]string{
		"UserName": name,
	}
	resp, err := self.ramRequest("GetUser", params)
	if err != nil {
		return nil, errors.Wrap(err, "ramRequest.CreateUser")
	}
	user := &SClouduser{client: self}
	err = resp.Unmarshal(user, "User")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return user, nil
}

func (self *SAliyunClient) GetIClouduserByName(name string) (cloudprovider.IClouduser, error) {
	return self.GetClouduser(name)
}

type SLoginProfile struct {
	CreateDate            string
	MFABindRequired       bool
	PasswordResetRequired bool
	UserName              string
}

func (self *SAliyunClient) GetLoginProfile(name string) (*SLoginProfile, error) {
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

func (self *SAliyunClient) DeleteLoginProfile(name string) error {
	params := map[string]string{
		"UserName": name,
	}
	_, err := self.ramRequest("DeleteLoginProfile", params)
	return err
}

func (self *SAliyunClient) CreateLoginProfile(name, password string) (*SLoginProfile, error) {
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

func (self *SAliyunClient) UpdateLoginProfile(name, password string) error {
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

func (self *SAliyunClient) ResetClouduserPassword(name, password string) error {
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
func (self *SAliyunClient) ListGroupsForUser(user string) ([]SCloudgroup, error) {
	params := map[string]string{
		"UserName": user,
	}
	resp, err := self.ramRequest("ListGroupsForUser", params)
	if err != nil {
		return nil, errors.Wrap(err, "ListGroupsForUser")
	}
	groups := []SCloudgroup{}
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
