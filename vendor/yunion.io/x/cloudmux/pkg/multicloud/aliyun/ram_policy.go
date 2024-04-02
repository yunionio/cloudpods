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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/cloudid"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

const (
	POLICY_TYPE_SYSTEM = "System"
	POLICY_TYPE_CUSTOM = "Custom"
)

/**
 {"AttachmentCount":0,
"CreateDate":"2018-10-12T05:05:16Z",
"DefaultVersion":"v1",
"Description":"只读访问Data Lake Analytics的权限",
"PolicyName":"AliyunDLAReadOnlyAccess",
"PolicyType":"System",
"UpdateDate":"2018-10-12T05:05:16Z"}
*/

type SDefaultPolicyVersion struct {
	CreateDate       time.Time
	IsDefaultVersion bool
	PolicyDocument   string
	VersionId        string
}

type SPolicyDetails struct {
	Policy               SPolicy
	DefaultPolicyVersion SDefaultPolicyVersion
}

type sPolicies struct {
	Policy []SPolicy
}

type SPolicies struct {
	Policies    sPolicies
	Marker      string
	IsTruncated bool
}

type SPolicy struct {
	client          *SAliyunClient
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

func (policy *SPolicy) GetDescription() string {
	return policy.Description
}

func (policy *SPolicy) GetGlobalId() string {
	return policy.PolicyName
}

func (policy *SPolicy) UpdateDocument(document *jsonutils.JSONDict) error {
	return policy.client.CreatePolicyVersion(policy.PolicyName, document.String(), true)
}

func (policy *SPolicy) Delete() error {
	return policy.client.DeletePolicy(policy.PolicyType, policy.PolicyName)
}

func (policy *SPolicy) GetPolicyType() api.TPolicyType {
	if policy.PolicyType == "System" {
		return api.PolicyTypeSystem
	}
	return api.PolicyTypeCustom
}

func (policy *SPolicy) GetDocument() (*jsonutils.JSONDict, error) {
	details, err := policy.client.GetPolicy(policy.PolicyType, policy.PolicyName)
	if err != nil {
		return nil, errors.Wrapf(err, "GetPolicy(%s,%s)", policy.PolicyType, policy.PolicyName)
	}
	obj, err := jsonutils.Parse([]byte(details.DefaultPolicyVersion.PolicyDocument))
	if err != nil {
		return nil, errors.Wrap(err, "jsonutils.Parse")
	}
	return obj.(*jsonutils.JSONDict), nil
}

func (self *SAliyunClient) GetICloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	ret := []cloudprovider.ICloudpolicy{}
	offset := ""
	for {
		part, err := self.ListPolicies("", offset, 1000)
		if err != nil {
			return nil, errors.Wrapf(err, "ListPolicies")
		}
		for i := range part.Policies.Policy {
			part.Policies.Policy[i].client = self
			ret = append(ret, &part.Policies.Policy[i])
		}
		offset = part.Marker
		if len(offset) == 0 || !part.IsTruncated {
			break
		}
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
func (self *SAliyunClient) ListPolicies(policyType string, offset string, limit int) (*SPolicies, error) {
	if limit < 1 || limit > 1000 {
		limit = 1000
	}
	params := map[string]string{
		"MaxItems": fmt.Sprintf("%d", limit),
	}
	if len(policyType) > 0 {
		params["PolicyType"] = policyType
	}
	if len(offset) > 0 {
		params["Marker"] = offset
	}

	body, err := self.ramRequest("ListPolicies", params)
	if err != nil {
		return nil, errors.Wrapf(err, "ListPolicies")
	}
	policies := &SPolicies{}

	err = body.Unmarshal(&policies)
	if err != nil {
		return nil, errors.Wrapf(err, "body.Unmarshal")
	}

	return policies, nil
}

func (self *SAliyunClient) GetPolicy(policyType string, policyName string) (*SPolicyDetails, error) {
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

	policy := SPolicyDetails{}

	err = body.Unmarshal(&policy)
	if err != nil {
		return nil, err
	}

	return &policy, nil
}

type SStatement struct {
	Action   []string `json:"Action,allowempty"`
	Effect   string   `json:"Effect"`
	Resource []string `json:"Resource"`
}

type SPolicyDocument struct {
	Statement []SStatement `json:"Statement,allowempty"`
	Version   string       `json:"Version"`
}

func (self *SAliyunClient) CreateICloudpolicy(opts *cloudprovider.SCloudpolicyCreateOptions) (cloudprovider.ICloudpolicy, error) {
	if opts.Document == nil {
		return nil, errors.Error("nil document")
	}
	policy, err := self.CreatePolicy(opts.Name, opts.Document.String(), opts.Desc)
	if err != nil {
		return nil, errors.Wrapf(err, "CreatePolicy")
	}
	return policy, nil
}

func (self *SAliyunClient) CreatePolicy(name string, document string, desc string) (*SPolicy, error) {
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

	policy := SPolicy{client: self}

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

func (self *SAliyunClient) AttachPolicy2Role(policyType string, policyName string, roleName string) error {
	params := make(map[string]string)
	params["PolicyType"] = policyType
	params["PolicyName"] = policyName
	params["RoleName"] = roleName

	_, err := self.ramRequest("AttachPolicyToRole", params)
	if err != nil {
		return errors.Wrap(err, "AttachPolicyToRole")
	}

	return nil
}

func (self *SAliyunClient) CreatePolicyVersion(name, document string, isDefault bool) error {
	params := map[string]string{
		"PolicyName":     name,
		"PolicyDocument": document,
		"RotateStrategy": "DeleteOldestNonDefaultVersionWhenLimitExceeded",
	}
	if isDefault {
		params["SetAsDefault"] = "true"
	}
	_, err := self.ramRequest("CreatePolicyVersion", params)
	return err
}
