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
	"strings"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/utils"
)

type SSecurityGroupRule struct {
	secgroup *SSecurityGroup

	Direction       string
	Priority        int
	Ethertype       string
	Protocol        string
	DestCidrIP      string
	Description     string
	Origin          string
	CreateTime      time.Time
	Id              string
	Action          string
	SecurityGroupId string
	Range           string
}

func (self *SSecurityGroupRule) GetGlobalId() string {
	return self.Id
}

func (self *SSecurityGroupRule) GetDirection() secrules.TSecurityRuleDirection {
	if self.Direction == "egress" {
		return secrules.DIR_OUT
	}
	return secrules.DIR_IN
}

func (self *SSecurityGroupRule) GetPriority() int {
	return self.Priority
}

func (self *SSecurityGroupRule) GetAction() secrules.TSecurityRuleAction {
	if self.Action == "accept" {
		return secrules.SecurityRuleAllow
	}
	return secrules.SecurityRuleDeny
}

func (self *SSecurityGroupRule) GetProtocol() string {
	return strings.ToLower(self.Protocol)
}

func (self *SSecurityGroupRule) GetPorts() string {
	if strings.ToLower(self.Range) == "any" {
		return ""
	}
	return self.Range
}

func (self *SSecurityGroupRule) GetDescription() string {
	return self.Description
}

func (self *SSecurityGroupRule) GetCIDRs() []string {
	return []string{self.DestCidrIP}
}

func (self *SSecurityGroupRule) Delete() error {
	err := self.secgroup.region.DeleteSecgroupRule(self.SecurityGroupId, self.Id, self.GetDirection())
	if err != nil {
		return errors.Wrapf(err, "Delete")
	}
	// wait rule deleted
	cloudprovider.Wait(time.Second*5, time.Minute, func() (bool, error) {
		secgroup, err := self.secgroup.region.GetSecurityGroup(self.secgroup.Id)
		if err != nil {
			return false, nil
		}
		for _, rule := range secgroup.SecurityGroupRuleList {
			if rule.Id == self.Id {
				return false, nil
			}
		}
		return true, nil
	})
	return nil
}

func (self *SRegion) CreateSecurityGroupRule(groupId string, opts *cloudprovider.SecurityGroupRuleCreateOptions) error {
	rule := map[string]interface{}{
		"direction":   "egress",
		"action":      "accept",
		"priority":    opts.Priority,
		"protocol":    strings.ToUpper(opts.Protocol),
		"ethertype":   "IPv4",
		"destCidrIp":  opts.CIDR,
		"description": opts.Desc,
		"range":       "1-65535",
	}
	if _, err := netutils.NewIPV6Prefix(opts.CIDR); err == nil {
		rule["ethertype"] = "IPv6"
	}
	api := "/v4/vpc/create-security-group-egress"
	if opts.Direction == secrules.DIR_IN {
		rule["direction"] = "ingress"
		api = "/v4/vpc/create-security-group-ingress"
	}
	if len(opts.Ports) > 0 {
		rule["range"] = opts.Ports
	}
	if opts.Action == secrules.SecurityRuleDeny {
		rule["action"] = "drop"
	}
	params := map[string]interface{}{
		"securityGroupID":    groupId,
		"clientToken":        utils.GenRequestId(20),
		"securityGroupRules": []map[string]interface{}{rule},
	}
	_, err := self.post(SERVICE_VPC, api, params)
	return err
}

func (self *SRegion) DeleteSecgroupRule(groupId, ruleId string, direction secrules.TSecurityRuleDirection) error {
	params := map[string]interface{}{
		"securityGroupID":     groupId,
		"securityGroupRuleID": ruleId,
		"clientToken":         utils.GenRequestId(20),
	}
	api := "/v4/vpc/revoke-security-group-egress"
	if direction == secrules.DIR_IN {
		api = "/v4/vpc/revoke-security-group-ingress"
	}
	_, err := self.post(SERVICE_VPC, api, params)
	return err
}

func (self *SSecurityGroupRule) Update(opts *cloudprovider.SecurityGroupRuleUpdateOptions) error {
	return self.secgroup.region.UpdateSecurityGroupRule(self.secgroup.Id, self.Id, self.GetDirection(), opts)
}

func (self *SRegion) UpdateSecurityGroupRule(groupId, id string, direction secrules.TSecurityRuleDirection, opts *cloudprovider.SecurityGroupRuleUpdateOptions) error {
	api := "/v4/vpc/modify-security-group-egress"
	if direction == secrules.DIR_IN {
		api = "/v4/vpc/modify-security-group-ingress"
	}
	params := map[string]interface{}{
		"securityGroupID":     groupId,
		"securityGroupRuleID": id,
		"description":         opts.Desc,
		"clientToken":         utils.GenRequestId(20),
	}
	_, err := self.post(SERVICE_VPC, api, params)
	return err
}
