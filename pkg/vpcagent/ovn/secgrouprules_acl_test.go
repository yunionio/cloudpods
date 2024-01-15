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
// limitations under the License

package ovn

import (
	"fmt"
	"reflect"
	"testing"

	"yunion.io/x/jsonutils"
	"yunion.io/x/ovsdb/schema/ovn_nb"
	"yunion.io/x/pkg/util/secrules"

	"yunion.io/x/onecloud/pkg/compute/models"
	agentmodels "yunion.io/x/onecloud/pkg/vpcagent/models"
)

func TestRuleToACL(t *testing.T) {
	lport := "local-port120"
	cases := []struct {
		rule *agentmodels.SecurityGroupRule
		ipv6 bool
		acl  *ovn_nb.ACL
	}{
		{
			// egress deny 100.10.10.0/24
			rule: &agentmodels.SecurityGroupRule{
				SSecurityGroupRule: models.SSecurityGroupRule{
					Direction: string(secrules.SecurityRuleEgress),
					CIDR:      "100.10.10.0/24",
					Action:    string(secrules.SecurityRuleDeny),
					Protocol:  secrules.PROTO_ANY,
					Priority:  100,
				},
			},
			acl: &ovn_nb.ACL{
				Direction: aclDirFromLport,
				Action:    "drop",
				Match:     fmt.Sprintf("inport == %q && ip4 && ip4.dst == 100.10.10.0/24", lport),
				Priority:  100,
			},
		},
		{
			// egress allow any
			rule: &agentmodels.SecurityGroupRule{
				SSecurityGroupRule: models.SSecurityGroupRule{
					Direction: string(secrules.SecurityRuleEgress),
					CIDR:      "",
					Action:    string(secrules.SecurityRuleAllow),
					Protocol:  secrules.PROTO_ANY,
					Priority:  10,
				},
			},
			acl: &ovn_nb.ACL{
				Direction: aclDirFromLport,
				Action:    "allow-related",
				Match:     fmt.Sprintf("inport == %q && ip4", lport),
				Priority:  10,
			},
		},
		{
			// egress allow any
			rule: &agentmodels.SecurityGroupRule{
				SSecurityGroupRule: models.SSecurityGroupRule{
					Direction: string(secrules.SecurityRuleEgress),
					CIDR:      "",
					Action:    string(secrules.SecurityRuleAllow),
					Protocol:  secrules.PROTO_ANY,
					Priority:  10,
				},
			},
			ipv6: true,
			acl: &ovn_nb.ACL{
				Direction: aclDirFromLport,
				Action:    "allow-related",
				Match:     fmt.Sprintf("inport == %q && (ip4 || ip6)", lport),
				Priority:  10,
			},
		},
		{
			// ingress deny all
			rule: &agentmodels.SecurityGroupRule{
				SSecurityGroupRule: models.SSecurityGroupRule{
					Direction: string(secrules.SecurityRuleIngress),
					CIDR:      "",
					Action:    string(secrules.SecurityRuleDeny),
					Protocol:  secrules.PROTO_ANY,
					Priority:  100,
				},
			},
			acl: &ovn_nb.ACL{
				Direction: aclDirToLport,
				Action:    "drop",
				Match:     fmt.Sprintf("outport == %q && ip4", lport),
				Priority:  100,
			},
		},
		{
			// ingress allow ssh
			rule: &agentmodels.SecurityGroupRule{
				SSecurityGroupRule: models.SSecurityGroupRule{
					Direction: string(secrules.SecurityRuleIngress),
					CIDR:      "",
					Action:    string(secrules.SecurityRuleAllow),
					Protocol:  secrules.PROTO_TCP,
					Ports:     "22",
					Priority:  100,
				},
			},
			acl: &ovn_nb.ACL{
				Direction: aclDirToLport,
				Action:    "allow-related",
				Match:     fmt.Sprintf("outport == %q && ip4 && tcp && tcp.dst == 22", lport),
				Priority:  100,
			},
		},
		{
			// ingress allow ssh
			rule: &agentmodels.SecurityGroupRule{
				SSecurityGroupRule: models.SSecurityGroupRule{
					Direction: string(secrules.SecurityRuleIngress),
					CIDR:      "",
					Action:    string(secrules.SecurityRuleAllow),
					Protocol:  secrules.PROTO_TCP,
					Ports:     "22",
					Priority:  100,
				},
			},
			ipv6: true,
			acl: &ovn_nb.ACL{
				Direction: aclDirToLport,
				Action:    "allow-related",
				Match:     fmt.Sprintf("outport == %q && (ip4 || ip6) && tcp && tcp.dst == 22", lport),
				Priority:  100,
			},
		},
	}

	for _, c := range cases {
		got, err := ruleToAcl(lport, c.rule, c.ipv6)
		if err != nil {
			t.Errorf("ruleToACL fail %s", err)
		} else {
			if !reflect.DeepEqual(got, c.acl) {
				t.Errorf("want: %s got: %s", jsonutils.Marshal(c.acl), jsonutils.Marshal(got))
			}
		}
	}
}
