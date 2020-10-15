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
	"sort"
	"testing"

	"yunion.io/x/pkg/util/secrules"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

func TestQcloudRuleSync(t *testing.T) {
	driver := SQcloudRegionDriver{}
	maxPriority := driver.GetSecurityGroupRuleMaxPriority()
	minPriority := driver.GetSecurityGroupRuleMinPriority()

	defaultInRule := driver.GetDefaultSecurityGroupInRule()
	defaultOutRule := driver.GetDefaultSecurityGroupOutRule()
	order := driver.GetSecurityGroupRuleOrder()
	isOnlyAllowRules := driver.IsOnlySupportAllowRules()

	data := []TestData{
		{
			Name: "Test out rules",
			LocalRules: secrules.SecurityRuleSet{
				localRuleWithPriority("out:allow any", 11),
				localRuleWithPriority("out:deny any", 10),
			},
			RemoteRules: []cloudprovider.SecurityRule{},
			Common:      []cloudprovider.SecurityRule{},
			InAdds:      []cloudprovider.SecurityRule{},
			OutAdds: []cloudprovider.SecurityRule{
				remoteRuleWithName("", "out:allow any", 100),
			},
			InDels:  []cloudprovider.SecurityRule{},
			OutDels: []cloudprovider.SecurityRule{},
		},
	}

	for _, d := range data {
		t.Logf("check %s", d.Name)
		common, inAdds, outAdds, inDels, outDels := cloudprovider.CompareRules(minPriority, maxPriority, order, d.LocalRules, d.RemoteRules, defaultInRule, defaultOutRule, isOnlyAllowRules, true)
		sort.Sort(cloudprovider.SecurityRuleSet(common))
		sort.Sort(cloudprovider.SecurityRuleSet(inAdds))
		sort.Sort(cloudprovider.SecurityRuleSet(outAdds))
		sort.Sort(cloudprovider.SecurityRuleSet(inDels))
		sort.Sort(cloudprovider.SecurityRuleSet(outDels))
		check(t, "common", common, d.Common)
		check(t, "inAdds", inAdds, d.InAdds)
		check(t, "outAdds", outAdds, d.OutAdds)
		check(t, "inDels", inDels, d.InDels)
		check(t, "outDels", outDels, d.OutDels)
	}
}
