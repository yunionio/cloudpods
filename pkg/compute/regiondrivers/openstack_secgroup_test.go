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

func TestOpenStackRuleSync(t *testing.T) {
	driver := SOpenStackRegionDriver{}
	maxPriority := driver.GetSecurityGroupRuleMaxPriority()
	minPriority := driver.GetSecurityGroupRuleMinPriority()

	defaultInRule := driver.GetDefaultSecurityGroupInRule()
	defaultOutRule := driver.GetDefaultSecurityGroupOutRule()
	order := driver.GetSecurityGroupRuleOrder()
	isOnlyAllowRules := driver.IsOnlySupportAllowRules()

	data := []TestData{
		{
			Name: "Test deny rules",
			LocalRules: secrules.SecurityRuleSet{
				secrules.SecurityRule{
					Priority: 100,
					Action:   secrules.SecurityRuleDeny,
					IPNet: &net.IPNet{
						IP:   net.IPv4zero,
						Mask: net.CIDRMask(0, 32),
					},
					Protocol:  secrules.PROTO_ANY,
					Direction: secrules.DIR_IN,
				},
				secrules.SecurityRule{
					Priority: 99,
					Action:   secrules.SecurityRuleAllow,
					IPNet: &net.IPNet{
						IP:   net.IPv4zero,
						Mask: net.CIDRMask(0, 32),
					},
					Protocol:  secrules.PROTO_ANY,
					Direction: secrules.DIR_IN,
				},
				secrules.SecurityRule{
					Priority: 100,
					Action:   secrules.SecurityRuleAllow,
					IPNet: &net.IPNet{
						IP:   net.IPv4zero,
						Mask: net.CIDRMask(0, 32),
					},
					Protocol:  secrules.PROTO_ANY,
					Direction: secrules.DIR_OUT,
				},
			},
			RemoteRules: []cloudprovider.SecurityRule{
				cloudprovider.SecurityRule{
					SecurityRule: secrules.SecurityRule{
						Priority: 1,
						Action:   secrules.SecurityRuleAllow,
						IPNet: &net.IPNet{
							IP:   net.IPv4zero,
							Mask: net.CIDRMask(0, 32),
						},
						Protocol:  secrules.PROTO_ANY,
						Direction: secrules.DIR_IN,
					},
				},
			},
			Common: []cloudprovider.SecurityRule{},
			InAdds: []cloudprovider.SecurityRule{},
			OutAdds: []cloudprovider.SecurityRule{
				cloudprovider.SecurityRule{
					SecurityRule: secrules.SecurityRule{
						Priority: 0,
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
			InDels: []cloudprovider.SecurityRule{
				cloudprovider.SecurityRule{
					SecurityRule: secrules.SecurityRule{
						Priority: 1,
						Action:   secrules.SecurityRuleAllow,
						IPNet: &net.IPNet{
							IP:   net.IPv4zero,
							Mask: net.CIDRMask(0, 32),
						},
						Protocol:  secrules.PROTO_ANY,
						Direction: secrules.DIR_IN,
					},
				},
			},
			OutDels: []cloudprovider.SecurityRule{},
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
