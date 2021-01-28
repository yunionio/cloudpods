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

package cloudprovider

import (
	"sort"
	"strings"

	"gopkg.in/fatih/set.v0"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/utils"
)

type SecDriver interface {
	GetDefaultSecurityGroupInRule() SecurityRule
	GetDefaultSecurityGroupOutRule() SecurityRule
	GetSecurityGroupRuleMaxPriority() int
	GetSecurityGroupRuleMinPriority() int
	IsOnlySupportAllowRules() bool
}

func NewSecRuleInfo(driver SecDriver) SecRuleInfo {
	return SecRuleInfo{
		InDefaultRule:           driver.GetDefaultSecurityGroupInRule(),
		OutDefaultRule:          driver.GetDefaultSecurityGroupOutRule(),
		MinPriority:             driver.GetSecurityGroupRuleMinPriority(),
		MaxPriority:             driver.GetSecurityGroupRuleMaxPriority(),
		IsOnlySupportAllowRules: driver.IsOnlySupportAllowRules(),
	}
}

const DEFAULT_DEST_RULE_ID = "default_dest_rule_id"
const DEFAULT_SRC_RULE_ID = "default_src_rule_id"

type SecRuleInfo struct {
	InDefaultRule           SecurityRule
	OutDefaultRule          SecurityRule
	Rules                   SecurityRuleSet
	MinPriority             int
	MaxPriority             int
	IsOnlySupportAllowRules bool
}

func (r SecRuleInfo) AddDefaultRule(d SecRuleInfo, inRules, outRules []SecurityRule, isSrc bool) ([]SecurityRule, []SecurityRule) {
	min, max := r.MinPriority, r.MaxPriority
	r.InDefaultRule.Priority = min + 1
	r.OutDefaultRule.Priority = min + 1
	if max >= min {
		r.InDefaultRule.Priority = min - 1
		r.OutDefaultRule.Priority = min - 1
	}

	if isSrc {
		r.InDefaultRule.Id = DEFAULT_SRC_RULE_ID
		r.OutDefaultRule.Id = DEFAULT_SRC_RULE_ID
	} else {
		r.InDefaultRule.ExternalId = DEFAULT_DEST_RULE_ID
		r.OutDefaultRule.ExternalId = DEFAULT_DEST_RULE_ID
	}

	inRules = append(inRules, r.InDefaultRule)
	outRules = append(outRules, r.OutDefaultRule)
	return inRules, outRules
}

type SecurityGroupFilterOptions struct {
	VpcId     string
	Name      string
	ProjectId string
}

type SecurityGroupCreateInput struct {
	Name      string
	Desc      string
	VpcId     string
	ProjectId string
	Rules     []secrules.SecurityRule
}

type SecurityRule struct {
	secrules.SecurityRule
	Name       string
	ExternalId string
	Id         string
}

func (r SecurityRule) String() string {
	return r.SecurityRule.String()
}

type SecurityRuleSet []SecurityRule

func (rules SecurityRuleSet) Split() (in, out SecurityRuleSet) {
	for i := 0; i < len(rules); i++ {
		if rules[i].Direction == secrules.DIR_IN {
			in = append(in, rules[i])
		} else {
			out = append(out, rules[i])
		}
	}
	return
}

func (srs SecurityRuleSet) Len() int {
	return len(srs)
}

func (srs SecurityRuleSet) Swap(i, j int) {
	srs[i], srs[j] = srs[j], srs[i]
}

func (srs SecurityRuleSet) Less(i, j int) bool {
	return srs[i].Priority < srs[j].Priority || (srs[i].Priority == srs[j].Priority && srs[i].String() < srs[j].String())
}

func (srs SecurityRuleSet) AllowList() secrules.SecurityRuleSet {
	rules := secrules.SecurityRuleSet{}
	for _, r := range srs {
		rules = append(rules, r.SecurityRule)
	}
	return rules.AllowList()
}

func (srs SecurityRuleSet) Debug() {
	for i := 0; i < len(srs); i++ {
		log.Debugf("Name: %s id: %s external_id: %s priority: %d %s", srs[i].Name, srs[i].Id, srs[i].ExternalId, srs[i].Priority, srs[i].String())
	}
}

func SortSecurityRule(rules SecurityRuleSet, max, min int, isAsc, onlyAllowRules bool) {
	if (max >= min || onlyAllowRules) && !isAsc {
		sort.Sort(sort.Reverse(rules))
		return
	}
	sort.Sort(rules)
	return
}

func isAllowListEqual(src, dest secrules.SecurityRuleSet) bool {
	if len(src) != len(dest) {
		return false
	}
	s1, s2 := set.New(set.ThreadSafe), set.New(set.ThreadSafe)
	for i := 0; i < len(src); i++ {
		s1.Add(src[i].String())
		s2.Add(dest[i].String())
	}
	return s1.IsEqual(s2)
}

func CompareRules(src, dest SecRuleInfo, debug bool) (common, inAdds, outAdds, inDels, outDels SecurityRuleSet) {
	srcInRules, srcOutRules := src.Rules.Split()
	destInRules, destOutRules := dest.Rules.Split()

	srcInRules, srcOutRules = src.AddDefaultRule(dest, srcInRules, srcOutRules, true)
	destInRules, destOutRules = dest.AddDefaultRule(src, destInRules, destOutRules, false)

	if debug {
		log.Debugf("src in rules: ")
		srcInRules.Debug()
	}

	// AllowList 需要优先级从高到低排序
	SortSecurityRule(srcInRules, src.MaxPriority, src.MinPriority, false, src.IsOnlySupportAllowRules)
	SortSecurityRule(srcOutRules, src.MaxPriority, src.MinPriority, false, src.IsOnlySupportAllowRules)

	SortSecurityRule(destInRules, dest.MaxPriority, dest.MinPriority, false, dest.IsOnlySupportAllowRules)
	SortSecurityRule(destOutRules, dest.MaxPriority, dest.MinPriority, false, dest.IsOnlySupportAllowRules)

	srcInAllowList := srcInRules.AllowList()
	srcOutAllowList := srcOutRules.AllowList()

	destInAllowList := destInRules.AllowList()
	destOutAllowList := destOutRules.AllowList()
	inEquals, outEquals := isAllowListEqual(srcInAllowList, destInAllowList), isAllowListEqual(srcOutAllowList, destOutAllowList)

	if inEquals && outEquals {
		return
	}

	if debug {
		log.Debugf("In: src: %s dest: %s result: %v", srcInAllowList.String(), destInAllowList.String(), inEquals)
		log.Debugf("Out: src: %s dest: %s result: %v", srcOutAllowList.String(), destOutAllowList.String(), outEquals)
	}

	var tryUseAllowList = func(defaultRule SecurityRule, allowList secrules.SecurityRuleSet, rules SecurityRuleSet, isOnlyAllowList bool) SecurityRuleSet {
		if len(allowList) < len(rules) || isOnlyAllowList {
			rules = SecurityRuleSet{}
			for i := range allowList {
				rule := SecurityRule{}
				rule.SecurityRule = allowList[i]
				rules = append(rules, rule)
			}

			if !utils.IsInStringArray(allowList.String(), []string{
				"",
				"in:allow any",
				"out:allow any",
				"in:deny any",
				"out:deny any",
			}) && strings.HasSuffix(defaultRule.SecurityRule.String(), "deny any") {
				rules = append(rules, defaultRule)
			}
		}
		return rules
	}

	srcInRules = tryUseAllowList(src.InDefaultRule, srcInAllowList, srcInRules, dest.IsOnlySupportAllowRules)
	srcOutRules = tryUseAllowList(src.OutDefaultRule, srcOutAllowList, srcOutRules, dest.IsOnlySupportAllowRules)

	if inEquals {
		srcInRules, destInRules = []SecurityRule{}, []SecurityRule{}
	}
	if outEquals {
		srcOutRules, destOutRules = []SecurityRule{}, []SecurityRule{}
	}

	if debug {
		log.Debugf("src in rules: ")
		srcInRules.Debug()
	}

	// 默认从优先级低到高比较
	SortSecurityRule(srcInRules, src.MaxPriority, src.MinPriority, true, src.IsOnlySupportAllowRules)
	SortSecurityRule(srcOutRules, src.MaxPriority, src.MinPriority, true, src.IsOnlySupportAllowRules)

	SortSecurityRule(destInRules, dest.MaxPriority, dest.MinPriority, true, dest.IsOnlySupportAllowRules)
	SortSecurityRule(destOutRules, dest.MaxPriority, dest.MinPriority, true, dest.IsOnlySupportAllowRules)

	var addPriority = func(priority int, min, max int, onlyAllowRules bool) int {
		if onlyAllowRules {
			return priority
		}
		inc := 1
		if max < min {
			max, min, inc = min, max, -1
		}
		if priority >= max || priority <= min {
			return priority
		}
		return priority + inc
	}

	var _compare = func(srcRules SecurityRuleSet, destRules SecurityRuleSet) (common, add, del SecurityRuleSet) {
		i, j, priority := 0, 0, (dest.MinPriority-1+dest.MaxPriority)/2
		for i < len(srcRules) || j < len(destRules) {
			if i < len(srcRules) && j < len(destRules) {
				destRuleStr := destRules[j].String()
				srcRuleStr := srcRules[i].String()
				if debug {
					log.Debugf("compare src %s(%s) priority(%d) %s -> dest name(%s) %s(%s) priority(%d) %s\n",
						srcRules[i].Id, srcRules[i].ExternalId, srcRules[i].Priority, srcRules[i].String(),
						destRules[j].Name, destRules[j].ExternalId, destRules[j].Id, destRules[j].Priority, destRules[j].String())
				}
				cmp := strings.Compare(destRuleStr, srcRuleStr)
				if cmp == 0 {
					destRules[j].Id = srcRules[i].Id
					common = append(common, destRules[j])
					if destRules[j].ExternalId != DEFAULT_DEST_RULE_ID {
						priority = destRules[j].Priority
					}
					i++
					j++
				} else if cmp < 0 {
					del = append(del, destRules[j])
					j++
				} else {
					priority = addPriority(priority, dest.MinPriority, dest.MaxPriority, dest.IsOnlySupportAllowRules)
					srcRules[i].Priority = priority
					add = append(add, srcRules[i])
					i++
				}
			} else if i >= len(srcRules) {
				del = append(del, destRules[j])
				j++
			} else if j >= len(destRules) {
				priority = addPriority(priority, dest.MinPriority, dest.MaxPriority, dest.IsOnlySupportAllowRules)
				srcRules[i].Priority = priority
				add = append(add, srcRules[i])
				i++
			}
		}
		return
	}

	type rulePair struct {
		srcRules  SecurityRuleSet
		destRules SecurityRuleSet
		protocol  string
	}

	var splitRules = func(src, dest SecurityRuleSet) []rulePair {
		rules := map[string]rulePair{}
		for _, r := range src {
			pair, ok := rules[r.Protocol]
			if !ok {
				pair = rulePair{srcRules: SecurityRuleSet{}, destRules: SecurityRuleSet{}, protocol: r.Protocol}
			}
			pair.srcRules = append(pair.srcRules, r)
			rules[r.Protocol] = pair
		}

		for _, r := range dest {
			pair, ok := rules[r.Protocol]
			if !ok {
				pair = rulePair{srcRules: SecurityRuleSet{}, destRules: SecurityRuleSet{}, protocol: r.Protocol}
			}
			pair.destRules = append(pair.destRules, r)
			rules[r.Protocol] = pair
		}

		ret := []rulePair{}
		for _, r := range rules {
			ret = append(ret, r)
		}
		return ret
	}

	var compare = func(src, dest SecurityRuleSet) (common, added, dels SecurityRuleSet) {
		pairs := splitRules(src, dest)
		for _, r := range pairs {
			_common, _add, _dels := _compare(r.srcRules, r.destRules)
			common = append(common, _common...)
			added = append(added, _add...)
			dels = append(dels, _dels...)
		}
		return
	}

	var inCommon, outCommon SecurityRuleSet
	inCommon, inAdds, inDels = compare(srcInRules, destInRules)
	outCommon, outAdds, outDels = compare(srcOutRules, destOutRules)

	var handleDefaultRules = func(removed, added []SecurityRule, isOnlyAllowList bool) ([]SecurityRule, []SecurityRule) {
		ret := []SecurityRule{}
		for _, rule := range removed {
			if rule.ExternalId == DEFAULT_DEST_RULE_ID {
				if debug {
					log.Debugf("remove dest default rule: %s external id %s priority: %d", rule.String(), rule.ExternalId, rule.Priority)
				}
				if rule.Action == secrules.SecurityRuleDeny && isOnlyAllowList {
					continue
				}
				switch rule.Action {
				case secrules.SecurityRuleDeny:
					rule.Action = secrules.SecurityRuleAllow
				case secrules.SecurityRuleAllow:
					rule.Action = secrules.SecurityRuleDeny
				}
				rule.Priority = dest.MinPriority

				find := false
				for i := range added {
					if added[i].String() == rule.String() {
						find = true
						break
					}
				}
				if !find {
					if debug {
						log.Debugf("add new default rule: %s external id %s priority: %d", rule.String(), rule.ExternalId, rule.Priority)
					}
					added = append(added, rule)
				}
			} else {
				ret = append(ret, rule)
			}
		}
		return ret, added
	}

	inDels, inAdds = handleDefaultRules(inDels, inAdds, dest.IsOnlySupportAllowRules)
	outDels, outAdds = handleDefaultRules(outDels, outAdds, dest.IsOnlySupportAllowRules)
	common, _ = handleDefaultRules(append(inCommon, outCommon...), []SecurityRule{}, dest.IsOnlySupportAllowRules)
	return
}

func SortUniqPriority(rules SecurityRuleSet) []SecurityRule {
	sort.Sort(rules)
	priMap := map[int]bool{}
	for i := range rules {
		for {
			_, ok := priMap[rules[i].Priority]
			if !ok {
				priMap[rules[i].Priority] = true
				break
			}
			rules[i].Priority = rules[i].Priority - 1
		}
	}
	return rules
}
