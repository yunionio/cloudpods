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

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/secrules"
)

const DEFAULT_CLOUD_RULE_ID = "default_cloud_rule_id"
const DEFAULT_LOCAL_RULE_ID = "default_local_rule_id"

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
	Name             string
	ExternalId       string
	LocalRulePrority int
}

type LocalSecurityRule struct {
	secrules.SecurityRule
	ExternalId string
}

func (r LocalSecurityRule) String() string {
	return r.SecurityRule.String()
}

type LocalSecurityRuleSet []LocalSecurityRule

func (srs LocalSecurityRuleSet) Len() int {
	return len(srs)
}

func (srs LocalSecurityRuleSet) Swap(i, j int) {
	srs[i], srs[j] = srs[j], srs[i]
}

func (srs LocalSecurityRuleSet) Less(i, j int) bool {
	if srs[i].Priority > srs[j].Priority {
		return true
	} else if srs[i].Priority == srs[j].Priority {
		return srs[i].String() < srs[j].String()
	}
	return false
}

func (srs LocalSecurityRuleSet) AllowList() secrules.SecurityRuleSet {
	rules := secrules.SecurityRuleSet{}
	for _, r := range srs {
		rules = append(rules, r.SecurityRule)
	}
	return rules.AllowList()
}

func (r SecurityRule) String() string {
	return r.SecurityRule.String()
}

type SecurityRuleSet []SecurityRule

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

func AddDefaultRule(rules []SecurityRule, defaultRule SecurityRule, localRuleStr string, min, max int, onlyAllowRules bool) []SecurityRule {
	if defaultRule.String() == localRuleStr {
		return rules
	}
	defaultRule.ExternalId = DEFAULT_CLOUD_RULE_ID
	if max > min {
		defaultRule.Priority = min - 1
	} else {
		defaultRule.Priority = max + 1
	}
	return append(rules, defaultRule)
}

func SortSecurityRule(rules SecurityRuleSet, max, min int, onlyAllowRules bool) {
	if onlyAllowRules {
		return
	}
	if max < min {
		sort.Sort(sort.Reverse(rules))
		return
	}
	sort.Sort(rules)
}

func CompareRules(
	minPriority, maxPriority int,
	localRules LocalSecurityRuleSet, remoteRules []SecurityRule,
	defaultInRule, defaultOutRule SecurityRule,
	onlyAllowRules bool, debug bool, refreshLocalRule bool,
) (common, inAdds, outAdds, inDels, outDels []SecurityRule) {
	localInRules := LocalSecurityRuleSet{}
	localOutRules := LocalSecurityRuleSet{}
	for i := range localRules {
		if localRules[i].Direction == secrules.DIR_IN {
			localInRules = append(localInRules, localRules[i])
		} else {
			localOutRules = append(localOutRules, localRules[i])
		}
	}
	inRules := SecurityRuleSet{}
	outRules := SecurityRuleSet{}
	for i := 0; i < len(remoteRules); i++ {
		if remoteRules[i].Direction == secrules.DIR_IN {
			inRules = append(inRules, remoteRules[i])
		} else {
			outRules = append(outRules, remoteRules[i])
		}
	}
	var inCommon, outCommon = inRules, outRules

	defaultLocalInRule := LocalSecurityRule{ExternalId: DEFAULT_LOCAL_RULE_ID}
	defaultLocalInRule.SecurityRule = *secrules.MustParseSecurityRule("in:deny any")
	defaultLocalOutRule := LocalSecurityRule{ExternalId: DEFAULT_LOCAL_RULE_ID}
	defaultLocalOutRule.SecurityRule = *secrules.MustParseSecurityRule("out:allow any")

	inRules = AddDefaultRule(inRules, defaultInRule, defaultLocalInRule.String(), minPriority, maxPriority, onlyAllowRules)
	outRules = AddDefaultRule(outRules, defaultOutRule, defaultLocalOutRule.String(), minPriority, maxPriority, onlyAllowRules)

	defaultInEquals, defaultOutEquals := true, true
	if defaultLocalInRule.String() != defaultInRule.String() {
		localInRules = append(localInRules, defaultLocalInRule)
		defaultInEquals = false
	}
	if defaultLocalOutRule.String() != defaultOutRule.String() {
		localOutRules = append(localOutRules, defaultLocalOutRule)
		defaultOutEquals = false
	}

	sort.Sort(localInRules)
	sort.Sort(localOutRules)

	localInAllowList := localInRules.AllowList()
	localOutAllowList := localOutRules.AllowList()
	_localInRules := LocalSecurityRuleSet{}
	for i := range localInAllowList {
		rule := LocalSecurityRule{}
		rule.SecurityRule = localInAllowList[i]
		_localInRules = append(_localInRules, rule)
	}
	_localOutRules := LocalSecurityRuleSet{}
	for i := range localOutAllowList {
		rule := LocalSecurityRule{}
		rule.SecurityRule = localOutAllowList[i]
		_localOutRules = append(_localOutRules, rule)
	}
	if !refreshLocalRule && onlyAllowRules {
		localOutRules, localInRules = _localOutRules, _localInRules
	}
	if !refreshLocalRule && len(_localInRules) < len(localInRules) {
		localInRules = _localInRules
	}
	if !refreshLocalRule && len(_localOutRules) < len(localOutRules) {
		localOutRules = _localOutRules
	}

	SortSecurityRule(inRules, maxPriority, minPriority, onlyAllowRules)
	SortSecurityRule(outRules, maxPriority, minPriority, onlyAllowRules)

	inAllowList := inRules.AllowList()
	outAllowList := outRules.AllowList()
	inEquals, outEquals := inAllowList.Equals(localInAllowList), outAllowList.Equals(localOutAllowList)
	if inEquals && outEquals {
		return
	}

	// priority从小到大排列(从默认规则开始对比)
	sort.Sort(sort.Reverse(localInRules))
	sort.Sort(sort.Reverse(localOutRules))

	sort.Sort(sort.Reverse(inRules))
	sort.Sort(sort.Reverse(outRules))

	startPriority := minPriority - 1
	if maxPriority < minPriority {
		startPriority = maxPriority + 1
	}

	var addPriority = func(priority int, inc int, min, max int, onlyAllowRules bool) int {
		if onlyAllowRules {
			return priority
		}
		if max < min {
			max, min, inc = min, max, inc*-1
		}
		if priority >= max || priority <= min {
			return priority
		}
		return priority + inc
	}

	var getInitPriority = func(init, min, max int) int {
		if init < min || init > max {
			return (min + max) / 2
		}
		return init
	}

	var compare = func(localRules LocalSecurityRuleSet, remoteRules SecurityRuleSet) (common, add, del []SecurityRule) {
		i, j, inc, prePriority, localPrority := 0, 0, 1, 0, 50
		for i < len(localRules) || j < len(remoteRules) {
			if i < len(localRules) && j < len(remoteRules) {
				ruleStr := remoteRules[j].String()
				localRuleStr := localRules[i].String()
				if debug {
					log.Debugf("compare local priority(%d) %s -> remote name(%s) priority(%d) %s\n", localRules[i].Priority, localRules[i].String(), remoteRules[j].Name, remoteRules[j].Priority, remoteRules[j].String())
				}
				cmp := strings.Compare(ruleStr, localRuleStr)
				if cmp == 0 {
					prePriority = remoteRules[j].Priority
					if remoteRules[j].ExternalId == DEFAULT_CLOUD_RULE_ID {
						remoteRules[j].Priority = addPriority(remoteRules[j].Priority, 1, minPriority, maxPriority, onlyAllowRules)
					}
					if localRules[i].ExternalId != DEFAULT_LOCAL_RULE_ID ||
						(localRules[i].Direction == secrules.DIR_IN && !defaultInEquals) ||
						(localRules[i].Direction == secrules.DIR_OUT && !defaultOutEquals) {
						common = append(common, remoteRules[j])
					}
					localPrority = localRules[i].Priority
					i++
					j++
				} else if cmp < 0 {
					if remoteRules[j].ExternalId != DEFAULT_CLOUD_RULE_ID {
						remoteRules[j].LocalRulePrority = localPrority
						localPrority = addPriority(localPrority, 1, 1, 100, false)
						del = append(del, remoteRules[j])
					}
					j++
				} else {
					initPriority := getInitPriority(prePriority, minPriority, maxPriority)
					localRules[i].Priority = addPriority(initPriority, inc, minPriority, maxPriority, onlyAllowRules)
					if localRules[i].ExternalId != DEFAULT_LOCAL_RULE_ID ||
						(localRules[i].Direction == secrules.DIR_IN && !defaultInEquals) ||
						(localRules[i].Direction == secrules.DIR_OUT && !defaultOutEquals) {
						add = append(add, SecurityRule{SecurityRule: localRules[i].SecurityRule, ExternalId: localRules[i].ExternalId})
					}
					i++
					inc++
				}
			} else if i >= len(localRules) {
				if remoteRules[j].ExternalId != DEFAULT_CLOUD_RULE_ID {
					remoteRules[j].LocalRulePrority = localPrority
					localPrority = addPriority(localPrority, 1, 1, 100, false)
					del = append(del, remoteRules[j])
				}
				j++
			} else if j >= len(remoteRules) {
				initPriority := startPriority
				if len(remoteRules) > 0 {
					initPriority = remoteRules[len(remoteRules)-1].Priority
				}
				initPriority = getInitPriority(initPriority, minPriority, maxPriority) // 若是初始添加规则，尽量以中间为节点，避免仅出现天地规则
				localRules[i].Priority = addPriority(initPriority, inc, minPriority, maxPriority, onlyAllowRules)
				if localRules[i].ExternalId != DEFAULT_LOCAL_RULE_ID ||
					(localRules[i].Direction == secrules.DIR_IN && !defaultInEquals) ||
					(localRules[i].Direction == secrules.DIR_OUT && !defaultOutEquals) {
					add = append(add, SecurityRule{SecurityRule: localRules[i].SecurityRule, ExternalId: localRules[i].ExternalId})
				}
				i++
				inc++
			}
		}
		return
	}

	type rulePair struct {
		localRules  LocalSecurityRuleSet
		remoteRules []SecurityRule
		protocol    string
	}

	var splitRules = func(localRules LocalSecurityRuleSet, remoteRules []SecurityRule) []rulePair {
		rules := map[string]rulePair{}
		for _, r := range localRules {
			pair, ok := rules[r.Protocol]
			if !ok {
				pair = rulePair{localRules: LocalSecurityRuleSet{}, remoteRules: []SecurityRule{}, protocol: r.Protocol}
			}
			pair.localRules = append(pair.localRules, r)
			rules[r.Protocol] = pair
		}

		for _, r := range remoteRules {
			pair, ok := rules[r.Protocol]
			if !ok {
				pair = rulePair{localRules: LocalSecurityRuleSet{}, remoteRules: []SecurityRule{}, protocol: r.Protocol}
			}
			pair.remoteRules = append(pair.remoteRules, r)
			rules[r.Protocol] = pair
		}

		ret := []rulePair{}
		for _, r := range rules {
			ret = append(ret, r)
		}
		return ret
	}

	var compareRules = func(localRules LocalSecurityRuleSet, remoteRules []SecurityRule) (common, add, dels []SecurityRule) {
		pairs := splitRules(localRules, remoteRules)
		for _, r := range pairs {
			_common, _add, _dels := compare(r.localRules, r.remoteRules)
			common = append(common, _common...)
			add = append(add, _add...)
			dels = append(dels, _dels...)
		}
		return
	}

	if !inEquals {
		inCommon, inAdds, inDels = compareRules(localInRules, inRules)
	}
	if !outEquals {
		outCommon, outAdds, outDels = compareRules(localOutRules, outRules)
	}
	common = append(inCommon, outCommon...)
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
