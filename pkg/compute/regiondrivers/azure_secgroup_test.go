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
	"net"
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
				cloudprovider.SecurityRule{
					SecurityRule: secrules.SecurityRule{
						Priority: 2099,
						Action:   secrules.SecurityRuleAllow,
						IPNet: &net.IPNet{
							IP:   net.IPv4zero,
							Mask: net.CIDRMask(0, 32),
						},
						Protocol:  secrules.PROTO_ANY,
						Direction: secrules.DIR_OUT,
					},
				},
			},
			InDels:  []cloudprovider.SecurityRule{},
			OutDels: []cloudprovider.SecurityRule{},
		},
		{
			Name:       "Test commons rules",
			LocalRules: secrules.SecurityRuleSet{},
			RemoteRules: []cloudprovider.SecurityRule{
				cloudprovider.SecurityRule{
					Name: "test-name",
					SecurityRule: secrules.SecurityRule{
						Priority: 1000,
						Action:   secrules.SecurityRuleAllow,
						IPNet: &net.IPNet{
							IP:   net.IPv4zero,
							Mask: net.CIDRMask(0, 32),
						},
						Protocol:  secrules.PROTO_ANY,
						Direction: secrules.DIR_OUT,
					},
				},
			},
			Common: []cloudprovider.SecurityRule{
				cloudprovider.SecurityRule{
					Name: "test-name",
					SecurityRule: secrules.SecurityRule{
						Priority: 1000,
						Action:   secrules.SecurityRuleAllow,
						IPNet: &net.IPNet{
							IP:   net.IPv4zero,
							Mask: net.CIDRMask(0, 32),
						},
						Protocol:  secrules.PROTO_ANY,
						Direction: secrules.DIR_OUT,
					},
				},
			},
			InAdds:  []cloudprovider.SecurityRule{},
			OutAdds: []cloudprovider.SecurityRule{},
			InDels:  []cloudprovider.SecurityRule{},
			OutDels: []cloudprovider.SecurityRule{},
		},
		{
			Name: "Test diff rules",
			LocalRules: secrules.SecurityRuleSet{
				secrules.SecurityRule{
					Priority: 99,
					Action:   secrules.SecurityRuleAllow,
					IPNet: &net.IPNet{
						IP:   net.IPv4zero,
						Mask: net.CIDRMask(0, 32),
					},
					Protocol:  secrules.PROTO_TCP,
					PortStart: 100,
					PortEnd:   200,
					Direction: secrules.DIR_OUT,
				},
				secrules.SecurityRule{
					Priority: 98,
					Action:   secrules.SecurityRuleDeny,
					IPNet: &net.IPNet{
						IP:   net.IPv4zero,
						Mask: net.CIDRMask(0, 32),
					},
					Protocol:  secrules.PROTO_UDP,
					PortStart: 200,
					PortEnd:   300,
					Direction: secrules.DIR_OUT,
				},
			},
			RemoteRules: []cloudprovider.SecurityRule{
				cloudprovider.SecurityRule{
					Name: "test-tcp",
					SecurityRule: secrules.SecurityRule{
						Priority: 1000,
						Action:   secrules.SecurityRuleAllow,
						IPNet: &net.IPNet{
							IP:   net.IPv4zero,
							Mask: net.CIDRMask(0, 32),
						},
						PortStart: 100,
						PortEnd:   200,
						Protocol:  secrules.PROTO_TCP,
						Direction: secrules.DIR_OUT,
					},
				},
				cloudprovider.SecurityRule{
					Name: "test-udp",
					SecurityRule: secrules.SecurityRule{
						Priority: 1002,
						Action:   secrules.SecurityRuleDeny,
						IPNet: &net.IPNet{
							IP:   net.IPv4zero,
							Mask: net.CIDRMask(0, 32),
						},
						PortStart: 200,
						PortEnd:   300,
						Protocol:  secrules.PROTO_UDP,
						Direction: secrules.DIR_OUT,
					},
				},
			},
			Common: []cloudprovider.SecurityRule{
				cloudprovider.SecurityRule{
					Name: "test-tcp",
					SecurityRule: secrules.SecurityRule{
						Priority: 1000,
						Action:   secrules.SecurityRuleAllow,
						IPNet: &net.IPNet{
							IP:   net.IPv4zero,
							Mask: net.CIDRMask(0, 32),
						},
						PortStart: 100,
						PortEnd:   200,
						Protocol:  secrules.PROTO_TCP,
						Direction: secrules.DIR_OUT,
					},
				},
			},
			InAdds: []cloudprovider.SecurityRule{},
			OutAdds: []cloudprovider.SecurityRule{
				cloudprovider.SecurityRule{
					SecurityRule: secrules.SecurityRule{
						Priority: 1001,
						Action:   secrules.SecurityRuleDeny,
						IPNet: &net.IPNet{
							IP:   net.IPv4zero,
							Mask: net.CIDRMask(0, 32),
						},
						PortStart: 200,
						PortEnd:   300,
						Protocol:  secrules.PROTO_UDP,
						Direction: secrules.DIR_OUT,
					},
				},
				cloudprovider.SecurityRule{
					SecurityRule: secrules.SecurityRule{
						Priority: 1002,
						Action:   secrules.SecurityRuleAllow,
						IPNet: &net.IPNet{
							IP:   net.IPv4zero,
							Mask: net.CIDRMask(0, 32),
						},
						Protocol:  secrules.PROTO_ANY,
						Direction: secrules.DIR_OUT,
					},
				},
			},
			InDels: []cloudprovider.SecurityRule{},
			OutDels: []cloudprovider.SecurityRule{
				cloudprovider.SecurityRule{
					Name: "test-udp",
					SecurityRule: secrules.SecurityRule{
						Priority: 1002,
						Action:   secrules.SecurityRuleDeny,
						IPNet: &net.IPNet{
							IP:   net.IPv4zero,
							Mask: net.CIDRMask(0, 32),
						},
						PortStart: 200,
						PortEnd:   300,
						Protocol:  secrules.PROTO_UDP,
						Direction: secrules.DIR_OUT,
					},
				},
			},
		},
	}

	for _, d := range data {
		t.Logf("check %s", d.Name)
		common, inAdds, outAdds, inDels, outDels := cloudprovider.CompareRules(minPriority, maxPriority, order, d.LocalRules, d.RemoteRules, defaultInRule, defaultOutRule, isOnlyAllowRules, true)
		check(t, "common", common, d.Common)
		check(t, "inAdds", inAdds, d.InAdds)
		check(t, "outAdds", outAdds, d.OutAdds)
		check(t, "inDels", inDels, d.InDels)
		check(t, "outDels", outDels, d.OutDels)
	}
}
