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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

func (self *SAliyunClient) ramRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := self.getDefaultClient()
	if err != nil {
		return nil, err
	}
	return jsonRequest(cli, "ram.aliyuncs.com", ALIYUN_RAM_API_VERSION, apiName, params, self.debug)
}

type SRole struct {
	Arn         string
	CreateDate  time.Time
	Description string
	RoleId      string
	RoleName    string

	AssumeRolePolicyDocument string
}

func (self *SAliyunClient) ListRoles() ([]SRole, error) {
	body, err := self.ramRequest("ListRoles", nil)
	if err != nil {
		log.Errorf("listRoles fail %s", err)
		return nil, err
	}

	roles := make([]SRole, 0)

	err = body.Unmarshal(&roles, "Roles", "Role")
	if err != nil {
		return nil, err
	}

	return roles, nil
}

func (self *SAliyunClient) GetRole(roleName string) (*SRole, error) {
	params := make(map[string]string)
	params["RoleName"] = roleName

	body, err := self.ramRequest("GetRole", params)
	if err != nil {
		if isError(err, "EntityNotExist.Role") {
			return nil, cloudprovider.ErrNotFound
		}
		return nil, err
	}

	role := SRole{}

	err = body.Unmarshal(&role, "Role")
	if err != nil {
		return nil, err
	}

	return &role, nil
}

func (self *SAliyunClient) createRole(roleName string, document string, desc string) (*SRole, error) {
	params := make(map[string]string)
	params["RoleName"] = roleName
	params["AssumeRolePolicyDocument"] = document
	if len(desc) > 0 {
		params["Description"] = desc
	}

	body, err := self.ramRequest("CreateRole", params)
	if err != nil {
		return nil, err
	}

	role := SRole{}

	err = body.Unmarshal(&role, "Role")
	if err != nil {
		return nil, err
	}

	return &role, nil
}

/**
 {"AttachmentCount":0,
"CreateDate":"2018-10-12T05:05:16Z",
"DefaultVersion":"v1",
"Description":"只读访问Data Lake Analytics的权限",
"PolicyName":"AliyunDLAReadOnlyAccess",
"PolicyType":"System",
"UpdateDate":"2018-10-12T05:05:16Z"}
*/

type SPolicy struct {
	AttachmentCount int
	CreateDate      time.Time
	UpdateDate      time.Time
	DefaultVersion  string
	Description     string
	PolicyName      string
	PolicyType      string
}

func (policy *SPolicy) GetName() string {
	return policy.PolicyName
}

func (policy *SPolicy) GetPolicyType() string {
	return policy.PolicyType
}

func (policy *SPolicy) GetDescription() string {
	return policy.Description
}

func (policy *SPolicy) GetGlobalId() string {
	return policy.PolicyName
}

func (self *SAliyunClient) GetISystemCloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	policies, err := self.ListPolicies("System", "")
	if err != nil {
		return nil, errors.Wrap(err, "GetPolicy")
	}
	ret := []cloudprovider.ICloudpolicy{}
	for i := range policies {
		ret = append(ret, &policies[i])
	}
	return ret, nil
}

func (self *SAliyunClient) GetICloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	policies, err := self.ListPolicies("", "")
	if err != nil {
		return nil, errors.Wrap(err, "GetPolicy")
	}
	ret := []cloudprovider.ICloudpolicy{}
	for i := range policies {
		ret = append(ret, &policies[i])
	}
	return ret, nil
}

func (self *SAliyunClient) AttachPolicyToUser(policyName, policyType, userName string) error {
	params := map[string]string{
		"UserName":   userName,
		"PolicyName": policyName,
		"PolicyType": policyType,
	}
	_, err := self.ramRequest("AttachPolicyToUser", params)
	if err != nil && !strings.Contains(err.Error(), "EntityAlreadyExists.User.Policy") {
		return errors.Wrap(err, "AttachPolicyToUser")
	}
	return nil
}

func (self *SAliyunClient) DetachPolicyFromUser(policyName, policyType, userName string) error {
	if len(policyType) == 0 {
		policyType = "System"
	}
	params := map[string]string{
		"UserName":   userName,
		"PolicyName": policyName,
		"PolicyType": policyType,
	}
	_, err := self.ramRequest("DetachPolicyFromUser", params)
	if err != nil && errors.Cause(err) != cloudprovider.ErrNotFound {
		return errors.Wrap(err, "DetachPolicyFromUser")
	}
	return nil
}

// https://help.aliyun.com/document_detail/28719.html?spm=a2c4g.11174283.6.764.27055662H6TGg5
func (self *SAliyunClient) ListPolicies(policyType string, role string) ([]SPolicy, error) {
	var action string
	params := make(map[string]string)
	if len(role) > 0 {
		params["RoleName"] = role
		action = "ListPoliciesForRole"
	} else {
		params["MaxItems"] = "1000"
		if len(policyType) > 0 {
			params["PolicyType"] = policyType
		}
		action = "ListPolicies"
	}

	body, err := self.ramRequest(action, params)
	if err != nil {
		log.Errorf("listPolicies fail %s", err)
		return nil, err
	}

	policies := make([]SPolicy, 0)

	err = body.Unmarshal(&policies, "Policies", "Policy")
	if err != nil {
		return nil, err
	}

	return policies, nil
}

func (self *SAliyunClient) GetPolicy(policyType string, policyName string) (*SPolicy, error) {
	params := make(map[string]string)
	params["PolicyType"] = policyType
	params["PolicyName"] = policyName

	body, err := self.ramRequest("GetPolicy", params)
	if err != nil {
		if isError(err, "EntityNotExist.Role") {
			return nil, cloudprovider.ErrNotFound
		}
		return nil, err
	}

	policy := SPolicy{}

	err = body.Unmarshal(&policy, "Policy")
	if err != nil {
		return nil, err
	}

	return &policy, nil
}

func (self *SAliyunClient) createPolicy(name string, document string, desc string) (*SPolicy, error) {
	params := make(map[string]string)
	params["PolicyName"] = name
	params["PolicyDocument"] = document
	if len(desc) > 0 {
		params["Description"] = desc
	}

	body, err := self.ramRequest("CreatePolicy", params)
	if err != nil {
		return nil, err
	}

	policy := SPolicy{}

	err = body.Unmarshal(&policy, "Policy")
	if err != nil {
		return nil, err
	}

	return &policy, nil
}

func (self *SAliyunClient) DeletePolicy(policyType string, policyName string) error {
	params := make(map[string]string)
	params["PolicyName"] = policyName
	params["PolicyType"] = policyType

	_, err := self.ramRequest("DeletePolicy", params)
	return err
}

func (self *SAliyunClient) DeleteRole(roleName string) error {
	params := make(map[string]string)
	params["RoleName"] = roleName

	_, err := self.ramRequest("DeleteRole", params)
	return err
}

func (self *SAliyunClient) attachPolicy2Role(policyType string, policyName string, roleName string) error {
	params := make(map[string]string)
	params["PolicyType"] = policyType
	params["PolicyName"] = policyName
	params["RoleName"] = roleName

	_, err := self.ramRequest("AttachPolicyToRole", params)
	if err != nil {
		return err
	}

	return nil
}
