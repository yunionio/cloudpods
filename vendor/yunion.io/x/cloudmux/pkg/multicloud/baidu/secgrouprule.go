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
	"strings"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/util/secrules"
)

type SSecurityGroupRule struct {
	region *SRegion

	Remark              string
	Direction           string
	Ethertype           string
	PortRange           string
	DestGroupId         string
	DestIp              string
	SourceIp            string
	SourceGroupId       string
	SecurityGroupId     string
	SecurityGroupRuleId string
	CreatedTime         time.Time
	UpdatedTime         time.Time
	Protocol            string
}

func (self *SSecurityGroupRule) GetGlobalId() string {
	return self.SecurityGroupRuleId
}

func (self *SSecurityGroupRule) GetAction() secrules.TSecurityRuleAction {
	return secrules.SecurityRuleAllow
}

func (self *SSecurityGroupRule) GetDescription() string {
	return self.Remark
}

func (self *SSecurityGroupRule) GetDirection() secrules.TSecurityRuleDirection {
	if self.Direction == "ingress" {
		return secrules.DIR_IN
	}
	return secrules.DIR_OUT
}

func getCidr(ip string, version string) string {
	switch version {
	case "IPv6":
		if ip == "all" {
			return "::/0"
		}
		return ip
	case "IPv4":
		if ip == "all" {
			return "0.0.0.0/0"
		}
		return ip
	default:
		return ip
	}
}

func (self *SSecurityGroupRule) GetCIDRs() []string {
	ret := []string{}
	if len(self.DestGroupId) > 0 {
		ret = append(ret, self.DestGroupId)
	}
	if len(self.DestIp) > 0 {
		ret = append(ret, getCidr(self.DestIp, self.Ethertype))
	}
	if len(self.SourceIp) > 0 {
		ret = append(ret, getCidr(self.SourceIp, self.Ethertype))
	}
	if len(self.SourceGroupId) > 0 {
		ret = append(ret, self.SourceGroupId)
	}
	return ret
}

func (self *SSecurityGroupRule) GetProtocol() string {
	if strings.ToLower(self.Protocol) == "all" || len(self.Protocol) == 0 {
		return secrules.PROTO_ANY
	}
	return strings.ToLower(self.Protocol)
}

func (self *SSecurityGroupRule) GetPorts() string {
	if self.PortRange == "1-65535" || self.PortRange == "" {
		return ""
	}
	return self.PortRange
}

func (self *SSecurityGroupRule) GetPriority() int {
	return 1
}

func (self *SSecurityGroupRule) Delete() error {
	return self.region.DeleteSecurityGroupRule(self.SecurityGroupRuleId)
}

func (region *SRegion) DeleteSecurityGroupRule(id string) error {
	_, err := region.bccDelete(fmt.Sprintf("v2/securityGroup/rule/%s", id), nil)
	return err
}

func (self *SSecurityGroupRule) Update(opts *cloudprovider.SecurityGroupRuleUpdateOptions) error {
	return self.region.UpdateSecurityGroupRule(self.GetGlobalId(), self.Direction, opts)
}

func (region *SRegion) UpdateSecurityGroupRule(ruleId string, direction string, opts *cloudprovider.SecurityGroupRuleUpdateOptions) error {
	params := url.Values{}
	body := map[string]interface{}{
		"remark":              opts.Desc,
		"protocol":            opts.Protocol,
		"portRange":           opts.Ports,
		"securityGroupRuleId": ruleId,
	}
	if len(opts.CIDR) > 0 {
		if direction == "egress" {
			body["destIp"] = opts.CIDR
		} else {
			body["sourceIp"] = opts.CIDR
		}
	}
	_, err := region.bccUpdate("v2/securityGroup/rule/update", params, body)
	return err
}
