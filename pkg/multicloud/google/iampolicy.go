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

package google

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SBinding struct {
	Role    string
	Members []string
}

type SIamPolicy struct {
	client   *SGoogleClient
	Version  int
	Etag     string
	Bindings []SBinding
}

func (self *SGoogleClient) GetIamPolicy() (*SIamPolicy, error) {
	resource := fmt.Sprintf("projects/%s:getIamPolicy", self.projectId)
	resp, err := self.managerPost(resource, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "managerPost")
	}
	policy := &SIamPolicy{client: self}
	err = resp.Unmarshal(policy)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	return policy, nil
}

func (self *SGoogleClient) TestIam(permissions []string) ([]string, error) {
	resource := fmt.Sprintf("projects/%s:testIamPermissions", self.projectId)
	body := jsonutils.Marshal(map[string]interface{}{"permissions": permissions})
	resp, err := self.managerPost(resource, nil, body)
	if err != nil {
		return nil, errors.Wrap(err, "testIamPermissions")
	}
	ret := []string{}
	err = resp.Unmarshal(&ret, "permissions")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return ret, nil
}

func (self *SGoogleClient) IsSupportCloudId() bool {
	permissions, err := self.TestIam([]string{"resourcemanager.projects.setIamPolicy"})
	if err != nil {
		return false
	}
	return len(permissions) == 1
}

func (self *SGoogleClient) SetIamPlicy(policy *SIamPolicy) error {
	resource := fmt.Sprintf("projects/%s:setIamPolicy", self.projectId)
	body := jsonutils.Marshal(map[string]interface{}{"policy": map[string]interface{}{"bindings": policy.Bindings}})
	_, err := self.managerPost(resource, nil, body)
	if err != nil {
		return errors.Wrap(err, "managerPost")
	}
	return nil
}

type SClouduserRole struct {
	role string
}

func (role *SClouduserRole) GetDescription() string {
	return ""
}

func (role *SClouduserRole) GetName() string {
	return role.role
}

func (role *SClouduserRole) GetPolicyType() string {
	return role.role
}

func (role *SClouduserRole) GetGlobalId() string {
	return role.role
}

type SClouduser struct {
	policy *SIamPolicy
	Name   string
	Roles  []string
}

func (self *SGoogleClient) GetISystemCloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	roles, err := self.GetRoles()
	if err != nil {
		return nil, errors.Wrap(err, "GetRoles")
	}
	ret := []cloudprovider.ICloudpolicy{}
	for i := range roles {
		ret = append(ret, &roles[i])
	}
	return ret, nil
}

type SRole struct {
	Name        string
	Title       string
	Description string
	Stage       string
	Etag        string
}

func (role *SRole) GetName() string {
	return role.Title
}

func (role *SRole) GetGlobalId() string {
	return role.Name
}

func (role *SRole) GetDescription() string {
	return role.Description
}

func (role *SRole) GetPolicyType() string {
	return "System"
}

// https://cloud.google.com/iam/docs/reference/rest/v1/roles/list
func (self *SGoogleClient) GetRoles() ([]SRole, error) {
	roles := []SRole{}
	err := self.iamListAll("roles", nil, &roles)
	if err != nil {
		return nil, errors.Wrap(err, "iamListAll.roles")
	}
	return roles, nil
}

func (user *SClouduser) GetGlobalId() string {
	return user.Name
}

func (user *SClouduser) GetName() string {
	return user.Name
}

func (user *SClouduser) IsConsoleLogin() bool {
	return true
}

func (user *SClouduser) GetISystemCloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	ret := []cloudprovider.ICloudpolicy{}
	for _, role := range user.Roles {
		ret = append(ret, &SClouduserRole{role: role})
	}
	return ret, nil
}

var getUserName = func(user string) string {
	if strings.HasSuffix(user, "gserviceaccount.com") {
		return "serviceAccount:" + user
	}
	return "user:" + user
}

func (policy *SIamPolicy) AttachPolicy(user string, roles []string) error {
	for _, role := range roles {
		find := false
		for i := range policy.Bindings {
			if policy.Bindings[i].Role == role {
				if !utils.IsInStringArray(getUserName(user), policy.Bindings[i].Members) {
					policy.Bindings[i].Members = append(policy.Bindings[i].Members, getUserName(user))
					find = true
				}
			}
		}
		if !find {
			policy.Bindings = append(policy.Bindings, SBinding{Role: role, Members: []string{getUserName(user)}})
		}
	}
	return policy.client.SetIamPlicy(policy)
}

func (user *SClouduser) AttachSystemPolicy(role string) error {
	return user.policy.AttachPolicy(user.Name, []string{role})
}

func (policy *SIamPolicy) DetachPolicy(user, role string) error {
	change := false
	for i := range policy.Bindings {
		if policy.Bindings[i].Role == role && utils.IsInStringArray(getUserName(user), policy.Bindings[i].Members) {
			change = true
			members := []string{}
			for _, member := range policy.Bindings[i].Members {
				if member != getUserName(user) {
					members = append(members, member)
				}
			}
			policy.Bindings[i].Members = members
		}
	}
	if change {
		return policy.client.SetIamPlicy(policy)
	}
	return nil
}

func (client *SGoogleClient) CreateIClouduser(conf *cloudprovider.SClouduserCreateConfig) (cloudprovider.IClouduser, error) {
	policy, err := client.GetIamPolicy()
	if err != nil {
		return nil, errors.Wrap(err, "GetIamPolicy")
	}
	if len(conf.ExternalPolicyIds) == 0 {
		return nil, fmt.Errorf("missing policy info")
	}
	err = policy.AttachPolicy(conf.Name, conf.ExternalPolicyIds)
	if err != nil {
		return nil, errors.Wrap(err, "policy.AttachPolicy")
	}
	return client.GetIClouduserByName(conf.Name)
}

func (client *SGoogleClient) GetIClouduserByName(name string) (cloudprovider.IClouduser, error) {
	policy, err := client.GetIamPolicy()
	if err != nil {
		return nil, errors.Wrap(err, "GetIamPolicy")
	}
	users, err := policy.GetICloudusers()
	if err != nil {
		return nil, errors.Wrap(err, "policy.GetICloudusers")
	}
	for i := range users {
		if users[i].GetName() == name {
			return users[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (user *SClouduser) DetachSystemPolicy(role string) error {
	return user.policy.DetachPolicy(user.Name, role)
}

func (policy *SIamPolicy) DeleteUser(user string) error {
	change := false
	for i := range policy.Bindings {
		if utils.IsInStringArray(getUserName(user), policy.Bindings[i].Members) {
			change = true
			members := []string{}
			for _, member := range policy.Bindings[i].Members {
				if member != getUserName(user) {
					members = append(members, member)
				}
			}
			policy.Bindings[i].Members = members
		}
	}
	if change {
		return policy.client.SetIamPlicy(policy)
	}
	return nil
}

func (user *SClouduser) Delete() error {
	return user.policy.DeleteUser(user.Name)
}

func (user *SClouduser) GetICloudgroups() ([]cloudprovider.ICloudgroup, error) {
	return []cloudprovider.ICloudgroup{}, nil
}

func (user *SClouduser) ResetPassword(password string) error {
	return cloudprovider.ErrNotSupported
}

func (client *SGoogleClient) GetICloudusers() ([]cloudprovider.IClouduser, error) {
	policy, err := client.GetIamPolicy()
	if err != nil {
		return nil, errors.Wrap(err, "GetIamPolicy")
	}
	return policy.GetICloudusers()
}

func (policy *SIamPolicy) GetICloudusers() ([]cloudprovider.IClouduser, error) {
	users := map[string]*SClouduser{}
	for _, binding := range policy.Bindings {
		for _, member := range binding.Members {
			if strings.HasPrefix(member, "user:") {
				user := strings.TrimPrefix(member, "user:")
				if _, ok := users[user]; !ok {
					users[user] = &SClouduser{Name: user, Roles: []string{}}
				}
				roles := users[user].Roles
				if !utils.IsInStringArray(binding.Role, roles) {
					roles = append(roles, binding.Role)
				}
				users[user].Roles = roles
			}
		}
	}
	cloudusers := []cloudprovider.IClouduser{}
	for i := range users {
		users[i].policy = policy
		cloudusers = append(cloudusers, users[i])
	}
	return cloudusers, nil
}
