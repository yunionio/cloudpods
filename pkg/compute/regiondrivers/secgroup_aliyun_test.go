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

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

func TestAliyunRuleSync(t *testing.T) {
	data := []TestData{
		{
			Name: "Test out rules",
			SrcRules: cloudprovider.SecurityRuleSet{
				ruleWithPriority("in:allow tcp 1212", 52),
				ruleWithPriority("in:allow tcp 22", 51),
				ruleWithPriority("in:allow tcp 3389", 50),
				ruleWithPriority("in:allow udp 1231", 49),
				ruleWithPriority("in:deny tcp 443", 48),
			},
			DestRules: []cloudprovider.SecurityRule{
				ruleWithName("", "in:deny tcp 443", 1),
				ruleWithName("", "in:allow udp 1231", 1),
				ruleWithName("", "in:allow tcp 3389", 100),
				ruleWithName("", "in:allow tcp 22", 100),
				ruleWithName("", "in:allow tcp 1212", 100),
			},
			Common: []cloudprovider.SecurityRule{
				ruleWithName("", "in:deny tcp 443", 1),
				ruleWithName("", "in:allow udp 1231", 1),
				ruleWithName("", "in:allow tcp 3389", 100),
				ruleWithName("", "in:allow tcp 22", 100),
				ruleWithName("", "in:allow tcp 1212", 100),
			},
			InAdds:  []cloudprovider.SecurityRule{},
			OutAdds: []cloudprovider.SecurityRule{},
			InDels:  []cloudprovider.SecurityRule{},
			OutDels: []cloudprovider.SecurityRule{},
		},
		{
			Name: "Test tcp rules",
			SrcRules: cloudprovider.SecurityRuleSet{
				ruleWithPriority("out:deny tcp 443", 48),
			},
			DestRules: []cloudprovider.SecurityRule{},
			Common:    []cloudprovider.SecurityRule{},
			InAdds:    []cloudprovider.SecurityRule{},
			OutAdds: []cloudprovider.SecurityRule{
				ruleWithName("", "out:deny tcp 443", 100),
			},
			InDels:  []cloudprovider.SecurityRule{},
			OutDels: []cloudprovider.SecurityRule{},
		},
	}

	for _, d := range data {
		d.Test(t, &SKVMRegionDriver{}, &SAliyunRegionDriver{})
	}
}
