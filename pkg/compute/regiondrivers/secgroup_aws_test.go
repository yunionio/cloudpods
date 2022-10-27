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

func TestAwsRuleSync(t *testing.T) {
	data := []TestData{
		{
			Name: "Test remove out allow rules",
			SrcRules: cloudprovider.SecurityRuleSet{
				ruleWithPriority("out:deny any", 1),
			},
			DestRules: []cloudprovider.SecurityRule{
				ruleWithName("test-allow any", "out:allow any", 1),
			},
			Common:  []cloudprovider.SecurityRule{},
			InAdds:  []cloudprovider.SecurityRule{},
			OutAdds: []cloudprovider.SecurityRule{},
			InDels:  []cloudprovider.SecurityRule{},
			OutDels: []cloudprovider.SecurityRule{
				ruleWithName("test-allow any", "out:allow any", 1),
			},
		},
		{
			Name: "Test out deny rules",
			SrcRules: cloudprovider.SecurityRuleSet{
				ruleWithPriority("out:deny any", 1),
			},
			DestRules: []cloudprovider.SecurityRule{},
			Common:    []cloudprovider.SecurityRule{},
			InAdds:    []cloudprovider.SecurityRule{},
			OutAdds:   []cloudprovider.SecurityRule{},
			InDels:    []cloudprovider.SecurityRule{},
			OutDels:   []cloudprovider.SecurityRule{},
		},
		{
			Name:      "Test out allow rules",
			SrcRules:  cloudprovider.SecurityRuleSet{},
			DestRules: []cloudprovider.SecurityRule{},
			Common:    []cloudprovider.SecurityRule{},
			InAdds:    []cloudprovider.SecurityRule{},
			OutAdds: []cloudprovider.SecurityRule{
				ruleWithName("", "out:allow any", 0),
			},
			InDels:  []cloudprovider.SecurityRule{},
			OutDels: []cloudprovider.SecurityRule{},
		},
	}

	for _, d := range data {
		d.Test(t, &SKVMRegionDriver{}, &SAwsRegionDriver{})
	}
}
