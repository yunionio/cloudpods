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

package zstack

import (
	"fmt"
	"net"
	"sort"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/secrules"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

type SSecurityGroupRule struct {
	ZStackBasic
	SecurityGroupUUID       string `json:"securityGroupUuid"`
	Type                    string `json:"type"`
	IPVersion               int    `json:"ipVersion"`
	StartPort               int    `json:"startPort"`
	EndPort                 int    `json:"endPort"`
	Protocol                string `json:"protocol"`
	State                   string `json:"state"`
	AllowedCIDR             string `json:"allowedCidr"`
	RemoteSecurityGroupUUID string `json:"remoteSecurityGroupUuid"`
	ZStackTime
}

type SSecurityGroupRuleSet []SSecurityGroupRule

func (v SSecurityGroupRuleSet) Len() int {
	return len(v)
}

func (v SSecurityGroupRuleSet) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func (v SSecurityGroupRuleSet) Less(i, j int) bool {
	rule, err := v[i].toRule()
	if err != nil {
		return false
	}
	_rule, err := v[j].toRule()
	if err != nil {
		return false
	}
	return strings.Compare(rule.String(), _rule.String()) <= 0
}

type SSecurityGroup struct {
	region *SRegion

	ZStackBasic
	State     string `json:"state"`
	IPVersion int    `json:"ipVersion"`
	ZStackTime
	InternalID             int                  `json:"internalId"`
	Rules                  []SSecurityGroupRule `json:"rules"`
	AttachedL3NetworkUUIDs []string             `json:"attachedL3NetworkUuids"`
}

func (region *SRegion) GetSecurityGroup(secgroupId string) (*SSecurityGroup, error) {
	secgroup := &SSecurityGroup{region: region}
	return secgroup, region.client.getResource("security-groups", secgroupId, secgroup)
}

func (region *SRegion) GetSecurityGroups(secgroupId string, instanceId string, name string) ([]SSecurityGroup, error) {
	secgroups := []SSecurityGroup{}
	params := []string{}
	if len(secgroupId) > 0 {
		params = append(params, "q=uuid="+secgroupId)
	}
	if len(instanceId) > 0 {
		params = append(params, "q=vmNic.vmInstanceUuid="+instanceId)
	}
	if len(name) > 0 {
		params = append(params, "q=name="+name)
	}
	err := region.client.listAll("security-groups", params, &secgroups)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(secgroups); i++ {
		secgroups[i].region = region
	}
	return secgroups, nil
}

func (self *SSecurityGroup) GetVpcId() string {
	return api.NORMAL_VPC_ID
}

func (self *SSecurityGroup) GetMetadata() *jsonutils.JSONDict {
	data := jsonutils.NewDict()
	return data
}

func (self *SSecurityGroup) GetId() string {
	return self.UUID
}

func (self *SSecurityGroup) GetGlobalId() string {
	return self.UUID
}

func (self *SSecurityGroup) GetDescription() string {
	return self.Description
}

func (rule *SSecurityGroupRule) toRule() (*secrules.SecurityRule, error) {
	r := &secrules.SecurityRule{
		Direction: secrules.DIR_IN,
		Action:    secrules.SecurityRuleAllow,
		Priority:  1,
		Protocol:  secrules.PROTO_ANY,
		PortStart: rule.StartPort,
		PortEnd:   rule.EndPort,
	}
	_, ipNet, err := net.ParseCIDR(rule.AllowedCIDR)
	if err != nil {
		return nil, err
	}
	r.IPNet = ipNet
	if rule.Type == "Egress" {
		r.Direction = secrules.DIR_OUT
	}
	if rule.Protocol != "ALL" {
		r.Protocol = strings.ToLower(rule.Protocol)
	}
	return r, nil
}

func (self *SSecurityGroup) GetRules() ([]secrules.SecurityRule, error) {
	rules := []secrules.SecurityRule{}
	priority := 100
	outRuleCount := 0
	for i := 0; i < len(self.Rules); i++ {
		if self.Rules[i].IPVersion == 4 {
			rule, err := self.Rules[i].toRule()
			if err != nil {
				return nil, err
			}
			if rule.Direction == secrules.DIR_OUT {
				outRuleCount++
			}
			rule.Priority = priority
			rules = append(rules, *rule)
			priority--
		}
	}
	if outRuleCount != 0 {
		rule := secrules.MustParseSecurityRule("out:deny any")
		rule.Priority = 1
		rules = append(rules, *rule)
	}
	return rules, nil
}

func (self *SSecurityGroup) GetName() string {
	return self.Name
}

func (self *SSecurityGroup) GetStatus() string {
	return ""
}

func (self *SSecurityGroup) IsEmulated() bool {
	return false
}

func (self *SSecurityGroup) Refresh() error {
	new, err := self.region.GetSecurityGroup(self.UUID)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SSecurityGroup) GetProjectId() string {
	return ""
}

func (region *SRegion) AddSecurityGroupRule(secgroupId string, rules []secrules.SecurityRule) error {
	ruleParam := []map[string]interface{}{}
	for _, rule := range rules {
		Type := "Ingress"
		if rule.Direction == secrules.DIR_OUT {
			Type = "Egress"
		}
		protocol := "ALL"
		if rule.Protocol != secrules.PROTO_ANY {
			protocol = strings.ToUpper(rule.Protocol)
		}
		if rule.Protocol == secrules.PROTO_ICMP || rule.Protocol == secrules.PROTO_ANY {
			rule.PortStart = -1
			rule.PortEnd = -1
			rule.Ports = []int{}
		}
		if len(rule.Ports) > 0 {
			for _, port := range rule.Ports {
				ruleParam = append(ruleParam, map[string]interface{}{
					"type":        Type,
					"startPort":   port,
					"endPort":     port,
					"protocol":    protocol,
					"allowedCidr": rule.IPNet.String(),
				})
			}
		} else {
			if protocol != "ALL" {
				// TCP UDP端口不能为-1
				if rule.Protocol == secrules.PROTO_TCP || rule.Protocol == secrules.PROTO_UDP &&
					(rule.PortStart <= 0 && rule.PortEnd <= 0) {
					rule.PortStart = 0
					rule.PortEnd = 65535
				}
				ruleParam = append(ruleParam, map[string]interface{}{
					"type":        Type,
					"startPort":   rule.PortStart,
					"endPort":     rule.PortEnd,
					"protocol":    protocol,
					"allowedCidr": rule.IPNet.String(),
				})
			} else {
				ruleParam = append(ruleParam, map[string]interface{}{
					"type":        Type,
					"protocol":    protocol,
					"allowedCidr": rule.IPNet.String(),
				})
			}
		}
	}
	if len(ruleParam) > 0 {
		params := map[string]interface{}{
			"params": map[string]interface{}{
				"rules": ruleParam,
			},
		}
		return region.client.create(fmt.Sprintf("security-groups/%s/rules", secgroupId), jsonutils.Marshal(params), nil)
	}
	return nil
}

func (region *SRegion) DeleteSecurityGroupRules(ruleIds []string) error {
	if len(ruleIds) > 0 {
		ids := []string{}
		for _, ruleId := range ruleIds {
			ids = append(ids, fmt.Sprintf("ruleUuids=%s", ruleId))
		}
		resource := fmt.Sprintf("security-groups/rules?%s", strings.Join(ids, "&"))
		return region.client.delete(resource, "", "")
	}
	return nil
}

func (region *SRegion) CreateSecurityGroup(name, desc string) (*SSecurityGroup, error) {
	secgroup := &SSecurityGroup{region: region}
	params := map[string]map[string]string{
		"params": {
			"name":        name,
			"description": desc,
		},
	}
	return secgroup, region.client.create("security-groups", jsonutils.Marshal(params), secgroup)
}

func (region *SRegion) syncSecgroupRules(secgroupId string, rules []secrules.SecurityRule) error {
	secgroup, err := region.GetSecurityGroup(secgroupId)
	if err != nil {
		return err
	}

	inRules, outRules := secrules.SecurityRuleSet{}, secrules.SecurityRuleSet{}
	for i := 0; i < len(rules); i++ {
		if rules[i].Direction == secrules.DIR_IN {
			inRules = append(inRules, rules[i])
		} else {
			outRules = append(outRules, rules[i])
		}
	}

	if len(outRules) > 0 {
		// 避免出现 {"error":{"class":"SYS.1007","code":503,"details":"rule should not be duplicated. rule dump: {\"type\":\"Egress\",\"ipVersion\":4,\"startPort\":-1,\"endPort\":-1,\"protocol\":\"ALL\",\"allowedCidr\":\"0.0.0.0/0\"}"}}
		find := false
		for _, _rule := range outRules {
			if _rule.String() == "out:allow any" {
				find = true
				break
			}
		}
		if !find {
			rule := secrules.MustParseSecurityRule("out:allow any")
			outRules = append(outRules, *rule)
		}
	}

	rules = inRules.AllowList()
	rules = append(rules, outRules.AllowList()...)
	for i := 0; i < len(rules); i++ {
		rules[i].Priority = 1
	}

	sort.Sort(secrules.SecurityRuleSet(rules))
	sort.Sort(SSecurityGroupRuleSet(secgroup.Rules))

	delRuleIds := []string{}
	addRules := []secrules.SecurityRule{}

	i, j := 0, 0
	for i < len(rules) || j < len(secgroup.Rules) {
		if i < len(rules) && j < len(secgroup.Rules) {
			_rule, err := secgroup.Rules[j].toRule()
			if err != nil {
				return err
			}
			_ruleStr := _rule.String()
			ruleStr := rules[i].String()
			cmp := strings.Compare(_ruleStr, ruleStr)
			if cmp == 0 {
				if len(secgroup.Rules[j].RemoteSecurityGroupUUID) > 0 {
					delRuleIds = append(delRuleIds, secgroup.Rules[j].UUID)
					addRules = append(addRules, rules[i])
				}
				i++
				j++
			} else if cmp > 0 {
				delRuleIds = append(delRuleIds, secgroup.Rules[j].UUID)
				j++
			} else {
				addRules = append(addRules, rules[i])
				i++
			}
		} else if i >= len(rules) {
			delRuleIds = append(delRuleIds, secgroup.Rules[j].UUID)
			j++
		} else if j >= len(secgroup.Rules) {
			addRules = append(addRules, rules[i])
			i++
		}
	}
	if len(delRuleIds) > 0 {
		err = region.DeleteSecurityGroupRules(delRuleIds)
		if err != nil {
			return err
		}
	}
	return region.AddSecurityGroupRule(secgroupId, addRules)
}

func (self *SSecurityGroup) SyncRules(rules []secrules.SecurityRule) error {
	return self.region.syncSecgroupRules(self.UUID, rules)
}

func (self *SSecurityGroup) Delete() error {
	return self.region.client.delete("security-groups", self.UUID, "Permissive")
}
