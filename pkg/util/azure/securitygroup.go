package azure

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/secrules"

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
}

func (self *SecurityRulePropertiesFormat) String() string {
	log.Debugf("serize rule: %s", jsonutils.Marshal(self).PrettyString())
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
	log.Debugf("result: %s", result)
	return result
}

func (self *SSecurityGroup) GetId() string {
	return self.ID
}

func (self *SSecurityGroup) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SSecurityGroup) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", self.vpc.GetGlobalId(), self.Name)
}

func (self *SSecurityGroup) GetDescription() string {
	return ""
}

func (self *SSecurityGroup) GetName() string {
	return self.Name
}

func (self *SSecurityGroup) GetRules() ([]secrules.SecurityRule, error) {
	rules := make([]secrules.SecurityRule, 0)
	for _, _rule := range self.Properties.SecurityRules {
		for _, ruleString := range strings.Split(_rule.Properties.String(), ";") {
			if rule, err := secrules.ParseSecurityRule(ruleString); err != nil {
				return rules, err
			} else {
				rule.Priority = 100 - int(_rule.Properties.Priority)
				rule.Description = _rule.Properties.Description
				if rule.Priority < 0 {
					rule.Priority = 1
				}
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

func (self *SSecurityGroup) Refresh() error {
	if resourceGroup, _, err := PareResourceGroupWithName(self.ID); err != nil {
		log.Errorf("Refresh SecurityGroup error %v", err)
		return err
	} else {
		networkClient := network.NewSecurityGroupsClientWithBaseURI(self.vpc.region.client.baseUrl, self.vpc.region.SubscriptionID)
		networkClient.Authorizer = self.vpc.region.client.authorizer
		if secgrp, err := networkClient.Get(context.Background(), resourceGroup, self.Name, ""); err != nil {
			return err
		} else if err := jsonutils.Update(self, secgrp); err != nil {
			return err
		}
	}
	return nil
}
