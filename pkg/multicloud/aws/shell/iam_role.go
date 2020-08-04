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
	"yunion.io/x/onecloud/pkg/multicloud/aws"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type RoleListOptions struct {
		Offset     string
		Limit      int
		PathPrefix string
	}
	shellutils.R(&RoleListOptions{}, "cloud-role-list", "List roles", func(cli *aws.SRegion, args *RoleListOptions) error {
		roles, err := cli.GetClient().ListRoles(args.Offset, args.Limit, args.PathPrefix)
		if err != nil {
			return err
		}
		printList(roles.Roles, 0, 0, 0, []string{})
		return nil
	})

	type RoleNameOptions struct {
		ROLE string
	}

	shellutils.R(&RoleNameOptions{}, "cloud-role-show", "Show role", func(cli *aws.SRegion, args *RoleNameOptions) error {
		role, err := cli.GetClient().GetRole(args.ROLE)
		if err != nil {
			return err
		}
		printObject(role)
		return nil
	})

	shellutils.R(&RoleNameOptions{}, "cloud-role-delete", "Delete role", func(cli *aws.SRegion, args *RoleNameOptions) error {
		return cli.GetClient().DeleteRole(args.ROLE)
	})

}
