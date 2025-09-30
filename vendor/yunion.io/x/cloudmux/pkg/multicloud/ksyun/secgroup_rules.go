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
	"fmt"
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"
)

type SPermission struct {
	region *SRegion

	SecurityGroupId      string `json:"SecurityGroupId"`
	Policy               string `json:"Policy"`
	PortRangeTo          int    `json:"PortRangeTo"`
	Description          string `json:"Description"`
	IcmpCode             int    `json:"IcmpCode"`
	IcmpType             int    `json:"IcmpType"`
	Priority             int    `json:"Priority"`
	CreateTime           string `json:"CreateTime"`
	CidrBlock            string `json:"CidrBlock"`
	Direction            string `json:"Direction"`
	PortRangeFrom        int    `json:"PortRangeFrom"`
	SecurityGroupEntryId string `json:"SecurityGroupEntryId"`
	RuleTag              string `json:"RuleTag"`
	Protocol             string `json:"Protocol"`
}

func (rule *SPermission) GetGlobalId() string {
	return rule.SecurityGroupEntryId
}

func (rule *SPermission) GetDirection() secrules.TSecurityRuleDirection {
	return secrules.TSecurityRuleDirection(rule.Direction)
}

func (rule *SPermission) GetPriority() int {
	return rule.Priority
}

func (rule *SPermission) GetAction() secrules.TSecurityRuleAction {
	if rule.Policy == "Drop" {
		return secrules.SecurityRuleDeny
	}
	return secrules.SecurityRuleAllow
}

func (rule *SPermission) GetProtocol() string {
	if rule.Protocol == "ip" {
		return secrules.PROTO_ANY
	}
	return rule.Protocol
}

func (rule *SPermission) GetPorts() string {
	if rule.PortRangeFrom > 0 && rule.PortRangeTo > 0 {
		if rule.PortRangeFrom == rule.PortRangeTo {
			return fmt.Sprintf("%d", rule.PortRangeFrom)
		}
		return fmt.Sprintf("%d-%d", rule.PortRangeFrom, rule.PortRangeTo)
	}
	return ""
}

func (rule *SPermission) GetDescription() string {
	return rule.Description
}

func (rule *SPermission) GetCIDRs() []string {
	return []string{rule.CidrBlock}
}

func (rule *SPermission) Update(opts *cloudprovider.SecurityGroupRuleUpdateOptions) error {
	return errors.ErrNotSupported
}

func (rule *SPermission) Delete() error {
	return rule.region.DeleteSecurityGroupRule(rule.SecurityGroupId, rule.SecurityGroupEntryId)
}

func (region *SRegion) DeleteSecurityGroupRule(groupId string, ruleId string) error {
	params := map[string]interface{}{
		"SecurityGroupId":      groupId,
		"SecurityGroupEntryId": ruleId,
	}
	_, err := region.ecsRequest("RevokeSecurityGroupEntry", params)
	return err
}

func (group *SSecurityGroup) CreateRule(opts *cloudprovider.SecurityGroupRuleCreateOptions) (cloudprovider.ISecurityGroupRule, error) {
	rule, err := group.region.CreateSecurityGroupRule(group.SecurityGroupId, opts)
	if err != nil {
		return nil, err
	}
	return rule, nil
}

func (region *SRegion) CreateSecurityGroupRule(groupId string, opts *cloudprovider.SecurityGroupRuleCreateOptions) (*SPermission, error) {
	params := map[string]interface{}{
		"SecurityGroupId": groupId,
		"CidrBlock":       opts.CIDR,
		"Direction":       string(opts.Direction),
		"Protocol":        opts.Protocol,
		"Priority":        fmt.Sprintf("%d", opts.Priority),
		"Policy":          "Accept",
	}
	if opts.Action == secrules.SecurityRuleDeny {
		params["Policy"] = "Drop"
	}
	if len(opts.Desc) > 0 {
		params["Description"] = opts.Desc
	}
	switch opts.Protocol {
	case secrules.PROTO_ANY:
		params["Protocol"] = "ip"
	case secrules.PROTO_TCP, secrules.PROTO_UDP:
		if len(opts.Ports) == 0 {
			params["PortRangeFrom"] = "1"
			params["PortRangeTo"] = "65535"
		} else {
			if strings.Contains(opts.Ports, "-") {
				info := strings.Split(opts.Ports, "-")
				if len(info) == 2 {
					params["PortRangeFrom"] = info[0]
					params["PortRangeTo"] = info[1]
				}
			} else {
				params["PortRangeFrom"] = opts.Ports
				params["PortRangeTo"] = opts.Ports
			}
		}
	case secrules.PROTO_ICMP:
		params["IcmpType"] = "-1"
		params["IcmpCode"] = "-1"
	}
	resp, err := region.vpcRequest("AuthorizeSecurityGroupEntry", params)
	if err != nil {
		return nil, err
	}
	ret := []string{}
	err = resp.Unmarshal(&ret, "SecurityGroupEntryIdSet")
	if err != nil {
		return nil, err
	}
	ruleId := ""
	for i := range ret {
		ruleId = ret[i]
	}
	if len(ruleId) == 0 {
		return nil, fmt.Errorf("invalid rule create response %s", resp.String())
	}
	group, err := region.GetSecurityGroup(groupId)
	if err != nil {
		return nil, err
	}
	for i := range group.SecurityGroupEntrySet {
		if group.SecurityGroupEntrySet[i].SecurityGroupEntryId == ruleId {
			group.SecurityGroupEntrySet[i].region = region
			group.SecurityGroupEntrySet[i].SecurityGroupId = groupId
			return &group.SecurityGroupEntrySet[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "after create %s", jsonutils.Marshal(opts))
}
