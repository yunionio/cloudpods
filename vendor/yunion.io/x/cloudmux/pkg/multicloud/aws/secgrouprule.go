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
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/utils"
)

type SSecurityGroupRule struct {
	group *SSecurityGroup

	FromPort            int    `xml:"fromPort"`
	GroupId             string `xml:"groupId"`
	IpProtocol          string `xml:"ipProtocol"`
	GroupOwnerId        string `xml:"groupOwnerId"`
	IsEgress            bool   `xml:"isEgress"`
	SecurityGroupRuleId string `xml:"securityGroupRuleId"`
	ReferencedGroupInfo struct {
		GroupId string `xml:"groupId"`
		UserId  string `xml:"userId"`
	} `xml:"referencedGroupInfo"`
	CidrIpv4     string `xml:"cidrIpv4"`
	CidrIpv6     string `xml:"cidrIpv6"`
	Description  string `xml:"description"`
	PrefixListId string `xml:"prefixListId"`
	ToPort       int    `xml:"toPort"`
}

func (self *SSecurityGroupRule) GetGlobalId() string {
	return self.SecurityGroupRuleId
}

func (self *SSecurityGroupRule) GetAction() secrules.TSecurityRuleAction {
	return secrules.SecurityRuleAllow
}

func (self *SSecurityGroupRule) GetDescription() string {
	return self.Description
}

func (self *SSecurityGroupRule) GetDirection() secrules.TSecurityRuleDirection {
	if self.IsEgress {
		return secrules.DIR_OUT
	}
	return secrules.DIR_IN
}

func (self *SSecurityGroupRule) GetCIDRs() []string {
	ret := []string{self.CidrIpv4 + self.CidrIpv6 + self.PrefixListId}
	return ret
}

func (self *SSecurityGroupRule) GetProtocol() string {
	if self.IpProtocol == "-1" {
		return secrules.PROTO_ANY
	}
	return strings.ToLower(self.IpProtocol)
}

func (self *SSecurityGroupRule) GetPorts() string {
	if self.FromPort > 0 && self.ToPort > 0 {
		if self.FromPort == self.ToPort {
			return fmt.Sprintf("%d", self.FromPort)
		}
		return fmt.Sprintf("%d-%d", self.FromPort, self.ToPort)
	}
	return ""
}

func (self *SSecurityGroupRule) GetPriority() int {
	return 0
}

func (self *SSecurityGroupRule) Delete() error {
	return self.group.region.DeleteSecurityGroupRule(self.GroupId, string(self.GetDirection()), self.SecurityGroupRuleId)
}

func (self *SRegion) GetSecurityGroupRules(id string) ([]SSecurityGroupRule, error) {
	ret := []SSecurityGroupRule{}
	params := map[string]string{
		"Filter.1.Name":    "group-id",
		"Filter.1.Value.1": id,
	}
	for {
		part := struct {
			NextToken            string               `xml:"nextToken"`
			SecurityGroupRuleSet []SSecurityGroupRule `xml:"securityGroupRuleSet>item"`
		}{}
		err := self.ec2Request("DescribeSecurityGroupRules", params, &part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.SecurityGroupRuleSet...)
		if len(part.NextToken) == 0 || len(part.SecurityGroupRuleSet) == 0 {
			break
		}
		params["NextToken"] = part.NextToken
	}
	return ret, nil
}

func (self *SSecurityGroupRule) Update(opts *cloudprovider.SecurityGroupRuleUpdateOptions) error {
	return self.group.region.UpdateSecurityGroupRule(self.group.GroupId, self.SecurityGroupRuleId, opts)
}

func (self *SRegion) UpdateSecurityGroupRule(secgroupId, ruleId string, opts *cloudprovider.SecurityGroupRuleUpdateOptions) error {
	if opts.Protocol == secrules.PROTO_ANY {
		opts.Protocol = "-1"
	}
	from, to := "-1", "-1"
	if len(opts.Ports) > 0 && utils.IsInStringArray(opts.Protocol, []string{secrules.PROTO_TCP, secrules.PROTO_UDP}) {
		r := secrules.SecurityRule{}
		r.ParsePorts(opts.Ports)
		if r.PortStart > 0 && r.PortEnd > 0 {
			from, to = fmt.Sprintf("%d", r.PortStart), fmt.Sprintf("%d", r.PortEnd)
		}
	}
	params := map[string]string{
		"GroupId": secgroupId,
		"SecurityGroupRule.1.SecurityGroupRuleId":           ruleId,
		"SecurityGroupRule.1.SecurityGroupRule.CidrIpv4":    opts.CIDR,
		"SecurityGroupRule.1.SecurityGroupRule.Description": opts.Desc,
		"SecurityGroupRule.1.SecurityGroupRule.IpProtocol":  opts.Protocol,
		"SecurityGroupRule.1.SecurityGroupRule.FromPort":    from,
		"SecurityGroupRule.1.SecurityGroupRule.ToPort":      to,
	}
	return self.ec2Request("ModifySecurityGroupRules", params, nil)
}
