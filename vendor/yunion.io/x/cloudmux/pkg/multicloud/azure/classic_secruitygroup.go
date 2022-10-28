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
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SClassicSecurityGroup struct {
	multicloud.SSecurityGroup
	AzureTags
	region     *SRegion
	vpc        *SClassicVpc
	Properties ClassicSecurityGroupProperties `json:"properties,omitempty"`
	ID         string
	Name       string
	Location   string
	Type       string
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

func (self *ClassicSecurityGroupRuleProperties) toRule() *cloudprovider.SecurityRule {
	rule := cloudprovider.SecurityRule{
		SecurityRule: secrules.SecurityRule{
			Action:    secrules.TSecurityRuleAction(strings.ToLower(self.Action)),
			Direction: secrules.TSecurityRuleDirection(strings.Replace(strings.ToLower(self.Type), "bound", "", -1)),
			Protocol:  strings.ToLower(self.Protocol),
			Priority:  int(self.Priority),
		},
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
		return nil
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
	return &rule
}

func (self *SClassicSecurityGroup) GetVpcId() string {
	return "classic"
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

func (self *SClassicSecurityGroup) GetRules() ([]cloudprovider.SecurityRule, error) {
	rules := make([]cloudprovider.SecurityRule, 0)
	secgrouprules, err := self.region.getClassicSecurityGroupRules(self.ID)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(secgrouprules); i++ {
		if secgrouprules[i].Properties.Priority >= 65000 {
			continue
		}
		rule := secgrouprules[i].Properties.toRule()
		if rule == nil {
			continue
		}
		rule.Name = secgrouprules[i].Name
		rule.ExternalId = secgrouprules[i].ID
		if err := rule.ValidateRule(); err != nil && err != secrules.ErrInvalidPriority {
			return nil, errors.Wrap(err, "rule.ValidateRule")
		}
		rules = append(rules, *rule)
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
		region:   region,
		Name:     name,
		Type:     "Microsoft.ClassicNetwork/networkSecurityGroups",
		Location: region.Name,
	}
	return &secgroup, region.create("", jsonutils.Marshal(secgroup), &secgroup)
}

func (region *SRegion) GetClassicSecurityGroups(name string) ([]SClassicSecurityGroup, error) {
	secgroups := []SClassicSecurityGroup{}
	err := region.client.list("Microsoft.ClassicNetwork/networkSecurityGroups", url.Values{}, &secgroups)
	if err != nil {
		return nil, err
	}
	result := []SClassicSecurityGroup{}
	for i := 0; i < len(secgroups); i++ {
		if secgroups[i].Location == region.Name && (len(name) == 0 || strings.ToLower(secgroups[i].Name) == strings.ToLower(name)) {
			result = append(result, secgroups[i])
		}
	}
	return result, err
}

func (region *SRegion) GetClassicSecurityGroupDetails(secgroupId string) (*SClassicSecurityGroup, error) {
	secgroup := SClassicSecurityGroup{region: region}
	return &secgroup, region.get(secgroupId, url.Values{}, &secgroup)
}

func (region *SRegion) deleteClassicSecurityGroup(secgroupId string) error {
	return region.del(secgroupId)
}

func (self *SClassicSecurityGroup) Delete() error {
	return self.region.deleteClassicSecurityGroup(self.ID)
}

func (self *SClassicSecurityGroup) Refresh() error {
	sec, err := self.region.GetClassicSecurityGroupDetails(self.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, sec)
}

func convertClassicSecurityGroupRules(rule cloudprovider.SecurityRule) ([]SClassicSecurityGroupRule, error) {
	rules := []SClassicSecurityGroupRule{}
	if len(rule.Name) == 0 {
		rule.Name = fmt.Sprintf("%s_%d", rule.String(), rule.Priority)
	}
	rule.Name = func(name string) string {
		// 名称必须以字母或数字开头，以字母、数字或下划线结尾，并且只能包含字母、数字、下划线、句点或连字符
		for _, s := range name {
			if !(unicode.IsDigit(s) || unicode.IsLetter(s) || s == '.' || s == '-' || s == '_') {
				name = strings.ReplaceAll(name, string(s), "_")
			}
		}
		if !unicode.IsDigit(rune(name[0])) && !unicode.IsLetter(rune(name[0])) {
			name = fmt.Sprintf("r_%s", name)
		}
		last := len(name) - 1
		if !unicode.IsDigit(rune(name[last])) && !unicode.IsLetter(rune(name[last])) && name[last] != '_' {
			name = fmt.Sprintf("%s_", name)
		}
		return name
	}(rule.Name)
	secRule := SClassicSecurityGroupRule{
		Name: rule.Name,
		Properties: ClassicSecurityGroupRuleProperties{
			Action:                   utils.Capitalize(string(rule.Action)),
			Priority:                 int32(rule.Priority),
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
		return nil, fmt.Errorf("not support icmp protocol")
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
	params := url.Values{}
	params.Set("api-version", "2015-06-01")
	resource := fmt.Sprintf("%s/securityRules", secgroupId)
	err := self.client.list(resource, params, &rules)
	if err != nil {
		return nil, errors.Wrapf(err, "list")
	}
	return rules, nil
}

func (self *SRegion) addClassicSecgroupRule(secgroupId string, rule SClassicSecurityGroupRule) error {
	resource := fmt.Sprintf("%s/securityRules/%s", secgroupId, rule.Name)
	_, err := self.put(resource, jsonutils.Marshal(rule))
	return err
}

func (self *SClassicSecurityGroup) GetProjectId() string {
	return getResourceGroup(self.ID)
}

func (self *SClassicSecurityGroup) SyncRules(common, inAdds, outAdds, inDels, outDels []cloudprovider.SecurityRule) error {
	for _, r := range append(inDels, outDels...) {
		err := self.region.del(r.ExternalId)
		if err != nil {
			return errors.Wrapf(err, "Delete(%s)", r.ExternalId)
		}
		for _, r := range append(inAdds, outAdds...) {
			_rules, err := convertClassicSecurityGroupRules(r)
			if err != nil {
				return errors.Wrapf(err, "convertClassicSecurityGroupRules(%s)", r.String())
			}
			names := []string{}
			for _, r := range _rules {
				for {
					if !utils.IsInStringArray(r.Name, names) {
						names = append(names, r.Name)
						break
					}
					r.Name = fmt.Sprintf("%s_", r.Name)
				}
				err = self.region.addClassicSecgroupRule(self.ID, r)
				if err != nil {
					return errors.Wrap(err, "addClassicSecgroupRule")
				}
			}
		}
	}
	return nil
}
