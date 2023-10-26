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
	//VpcName           string
	//VpcId             string
	CreationTime time.Time
	Description  string
	ProjectId    string

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
	return api.SECGROUP_STATUS_READY
}

func (self *SSecurityGroup) Refresh() error {
	sec, err := self.region.GetSecurityGroup(self.GetId())
	if err != nil {
		return err
	}
	self.SecurityGroupRuleList = nil
	return jsonutils.Update(self, sec)
}

func (self *SSecurityGroup) GetDescription() string {
	return self.Description
}

func (self *SSecurityGroup) GetTags() (map[string]string, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotSupported, "GetTags")
}

func (self *SSecurityGroup) GetRules() ([]cloudprovider.ISecurityGroupRule, error) {
	rules := make([]cloudprovider.ISecurityGroupRule, 0)
	for i := range self.SecurityGroupRuleList {
		self.SecurityGroupRuleList[i].secgroup = self
		rules = append(rules, &self.SecurityGroupRuleList[i])
	}
	return rules, nil
}

func (self *SSecurityGroup) GetVpcId() string {
	return ""
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

func (self *SRegion) GetSecurityGroups() ([]SSecurityGroup, error) {
	pageNo := 1
	params := map[string]interface{}{
		"pageNo":   pageNo,
		"pageSize": 50,
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

func (self *SSecurityGroup) CreateRule(opts *cloudprovider.SecurityGroupRuleCreateOptions) (cloudprovider.ISecurityGroupRule, error) {
	ruleIds := []string{}
	for _, rule := range self.SecurityGroupRuleList {
		if !utils.IsInStringArray(rule.Id, ruleIds) {
			ruleIds = append(ruleIds, rule.Id)
		}
	}
	err := self.region.CreateSecurityGroupRule(self.Id, opts)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateSecurityGroupRule")
	}
	for i := 0; i < 3; i++ {
		err := self.Refresh()
		if err != nil {
			return nil, errors.Wrapf(err, "Refresh")
		}
		for i := range self.SecurityGroupRuleList {
			if !utils.IsInStringArray(self.SecurityGroupRuleList[i].Id, ruleIds) {
				self.SecurityGroupRuleList[i].secgroup = self
				return &self.SecurityGroupRuleList[i], nil
			}
		}
		time.Sleep(time.Second * 3)
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "after created")
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
