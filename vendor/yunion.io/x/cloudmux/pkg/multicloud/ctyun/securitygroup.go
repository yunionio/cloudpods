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

package ctyun

import (
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SSecurityGroup struct {
	multicloud.SSecurityGroup
	CtyunTags
	region *SRegion

	SecurityGroupName string
	Id                string
	VMNum             int
	Origin            string
	VpcName           string
	VpcId             string
	CreationTime      time.Time
	Description       string
	ProjectId         string

	SecurityGroupRuleList []SSecurityGroupRule
}

func (self *SSecurityGroup) Delete() error {
	return self.region.DeleteSecurityGroup(self.GetId())
}

func (self *SSecurityGroup) GetId() string {
	return self.Id
}

func (self *SSecurityGroup) GetName() string {
	return self.SecurityGroupName
}

func (self *SSecurityGroup) GetGlobalId() string {
	return self.GetId()
}

func (self *SSecurityGroup) GetStatus() string {
	return ""
}

func (self *SSecurityGroup) Refresh() error {
	sec, err := self.region.GetSecurityGroup(self.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, sec)
}

func (self *SSecurityGroup) GetDescription() string {
	return self.Description
}

func compatibleSecurityGroupRule(r SSecurityGroupRule) bool {
	// 忽略IPV6
	if r.Ethertype != "IPv4" {
		return false
	}
	if len(r.DestCidrIP) == 0 {
		return false
	}
	if !utils.IsInStringArray(r.Protocol, []string{"ANY", "TCP", "UDP", "ICMP"}) {
		return false
	}
	return true
}

func (self *SSecurityGroup) GetRules() ([]cloudprovider.SecurityRule, error) {
	rules := make([]cloudprovider.SecurityRule, 0)
	for _, r := range self.SecurityGroupRuleList {
		if !compatibleSecurityGroupRule(r) {
			continue
		}

		rule, err := r.toRule()
		if err != nil {
			return nil, err
		}

		rules = append(rules, rule)
	}

	return rules, nil
}

func (self *SSecurityGroup) GetVpcId() string {
	if len(self.VpcId) == 0 {
		return api.NORMAL_VPC_ID
	}
	return self.VpcId
}

func (self *SRegion) GetSecurityGroup(id string) (*SSecurityGroup, error) {
	params := map[string]interface{}{
		"securityGroupID": id,
	}
	resp, err := self.list(SERVICE_VPC, "/v4/vpc/describe-security-group-attribute", params)
	if err != nil {
		return nil, err
	}
	ret := &SSecurityGroup{region: self}
	return ret, resp.Unmarshal(ret, "returnObj")
}

func (self *SRegion) GetSecurityGroups(vpcId string) ([]SSecurityGroup, error) {
	pageNo := 1
	params := map[string]interface{}{
		"pageNo":   pageNo,
		"pageSize": 50,
	}
	if len(vpcId) > 0 {
		params["vpcID"] = vpcId
	}
	ret := []SSecurityGroup{}
	for {
		resp, err := self.list(SERVICE_VPC, "/v4/vpc/query-security-groups", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			ReturnObj  []SSecurityGroup
			TotalCount int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.ReturnObj...)
		if len(ret) >= part.TotalCount || len(part.ReturnObj) == 0 {
			break
		}
		pageNo++
		params["pageNo"] = pageNo
	}
	return ret, nil
}

func (self *SRegion) CreateSecurityGroup(opts *cloudprovider.SecurityGroupCreateInput) (*SSecurityGroup, error) {
	params := map[string]interface{}{
		"clientToken": utils.GenRequestId(20),
		"vpcID":       opts.VpcId,
		"name":        opts.Name,
		"description": opts.Desc,
	}
	resp, err := self.post(SERVICE_VPC, "/v4/vpc/create-security-group", params)
	if err != nil {
		return nil, err
	}
	id, err := resp.GetString("returnObj", "securityGroupID")
	if err != nil {
		return nil, errors.Wrapf(err, "get secgroup id")
	}
	for _, r := range opts.InRules {
		err := self.CreateSecurityGroupRule(id, r.SecurityRule)
		if err != nil {
			return nil, errors.Wrapf(err, "CreateRule")
		}
	}
	for _, r := range opts.OutRules {
		err := self.CreateSecurityGroupRule(id, r.SecurityRule)
		if err != nil {
			return nil, errors.Wrapf(err, "CreateRule")
		}
	}
	return self.GetSecurityGroup(id)
}

func (self *SRegion) DeleteSecurityGroup(id string) error {
	params := map[string]interface{}{
		"clientToken":     utils.GenRequestId(20),
		"securityGroupID": id,
	}
	_, err := self.post(SERVICE_VPC, "/v4/vpc/delete-security-group", params)
	return err
}
