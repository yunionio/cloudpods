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

package azure

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SClassicSecurityGroup struct {
	vpc        *SClassicVpc
	Properties ClassicSecurityGroupProperties `json:"properties,omitempty"`
	ID         string
	Name       string
	Location   string
	Type       string
	Tags       map[string]string
}

type ClassicSecurityGroupProperties struct {
	NetworkSecurityGroupId string `json:"networkSecurityGroupId,omitempty"`
	State                  string `json:"state,omitempty"`
}

type ClassicSecurityGroupRuleProperties struct {
	State                    string `json:"state,omitempty"`
	Protocol                 string `json:"protocol,omitempty"`
	SourcePortRange          string `json:"sourcePortRange,omitempty"`
	DestinationPortRange     string `json:"destinationPortRange,omitempty"`
	SourceAddressPrefix      string `json:"sourceAddressPrefix,omitempty"`
	DestinationAddressPrefix string `json:"destinationAddressPrefix,omitempty"`
	Action                   string `json:"action,omitempty"`
	Priority                 int32  `json:"priority,omitempty"`
	Type                     string `json:"type,omitempty"`
	IsDefault                bool   `json:"isDefault,omitempty"`
}

type SClassicSecurityGroupRule struct {
	Properties ClassicSecurityGroupRuleProperties `json:"properties,omitempty"`
	ID         string
	Name       string
	Type       string
}

type ClassicSecurityRulesSet []SClassicSecurityGroupRule

func (v ClassicSecurityRulesSet) Len() int {
	return len(v)
}

func (v ClassicSecurityRulesSet) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func (v ClassicSecurityRulesSet) Less(i, j int) bool {
	if v[i].Properties.Priority < v[j].Properties.Priority {
		return true
	} else if v[i].Properties.Priority == v[j].Properties.Priority {
		return strings.Compare(v[i].Properties.String(), v[j].Properties.String()) <= 0
	}
	return false
}

func (self *ClassicSecurityGroupRuleProperties) toRules() []secrules.SecurityRule {
	rules := []secrules.SecurityRule{}
	rule := secrules.SecurityRule{
		Action:    secrules.TSecurityRuleAction(strings.ToLower(self.Action)),
		Direction: secrules.TSecurityRuleDirection(strings.Replace(strings.ToLower(self.Type), "bound", "", -1)),
		Protocol:  strings.ToLower(self.Protocol),
	}
	if rule.Protocol == "*" {
		rule.Protocol = "any"
	}
	port := self.DestinationPortRange
	ip := self.DestinationAddressPrefix
	if rule.Direction == secrules.SecurityRuleEgress {
		port = self.SourcePortRange
		ip = self.SourceAddressPrefix
	}

	if utils.IsInStringArray(ip, []string{"INTERNET", "VIRTUAL_NETWORK", "AZURE_LOADBALANCER"}) {
		return rules
	}

	if ip == "*" {
		ip = "0.0.0.0/24"
	}
	if idx := strings.Index(ip, "/"); idx > -1 {
		_, ipnet, err := net.ParseCIDR(ip)
		if err == nil {
			rule.IPNet = ipnet
		}
	} else if _ip := net.ParseIP(ip); _ip != nil {
		rule.IPNet = &net.IPNet{
			IP:   _ip,
			Mask: net.CIDRMask(32, 32),
		}
	}
	ports := strings.Split(port, "-")
	if len(ports) == 2 {
		rule.PortStart, _ = strconv.Atoi(ports[0])
		rule.PortEnd, _ = strconv.Atoi(ports[1])
	} else if len(ports) == 1 {
		rule.PortStart, _ = strconv.Atoi(ports[0])
		rule.PortEnd, _ = strconv.Atoi(ports[0])
	}
	if rule.PortStart > 0 && rule.Protocol == "any" {
		rule.Protocol = secrules.PROTO_TCP
		rules = append(rules, rule)
		rule.Protocol = secrules.PROTO_UDP
		rules = append(rules, rule)
		rule.Protocol = secrules.PROTO_ICMP
		rules = append(rules, rule)
	}
	return rules
}

func (self *ClassicSecurityGroupRuleProperties) String() string {
	result := []string{}
	for _, rule := range self.toRules() {
		result = append(result, rule.String())
	}
	return strings.Join(result, ";")
}

func (self *SClassicSecurityGroup) GetVpcId() string {
	return "classic"
}

func (self *SClassicSecurityGroup) GetMetadata() *jsonutils.JSONDict {
	if len(self.Tags) == 0 {
		return nil
	}
	data := jsonutils.Marshal(self.Tags).(*jsonutils.JSONDict)
	return data
}

func (self *SClassicSecurityGroup) GetId() string {
	return self.ID
}

func (self *SClassicSecurityGroup) GetGlobalId() string {
	return strings.ToLower(self.ID)
}

func (self *SClassicSecurityGroup) GetDescription() string {
	return ""
}

func (self *SClassicSecurityGroup) GetName() string {
	return self.Name
}

func (self *SClassicSecurityGroup) GetRules() ([]secrules.SecurityRule, error) {
	rules := make([]secrules.SecurityRule, 0)
	secgrouprules, err := self.vpc.region.getClassicSecurityGroupRules(self.ID)
	if err != nil {
		return nil, err
	}
	sort.Sort(ClassicSecurityRulesSet(secgrouprules))
	priority := 100

	for i := 0; i < len(secgrouprules); i++ {
		if secgrouprules[i].Properties.Priority >= 65000 {
			continue
		}
		_rules := secgrouprules[i].Properties.toRules()
		for i := 0; i < len(_rules); i++ {
			rule := _rules[i]
			rule.Priority = priority
			rule.Description = secgrouprules[i].Name
			if err := rule.ValidateRule(); err != nil {
				log.Errorf("Azure classic secgroup get rules error: %v", err)
				return nil, err
			}
			rules = append(rules, rule)
		}
		if len(_rules) > 0 {
			priority--
		}
	}
	return rules, nil
}

func (self *SClassicSecurityGroup) GetStatus() string {
	return ""
}

func (self *SClassicSecurityGroup) IsEmulated() bool {
	return false
}

func (region *SRegion) CreateClassicSecurityGroup(name string) (*SClassicSecurityGroup, error) {
	if name == "Default" {
		name = "Default-copy"
	}
	secgroup := SClassicSecurityGroup{
		Name:     name,
		Type:     "Microsoft.ClassicNetwork/networkSecurityGroups",
		Location: region.Name,
	}
	return &secgroup, region.client.Create(jsonutils.Marshal(secgroup), &secgroup)
}

func (region *SRegion) GetClassicSecurityGroups() ([]SClassicSecurityGroup, error) {
	secgroups := []SClassicSecurityGroup{}
	err := region.client.ListAll("Microsoft.ClassicNetwork/networkSecurityGroups", &secgroups)
	if err != nil {
		return nil, err
	}
	result := []SClassicSecurityGroup{}
	for i := 0; i < len(secgroups); i++ {
		if secgroups[i].Location == region.Name {
			result = append(result, secgroups[i])
		}
	}
	return result, err
}

func (region *SRegion) GetClassicSecurityGroupDetails(secgroupId string) (*SClassicSecurityGroup, error) {
	secgroup := SClassicSecurityGroup{region: region}
	return &secgroup, region.client.Get(secgroupId, []string{}, &secgroup)
}

func (region *SRegion) deleteClassicSecurityGroup(secgroupId string) error {
	return region.client.Delete(secgroupId)
}

func (self *SClassicSecurityGroup) Refresh() error {
	sec, err := self.vpc.region.GetClassicSecurityGroupDetails(self.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, sec)
}

func convertClassicSecurityGroupRules(rule secrules.SecurityRule, priority int32) ([]SClassicSecurityGroupRule, error) {
	name := strings.Replace(rule.String(), ":", "_", -1)
	name = strings.Replace(name, " ", "_", -1)
	name = strings.Replace(name, "-", "_", -1)
	name = strings.Replace(name, "/", "_", -1)
	name = fmt.Sprintf("%s_%d", name, rule.Priority)
	rules := []SClassicSecurityGroupRule{}
	secRule := SClassicSecurityGroupRule{
		Name: name,
		Properties: ClassicSecurityGroupRuleProperties{
			Action:                   utils.Capitalize(string(rule.Action)),
			Priority:                 priority,
			Type:                     utils.Capitalize(string(rule.Direction)) + "bound",
			Protocol:                 utils.Capitalize(rule.Protocol),
			SourcePortRange:          "*",
			DestinationPortRange:     "*",
			SourceAddressPrefix:      "*",
			DestinationAddressPrefix: "*",
		},
	}
	if rule.Protocol == secrules.PROTO_ANY {
		secRule.Properties.Protocol = "*"
	}
	if rule.Protocol == secrules.PROTO_ICMP {
		return nil, nil
	}
	ipAddr := "*"
	if rule.IPNet != nil {
		ipAddr = rule.IPNet.String()
	}
	if rule.Direction == secrules.DIR_IN {
		secRule.Properties.SourceAddressPrefix = ipAddr
	} else {
		secRule.Properties.DestinationAddressPrefix = ipAddr
	}
	if len(rule.Ports) > 0 {
		for _, port := range rule.Ports {
			secRule.Properties.DestinationPortRange = fmt.Sprintf("%d", port)
			rules = append(rules, secRule)
		}
		return rules, nil
	} else if rule.PortStart > 0 && rule.PortEnd > 0 {
		secRule.Properties.DestinationPortRange = fmt.Sprintf("%d-%d", rule.PortStart, rule.PortEnd)
	}
	rules = append(rules, secRule)
	return rules, nil
}

func (self *SRegion) getClassicSecurityGroupRules(secgroupId string) ([]SClassicSecurityGroupRule, error) {
	rules := []SClassicSecurityGroupRule{}
	result, err := self.client.jsonRequest("GET", fmt.Sprintf("%s/securityRules?api-version=2015-06-01", secgroupId), "")
	if err != nil {
		return nil, err
	}
	return rules, result.Unmarshal(&rules, "value")
}

func (self *SRegion) syncClassicSecgroupRules(secgroupId string, rules []secrules.SecurityRule) (string, error) {
	secgrouprules, err := self.getClassicSecurityGroupRules(secgroupId)
	if err != nil {
		return "", err
	}
	for _, rule := range secgrouprules {
		if rule.Properties.Priority >= 65000 {
			continue
		}
		if err := self.client.Delete(rule.ID); err != nil {
			return "", err
		}
	}
	sort.Sort(secrules.SecurityRuleSet(rules))
	priority := int32(100)
	ruleStrs := []string{}
	for i, _rule := range rules {
		ruleStr := rules[i].String()
		if !utils.IsInStringArray(ruleStr, ruleStrs) {
			_rules, err := convertClassicSecurityGroupRules(_rule, priority)
			if err != nil {
				return "", err
			}
			priority++
			ruleStrs = append(ruleStrs, ruleStr)
			for _, rule := range _rules {
				if err := self.addClassicSecgroupRule(secgroupId, rule); err != nil {
					return "", err
				}
			}
		}
	}
	return secgroupId, nil
}

func (self *SRegion) addClassicSecgroupRule(secgroupId string, rule SClassicSecurityGroupRule) error {
	url := fmt.Sprintf("%s/securityRules/%s?api-version=2015-06-01", secgroupId, rule.Name)
	_, err := self.client.jsonRequest("PUT", url, jsonutils.Marshal(rule).String())
	return err
}

func (region *SRegion) syncClassicSecurityGroup(secgroupId, name, desc string, rules []secrules.SecurityRule) (string, error) {
	if len(secgroupId) > 0 {
		if _, err := region.GetClassicSecurityGroupDetails(secgroupId); err != nil {
			if err != cloudprovider.ErrNotFound {
				return "", err
			}
			secgroupId = ""
		}
	}

	if len(secgroupId) == 0 {
		secgroup, err := region.CreateClassicSecurityGroup(name)
		if err != nil {
			return "", err
		}
		secgroupId = secgroup.ID
	}
	return region.syncClassicSecgroupRules(secgroupId, rules)
}

func (self *SClassicSecurityGroup) GetProjectId() string {
	return getResourceGroup(self.ID)
}
