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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"
)

type SPermission struct {
	region *SRegion

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
	SecurityGroupEntryID string `json:"SecurityGroupEntryId"`
	RuleTag              string `json:"RuleTag"`
	Protocol             string `json:"Protocol"`
}

func (rule *SPermission) GetGlobalId() string {
	return rule.SecurityGroupEntryID
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
	return errors.ErrNotImplemented
}

func (rule *SPermission) Delete() error {
	return errors.ErrNotImplemented
}
