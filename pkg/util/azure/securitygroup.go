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
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/utils"
)

type SecurityRulePropertiesFormat struct {
	Description                string   `json:"description,omitempty"`
	Protocol                   string   `json:"protocol,omitempty"`
	SourcePortRange            string   `json:"sourcePortRange,omitempty"`
	DestinationPortRange       string   `json:"destinationPortRange,omitempty"`
	SourceAddressPrefix        string   `json:"sourceAddressPrefix,omitempty"`
	SourceAddressPrefixes      []string `json:"sourceAddressPrefixes,omitempty"`
	DestinationAddressPrefix   string   `json:"destinationAddressPrefix,omitempty"`
	DestinationAddressPrefixes []string `json:"destinationAddressPrefixes,omitempty"`
	SourcePortRanges           []string `json:"sourcePortRanges,omitempty"`
	DestinationPortRanges      []string `json:"destinationPortRanges,omitempty"`
	Access                     string   `json:"access,omitempty"` // Allow or Deny
	Priority                   int32    `json:"priority,omitempty"`
	Direction                  string   `json:"direction,omitempty"` //Inbound or Outbound
	ProvisioningState          string   `json:"-"`
}
type SecurityRules struct {
	Properties SecurityRulePropertiesFormat
	Name       string
	ID         string
}

type SecurityRulesSet []SecurityRules

func (v SecurityRulesSet) Len() int {
	return len(v)
}

func (v SecurityRulesSet) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func (v SecurityRulesSet) Less(i, j int) bool {
	if v[i].Properties.Priority < v[j].Properties.Priority {
		return true
	} else if v[i].Properties.Priority == v[j].Properties.Priority {
		return strings.Compare(v[i].Properties.String(), v[j].Properties.String()) <= 0
	}
	return false
}

type Interface struct {
	ID string
}

type SecurityGroupPropertiesFormat struct {
	SecurityRules        []SecurityRules `json:"securityRules,omitempty"`
	DefaultSecurityRules []SecurityRules `json:"defaultSecurityRules,omitempty"`
	NetworkInterfaces    *[]Interface    `json:"networkInterfaces,omitempty"`
	Subnets              *[]Subnet       `json:"subnets,omitempty"`
	ProvisioningState    string          //Possible values are: 'Updating', 'Deleting', and 'Failed'
}
type SSecurityGroup struct {
	vpc        *SVpc
	Properties *SecurityGroupPropertiesFormat `json:"properties,omitempty"`
	ID         string
	Name       string
	Location   string
	Type       string
	Tags       map[string]string
}

func (self *SSecurityGroup) GetMetadata() *jsonutils.JSONDict {
	if len(self.Tags) == 0 {
		return nil
	}
	data := jsonutils.Marshal(self.Tags).(*jsonutils.JSONDict)
	return data
}

func parseCIDR(cidr string) (*net.IPNet, error) {
	if cidr == "*" || strings.ToLower(cidr) == "internet" {
		cidr = "0.0.0.0/0"
	}
	if strings.Index(cidr, "/") > 0 {
		_, ipnet, err := net.ParseCIDR(cidr)
		return ipnet, err
	}
	ip := net.ParseIP(cidr)
	if ip == nil {
		return nil, fmt.Errorf("Parse ip %s error", cidr)
	}
	return &net.IPNet{IP: ip, Mask: net.CIDRMask(32, 32)}, nil
}

type rulePorts struct {
	ports     []int
	portStart int
	portEnd   int
}

func parsePorts(ports string) (rulePorts, error) {
	result := rulePorts{
		portStart: -1,
		portEnd:   -1,
		ports:     []int{},
	}
	if ports == "*" {
		return result, nil
	} else if strings.Index(ports, ",") > 0 {
		for _, _port := range strings.Split(ports, ",") {
			port, err := strconv.Atoi(_port)
			if err != nil {
				msg := fmt.Sprintf("parse rule port %s error: %v", ports, err)
				log.Errorf(msg)
				return result, fmt.Errorf(msg)
			}
			result.ports = append(result.ports, port)
		}
	} else if strings.Index(ports, "-") > 0 {
		_ports := strings.Split(ports, "-")
		if len(_ports) == 2 {
			portStart, err := strconv.Atoi(_ports[0])
			if err != nil {
				msg := fmt.Sprintf("parse rule port %s error: %v", ports, err)
				log.Errorf(msg)
				return result, fmt.Errorf(msg)
			}
			result.portStart = portStart
			portEnd, err := strconv.Atoi(_ports[1])
			if err != nil {
				msg := fmt.Sprintf("parse rule port %s error: %v", ports, err)
				log.Errorf(msg)
				return result, fmt.Errorf(msg)
			}
			result.portEnd = portEnd
		}
	} else {
		_port, err := strconv.Atoi(ports)
		if err != nil {
			msg := fmt.Sprintf("parse rule port %s error: %v", ports, err)
			log.Errorf(msg)
			return result, fmt.Errorf(msg)
		}
		result.ports = append(result.ports, _port)
	}
	return result, nil
}

func paresPortsWithIpNet(port string, ports []string, ip string, ips []string) ([]rulePorts, []*net.IPNet, error) {
	portsResult, ipResult := []rulePorts{}, []*net.IPNet{}
	if len(port) > 0 {
		_ports, err := parsePorts(port)
		if err != nil {
			return nil, nil, err
		}
		portsResult = append(portsResult, _ports)
	} else if len(ports) > 0 {
		for i := 0; i < len(ports); i++ {
			_ports, err := parsePorts(ports[i])
			if err != nil {
				return nil, nil, err
			}
			portsResult = append(portsResult, _ports)
		}
	}

	if len(ip) > 0 {
		ipnet, err := parseCIDR(ip)
		if err != nil {
			return nil, nil, err
		}
		ipResult = append(ipResult, ipnet)
	} else if len(ips) > 0 {
		for i := 0; i < len(ips); i++ {
			ipnet, err := parseCIDR(ips[i])
			if err != nil {
				return nil, nil, err
			}
			ipResult = append(ipResult, ipnet)
		}
	}
	return portsResult, ipResult, nil
}

func (self *SecurityRulePropertiesFormat) toRules() ([]secrules.SecurityRule, error) {
	result := []secrules.SecurityRule{}
	rule := secrules.SecurityRule{
		Action:      secrules.TSecurityRuleAction(strings.ToLower(self.Access)),
		Direction:   secrules.TSecurityRuleDirection(strings.Replace(strings.ToLower(self.Direction), "bound", "", -1)),
		Protocol:    strings.ToLower(self.Protocol),
		Priority:    int(self.Priority),
		Description: self.Description,
	}

	if rule.Protocol == "*" {
		rule.Protocol = "any"
	}

	addressPrefix, addressPrefixes := "", []string{}
	if rule.Direction == secrules.DIR_IN {
		addressPrefix, addressPrefixes = self.SourceAddressPrefix, self.SourceAddressPrefixes
	} else {
		addressPrefix, addressPrefixes = self.DestinationAddressPrefix, self.DestinationAddressPrefixes
	}

	if strings.ToLower(addressPrefix) == "internet" || addressPrefix == "*" {
		addressPrefix = "0.0.0.0/0"
	}

	if !regutils.MatchIPAddr(addressPrefix) && !regutils.MatchCIDR(addressPrefix) && len(addressPrefixes) == 0 {
		return nil, nil
	}

	ports, ips, err := paresPortsWithIpNet(self.DestinationPortRange, self.DestinationPortRanges, addressPrefix, addressPrefixes)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(ips); i++ {
		rule.IPNet = ips[i]
		withICMP := false
		for j := 0; j < len(ports); j++ {
			rule.Ports = ports[j].ports
			rule.PortStart = ports[j].portStart
			rule.PortEnd = ports[j].portEnd
			if rule.Protocol == secrules.PROTO_ANY && len(rule.Ports) > 0 || (rule.PortStart+rule.PortStart > 0) {
				tcp := rule
				tcp.Protocol = secrules.PROTO_TCP
				err := tcp.ValidateRule()
				if err != nil {
					return nil, err
				}
				result = append(result, tcp)

				udp := rule
				udp.Protocol = secrules.PROTO_UDP
				err = udp.ValidateRule()
				if err != nil {
					return nil, err
				}
				result = append(result, udp)
				withICMP = true
			} else {
				err := rule.ValidateRule()
				if err != nil {
					return nil, err
				}
				result = append(result, rule)
			}
		}
		if withICMP {
			icmp := rule
			icmp.Protocol = secrules.PROTO_ICMP
			icmp.PortStart = -1
			icmp.PortEnd = -1
			icmp.Ports = []int{}
			err := icmp.ValidateRule()
			if err != nil {
				return nil, err
			}
			result = append(result, icmp)
		}
	}

	return result, nil
}

func (self *SecurityRulePropertiesFormat) String() string {
	rules, err := self.toRules()
	if err != nil {
		log.Errorf("convert secrules error: %v", err)
		return ""
	}
	result := []string{}
	for i := 0; i < len(rules); i++ {
		result = append(result, rules[i].String())
	}
	return strings.Join(result, ";")
}

func (self *SSecurityGroup) GetId() string {
	return self.ID
}

func (self *SSecurityGroup) GetGlobalId() string {
	return strings.ToLower(self.ID)
}

func (self *SSecurityGroup) GetDescription() string {
	return ""
}

func (self *SSecurityGroup) GetName() string {
	return self.Name
}

func (self *SSecurityGroup) GetRules() ([]secrules.SecurityRule, error) {
	rules := make([]secrules.SecurityRule, 0)
	if self.Properties.SecurityRules == nil {
		return rules, nil
	}
	sort.Sort(SecurityRulesSet(self.Properties.SecurityRules))
	priority := 100
	for _, _rule := range self.Properties.SecurityRules {
		_rule.Properties.Priority = int32(priority)
		secRules, err := _rule.Properties.toRules()
		if err != nil {
			log.Errorf("Azure convert rule %v error: %v", _rule, err)
			return nil, err
		}
		if len(secRules) > 0 {
			priority--
		}
		rules = append(rules, secRules...)
	}
	return rules, nil
}

func (self *SSecurityGroup) GetStatus() string {
	return ""
}

func (self *SSecurityGroup) IsEmulated() bool {
	return false
}

func (self *SSecurityGroup) GetVpcId() string {
	return "normal"
}

func (region *SRegion) CreateSecurityGroup(secName string) (*SSecurityGroup, error) {
	if secName == "Default" {
		secName = "Default-copy"
	}
	secgroup := SSecurityGroup{
		Name:     secName,
		Type:     "Microsoft.Network/networkSecurityGroups",
		Location: region.Name,
	}
	return &secgroup, region.client.Create(jsonutils.Marshal(secgroup), &secgroup)
}

func (region *SRegion) GetSecurityGroups() ([]SSecurityGroup, error) {
	secgroups := []SSecurityGroup{}
	err := region.client.ListAll("Microsoft.Network/networkSecurityGroups", &secgroups)
	if err != nil {
		return nil, err
	}
	result := []SSecurityGroup{}
	for i := 0; i < len(secgroups); i++ {
		if secgroups[i].Location == region.Name {
			result = append(result, secgroups[i])
		}
	}
	return result, err
}

func (region *SRegion) GetSecurityGroupDetails(secgroupId string) (*SSecurityGroup, error) {
	secgroup := SSecurityGroup{}
	return &secgroup, region.client.Get(secgroupId, []string{}, &secgroup)
}

func (self *SSecurityGroup) Refresh() error {
	sec, err := self.vpc.region.GetSecurityGroupDetails(self.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, sec)
}

func convertRulePort(rule secrules.SecurityRule) []string {
	ports := []string{}
	if len(rule.Ports) > 0 {
		for i := 0; i < len(rule.Ports); i++ {
			ports = append(ports, fmt.Sprintf("%d", rule.Ports[i]))
		}
		return ports
	}
	if rule.PortStart > 0 && rule.PortEnd < 65535 {
		if rule.PortStart == rule.PortEnd {
			return []string{fmt.Sprintf("%d", rule.PortStart)}
		}
		ports = append(ports, fmt.Sprintf("%d-%d", rule.PortStart, rule.PortEnd))
	}
	return ports
}

func convertSecurityGroupRule(rule secrules.SecurityRule, priority int32) *SecurityRules {
	name := strings.Replace(rule.String(), ":", "_", -1)
	name = strings.Replace(name, " ", "_", -1)
	name = strings.Replace(name, "-", "_", -1)
	name = strings.Replace(name, "/", "_", -1)
	name = strings.Replace(name, ",", "_", -1)
	name = fmt.Sprintf("%s_%d", name, rule.Priority)
	destRule := SecurityRules{
		Name: name,
		Properties: SecurityRulePropertiesFormat{
			Access:                   utils.Capitalize(string(rule.Action)),
			Priority:                 priority,
			Protocol:                 "*",
			Direction:                utils.Capitalize((string(rule.Direction) + "bound")),
			Description:              rule.Description,
			DestinationAddressPrefix: "*",
			DestinationPortRanges:    convertRulePort(rule),
			SourcePortRange:          "*",
			SourceAddressPrefix:      "*",
			DestinationPortRange:     "*",
		},
	}
	if rule.Protocol != secrules.PROTO_ANY {
		destRule.Properties.Protocol = utils.Capitalize(rule.Protocol)
	}

	if rule.Protocol == secrules.PROTO_ICMP {
		return nil
	}

	if len(destRule.Properties.DestinationPortRanges) > 0 {
		destRule.Properties.DestinationPortRange = ""
	}

	ipAddr := "*"
	if rule.IPNet != nil {
		ipAddr = rule.IPNet.String()
	}
	if rule.Direction == secrules.DIR_IN {
		destRule.Properties.SourceAddressPrefix = ipAddr
	} else {
		destRule.Properties.DestinationAddressPrefix = ipAddr
	}
	return &destRule
}

func (region *SRegion) updateSecurityGroupRules(secgroupId string, rules []secrules.SecurityRule) (string, error) {
	secgroup, err := region.GetSecurityGroupDetails(secgroupId)
	if err != nil {
		return "", err
	}
	sort.Sort(secrules.SecurityRuleSet(rules))
	securityRules := []SecurityRules{}
	priority := int32(100)
	ruleStrs := []string{}
	for i := 0; i < len(rules); i++ {
		ruleStr := rules[i].String()
		if !utils.IsInStringArray(ruleStr, ruleStrs) {
			rule := convertSecurityGroupRule(rules[i], priority)
			if rule != nil {
				securityRules = append(securityRules, *rule)
				priority++
			}
			ruleStrs = append(ruleStrs, ruleStr)
		}
	}
	secgroup.Properties.SecurityRules = securityRules
	secgroup.Properties.ProvisioningState = ""
	return secgroup.ID, region.client.Update(jsonutils.Marshal(secgroup), nil)
}

func (region *SRegion) AttachSecurityToInterfaces(secgroupId string, nicIds []string) error {
	for _, nicId := range nicIds {
		nic, err := region.GetNetworkInterfaceDetail(nicId)
		if err != nil {
			return err
		}
		nic.Properties.NetworkSecurityGroup = &SSecurityGroup{ID: secgroupId}
		if err := region.client.Update(jsonutils.Marshal(nic), nil); err != nil {
			return err
		}
	}
	return nil
}

func (region *SRegion) SetSecurityGroup(instanceId, secgroupId string) error {
	instance, err := region.GetInstance(instanceId)
	if err != nil {
		return err
	}
	nicIds := []string{}
	for _, nic := range instance.Properties.NetworkProfile.NetworkInterfaces {
		nicIds = append(nicIds, nic.ID)
	}
	return region.AttachSecurityToInterfaces(secgroupId, nicIds)
}

func (self *SSecurityGroup) GetProjectId() string {
	return getResourceGroup(self.ID)
}
