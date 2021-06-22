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

import "net/url"

type SWafRule struct {
	RuleId        string
	Description   string
	DefaultAction string
	DefaultState  string
}

type SRuleGroup struct {
	ruleGroupName string
	description   string
	Rules         []SWafRule
}

type SManagedRuleGroupProperties struct {
	ProvisioningState string
	RuleSetId         string
	RuleSetType       string
	RuleSetVersion    string
	RuleGroups        []SRuleGroup
}

type SManagedRuleGroup struct {
	Name       string
	Id         string
	Type       string
	Properties SManagedRuleGroupProperties
}

func (self *SRegion) ListManagedRuleGroups() ([]SManagedRuleGroup, error) {
	groups := []SManagedRuleGroup{}
	err := self.list("Microsoft.Network/FrontDoorWebApplicationFirewallManagedRuleSets", url.Values{}, &groups)
	if err != nil {
		return nil, err
	}
	return groups, nil
}
