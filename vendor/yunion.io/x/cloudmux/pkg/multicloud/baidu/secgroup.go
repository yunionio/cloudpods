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

package baidu

import (
	"fmt"
	"net/url"
	"time"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/utils"
)

type SSecurityGroup struct {
	multicloud.SSecurityGroup
	SBaiduTag
	region *SRegion

	Id          string
	Name        string
	VpcId       string
	Desc        string
	CreatedTime time.Time
	UpdatedTime time.Time
	SgVersion   string
	Rules       []SSecurityGroupRule
}

func (self *SSecurityGroup) GetVpcId() string {
	return self.VpcId
}

func (self *SSecurityGroup) GetId() string {
	return self.Id
}

func (self *SSecurityGroup) GetGlobalId() string {
	return self.Id
}

func (self *SSecurityGroup) GetDescription() string {
	return self.Desc
}

func (self *SSecurityGroup) GetName() string {
	if len(self.Name) > 0 {
		return self.Name
	}
	return self.Name
}

func (self *SSecurityGroup) GetStatus() string {
	return api.SECGROUP_STATUS_READY
}

func (self *SSecurityGroup) Refresh() error {
	group, err := self.region.GetSecurityGroup(self.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, group)
}

func (self *SSecurityGroup) Delete() error {
	return self.region.DeleteSecurityGroup(self.Id)
}

func (self *SSecurityGroup) GetRules() ([]cloudprovider.ISecurityGroupRule, error) {
	ret := make([]cloudprovider.ISecurityGroupRule, 0)
	for i := range self.Rules {
		self.Rules[i].region = self.region
		ret = append(ret, &self.Rules[i])
	}
	return ret, nil
}

func (self *SSecurityGroup) CreateRule(opts *cloudprovider.SecurityGroupRuleCreateOptions) (cloudprovider.ISecurityGroupRule, error) {
	ruleIds := []string{}
	for _, rule := range self.Rules {
		ruleIds = append(ruleIds, rule.GetGlobalId())
	}
	err := self.region.CreateSecurityGroupRule(self.Id, opts)
	if err != nil {
		return nil, err
	}
	err = self.Refresh()
	if err != nil {
		return nil, err
	}
	for i := range self.Rules {
		if !utils.IsInStringArray(self.Rules[i].GetGlobalId(), ruleIds) {
			self.Rules[i].region = self.region
			return &self.Rules[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "After created")
}

func (region *SRegion) CreateSecurityGroupRule(groupId string, opts *cloudprovider.SecurityGroupRuleCreateOptions) error {
	params := url.Values{}
	params.Set("clientToken", utils.GenRequestId(20))
	params.Set("authorizeRule", "")
	rule := map[string]interface{}{
		"remark":    opts.Desc,
		"protocol":  opts.Protocol,
		"portRange": opts.Ports,
	}
	switch opts.Direction {
	case secrules.DIR_OUT:
		rule["direction"] = "egress"
		rule["destIp"] = opts.CIDR
	case secrules.DIR_IN:
		rule["direction"] = "ingress"
		rule["sourceIp"] = opts.CIDR
	}
	if opts.Protocol == secrules.PROTO_ANY {
		rule["protocol"] = "all"
	}
	body := map[string]interface{}{
		"rule": rule,
	}
	_, err := region.bccUpdate(fmt.Sprintf("v2/securityGroup/%s", groupId), params, body)
	return err
}

func (region *SRegion) GetSecurityGroups(vpcId string) ([]SSecurityGroup, error) {
	params := url.Values{}
	if len(vpcId) > 0 {
		params.Set("vpcId", vpcId)
	}
	ret := []SSecurityGroup{}
	for {
		resp, err := region.bccList("v2/securityGroup", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			NextMarker     string
			SecurityGroups []SSecurityGroup
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.SecurityGroups...)
		if len(part.NextMarker) == 0 {
			break
		}
		params.Set("marker", part.NextMarker)
	}
	return ret, nil
}

func (region *SRegion) DeleteSecurityGroup(id string) error {
	_, err := region.bccDelete(fmt.Sprintf("v2/securityGroup/%s", id), nil)
	return err
}

func (region *SRegion) GetSecurityGroup(id string) (*SSecurityGroup, error) {
	resp, err := region.bccList(fmt.Sprintf("v2/securityGroup/%s", id), nil)
	if err != nil {
		return nil, err
	}
	ret := &SSecurityGroup{region: region}
	err = resp.Unmarshal(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (region *SRegion) CreateSecurityGroup(opts *cloudprovider.SecurityGroupCreateInput) (*SSecurityGroup, error) {
	params := url.Values{}
	params.Set("clientToken", utils.GenRequestId(20))
	tags := []BaiduTag{}
	for k, v := range opts.Tags {
		tags = append(tags, BaiduTag{
			TagKey:   k,
			TagValue: v,
		})
	}
	body := map[string]interface{}{
		"name":  opts.Name,
		"vpcId": opts.VpcId,
		"desc":  opts.Desc,
		"tags":  tags,
	}
	resp, err := region.bccPost("v2/securityGroup", params, body)
	if err != nil {
		return nil, err
	}
	groupId, err := resp.GetString("securityGroupId")
	if err != nil {
		return nil, err
	}
	return region.GetSecurityGroup(groupId)
}
