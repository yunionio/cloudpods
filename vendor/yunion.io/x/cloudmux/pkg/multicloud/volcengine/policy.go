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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/cloudid"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

const (
	POLICY_TYPE_SYSTEM = "System"
	POLICY_TYPE_CUSTOM = "Custom"
)

type SPolicy struct {
	client *SVolcEngineClient

	CreateDate          string
	UpdateDate          string
	PolicyDocument      string
	Status              string
	PolicyName          string
	PolicyType          string
	Description         string
	Category            string
	IsServiceRolePolicy int
	AttachmentCount     int
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

func (policy *SPolicy) GetPolicyType() api.TPolicyType {
	if policy.PolicyType == "System" {
		return api.PolicyTypeSystem
	}
	return api.PolicyTypeCustom
}

func (policy *SPolicy) UpdateDocument(document *jsonutils.JSONDict) error {
	return cloudprovider.ErrNotImplemented
}

func (policy *SPolicy) Delete() error {
	return policy.client.DeletePolicy(policy.PolicyName)
}

func (policy *SPolicy) GetDocument() (*jsonutils.JSONDict, error) {
	doc, err := jsonutils.Parse([]byte(policy.PolicyDocument))
	if err != nil {
		return nil, err
	}
	ret, ok := doc.(*jsonutils.JSONDict)
	if !ok {
		return nil, errors.Wrapf(cloudprovider.ErrNotSupported, policy.PolicyDocument)
	}
	return ret, nil
}

func (self *SVolcEngineClient) GetICloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	policies, err := self.ListPolicies("")
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudpolicy{}
	for i := range policies {
		policies[i].client = self
		ret = append(ret, &policies[i])
	}
	return ret, nil
}

func (client *SVolcEngineClient) ListPolicies(scope string) ([]SPolicy, error) {
	params := map[string]string{
		"Limit": "50",
	}
	if len(scope) > 0 {
		params["Scope"] = scope
	}
	offset := 0
	ret := []SPolicy{}
	for {
		params["Offset"] = fmt.Sprintf("%d", offset)
		resp, err := client.iamRequest("", "ListPolicies", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			PolicyMetadata []SPolicy
			Total          int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.PolicyMetadata...)
		if len(part.PolicyMetadata) == 0 || len(ret) >= part.Total {
			break
		}
		offset = len(ret)
	}
	return ret, nil

}

func (client *SVolcEngineClient) DeletePolicy(name string) error {
	params := map[string]string{
		"PolicyName": name,
	}
	_, err := client.iamRequest("", "DeletePolicy", params)
	return err
}
