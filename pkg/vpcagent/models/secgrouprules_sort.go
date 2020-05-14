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
