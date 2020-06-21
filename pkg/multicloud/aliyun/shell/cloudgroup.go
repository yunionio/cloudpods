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
	"yunion.io/x/onecloud/pkg/multicloud/aliyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type CloudgroupCreateOptions struct {
		NAME     string
		Comments string
	}

	shellutils.R(&CloudgroupCreateOptions{}, "cloud-group-create", "Create Cloud group", func(cli *aliyun.SRegion, args *CloudgroupCreateOptions) error {
		group, err := cli.GetClient().CreateGroup(args.NAME, args.Comments)
		if err != nil {
			return err
		}
		printObject(group)
		return nil
	})

	type CloudgroupListOptions struct {
		Marker   string
		MaxItems int
	}

	shellutils.R(&CloudgroupListOptions{}, "cloud-group-list", "List Cloud groups", func(cli *aliyun.SRegion, args *CloudgroupListOptions) error {
		groups, err := cli.GetClient().GetCloudgroups(args.Marker, args.MaxItems)
		if err != nil {
			return err
		}
		printList(groups.Groups.Group, 0, 0, 0, nil)
		return nil
	})

	type CloudgroupDeleteOptions struct {
		NAME string
	}

	shellutils.R(&CloudgroupDeleteOptions{}, "cloud-group-delete", "Delete Cloud group", func(cli *aliyun.SRegion, args *CloudgroupDeleteOptions) error {
		return cli.GetClient().DeleteGroup(args.NAME)
	})

	type GroupExtListOptions struct {
		GROUP    string
		Marker   string
		MaxItems int
	}
	shellutils.R(&GroupExtListOptions{}, "cloud-group-user-list", "List Cloud group users", func(cli *aliyun.SRegion, args *GroupExtListOptions) error {
		users, err := cli.GetClient().ListGroupUsers(args.GROUP, args.Marker, args.MaxItems)
		if err != nil {
			return err
		}
		printList(users.Users.User, 0, 0, 0, nil)
		return nil
	})

	type GroupUserOptions struct {
		GROUP string
		USER  string
	}

	shellutils.R(&GroupUserOptions{}, "cloud-group-remove-user", "Remove user from group", func(cli *aliyun.SRegion, args *GroupUserOptions) error {
		return cli.GetClient().RemoveUserFromGroup(args.GROUP, args.USER)
	})

	shellutils.R(&GroupUserOptions{}, "cloud-group-add-user", "Add user to group", func(cli *aliyun.SRegion, args *GroupUserOptions) error {
		return cli.GetClient().AddUserToGroup(args.GROUP, args.USER)
	})

	shellutils.R(&GroupExtListOptions{}, "cloud-group-policy-list", "List Cloud group policies", func(cli *aliyun.SRegion, args *GroupExtListOptions) error {
		policies, err := cli.GetClient().ListGroupPolicies(args.GROUP)
		if err != nil {
			return err
		}
		printList(policies, 0, 0, 0, nil)
		return nil
	})

	type GroupPolicyOptions struct {
		GROUP      string
		PolicyType string `default:"System" choices:"System|Custom"`
		POLICY     string
	}

	shellutils.R(&GroupPolicyOptions{}, "cloud-group-attach-policy", "Attach policy for group", func(cli *aliyun.SRegion, args *GroupPolicyOptions) error {
		return cli.GetClient().AttachGroupPolicy(args.GROUP, args.POLICY, args.PolicyType)
	})

	shellutils.R(&GroupPolicyOptions{}, "cloud-group-detach-policy", "Detach policy from group", func(cli *aliyun.SRegion, args *GroupPolicyOptions) error {
		return cli.GetClient().DetachPolicyFromGroup(args.GROUP, args.POLICY, args.PolicyType)
	})

}
