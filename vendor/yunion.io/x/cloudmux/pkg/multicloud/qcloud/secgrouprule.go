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

package qcloud

import (
	"fmt"
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"
)

type ServiceTemplateSpecification struct {
	ServiceId      string //	协议端口ID，例如：ppm-f5n1f8da。
	ServiceGroupId string //	协议端口组ID，例如：ppmg-f5n1f8da。
}

func (self *ServiceTemplateSpecification) GetGlobalId() string {
	return self.ServiceId + self.ServiceGroupId
}

type AddressTemplateSpecification struct {
	AddressId      string //	IP地址ID，例如：ipm-2uw6ujo6。
	AddressGroupId string //	IP地址组ID，例如：ipmg-2uw6ujo6。
}

func (self *AddressTemplateSpecification) GetGlobalId() string {
	return self.AddressId + self.AddressGroupId
}

type SecurityGroupRule struct {
	secgroup *SSecurityGroup

	Direction         secrules.TSecurityRuleDirection
	PolicyIndex       int                          // 安全组规则索引号。
	Protocol          string                       // 协议, 取值: TCP,UDP, ICMP。
	Port              string                       // 端口(all, 离散port, range)。
	ServiceTemplate   ServiceTemplateSpecification // 协议端口ID或者协议端口组ID。ServiceTemplate和Protocol+Port互斥。
	CidrBlock         string                       // 网段或IP(互斥)。
	Ipv6CidrBlock     string
	SecurityGroupId   string                       // 已绑定安全组的网段或IP。
	AddressTemplate   AddressTemplateSpecification // IP地址ID或者ID地址组ID。
	Action            string                       // ACCEPT 或 DROP。
	PolicyDescription string                       // 安全组规则描述。
}

func (self *SecurityGroupRule) GetGlobalId() string {
	return fmt.Sprintf("%s-%d", self.Direction, self.PolicyIndex)
}

func (self *SecurityGroupRule) GetAction() secrules.TSecurityRuleAction {
	if self.Action == "DROP" {
		return secrules.SecurityRuleDeny
	}
	return secrules.SecurityRuleAllow
}

func (self *SecurityGroupRule) GetDescription() string {
	return self.PolicyDescription
}

func (self *SecurityGroupRule) GetDirection() secrules.TSecurityRuleDirection {
	return self.Direction
}

func (self *SecurityGroupRule) GetCIDRs() []string {
	ret := []string{self.CidrBlock + self.SecurityGroupId + self.Ipv6CidrBlock + self.AddressTemplate.GetGlobalId()}
	return ret
}

func (self *SecurityGroupRule) GetProtocol() string {
	if len(self.Protocol) == 0 {
		return self.ServiceTemplate.GetGlobalId()
	}
	if self.Protocol == "ALL" {
		return secrules.PROTO_ANY
	}
	return strings.ToLower(self.Protocol)
}

func (self *SecurityGroupRule) GetPorts() string {
	if len(self.ServiceTemplate.GetGlobalId()) > 0 {
		return self.ServiceTemplate.GetGlobalId()
	}
	if self.Port == "ALL" {
		return ""
	}
	return self.Port
}

func (self *SecurityGroupRule) GetPriority() int {
	return self.PolicyIndex
}

func (self *SecurityGroupRule) Delete() error {
	return self.secgroup.region.DeleteSecgroupRule(self.secgroup.SecurityGroupId, self.GetDirection(), self.PolicyIndex)
}

func (self *SRegion) DeleteSecgroupRule(secId string, direction secrules.TSecurityRuleDirection, index int) error {
	params := map[string]string{
		"SecurityGroupId": secId,
	}
	switch direction {
	case secrules.DIR_IN:
		params["SecurityGroupPolicySet.Ingress.0.PolicyIndex"] = fmt.Sprintf("%d", index)
	case secrules.DIR_OUT:
		params["SecurityGroupPolicySet.Egress.0.PolicyIndex"] = fmt.Sprintf("%d", index)
	}
	_, err := self.vpcRequest("DeleteSecurityGroupPolicies", params)
	return errors.Wrapf(err, "DeleteSecurityGroupPolicies")
}

type SecurityGroupPolicySet struct {
	Version string
	Egress  []SecurityGroupRule //	出站规则。
	Ingress []SecurityGroupRule //	入站规则。
}

func (self *SRegion) GetSecurityGroupRules(secGroupId string) ([]SecurityGroupRule, error) {
	params := make(map[string]string)
	params["Region"] = self.Region
	params["SecurityGroupId"] = secGroupId

	body, err := self.vpcRequest("DescribeSecurityGroupPolicies", params)
	if err != nil {
		log.Errorf("DescribeSecurityGroupAttribute fail %s", err)
		return nil, err
	}

	policies := SecurityGroupPolicySet{}
	err = body.Unmarshal(&policies, "SecurityGroupPolicySet")
	if err != nil {
		return nil, errors.Wrapf(err, "body.Unmarshal")
	}
	ret := []SecurityGroupRule{}
	for i := range policies.Egress {
		policies.Egress[i].Direction = secrules.DIR_OUT
		ret = append(ret, policies.Egress[i])
	}
	for i := range policies.Ingress {
		policies.Ingress[i].Direction = secrules.DIR_IN
		ret = append(ret, policies.Ingress[i])
	}
	return ret, nil
}

func (self *SecurityGroupRule) Update(opts *cloudprovider.SecurityGroupRuleUpdateOptions) error {
	return self.secgroup.region.UpdateSecurityGroupRule(self.secgroup.SecurityGroupId, self.PolicyIndex, self.GetDirection(), opts)
}

func (self *SRegion) UpdateSecurityGroupRule(groupId string, idx int, direction secrules.TSecurityRuleDirection, opts *cloudprovider.SecurityGroupRuleUpdateOptions) error {
	params := map[string]string{
		"SecurityGroupId": groupId,
	}
	prefix := "SecurityGroupPolicySet.Ingress.0."
	if direction == secrules.DIR_OUT {
		prefix = "SecurityGroupPolicySet.Egress.0."
	}
	if len(opts.Ports) == 0 || opts.Protocol == secrules.PROTO_ANY {
		opts.Ports = "all"
	}
	if opts.Protocol == secrules.PROTO_ANY {
		opts.Protocol = "ALL"
	}
	action := "ACCEPT"
	if opts.Action == secrules.SecurityRuleDeny {
		action = "DROP"
	}
	params[prefix+"PolicyIndex"] = fmt.Sprintf("%d", idx)
	params[prefix+"Protocol"] = strings.ToUpper(opts.Protocol)
	params[prefix+"Port"] = opts.Ports
	params[prefix+"CidrBlock"] = opts.CIDR
	params[prefix+"Action"] = action
	params[prefix+"PolicyDescription"] = opts.Desc
	_, err := self.vpcRequest("ReplaceSecurityGroupPolicy", params)
	return err
}
