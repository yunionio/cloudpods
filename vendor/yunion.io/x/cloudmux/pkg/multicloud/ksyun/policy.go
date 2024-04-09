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

package ksyun

import (
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

type SPolicy struct {
	client *SKsyunClient

	CreateDate       time.Time
	DefaultVersionId string
	Description      string
	Krn              string
	PolicyKrn        string
	Path             string
	PolicyId         string
	PolicyName       string
	ServiceId        string
	ServiceName      string
	ServiceViewName  string
	PolicyType       int
	CreateMode       int
	UpdateDate       time.Time
	AttachmentCount  int
}

func (policy *SPolicy) GetName() string {
	return policy.PolicyName
}

func (policy *SPolicy) GetDescription() string {
	return policy.Description
}

func (policy *SPolicy) GetGlobalId() string {
	return policy.Krn + policy.PolicyKrn
}

func (policy *SPolicy) GetPolicyType() api.TPolicyType {
	if policy.PolicyType == 1 {
		return api.PolicyTypeSystem
	}
	return api.PolicyTypeCustom
}

func (policy *SPolicy) UpdateDocument(document *jsonutils.JSONDict) error {
	return cloudprovider.ErrNotImplemented
}

func (policy *SPolicy) Delete() error {
	return policy.client.DeletePolicy(policy.Krn)
}

func (policy *SPolicy) GetDocument() (*jsonutils.JSONDict, error) {
	doc, err := policy.client.GetPolicyVersion(policy.Krn, policy.DefaultVersionId)
	if err != nil {
		return nil, err
	}
	obj, err := jsonutils.ParseString(doc.Document)
	if err != nil {
		return nil, errors.Wrapf(err, "ParseString %s", doc.Document)
	}
	return obj.(*jsonutils.JSONDict), nil
}

func (client *SKsyunClient) GetICloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	policies, err := client.ListPolicies("")
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudpolicy{}
	for i := range policies {
		policies[i].client = client
		ret = append(ret, &policies[i])
	}
	return ret, nil
}

func (client *SKsyunClient) ListPolicies(scope string) ([]SPolicy, error) {
	params := map[string]string{
		"MaxItems": "100",
	}
	if len(scope) > 0 {
		params["Scope"] = scope
	}
	ret := []SPolicy{}
	for {
		resp, err := client.iamRequest("", "ListPolicies", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			Policies struct {
				Member []SPolicy
			}
			Marker string
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.Policies.Member...)
		if len(part.Policies.Member) == 0 || len(part.Marker) == 0 {
			break
		}
		params["Marker"] = part.Marker
	}
	return ret, nil
}

func (client *SKsyunClient) DeletePolicy(krn string) error {
	params := map[string]string{
		"PolicyKrn": krn,
	}
	_, err := client.iamRequest("", "DeletePolicy", params)
	return err
}

type SPolicyVersion struct {
	Document string
}

func (client *SKsyunClient) GetPolicyVersion(krn, version string) (*SPolicyVersion, error) {
	params := map[string]string{
		"PolicyKrn": krn,
		"VersionId": version,
	}
	resp, err := client.iamRequest("", "GetPolicyVersion", params)
	if err != nil {
		return nil, err
	}
	ret := &SPolicyVersion{}
	err = resp.Unmarshal(ret, "PolicyVersion")
	if err != nil {
		return nil, err
	}
	return ret, nil
}
