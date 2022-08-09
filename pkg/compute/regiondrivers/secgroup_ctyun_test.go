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

func TestCtyunRuleSync(t *testing.T) {

	data := []TestData{
		{
			Name: "Test out deny rules",
			SrcRules: cloudprovider.SecurityRuleSet{
				ruleWithPriority("out:deny tcp 200", 1),
			},
			DestRules: []cloudprovider.SecurityRule{},
			Common:    []cloudprovider.SecurityRule{},
			InAdds:    []cloudprovider.SecurityRule{},
			OutAdds: []cloudprovider.SecurityRule{
				ruleWithName("", "out:allow udp", 0),
				ruleWithName("", "out:allow tcp 201-65535", 0),
				ruleWithName("", "out:allow tcp 1-199", 0),
				ruleWithName("", "out:allow icmp", 0),
			},
			InDels:  []cloudprovider.SecurityRule{},
			OutDels: []cloudprovider.SecurityRule{},
		},
	}

	for _, d := range data {
		d.Test(t, &SKVMRegionDriver{}, &SCtyunRegionDriver{})
	}
}
