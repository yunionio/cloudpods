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

package aws

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

func (cli *SAwsClient) getIamArn(arn string) string {
	switch cli.GetAccessEnv() {
	case api.CLOUD_ACCESS_ENV_AWS_GLOBAL:
		return fmt.Sprintf("arn:aws:iam::aws:policy/%s", arn)
	default:
		return fmt.Sprintf("arn:aws-cn:iam::aws:policy/%s", arn)
	}
}

func getIamCommonArn(arn string) string {
	if info := strings.Split(arn, "/"); len(info) > 1 {
		return strings.Join(info[1:], "/")
	}
	return arn
}

func (self *SAwsClient) ListAllPolicies() ([]SPolicy, error) {
	policies := []SPolicy{}
	marker := ""
	for {
		part, err := self.ListPolicies(marker, 1000, false, "", "", "")
		if err != nil {
			return nil, errors.Wrap(err, "GetPolicies")
		}
		policies = append(policies, part.Policies...)
		marker = part.Marker
		if len(marker) == 0 {
			break
		}
	}
	return policies, nil
}

func (self *SAwsClient) GetICloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	policies, err := self.ListAllPolicies()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudpolicy{}
	for i := range policies {
		ret = append(ret, &policies[i])
	}
	return ret, nil
}

func (cli *SAwsClient) ListUserAttachedPolicies(userName string) ([]SAttachedPolicy, error) {
	policies := []SAttachedPolicy{}
	marker := ""
	for {
		part, err := cli.ListAttachedUserPolicies(userName, marker, 1000, "")
		if err != nil {
			return nil, errors.Wrap(err, "GetAttachedPolicies")
		}
		policies = append(policies, part.AttachedPolicies...)
		marker = part.Marker
		if len(marker) == 0 {
			break
		}
	}
	return policies, nil
}

type SClouduserPolicy struct {
	Member []string `xml:"member"`
}

type SClouduserPolicies struct {
	PolicyNames SClouduserPolicy `xml:"PolicyNames"`
	IsTruncated bool             `xml:"IsTruncated"`
}

func (self *SAwsClient) ListUserpolicies(userName string, marker string, maxItems int) (*SClouduserPolicies, error) {
	if maxItems <= 0 || maxItems > 1000 {
		maxItems = 1000
	}
	params := map[string]string{
		"UserName": userName,
		"MaxItems": fmt.Sprintf("%d", maxItems),
	}
	if len(marker) > 0 {
		params["Marker"] = marker
	}
	policies := SClouduserPolicies{}
	err := self.iamRequest("ListUserPolicies", params, &policies)
	if err != nil {
		return nil, errors.Wrap(err, "iamRequest.ListUserPolicies")
	}
	return &policies, nil
}

type SPolicy struct {
	PolicyName       string    `xml:"PolicyName"`
	DefaultVersionId string    `xml:"DefaultVersionId"`
	PolicyId         string    `xml:"PolicyId"`
	Path             string    `xml:"Path"`
	Arn              string    `xml:"Arn"`
	AttachmentCount  int       `xml:"AttachmentCount"`
	CreateDate       time.Time `xml:"CreateDate"`
	UpdateDate       time.Time `xml:"UpdateDate"`
}

type SPolicies struct {
	IsTruncated bool      `xml:"IsTruncated"`
	Marker      string    `xml:"Marker"`
	Policies    []SPolicy `xml:"Policies>member"`
}

func (policy *SPolicy) GetName() string {
	return policy.PolicyName
}

func (policy *SPolicy) GetGlobalId() string {
	return getIamCommonArn(policy.Arn)
}

func (policy *SPolicy) GetDescription() string {
	return ""
}

func (self *SAwsClient) ListPolicies(marker string, maxItems int, onlyAttached bool, pathPrefix string, policyUsageFilter string, scope string) (*SPolicies, error) {
	if maxItems <= 0 || maxItems > 1000 {
		maxItems = 1000
	}

	params := map[string]string{
		"MaxItems": fmt.Sprintf("%d", maxItems),
	}
	if len(marker) > 0 {
		params["Marker"] = marker
	}
	if onlyAttached {
		params["OnlyAttached"] = "true"
	}
	if len(pathPrefix) > 0 {
		params["PathPrefix"] = pathPrefix
	}
	if len(policyUsageFilter) > 0 {
		params["PolicyUsageFilter"] = policyUsageFilter
	}
	if len(scope) > 0 {
		params["Scope"] = scope
	}
	policies := &SPolicies{}
	err := self.iamRequest("ListPolicies", params, policies)
	if err != nil {
		return nil, errors.Wrap(err, "iamRequest.ListPolicies")
	}
	return policies, nil
}

func (self *SAwsClient) ListAttachedUserPolicies(userName string, marker string, maxItems int, pathPrefix string) (*SAttachedPolicies, error) {
	if maxItems <= 0 || maxItems > 1000 {
		maxItems = 1000
	}
	params := map[string]string{
		"MaxItems": fmt.Sprintf("%d", maxItems),
	}
	if len(marker) > 0 {
		params["Marker"] = marker
	}
	if len(pathPrefix) > 0 {
		params["PathPrefix"] = pathPrefix
	}
	if len(userName) > 0 {
		params["UserName"] = userName
	}
	policies := &SAttachedPolicies{}
	err := self.iamRequest("ListAttachedUserPolicies", params, policies)
	if err != nil {
		return nil, errors.Wrap(err, "ListAttachedUserPolicies")
	}
	return policies, nil
}

type SAttachedPolicies struct {
	IsTruncated      bool              `xml:"IsTruncated"`
	Marker           string            `xml:"Marker"`
	AttachedPolicies []SAttachedPolicy `xml:"AttachedPolicies>member"`
}

type SAttachedPolicy struct {
	client     *SAwsClient
	PolicyName string `xml:"PolicyName"`
	PolicyArn  string `xml:"PolicyArn"`
}

func (policy *SAttachedPolicy) GetGlobalId() string {
	return getIamCommonArn(policy.PolicyArn)
}

func (policy *SAttachedPolicy) GetName() string {
	return policy.PolicyName
}

func (policy *SAttachedPolicy) GetDescription() string {
	return ""
}

func (self *SAwsClient) ListAttachedGroupPolicies(name string, marker string, maxItems int) (*SAttachedPolicies, error) {
	if maxItems <= 0 || maxItems > 1000 {
		maxItems = 1000
	}

	params := map[string]string{
		"GroupName": name,
		"MaxItems":  fmt.Sprintf("%d", maxItems),
	}
	if len(marker) > 0 {
		params["Marker"] = marker
	}
	policies := &SAttachedPolicies{}
	err := self.iamRequest("ListAttachedGroupPolicies", params, policies)
	if err != nil {
		return nil, errors.Wrap(err, "iamRequest.ListGroupPolicies")
	}
	return policies, nil
}
