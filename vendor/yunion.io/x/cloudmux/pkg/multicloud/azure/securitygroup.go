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
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
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

type Interface struct {
	ID string
}

type SecurityGroupPropertiesFormat struct {
	SecurityRules        []SecurityRules `json:"securityRules,omitempty"`
	DefaultSecurityRules []SecurityRules `json:"defaultSecurityRules,omitempty"`
	NetworkInterfaces    *[]Interface    `json:"networkInterfaces,omitempty"`
	Subnets              *[]SNetwork     `json:"subnets,omitempty"`
	ProvisioningState    string          //Possible values are: 'Updating', 'Deleting', and 'Failed'
}
type SSecurityGroup struct {
	multicloud.SSecurityGroup
	AzureTags

	region     *SRegion
	Properties *SecurityGroupPropertiesFormat `json:"properties,omitempty"`
	ID         string
	Name       string
	Location   string
	Type       string
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

func (self *SecurityRulePropertiesFormat) toRules() ([]cloudprovider.SecurityRule, error) {
	result := []cloudprovider.SecurityRule{}
	rule := cloudprovider.SecurityRule{
		SecurityRule: secrules.SecurityRule{
			Action:      secrules.TSecurityRuleAction(strings.ToLower(self.Access)),
			Direction:   secrules.TSecurityRuleDirection(strings.Replace(strings.ToLower(self.Direction), "bound", "", -1)),
			Protocol:    strings.ToLower(self.Protocol),
			Priority:    int(self.Priority),
			Description: self.Description,
		}}

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
		for j := 0; j < len(ports); j++ {
			rule.Ports = ports[j].ports
			rule.PortStart = ports[j].portStart
			rule.PortEnd = ports[j].portEnd
			err := rule.ValidateRule()
			if err != nil && err != secrules.ErrInvalidPriority {
				return nil, err
			}
			result = append(result, rule)
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

func (self *SSecurityGroup) GetRules() ([]cloudprovider.SecurityRule, error) {
	rules := make([]cloudprovider.SecurityRule, 0)
	if self.Properties == nil || self.Properties.SecurityRules == nil {
		return rules, nil
	}
	for _, _rule := range self.Properties.SecurityRules {
		secRules, err := _rule.Properties.toRules()
		if err != nil {
			return nil, errors.Wrap(err, "_rule.Properties.toRules")
		}
		for i := range secRules {
			secRules[i].Name = _rule.Name
			rules = append(rules, secRules[i])
		}
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
	secgroup := &SSecurityGroup{region: region}
	params := map[string]string{
		"Name":     secName,
		"Type":     "Microsoft.Network/networkSecurityGroups",
		"Location": region.Name,
	}
	return secgroup, region.create("", jsonutils.Marshal(params), secgroup)
}

func (region *SRegion) ListSecgroups() ([]SSecurityGroup, error) {
	secgroups := []SSecurityGroup{}
	err := region.list("Microsoft.Network/networkSecurityGroups", url.Values{}, &secgroups)
	if err != nil {
		return nil, errors.Wrapf(err, "list")
	}
	return secgroups, nil
}

func (region *SRegion) GetSecurityGroupDetails(secgroupId string) (*SSecurityGroup, error) {
	secgroup := SSecurityGroup{region: region}
	return &secgroup, region.get(secgroupId, url.Values{}, &secgroup)
}

func (self *SSecurityGroup) Refresh() error {
	sec, err := self.region.GetSecurityGroupDetails(self.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, sec)
}

func convertRulePort(rule cloudprovider.SecurityRule) []string {
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

func convertSecurityGroupRule(rule cloudprovider.SecurityRule) *SecurityRules {
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
	destRule := SecurityRules{
		Name: rule.Name,
		Properties: SecurityRulePropertiesFormat{
			Access:                   utils.Capitalize(string(rule.Action)),
			Priority:                 int32(rule.Priority),
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

func (region *SRegion) AttachSecurityToInterfaces(secgroupId string, nicIds []string) error {
	for _, nicId := range nicIds {
		nic, err := region.GetNetworkInterface(nicId)
		if err != nil {
			return err
		}
		nic.Properties.NetworkSecurityGroup = SSecurityGroup{ID: secgroupId}
		if err := region.update(jsonutils.Marshal(nic), nil); err != nil {
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

func (self *SSecurityGroup) Delete() error {
	if self.Properties != nil && self.Properties.NetworkInterfaces != nil {
		for _, nic := range *self.Properties.NetworkInterfaces {
			nic, err := self.region.GetNetworkInterface(nic.ID)
			if err != nil {
				return err
			}
			if err := self.region.update(jsonutils.Marshal(nic), nil); err != nil {
				return err
			}
		}
	}
	return self.region.del(self.ID)
}

func (self *SSecurityGroup) SetRules(rules []cloudprovider.SecurityRule) error {
	names := []string{}
	securityRules := []SecurityRules{}
	for i := 0; i < len(rules); i++ {
		rule := convertSecurityGroupRule(rules[i])
		if rule != nil {
			for {
				if !utils.IsInStringArray(rule.Name, names) {
					names = append(names, rule.Name)
					break
				}
				rule.Name = fmt.Sprintf("%s_", rule.Name)
			}
			securityRules = append(securityRules, *rule)
		}
	}
	if self.Properties == nil {
		self.Properties = &SecurityGroupPropertiesFormat{}
	}
	self.Properties.SecurityRules = securityRules
	self.Properties.ProvisioningState = ""
	return self.region.update(jsonutils.Marshal(self), nil)
}

func (self *SSecurityGroup) SyncRules(common, inAdds, outAdds, inDels, outDels []cloudprovider.SecurityRule) error {
	for i := range common {
		switch common[i].Direction {
		case secrules.DIR_IN:
			inAdds = append(inAdds, common[i])
		case secrules.DIR_OUT:
			outAdds = append(outAdds, common[i])
		default:
			return fmt.Errorf("invalid rule %s direction %s", common[i].String(), common[i].Direction)
		}
	}
	// Azure 不允许同方向的规则优先级相同
	inRules := cloudprovider.SortUniqPriority(inAdds)
	outRules := cloudprovider.SortUniqPriority(outAdds)
	rules := append(inRules, outRules...)
	return self.SetRules(rules)
}
