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
	"yunion.io/x/onecloud/pkg/multicloud/qcloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type CloudgroupListOptions struct {
		Keyword string
		Page    int
		Rp      int
	}
	shellutils.R(&CloudgroupListOptions{}, "cloud-group-list", "List cloudgroups", func(cli *qcloud.SRegion, args *CloudgroupListOptions) error {
		groups, _, err := cli.GetClient().ListGroups(args.Keyword, args.Page, args.Rp)
		if err != nil {
			return err
		}
		printList(groups, 0, 0, 0, nil)
		return nil
	})

	type CloudgroupIdOptions struct {
		ID int
	}

	shellutils.R(&CloudgroupIdOptions{}, "cloud-group-delete", "Delete cloudgroup", func(cli *qcloud.SRegion, args *CloudgroupIdOptions) error {
		return cli.GetClient().DeleteGroup(args.ID)
	})

	shellutils.R(&CloudgroupIdOptions{}, "cloud-group-show", "Show cloudgroup", func(cli *qcloud.SRegion, args *CloudgroupIdOptions) error {
		group, err := cli.GetClient().GetGroup(args.ID)
		if err != nil {
			return err
		}
		printObject(group)
		return nil
	})

	type CloudgroupCreateOptions struct {
		NAME   string
		Remark string
	}

	shellutils.R(&CloudgroupCreateOptions{}, "cloud-group-create", "Create cloudgroup", func(cli *qcloud.SRegion, args *CloudgroupCreateOptions) error {
		group, err := cli.GetClient().CreateGroup(args.NAME, args.Remark)
		if err != nil {
			return err
		}
		printObject(group)
		return nil
	})

	type CloudgroupUserListOptions struct {
		GROUPID int
		Page    int
		Rp      int
	}
	shellutils.R(&CloudgroupUserListOptions{}, "cloud-group-user-list", "List cloudgroup users", func(cli *qcloud.SRegion, args *CloudgroupUserListOptions) error {
		users, _, err := cli.GetClient().ListGroupUsers(args.GROUPID, args.Page, args.Rp)
		if err != nil {
			return err
		}
		printList(users, 0, 0, 0, nil)
		return nil
	})

	type GroupUserOptions struct {
		GROUP int
		USER  int
	}

	shellutils.R(&GroupUserOptions{}, "cloud-group-add-user", "Add user to cloudgroup", func(cli *qcloud.SRegion, args *GroupUserOptions) error {
		return cli.GetClient().AddUserToGroup(args.GROUP, args.USER)
	})

	shellutils.R(&GroupUserOptions{}, "cloud-group-remove-user", "Remove user from cloudgroup", func(cli *qcloud.SRegion, args *GroupUserOptions) error {
		return cli.GetClient().RemoveUserFromGroup(args.GROUP, args.USER)
	})

	type GroupPolicyOptions struct {
		GROUP  int
		POLICY string
	}

	shellutils.R(&GroupPolicyOptions{}, "cloud-group-attach-policy", "Attach policy to cloudgroup", func(cli *qcloud.SRegion, args *GroupPolicyOptions) error {
		return cli.GetClient().AttachGroupPolicy(args.GROUP, args.POLICY)
	})

	shellutils.R(&GroupPolicyOptions{}, "cloud-group-detach-policy", "Detach policy from cloudgroup", func(cli *qcloud.SRegion, args *GroupPolicyOptions) error {
		return cli.GetClient().DetachGroupPolicy(args.GROUP, args.POLICY)
	})

}
