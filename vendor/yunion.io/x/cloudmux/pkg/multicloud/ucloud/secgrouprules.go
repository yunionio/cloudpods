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

package ucloud

import (
	"fmt"
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/util/secrules"
)

type SecurityGroupRule struct {
	secgroup *SSecurityGroup

	DstPort      string `json:"DstPort"`
	Priority     string `json:"Priority"`
	ProtocolType string `json:"ProtocolType"`
	RuleAction   string `json:"RuleAction"`
	SrcIP        string `json:"SrcIP"`
	Remark       string
}

func (self SecurityGroupRule) String() string {
	return fmt.Sprintf("%s|%s|%s|%s|%s|%s", self.ProtocolType, self.DstPort, self.SrcIP, self.RuleAction, self.Priority, self.Remark)
}

func (self *SecurityGroupRule) GetGlobalId() string {
	return fmt.Sprintf("%s|%s|%s|%s|%s", self.Priority, self.RuleAction, self.SrcIP, self.DstPort, self.ProtocolType)
}

func (self *SecurityGroupRule) GetAction() secrules.TSecurityRuleAction {
	if self.RuleAction == "ACCEPT" {
		return secrules.SecurityRuleAllow
	}
	return secrules.SecurityRuleDeny
}

func (self *SecurityGroupRule) GetDescription() string {
	return self.Remark
}

func (self *SecurityGroupRule) GetDirection() secrules.TSecurityRuleDirection {
	return secrules.DIR_IN
}

func (self *SecurityGroupRule) GetCIDRs() []string {
	return []string{self.SrcIP}
}

func (self *SecurityGroupRule) GetProtocol() string {
	return strings.ToLower(self.ProtocolType)
}

func (self *SecurityGroupRule) GetPorts() string {
	return self.DstPort
}

func (self *SecurityGroupRule) GetPriority() int {
	switch self.Priority {
	case "HIGH":
		return 1
	case "MEDIUM":
		return 2
	case "LOW":
		return 3
	}
	return 1
}

func (self *SecurityGroupRule) Delete() error {
	params := NewUcloudParams()
	params.Set("FWId", self.secgroup.FWID)
	idx := 0
	for _, rule := range self.secgroup.Rule {
		if rule.GetGlobalId() == self.GetGlobalId() {
			continue
		}
		params.Set(fmt.Sprintf("Rule.%d", idx), rule.String())
		idx++
	}
	return self.secgroup.region.DoAction("UpdateFirewall", params, nil)
}

func (self *SecurityGroupRule) Update(opts *cloudprovider.SecurityGroupRuleUpdateOptions) error {
	return cloudprovider.ErrNotImplemented
}
