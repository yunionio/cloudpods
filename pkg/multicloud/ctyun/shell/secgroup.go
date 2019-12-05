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

package shell

import (
	"yunion.io/x/onecloud/pkg/multicloud/ctyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type VSecurityGroupListOptions struct {
		Vpc string `help:"Vpc ID"`
	}
	shellutils.R(&VSecurityGroupListOptions{}, "secgroup-list", "List secgroups", func(cli *ctyun.SRegion, args *VSecurityGroupListOptions) error {
		secgroups, e := cli.GetSecurityGroups(args.Vpc)
		if e != nil {
			return e
		}
		printList(secgroups, 0, 0, 0, nil)
		return nil
	})

	type VSecurityGroupRuleListOptions struct {
		Group string `help:"Security Group ID"`
	}
	shellutils.R(&VSecurityGroupRuleListOptions{}, "secrule-list", "List secgroup rules", func(cli *ctyun.SRegion, args *VSecurityGroupRuleListOptions) error {
		secrules, e := cli.GetSecurityGroupRules(args.Group)
		if e != nil {
			return e
		}
		printList(secrules, 0, 0, 0, nil)
		return nil
	})

	type SecurityGroupCreateOptions struct {
		VpcId string `help:"vpc id"`
		Name  string `help:"secgroup name"`
	}
	shellutils.R(&SecurityGroupCreateOptions{}, "secgroup-create", "Create secgroup", func(cli *ctyun.SRegion, args *SecurityGroupCreateOptions) error {
		vpc, e := cli.CreateSecurityGroup(args.VpcId, args.Name)
		if e != nil {
			return e
		}
		printObject(vpc)
		return nil
	})

	type SecurityGroupRuleCreateOptions struct {
		Group     string `help:"secgroup id"`
		Direction string `help:"direction"`
		Ethertype string `help:"ethertype" choice:"IPv4|IPv6"`
		Protocol  string `help:"protocol,icmp，tcp，udp，and so on "`
		IpPrefix  string `help:"remote ip prefix"`
		PortMin   int64  `help:"portRangeMin"`
		PortMax   int64  `help:"portRangeMax"`
	}
	shellutils.R(&SecurityGroupRuleCreateOptions{}, "secrule-create", "Create secgroup rule", func(cli *ctyun.SRegion, args *SecurityGroupRuleCreateOptions) error {
		e := cli.CreateSecurityGroupRule(args.Group, args.Direction, args.Ethertype, args.Protocol, args.IpPrefix, args.PortMin, args.PortMax)
		if e != nil {
			return e
		}

		return nil
	})
}
