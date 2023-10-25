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
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/util/secrules"
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
	Priority                   int      `json:"priority,omitempty"`
	Direction                  string   `json:"direction,omitempty"` //Inbound or Outbound
	ProvisioningState          string   `json:"-"`
}

type SecurityRules struct {
	region *SRegion

	Properties SecurityRulePropertiesFormat
	Name       string
	ID         string
}

func (self *SecurityRules) GetGlobalId() string {
	return strings.ToLower(self.ID)
}

func (self *SecurityRules) GetDescription() string {
	return self.Properties.Description
}

func (self *SecurityRules) GetPriority() int {
	return self.Properties.Priority
}

func (self *SecurityRules) GetDirection() secrules.TSecurityRuleDirection {
	if strings.ToLower(self.Properties.Direction) == "inbound" {
		return secrules.DIR_IN
	}
	return secrules.DIR_OUT
}

func (self *SecurityRules) Delete() error {
	return self.region.del(self.ID)
}

func (self *SecurityRules) GetAction() secrules.TSecurityRuleAction {
	if strings.ToLower(self.Properties.Access) == "allow" {
		return secrules.SecurityRuleAllow
	}
	return secrules.SecurityRuleDeny
}

func (self *SecurityRules) GetProtocol() string {
	if self.Properties.Protocol == "*" {
		return secrules.PROTO_ANY
	}
	return self.Properties.Protocol
}

func (self *SecurityRules) GetCIDRs() []string {
	ret := []string{}
	if len(self.Properties.DestinationAddressPrefix) > 0 && self.Properties.DestinationAddressPrefix != "*" {
		ret = append(ret, self.Properties.DestinationAddressPrefix)
	}
	for _, ip := range self.Properties.DestinationAddressPrefixes {
		if ip != "*" {
			ret = append(ret, ip)
		}
	}
	if len(ret) == 0 {
		ret = append(ret, "0.0.0.0/0")
	}
	return ret
}

func (self *SecurityRules) GetPorts() string {
	ports := []string{}
	if len(self.Properties.DestinationPortRange) > 0 && self.Properties.DestinationPortRange != "*" {
		ports = append(ports, self.Properties.DestinationPortRange)
	}
	for _, port := range self.Properties.DestinationPortRanges {
		if port != "*" {
			ports = append(ports, port)
		}
	}
	return strings.Join(ports, ",")
}

func (self *SecurityRules) Update(opts *cloudprovider.SecurityGroupRuleUpdateOptions) error {
	return cloudprovider.ErrNotImplemented
}
