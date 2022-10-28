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
	"testing"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

func TestOpenStackRuleSync(t *testing.T) {
	data := []TestData{
		{
			Name: "Test deny rules",
			SrcRules: cloudprovider.SecurityRuleSet{
				ruleWithPriority("in:deny any", 100),
				ruleWithPriority("in:allow any", 99),
				ruleWithPriority("out:allow any", 100),
			},
			DestRules: []cloudprovider.SecurityRule{
				ruleWithName("", "in:allow any", 1),
			},
			Common: []cloudprovider.SecurityRule{},
			InAdds: []cloudprovider.SecurityRule{},
			OutAdds: []cloudprovider.SecurityRule{
				ruleWithName("", "out:allow any", 0),
			},
			InDels: []cloudprovider.SecurityRule{
				ruleWithName("", "in:allow any", 1),
			},
			OutDels: []cloudprovider.SecurityRule{},
		},
		{
			Name: "Test deny rules 2",
			SrcRules: cloudprovider.SecurityRuleSet{
				ruleWithPriority("in:deny any", 100),
				ruleWithPriority("in:allow any", 99),
				ruleWithPriority("out:allow any", 100),
			},
			DestRules: []cloudprovider.SecurityRule{
				ruleWithName("", "in:allow any", 0),
				ruleWithName("", "out:allow any", 0),
			},
			Common: []cloudprovider.SecurityRule{
				ruleWithName("", "out:allow any", 0),
			},
			InAdds:  []cloudprovider.SecurityRule{},
			OutAdds: []cloudprovider.SecurityRule{},
			InDels: []cloudprovider.SecurityRule{
				ruleWithName("", "in:allow any", 0),
			},
			OutDels: []cloudprovider.SecurityRule{},
		},
	}

	for _, d := range data {
		d.Test(t, &SKVMRegionDriver{}, &SOpenStackRegionDriver{})
	}
}
