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

package volcengine

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/cloudid"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type PolicyDocument struct {
	Statement []struct {
		Effect    string
		Action    []string
		Principal struct {
			Federated []string
		}
	}
}

type SRole struct {
	client *SVolcEngineClient

	RoleName            string
	DisplayName         string
	TrustPolicyDocument string
	Description         string
}

func (self *SRole) GetGlobalId() string {
	return self.RoleName
}

func (self *SRole) GetName() string {
	return self.RoleName
}

func (self *SRole) GetICloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	policies, err := self.client.ListAttachedRolePolicies(self.RoleName)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudpolicy{}
	for i := range policies {
		policies[i].client = self.client
		ret = append(ret, &policies[i])
	}
	return ret, nil
}

func (self *SRole) AttachPolicy(policyName string, policyType api.TPolicyType) error {
	return self.client.AttachRolePolicy(self.RoleName, policyName, utils.Capitalize(string(policyType)))
}

func (self *SRole) DetachPolicy(policyName string, policyType api.TPolicyType) error {
	return self.client.DetachRolePolicy(self.RoleName, policyName, utils.Capitalize(string(policyType)))
}

func (self *SRole) Delete() error {
	return self.client.DeleteRole(self.RoleName)
}

func (self *SRole) GetDocument() *jsonutils.JSONDict {
	doc, err := jsonutils.ParseString(self.TrustPolicyDocument)
	if err != nil {
		return nil
	}
	return doc.(*jsonutils.JSONDict)
}

func (self *SRole) GetSAMLProvider() string {
	document := self.GetDocument()
	if document == nil {
		return ""
	}
	info := &PolicyDocument{}
	document.Unmarshal(info)
	for _, statement := range info.Statement {
		for _, sp := range statement.Principal.Federated {
			info := strings.Split(sp, "/")
			if len(info) == 2 {
				return info[1]
			}
		}
	}
	return ""
}

func (self *SVolcEngineClient) GetICloudroles() ([]cloudprovider.ICloudrole, error) {
	roles, err := self.ListRoles()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudrole{}
	for i := range roles {
		roles[i].client = self
		ret = append(ret, &roles[i])
	}
	return ret, nil
}

func (client *SVolcEngineClient) ListRoles() ([]SRole, error) {
	params := map[string]string{
		"Limit": "50",
	}
	offset := 0
	ret := []SRole{}
	for {
		params["Offset"] = fmt.Sprintf("%d", offset)
		resp, err := client.iamRequest("", "ListRoles", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			RoleMetadata []SRole
			Total        int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.RoleMetadata...)
		if len(part.RoleMetadata) == 0 || len(ret) >= part.Total {
			break
		}
		offset = len(ret)
	}
	return ret, nil
}

func (client *SVolcEngineClient) GetRole(name string) (*SRole, error) {
	params := map[string]string{
		"RoleName": name,
	}
	resp, err := client.iamRequest("", "GetRole", params)
	if err != nil {
		return nil, err
	}
	ret := &SRole{client: client}
	err = resp.Unmarshal(ret, "Role")
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (client *SVolcEngineClient) CreateRole(name, statement, desc string) (*SRole, error) {
	params := map[string]string{
		"RoleName":            name,
		"DisplayName":         name,
		"TrustPolicyDocument": statement,
		"Description":         desc,
	}
	resp, err := client.iamRequest("", "CreateRole", params)
	if err != nil {
		return nil, err
	}
	ret := &SRole{client: client}
	err = resp.Unmarshal(ret, "Role")
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (client *SVolcEngineClient) DeleteRole(name string) error {
	params := map[string]string{
		"RoleName": name,
	}
	_, err := client.iamRequest("", "DeleteRole", params)
	return err
}

func (client *SVolcEngineClient) ListAttachedRolePolicies(name string) ([]SPolicy, error) {
	params := map[string]string{
		"RoleName": name,
	}
	resp, err := client.iamRequest("", "ListAttachedRolePolicies", params)
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

func (client *SVolcEngineClient) AttachRolePolicy(name, policy, policyType string) error {
	params := map[string]string{
		"RoleName":   name,
		"PolicyName": policy,
		"PolicyType": policyType,
	}
	_, err := client.iamRequest("", "AttachRolePolicy", params)
	return err
}

func (client *SVolcEngineClient) DetachRolePolicy(name, policy, policyType string) error {
	params := map[string]string{
		"RoleName":   name,
		"PolicyName": policy,
		"PolicyType": policyType,
	}
	_, err := client.iamRequest("", "DetachRolePolicy", params)
	return err
}
