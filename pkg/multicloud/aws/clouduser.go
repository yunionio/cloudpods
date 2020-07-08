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

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SClouduser struct {
	client           *SAwsClient
	UserId           string    `xml:"UserId"`
	Path             string    `xml:"Path"`
	UserName         string    `xml:"UserName"`
	Arn              string    `xml:"Arn"`
	CreateDate       time.Time `xml:"CreateDate"`
	PasswordLastUsed time.Time `xml:"PasswordLastUsed"`
}

func (user *SClouduser) AttachSystemPolicy(policyArn string) error {
	return user.client.AttachPolicy(user.UserName, user.client.getIamArn(policyArn))
}

func (user *SClouduser) DetachSystemPolicy(policyArn string) error {
	return user.client.DetachPolicy(user.UserName, user.client.getIamArn(policyArn))
}

func (user *SClouduser) GetGlobalId() string {
	return user.UserId
}

func (user *SClouduser) GetName() string {
	return user.UserName
}

func (user *SClouduser) ResetPassword(password string) error {
	return user.client.ResetClouduserPassword(user.UserName, password)
}

func (user *SClouduser) IsConsoleLogin() bool {
	_, err := user.client.GetLoginProfile(user.UserName)
	if err == cloudprovider.ErrNotFound {
		return false
	}
	return true
}

func (user *SClouduser) GetICloudgroups() ([]cloudprovider.ICloudgroup, error) {
	groups := []SCloudgroup{}
	marker := ""
	for {
		part, err := user.client.ListGroupsForUser(user.UserName, marker, 1000)
		if err != nil {
			return nil, errors.Wrap(err, "ListGroupsForUser")
		}
		groups = append(groups, part.Groups...)
		marker = part.Marker
		if len(marker) == 0 {
			break
		}
	}
	ret := []cloudprovider.ICloudgroup{}
	for i := range groups {
		groups[i].client = user.client
		ret = append(ret, &groups[i])
	}
	return ret, nil
}

func (user *SClouduser) GetISystemCloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	policies, err := user.client.ListUserAttachedPolicies(user.UserName)
	if err != nil {
		return nil, errors.Wrap(err, "ListUserAttachPolicies")
	}
	ret := []cloudprovider.ICloudpolicy{}
	for i := range policies {
		policies[i].client = user.client
		ret = append(ret, &policies[i])
	}
	return ret, nil
}

func (user *SClouduser) Delete() error {
	marker := ""
	for {
		groups, err := user.client.ListGroupsForUser(user.UserName, marker, 1000)
		if err != nil {
			return errors.Wrap(err, "ListGroupsForUser")
		}
		for _, group := range groups.Groups {
			err = user.client.RemoveUserFromGroup(group.GroupName, user.UserName)
			if err != nil {
				return errors.Wrap(err, "RemoveUserFromGroup")
			}
		}
		marker = groups.Marker
		if len(marker) == 0 {
			break
		}
	}
	policies, err := user.client.ListUserAttachedPolicies(user.UserName)
	if err != nil {
		return errors.Wrap(err, "ListUserAttachPolicies")
	}
	for i := range policies {
		err = user.DetachSystemPolicy(policies[i].PolicyArn)
		if err != nil {
			return errors.Wrap(err, "DetachPolicy")
		}
	}
	return user.client.DeleteClouduser(user.UserName)
}

type SCloudusers struct {
	Users       []SClouduser `xml:"Users>member"`
	IsTruncated bool         `xml:"IsTruncated"`
	Marker      string       `xml:"Marker"`
}

func (self *SAwsClient) GetCloudusers(marker string, maxItems int, pathPrefix string) (*SCloudusers, error) {
	if maxItems <= 0 || maxItems > 1000 {
		maxItems = 1000
	}
	params := map[string]string{
		"MaxItems": fmt.Sprintf("%d", maxItems),
	}
	if len(marker) > 0 {
		params["Marker"] = marker
	}
	if len(pathPrefix) > 0 {
		params["PathPrefix"] = pathPrefix
	}
	users := &SCloudusers{}
	err := self.iamRequest("ListUsers", params, users)
	if err != nil {
		return nil, errors.Wrap(err, "iamRequest.ListUsers")
	}
	return users, nil
}

func (self *SAwsClient) CreateClouduser(path string, username string) (*SClouduser, error) {
	params := map[string]string{
		"UserName": username,
	}
	if len(path) > 0 {
		params["Path"] = path
	}
	user := struct {
		User SClouduser `xml:"User"`
	}{}
	err := self.iamRequest("CreateUser", params, &user)
	if err != nil {
		return nil, errors.Wrap(err, "iamRequest.CreateUser")
	}
	user.User.client = self
	return &user.User, nil
}

func (self *SAwsClient) DeleteClouduser(name string) error {
	params := map[string]string{
		"UserName": name,
	}
	return self.iamRequest("DeleteUser", params, nil)
}

func (self *SAwsClient) AttachPolicy(userName string, policyArn string) error {
	params := map[string]string{
		"PolicyArn": policyArn,
		"UserName":  userName,
	}
	return self.iamRequest("AttachUserPolicy", params, nil)
}

func (self *SAwsClient) DetachPolicy(userName string, policyArn string) error {
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
	user, err := self.CreateClouduser("", conf.Name)
	if err != nil {
		return nil, errors.Wrap(err, "CreateClouduser")
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
	return self.GetClouduser(name)
}

func (self *SAwsClient) GetClouduser(name string) (*SClouduser, error) {
	user := struct {
		User SClouduser `xml:"User"`
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

func (self *SAwsClient) ListUsers() ([]SClouduser, error) {
	users := []SClouduser{}
	marker := ""
	for {
		part, err := self.GetCloudusers(marker, 1000, "")
		if err != nil {
			return nil, errors.Wrap(err, "GetCloudusers")
		}
		users = append(users, part.Users...)
		if !part.IsTruncated {
			break
		}
		marker = part.Marker
	}
	return users, nil
}

func (self *SAwsClient) GetICloudusers() ([]cloudprovider.IClouduser, error) {
	users, err := self.ListUsers()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.IClouduser{}
	for i := range users {
		users[i].client = self
		ret = append(ret, &users[i])
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

func (self *SAwsClient) ResetClouduserPassword(name, password string) error {
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

func (self *SAwsClient) ListGroupsForUser(name string, marker string, maxItems int) (*SCloudgroups, error) {
	if maxItems < 1 || maxItems > 1000 {
		maxItems = 1000
	}
	params := map[string]string{
		"UserName": name,
		"MaxItems": fmt.Sprintf("%d", maxItems),
	}
	if len(marker) > 0 {
		params["Marker"] = marker
	}
	groups := &SCloudgroups{}
	err := self.iamRequest("ListGroupsForUser", params, groups)
	if err != nil {
		return nil, errors.Wrap(err, "ListGroupsForUser")
	}
	return groups, nil
}
