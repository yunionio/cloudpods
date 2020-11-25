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

	"github.com/pkg/errors"
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
	client *SApsaraClient

	Arn         string
	CreateDate  time.Time
	Description string
	RoleId      string
	RoleName    string

	AssumeRolePolicyDocument string
}

func (self *SApsaraClient) ListRoles(offset string, limit int) (*SRoles, error) {
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

func (self *SApsaraClient) CreateRole(roleName string, document string, desc string) (*SRole, error) {
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

func (self *SApsaraClient) GetRole(roleName string) (*SRole, error) {
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

func (self *SApsaraClient) ListPoliciesForRole(name string) ([]SPolicy, error) {
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

func (self *SApsaraClient) DetachPolicyFromRole(policyType, policyName, roleName string) error {
	params := map[string]string{
		"PolicyName": policyName,
		"PolicyType": policyType,
		"RoleName":   roleName,
	}
	_, err := self.ramRequest("DetachPolicyFromRole", params)
	return err
}
