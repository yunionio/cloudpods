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
)

func (el *Guest) OrderedSecurityGroupRules() []*SecurityGroupRule {
	rs := []*SecurityGroupRule{}
	for _, secgroup := range el.SecurityGroups {
		rs = append(rs, secgroup.securityGroupRules()...)
	}
	sort.Slice(rs, SecurityGroupRuleLessFunc(rs))
	if el.AdminSecurityGroup != nil {
		rs = append(rs, el.AdminSecurityGroup.OrderedSecurityGroupRules()...)
	}
	return rs
}

func (el *SecurityGroup) securityGroupRules() []*SecurityGroupRule {
	rs := make([]*SecurityGroupRule, 0, len(el.SecurityGroupRules))
	for _, r := range el.SecurityGroupRules {
		rs = append(rs, r)
	}
	return rs
}

func (el *SecurityGroup) OrderedSecurityGroupRules() []*SecurityGroupRule {
	rs := el.securityGroupRules()
	sort.Slice(rs, SecurityGroupRuleLessFunc(rs))
	return rs
}

func SecurityGroupRuleLessFunc(rs []*SecurityGroupRule) func(i, j int) bool {
	return func(i, j int) bool {
		return rs[i].Priority < rs[i].Priority
	}
}
