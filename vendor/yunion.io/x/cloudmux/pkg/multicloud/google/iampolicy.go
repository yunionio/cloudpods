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
	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/cloudid"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
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

func (self *SGoogleClient) CreateICloudpolicy(opts *cloudprovider.SCloudpolicyCreateOptions) (cloudprovider.ICloudpolicy, error) {
	permission := struct {
		IncludedPermissions []string
	}{}
	err := opts.Document.Unmarshal(&permission)
	if err != nil {
		return nil, errors.Wrapf(err, "Document.Unmarshal(")
	}
	return self.CreateRole(permission.IncludedPermissions, opts.Name, opts.Desc)
}

func (self *SGoogleClient) CreateRole(permissions []string, name, desc string) (*SRole, error) {
	resource := fmt.Sprintf("projects/%s/roles", self.projectId)
	params := map[string]interface{}{
		"roleId": strings.ReplaceAll(stringutils.UUID4(), "-", "_"),
		"role": map[string]interface{}{
			"title":               name,
			"description":         desc,
			"includedPermissions": permissions,
			"stage":               "GA",
		},
	}
	resp, err := self.iamPost(resource, nil, jsonutils.Marshal(params))
	if err != nil {
		return nil, errors.Wrap(err, "managerPost")
	}
	role := SRole{client: self}
	err = resp.Unmarshal(&role)
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return &role, nil
}

type SClouduser struct {
	multicloud.SBaseClouduser

	policy *SIamPolicy
	Name   string
	Roles  []string
}

func (self *SClouduser) GetEmailAddr() string {
	return ""
}

func (self *SClouduser) GetInviteUrl() string {
	return ""
}

func (self *SGoogleClient) GetICloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	roles, err := self.GetRoles("")
	if err != nil {
		return nil, errors.Wrap(err, "GetRoles")
	}
	ret := []cloudprovider.ICloudpolicy{}
	for i := range roles {
		roles[i].client = self
		ret = append(ret, &roles[i])
	}
	return ret, nil
}

type SRole struct {
	client *SGoogleClient

	Name                string
	Title               string
	Description         string
	IncludedPermissions []string
	Stage               string
	Etag                string
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

func (role *SRole) GetPolicyType() api.TPolicyType {
	return api.PolicyTypeSystem
}

func (role *SRole) UpdateDocument(document *jsonutils.JSONDict) error {
	permissions := struct {
		IncludedPermissions []string
	}{}
	err := document.Unmarshal(&permissions)
	if err != nil {
		return errors.Wrapf(err, "document.Unmarshal")
	}
	return role.client.UpdateRole(role.Name, permissions.IncludedPermissions)
}

func (role *SRole) Delete() error {
	return role.client.DeleteRole(role.Name)
}

func (role *SRole) GetDocument() (*jsonutils.JSONDict, error) {
	permissions := jsonutils.Marshal(role.IncludedPermissions)
	result := jsonutils.NewDict()
	result.Add(permissions, "includedPermissions")
	return result, nil
}

func (self *SGoogleClient) GetRole(roleId string) (*SRole, error) {
	role := &SRole{}
	resp, err := self.iamGet(roleId)
	if err != nil {
		return nil, errors.Wrapf(err, "iamGet(%s)", roleId)
	}
	err = resp.Unmarshal(role)
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return role, nil
}

// https://cloud.google.com/iam/docs/reference/rest/v1/roles/list
func (self *SGoogleClient) GetRoles(projectId string) ([]SRole, error) {
	roles := []SRole{}
	params := map[string]string{"view": "FULL"}
	resource := "roles"
	if len(projectId) > 0 {
		resource = fmt.Sprintf("projects/%s/roles", projectId)
	}
	err := self.iamListAll(resource, params, &roles)
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

func (user *SClouduser) GetICloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	ret := []cloudprovider.ICloudpolicy{}
	for _, roleStr := range user.Roles {
		if strings.HasPrefix(roleStr, "roles/") {
			role, err := user.policy.client.GetRole(roleStr)
			if err != nil {
				return nil, errors.Wrap(err, "GetRole")
			}
			ret = append(ret, role)
		}
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

func (user *SClouduser) AttachPolicy(role string, policyType api.TPolicyType) error {
	return user.policy.AttachPolicy(user.Name, []string{role})
}

func (user *SClouduser) AttachCustomPolicy(role string) error {
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
	return nil, cloudprovider.ErrNotImplemented
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
	return &SClouduser{policy: policy, Name: name, Roles: []string{}}, nil
}

func (user *SClouduser) DetachPolicy(role string, policyType api.TPolicyType) error {
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

func (self *SGoogleClient) DeleteRole(id string) error {
	return self.iamDelete(id, nil)
}

func (self *SGoogleClient) UpdateRole(id string, permissions []string) error {
	query := map[string]string{
		"updateMask": "includedPermissions",
	}
	params := map[string]interface{}{
		"includedPermissions": permissions,
	}
	_, err := self.iamPatch(id, query, jsonutils.Marshal(params))
	return err
}
