package azure

import (
	"fmt"
	"sort"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/secrules"
)

type SecurityRulePropertiesFormat struct {
	Description                string    `json:"description,omitempty"`
	Protocol                   string    `json:"protocol,omitempty"`
	SourcePortRange            string    `json:"sourcePortRange,omitempty"`
	DestinationPortRange       string    `json:"destinationPortRange,omitempty"`
	SourceAddressPrefix        string    `json:"sourceAddressPrefix,omitempty"`
	SourceAddressPrefixes      *[]string `json:"sourceAddressPrefixes,omitempty"`
	DestinationAddressPrefix   string    `json:"destinationAddressPrefix,omitempty"`
	DestinationAddressPrefixes *[]string `json:"destinationAddressPrefixes,omitempty"`
	SourcePortRanges           *[]string `json:"sourcePortRanges,omitempty"`
	DestinationPortRanges      *[]string `json:"destinationPortRanges,omitempty"`
	Access                     string    `json:"access,omitempty"` // Allow or Deny
	Priority                   int32     `json:"priority,omitempty"`
	Direction                  string    `json:"direction,omitempty"` //Inbound or Outbound
	ProvisioningState          string    `json:"-"`
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
	SecurityRules        *[]SecurityRules `json:"securityRules,omitempty"`
	DefaultSecurityRules *[]SecurityRules `json:"defaultSecurityRules,omitempty"`
	NetworkInterfaces    *[]Interface     `json:"networkInterfaces,omitempty"`
	Subnets              *[]Subnet        `json:"subnets,omitempty"`
	ProvisioningState    string           //Possible values are: 'Updating', 'Deleting', and 'Failed'
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

func (self *SecurityRulePropertiesFormat) String() string {
	//log.Debugf("serize rule: %s", jsonutils.Marshal(self).PrettyString())
	action := secrules.SecurityRuleDeny
	if self.Access == "Allow" {
		action = secrules.SecurityRuleAllow
	}
	direction := "in"
	if self.Direction == "Outbound" {
		direction = "out"
	}
	cidr := self.SourceAddressPrefix
	port := self.SourcePortRange
	if self.SourcePortRanges != nil && len(*self.SourcePortRanges) > 0 {
		port = strings.Join(*self.SourcePortRanges, ",")
	}
	if direction == "out" {
		cidr = self.DestinationAddressPrefix
		port = self.DestinationPortRange
		if self.DestinationPortRanges != nil && len(*self.DestinationPortRanges) > 0 {
			port = strings.Join(*self.DestinationPortRanges, ",")
		}
	}
	if cidr == "*" || cidr == "0.0.0.0/0" {
		cidr = ""
	}
	if port == "*" {
		port = ""
	}
	protocol := strings.ToLower(self.Protocol)
	if protocol == "*" {
		protocol = "any"
	}

	result := fmt.Sprintf("%s:%s", direction, string(action))
	if len(cidr) > 0 {
		result += fmt.Sprintf(" %s", cidr)
	}
	result += fmt.Sprintf(" %s", protocol)
	if len(port) > 0 {
		if strings.Index(port, ",") > 0 && strings.Index(port, "-") > 0 {
			results := make([]string, 0)
			for _, _port := range strings.Split(port, ",") {
				results = append(results, fmt.Sprintf("%s %s", result, _port))
			}
			result = strings.Join(results, ";")
		} else {
			result += fmt.Sprintf(" %s", port)
		}
	}
	return result
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
	sort.Sort(SecurityRulesSet(*self.Properties.SecurityRules))
	priority := 100

	for _, _rule := range *self.Properties.SecurityRules {
		for _, ruleString := range strings.Split(_rule.Properties.String(), ";") {
			if rule, err := secrules.ParseSecurityRule(ruleString); err != nil {
				return rules, err
			} else {
				rule.Priority = priority
				priority--
				rule.Description = _rule.Properties.Description
				rules = append(rules, *rule)
			}
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

func (region *SRegion) CreateSecurityGroup(secName string, tagId string) (*SSecurityGroup, error) {
	securityName := fmt.Sprintf("%s-%s", region.Name, secName)
	secgroup := SSecurityGroup{
		Name:     securityName,
		Type:     "Microsoft.Network/networkSecurityGroups",
		Location: region.Name,
		Tags:     map[string]string{"id": tagId},
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
	return &secgroup, region.client.Get(secgroupId, &secgroup)
}

func (self *SSecurityGroup) Refresh() error {
	sec, err := self.vpc.region.GetSecurityGroupDetails(self.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, sec)
}

func (region *SRegion) checkSecurityGroup(tagId, name string) (*SSecurityGroup, error) {
	secgroups, err := region.GetSecurityGroups()
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
	return region.CreateSecurityGroup(name, tagId)
}

func convertRulePort(rule secrules.SecurityRule) []string {
	ports := []string{}
	for i := 0; i < len(rule.Ports); i++ {
		ports = append(ports, fmt.Sprintf("%d", rule.Ports[i]))
	}
	if rule.PortStart > 0 && rule.PortEnd < 65535 {
		ports = append(ports, fmt.Sprintf("%d-%d", rule.PortStart, rule.PortEnd))
	}
	return ports
}

func convertSecurityGroupRule(rule secrules.SecurityRule) *SecurityRules {
	name := strings.Replace(rule.String(), ":", "_", -1)
	name = strings.Replace(name, " ", "_", -1)
	name = strings.Replace(name, "-", "_", -1)
	name = fmt.Sprintf("%s_%d", name, rule.Priority)
	destRule := SecurityRules{
		Name:       name,
		Properties: SecurityRulePropertiesFormat{},
	}
	protocol := "*"
	if len(rule.Protocol) == 0 || rule.Protocol == secrules.PROTO_ANY {
		protocol = "*"
	} else if rule.Protocol == secrules.PROTO_TCP {
		protocol = "Tcp"
	} else if rule.Protocol == secrules.PROTO_UDP {
		protocol = "Udp"
	} else {
		return nil
	}
	destRule.Properties.Protocol = protocol
	destRule.Properties.Description = rule.Description
	direction := "Inbound"
	if rule.Direction == secrules.SecurityRuleEgress {
		direction = "Outbound"
	}
	destRule.Properties.Direction = direction
	ipAddr := rule.IPNet.String()
	ports := convertRulePort(rule)
	if len(ports) == 0 {
		port := "*"
		destRule.Properties.SourcePortRange = port
		destRule.Properties.DestinationPortRange = port
	} else {
		destRule.Properties.SourcePortRanges = &ports
		destRule.Properties.DestinationPortRanges = &ports
	}
	destRule.Properties.DestinationAddressPrefix = ipAddr
	destRule.Properties.SourceAddressPrefix = ipAddr

	access := "Allow"
	if rule.Action == secrules.SecurityRuleDeny {
		access = "Deny"
	}
	destRule.Properties.Access = access
	priority := int32(rule.Priority)
	destRule.Properties.Priority = priority
	return &destRule
}

func (region *SRegion) updateSecurityGroupRules(secgroupId string, rules []secrules.SecurityRule) (string, error) {
	secgroup, err := region.GetSecurityGroupDetails(secgroupId)
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
	_, err = region.client.Update(jsonutils.Marshal(secgroup))
	return secgroup.ID, err
}

func (region *SRegion) AttachSecurityToInterfaces(secgroupId string, nicIds []string) error {
	secgroup, err := region.GetSecurityGroupDetails(secgroupId)
	if err != nil {
		return err
	}
	interfaces := []Interface{}
	for i := 0; i < len(nicIds); i++ {
		interfaces = append(interfaces, Interface{ID: nicIds[i]})
	}
	secgroup.Properties.NetworkInterfaces = &interfaces
	secgroup.Properties.ProvisioningState = ""
	_, err = region.client.Update(jsonutils.Marshal(secgroup))
	return err
}

func (region *SRegion) AssiginSecurityGroup(instanceId, secgroupId string) error {
	if instance, err := region.GetInstance(instanceId); err != nil {
		return err
	} else {
		nicIds := []string{}
		for _, nic := range instance.Properties.NetworkProfile.NetworkInterfaces {
			nicIds = append(nicIds, nic.ID)
		}
		return region.AttachSecurityToInterfaces(secgroupId, nicIds)
	}
}

func (self *SRegion) syncSecgroupRules(secgroupId string, rules []secrules.SecurityRule) (string, error) {
	secgroup, err := self.GetSecurityGroupDetails(secgroupId)
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
	return self.updateSecurityGroupRules(secgroup.ID, newRules)

}

func (self *SRegion) syncSecurityGroup(tagId, name string, rules []secrules.SecurityRule) (string, error) {
	secgroup, err := self.checkSecurityGroup(tagId, name)
	if err != nil {
		return "", err
	}
	return self.syncSecgroupRules(secgroup.ID, rules)
}
