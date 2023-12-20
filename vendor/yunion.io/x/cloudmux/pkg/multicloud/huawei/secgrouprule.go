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

package huawei

import (
	"fmt"
	"net/url"
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"
)

type SecurityGroupRule struct {
	secgroup *SSecurityGroup

	Direction       string
	Ethertype       string
	Id              string
	Description     string
	PortRangeMax    int64
	PortRangeMin    int64
	Protocol        string
	RemoteGroupId   string
	RemoteIPPrefix  string
	SecurityGroupId string
	TenantId        string
	Priority        int
}

func (self *SecurityGroupRule) GetGlobalId() string {
	return self.Id
}

func (self *SecurityGroupRule) GetDescription() string {
	return self.Description
}

func (self *SecurityGroupRule) GetDirection() secrules.TSecurityRuleDirection {
	if self.Direction == "egress" {
		return secrules.DIR_OUT
	}
	return secrules.DIR_IN
}

func (self *SecurityGroupRule) GetPriority() int {
	return self.Priority
}

func (self *SecurityGroupRule) GetAction() secrules.TSecurityRuleAction {
	return secrules.SecurityRuleAllow
}

func (self *SecurityGroupRule) GetProtocol() string {
	if len(self.Protocol) == 0 {
		self.Protocol = secrules.PROTO_ANY
	}
	return strings.ToLower(self.Protocol)
}

func (self *SecurityGroupRule) GetPorts() string {
	if self.PortRangeMax > 0 && self.PortRangeMin > 0 {
		if self.PortRangeMax == self.PortRangeMin {
			return fmt.Sprintf("%d", self.PortRangeMax)
		}
		return fmt.Sprintf("%d-%d", self.PortRangeMin, self.PortRangeMax)
	}
	return ""
}

type SPageInfo struct {
	NextMarker string
}

func (self *SecurityGroupRule) GetCIDRs() []string {
	ip := self.RemoteIPPrefix + self.RemoteGroupId
	if len(ip) == 0 {
		ip = "0.0.0.0"
		if self.Ethertype == "IPv6" {
			ip = "::/0"
		}
	}
	ret := []string{ip}
	return ret
}

func (self *SecurityGroupRule) Delete() error {
	return self.secgroup.region.DeleteSecurityGroupRule(self.Id)
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/VPC/doc?version=v3&api=DeleteSecurityGroupRule
func (self *SRegion) DeleteSecurityGroupRule(id string) error {
	_, err := self.delete(SERVICE_VPC_V3, "vpc/security-group-rules/"+id)
	return err
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/VPC/doc?version=v3&api=ListSecurityGroupRules
func (self *SRegion) GetSecurityGroupRules(groupId string) ([]SecurityGroupRule, error) {
	params := url.Values{}
	params.Set("security_group_id", groupId)

	ret := []SecurityGroupRule{}
	for {
		resp, err := self.list(SERVICE_VPC_V3, "vpc/security-group-rules", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			SecurityGroupRules []SecurityGroupRule
			PageInfo           SPageInfo
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrapf(err, "Unmarshal")
		}
		ret = append(ret, part.SecurityGroupRules...)
		if len(part.PageInfo.NextMarker) == 0 || len(part.SecurityGroupRules) == 0 {
			break
		}
		params.Set("marker", part.PageInfo.NextMarker)
	}
	return ret, nil
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/VPC/doc?version=v3&api=CreateSecurityGroupRule
func (self *SRegion) CreateSecurityGroupRule(groupId string, opts *cloudprovider.SecurityGroupRuleCreateOptions) (*SecurityGroupRule, error) {
	rule := map[string]interface{}{
		"security_group_id": groupId,
		"description":       opts.Desc,
		"direction":         "ingress",
		"ethertype":         "IPv4",
		"protocol":          strings.ToLower(opts.Protocol),
		"action":            "allow",
		"priority":          opts.Priority,
	}
	if len(opts.CIDR) > 0 {
		rule["remote_ip_prefix"] = opts.CIDR
	}
	if opts.Action == secrules.SecurityRuleDeny {
		rule["action"] = "deny"
	}
	if opts.Protocol == secrules.PROTO_ANY {
		delete(rule, "protocol")
	}
	if len(opts.Ports) > 0 {
		rule["multiport"] = opts.Ports
	}
	if opts.Direction == secrules.DIR_OUT {
		rule["direction"] = "egress"
	}
	params := map[string]interface{}{
		"security_group_rule": rule,
	}
	resp, err := self.post(SERVICE_VPC_V3, "vpc/security-group-rules", params)
	if err != nil {
		return nil, errors.Wrapf(err, "create rule")
	}
	ret := &SecurityGroupRule{}
	return ret, resp.Unmarshal(ret, "security_group_rule")
}

func (self *SecurityGroupRule) Update(opts *cloudprovider.SecurityGroupRuleUpdateOptions) error {
	return cloudprovider.ErrNotSupported
}
