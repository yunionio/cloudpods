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

func TestAzureRuleSync(t *testing.T) {
	driver := SAzureRegionDriver{}
	maxPriority := driver.GetSecurityGroupRuleMaxPriority()
	minPriority := driver.GetSecurityGroupRuleMinPriority()

	defaultInRule := driver.GetDefaultSecurityGroupInRule()
	defaultOutRule := driver.GetDefaultSecurityGroupOutRule()
	order := driver.GetSecurityGroupRuleOrder()
	isOnlyAllowRules := driver.IsOnlySupportAllowRules()

	data := []TestData{
		{
			Name:        "Test empty rules",
			LocalRules:  secrules.SecurityRuleSet{},
			RemoteRules: []cloudprovider.SecurityRule{},
			Common:      []cloudprovider.SecurityRule{},
			InAdds:      []cloudprovider.SecurityRule{},
			OutAdds: []cloudprovider.SecurityRule{
				remoteRuleWithName("", "out:allow any", 2097),
			},
			InDels:  []cloudprovider.SecurityRule{},
			OutDels: []cloudprovider.SecurityRule{},
		},
		{
			Name:       "Test remove rules",
			LocalRules: secrules.SecurityRuleSet{},
			RemoteRules: []cloudprovider.SecurityRule{
				remoteRuleWithName("test-name", "out:allow any", 1000),
			},
			Common: []cloudprovider.SecurityRule{},
			InAdds: []cloudprovider.SecurityRule{},
			OutAdds: []cloudprovider.SecurityRule{
				remoteRuleWithName("", "out:allow any", 2097),
			},
			InDels: []cloudprovider.SecurityRule{},
			OutDels: []cloudprovider.SecurityRule{
				remoteRuleWithName("test-name", "out:allow any", 1000),
			},
		},
		{
			Name: "Test diff rules",
			LocalRules: secrules.SecurityRuleSet{
				localRuleWithPriority("out:allow tcp 100-200", 99),
				localRuleWithPriority("out:allow udp 200-300", 98),
			},
			RemoteRules: []cloudprovider.SecurityRule{
				remoteRuleWithName("test-tcp", "out:allow tcp 100-200", 1000),
				remoteRuleWithName("test-udp", "out:allow udp 200-300", 1002),
			},
			Common: []cloudprovider.SecurityRule{
				remoteRuleWithName("test-tcp", "out:allow tcp 100-200", 1000),
				remoteRuleWithName("test-udp", "out:allow udp 200-300", 1002),
			},
			InAdds: []cloudprovider.SecurityRule{},
			OutAdds: []cloudprovider.SecurityRule{
				remoteRuleWithName("", "out:allow any", 2097),
			},
			InDels:  []cloudprovider.SecurityRule{},
			OutDels: []cloudprovider.SecurityRule{},
		},
		{
			Name: "Test add rules",
			LocalRules: secrules.SecurityRuleSet{
				localRuleWithPriority("in:allow tcp", 100),
				localRuleWithPriority("in:allow udp", 99),
				localRuleWithPriority("out:deny any", 1),
			},
			RemoteRules: []cloudprovider.SecurityRule{
				remoteRuleWithName("allow-ssh", "in:allow tcp 22", 300),
			},
			Common: []cloudprovider.SecurityRule{},
			InAdds: []cloudprovider.SecurityRule{
				remoteRuleWithName("", "in:allow tcp", 2097),
				remoteRuleWithName("", "in:allow udp", 2097),
			},
			OutAdds: []cloudprovider.SecurityRule{},
			InDels: []cloudprovider.SecurityRule{
				remoteRuleWithName("allow-ssh", "in:allow tcp 22", 300),
			},
			OutDels: []cloudprovider.SecurityRule{},
		},
		{
			Name: "Test insert rules",
			LocalRules: secrules.SecurityRuleSet{
				localRuleWithPriority("in:allow tcp", 100),
				localRuleWithPriority("in:allow udp", 99),
				localRuleWithPriority("in:allow icmp", 98),
				localRuleWithPriority("out:deny any", 1),
			},
			RemoteRules: []cloudprovider.SecurityRule{
				remoteRuleWithName("allow-tcp", "in:allow tcp", 300),
				remoteRuleWithName("allow-icmp", "in:allow icmp", 400),
			},
			Common: []cloudprovider.SecurityRule{
				remoteRuleWithName("allow-tcp", "in:allow tcp", 300),
				remoteRuleWithName("allow-icmp", "in:allow icmp", 400),
			},
			InAdds: []cloudprovider.SecurityRule{
				remoteRuleWithName("", "in:allow udp", 2097),
			},
			OutAdds: []cloudprovider.SecurityRule{},
			InDels:  []cloudprovider.SecurityRule{},
			OutDels: []cloudprovider.SecurityRule{},
		},
		{
			Name: "Test icmp rules",
			LocalRules: secrules.SecurityRuleSet{
				localRuleWithPriority("in:allow tcp 33", 10),
				localRuleWithPriority("in:allow tcp 22", 1),
				localRuleWithPriority("out:deny any", 1),
			},
			RemoteRules: []cloudprovider.SecurityRule{
				remoteRuleWithName("allow-tcp-22", "in:allow tcp 22", 300),
			},
			Common: []cloudprovider.SecurityRule{
				remoteRuleWithName("allow-tcp-22", "in:allow tcp 22", 300),
			},
			InAdds: []cloudprovider.SecurityRule{
				remoteRuleWithName("", "in:allow tcp 33", 299),
			},
			OutAdds: []cloudprovider.SecurityRule{},
			InDels:  []cloudprovider.SecurityRule{},
			OutDels: []cloudprovider.SecurityRule{},
		},
		{
			Name: "Test a rules",
			LocalRules: secrules.SecurityRuleSet{
				localRuleWithPriority("in:allow tcp 1050", 5),
				localRuleWithPriority("in:allow tcp 1011", 4),
				localRuleWithPriority("in:allow tcp 1002", 3),
				localRuleWithPriority("in:allow tcp 22", 2),
				localRuleWithPriority("in:allow udp 55", 1),
				localRuleWithPriority("out:deny any", 1),
			},
			RemoteRules: []cloudprovider.SecurityRule{
				remoteRuleWithName("in_allow_udp_55_4014", "in:allow udp 55", 4014),
				remoteRuleWithName("in_allow_tcp_22_4013", "in:allow tcp 22", 4013),
				remoteRuleWithName("in_allow_tcp_1002_4012", "in:allow tcp 1002", 4012),
				remoteRuleWithName("in_allow_tcp_1010_4011", "in:allow tcp 1010", 4011),
				remoteRuleWithName("in_allow_tcp_1050_4010", "in:allow tcp 1050", 4010),
			},
			Common: []cloudprovider.SecurityRule{
				remoteRuleWithName("in_allow_tcp_1050_4010", "in:allow tcp 1050", 4010),
				remoteRuleWithName("in_allow_tcp_1002_4012", "in:allow tcp 1002", 4012),
				remoteRuleWithName("in_allow_tcp_22_4013", "in:allow tcp 22", 4013),
				remoteRuleWithName("in_allow_udp_55_4014", "in:allow udp 55", 4014),
			},
			InAdds: []cloudprovider.SecurityRule{
				remoteRuleWithName("", "in:allow tcp 1011", 4011),
			},
			OutAdds: []cloudprovider.SecurityRule{},
			InDels: []cloudprovider.SecurityRule{
				remoteRuleWithName("in_allow_tcp_1010_4011", "in:allow tcp 1010", 4011),
			},
			OutDels: []cloudprovider.SecurityRule{},
		},
		{
			Name: "Test b rules",
			LocalRules: secrules.SecurityRuleSet{
				localRuleWithPriority("in:allow udp 1055", 20),
				localRuleWithPriority("in:allow icmp", 15),
				localRuleWithPriority("in:allow tcp 1050", 5),
				localRuleWithPriority("in:allow tcp 1012", 4),
				localRuleWithPriority("in:allow tcp 1002", 3),
				localRuleWithPriority("in:allow tcp 22", 2),
				localRuleWithPriority("in:allow udp 55", 1),
				localRuleWithPriority("out:deny any", 1),
			},
			RemoteRules: []cloudprovider.SecurityRule{
				remoteRuleWithName("in_allow_udp_55_4014", "in:allow udp 55", 4014),
				remoteRuleWithName("in_allow_tcp_22_4013", "in:allow tcp 22", 4013),
				remoteRuleWithName("in_allow_tcp_1002_4012", "in:allow tcp 1002", 4012),
				remoteRuleWithName("in_allow_tcp_1012_4011", "in:allow tcp 1012", 4011),
				remoteRuleWithName("in_allow_tcp_1050_4010", "in:allow tcp 1050", 4010),
				remoteRuleWithName("in_allow_tcp_1055_4009", "in:allow tcp 1055", 4009),
			},
			Common: []cloudprovider.SecurityRule{
				remoteRuleWithName("in_allow_tcp_1050_4010", "in:allow tcp 1050", 4010),
				remoteRuleWithName("in_allow_tcp_1012_4011", "in:allow tcp 1012", 4011),
				remoteRuleWithName("in_allow_tcp_1002_4012", "in:allow tcp 1002", 4012),
				remoteRuleWithName("in_allow_tcp_22_4013", "in:allow tcp 22", 4013),
				remoteRuleWithName("in_allow_udp_55_4014", "in:allow udp 55", 4014),
			},
			InAdds: []cloudprovider.SecurityRule{
				remoteRuleWithName("", "in:allow icmp", 2097),
				remoteRuleWithName("", "in:allow udp 1055", 4013),
			},
			OutAdds: []cloudprovider.SecurityRule{},
			InDels: []cloudprovider.SecurityRule{
				remoteRuleWithName("in_allow_tcp_1055_4009", "in:allow tcp 1055", 4009),
			},
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
