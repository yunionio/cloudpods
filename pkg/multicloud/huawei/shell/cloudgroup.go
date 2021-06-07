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
	"yunion.io/x/onecloud/pkg/multicloud/huawei"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type CloudgroupListOptions struct {
		DomainId string
		Name     string
	}
	shellutils.R(&CloudgroupListOptions{}, "cloud-group-list", "List cloudgroups", func(cli *huawei.SRegion, args *CloudgroupListOptions) error {
		groups, err := cli.GetClient().GetGroups(args.DomainId, args.Name)
		if err != nil {
			return err
		}
		printList(groups, 0, 0, 0, nil)
		return nil
	})

	type CloudgroupIdOptions struct {
		ID string
	}

	shellutils.R(&CloudgroupIdOptions{}, "cloud-group-delete", "Delete cloudgroup", func(cli *huawei.SRegion, args *CloudgroupIdOptions) error {
		return cli.GetClient().DeleteGroup(args.ID)
	})

	type CloudgroupCreateOptions struct {
		NAME string
		Desc string
	}

	shellutils.R(&CloudgroupCreateOptions{}, "cloud-group-create", "Create cloudgroup", func(cli *huawei.SRegion, args *CloudgroupCreateOptions) error {
		group, err := cli.GetClient().CreateGroup(args.NAME, args.Desc)
		if err != nil {
			return err
		}
		printObject(group)
		return nil
	})

	type GroupRoleListOptions struct {
		GROUP_ID string
	}

	shellutils.R(&GroupRoleListOptions{}, "cloud-group-policy-list", "List role", func(cli *huawei.SRegion, args *GroupRoleListOptions) error {
		roles, err := cli.GetClient().GetGroupRoles(args.GROUP_ID)
		if err != nil {
			return err
		}
		printList(roles, 0, 0, 0, nil)
		return nil
	})

	type GroupRoleOptions struct {
		GROUP_ID string
		ROLE_ID  string
	}

	shellutils.R(&GroupRoleOptions{}, "cloud-group-detach-policy", "Detach group role", func(cli *huawei.SRegion, args *GroupRoleOptions) error {
		return cli.GetClient().DetachGroupRole(args.GROUP_ID, args.ROLE_ID)
	})

	shellutils.R(&GroupRoleOptions{}, "cloud-group-attach-policy", "Detach group role", func(cli *huawei.SRegion, args *GroupRoleOptions) error {
		return cli.GetClient().AttachGroupRole(args.GROUP_ID, args.ROLE_ID)
	})

	type GroupUserListOptions struct {
		GROUP_ID string
	}

	shellutils.R(&GroupUserListOptions{}, "cloud-group-user-list", "List user", func(cli *huawei.SRegion, args *GroupUserListOptions) error {
		users, err := cli.GetClient().GetGroupUsers(args.GROUP_ID)
		if err != nil {
			return err
		}
		printList(users, 0, 0, 0, nil)
		return nil
	})

}
