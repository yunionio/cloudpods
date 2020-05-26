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

type SecurityGroupCreateInput struct {
	Name  string
	Desc  string
	VpcId string
	Rules []secrules.SecurityRule
}

type SecurityRule struct {
	secrules.SecurityRule
	Name       string
	ExternalId string
}

type TPriorityOrder int

var (
	PriorityOrderByDesc = TPriorityOrder(-1)
	PriorityOrderByAsc  = TPriorityOrder(1)
)

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

func AddDefaultRule(rules []SecurityRule, defaultRule SecurityRule, localRuleStr string, order TPriorityOrder, min, max int, onlyAllowRules bool) []SecurityRule {
	if defaultRule.String() == localRuleStr {
		return rules
	}
	defaultRule.ExternalId = DEFAULT_CLOUD_RULE_ID
	if order == PriorityOrderByDesc {
		defaultRule.Priority = min
	} else {
		defaultRule.Priority = max
	}
	defaultRule.Priority += int(order)
	if onlyAllowRules {
		defaultRule.Priority = -1
	}
	return append(rules, defaultRule)
}

func SortSecurityRule(rules SecurityRuleSet, order TPriorityOrder, onlyAllowRules bool) {
	if order == PriorityOrderByDesc || onlyAllowRules {
		sort.Sort(rules)
	} else {
		sort.Sort(sort.Reverse(rules))
	}
}

func CompareRules(
	minPriority, maxPriority int, order TPriorityOrder,
	localRules secrules.SecurityRuleSet, remoteRules []SecurityRule,
	defaultInRule, defaultOutRule SecurityRule,
	onlyAllowRules bool, debug bool,
) (common, inAdds, outAdds, inDels, outDels []SecurityRule) {
	localInRules := secrules.SecurityRuleSet{}
	localOutRules := secrules.SecurityRuleSet{}
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

	defaultLocalInRule := *secrules.MustParseSecurityRule("in:deny any")
	defaultLocalOutRule := *secrules.MustParseSecurityRule("out:allow any")

	inRules = AddDefaultRule(inRules, defaultInRule, defaultLocalInRule.String(), order, minPriority, maxPriority, onlyAllowRules)
	outRules = AddDefaultRule(outRules, defaultOutRule, defaultLocalOutRule.String(), order, minPriority, maxPriority, onlyAllowRules)

	if defaultLocalInRule.String() != defaultInRule.String() {
		localInRules = append(localInRules, defaultLocalInRule)
	}
	if defaultLocalOutRule.String() != defaultOutRule.String() {
		localOutRules = append(localOutRules, defaultLocalOutRule)
	}

	sort.Sort(localInRules)
	sort.Sort(localOutRules)

	localInAllowList := localInRules.AllowList()
	localOutAllowList := localOutRules.AllowList()
	if onlyAllowRules {
		localInRules = localInAllowList
		localOutRules = localOutAllowList
	}

	SortSecurityRule(inRules, order, onlyAllowRules)
	SortSecurityRule(outRules, order, onlyAllowRules)

	inAllowList := inRules.AllowList()
	outAllowList := outRules.AllowList()
	if inAllowList.Equals(localInAllowList) && outAllowList.Equals(localOutAllowList) {
		return
	}

	startPriority := minPriority - 1
	if order == PriorityOrderByAsc {
		startPriority = maxPriority + 1
	}

	var addPriority = func(priority int, order TPriorityOrder, inc int, min, max int, onlyAllowRules bool) int {
		if onlyAllowRules {
			return 0
		}
		inc = inc*int(order) + int(order)
		priority += inc
		if priority < min {
			return min
		}
		if priority > max {
			return max
		}
		return priority
	}

	var compare = func(localRules []secrules.SecurityRule, remoteRules []SecurityRule) (common, add, del []SecurityRule) {
		i, j, inc := 0, 0, 0
		for i < len(localRules) || j < len(remoteRules) {
			if i < len(localRules) && j < len(remoteRules) {
				ruleStr := remoteRules[j].String()
				localRuleStr := localRules[i].String()
				if debug {
					log.Debugf("compare local priority(%d) %s -> remote name(%s) priority(%d) %s\n", localRules[i].Priority, localRules[i].String(), remoteRules[j].Name, remoteRules[j].Priority, remoteRules[j].String())
				}
				cmp := strings.Compare(ruleStr, localRuleStr)
				if cmp == 0 {
					common = append(common, remoteRules[j])
					i++
					j++
				} else if cmp > 0 {
					if remoteRules[j].ExternalId != DEFAULT_CLOUD_RULE_ID {
						del = append(del, remoteRules[j])
					}
					j++
				} else {
					idx := j
					if idx >= 1 {
						idx--
					}
					localRules[i].Priority = addPriority(remoteRules[idx].Priority, order, 0, minPriority, maxPriority, onlyAllowRules)
					add = append(add, SecurityRule{SecurityRule: localRules[i]})
					i++
				}
			} else if i >= len(localRules) {
				if remoteRules[j].ExternalId != DEFAULT_CLOUD_RULE_ID {
					del = append(del, remoteRules[j])
				}
				j++
			} else if j >= len(remoteRules) {
				initPriority := startPriority
				if len(remoteRules) > 0 {
					initPriority = remoteRules[len(remoteRules)-1].Priority
				}
				if initPriority < minPriority || initPriority > maxPriority { // 若是初始添加规则，尽量以中间为节点，避免仅出现天地规则
					initPriority = (minPriority + maxPriority) / 2
				}
				localRules[i].Priority = addPriority(initPriority, order, inc, minPriority, maxPriority, onlyAllowRules)
				add = append(add, SecurityRule{SecurityRule: localRules[i]})
				i++
				inc++
			}
		}
		return
	}
	var inCommon, outCommon = []SecurityRule{}, []SecurityRule{}
	inCommon, inAdds, inDels = compare(localInRules, inRules)
	outCommon, outAdds, outDels = compare(localOutRules, outRules)
	common = append(inCommon, outCommon...)
	return
}
