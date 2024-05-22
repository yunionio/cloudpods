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
	"time"
)

type SOrganizationMember struct {
	AppId       string
	AuthName    string
	AuthStatus  string
	BindStatus  string
	CreateTime  time.Time
	IsAllowQuit string
	MemberType  string
	MemberUin   int
	Name        string
	NodeId      int
	NodeName    string
	OrgIdentity []struct {
		IdentityAliasName string
		IdentityId        int
	}
	OrgPermission []struct {
		Id   string
		Name string
	}
	OrgPolicyName    string
	OrgPolicyType    string
	PayName          string
	PayUin           string
	PermissionStatus string
	Remark           string
}

func (self *SQcloudClient) DescribeOrganizationMembers() ([]SOrganizationMember, error) {
	params := map[string]string{
		"Limit":  "50",
		"Offset": "0",
	}
	ret := []SOrganizationMember{}
	for {
		resp, err := self.orgRequest("DescribeOrganizationMembers", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			Items []SOrganizationMember
			Total int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.Items...)
		if len(ret) >= part.Total || len(part.Items) == 0 {
			break
		}
		params["Offset"] = fmt.Sprintf("%d", len(ret))
	}
	return ret, nil
}

func (self *SQcloudClient) DescribeOrganizationMember(uin string) (*SOrganizationMember, error) {
	params := map[string]string{
		"MemberUin": uin,
	}
	resp, err := self.orgRequest("DescribeOrganizationMember", params)
	if err != nil {
		return nil, err
	}
	ret := &SOrganizationMember{}
	err = resp.Unmarshal(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
