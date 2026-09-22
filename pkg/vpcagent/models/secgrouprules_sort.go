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

package models

import (
	"sort"

	"yunion.io/x/pkg/util/secrules"

	compute_models "yunion.io/x/onecloud/pkg/compute/models"
)

func (el *Guest) OrderedSecurityGroupRules() []*SecurityGroupRule {
	// deny any incoming traffic and allow ARP
	rs := []*SecurityGroupRule{
		{
			// deny all in-bound traffic
			SSecurityGroupRule: compute_models.SSecurityGroupRule{
				Priority:  1,
				Direction: string(secrules.SecurityRuleIngress),
				Action:    string(secrules.SecurityRuleDeny),
			},
		},
		{
			// allow in-bound arp traffic
			SSecurityGroupRule: compute_models.SSecurityGroupRule{
				Priority:  2,
				Direction: string(secrules.SecurityRuleIngress),
				Protocol:  "arp",
				Action:    string(secrules.SecurityRuleAllow),
			},
		},
	}
	for _, secgroup := range el.SecurityGroups {
		rs = append(rs, secgroup.securityGroupRules(100)...)
	}
	if el.AdminSecurityGroup != nil {
		rs = append(rs, el.AdminSecurityGroup.securityGroupRules(1000)...)
	}
	sort.Slice(rs, SecurityGroupRuleLessFunc(rs))
	return rs
}

func (el *SecurityGroup) securityGroupRules(basePriority int64) []*SecurityGroupRule {
	rs := make([]*SecurityGroupRule, 0, len(el.SecurityGroupRules))
	for _, r := range el.SecurityGroupRules {
		r = r.Copy()
		r.Priority += int(basePriority)
		rs = append(rs, r)
	}
	return rs
}

// SecurityGroupRuleLessFunc orders rules by priority, which is what the
// prefixes of OrderedSecurityGroupRules rely on: the deny-all and the ARP
// rules come first, then the rules of the security groups of the guest, and
// the rules of its admin security group last.
//
// Rules that end up on the same priority are ordered by id.  The rules of a
// security group come out of a map, so without a tie breaker the same model
// sets would produce a different list on every sync, and OVN matches on the
// priority while the list is what gets written out.
func SecurityGroupRuleLessFunc(rs []*SecurityGroupRule) func(i, j int) bool {
	return func(i, j int) bool {
		if rs[i].Priority != rs[j].Priority {
			return rs[i].Priority < rs[j].Priority
		}
		return rs[i].Id < rs[j].Id
	}
}
