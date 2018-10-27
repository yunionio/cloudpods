package azure

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/utils"
)

type SClassicSecurityGroup struct {
	vpc        *SClassicVpc
	Properties *SecurityGroupPropertiesFormat `json:"properties,omitempty"`
	ID         string
	Name       string
	Location   string
	Type       string
	Tags       map[string]string
}

type ClassicSecurityGroupRuleProperties struct {
	State                    string
	Protocol                 string
	SourcePortRange          string
	DestinationPortRange     string
	SourceAddressPrefix      string
	DestinationAddressPrefix string
	Action                   string
	Priority                 uint32
	Type                     string
	IsDefault                bool
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
	rule := secrules.SecurityRule{}
	rule.Action = secrules.TSecurityRuleAction(strings.ToLower(self.Action))
	rule.Direction = secrules.TSecurityRuleDirection(strings.Replace(strings.ToLower(self.Type), "bound", "", -1))
	rule.Protocol = strings.ToLower(self.Protocol)
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
		rule.Protocol = "tcp"
		rules = append(rules, rule)
		rule.Protocol = "udp"
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
	secgrouprules := []SClassicSecurityGroupRule{}
	body, err := self.vpc.region.client.jsonRequest("GET", fmt.Sprintf("%s/securityRules", self.ID), "")
	if err != nil {
		return nil, err
	}
	err = body.Unmarshal(&secgrouprules, "value")
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

func (region *SRegion) CreateClassicSecurityGroup(secName, tagId string) (*SClassicSecurityGroup, error) {
	return nil, cloudprovider.ErrNotImplemented
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
	secgroup := SClassicSecurityGroup{}
	return &secgroup, region.client.Get(secgroupId, []string{}, &secgroup)
}

func (self *SClassicSecurityGroup) Refresh() error {
	sec, err := self.vpc.region.GetClassicSecurityGroupDetails(self.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, sec)
}

func (region *SRegion) checkClassicSecurityGroup(tagId, name string) (*SClassicSecurityGroup, error) {
	secgroups, err := region.GetClassicSecurityGroups()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(secgroups); i++ {
		for k, v := range secgroups[i].Tags {
			if k == "id" && v == tagId {
				return &secgroups[i], nil
			}
		}
	}
	return region.CreateClassicSecurityGroup(name, tagId)
}

func (region *SRegion) updateClassicSecurityGroupRules(secgroupId string, rules []secrules.SecurityRule) (string, error) {
	secgroup, err := region.GetClassicSecurityGroupDetails(secgroupId)
	if err != nil {
		return "", err
	}
	securityRules := []SecurityRules{}
	priority := int32(100)
	for i := 0; i < len(rules); i++ {
		rules[i].Priority = int(priority)
		if rule := convertSecurityGroupRule(rules[i]); rule != nil {
			securityRules = append(securityRules, *rule)
			priority++
		}
	}
	secgroup.Properties.SecurityRules = &securityRules
	secgroup.Properties.ProvisioningState = ""
	return secgroup.ID, region.client.Update(jsonutils.Marshal(secgroup), nil)
}

func (region *SRegion) AssiginClassicSecurityGroup(instanceId, secgroupId string) error {
	instance, err := region.GetClassicInstance(instanceId)
	if err != nil {
		return err
	}
	secgroup, err := region.GetClassicSecurityGroupDetails(secgroupId)
	if err != nil {
		return err
	}
	instance.Properties.NetworkProfile.NetworkSecurityGroup = &SubResource{
		ID:   secgroupId,
		Name: secgroup.Name,
		Type: secgroup.Type,
	}
	return region.client.Update(jsonutils.Marshal(instance), nil)
}

func (self *SRegion) syncClassicSecgroupRules(secgroupId string, rules []secrules.SecurityRule) (string, error) {
	secgroup, err := self.GetClassicSecurityGroupDetails(secgroupId)
	if err != nil {
		return "", err
	}
	sort.Sort(secrules.SecurityRuleSet(rules))
	sort.Sort(SecurityRulesSet(*secgroup.Properties.SecurityRules))

	newRules := []secrules.SecurityRule{}

	i, j := 0, 0
	for i < len(rules) || j < len(*secgroup.Properties.SecurityRules) {
		if i < len(rules) && j < len(*secgroup.Properties.SecurityRules) {
			srcRule := (*secgroup.Properties.SecurityRules)[j].Properties.String()
			destRule := rules[i].String()
			cmp := strings.Compare(srcRule, destRule)
			if cmp == 0 {
				// keep secRule
				newRules = append(newRules, rules[i])
				i++
				j++
			} else if cmp > 0 {
				// remove srcRule
				j++
			} else {
				// add destRule
				newRules = append(newRules, rules[i])
				i++
			}
		} else if i >= len(rules) {
			// del other rules
			j++
		} else if j >= len(*secgroup.Properties.SecurityRules) {
			// add rule
			newRules = append(newRules, rules[i])
			i++
		}
	}
	return self.updateClassicSecurityGroupRules(secgroup.ID, newRules)

}

func (self *SRegion) syncClassicSecurityGroup(tagId, name string, rules []secrules.SecurityRule) (string, error) {
	secgroup, err := self.checkClassicSecurityGroup(tagId, name)
	if err != nil {
		return "", err
	}
	return self.syncClassicSecgroupRules(secgroup.ID, rules)
}
