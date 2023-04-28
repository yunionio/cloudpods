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
	"net"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SSecurityGroupRule struct {
	IpProtocol string `xml:"ipProtocol"`
	Groups     []struct {
		GroupId string `xml:"groupId"`
	} `xml:"groups>item"`
	IpRanges []struct {
		CidrIp string `xml:"cidrIp"`
	} `xml:"ipRanges>item"`
	Ipv6Ranges []struct {
		CidrIpv6 string `xml:"cidrIpv6"`
	} `xml:"ipv6Ranges>item"`
	PrefixListIds []struct {
		PrefixListId string `xml:"prefixListId"`
	} `xml:"prefixListIds>item"`
	FromPort int `xml:"fromPort"`
	ToPort   int `xml:"toPort"`
}

type SSecurityGroup struct {
	multicloud.SSecurityGroup
	AwsTags
	region *SRegion

	GroupId             string               `xml:"groupId"`
	VpcId               string               `xml:"vpcId"`
	GroupName           string               `xml:"groupName"`
	GroupDescription    string               `xml:"groupDescription"`
	IpPermissions       []SSecurityGroupRule `xml:"ipPermissions>item"`
	IpPermissionsEgress []SSecurityGroupRule `xml:"ipPermissionsEgress>item"`
}

func (self *SSecurityGroup) GetId() string {
	return self.GroupId
}

func (self *SSecurityGroup) GetVpcId() string {
	return self.VpcId
}

func (self *SSecurityGroup) GetName() string {
	if len(self.GroupName) > 0 {
		return self.GroupName
	}
	return self.GroupId
}

func (self *SSecurityGroup) GetGlobalId() string {
	return self.GroupId
}

func (self *SSecurityGroup) GetStatus() string {
	return ""
}

func (self *SSecurityGroup) Refresh() error {
	group, err := self.region.GetSecurityGroup(self.GroupId)
	if err != nil {
		return err
	}
	self.IpPermissions = group.IpPermissions
	self.IpPermissionsEgress = group.IpPermissionsEgress
	return jsonutils.Update(self, group)
}

func (self *SSecurityGroup) GetDescription() string {
	return self.GroupDescription
}

func (self *SSecurityGroup) GetRules() ([]cloudprovider.SecurityRule, error) {
	ret := []cloudprovider.SecurityRule{}
	for direction, rules := range map[secrules.TSecurityRuleDirection][]SSecurityGroupRule{
		secrules.DIR_IN:  self.IpPermissions,
		secrules.DIR_OUT: self.IpPermissionsEgress,
	} {
		for i := range rules {
			if len(rules[i].IpRanges) == 0 { // 非cidr安全组规则不支持
				continue
			}
			if !utils.IsInStringArray(rules[i].IpProtocol, []string{"-1", "tcp", "udp", "icmp"}) {
				continue
			}
			protocol := rules[i].IpProtocol
			if protocol == "-1" {
				protocol = secrules.PROTO_ANY
			}
			portStart, portEnd := -1, -1
			if rules[i].FromPort > 0 && rules[i].ToPort > 0 {
				portStart, portEnd = rules[i].FromPort, rules[i].ToPort
			}
			for _, cidr := range rules[i].IpRanges {
				_, ipNet, err := net.ParseCIDR(cidr.CidrIp)
				if err != nil {
					return nil, errors.Wrapf(err, "net.ParseCIDR(%s)", cidr.CidrIp)
				}
				ret = append(ret, cloudprovider.SecurityRule{
					SecurityRule: secrules.SecurityRule{
						Priority:  1,
						Action:    secrules.SecurityRuleAllow,
						Direction: direction,
						IPNet:     ipNet,
						Protocol:  protocol,
						PortStart: portStart,
						PortEnd:   portEnd,
					},
				})
			}
		}
	}
	return ret, nil
}

func (self *SRegion) RemoveSecurityGroupRule(secGrpId string, rule secrules.SecurityRule) error {
	params := map[string]string{
		"GroupId": secGrpId,
	}
	idx := 1
	params[fmt.Sprintf("IpPermissions.%d.IpProtocol", idx)] = "-1"
	if rule.Protocol != secrules.PROTO_ANY {
		params[fmt.Sprintf("IpPermissions.%d.IpProtocol", idx)] = strings.ToLower(rule.Protocol)
	}
	if rule.IPNet != nil {
		params[fmt.Sprintf("IpPermissions.%d.IpRanges.1.CidrIp", idx)] = rule.IPNet.String()
	}
	if rule.PortStart > 0 && rule.PortEnd > 0 {
		params[fmt.Sprintf("IpPermissions.%d.FromPort", idx)] = fmt.Sprintf("%d", rule.PortStart)
		params[fmt.Sprintf("IpPermissions.%d.ToPort", idx)] = fmt.Sprintf("%d", rule.PortEnd)
	}
	action := "RevokeSecurityGroupIngress"
	if rule.Direction == secrules.DIR_OUT {
		action = "RevokeSecurityGroupEgress"
	}
	return self.ec2Request(action, params, nil)
}

func (self *SRegion) AddSecurityGroupRule(secGrpId string, direction secrules.TSecurityRuleDirection, rules []secrules.SecurityRule) error {
	if len(rules) == 0 {
		return nil
	}
	params := map[string]string{
		"GroupId": secGrpId,
	}
	idx := 1
	for i := range rules {
		rule := rules[i]
		if len(rule.Ports) > 0 {
			for _, port := range rule.Ports {
				if rule.Protocol != secrules.PROTO_ANY {
					params[fmt.Sprintf("IpPermissions.%d.IpProtocol", idx)] = strings.ToLower(rule.Protocol)
				}
				if rule.IPNet != nil {
					params[fmt.Sprintf("IpPermissions.%d.IpRanges.1.CidrIp", idx)] = rule.IPNet.String()
				}
				params[fmt.Sprintf("IpPermissions.%d.FromPort", idx)] = fmt.Sprintf("%d", port)
				params[fmt.Sprintf("IpPermissions.%d.ToPort", idx)] = fmt.Sprintf("%d", port)
				idx++
			}
			continue
		}
		if rule.Protocol != secrules.PROTO_ANY {
			params[fmt.Sprintf("IpPermissions.%d.IpProtocol", idx)] = strings.ToLower(rule.Protocol)
		}
		if rule.IPNet != nil {
			params[fmt.Sprintf("IpPermissions.%d.IpRanges.1.CidrIp", idx)] = rule.IPNet.String()
		}
		params[fmt.Sprintf("IpPermissions.%d.FromPort", idx)] = "0"
		params[fmt.Sprintf("IpPermissions.%d.ToPort", idx)] = "65535"
		if rule.Protocol == secrules.PROTO_ICMP {
			params[fmt.Sprintf("IpPermissions.%d.FromPort", idx)] = "-1"
			params[fmt.Sprintf("IpPermissions.%d.ToPort", idx)] = "-1"
		}
		if rule.PortStart > 0 && rule.PortEnd > 0 {
			params[fmt.Sprintf("IpPermissions.%d.FromPort", idx)] = fmt.Sprintf("%d", rule.PortStart)
			params[fmt.Sprintf("IpPermissions.%d.ToPort", idx)] = fmt.Sprintf("%d", rule.PortEnd)
		}
		idx++
	}
	action := "AuthorizeSecurityGroupIngress"
	if direction == secrules.DIR_OUT {
		action = "AuthorizeSecurityGroupEgress"
	}
	return self.ec2Request(action, params, nil)
}

func (self *SRegion) DelSecurityGroupRule(secGrpId string, ruleId string) error {
	params := map[string]string{
		"GroupId":               secGrpId,
		"SecurityGroupRuleId.1": ruleId,
	}
	return self.ec2Request("RevokeSecurityGroupEgress", params, nil)
}

func (self *SRegion) CreateSecurityGroup(vpcId string, name string, desc string) (string, error) {
	params := map[string]string{
		"VpcId":            vpcId,
		"GroupDescription": desc,
		"GroupName":        name,
	}
	if len(desc) == 0 {
		params["GroupDescription"] = "auto create by cloudpods"
	}
	ret := struct {
		GroupId string `xml:"groupId"`
	}{}
	err := self.ec2Request("CreateSecurityGroup", params, &ret)
	if err != nil {
		return "", err
	}
	return ret.GroupId, nil
}

func (self *SRegion) GetSecurityGroup(id string) (*SSecurityGroup, error) {
	groups, err := self.GetSecurityGroups("", "", id)
	if err != nil {
		return nil, err
	}
	for i := range groups {
		if groups[i].GetGlobalId() == id {
			groups[i].region = self
			return &groups[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetSecurityGroups(vpcId string, name string, secgroupId string) ([]SSecurityGroup, error) {
	params := map[string]string{}
	idx := 1
	if len(vpcId) > 0 {
		params[fmt.Sprintf("Filter.%d.Name", idx)] = "vpc-id"
		params[fmt.Sprintf("Filter.%d.Value.1", idx)] = vpcId
		idx++
	}
	if len(name) > 0 {
		params[fmt.Sprintf("Filter.%d.Name", idx)] = "group-name"
		params[fmt.Sprintf("Filter.%d.Value.1", idx)] = name
		idx++
	}
	if len(secgroupId) > 0 {
		params["GroupId.0"] = secgroupId
	}
	result := []SSecurityGroup{}
	for {
		ret := struct {
			NextToken         string           `xml:"nextToken"`
			SecurityGroupInfo []SSecurityGroup `xml:"securityGroupInfo>item"`
		}{}
		err := self.ec2Request("DescribeSecurityGroups", params, &ret)
		if err != nil {
			return nil, err
		}
		result = append(result, ret.SecurityGroupInfo...)
		if len(ret.NextToken) == 0 || len(ret.SecurityGroupInfo) == 0 {
			break
		}
		params["NextToken"] = ret.NextToken
	}
	return result, nil
}

func (self *SSecurityGroup) GetProjectId() string {
	return ""
}

func (self *SSecurityGroup) Delete() error {
	return self.region.DeleteSecurityGroup(self.GroupId)
}
