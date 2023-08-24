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

func TestAzureRuleSync(t *testing.T) {
	data := []TestData{
		{
			Name:      "Test empty rules",
			SrcRules:  cloudprovider.SecurityRuleSet{},
			DestRules: []cloudprovider.SecurityRule{},
			Common:    []cloudprovider.SecurityRule{},
			InAdds:    []cloudprovider.SecurityRule{},
			OutAdds: []cloudprovider.SecurityRule{
				ruleWithName("", "out:allow any", 4096),
			},
			InDels:  []cloudprovider.SecurityRule{},
			OutDels: []cloudprovider.SecurityRule{},
		},
		{
			Name: "Test diff rules",
			SrcRules: cloudprovider.SecurityRuleSet{
				ruleWithPriority("out:allow tcp 100-200", 99),
				ruleWithPriority("out:allow udp 200-300", 98),
			},
			DestRules: []cloudprovider.SecurityRule{
				ruleWithName("test-tcp", "out:allow tcp 100-200", 1000),
				ruleWithName("test-udp", "out:allow udp 200-300", 1002),
			},
			Common: []cloudprovider.SecurityRule{},
			InAdds: []cloudprovider.SecurityRule{},
			OutAdds: []cloudprovider.SecurityRule{
				ruleWithName("", "out:allow tcp 100-200", 4094),
				ruleWithName("", "out:allow udp 200-300", 4095),
				ruleWithName("", "out:allow any", 4096),
			},
			InDels: []cloudprovider.SecurityRule{},
			OutDels: []cloudprovider.SecurityRule{
				ruleWithName("test-tcp", "out:allow tcp 100-200", 1000),
				ruleWithName("test-udp", "out:allow udp 200-300", 1002),
			},
		},
		{
			Name: "Test add rules",
			SrcRules: cloudprovider.SecurityRuleSet{
				ruleWithPriority("in:allow tcp", 100),
				ruleWithPriority("in:allow udp", 99),
				ruleWithPriority("out:deny any", 1),
			},
			DestRules: []cloudprovider.SecurityRule{
				ruleWithName("allow-ssh", "in:allow tcp 22", 300),
			},
			Common: []cloudprovider.SecurityRule{},
			InAdds: []cloudprovider.SecurityRule{
				ruleWithName("", "in:allow udp", 4094),
				ruleWithName("", "in:allow tcp", 4095),
			},
			OutAdds: []cloudprovider.SecurityRule{},
			InDels: []cloudprovider.SecurityRule{
				ruleWithName("allow-ssh", "in:allow tcp 22", 300),
			},
			OutDels: []cloudprovider.SecurityRule{},
		},
		{
			Name: "Test insert rules",
			SrcRules: cloudprovider.SecurityRuleSet{
				ruleWithPriority("in:allow tcp", 100),
				ruleWithPriority("in:allow udp", 99),
				ruleWithPriority("in:allow icmp", 98),
				ruleWithPriority("out:deny any", 1),
			},
			DestRules: []cloudprovider.SecurityRule{
				ruleWithName("allow-tcp", "in:allow tcp", 300),
				ruleWithName("allow-icmp", "in:allow icmp", 400),
			},
			Common: []cloudprovider.SecurityRule{
				ruleWithName("allow-icmp", "in:allow icmp", 400),
				ruleWithName("allow-tcp", "in:allow tcp", 300),
			},
			InAdds: []cloudprovider.SecurityRule{
				ruleWithName("", "in:allow udp", 300),
			},
			OutAdds: []cloudprovider.SecurityRule{},
			InDels:  []cloudprovider.SecurityRule{},
			OutDels: []cloudprovider.SecurityRule{},
		},
		{
			Name: "Test icmp rules",
			SrcRules: cloudprovider.SecurityRuleSet{
				ruleWithPriority("in:allow tcp 33", 10),
				ruleWithPriority("in:allow tcp 22", 1),
				ruleWithPriority("out:deny any", 1),
			},
			DestRules: []cloudprovider.SecurityRule{
				ruleWithName("allow-tcp-22", "in:allow tcp 22", 300),
			},
			Common: []cloudprovider.SecurityRule{},
			InAdds: []cloudprovider.SecurityRule{
				ruleWithName("", "in:allow tcp 22,33", 4095),
			},
			OutAdds: []cloudprovider.SecurityRule{},
			InDels: []cloudprovider.SecurityRule{
				ruleWithName("allow-tcp-22", "in:allow tcp 22", 300),
			},
			OutDels: []cloudprovider.SecurityRule{},
		},
		{
			Name: "Test a rules",
			SrcRules: cloudprovider.SecurityRuleSet{
				ruleWithPriority("in:allow tcp 1050", 5),
				ruleWithPriority("in:allow tcp 1011", 4),
				ruleWithPriority("in:allow tcp 1002", 3),
				ruleWithPriority("in:allow tcp 22", 2),
				ruleWithPriority("in:allow udp 55", 1),
				ruleWithPriority("out:deny any", 1),
			},
			DestRules: []cloudprovider.SecurityRule{
				ruleWithName("in_allow_udp_55_4014", "in:allow udp 55", 4014),
				ruleWithName("in_allow_tcp_22_4013", "in:allow tcp 22", 4013),
				ruleWithName("in_allow_tcp_1002_4012", "in:allow tcp 1002", 4012),
				ruleWithName("in_allow_tcp_1010_4011", "in:allow tcp 1010", 4011),
				ruleWithName("in_allow_tcp_1050_4010", "in:allow tcp 1050", 4010),
			},
			Common: []cloudprovider.SecurityRule{
				ruleWithName("in_allow_udp_55_4014", "in:allow udp 55", 4014),
			},
			InAdds: []cloudprovider.SecurityRule{
				ruleWithName("", "in:allow tcp 22,1002,1011,1050", 4095),
			},
			OutAdds: []cloudprovider.SecurityRule{},
			InDels: []cloudprovider.SecurityRule{
				ruleWithName("in_allow_tcp_22_4013", "in:allow tcp 22", 4013),
				ruleWithName("in_allow_tcp_1002_4012", "in:allow tcp 1002", 4012),
				ruleWithName("in_allow_tcp_1010_4011", "in:allow tcp 1010", 4011),
				ruleWithName("in_allow_tcp_1050_4010", "in:allow tcp 1050", 4010),
			},
			OutDels: []cloudprovider.SecurityRule{},
		},
		{
			Name: "Test b rules",
			SrcRules: cloudprovider.SecurityRuleSet{
				ruleWithPriority("in:allow udp 1055", 20),
				ruleWithPriority("in:allow icmp", 15),
				ruleWithPriority("in:allow tcp 1050", 5),
				ruleWithPriority("in:allow tcp 1012", 4),
				ruleWithPriority("in:allow tcp 1002", 3),
				ruleWithPriority("in:allow tcp 22", 2),
				ruleWithPriority("in:allow udp 55", 1),
				ruleWithPriority("out:deny any", 1),
			},
			DestRules: []cloudprovider.SecurityRule{
				ruleWithName("in_allow_udp_55_4014", "in:allow udp 55", 4014),
				ruleWithName("in_allow_tcp_22_4013", "in:allow tcp 22", 4013),
				ruleWithName("in_allow_tcp_1002_4012", "in:allow tcp 1002", 4012),
				ruleWithName("in_allow_tcp_1012_4011", "in:allow tcp 1012", 4011),
				ruleWithName("in_allow_tcp_1050_4010", "in:allow tcp 1050", 4010),
				ruleWithName("in_allow_tcp_1055_4009", "in:allow tcp 1055", 4009),
			},
			Common: []cloudprovider.SecurityRule{},
			InAdds: []cloudprovider.SecurityRule{
				ruleWithName("", "in:allow icmp", 4095),
				ruleWithName("", "in:allow tcp 22,1002,1012,1050", 4094),
				ruleWithName("", "in:allow udp 55,1055", 4093),
			},
			OutAdds: []cloudprovider.SecurityRule{},
			InDels: []cloudprovider.SecurityRule{
				ruleWithName("in_allow_udp_55_4014", "in:allow udp 55", 4014),
				ruleWithName("in_allow_tcp_22_4013", "in:allow tcp 22", 4013),
				ruleWithName("in_allow_tcp_1002_4012", "in:allow tcp 1002", 4012),
				ruleWithName("in_allow_tcp_1012_4011", "in:allow tcp 1012", 4011),
				ruleWithName("in_allow_tcp_1050_4010", "in:allow tcp 1050", 4010),
				ruleWithName("in_allow_tcp_1055_4009", "in:allow tcp 1055", 4009),
			},
			OutDels: []cloudprovider.SecurityRule{},
		},
	}

	for _, d := range data {
		d.Test(t, &SKVMRegionDriver{}, &SAzureRegionDriver{})
	}
}

func TestAzureUniqPriorityRuleSync(t *testing.T) {
	cases := []struct {
		in  cloudprovider.SecurityRuleSet
		out cloudprovider.SecurityRuleSet
	}{
		{
			in: cloudprovider.SecurityRuleSet{
				ruleWithName("", "out:allow icmp", 99),
				ruleWithName("", "out:allow tcp 100-200", 99),
				ruleWithName("", "out:allow udp 200-300", 99),
			},
			out: cloudprovider.SecurityRuleSet{
				ruleWithName("", "out:allow icmp", 99),
				ruleWithName("", "out:allow tcp 100-200", 98),
				ruleWithName("", "out:allow udp 200-300", 97),
			},
		},
	}

	for _, c := range cases {
		rules := cloudprovider.SortUniqPriority(c.in)
		if len(c.out) != len(rules) {
			t.Fatalf("invalid output")
		}
		for i := range rules {
			if rules[i].String() != c.out[i].String() {
				t.Fatalf("rule shoud be %s not %s", c.out[i].String(), rules[i].String())
			}
			if rules[i].Priority != c.out[i].Priority {
				t.Fatalf("rule %s priority should be %d not %d", rules[i].String(), c.out[i].Priority, rules[i].Priority)
			}
		}
	}

}
