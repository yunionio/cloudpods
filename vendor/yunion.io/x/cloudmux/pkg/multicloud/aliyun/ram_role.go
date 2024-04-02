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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/cloudid"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type sRoles struct {
	Role []SRole
}

type SRoles struct {
	Roles       sRoles
	Marker      string
	IsTruncated bool
}

type SRole struct {
	client *SAliyunClient

	Arn         string
	CreateDate  time.Time
	Description string
	RoleId      string
	RoleName    string

	AssumeRolePolicyDocument string
}

func (self *SRole) GetGlobalId() string {
	return self.Arn
}

func (self *SRole) GetName() string {
	return self.RoleName
}

func (self *SRole) GetICloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	policies, err := self.client.ListPoliciesForRole(self.RoleName)
	if err != nil {
		return nil, errors.Wrapf(err, "ListPoliciesForRole")
	}
	ret := []cloudprovider.ICloudpolicy{}
	for i := range policies {
		policies[i].client = self.client
		ret = append(ret, &policies[i])
	}
	return ret, nil
}

func (self *SRole) AttachPolicy(policyName string, policyType api.TPolicyType) error {
	return self.client.AttachPolicy2Role(utils.Capitalize(string(policyType)), policyName, self.RoleName)
}

func (self *SRole) DetachPolicy(policyName string, policyType api.TPolicyType) error {
	return self.client.DetachPolicyFromRole(utils.Capitalize(string(policyType)), policyName, self.RoleName)
}

func (self *SRole) Delete() error {
	return self.client.DeleteRole(self.RoleName)
}

func (self *SRole) GetDocument() *jsonutils.JSONDict {
	role, err := self.client.GetRole(self.RoleName)
	if err != nil {
		return nil
	}
	documet, err := jsonutils.Parse([]byte(role.AssumeRolePolicyDocument))
	if err != nil {
		return nil
	}
	return documet.(*jsonutils.JSONDict)
}

func (self *SRole) GetSAMLProvider() string {
	document := self.GetDocument()
	if document != nil {
		statement, err := document.GetArray("Statement")
		if err == nil {
			for i := range statement {
				if action, _ := statement[i].GetString("Action"); action == "sts:AssumeRole" {
					sp, _ := statement[i].GetString("Principal", "Federated")
					if len(sp) > 0 {
						return sp
					}
				}
			}
		}
	}
	return ""
}

func (self *SAliyunClient) GetICloudroles() ([]cloudprovider.ICloudrole, error) {
	roles := []SRole{}
	marker := ""
	for {
		part, err := self.ListRoles(marker, 1000)
		if err != nil {
			return nil, errors.Wrapf(err, "ListRoles(%s)", marker)
		}
		roles = append(roles, part.Roles.Role...)
		if len(part.Marker) == 0 {
			break
		}
		marker = part.Marker
	}
	ret := []cloudprovider.ICloudrole{}
	for i := range roles {
		roles[i].client = self
		ret = append(ret, &roles[i])
	}
	return ret, nil
}

func (self *SAliyunClient) ListRoles(offset string, limit int) (*SRoles, error) {
	if limit < 0 || limit > 1000 {
		limit = 1000
	}

	params := map[string]string{}
	if len(offset) > 0 {
		params["Marker"] = offset
	}
	if limit > 0 {
		params["MaxItems"] = fmt.Sprintf("%d", limit)
	}

	body, err := self.ramRequest("ListRoles", params)
	if err != nil {
		return nil, errors.Wrapf(err, "ListRoles")
	}

	roles := &SRoles{}
	err = body.Unmarshal(roles)
	if err != nil {
		return nil, errors.Wrap(err, "body.Unmarshal(&")
	}
	return roles, nil
}

func (self *SAliyunClient) CreateRole(roleName string, document string, desc string) (*SRole, error) {
	params := make(map[string]string)
	params["RoleName"] = roleName
	params["AssumeRolePolicyDocument"] = document
	if len(desc) > 0 {
		params["Description"] = desc
	}

	body, err := self.ramRequest("CreateRole", params)
	if err != nil {
		return nil, errors.Wrap(err, "CreateRole")
	}

	role := SRole{client: self}
	err = body.Unmarshal(&role, "Role")
	if err != nil {
		return nil, errors.Wrap(err, "body.Unmarshal")
	}

	return &role, nil
}

func (self *SAliyunClient) GetRole(roleName string) (*SRole, error) {
	params := make(map[string]string)
	params["RoleName"] = roleName

	body, err := self.ramRequest("GetRole", params)
	if err != nil {
		return nil, errors.Wrap(err, "GetRole")
	}

	role := SRole{client: self}
	err = body.Unmarshal(&role, "Role")
	if err != nil {
		return nil, errors.Wrap(err, "body.Unmarshal")
	}

	return &role, nil
}

func (self *SAliyunClient) ListPoliciesForRole(name string) ([]SPolicy, error) {
	params := map[string]string{
		"RoleName": name,
	}
	resp, err := self.ramRequest("ListPoliciesForRole", params)
	if err != nil {
		return nil, errors.Wrapf(err, "ListPoliciesForRole")
	}
	policies := []SPolicy{}
	err = resp.Unmarshal(&policies, "Policies", "Policy")
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return policies, nil
}

func (self *SAliyunClient) DetachPolicyFromRole(policyType, policyName, roleName string) error {
	params := map[string]string{
		"PolicyName": policyName,
		"PolicyType": policyType,
		"RoleName":   roleName,
	}
	_, err := self.ramRequest("DetachPolicyFromRole", params)
	return err
}
