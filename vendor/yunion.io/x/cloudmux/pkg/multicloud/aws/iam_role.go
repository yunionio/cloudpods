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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/apis/cloudid"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

var (
	samlRole = `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"sts:AssumeRoleWithSAML","Principal":{"Federated":"%s"},"Condition":{"StringEquals":{"SAML:aud":["%s"]}}}]}`
	k8sRole  = `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":"eks.amazonaws.com"},"Action":"sts:AssumeRole"}]}`
	nodeRole = `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":"ec2.amazonaws.com.cn"},"Action":"sts:AssumeRole"}]}`
)

type SRole struct {
	client *SAwsClient

	Path                     string    `xml:"Path"`
	AssumeRolePolicyDocument string    `xml:"AssumeRolePolicyDocument"`
	MaxSessionDuration       int       `xml:"MaxSessionDuration"`
	RoleId                   string    `xml:"RoleId"`
	RoleName                 string    `xml:"RoleName"`
	Description              string    `xml:"Description"`
	Arn                      string    `xml:"Arn"`
	CreateDate               time.Time `xml:"CreateDate"`
}

func (self *SRole) GetGlobalId() string {
	return self.Arn
}

func (self *SRole) GetName() string {
	return self.RoleName
}

func (self *SRole) Delete() error {
	return self.client.DeleteRole(self.RoleName)
}

func (self *SRole) GetDocument() *jsonutils.JSONDict {
	data, err := url.QueryUnescape(self.AssumeRolePolicyDocument)
	if err != nil {
		return nil
	}
	document, err := jsonutils.Parse([]byte(data))
	if err != nil {
		return nil
	}
	return document.(*jsonutils.JSONDict)
}

// [{"Action":"sts:AssumeRoleWithSAML","Condition":{"StringEquals":{"SAML:aud":"https://signin.aws.amazon.com/saml"}},"Effect":"Allow","Principal":{"Federated":"arn:aws:iam::879324515906:saml-provider/quxuan"}}]
func (self *SRole) GetSAMLProvider() string {
	document := self.GetDocument()
	if document != nil {
		statement, err := document.GetArray("Statement")
		if err == nil {
			for i := range statement {
				if action, _ := statement[i].GetString("Action"); action == "sts:AssumeRoleWithSAML" {
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

func (self *SRole) AttachPolicy(id string, policyType cloudid.TPolicyType) error {
	return self.client.AttachRolePolicy(self.RoleName, id)
}

func (self *SRole) DetachPolicy(id string, polityType cloudid.TPolicyType) error {
	return self.client.DetachRolePolicy(self.RoleName, id)
}

func (self *SRole) GetICloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	policies := []SAttachedPolicy{}
	marker := ""
	for {
		part, err := self.client.ListAttachedRolePolicies(self.RoleName, marker, 100, "")
		if err != nil {
			return nil, errors.Wrapf(err, "ListAttachedRolePolicies")
		}
		policies = append(policies, part.AttachedPolicies...)
		marker = part.Marker
		if len(marker) == 0 {
			break
		}
	}
	ret := []cloudprovider.ICloudpolicy{}
	for i := range policies {
		policies[i].client = self.client
		ret = append(ret, &policies[i])
	}
	return ret, nil
}

type SRoles struct {
	Roles       []SRole `xml:"Roles>member"`
	IsTruncated bool    `xml:"IsTruncated"`
	Marker      string  `xml:"Marker"`
}

func (self *SAwsClient) ListRoles(offset string, limit int, prefix string) (*SRoles, error) {
	if limit < 1 || limit > 1000 {
		limit = 1000
	}
	params := map[string]string{
		"MaxItems": fmt.Sprintf("%d", limit),
	}
	if len(offset) > 0 {
		params["Marker"] = offset
	}
	if len(prefix) > 0 {
		params["PathPrefix"] = prefix
	}
	roles := &SRoles{}
	err := self.iamRequest("ListRoles", params, roles)
	if err != nil {
		return nil, errors.Wrapf(err, "ListRoles")
	}
	return roles, nil
}

func (self *SAwsClient) GetRole(roleName string) (*SRole, error) {
	params := map[string]string{
		"RoleName": roleName,
	}
	result := struct {
		Role SRole `xml:"Role"`
	}{}
	err := self.iamRequest("GetRole", params, &result)
	if err != nil {
		return nil, errors.Wrap(err, "GetRole")
	}
	return &result.Role, nil
}

func (self *SAwsClient) DeleteRole(name string) error {
	params := map[string]string{
		"RoleName": name,
	}
	return self.iamRequest("DeleteRole", params, nil)
}

func (self *SAwsClient) GetICloudroles() ([]cloudprovider.ICloudrole, error) {
	roles := []SRole{}
	marker := ""
	for {
		part, err := self.ListRoles(marker, 100, "")
		if err != nil {
			return nil, errors.Wrapf(err, "ListRoles")
		}
		roles = append(roles, part.Roles...)
		marker = part.Marker
		if len(marker) == 0 {
			break
		}
	}
	ret := []cloudprovider.ICloudrole{}
	for i := range roles {
		roles[i].client = self
		ret = append(ret, &roles[i])
	}
	return ret, nil
}

func (self *SAwsClient) CreateRole(opts *cloudprovider.SRoleCreateOptions) (*SRole, error) {
	if len(opts.SAMLProvider) > 0 {
		aud := "https://signin.amazonaws.cn/saml"
		if self.GetAccessEnv() == api.CLOUD_ACCESS_ENV_AWS_GLOBAL {
			aud = "https://signin.aws.amazon.com/saml"
		}
		params := map[string]string{
			"RoleName":                 opts.Name,
			"Description":              opts.Desc,
			"AssumeRolePolicyDocument": fmt.Sprintf(samlRole, opts.SAMLProvider, aud),
		}
		role := struct {
			Role SRole
		}{}
		err := self.iamRequest("CreateRole", params, &role)
		if err != nil {
			return nil, errors.Wrapf(err, "CreateRole")
		}
		role.Role.client = self
		return &role.Role, nil
	}
	return nil, cloudprovider.ErrNotImplemented
}
