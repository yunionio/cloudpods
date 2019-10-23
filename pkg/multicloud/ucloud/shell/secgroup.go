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

	"yunion.io/x/onecloud/pkg/multicloud/ucloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type SecurityGroupListOptions struct {
	}
	shellutils.R(&SecurityGroupListOptions{}, "security-group-list", "List security group", func(cli *ucloud.SRegion, args *SecurityGroupListOptions) error {
		secgrps, e := cli.GetSecurityGroups("", "")
		if e != nil {
			return e
		}
		printList(secgrps, 0, 0, 0, nil)
		return nil
	})

	type SecurityGroupIdOptions struct {
		ID string `help:"ID or name of security group"`
	}
	shellutils.R(&SecurityGroupIdOptions{}, "security-group-show", "Show details of a security group", func(cli *ucloud.SRegion, args *SecurityGroupIdOptions) error {
		secgrp, err := cli.GetSecurityGroupById(args.ID)
		if err != nil {
			return err
		}
		printObject(secgrp)
		return nil
	})

	shellutils.R(&SecurityGroupIdOptions{}, "security-group-delete", "Show details of a security group", func(cli *ucloud.SRegion, args *SecurityGroupIdOptions) error {
		return cli.DeleteSecurityGroup(args.ID)
	})

	type SecurityGroupCreateOptions struct {
		NAME string `help:"Name of security group"`
		Desc string `help:"Description of secgroup"`
	}

	shellutils.R(&SecurityGroupCreateOptions{}, "security-group-create", "Create security group", func(cli *ucloud.SRegion, args *SecurityGroupCreateOptions) error {
		secgrpId, err := cli.CreateDefaultSecurityGroup(args.NAME, args.Desc)
		if err != nil {
			return err
		}
		fmt.Println(secgrpId)
		return nil
	})

}
