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

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SClouduser struct {
	client       *SQcloudClient
	Uin          int64
	Name         string
	Uid          int64
	Remark       string
	ConsoleLogin int
	CountryCode  string
	Email        string
}

func (user *SClouduser) GetGlobalId() string {
	return fmt.Sprintf("%d", user.Uin)
}

func (user *SClouduser) GetISystemCloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	policies := []SClouduserPolicy{}
	page := 1
	for {
		part, total, err := user.client.ListAttachedUserPolicies(user.GetGlobalId(), page, 50)
		if err != nil {
			return nil, errors.Wrap(err, "GetClouduserPolicy")
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
			policies[i].client = user.client
			ret = append(ret, &policies[i])
		}
	}
	return ret, nil
}

func (self *SQcloudClient) GetISystemCloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	policies := []SClouduserPolicy{}
	i := 1
	for {
		part, total, err := self.ListPolicies("", "QCS", i, 100)
		if err != nil {
			return nil, errors.Wrap(err, "GetPolicies")
		}
		policies = append(policies, part...)
		if len(policies) >= total {
			break
		}
		i += 1
	}
	ret := []cloudprovider.ICloudpolicy{}
	for i := range policies {
		policies[i].client = self
		ret = append(ret, &policies[i])
	}
	return ret, nil
}

func (user *SClouduser) AttachSystemPolicy(policyId string) error {
	return user.client.AttachPolicy(user.GetGlobalId(), policyId)
}

func (user *SClouduser) DetachSystemPolicy(policyId string) error {
	return user.client.DetachPolicy(user.GetGlobalId(), policyId)
}

func (user *SClouduser) IsConsoleLogin() bool {
	return user.ConsoleLogin == 1
}

func (user *SClouduser) Delete() error {
	return user.client.DeleteUser(user.Name)
}

func (user *SClouduser) ResetPassword(password string) error {
	return user.client.ResetClouduserPassword(user.Name, password)
}

func (user *SClouduser) GetICloudgroups() ([]cloudprovider.ICloudgroup, error) {
	groups := []SCloudgroup{}
	page := 1
	for {
		part, total, err := user.client.ListGroupsForUser(fmt.Sprintf("%d", user.Uin), page, 50)
		if err != nil {
			return nil, errors.Wrap(err, "ListGroupsForUser")
		}
		groups = append(groups, part...)
		if len(groups) >= total {
			break
		}
		page += 1
	}
	ret := []cloudprovider.ICloudgroup{}
	for i := range groups {
		groups[i].client = user.client
		ret = append(ret, &groups[i])
	}
	return ret, nil
}

func (user *SClouduser) GetName() string {
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

func (self *SQcloudClient) ListUsers() ([]SClouduser, error) {
	resp, err := self.camRequest("ListUsers", nil)
	if err != nil {
		return nil, errors.Wrap(err, "camRequest.ListUsers")
	}
	users := []SClouduser{}
	err = resp.Unmarshal(&users, "Data")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return users, nil
}

func (self *SQcloudClient) ListGroupsForUser(uin string, rp, page int) ([]SCloudgroup, int, error) {
	if page < 1 {
		page = 1
	}
	if rp <= 0 || rp > 50 {
		rp = 50
	}
	params := map[string]string{
		"SubUin": uin,
		"Page":   fmt.Sprintf("%d", page),
		"Rp":     fmt.Sprintf("%d", rp),
	}
	resp, err := self.camRequest("ListGroupsForUser", params)
	if err != nil {
		return nil, 0, errors.Wrap(err, "camRequest.ListGroupsForUser")
	}
	groups := []SCloudgroup{}
	err = resp.Unmarshal(&groups, "GroupInfo")
	if err != nil {
		return nil, 0, errors.Wrap(err, "resp.Unmarshal")
	}
	total, _ := resp.Int("TotalNum")
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
	return ret, nil
}

func (self *SQcloudClient) GetIClouduserByName(name string) (cloudprovider.IClouduser, error) {
	return self.GetClouduser(name)
}

func (self *SQcloudClient) GetClouduser(name string) (*SClouduser, error) {
	params := map[string]string{
		"Name": name,
	}
	resp, err := self.camRequest("GetUser", params)
	if err != nil {
		return nil, errors.Wrap(err, "camRequest.GetUser")
	}
	user := &SClouduser{client: self}
	err = resp.Unmarshal(user)
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return user, nil
}

func (self *SQcloudClient) CreateIClouduser(conf *cloudprovider.SClouduserCreateConfig) (cloudprovider.IClouduser, error) {
	user, err := self.CreateClouduser(conf.Name, conf.Password, conf.Desc, conf.IsConsoleLogin)
	if err != nil {
		return nil, errors.Wrap(err, "CreateClouduser")
	}
	for _, policyId := range conf.ExternalPolicyIds {
		err = user.client.AttachPolicy(fmt.Sprintf("%d", user.Uin), policyId)
		if err != nil {
			log.Errorf("attach policy %s for user %s error: %v", policyId, conf.Name, err)
		}
	}
	return user, nil
}

func (self *SQcloudClient) CreateClouduser(name, password, desc string, consoleLogin bool) (*SClouduser, error) {
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
	user := &SClouduser{client: self}
	err = resp.Unmarshal(user)
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return user, nil
}

type SClouduserPolicy struct {
	client      *SQcloudClient
	PolicyId    int64
	PolicyName  string
	AddTime     time.Time
	CreateMode  string
	PolicyType  string
	Description string
}

func (policy *SClouduserPolicy) GetGlobalId() string {
	return fmt.Sprintf("%d", policy.PolicyId)
}
func (policy *SClouduserPolicy) GetName() string {
	return policy.PolicyName
}

func (policy *SClouduserPolicy) GetPolicyType() string {
	return policy.PolicyName
}

func (policy *SClouduserPolicy) GetDescription() string {
	p, err := policy.client.GetPolicy(policy.GetGlobalId())
	if err != nil {
		return p.Description
	}
	return ""
}

func (self *SQcloudClient) ListAttachedUserPolicies(uin string, page int, rp int) ([]SClouduserPolicy, int, error) {
	if page < 1 {
		page = 1
	}
	if rp <= 0 || rp > 50 {
		rp = 50
	}
	params := map[string]string{
		"TargetUin": uin,
		"Page":      fmt.Sprintf("%d", page),
		"Rp":        fmt.Sprintf("%d", rp),
	}
	resp, err := self.camRequest("ListAttachedUserPolicies", params)
	if err != nil {
		return nil, 0, errors.Wrap(err, "camRequest.ListAttachedUserPolicies")
	}
	policies := []SClouduserPolicy{}
	err = resp.Unmarshal(&policies, "List")
	if err != nil {
		return nil, 0, errors.Wrap(err, "resp.Unmarshal")
	}
	total, _ := resp.Int("TotalNum")
	return policies, int(total), nil
}

func (self *SQcloudClient) AttachPolicy(uin, policyId string) error {
	params := map[string]string{
		"AttachUin": uin,
		"PolicyId":  policyId,
	}
	_, err := self.camRequest("AttachUserPolicy", params)
	return err
}

func (self *SQcloudClient) DetachPolicy(uin, policyId string) error {
	params := map[string]string{
		"DetachUin": uin,
		"PolicyId":  policyId,
	}
	_, err := self.camRequest("DetachUserPolicy", params)
	return err
}

// https://cloud.tencent.com/document/api/598/34570
func (self *SQcloudClient) ListPolicies(keyword, scope string, page int, rp int) ([]SClouduserPolicy, int, error) {
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
	if len(scope) > 0 {
		params["Scope"] = scope
	}
	if len(keyword) > 0 {
		params["Keyword"] = keyword
	}
	resp, err := self.camRequest("ListPolicies", params)
	if err != nil {
		return nil, 0, errors.Wrap(err, "camRequest.ListPolicies")
	}
	policies := []SClouduserPolicy{}
	err = resp.Unmarshal(&policies, "List")
	if err != nil {
		return nil, 0, errors.Wrap(err, "resp.Unmarshal")
	}
	total, _ := resp.Int("TotalNum")
	return policies, int(total), nil
}

func (self *SQcloudClient) GetPolicy(policyId string) (*SClouduserPolicy, error) {
	params := map[string]string{
		"PolicyId": policyId,
	}
	resp, err := self.camRequest("GetPolicy", params)
	if err != nil {
		return nil, errors.Wrap(err, "GetPolicy")
	}
	policy := SClouduserPolicy{}
	err = resp.Unmarshal(&policy)
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return &policy, nil
}

func (self *SQcloudClient) ResetClouduserPassword(name, password string) error {
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
