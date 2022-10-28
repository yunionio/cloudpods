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

package regiondrivers

import (
	"fmt"
	"sort"
	"testing"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/util/stringutils"
)

type TestData struct {
	Name      string
	SrcRules  cloudprovider.SecurityRuleSet
	DestRules cloudprovider.SecurityRuleSet
	Common    cloudprovider.SecurityRuleSet
	InAdds    cloudprovider.SecurityRuleSet
	OutAdds   cloudprovider.SecurityRuleSet
	InDels    cloudprovider.SecurityRuleSet
	OutDels   cloudprovider.SecurityRuleSet
}

func (d TestData) Clone() TestData {
	return TestData{
		Name:      d.Name,
		SrcRules:  d.SrcRules,
		DestRules: d.DestRules,
		Common:    cloudprovider.SecurityRuleSet{},
		InAdds:    cloudprovider.SecurityRuleSet{},
		OutAdds:   cloudprovider.SecurityRuleSet{},
		InDels:    cloudprovider.SecurityRuleSet{},
		OutDels:   cloudprovider.SecurityRuleSet{},
	}
}

func (d TestData) Test(t *testing.T, srcD, destD cloudprovider.SecDriver) {
	t.Logf("check %s", d.Name)
	src, dest := cloudprovider.NewSecRuleInfo(srcD), cloudprovider.NewSecRuleInfo(destD)
	src.Rules, dest.Rules = d.SrcRules, d.DestRules
	common, inAdds, outAdds, inDels, outDels := cloudprovider.CompareRules(src, dest, true)
	check(t, "common", common, d.Common, 0, 0)
	check(t, "inAdds", inAdds, d.InAdds, dest.MinPriority, dest.MaxPriority)
	check(t, "outAdds", outAdds, d.OutAdds, dest.MinPriority, dest.MaxPriority)
	check(t, "inDels", inDels, d.InDels, 0, 0)
	check(t, "outDels", outDels, d.OutDels, 0, 0)

	t.Logf("check %s reverse", d.Name)

	rd := d.Clone()
	dest.Rules = append(dest.Rules, inAdds...)
	dest.Rules = append(dest.Rules, outAdds...)
	externalIds := []string{}
	for i := range inDels {
		if len(inDels[i].ExternalId) > 0 {
			externalIds = append(externalIds, inDels[i].ExternalId)
		} else {
			externalIds = append(externalIds, fmt.Sprintf("%s-%d", inDels[i].String(), inDels[i].Priority))
		}
	}
	for i := range outDels {
		if len(outDels[i].ExternalId) > 0 {
			externalIds = append(externalIds, outDels[i].ExternalId)
		} else {
			externalIds = append(externalIds, fmt.Sprintf("%s-%d", outDels[i].String(), outDels[i].Priority))
		}
	}
	dest.Rules = append(append(common, inAdds...), outAdds...)
	_, inAdds, outAdds, inDels, outDels = cloudprovider.CompareRules(dest, src, true)
	//check(t, "common", common, rd.Common)
	check(t, "inAdds", inAdds, rd.InAdds, dest.MinPriority, dest.MaxPriority)
	check(t, "outAdds", outAdds, rd.OutAdds, dest.MinPriority, dest.MaxPriority)
	check(t, "inDels", inDels, rd.InDels, 0, 0)
	check(t, "outDels", outDels, rd.OutDels, 0, 0)
}

var ruleWithPriority = func(ruleStr string, priority int) cloudprovider.SecurityRule {
	rule := secrules.MustParseSecurityRule(ruleStr)
	if rule == nil {
		panic(fmt.Sprintf("invalid rule str %s", ruleStr))
	}
	rule.Priority = priority
	return cloudprovider.SecurityRule{SecurityRule: *rule, Id: stringutils.UUID4()}
}

var ruleWithName = func(name, ruleStr string, priority int) cloudprovider.SecurityRule {
	return cloudprovider.SecurityRule{
		Name:         name,
		ExternalId:   name,
		SecurityRule: ruleWithPriority(ruleStr, priority).SecurityRule,
	}
}

var ruleWithPeerSecgroup = func(name, ruleStr string, priority int, peerSecgroup string) cloudprovider.SecurityRule {
	return cloudprovider.SecurityRule{
		Name:           name,
		ExternalId:     name,
		PeerSecgroupId: peerSecgroup,
		SecurityRule:   ruleWithPriority(ruleStr, priority).SecurityRule,
	}
}

var check = func(t *testing.T, name string, ret, expect []cloudprovider.SecurityRule, min, max int) {
	var show = func(info string, rules []cloudprovider.SecurityRule) {
		t.Logf("%s: %d\n", info, len(rules))
		for _, r := range rules {
			t.Logf("Name: %s id: %s external id: %s priority: %d %s\n", r.Name, r.Id, r.ExternalId, r.Priority, r.String())
		}
	}
	if len(ret) != len(expect) {
		show(fmt.Sprintf("%s rule", name), ret)
		show(fmt.Sprintf("%s expect", name), expect)
		t.Fatalf("invalid rules for %s current is %d expect %d", name, len(ret), len(expect))
	}
	sort.Sort(cloudprovider.SecurityRuleSet(ret))
	sort.Sort(cloudprovider.SecurityRuleSet(expect))
	if max < min {
		max, min = min, max
	}
	for i := range ret {
		if ret[i].Name != expect[i].Name {
			show(fmt.Sprintf("%s rule", name), ret)
			show(fmt.Sprintf("%s expect", name), expect)
			t.Fatalf("invalid index(%d) %s rule name %s expect %s", i, name, ret[i].Name, expect[i].Name)
		}
		if ret[i].Priority != expect[i].Priority {
			show(fmt.Sprintf("%s rule", name), ret)
			show(fmt.Sprintf("%s expect", name), expect)
			t.Fatalf("invalid index(%d) %s rule priority %d expect %d", i, name, ret[i].Priority, expect[i].Priority)
		}
		if max != min && (ret[i].Priority < min || ret[i].Priority > max) {
			t.Fatalf("invalid index(%d) %s rules %s priority should be in [%d, %d] current is %d", i, name, ret[i].String(), min, max, ret[i].Priority)
		}

		if ret[i].String() != expect[i].String() {
			show(fmt.Sprintf("%s rule", name), ret)
			show(fmt.Sprintf("%s expect", name), expect)
			t.Fatalf("invalid index(%d) %s rules %s expect %s", i, name, ret[i].String(), expect[i].String())
		}

	}
}
