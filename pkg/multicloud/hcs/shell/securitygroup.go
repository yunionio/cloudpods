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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/hcs"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type SecurityGroupListOptions struct {
		VpcId string `help:"VPC ID"`
		Name  string `help:"Secgroup name"`
	}
	shellutils.R(&SecurityGroupListOptions{}, "security-group-list", "List security group", func(cli *hcs.SRegion, args *SecurityGroupListOptions) error {
		secgrps, e := cli.GetSecurityGroups(args.VpcId)
		if e != nil {
			return e
		}
		printList(secgrps, 0, 0, 0, nil)
		return nil
	})

	type SecurityGroupShowOptions struct {
		ID string `help:"ID or name of security group"`
	}
	shellutils.R(&SecurityGroupShowOptions{}, "security-group-show", "Show details of a security group", func(cli *hcs.SRegion, args *SecurityGroupShowOptions) error {
		secgrp, err := cli.GetSecurityGroupDetails(args.ID)
		if err != nil {
			return err
		}
		printObject(secgrp)
		return nil
	})

	type SecurityGroupCreateOptions struct {
		NAME string `help:"secgroup name"`
		VPC  string `help:"ID of VPC"`
		Desc string `help:"description"`
	}
	shellutils.R(&SecurityGroupCreateOptions{}, "security-group-create", "Create security group", func(cli *hcs.SRegion, args *SecurityGroupCreateOptions) error {
		result, err := cli.CreateSecurityGroup(args.VPC, args.NAME, args.Desc)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type SecurityGroupRuleIdOptions struct {
		ID string
	}

	shellutils.R(&SecurityGroupRuleIdOptions{}, "security-group-rule-delete", "Delete security group rule", func(cli *hcs.SRegion, args *SecurityGroupRuleIdOptions) error {
		return cli.DeleteSecurityGroupRule(args.ID)
	})

	type SecurityGroupRuleCreateOptions struct {
		SECGROUP_ID string
		RULE        string
	}

	shellutils.R(&SecurityGroupRuleCreateOptions{}, "security-group-rule-create", "Create security group rule", func(cli *hcs.SRegion, args *SecurityGroupRuleCreateOptions) error {
		_rule, err := secrules.ParseSecurityRule(args.RULE)
		if err != nil {
			return errors.Wrapf(err, "invalid rule %s", args.RULE)
		}
		rule := cloudprovider.SecurityRule{
			SecurityRule: *_rule,
		}
		return cli.CreateSecurityGroupRule(args.SECGROUP_ID, rule)
	})

}
