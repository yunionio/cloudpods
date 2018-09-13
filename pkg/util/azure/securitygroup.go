package azure

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/utils"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
)

type SecurityRuleAccess string

const (
	// SecurityRuleAccessAllow ...
	SecurityRuleAccessAllow SecurityRuleAccess = "Allow"
	// SecurityRuleAccessDeny ...
	SecurityRuleAccessDeny SecurityRuleAccess = "Deny"
)

const (
	// SecurityRuleDirectionInbound ...
	SecurityRuleDirectionInbound SecurityRuleDirection = "Inbound"
	// SecurityRuleDirectionOutbound ...
	SecurityRuleDirectionOutbound SecurityRuleDirection = "Outbound"
)

type SecurityRuleDirection string
type SecurityRulePropertiesFormat struct {
	Description                string
	Protocol                   string
	SourcePortRange            string
	DestinationPortRange       string
	SourceAddressPrefix        string
	SourceAddressPrefixes      []string
	DestinationAddressPrefix   string
	DestinationAddressPrefixes []string
	SourcePortRanges           []string
	DestinationPortRanges      []string
	Access                     SecurityRuleAccess
	Priority                   int32
	Direction                  SecurityRuleDirection //Possible values include: 'SecurityRuleDirectionInbound', 'SecurityRuleDirectionOutbound'
	ProvisioningState          string
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
}

type SecurityGroupPropertiesFormat struct {
	SecurityRules        []SecurityRules
	DefaultSecurityRules []SecurityRules
	NetworkInterfaces    []Interface
	Subnets              []Subnet
	ProvisioningState    string //Possible values are: 'Updating', 'Deleting', and 'Failed'
}
type SSecurityGroup struct {
	vpc        *SVpc
	Properties SecurityGroupPropertiesFormat
	ID         string
	Name       string
	Location   string
	Tags       map[string]string
}

func (self *SecurityRulePropertiesFormat) String() string {
	//log.Debugf("serize rule: %s", jsonutils.Marshal(self).PrettyString())
	action := secrules.SecurityRuleDeny
	if self.Access == SecurityRuleAccessAllow {
		action = secrules.SecurityRuleAllow
	}
	direction := "in"
	if self.Direction == SecurityRuleDirectionOutbound {
		direction = "out"
	}
	cidr := self.SourceAddressPrefix
	port := self.SourcePortRange
	if len(self.SourcePortRanges) > 0 {
		port = strings.Join(self.SourcePortRanges, ",")
	}
	if direction == "out" {
		cidr = self.DestinationAddressPrefix
		port = self.DestinationPortRange
		if len(self.DestinationPortRanges) > 0 {
			port = strings.Join(self.DestinationPortRanges, ",")
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

func (self *SSecurityGroup) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SSecurityGroup) GetGlobalId() string {
	globalId, _, _ := pareResourceGroupWithName(self.ID, SECGRP_RESOURCE)
	return globalId
}

func (self *SSecurityGroup) GetDescription() string {
	return ""
}

func (self *SSecurityGroup) GetName() string {
	return strings.TrimPrefix(self.Name, self.vpc.region.Name+"-")
}

func (self *SSecurityGroup) GetRules() ([]secrules.SecurityRule, error) {
	rules := make([]secrules.SecurityRule, 0)
	sort.Sort(SecurityRulesSet(self.Properties.SecurityRules))
	priority := 100

	for _, _rule := range self.Properties.SecurityRules {
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

func (region *SRegion) CreateSecurityGroup(secName string) (*SSecurityGroup, error) {
	securityName := fmt.Sprintf("%s-%s", region.Name, secName)
	globalId, resourceGroup, securityName := pareResourceGroupWithName(securityName, SECGRP_RESOURCE)
	secClient := network.NewSecurityGroupsClientWithBaseURI(region.client.baseUrl, region.SubscriptionID)
	secClient.Authorizer = region.client.authorizer
	params := network.SecurityGroup{
		Location: &region.Name,
		Name:     &securityName,
	}
	if result, err := secClient.CreateOrUpdate(context.Background(), resourceGroup, securityName, params); err != nil {
		return nil, err
	} else if result.WaitForCompletion(context.Background(), secClient.Client); err != nil {
		return nil, err
	}
	return region.GetSecurityGroupDetails(globalId)
}

func (region *SRegion) GetSecurityGroups() ([]SSecurityGroup, error) {
	secgroups := make([]SSecurityGroup, 0)
	secClient := network.NewSecurityGroupsClientWithBaseURI(region.client.baseUrl, region.SubscriptionID)
	secClient.Authorizer = region.client.authorizer
	secgroupIds := make([]string, 0)
	if result, err := secClient.ListAll(context.Background()); err != nil {
		return nil, err
	} else {
		for _, secgrp := range result.Values() {
			if *secgrp.Location == region.Name {
				securityGroup := SSecurityGroup{}
				if secgroupId, ok := secgrp.Tags["id"]; ok {
					if utils.IsInStringArray(*secgroupId, secgroupIds) {
						continue
					} else {
						secgroupIds = append(secgroupIds, *secgroupId)
					}
				}
				if err := jsonutils.Update(&securityGroup, secgrp); err != nil {
					return nil, err
				}
				securityGroup.Name = strings.TrimPrefix(securityGroup.Name, region.Name+"-")
				secgroups = append(secgroups, securityGroup)
			}
		}
	}
	return secgroups, nil
}

func (region *SRegion) GetSecurityGroupDetails(secgroupId string) (*SSecurityGroup, error) {
	sec := SSecurityGroup{}
	secClient := network.NewSecurityGroupsClientWithBaseURI(region.client.baseUrl, region.SubscriptionID)
	secClient.Authorizer = region.client.authorizer
	_, resourceGroup, secName := pareResourceGroupWithName(secgroupId, SECGRP_RESOURCE)
	if len(secName) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	if result, err := secClient.Get(context.Background(), resourceGroup, secName, ""); err != nil {
		if result.Response.StatusCode == 404 {
			return nil, cloudprovider.ErrNotFound
		}
		return nil, err
	} else if err := jsonutils.Update(&sec, result); err != nil {
		return nil, err
	}
	return &sec, nil
}

func (self *SSecurityGroup) Refresh() error {
	if sec, err := self.vpc.region.GetSecurityGroupDetails(self.ID); err != nil {
		return err
	} else if err := jsonutils.Update(self, sec); err != nil {
		return err
	}
	return nil
}

func (region *SRegion) addTagToSecurityGroup(secgroupId, value string) error {
	secClient := network.NewSecurityGroupsClientWithBaseURI(region.client.baseUrl, region.SubscriptionID)
	secClient.Authorizer = region.client.authorizer
	_, resourceGroup, secName := pareResourceGroupWithName(secgroupId, SECGRP_RESOURCE)
	params := network.TagsObject{Tags: map[string]*string{"id": &value}}
	if result, err := secClient.UpdateTags(context.Background(), resourceGroup, secName, params); err != nil {
		return err
	} else if result.WaitForCompletion(context.Background(), secClient.Client); err != nil {
		return err
	}
	return nil
}

func (region *SRegion) checkSecurityGroup(name, secgroupId string) (*SSecurityGroup, error) {
	secName := fmt.Sprintf("%s-%s", region.Name, name)
	globalId, _, _ := pareResourceGroupWithName(secName, SECGRP_RESOURCE)
	if _, err := region.GetSecurityGroupDetails(globalId); err != nil {
		if err == cloudprovider.ErrNotFound {
			if _, err := region.CreateSecurityGroup(name); err != nil {
				return nil, err
			} else if err := region.addTagToSecurityGroup(globalId, secgroupId); err != nil {
				return nil, err
			}
		}
	} else if err := region.addTagToSecurityGroup(globalId, secgroupId); err != nil {
		return nil, err
	}
	return region.GetSecurityGroupDetails(globalId)
}

func convertRulePort(rule secrules.SecurityRule) *[]string {
	ports := []string{}
	for i := 0; i < len(rule.Ports); i++ {
		ports = append(ports, fmt.Sprintf("%d", rule.Ports[i]))
	}
	if rule.PortStart > 0 && rule.PortEnd < 65535 {
		ports = append(ports, fmt.Sprintf("%d-%d", rule.PortStart, rule.PortEnd))
	}
	return &ports
}

func convertSecurityGroupRule(rule secrules.SecurityRule) *network.SecurityRule {
	name := strings.Replace(rule.String(), ":", "_", -1)
	name = strings.Replace(name, " ", "_", -1)
	name = strings.Replace(name, "-", "_", -1)
	destRule := network.SecurityRule{
		Name: &name,
		SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{},
	}
	protocol := network.SecurityRuleProtocolAsterisk
	if len(rule.Protocol) == 0 || rule.Protocol == secrules.PROTO_ANY {
		protocol = network.SecurityRuleProtocolAsterisk
	} else if rule.Protocol == secrules.PROTO_TCP {
		protocol = network.SecurityRuleProtocolTCP
	} else if rule.Protocol == secrules.PROTO_UDP {
		protocol = network.SecurityRuleProtocolUDP
	} else {
		return nil
	}
	destRule.SecurityRulePropertiesFormat.Protocol = protocol
	destRule.SecurityRulePropertiesFormat.Description = &rule.Description
	direction := network.SecurityRuleDirectionInbound
	if rule.Direction == secrules.SecurityRuleEgress {
		direction = network.SecurityRuleDirectionOutbound
	}
	destRule.SecurityRulePropertiesFormat.Direction = direction
	ipAddr := rule.IPNet.String()
	ports := convertRulePort(rule)
	if len(*ports) == 0 {
		port := "*"
		destRule.SecurityRulePropertiesFormat.SourcePortRange = &port
		destRule.SecurityRulePropertiesFormat.DestinationPortRange = &port
	} else {
		destRule.SecurityRulePropertiesFormat.SourcePortRanges = ports
		destRule.SecurityRulePropertiesFormat.DestinationPortRanges = ports
	}
	destRule.SecurityRulePropertiesFormat.DestinationAddressPrefix = &ipAddr
	destRule.SecurityRulePropertiesFormat.SourceAddressPrefix = &ipAddr

	access := network.SecurityRuleAccessAllow
	if rule.Action == secrules.SecurityRuleDeny {
		access = network.SecurityRuleAccessDeny
	}
	destRule.SecurityRulePropertiesFormat.Access = access
	priority := int32(rule.Priority)
	destRule.SecurityRulePropertiesFormat.Priority = &priority
	return &destRule
}

func (region *SRegion) updateSecurityGroupRules(secgroupId string, rules []secrules.SecurityRule) (string, error) {
	_, resourceGroup, secName := pareResourceGroupWithName(secgroupId, SECGRP_RESOURCE)
	secClient := network.NewSecurityGroupsClientWithBaseURI(region.client.baseUrl, region.SubscriptionID)
	secClient.Authorizer = region.client.authorizer
	securityRules := []network.SecurityRule{}
	priority := int32(100)
	for i := 0; i < len(rules); i++ {
		rules[i].Priority = int(priority)
		if rule := convertSecurityGroupRule(rules[i]); rule != nil {
			securityRules = append(securityRules, *rule)
			priority++
		}
	}

	params := network.SecurityGroup{
		Location: &region.Name,
		SecurityGroupPropertiesFormat: &network.SecurityGroupPropertiesFormat{
			SecurityRules: &securityRules,
		},
	}
	if result, err := secClient.CreateOrUpdate(context.Background(), resourceGroup, secName, params); err != nil {
		return "", err
	} else if err := result.WaitForCompletion(context.Background(), secClient.Client); err != nil {
		return "", err
	}
	return secgroupId, nil
}

func (self *SRegion) syncSecgroupRules(secgroupId string, rules []secrules.SecurityRule) (string, error) {
	if secgroup, err := self.GetSecurityGroupDetails(secgroupId); err != nil {
		return "", err
	} else {
		sort.Sort(secrules.SecurityRuleSet(rules))
		sort.Sort(SecurityRulesSet(secgroup.Properties.SecurityRules))

		newRules := []secrules.SecurityRule{}

		i, j := 0, 0
		for i < len(rules) || j < len(secgroup.Properties.SecurityRules) {
			if i < len(rules) && j < len(secgroup.Properties.SecurityRules) {
				srcRule := secgroup.Properties.SecurityRules[j].Properties.String()
				destRule := rules[i].String()
				cmp := strings.Compare(srcRule, destRule)
				if cmp == 0 {
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
			} else if j >= len(secgroup.Properties.SecurityRules) {
				// add rule
				newRules = append(newRules, rules[i])
				i++
			}
		}
		return self.updateSecurityGroupRules(secgroup.ID, newRules)
	}
}

func (self *SRegion) syncSecurityGroup(secgroupId, name string, rules []secrules.SecurityRule) (string, error) {
	if secgroup, err := self.checkSecurityGroup(name, secgroupId); err != nil {
		log.Errorf("check err: %v", err)
		return "", err
	} else {

		return self.syncSecgroupRules(secgroup.ID, rules)
	}
}
