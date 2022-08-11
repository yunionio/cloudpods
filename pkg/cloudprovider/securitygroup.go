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

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/secrules"
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

func NewSecRuleInfo(driver SecDriver) *SecRuleInfo {
	ret := &SecRuleInfo{
		MinPriority:             driver.GetSecurityGroupRuleMinPriority(),
		MaxPriority:             driver.GetSecurityGroupRuleMaxPriority(),
		IsOnlySupportAllowRules: driver.IsOnlySupportAllowRules(),
		IsSupportPeerSecgroup:   driver.IsSupportPeerSecgroup(),
		Rules:                   SecurityRuleSet{},
	}
	defaultInRule := driver.GetDefaultSecurityGroupInRule()
	defaultInRule.Priority = ret.defaultRulePriority()
	defaultOutRule := driver.GetDefaultSecurityGroupOutRule()
	defaultOutRule.Priority = ret.defaultRulePriority()

	ret.inDefaultRule = &defaultInRule
	ret.outDefaultRule = &defaultOutRule
	return ret
}

const DEFAULT_DEST_RULE_ID = "default_dest_rule_id"
const DEFAULT_SRC_RULE_ID = "default_src_rule_id"

const (
	RULE_ROLE_SRC  = "src"
	RULE_ROLE_DEST = "dest"
)

type SecRuleInfo struct {
	role                    string
	inDefaultRule           *SecurityRule
	outDefaultRule          *SecurityRule
	in                      SecurityRuleSet
	inWide                  bool
	inPeer                  bool
	out                     SecurityRuleSet
	outWide                 bool
	outPeer                 bool
	rules                   SecurityRuleSet
	Rules                   SecurityRuleSet
	MinPriority             int
	MaxPriority             int
	IsOnlySupportAllowRules bool
	IsSupportPeerSecgroup   bool
}

func (self *SecRuleInfo) defaultRulePriority() int {
	if self.MaxPriority == self.MinPriority {
		return self.MinPriority
	}
	ret := self.MinPriority + 1
	if self.MaxPriority > self.MinPriority {
		ret = self.MinPriority - 1
	}
	return ret
}

func (self *SecRuleInfo) setRole(role string) {
	self.role = role
	self.inDefaultRule.Id, self.inDefaultRule.ExternalId = DEFAULT_SRC_RULE_ID, DEFAULT_SRC_RULE_ID
	self.outDefaultRule.Id, self.outDefaultRule.ExternalId = DEFAULT_SRC_RULE_ID, DEFAULT_SRC_RULE_ID
	if self.role == RULE_ROLE_DEST {
		self.inDefaultRule.Id, self.inDefaultRule.ExternalId = DEFAULT_DEST_RULE_ID, DEFAULT_DEST_RULE_ID
		self.outDefaultRule.Id, self.outDefaultRule.ExternalId = DEFAULT_DEST_RULE_ID, DEFAULT_DEST_RULE_ID
	}
}

func (self *SecRuleInfo) addDefaultRule() {
	if !self.inWide {
		self.in = append(self.in, *self.inDefaultRule)
	}
	if !self.outWide {
		self.out = append(self.out, *self.outDefaultRule)
	}
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

	secrules.SecurityRule
	Name       string
	ExternalId string
	Id         string

	PeerSecgroupId string
}

const (
	PRIORITY_ORDER_ASC  = "asc"
	PRIORITY_ORDER_DESC = "desc"
)

func (self *SecurityRule) GetPriorityOrider() string {
	if self.minPriority > self.maxPriority {
		return PRIORITY_ORDER_ASC
	}
	return PRIORITY_ORDER_DESC
}

func (r SecurityRule) String() string {
	if len(r.PeerSecgroupId) == 0 {
		return r.SecurityRule.String()
	}
	return fmt.Sprintf("%s-%s", r.SecurityRule.String(), r.PeerSecgroupId)
}

type SecurityRuleSet []SecurityRule

func (self SecurityRuleSet) asc() {
	sort.Sort(self)
}

func (self SecurityRuleSet) desc() {
	sort.Sort(sort.Reverse(self))
}

func (srs SecurityRuleSet) equals(srs1 SecurityRuleSet) bool {
	srs.desc()
	srs1.desc()
	srsRules := secrules.SecurityRuleSet{}
	for i := range srs {
		srsRules = append(srsRules, srs[i].SecurityRule)
	}
	srsAllowList := srsRules.AllowList()
	srs1Rules := secrules.SecurityRuleSet{}
	for i := range srs1 {
		srs1Rules = append(srs1Rules, srs1[i].SecurityRule)
	}
	srs1AllowList := srs1Rules.AllowList()
	return srsAllowList.Equals(srs1AllowList)
}

func (self *SecRuleInfo) init(role string) {
	self.setRole(role)
	self.split()
	self.addDefaultRule()
}

func (self *SecRuleInfo) split() {
	self.rules = SecurityRuleSet{}
	self.in = SecurityRuleSet{}
	self.out = SecurityRuleSet{}
	for i := 0; i < len(self.Rules); i++ {
		self.Rules[i].minPriority, self.Rules[i].maxPriority = self.MinPriority, self.MaxPriority
		self.rules = append(self.rules, self.Rules[i])
		if self.Rules[i].Direction == secrules.DIR_IN {
			if len(self.Rules[i].PeerSecgroupId) == 0 && self.Rules[i].String() == "in:allow any" {
				self.inWide = true
			}
			if len(self.Rules[i].PeerSecgroupId) > 0 {
				self.inPeer = true
			}

			self.in = append(self.in, self.Rules[i])
			continue
		}
		self.out = append(self.out, self.Rules[i])
		if len(self.Rules[i].PeerSecgroupId) == 0 && self.Rules[i].String() == "out:allow any" {
			self.outWide = true
		}
		if len(self.Rules[i].PeerSecgroupId) > 0 {
			self.inPeer = true
		}
	}
}

func (srs SecurityRuleSet) Len() int {
	return len(srs)
}

func (srs SecurityRuleSet) Swap(i, j int) {
	srs[i], srs[j] = srs[j], srs[i]
}

func (srs SecurityRuleSet) Less(i, j int) bool {
	if srs[i].minPriority == srs[i].minPriority {
		return srs[i].String() < srs[j].String()
	}
	if srs[i].minPriority < srs[i].maxPriority {
		if srs[i].Priority != srs[j].Priority {
			return srs[i].Priority < srs[j].Priority
		}
		return srs[i].String() < srs[j].String()
	}
	if srs[i].Priority != srs[j].Priority {
		return srs[i].Priority > srs[j].Priority
	}
	return srs[i].String() > srs[j].String()
}

func (srs SecurityRuleSet) Debug() {
	for i := 0; i < len(srs); i++ {
		log.Debugf("Name: %s id: %s external_id: %s min: %d max: %d priority: %d %s", srs[i].Name, srs[i].Id, srs[i].ExternalId, srs[i].minPriority, srs[i].maxPriority, srs[i].Priority, srs[i].String())
	}
}

func CompareRules(src, dest *SecRuleInfo, debug bool) (SecurityRuleSet, SecurityRuleSet, SecurityRuleSet, SecurityRuleSet, SecurityRuleSet) {
	var common, inAdds, outAdds, inDels, outDels SecurityRuleSet

	src.init(RULE_ROLE_SRC)
	dest.init(RULE_ROLE_DEST)

	var compare = func(srcRules SecurityRuleSet, destRules SecurityRuleSet) (SecurityRuleSet, SecurityRuleSet, SecurityRuleSet) {
		var _common, _add, _del SecurityRuleSet
		defer func() {
			var resetPriority = func(rules SecurityRuleSet) SecurityRuleSet {
				ret := SecurityRuleSet{}
				for i := range rules {
					rules[i].minPriority, rules[i].maxPriority = 0, 0
					ret = append(ret, rules[i])
				}
				return ret
			}
			resetPriority(_common)
			resetPriority(_add)
			resetPriority(_del)
		}()
		var isPriorityChanged = func(old, new, min, max int) (int, bool) {
			if old == -1 || old == new {
				return 0, false
			}
			if max > min {
				return 1, true
			}
			return -1, true
		}
		i, j, srcPriority, destPriority := 0, 0, -1, -1
		for i < len(srcRules) || j < len(destRules) {
			if i < len(srcRules) && j < len(destRules) {
				destRuleStr := destRules[j].String()
				srcRuleStr := srcRules[i].String()
				if debug {
					log.Debugf("compare src %s(%s) priority(%d) %s -> dest name(%s) %s(%s) priority(%d) %s\n",
						srcRules[i].Id, srcRules[i].ExternalId, srcRules[i].Priority, srcRules[i].String(),
						destRules[j].Name, destRules[j].ExternalId, destRules[j].Id, destRules[j].Priority, destRules[j].String())
				}
				if destRuleStr == srcRuleStr {
					destRules[j].Id = srcRules[i].Id
					if srcRules[i].Id != DEFAULT_SRC_RULE_ID {
						_common = append(_common, destRules[j])
						destPriority = destRules[j].Priority
						srcPriority = srcRules[i].Priority
					}
					i++
					j++
					continue
				}
				if destRules[j].Id != DEFAULT_DEST_RULE_ID {
					destPriority = destRules[j].Priority
					_del = append(_del, destRules[j])
				}
				j++
			} else if i >= len(srcRules) {
				_del = append(_del, destRules[j])
				destPriority = destRules[j].Priority
				j++
			} else if j >= len(destRules) {
				offset, isChanged := isPriorityChanged(srcPriority, srcRules[i].Priority, src.MinPriority, src.MaxPriority)
				if srcRules[i].Id == DEFAULT_SRC_RULE_ID {
					srcRules[i].Priority = dest.MinPriority
				} else {
					srcPriority = srcRules[i].Priority
					srcRules[i].Priority = destPriority
					if isChanged {
						srcRules[i].Priority = destPriority + offset
					}
				}
				log.Infof("add: %s", srcRules[i].String())
				_add = append(_add, srcRules[i])
				i++
			}
		}
		return _common, _add, _del
	}

	if !src.inPeer && !dest.inPeer && src.in.equals(dest.in) {
		for i := range dest.in {
			if dest.in[i].Id != DEFAULT_DEST_RULE_ID {
				common = append(common, dest.in[i])
			}
		}
	} else {
		src.in.asc()
		dest.in.asc()
		_inCommon, _inAdd, _inDel := compare(src.in, dest.in)
		common = append(common, _inCommon...)
		inAdds = append(inAdds, _inAdd...)
		inDels = append(inDels, _inDel...)
	}

	if !src.outPeer && !dest.outPeer && src.out.equals(dest.out) {
		for i := range dest.out {
			if dest.out[i].Id != DEFAULT_DEST_RULE_ID {
				common = append(common, dest.out[i])
			}
		}
	} else {
		src.out.asc()
		dest.out.asc()
		_outCommon, _outAdd, _outDel := compare(src.out, dest.out)
		common = append(common, _outCommon...)
		outAdds = append(outAdds, _outAdd...)
		outDels = append(outDels, _outDel...)
	}
	return common, inAdds, outAdds, inDels, outDels
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
