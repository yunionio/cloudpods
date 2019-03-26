package secrules

import (
	"bytes"
	"fmt"
	"sort"
)

type SecurityGroupSubSubRuleSet struct {
	ipKey string
	rules []SecurityRule
}

func (sssrs *SecurityGroupSubSubRuleSet) addRule(rule SecurityRule) {
	if sssrs.rules == nil {
		sssrs.ipKey = rule.getIPKey()
		sssrs.rules = []SecurityRule{rule}
		return
	}
	if sssrs.ipKey != rule.getIPKey() {
		panic(fmt.Sprintf("rule ip key %s not equal %s", sssrs.ipKey, rule.getIPKey()))
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
			rule = rule.merge(sssrs.rules[idx])
			sssrs.rules = append(sssrs.rules[:idx], sssrs.rules[idx+1:]...)
			idx = 0
		}
	}
	sssrs.rules = append(sssrs.rules, rule)
}

type SecurityGroupSubRuleSet struct {
	allowRules *SecurityGroupSubSubRuleSet
	denyRules  *SecurityGroupSubSubRuleSet
}

func (ssrs *SecurityGroupSubRuleSet) addRule(rule SecurityRule) {
	if rule.Action == SecurityRuleAllow {
		if ssrs.allowRules == nil {
			ssrs.allowRules = &SecurityGroupSubSubRuleSet{}
		}
		ssrs.allowRules.addRule(rule)
	} else {
		if ssrs.denyRules == nil {
			ssrs.denyRules = &SecurityGroupSubSubRuleSet{}
		}
		ssrs.denyRules.addRule(rule)
	}
}

func (ssrs *SecurityGroupSubRuleSet) isRulesEqual(rules *SecurityGroupSubSubRuleSet, _rules *SecurityGroupSubSubRuleSet) bool {
	if rules == _rules { //都是nil
		return true
	}
	if rules == nil || _rules == nil { //其中一个是nil
		return false
	}
	if len(rules.rules) != len(_rules.rules) {
		return false
	}

	sort.Slice(rules.rules, func(i, j int) bool {
		return rules.rules[i].Priority < rules.rules[j].Priority
	})
	sort.Slice(_rules.rules, func(i, j int) bool {
		return _rules.rules[i].Priority < _rules.rules[j].Priority
	})

	rulePriority, initPriority := 0, 0
	_rulePriority, _initPriority := 0, 0
	for i := 0; i < len(rules.rules); i++ {
		if rulePriority != rules.rules[i].Priority {
			rulePriority = rules.rules[i].Priority
			initPriority++
		}
		find, ruleStr := false, rules.rules[i].String()
		for j := 0; j < len(_rules.rules); j++ {
			if _rulePriority != _rules.rules[j].Priority {
				_rulePriority = _rules.rules[j].Priority
				_initPriority++
			}
			//仅在每个优先级阶梯下进行对比
			if initPriority != _initPriority {
				continue
			}
			if _rules.rules[j].String() == ruleStr {
				find = true
			}
		}
		if !find {
			return false
		}
	}
	return true
}

func (ssrs *SecurityGroupSubRuleSet) isEqual(_ssrs *SecurityGroupSubRuleSet) bool {
	if _ssrs == nil {
		return false
	}
	return ssrs.isRulesEqual(ssrs.allowRules, _ssrs.allowRules) && ssrs.isRulesEqual(ssrs.denyRules, _ssrs.denyRules)
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
		//规则合并时，是依据PortStart和PortEnd,因此将多个不连续的端口拆分为单个端口连续的规则进行合并
		rule.Ports = []int{}
		for i := 0; i < len(ports); i++ {
			rule.PortStart = ports[i]
			rule.PortEnd = ports[i]
			ssrs.addRule(rule)
		}
		return
	}
	ssrs.addRule(rule)
}

func (srs *SecurityGroupRuleSet) getSubRuleSet(key string) *SecurityGroupSubRuleSet {
	if srs.rules == nil {
		return nil
	}
	s, ok := srs.rules[key]
	if !ok {
		return nil
	}
	return s
}

func (srs *SecurityGroupRuleSet) getAllowRules() []SecurityRule {
	rules := []SecurityRule{}
	for _, v := range srs.rules {
		if v.allowRules != nil {
			rules = append(rules, v.allowRules.rules...)
		}
	}
	return rules
}

func (srs *SecurityGroupRuleSet) getDenyRules() []SecurityRule {
	rules := []SecurityRule{}
	for _, v := range srs.rules {
		if v.denyRules != nil {
			rules = append(rules, v.denyRules.rules...)
		}
	}
	return rules
}

func (srs *SecurityGroupRuleSet) IsEqual(src SecurityGroupRuleSet) bool {
	if len(srs.rules) != len(src.rules) {
		return false
	}
	for k, v := range srs.rules {
		_v := src.getSubRuleSet(k)
		if !v.isEqual(_v) {
			return false
		}
	}
	return true
}

func (srs *SecurityGroupRuleSet) String() string {
	buf := bytes.Buffer{}
	for _, r := range srs.getAllowRules() {
		buf.WriteString(r.String())
		buf.WriteString(";")
	}
	for _, r := range srs.getDenyRules() {
		buf.WriteString(r.String())
		buf.WriteString(";")
	}
	return buf.String()
}
