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

package qcloud

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/cloudid"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

const (
	DEFAULT_ROLE_DOCUMENT = `{"version":"2.0","statement":[{"action":"name/sts:AssumeRole","effect":"allow","principal":{"service":["cvm.qcloud.com"]}}]}`
)

/*
"AddTime": "2020-08-11 17:03:30",
"ConsoleLogin": 0.000000,
"Description": "hello",
"PolicyDocument": "{\"version\":\"2.0\",\"statement\":[{\"action\":\"name/sts:AssumeRole\",\"effect\":\"allow\",\"principal\":{\"service\":[\"cdb.qcloud.com\",\"blueking.cloud.tencent.com\"]}}]}",
"RoleId": "4611686018428392276",
"RoleName": "test-role",
"RoleType": "user",
"SessionDuration": 0.000000,
"UpdateTime": "2020-08-11 17:03:30"
*/

type SPrincipal struct {
	Federated []string
}

type Statement struct {
	Action    string
	Effect    string
	Principal SPrincipal
}

type SRole struct {
	multicloud.SResourceBase
	client *SQcloudClient

	AddTime         time.Time
	ConsoleLogin    float32
	Description     string
	PolicyDocument  string
	RoleId          string
	RoleName        string
	RoleType        string
	SessionDuration float32
	UpdateTime      time.Time
}

func (self *SRole) GetGlobalId() string {
	return self.RoleName
}

func (self *SRole) GetName() string {
	return self.RoleName
}

func (self *SRole) GetDocument() *jsonutils.JSONDict {
	if len(self.PolicyDocument) > 0 {
		document, err := jsonutils.Parse([]byte(self.PolicyDocument))
		if err != nil {
			return nil
		}
		return document.(*jsonutils.JSONDict)
	}
	return nil
}

func (self *SRole) GetSAMLProvider() string {
	document := self.GetDocument()
	if document != nil {
		statements := []Statement{}
		document.Unmarshal(&statements, "statement")
		for i := range statements {
			if statements[i].Action == "name/sts:AssumeRoleWithSAML" {
				for _, federated := range statements[i].Principal.Federated {
					if strings.Contains(federated, ":saml-provider/") {
						info := strings.Split(federated, "/")
						return info[len(info)-1]
					}
				}
			}
		}
	}
	return ""
}

func (self *SRole) Delete() error {
	return self.client.DeleteRole(self.RoleName)
}

func (self *SRole) GetICloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	ret := []cloudprovider.ICloudpolicy{}
	for {
		part, total, err := self.client.ListAttachedRolePolicies(self.RoleName, "", len(ret), 50)
		if err != nil {
			return nil, errors.Wrapf(err, "ListAttachedRolePolicies")
		}
		for i := range part {
			part[i].client = self.client
			ret = append(ret, &part[i])
		}
		if len(ret) >= total {
			break
		}
	}
	return ret, nil
}

func (self *SRole) AttachPolicy(id string, policyType api.TPolicyType) error {
	return self.client.AttachRolePolicy(self.RoleName, id)
}

func (self *SRole) DetachPolicy(id string, policyType api.TPolicyType) error {
	return self.client.DetachRolePolicy(self.RoleName, id)
}

func (self *SQcloudClient) GetICloudroles() ([]cloudprovider.ICloudrole, error) {
	ret := []cloudprovider.ICloudrole{}
	for {
		part, total, err := self.DescribeRoleList(len(ret), 200)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeRoleList")
		}
		for i := range part {
			part[i].client = self
			ret = append(ret, &part[i])
		}
		if len(ret) >= total {
			break
		}
	}
	return ret, nil
}

func (self *SQcloudClient) DescribeRoleList(offset int, limit int) ([]SRole, int, error) {
	if limit < 1 || limit > 200 {
		limit = 200
	}
	if offset < 1 {
		offset = 1
	}
	params := map[string]string{
		"Page": fmt.Sprintf("%d", offset),
		"Rp":   fmt.Sprintf("%d", limit),
	}
	roles := []SRole{}
	resp, err := self.camRequest("DescribeRoleList", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeRoleList")
	}
	err = resp.Unmarshal(&roles, "List")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "resp.Unmarshal")
	}
	total, _ := resp.Float("TotalNum")
	return roles, int(total), nil
}

func (self *SQcloudClient) CreateRole(name, document, desc string) (*SRole, error) {
	if len(document) == 0 {
		document = DEFAULT_ROLE_DOCUMENT
	}
	params := map[string]string{
		"RoleName":        name,
		"PolicyDocument":  document,
		"ConsoleLogin":    "1",
		"SessionDuration": "43200",
		"Description":     desc,
	}
	_, err := self.camRequest("CreateRole", params)
	if err != nil {
		return nil, errors.Wrap(err, "CreateRole")
	}
	return self.GetRole(name)
}

func (self *SQcloudClient) GetRole(name string) (*SRole, error) {
	params := map[string]string{
		"RoleName": name,
	}
	resp, err := self.camRequest("GetRole", params)
	if err != nil {
		return nil, errors.Wrapf(err, "GetRole(%s)", name)
	}
	role := &SRole{client: self}
	err = resp.Unmarshal(role, "RoleInfo")
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return role, nil
}

func (self *SQcloudClient) ListAttachedRolePolicies(roleName, policyType string, offset, limit int) ([]SPolicy, int, error) {
	if limit < 1 || limit > 200 {
		limit = 200
	}
	if offset < 1 {
		offset = 1
	}
	params := map[string]string{
		"Page":     fmt.Sprintf("%d", offset),
		"Rp":       fmt.Sprintf("%d", limit),
		"RoleName": roleName,
	}
	if len(policyType) > 0 {
		params["PolicyType"] = policyType
	}
	resp, err := self.camRequest("ListAttachedRolePolicies", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "ListAttachedRolePolicies")
	}
	policies := []SPolicy{}
	err = resp.Unmarshal(&policies, "List")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "resp.Unmarshal")
	}
	total, _ := resp.Float("TotalNum")
	return policies, int(total), nil
}

func (self *SQcloudClient) DeleteRole(name string) error {
	params := map[string]string{
		"RoleName": name,
	}
	_, err := self.camRequest("DeleteRole", params)
	return err
}

func (self *SQcloudClient) AttachRolePolicy(roleName string, policyId string) error {
	params := map[string]string{
		"AttachRoleName": roleName,
	}
	if _id, _ := strconv.Atoi(policyId); _id > 0 {
		params["PolicyId"] = policyId
	} else {
		params["PolicyName"] = policyId
	}
	_, err := self.camRequest("AttachRolePolicy", params)
	return err
}

func (self *SQcloudClient) DetachRolePolicy(roleName string, policyId string) error {
	params := map[string]string{
		"DetachRoleName": roleName,
	}
	if _id, _ := strconv.Atoi(policyId); _id > 0 {
		params["PolicyId"] = policyId
	} else {
		params["PolicyName"] = policyId
	}
	_, err := self.camRequest("DetachRolePolicy", params)
	return err
}
