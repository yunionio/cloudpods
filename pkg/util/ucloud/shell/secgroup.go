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
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/ucloud"
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

	type SecurityGroupShowOptions struct {
		ID string `help:"ID or name of security group"`
	}
	shellutils.R(&SecurityGroupShowOptions{}, "security-group-show", "Show details of a security group", func(cli *ucloud.SRegion, args *SecurityGroupShowOptions) error {
		secgrp, err := cli.GetSecurityGroupById(args.ID)
		if err != nil {
			return err
		}
		printObject(secgrp)
		return nil
	})
}
