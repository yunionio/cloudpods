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
	"time"

	"yunion.io/x/pkg/errors"
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
