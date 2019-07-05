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

func (ssrs *SecurityGroupSubRuleSet) getDirectionRules() ([]SecurityRule, []SecurityRule) {
	inRules, outRules := []SecurityRule{}, []SecurityRule{}
	if ssrs.allowRules != nil {
		for _, rule := range ssrs.allowRules.rules {
			if rule.Direction == DIR_IN {
				inRules = append(inRules, rule)
			} else {
				outRules = append(outRules, rule)
			}
		}
	}
	return inRules, outRules
}

func isRuleEqual(rules []SecurityRule, _rules []SecurityRule) bool {
	if len(rules) != len(_rules) {
		return false
	}

	sort.SliceStable(rules, func(i, j int) bool {
		if rules[i].Priority < rules[j].Priority {
			return true
		} else if rules[i].Priority == rules[j].Priority {
			return rules[i].String() < rules[j].String()
		}
		return false
	})
	sort.SliceStable(_rules, func(i, j int) bool {
		if _rules[i].Priority < _rules[j].Priority {
			return true
		} else if _rules[i].Priority == _rules[j].Priority {
			return _rules[i].String() < _rules[j].String()
		}
		return false
	})

	for i := 0; i < len(rules); i++ {
		if rules[i].String() != _rules[i].String() {
			return false
		}
	}
	return true
}

func (ssrs *SecurityGroupSubRuleSet) isEqual(_ssrs *SecurityGroupSubRuleSet) bool {
	if _ssrs == nil {
		return false
	}

	inRules, outRules := ssrs.getDirectionRules()
	_inRules, _outRules := _ssrs.getDirectionRules()

	return isRuleEqual(inRules, _inRules) && isRuleEqual(outRules, _outRules)
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
