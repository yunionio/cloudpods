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
	"fmt"
	"sort"
	"strings"

	"gopkg.in/fatih/set.v0"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/utils"
)

type SecurityGroupReference struct {
	Id   string
	Name string
}

type SecDriver interface {
	GetDefaultSecurityGroupInRule() SecurityRule
	GetDefaultSecurityGroupOutRule() SecurityRule
	GetSecurityGroupRuleMaxPriority() int
	GetSecurityGroupRuleMinPriority() int
	IsOnlySupportAllowRules() bool
	IsSupportPeerSecgroup() bool
}

func NewSecRuleInfo(driver SecDriver) SecRuleInfo {
	return SecRuleInfo{
		InDefaultRule:           driver.GetDefaultSecurityGroupInRule(),
		OutDefaultRule:          driver.GetDefaultSecurityGroupOutRule(),
		MinPriority:             driver.GetSecurityGroupRuleMinPriority(),
		MaxPriority:             driver.GetSecurityGroupRuleMaxPriority(),
		IsOnlySupportAllowRules: driver.IsOnlySupportAllowRules(),
		IsSupportPeerSecgroup:   driver.IsSupportPeerSecgroup(),
	}
}

const DEFAULT_DEST_RULE_ID = "default_dest_rule_id"
const DEFAULT_SRC_RULE_ID = "default_src_rule_id"

type SecRuleInfo struct {
	InDefaultRule           SecurityRule
	OutDefaultRule          SecurityRule
	in                      SecurityRuleSet
	out                     SecurityRuleSet
	rules                   SecurityRuleSet
	Rules                   SecurityRuleSet
	MinPriority             int
	MaxPriority             int
	IsOnlySupportAllowRules bool
	IsSupportPeerSecgroup   bool
}

func (r SecRuleInfo) getOffest(priority int) (int, int) {
	if r.MinPriority < r.MaxPriority {
		if priority >= r.MaxPriority {
			return priority, -1
		}
		return priority + 1, 0
	} else {
		if priority <= r.MaxPriority {
			return priority, 1
		}
		return priority - 1, 0
	}
}

func (r SecRuleInfo) AddDefaultRule(d SecRuleInfo, inRules, outRules []SecurityRule, isSrc bool) ([]SecurityRule, []SecurityRule) {
	min, max := r.MinPriority, r.MaxPriority
	r.InDefaultRule.Priority = min + 1
	r.OutDefaultRule.Priority = min + 1
	r.InDefaultRule.minPriority, r.InDefaultRule.maxPriority = r.MinPriority, r.MaxPriority
	r.OutDefaultRule.minPriority, r.OutDefaultRule.maxPriority = r.MinPriority, r.MaxPriority
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

	var isWideRule = func(rules []SecurityRule) bool {
		for _, rule := range rules {
			if strings.HasSuffix(rule.String(), "allow any") || strings.HasSuffix(rule.String(), "deny any") {
				return true
			}
		}
		return false
	}

	if !isWideRule(inRules) {
		inRules = append(inRules, r.InDefaultRule)
	}
	if !isWideRule(outRules) {
		outRules = append(outRules, r.OutDefaultRule)
	}
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
	minPriority int
	maxPriority int
	offset      int

	secrules.SecurityRule
	Name       string
	ExternalId string
	Id         string

	PeerSecgroupId string
}

func (self *SecurityRule) addPriority(offset int) {
	self.Priority += offset
	self.offset += offset
}

func (r SecurityRule) String() string {
	if len(r.PeerSecgroupId) == 0 {
		return r.SecurityRule.String()
	}
	return fmt.Sprintf("%s-%s", r.SecurityRule.String(), r.PeerSecgroupId)
}

type SecurityRuleSet []SecurityRule

func (self *SecRuleInfo) Split() (in, out SecurityRuleSet) {
	self.rules = SecurityRuleSet{}
	self.in = SecurityRuleSet{}
	self.out = SecurityRuleSet{}
	for i := 0; i < len(self.Rules); i++ {
		self.rules = append(self.rules, self.Rules[i])
		if self.Rules[i].Direction == secrules.DIR_IN {
			self.in = append(self.in, self.Rules[i])
		} else {
			self.out = append(self.out, self.Rules[i])
		}
		self.Rules[i].minPriority, self.Rules[i].maxPriority = self.MinPriority, self.MaxPriority
		if !self.IsSupportPeerSecgroup && len(self.Rules[i].PeerSecgroupId) > 0 {
			continue
		}
		if self.Rules[i].Direction == secrules.DIR_IN {
			in = append(in, self.Rules[i])
		} else {
			out = append(out, self.Rules[i])
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
	if srs[i].Priority < srs[j].Priority {
		return true
	}
	if srs[i].Priority > srs[j].Priority {
		return false
	}
	if len(srs) > 0 {
		if (srs[0].minPriority <= srs[0].maxPriority && srs[i].String() < srs[j].String()) ||
			(srs[0].minPriority > srs[0].maxPriority && srs[i].String() > srs[j].String()) {
			return true
		}
	}
	return false
}

func (srs SecurityRuleSet) CanBeSplitByProtocol() bool {
	firstNormalRuleIndex, find := 0, false
	for idx, r := range srs {
		if !(strings.HasSuffix(r.String(), "allow any") || strings.HasSuffix(r.String(), "deny any")) && !find {
			firstNormalRuleIndex = idx
			find = true
		} else {
			if r.ExternalId == DEFAULT_SRC_RULE_ID || r.ExternalId == DEFAULT_DEST_RULE_ID {
				return true
			}
			if idx > firstNormalRuleIndex {
				return false
			}
		}
	}
	return true
}

func (srs SecurityRuleSet) AllowList() (secrules.SecurityRuleSet, bool) {
	rules := secrules.SecurityRuleSet{}
	isOk := true
	for _, r := range srs {
		if len(r.PeerSecgroupId) > 0 {
			isOk = false
		}
		rules = append(rules, r.SecurityRule)
	}
	return rules.AllowList(), isOk
}

func (srs SecurityRuleSet) Debug() {
	for i := 0; i < len(srs); i++ {
		log.Debugf("Name: %s id: %s external_id: %s min: %d max: %d priority: %d %s", srs[i].Name, srs[i].Id, srs[i].ExternalId, srs[i].minPriority, srs[i].maxPriority, srs[i].Priority, srs[i].String())
	}
}

func SortSecurityRule(rules SecurityRuleSet, isAsc bool, info SecRuleInfo) {
	if (info.MaxPriority > info.MinPriority && isAsc) || (info.MaxPriority < info.MinPriority && !isAsc) || (info.IsOnlySupportAllowRules && isAsc) {
		sort.Sort(rules)
		return
	}
	sort.Sort(sort.Reverse(rules))
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

func isPeerListEqual(src, dest SecurityRuleSet) bool {
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
	srcInRules, srcOutRules := src.Split()
	destInRules, destOutRules := dest.Split()

	srcInRules, srcOutRules = src.AddDefaultRule(dest, srcInRules, srcOutRules, true)
	destInRules, destOutRules = dest.AddDefaultRule(src, destInRules, destOutRules, false)

	// AllowList 需要优先级从高到低排序
	SortSecurityRule(srcInRules, false, src)
	SortSecurityRule(srcOutRules, false, src)

	SortSecurityRule(destInRules, false, dest)
	SortSecurityRule(destOutRules, false, dest)

	isInAllowSplit := srcInRules.CanBeSplitByProtocol() && destInRules.CanBeSplitByProtocol()
	isOutAllowSplit := srcOutRules.CanBeSplitByProtocol() && destOutRules.CanBeSplitByProtocol()

	srcInAllowList, srcInOk := srcInRules.AllowList()
	srcOutAllowList, srcOutOk := srcOutRules.AllowList()

	destInAllowList, destInOk := destInRules.AllowList()
	destOutAllowList, destOutOk := destOutRules.AllowList()
	inEquals, outEquals, inOk, outOk := isAllowListEqual(srcInAllowList, destInAllowList), isAllowListEqual(srcOutAllowList, destOutAllowList), srcInOk && destInOk, srcOutOk && destOutOk

	if inEquals && outEquals && inOk && outOk {
		common = dest.rules
		return
	}

	if debug {
		log.Debugf("====desc sort====")
		log.Debugf("src in rules: ")
		srcInRules.Debug()
		log.Debugf("src out rules: ")
		srcOutRules.Debug()
		log.Debugf("dest in rules: ")
		destInRules.Debug()
		log.Debugf("dest out rules: ")
		destOutRules.Debug()
		log.Debugf("====desc sort end====")

		log.Debugf("isInAllowSplit: %v isOutAllowSplit: %v", isInAllowSplit, isOutAllowSplit)

		log.Debugf("AllowList:")
		log.Debugf("In: src: %s dest: %s isEquals: %v", srcInAllowList.String(), destInAllowList.String(), inEquals)
		log.Debugf("Out: src: %s dest: %s isEquals: %v", srcOutAllowList.String(), destOutAllowList.String(), outEquals)
	}

	var tryUseAllowList = func(defaultRule SecurityRule, allowList secrules.SecurityRuleSet, rules SecurityRuleSet, isOnlyAllowList bool, isOk bool) SecurityRuleSet {
		if isOk && (len(allowList) < len(rules) || isOnlyAllowList) {
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

	srcInRules = tryUseAllowList(src.InDefaultRule, srcInAllowList, srcInRules, dest.IsOnlySupportAllowRules, inOk)
	srcOutRules = tryUseAllowList(src.OutDefaultRule, srcOutAllowList, srcOutRules, dest.IsOnlySupportAllowRules, outOk)

	if inEquals && inOk {
		common = append(common, dest.in...)
		srcInRules, destInRules = []SecurityRule{}, []SecurityRule{}
	}
	if outEquals && outOk {
		common = append(common, dest.out...)
		srcOutRules, destOutRules = []SecurityRule{}, []SecurityRule{}
	}

	// 默认从优先级低到高比较
	SortSecurityRule(srcInRules, true, src)
	SortSecurityRule(srcOutRules, true, src)

	SortSecurityRule(destInRules, true, dest)
	SortSecurityRule(destOutRules, true, dest)

	if debug {
		log.Debugf("====asc sort====")
		log.Debugf("src in rules: ")
		srcInRules.Debug()
		log.Debugf("src out rules: ")
		srcOutRules.Debug()
		log.Debugf("dest in rules: ")
		destInRules.Debug()
		log.Debugf("dest out rules: ")
		destOutRules.Debug()
		log.Debugf("====asc sort end====")
	}

	var _compare = func(srcRules SecurityRuleSet, destRules SecurityRuleSet) (common, add, del SecurityRuleSet) {
		i, j, priority := 0, 0, dest.MinPriority
		if len(destRules) > 0 && (destRules[i].ExternalId != DEFAULT_DEST_RULE_ID) {
			priority = destRules[0].Priority
		}
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
					offset := 0
					if !dest.IsOnlySupportAllowRules && i-1 >= 0 && srcRules[i-1].Priority != srcRules[i].Priority {
						priority, offset = dest.getOffest(priority)
					}
					if offset != 0 {
						for r := range add {
							add[r].addPriority(offset)
						}
						for r := range common {
							common[r].addPriority(offset)
						}
					}
					srcRules[i].Priority = priority
					srcRules[i].minPriority, srcRules[i].maxPriority = dest.MinPriority, dest.MaxPriority
					add = append(add, srcRules[i])
					i++
				}
			} else if i >= len(srcRules) {
				del = append(del, destRules[j])
				j++
			} else if j >= len(destRules) {
				offset := 0
				if !dest.IsOnlySupportAllowRules && i-1 >= 0 && srcRules[i-1].Priority != srcRules[i].Priority {
					priority, offset = dest.getOffest(priority)
				}
				srcRules[i].Priority = priority
				srcRules[i].minPriority, srcRules[i].maxPriority = dest.MinPriority, dest.MaxPriority
				if offset != 0 {
					for r := range add {
						add[r].addPriority(offset)
					}
					for r := range common {
						common[r].addPriority(offset)
					}
				}
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
	if isInAllowSplit {
		inCommon, inAdds, inDels = compare(srcInRules, destInRules)
	} else {
		inCommon, inAdds, inDels = _compare(srcInRules, destInRules)
	}
	if isOutAllowSplit {
		outCommon, outAdds, outDels = compare(srcOutRules, destOutRules)
	} else {
		outCommon, outAdds, outDels = _compare(srcOutRules, destOutRules)
	}

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
	_common, _ := handleDefaultRules(append(inCommon, outCommon...), []SecurityRule{}, dest.IsOnlySupportAllowRules)
	common = append(common, _common...)
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
