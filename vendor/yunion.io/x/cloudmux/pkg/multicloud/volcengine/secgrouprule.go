// Copyright 2023 Yunion
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

package volcengine

import (
	"fmt"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/util/secrules"
)

type SCidrList []string

type SSecurityGroupRule struct {
	secgroup *SSecurityGroup

	CreationTime    time.Time
	UpdateTime      time.Time
	Description     string
	Direction       string
	Protocol        string
	Policy          string
	PortStart       int
	PortEnd         int
	CidrIp          string
	PrefixListId    string
	PrefixListCidrs SCidrList
	Priority        int
	SourceGroupId   string
}

func (self *SSecurityGroupRule) GetGlobalId() string {
	return fmt.Sprintf("%d|%s|%s|%s|%s|%s|%s|%d|%d", self.Priority, self.Direction, self.Policy, self.Protocol, self.CidrIp, self.PrefixListId, self.SourceGroupId, self.PortStart, self.PortEnd)
}

func (self *SSecurityGroupRule) GetAction() secrules.TSecurityRuleAction {
	if self.Policy == "accept" {
		return secrules.SecurityRuleAllow
	}
	return secrules.SecurityRuleDeny
}

func (self *SSecurityGroupRule) GetDescription() string {
	return self.Description
}

func (self *SSecurityGroupRule) GetDirection() secrules.TSecurityRuleDirection {
	if self.Direction == "egress" {
		return secrules.DIR_OUT
	}
	return secrules.DIR_IN
}

func (self *SSecurityGroupRule) GetCIDRs() []string {
	ip := self.CidrIp + self.PrefixListId + self.SourceGroupId
	ret := []string{ip}
	if len(self.PrefixListCidrs) > 0 {
		ret = append(ret, self.PrefixListCidrs...)
	}
	return ret
}

func (self *SSecurityGroupRule) GetProtocol() string {
	if len(self.Protocol) == 0 || self.Protocol == "all" {
		return secrules.PROTO_ANY
	}
	return self.Protocol
}

func (self *SSecurityGroupRule) GetPorts() string {
	if self.PortStart > 0 && self.PortEnd > 0 {
		if self.PortStart == self.PortEnd {
			return fmt.Sprintf("%d", self.PortStart)
		}
		return fmt.Sprintf("%d-%d", self.PortStart, self.PortEnd)
	}
	return ""
}

func (self *SSecurityGroupRule) GetPriority() int {
	return self.Priority
}

func (self *SSecurityGroupRule) Delete() error {
	params := map[string]string{
		"SecurityGroupId": self.secgroup.SecurityGroupId,
		"Protocol":        self.Protocol,
		"PortStart":       fmt.Sprintf("%d", self.PortStart),
		"PortEnd":         fmt.Sprintf("%d", self.PortEnd),
		"Policy":          self.Policy,
		"Priority":        fmt.Sprintf("%d", self.Priority),
	}
	if len(self.CidrIp) > 0 {
		params["CidrIp"] = self.CidrIp
	}
	if len(self.PrefixListId) > 0 {
		params["PrefixListId"] = self.PrefixListId
	}
	if len(self.SourceGroupId) > 0 {
		params["SourceGroupId"] = self.SourceGroupId
	}
	action := "RevokeSecurityGroupIngress"
	if self.Direction == "egress" {
		action = "RevokeSecurityGroupEgress"
	}
	_, err := self.secgroup.region.vpcRequest(action, params)
	return err
}

func (self *SSecurityGroupRule) Update(opts *cloudprovider.SecurityGroupRuleUpdateOptions) error {
	return cloudprovider.ErrNotImplemented
}
