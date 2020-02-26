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

/*
https://support.huaweicloud.com/usermanual-vpc/zh-cn_topic_0073379079.html
安全组的限制
默认情况下，一个用户可以创建100个安全组。
默认情况下，一个安全组最多只允许拥有50条安全组规则。
默认情况下，一个弹性云服务器或辅助网卡最多只能被添加到5个安全组中。
在创建私网弹性负载均衡时，需要选择弹性负载均衡所在的安全组。请勿删除默认规则或者确保满足以下规则：
出方向：允许发往同一个安全组的报文可以通过，或者允许对端负载均衡器报文通过。
入方向：允许来自同一个安全组的报文可以通过，或者允许对端负载均衡器报文通过。
*/

import (
	"net"
	"sort"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/secrules"
)

type SecurityGroupRule struct {
	Direction       string `json:"direction"`
	Ethertype       string `json:"ethertype"`
	ID              string `json:"id"`
	Description     string `json:"description"`
	SecurityGroupID string `json:"security_group_id"`
	RemoteGroupID   string `json:"remote_group_id"`
}

type SecurityGroupRuleDetail struct {
	Direction       string `json:"direction"`
	Ethertype       string `json:"ethertype"`
	ID              string `json:"id"`
	Description     string `json:"description"`
	PortRangeMax    int64  `json:"port_range_max"`
	PortRangeMin    int64  `json:"port_range_min"`
	Protocol        string `json:"protocol"`
	RemoteGroupID   string `json:"remote_group_id"`
	RemoteIPPrefix  string `json:"remote_ip_prefix"`
	SecurityGroupID string `json:"security_group_id"`
	TenantID        string `json:"tenant_id"`
}

// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0020090615.html
type SSecurityGroup struct {
	region *SRegion

	ID                  string              `json:"id"`
	Name                string              `json:"name"`
	Description         string              `json:"description"`
	VpcID               string              `json:"vpc_id"`
	EnterpriseProjectID string              `json:"enterprise_project_id "`
	SecurityGroupRules  []SecurityGroupRule `json:"security_group_rules"`
}

// 判断是否兼容云端安全组规则
func compatibleSecurityGroupRule(r SecurityGroupRule) bool {
	// 忽略了源地址是安全组的规则
	if len(r.RemoteGroupID) > 0 {
		return false
	}

	// 忽略IPV6
	if r.Ethertype == "IPv6" {
		return false
	}

	return true
}

// 将安全组规则全部转换为等价的allow规则
func SecurityRuleSetToAllowSet(srs secrules.SecurityRuleSet) secrules.SecurityRuleSet {
	inRuleSet := secrules.SecurityRuleSet{}
	outRuleSet := secrules.SecurityRuleSet{}

	for _, rule := range srs {
		if rule.Direction == secrules.SecurityRuleIngress {
			inRuleSet = append(inRuleSet, rule)
		}

		if rule.Direction == secrules.SecurityRuleEgress {
			outRuleSet = append(outRuleSet, rule)
		}
	}

	sort.Sort(inRuleSet)
	sort.Sort(outRuleSet)

	inRuleSet = inRuleSet.AllowList()
	// out方向空规则默认全部放行
	if outRuleSet.Len() == 0 {
		_, ipNet, _ := net.ParseCIDR("0.0.0.0/0")
		outRuleSet = append(outRuleSet, secrules.SecurityRule{
			Priority:  0,
			Action:    secrules.SecurityRuleAllow,
			IPNet:     ipNet,
			Protocol:  secrules.PROTO_ANY,
			Direction: secrules.SecurityRuleEgress,
			PortStart: -1,
			PortEnd:   -1,
		})
	}
	outRuleSet = outRuleSet.AllowList()

	ret := secrules.SecurityRuleSet{}
	ret = append(ret, inRuleSet...)
	ret = append(ret, outRuleSet...)
	return ret
}

func (self *SSecurityGroup) GetId() string {
	return self.ID
}

func (self *SSecurityGroup) GetVpcId() string {
	// 无vpc关联的安全组统一返回classic
	if len(self.VpcID) == 0 {
		return "classic"
	}

	return self.VpcID
}

func (self *SSecurityGroup) GetName() string {
	if len(self.Name) > 0 {
		return self.Name
	}
	return self.ID
}

func (self *SSecurityGroup) GetGlobalId() string {
	return self.ID
}

func (self *SSecurityGroup) GetStatus() string {
	return ""
}

func (self *SSecurityGroup) Refresh() error {
	if new, err := self.region.GetSecurityGroupDetails(self.GetId()); err != nil {
		return err
	} else {
		return jsonutils.Update(self, new)
	}
}

func (self *SSecurityGroup) IsEmulated() bool {
	return false
}

func (self *SSecurityGroup) GetMetadata() *jsonutils.JSONDict {
	data := jsonutils.NewDict()
	return data
}

func (self *SSecurityGroup) GetDescription() string {
	if self.Description == self.VpcID {
		return ""
	}
	return self.Description
}

// todo: 这里需要优化查询太多了
func (self *SSecurityGroup) GetRules() ([]secrules.SecurityRule, error) {
	rules := make([]secrules.SecurityRule, 0)
	for _, r := range self.SecurityGroupRules {
		if !compatibleSecurityGroupRule(r) {
			continue
		}

		rule, err := self.GetSecurityRule(r.ID, false)
		if err != nil {
			return rules, err
		}

		rules = append(rules, rule)
	}

	return rules, nil
}

func (self *SSecurityGroup) GetRulesWithExtId() ([]secrules.SecurityRule, error) {
	rules := make([]secrules.SecurityRule, 0)
	for _, r := range self.SecurityGroupRules {
		if !compatibleSecurityGroupRule(r) {
			continue
		}

		rule, err := self.GetSecurityRule(r.ID, true)
		if err != nil {
			return rules, err
		}

		rules = append(rules, rule)
	}

	return rules, nil
}

// withRuleId.
func (self *SSecurityGroup) GetSecurityRule(ruleId string, withRuleId bool) (secrules.SecurityRule, error) {
	remoteRule := SecurityGroupRuleDetail{}
	err := DoGet(self.region.ecsClient.SecurityGroupRules.Get, ruleId, nil, &remoteRule)
	if err != nil {
		return secrules.SecurityRule{}, err
	}

	var direction secrules.TSecurityRuleDirection
	if remoteRule.Direction == "ingress" {
		direction = secrules.SecurityRuleIngress
	} else {
		direction = secrules.SecurityRuleEgress
	}

	protocol := secrules.PROTO_ANY
	if remoteRule.Protocol != "" {
		protocol = remoteRule.Protocol
	}

	var portStart int
	var portEnd int
	if protocol == secrules.PROTO_ICMP {
		portStart = -1
		portEnd = -1
	} else {
		portStart = int(remoteRule.PortRangeMin)
		portEnd = int(remoteRule.PortRangeMax)
	}

	ipNet := &net.IPNet{}
	if len(remoteRule.RemoteIPPrefix) > 0 {
		_, ipNet, err = net.ParseCIDR(remoteRule.RemoteIPPrefix)
	} else {
		_, ipNet, err = net.ParseCIDR("0.0.0.0/0")
	}

	if err != nil {
		return secrules.SecurityRule{}, err
	}

	// withRuleId.将ruleId附加到description字段。该hook有特殊目的，仅在同步安全组时使用。
	desc := ""
	if withRuleId {
		desc = ruleId
	} else {
		desc = remoteRule.Description
	}

	rule := secrules.SecurityRule{
		Priority:    1,
		Action:      secrules.SecurityRuleAllow,
		IPNet:       ipNet,
		Protocol:    protocol,
		Direction:   direction,
		PortStart:   portStart,
		PortEnd:     portEnd,
		Ports:       nil,
		Description: desc,
	}

	err = rule.ValidateRule()
	return rule, err
}

func (self *SRegion) GetSecurityGroupDetails(secGroupId string) (*SSecurityGroup, error) {
	securitygroup := SSecurityGroup{}
	err := DoGet(self.ecsClient.SecurityGroups.Get, secGroupId, nil, &securitygroup)
	if err != nil {
		return nil, err
	}

	securitygroup.region = self
	return &securitygroup, err
}

// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0020090617.html
func (self *SRegion) GetSecurityGroups(vpcId string, name string) ([]SSecurityGroup, error) {
	querys := map[string]string{}
	if len(vpcId) > 0 {
		querys["vpc_id"] = vpcId
	}

	securitygroups := make([]SSecurityGroup, 0)
	err := doListAllWithMarker(self.ecsClient.SecurityGroups.List, querys, &securitygroups)
	if err != nil {
		return nil, err
	}

	// security 中的vpc字段只是一个标识，实际可以跨vpc使用
	for i := range securitygroups {
		securitygroup := &securitygroups[i]
		securitygroup.region = self
	}

	result := []SSecurityGroup{}
	for _, secgroup := range securitygroups {
		if len(name) == 0 || secgroup.Name == name {
			result = append(result, secgroup)
		}
	}

	return result, nil
}

func (self *SSecurityGroup) GetProjectId() string {
	return ""
}

func (self *SSecurityGroup) Delete() error {
	return self.region.DeleteSecurityGroup(self.ID)
}

func (self *SSecurityGroup) SyncRules(rules []secrules.SecurityRule) error {
	rules = SecurityRuleSetToAllowSet(rules)
	return self.region.syncSecgroupRules(self.ID, rules)
}
