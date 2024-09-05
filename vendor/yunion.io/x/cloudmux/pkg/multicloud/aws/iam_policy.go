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
	"net/url"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/apis/cloudid"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SPolicies struct {
	IsTruncated bool      `xml:"IsTruncated"`
	Marker      string    `xml:"Marker"`
	Policies    []SPolicy `xml:"Policies>member"`
}

type SPolicy struct {
	client *SAwsClient

	PermissionsBoundaryUsageCount int       `xml:"PermissionsBoundaryUsageCount"`
	PolicyName                    string    `xml:"PolicyName"`
	Description                   string    `xml:"Description"`
	DefaultVersionId              string    `xml:"DefaultVersionId"`
	PolicyId                      string    `xml:"PolicyId"`
	Path                          string    `xml:"Path"`
	Arn                           string    `xml:"Arn"`
	IsAttachable                  bool      `xml:"IsAttachable"`
	AttachmentCount               int       `xml:"AttachmentCount"`
	CreateDate                    time.Time `xml:"CreateDate"`
	UpdateDate                    time.Time `xml:"UpdateDate"`
	PolicyType                    cloudid.TPolicyType
}

func (self *SPolicy) GetName() string {
	return self.PolicyName
}

func (self *SPolicy) GetGlobalId() string {
	return self.Arn
}

func (self *SPolicy) GetPolicyType() cloudid.TPolicyType {
	return self.PolicyType
}

func (self *SPolicy) GetDescription() string {
	policy, err := self.client.GetPolicy(self.Arn)
	if err != nil {
		return ""
	}
	return policy.Description
}

type SUserPolicies struct {
	PolicyNames []string `xml:"PolicyNames>member"`
	IsTruncated bool     `xml:"IsTruncated"`
}

func (self *SAwsClient) ListUserPolicies(userName string, marker string, maxItems int) (*SUserPolicies, error) {
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
	policies := SUserPolicies{}
	err := self.iamRequest("ListUserPolicies", params, &policies)
	if err != nil {
		return nil, errors.Wrap(err, "iamRequest.ListUserPolicies")
	}
	return &policies, nil
}

func (self *SPolicy) Delete() error {
	return self.client.DeletePolicy(self.Arn)
}

func (self *SPolicy) UpdateDocument(document *jsonutils.JSONDict) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SPolicy) GetDocument() (*jsonutils.JSONDict, error) {
	return self.client.GetDocument(self.Arn, self.DefaultVersionId)
}

func (self *SAwsClient) GetDocument(arn, versionId string) (*jsonutils.JSONDict, error) {
	version, err := self.GetPolicyVersion(arn, versionId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetPolicyVersion(%s, %s)", arn, versionId)
	}
	document, err := url.PathUnescape(version.Document)
	if err != nil {
		return nil, errors.Wrapf(err, "url.PathUnescape.document")
	}
	jsonObj, err := jsonutils.Parse([]byte(document))
	if err != nil {
		return nil, errors.Wrap(err, "jsonutils.Parse")
	}
	return jsonObj.(*jsonutils.JSONDict), nil
}

func (self *SAwsClient) GetICloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	ret := []cloudprovider.ICloudpolicy{}
	marker := ""
	for {
		part, err := self.ListPolicies(marker, 1000, false, "", "", "AWS")
		if err != nil {
			return nil, errors.Wrapf(err, "ListPolicies")
		}
		for i := range part.Policies {
			part.Policies[i].client = self
			part.Policies[i].PolicyType = cloudid.PolicyTypeSystem
			ret = append(ret, &part.Policies[i])
		}
		marker = part.Marker
		if len(marker) == 0 || !part.IsTruncated {
			break
		}
	}

	for {
		part, err := self.ListPolicies(marker, 1000, false, "", "", "Local")
		if err != nil {
			return nil, errors.Wrapf(err, "ListPolicies")
		}
		for i := range part.Policies {
			part.Policies[i].client = self
			part.Policies[i].PolicyType = cloudid.PolicyTypeCustom
			ret = append(ret, &part.Policies[i])
		}
		marker = part.Marker
		if len(marker) == 0 || !part.IsTruncated {
			break
		}
	}

	return ret, nil
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
	client *SAwsClient

	PolicyName string `xml:"PolicyName"`
	PolicyArn  string `xml:"PolicyArn"`
}

func (self *SAttachedPolicy) GetGlobalId() string {
	return self.PolicyArn
}

func (self *SAttachedPolicy) GetName() string {
	return self.PolicyName
}

func (self *SAttachedPolicy) GetPolicyType() cloudid.TPolicyType {
	return cloudid.PolicyTypeSystem
}

func (self *SAttachedPolicy) GetDescription() string {
	return ""
}

func (self *SAttachedPolicy) UpdateDocument(document *jsonutils.JSONDict) error {
	return self.client.CreatePolicyVersion(self.PolicyArn, document.String(), true)
}

func (self *SAttachedPolicy) GetDocument() (*jsonutils.JSONDict, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SAttachedPolicy) Delete() error {
	return cloudprovider.ErrNotImplemented
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

type SPolicyNames struct {
	PolicyNames []string `xml:"PolicyNames>member"`
	IsTruncated bool     `xml:"IsTruncated"`
}

func (self *SAwsClient) ListRolePolicies(roleName string, marker string, maxItems int) (*SPolicyNames, error) {
	if maxItems < 1 || maxItems > 1000 {
		maxItems = 1000
	}
	params := map[string]string{
		"RoleName": roleName,
		"MaxItems": fmt.Sprintf("%d", maxItems),
	}
	if len(marker) > 0 {
		params["Marker"] = marker
	}
	names := &SPolicyNames{}
	err := self.iamRequest("ListRolePolicies", params, names)
	if err != nil {
		return nil, errors.Wrapf(err, "ListRolePolicies")
	}
	return names, nil
}

func (self *SAwsClient) ListAttachedRolePolicies(roleName string, marker string, maxItems int, pathPrefix string) (*SAttachedPolicies, error) {
	if maxItems < 1 || maxItems > 1000 {
		maxItems = 1000
	}
	params := map[string]string{
		"RoleName": roleName,
		"MaxItems": fmt.Sprintf("%d", maxItems),
	}
	if len(marker) > 0 {
		params["marker"] = marker
	}
	policies := &SAttachedPolicies{}
	err := self.iamRequest("ListAttachedRolePolicies", params, policies)
	if err != nil {
		return nil, errors.Wrapf(err, "ListAttachedRolePolicies")
	}
	return policies, nil
}

func (self *SAwsClient) AttachRolePolicy(roleName, policyArn string) error {
	params := map[string]string{
		"PolicyArn": policyArn,
		"RoleName":  roleName,
	}
	return self.iamRequest("AttachRolePolicy", params, nil)
}

func (self *SAwsClient) DetachRolePolicy(roleName string, policyArn string) error {
	params := map[string]string{
		"PolicyArn": policyArn,
		"RoleName":  roleName,
	}
	err := self.iamRequest("DetachRolePolicy", params, nil)
	if err != nil && errors.Cause(err) != cloudprovider.ErrNotFound {
		return errors.Wrap(err, "DetachRolePolicy")
	}
	return nil
}

type SPolicyVersion struct {
	Document         string    `xml:"Document"`
	IsDefaultVersion bool      `xml:"IsDefaultVersion"`
	VersionId        string    `xml:"VersionId"`
	CreateDate       time.Time `xml:"CreateDate"`
}

type SPolicyVersions struct {
	Versions    []SPolicyVersion `xml:"Versions>member"`
	IsTruncated bool             `xml:"IsTruncated"`
	Marker      string           `xml:"Marker"`
}

func (self *SAwsClient) ListPolicyVersions(marker string, maxItems int, arn string) (*SPolicyVersions, error) {
	if maxItems < 1 || maxItems > 1000 {
		maxItems = 1000
	}
	params := map[string]string{
		"MaxItems":  fmt.Sprintf("%d", maxItems),
		"PolicyArn": arn,
	}
	if len(marker) > 0 {
		params["Marker"] = marker
	}
	versions := &SPolicyVersions{}
	err := self.iamRequest("ListPolicyVersions", params, versions)
	if err != nil {
		return nil, errors.Wrapf(err, "ListPolicyVersions")
	}
	return versions, nil
}

func (self *SAwsClient) GetPolicyVersion(arn, versionId string) (*SPolicyVersion, error) {
	params := map[string]string{
		"PolicyArn": arn,
		"VersionId": versionId,
	}
	result := struct {
		Version SPolicyVersion `xml:"PolicyVersion"`
	}{}
	err := self.iamRequest("GetPolicyVersion", params, &result)
	if err != nil {
		return nil, errors.Wrapf(err, "GetPolicyVersion")
	}
	return &result.Version, nil
}

func (self *SAwsClient) GetPolicy(arn string) (*SPolicy, error) {
	params := map[string]string{
		"PolicyArn": arn,
	}
	result := struct {
		Policy SPolicy `xml:"Policy"`
	}{}
	err := self.iamRequest("GetPolicy", params, &result)
	if err != nil {
		return nil, errors.Wrapf(err, "GetPolicy")
	}
	return &result.Policy, nil
}

func (self *SAwsClient) CreateICloudpolicy(opts *cloudprovider.SCloudpolicyCreateOptions) (cloudprovider.ICloudpolicy, error) {
	policy, err := self.CreatePolicy(opts.Name, opts.Document.String(), "", opts.Desc)
	if err != nil {
		return nil, errors.Wrapf(err, "CreatePolicy")
	}
	return policy, nil
}

func (self *SAwsClient) CreatePolicy(name, document, path, desc string) (*SPolicy, error) {
	document = self.convertDocument(document)
	params := map[string]string{
		"PolicyName":     name,
		"PolicyDocument": document,
	}
	if len(path) > 0 {
		params["Path"] = path
	}
	if len(desc) > 0 {
		params["Description"] = desc
	}
	ret := struct {
		Policy SPolicy `xml:"Policy"`
	}{}
	err := self.iamRequest("CreatePolicy", params, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "CreatePolicy")
	}
	ret.Policy.client = self
	return &ret.Policy, nil
}

func (self *SAwsClient) DeletePolicy(arn string) error {
	params := map[string]string{
		"PolicyArn": arn,
	}
	err := self.iamRequest("DeletePolicy", params, nil)
	if err != nil && errors.Cause(err) != cloudprovider.ErrNotFound {
		return err
	}
	return nil
}

func (self *SAwsClient) convertDocument(document string) string {
	switch self.GetAccessEnv() {
	case api.CLOUD_ACCESS_ENV_AWS_GLOBAL:
		return strings.ReplaceAll(document, "arn:aws-cn", "arn:aws")
	default:
		return strings.ReplaceAll(document, "arn:aws", "arn:aws-cn")
	}
}

func (self *SAwsClient) CreatePolicyVersion(arn, document string, isDefault bool) error {
	document = self.convertDocument(document)
	params := map[string]string{
		"PolicyArn":      arn,
		"PolicyDocument": document,
	}
	if isDefault {
		params["SetAsDefault"] = "true"
	}
	return self.iamRequest("CreatePolicyVersion", params, nil)
}
