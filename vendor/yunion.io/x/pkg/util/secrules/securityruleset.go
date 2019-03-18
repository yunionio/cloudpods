package secrules

import (
	"bytes"
	"sort"

	"yunion.io/x/log"
)

type SecurityGroupSubSubRuleSet struct {
	ipKey string
	rules []SecurityRule
}

func (sssrs *SecurityGroupSubSubRuleSet) AddRule(rule SecurityRule) {
	if sssrs.rules == nil {
		sssrs.ipKey = rule.getIPKey()
		sssrs.rules = []SecurityRule{rule}
		return
	}
	if sssrs.ipKey != rule.getIPKey() {
		panic("rule ip key not equal")
	}
	idx := 0
	for idx < len(sssrs.rules) {
		rel := sssrs.rules[idx].protoRelation(rule)
		switch rel {
		case RELATION_INDEPENDENT:
			idx++
		case RELATION_IDENTICAL, RELATION_SUPERSET:
			return
		case RELATION_SUBSET:
			sssrs.rules = append(sssrs.rules[:idx], sssrs.rules[idx+1:]...)
		case RELATION_NEXT_AHEAD, RELATION_NEXT_AFTER, RELATION_OVERLAP:
			r, err := rule.merge(sssrs.rules[idx])
			if err != nil {
				log.Errorf("Merge failed?? %v", err)
				continue
			}
			idx = 0
			sssrs.rules[idx] = r
		}
	}
	sssrs.rules = append(sssrs.rules, rule)
}

type SecurityGroupSubRuleSet struct {
	allowRules *SecurityGroupSubSubRuleSet
	denyRules  *SecurityGroupSubSubRuleSet
}

func (ssrs *SecurityGroupSubRuleSet) AddRule(rule SecurityRule) {
	if rule.Action == SecurityRuleAllow {
		if ssrs.allowRules == nil {
			ssrs.allowRules = &SecurityGroupSubSubRuleSet{}
		}
		ssrs.allowRules.AddRule(rule)
	} else {
		if ssrs.denyRules == nil {
			ssrs.denyRules = &SecurityGroupSubSubRuleSet{}
		}
		ssrs.denyRules.AddRule(rule)
	}
}

func (ssrs *SecurityGroupSubRuleSet) isEqual(_ssrs *SecurityGroupSubRuleSet) bool {
	allowRules := ssrs.GetAllowRules()
	_allowRules := _ssrs.GetAllowRules()
	if len(allowRules.rules) != len(_allowRules.rules) {
		return false
	}

	sort.Slice(allowRules.rules, func(i, j int) bool {
		return allowRules.rules[i].Priority < allowRules.rules[j].Priority
	})

	sort.Slice(_allowRules.rules, func(i, j int) bool {
		return _allowRules.rules[i].Priority < _allowRules.rules[j].Priority
	})
	for i := 0; i < len(allowRules.rules); i++ {
		if allowRules.rules[i].String() != _allowRules.rules[i].String() {
			return false
		}
	}

	denyRules := ssrs.GetDenyRules()
	_denyRuels := _ssrs.GetDenyRules()
	if len(denyRules.rules) != len(_denyRuels.rules) {
		return false
	}

	sort.Slice(denyRules.rules, func(i, j int) bool {
		return denyRules.rules[i].Priority < denyRules.rules[j].Priority
	})

	sort.Slice(_denyRuels.rules, func(i, j int) bool {
		return _denyRuels.rules[i].Priority < _denyRuels.rules[j].Priority
	})
	for i := 0; i < len(denyRules.rules); i++ {
		if denyRules.rules[i].String() != _denyRuels.rules[i].String() {
			return false
		}
	}

	return true
}

func (ssrs *SecurityGroupSubRuleSet) GetAllowRules() SecurityGroupSubSubRuleSet {
	if ssrs.allowRules == nil {
		return SecurityGroupSubSubRuleSet{}
	}
	return *ssrs.allowRules
}

func (ssrs *SecurityGroupSubRuleSet) GetDenyRules() SecurityGroupSubSubRuleSet {
	if ssrs.denyRules == nil {
		return SecurityGroupSubSubRuleSet{}
	}
	return *ssrs.denyRules
}

type SecurityGroupRuleSet struct {
	rules map[string]*SecurityGroupSubRuleSet
}

func (srs *SecurityGroupRuleSet) AddRule(rule SecurityRule) {
	if srs.rules == nil {
		srs.rules = map[string]*SecurityGroupSubRuleSet{}
	}
	key := rule.getIPKey()
	if _, ok := srs.rules[key]; !ok {
		srs.rules[key] = &SecurityGroupSubRuleSet{}
	}
	ssrs := srs.rules[key]
	if len(rule.Ports) > 0 {
		ports := rule.Ports
		rule.Ports = []int{}
		for i := 0; i < len(ports); i++ {
			rule.PortStart = ports[i]
			rule.PortEnd = ports[i]
			ssrs.AddRule(rule)
		}
		return
	}
	ssrs.AddRule(rule)
}

func (srs *SecurityGroupRuleSet) GetSubRuleSet(key string) *SecurityGroupSubRuleSet {
	if srs.rules == nil {
		return nil
	}
	s, ok := srs.rules[key]
	if !ok {
		return nil
	}
	return s
}

func (srs *SecurityGroupRuleSet) GetAllowRules() []SecurityRule {
	rules := []SecurityRule{}
	for _, v := range srs.rules {
		rules = append(rules, v.GetAllowRules().rules...)
	}
	return rules
}

func (srs *SecurityGroupRuleSet) GetDenyRules() []SecurityRule {
	rules := []SecurityRule{}
	for _, v := range srs.rules {
		rules = append(rules, v.GetDenyRules().rules...)
	}
	return rules
}

func (srs *SecurityGroupRuleSet) IsEqual(src SecurityGroupRuleSet) bool {
	if len(srs.rules) != len(src.rules) {
		return false
	}
	for k, v := range srs.rules {
		_v := src.GetSubRuleSet(k)
		if _v == nil {
			return false
		}
		if !v.isEqual(_v) {
			return false
		}
	}
	return true
}

func (srs *SecurityGroupRuleSet) String() string {
	buf := bytes.Buffer{}
	for _, r := range srs.GetAllowRules() {
		buf.WriteString(r.String())
		buf.WriteString(";")
	}
	for _, r := range srs.GetDenyRules() {
		buf.WriteString(r.String())
		buf.WriteString(";")
	}
	return buf.String()
}
