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
	"fmt"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/aws"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type SecurityGroupListOptions struct {
		VpcId string `help:"VPC ID"`
		Name  string `help:"Secgroup name"`
		Id    string
	}
	shellutils.R(&SecurityGroupListOptions{}, "security-group-list", "List security group", func(cli *aws.SRegion, args *SecurityGroupListOptions) error {
		secgrps, e := cli.GetSecurityGroups(args.VpcId, args.Name, args.Id)
		if e != nil {
			return e
		}
		printList(secgrps, 0, 0, 0, []string{})
		return nil
	})

	type SecurityGroupCreateOptions struct {
		VPC  string `help:"vpcId"`
		NAME string `help:"group name"`
		DESC string `help:"group desc"`
		Tag  string `help:"group tag"`
	}
	shellutils.R(&SecurityGroupCreateOptions{}, "security-group-create", "Create  security group", func(cli *aws.SRegion, args *SecurityGroupCreateOptions) error {
		id, err := cli.CreateSecurityGroup(args.VPC, args.NAME, args.Tag, args.DESC)
		if err != nil {
			return err
		}
		fmt.Println(id)
		return nil
	})

	type SecurityGroupRuleDeleteOption struct {
		SECGROUP_ID string
		RULE        string
	}

	shellutils.R(&SecurityGroupRuleDeleteOption{}, "security-group-rule-delete", "Delete  security group rule", func(cli *aws.SRegion, args *SecurityGroupRuleDeleteOption) error {
		rule, err := secrules.ParseSecurityRule(args.RULE)
		if err != nil {
			return errors.Wrapf(err, "ParseSecurityRule(%s)", args.RULE)
		}
		return cli.DelSecurityGroupRule(args.SECGROUP_ID, cloudprovider.SecurityRule{SecurityRule: *rule})
	})

}
