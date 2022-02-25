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
	"yunion.io/x/onecloud/pkg/multicloud/bingocloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type SecGroupListOptions struct {
	}
	shellutils.R(&SecGroupListOptions{}, "secgroup-list", "secgroup instances", func(cli *bingocloud.SRegion, args *SecGroupListOptions) error {
		secgroups, err := cli.DescribeSecurityGroups()
		if err != nil {
			return err
		}
		printList(secgroups, 0, 0, 0, []string{})
		return nil
	})

	type SecGroupIdOptions struct {
		ID string
	}

	shellutils.R(&SecGroupIdOptions{}, "secgroup-show", "show secgroup", func(cli *bingocloud.SRegion, args *SecGroupIdOptions) error {

		secgroup, err := cli.DescribeSecurityGroup(args.ID)
		if err != nil {
			return err
		}
		printObject(secgroup)
		return nil
	})

}
