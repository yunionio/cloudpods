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
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/secrules"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SSecurityGroup struct {
	multicloud.SSecurityGroup
	AwsTags
	region *SRegion

	GroupId          string `xml:"groupId"`
	VpcId            string `xml:"vpcId"`
	GroupName        string `xml:"groupName"`
	GroupDescription string `xml:"groupDescription"`
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
	return api.SECGROUP_STATUS_READY
}

func (self *SSecurityGroup) Refresh() error {
	group, err := self.region.GetSecurityGroup(self.GroupId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, group)
}

func (self *SSecurityGroup) SetTags(tags map[string]string, replace bool) error {
	return self.region.setTags("security-group", self.GroupId, tags, replace)
}

func (self *SSecurityGroup) GetDescription() string {
	return self.GroupDescription
}

func (self *SSecurityGroup) GetRules() ([]cloudprovider.ISecurityGroupRule, error) {
	ret := []cloudprovider.ISecurityGroupRule{}
	rules, err := self.region.GetSecurityGroupRules(self.GroupId)
	if err != nil {
		return nil, err
	}
	for i := range rules {
		rules[i].group = self
		ret = append(ret, &rules[i])
	}
	return ret, nil
}

func (self *SRegion) CreateSecurityGroupRule(secGrpId string, opts *cloudprovider.SecurityGroupRuleCreateOptions) (*SSecurityGroupRule, error) {
	params := map[string]string{
		"GroupId":                    secGrpId,
		"IpPermissions.1.IpProtocol": "-1",
		"IpPermissions.1.FromPort":   "0",
		"IpPermissions.1.ToPort":     "65535",
	}
	if opts.Protocol != secrules.PROTO_ANY {
		params["IpPermissions.1.IpProtocol"] = strings.ToLower(opts.Protocol)
	}
	if len(opts.CIDR) == 0 {
		opts.CIDR = "0.0.0.0/0"
	}
	if _, err := netutils.NewIPV6Prefix(opts.CIDR); err == nil {
		params["IpPermissions.1.Ipv6Ranges.1.CidrIpv6"] = opts.CIDR
		params["IpPermissions.1.Ipv6Ranges.1.Description"] = opts.Desc
	} else {
		if !strings.Contains(opts.CIDR, "/") {
			opts.CIDR = opts.CIDR + "/32"
		}
		params["IpPermissions.1.IpRanges.1.CidrIp"] = opts.CIDR
		params["IpPermissions.1.IpRanges.1.Description"] = opts.Desc
	}
	start, end := 0, 0
	if len(opts.Ports) > 0 {
		if strings.Contains(opts.Ports, "-") {
			ports := strings.Split(opts.Ports, "-")
			if len(ports) != 2 {
				return nil, errors.Errorf("invalid ports %s", opts.Ports)
			}
			var err error
			_start, _end := ports[0], ports[1]
			start, err = strconv.Atoi(_start)
			if err != nil {
				return nil, errors.Errorf("invalid start port %s", _start)
			}
			end, err = strconv.Atoi(_end)
			if err != nil {
				return nil, errors.Errorf("invalid end port %s", _end)
			}
		} else {
			port, err := strconv.Atoi(opts.Ports)
			if err != nil {
				return nil, errors.Errorf("invalid ports %s", opts.Ports)
			}
			start, end = port, port
		}
	}
	if start > 0 && end > 0 {
		params["IpPermissions.1.FromPort"] = fmt.Sprintf("%d", start)
		params["IpPermissions.1.ToPort"] = fmt.Sprintf("%d", end)
	}
	if opts.Protocol == secrules.PROTO_ICMP {
		params["IpPermissions.1.FromPort"] = "-1"
		params["IpPermissions.1.ToPort"] = "-1"
	}
	action := "AuthorizeSecurityGroupIngress"
	if opts.Direction == secrules.DIR_OUT {
		action = "AuthorizeSecurityGroupEgress"
	}
	ret := struct {
		Return               bool                 `xml:"return"`
		SecurityGroupRuleSet []SSecurityGroupRule `xml:"securityGroupRuleSet>item"`
	}{}

	err := self.ec2Request(action, params, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, action)
	}
	for i := range ret.SecurityGroupRuleSet {
		return &ret.SecurityGroupRuleSet[i], nil
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "after create %s", jsonutils.Marshal(opts))
}

func (self *SRegion) DeleteSecurityGroupRule(secGrpId string, direction, ruleId string) error {
	params := map[string]string{
		"GroupId":               secGrpId,
		"SecurityGroupRuleId.1": ruleId,
	}
	action := "RevokeSecurityGroupEgress"
	if direction == secrules.DIR_IN {
		action = "RevokeSecurityGroupIngress"
	}
	return self.ec2Request(action, params, nil)
}

func (self *SRegion) CreateSecurityGroup(opts *cloudprovider.SecurityGroupCreateInput) (string, error) {
	params := map[string]string{
		"VpcId":            opts.VpcId,
		"GroupDescription": opts.Desc,
		"GroupName":        opts.Name,
	}
	if len(opts.Desc) == 0 {
		params["GroupDescription"] = "auto create by cloudpods"
	}
	tagIdx := 1
	for k, v := range opts.Tags {
		params[fmt.Sprintf("TagSpecification.1.ResourceType")] = "security-group"
		params[fmt.Sprintf("TagSpecification.1.Tag.%d.Key", tagIdx)] = k
		params[fmt.Sprintf("TagSpecification.1.Tag.%d.Value", tagIdx)] = v
		tagIdx++
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

func (self *SSecurityGroup) Delete() error {
	if self.GroupName == "default" {
		return cloudprovider.ErrNotSupported
	}
	return self.region.DeleteSecurityGroup(self.GroupId)
}

func (self *SSecurityGroup) CreateRule(opts *cloudprovider.SecurityGroupRuleCreateOptions) (cloudprovider.ISecurityGroupRule, error) {
	rule, err := self.region.CreateSecurityGroupRule(self.GroupId, opts)
	if err != nil {
		return nil, err
	}
	rule.group = self
	return rule, nil
}
