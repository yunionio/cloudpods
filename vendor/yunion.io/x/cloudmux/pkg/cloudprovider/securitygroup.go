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

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"
)

type SecurityGroupReference struct {
	Id   string
	Name string
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
	// 安全组刚创建, 但是未设置安全组规则时
	OnCreated func(id string)

	// 默认优先级从低到高排序, 端口会被分离, 不会出现[10,20]这种规则
	InRules SecurityRuleSet
	// 同上
	OutRules SecurityRuleSet
}

type SecurityRule struct {
	secrules.SecurityRule
	Name string
	Id   string
}

type SecurityRuleSet []SecurityRule

func (srs SecurityRuleSet) Len() int {
	return len(srs)
}

func (srs SecurityRuleSet) Swap(i, j int) {
	srs[i], srs[j] = srs[j], srs[i]
}

func (srs SecurityRuleSet) Less(i, j int) bool {
	if srs[i].Priority > srs[j].Priority {
		return true
	} else if srs[i].Priority == srs[j].Priority {
		return srs[i].String() < srs[j].String()
	}
	return false
}

func (srs SecurityRuleSet) String() string {
	sort.Sort(srs)
	rules := secrules.SecurityRuleSet{}
	for i := range srs {
		rules = append(rules, srs[i].SecurityRule)
	}
	return rules.String()
}

func (srs SecurityRuleSet) AllowList() secrules.SecurityRuleSet {
	sort.Sort(srs)
	rules := secrules.SecurityRuleSet{}
	for i := range srs {
		rules = append(rules, srs[i].SecurityRule)
	}
	return rules.AllowList()
}

func GetSecurityGroupRules(sec ICloudSecurityGroup) (SecurityRuleSet, secrules.SecurityRuleSet, secrules.SecurityRuleSet, error) {
	rules, err := sec.GetRules()
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "GetRules")
	}
	for i := range rules {
		if err := rules[i].ValidateRule(); err != nil && errors.Cause(err) != secrules.ErrInvalidPriority {
			return nil, nil, nil, errors.Wrapf(err, "ValidateRule")
		}
	}
	in, out := SplitRulesByDirection(rules)
	sort.Sort(SecurityRuleSet(rules))
	return rules, in, out, nil
}

func SplitRulesByDirection(rules []SecurityRule) (secrules.SecurityRuleSet, secrules.SecurityRuleSet) {
	in, out := secrules.SecurityRuleSet{}, secrules.SecurityRuleSet{}
	for i := range rules {
		switch rules[i].Direction {
		case secrules.DIR_IN:
			in = append(in, rules[i].SecurityRule)
		case secrules.DIR_OUT:
			out = append(out, rules[i].SecurityRule)
		}
	}

	// 优先级高到低
	sort.Sort(in)
	sort.Sort(out)
	return in, out
}
