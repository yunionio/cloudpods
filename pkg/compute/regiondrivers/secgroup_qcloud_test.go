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

func TestQcloudRuleSync(t *testing.T) {
	data := []TestData{
		{
			Name: "Test out rules",
			SrcRules: cloudprovider.SecurityRuleSet{
				ruleWithPriority("out:deny any", 1),
				ruleWithPriority("out:allow 10.160.240.0/20 any", 10),
			},
			DestRules: []cloudprovider.SecurityRule{},
			Common:    []cloudprovider.SecurityRule{},
			InAdds:    []cloudprovider.SecurityRule{},
			OutAdds: []cloudprovider.SecurityRule{
				ruleWithPriority("out:allow 10.160.240.0/20 any", 100),
			},
			InDels:  []cloudprovider.SecurityRule{},
			OutDels: []cloudprovider.SecurityRule{},
		},
		{
			Name: "Test out rules",
			SrcRules: cloudprovider.SecurityRuleSet{
				ruleWithPriority("out:deny any", 1),
				ruleWithPriority("out:allow 10.160.240.0/20 any", 10),
			},
			DestRules: []cloudprovider.SecurityRule{
				ruleWithPriority("out:allow any", 2),
			},
			Common: []cloudprovider.SecurityRule{},
			InAdds: []cloudprovider.SecurityRule{},
			OutAdds: []cloudprovider.SecurityRule{
				ruleWithPriority("out:allow 10.160.240.0/20 any", 2),
			},
			InDels: []cloudprovider.SecurityRule{},
			OutDels: []cloudprovider.SecurityRule{
				ruleWithPriority("out:allow any", 2),
			},
		},

		{
			Name: "Test peer out rules",
			SrcRules: cloudprovider.SecurityRuleSet{
				ruleWithPeerSecgroup("", "out:allow tcp", 1, "sec2"),
				ruleWithPriority("out:deny any", 1),
			},
			DestRules: []cloudprovider.SecurityRule{
				ruleWithPeerSecgroup("", "out:allow tcp", 2, "sec2"),
				ruleWithPriority("out:deny any", 1),
			},
			Common: []cloudprovider.SecurityRule{
				ruleWithPeerSecgroup("", "out:allow tcp", 2, "sec2"),
				ruleWithPriority("out:deny any", 1),
			},
			InAdds:  []cloudprovider.SecurityRule{},
			OutAdds: []cloudprovider.SecurityRule{},
			InDels:  []cloudprovider.SecurityRule{},
			OutDels: []cloudprovider.SecurityRule{},
		},
		{
			Name: "Test peer out rules priority",
			SrcRules: cloudprovider.SecurityRuleSet{
				ruleWithPeerSecgroup("", "out:allow tcp", 2, "sec2"),
				ruleWithPriority("out:deny any", 1),
			},
			DestRules: []cloudprovider.SecurityRule{
				ruleWithPeerSecgroup("", "out:allow tcp", 2, "sec2"),
				ruleWithPriority("out:deny any", 1),
			},
			Common: []cloudprovider.SecurityRule{
				ruleWithPriority("out:deny any", 1),
			},
			InAdds: []cloudprovider.SecurityRule{},
			OutAdds: []cloudprovider.SecurityRule{
				ruleWithPeerSecgroup("", "out:allow tcp", 0, "sec2"),
			},
			InDels: []cloudprovider.SecurityRule{},
			OutDels: []cloudprovider.SecurityRule{
				ruleWithPeerSecgroup("", "out:allow tcp", 2, "sec2"),
			},
		},
		{
			Name: "Test peer out rules priority 2",
			SrcRules: cloudprovider.SecurityRuleSet{
				ruleWithPeerSecgroup("", "out:deny tcp", 4, "sec2"),
				ruleWithPriority("out:allow any", 5),
			},
			DestRules: []cloudprovider.SecurityRule{
				ruleWithPeerSecgroup("", "out:deny tcp", 0, "sec2"),
				ruleWithPriority("out:allow any", 3),
			},
			Common: []cloudprovider.SecurityRule{
				ruleWithPeerSecgroup("", "out:deny tcp", 1, "sec2"),
			},
			InAdds: []cloudprovider.SecurityRule{},
			OutAdds: []cloudprovider.SecurityRule{
				ruleWithPriority("out:allow any", 0),
			},
			InDels: []cloudprovider.SecurityRule{},
			OutDels: []cloudprovider.SecurityRule{
				ruleWithPriority("out:allow any", 3),
			},
		},
		{
			Name: "Test peer out rules 1",
			SrcRules: cloudprovider.SecurityRuleSet{
				ruleWithPeerSecgroup("", "out:deny tcp 22", 60, "Sys-Default"),
				ruleWithPriority("out:allow 10.0.0.0/8 udp", 10),
			},
			DestRules: []cloudprovider.SecurityRule{
				ruleWithPriority("out:allow any", 1),
				ruleWithPeerSecgroup("", "out:deny tcp 22", 2, "Sys-Default"),
				ruleWithPriority("out:allow 10.0.0.0/8 udp", 3),
				ruleWithPriority("out:allow any", 4),
			},
			Common: []cloudprovider.SecurityRule{
				ruleWithPeerSecgroup("", "out:deny tcp 22", 2, "Sys-Default"),
				ruleWithPriority("out:allow 10.0.0.0/8 udp", 3),
				ruleWithPriority("out:allow any", 4),
			},
			InAdds:  []cloudprovider.SecurityRule{},
			OutAdds: []cloudprovider.SecurityRule{},
			InDels:  []cloudprovider.SecurityRule{},
			OutDels: []cloudprovider.SecurityRule{
				ruleWithPriority("out:allow any", 1),
			},
		},
	}

	for _, d := range data {
		d.Test(t, &SKVMRegionDriver{}, &SQcloudRegionDriver{})
	}
}
