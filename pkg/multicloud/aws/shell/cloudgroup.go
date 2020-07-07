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

	type CloudgroupListOptions struct {
		Marker     string
		MaxItems   int
		PathPrefix string
	}
	shellutils.R(&CloudgroupListOptions{}, "cloud-group-list", "List cloudgroups", func(cli *aws.SRegion, args *CloudgroupListOptions) error {
		groups, err := cli.GetClient().ListGroups(args.Marker, args.MaxItems, args.PathPrefix)
		if err != nil {
			return err
		}
		printList(groups.Groups, 0, 0, 0, nil)
		return nil
	})

	type CloudgroupCreateOptions struct {
		NAME string
		Path string
	}

	shellutils.R(&CloudgroupCreateOptions{}, "cloud-group-create", "Create cloudgroup", func(cli *aws.SRegion, args *CloudgroupCreateOptions) error {
		group, err := cli.GetClient().CreateGroup(args.NAME, args.Path)
		if err != nil {
			return err
		}
		printObject(group)
		return nil
	})

	type CloudgroupShowOptions struct {
		NAME     string
		Marker   string
		MaxItems int
	}

	shellutils.R(&CloudgroupShowOptions{}, "cloud-group-show", "Show cloudgroup details", func(cli *aws.SRegion, args *CloudgroupShowOptions) error {
		group, err := cli.GetClient().GetGroup(args.NAME, args.Marker, args.MaxItems)
		if err != nil {
			return err
		}
		printObject(group)
		return nil
	})

	type CloudgroupOptions struct {
		NAME string
	}

	shellutils.R(&CloudgroupOptions{}, "cloud-group-user-list", "List cloudgroup users", func(cli *aws.SRegion, args *CloudgroupOptions) error {
		users, err := cli.GetClient().ListGroupUsers(args.NAME)
		if err != nil {
			return err
		}
		printList(users, 0, 0, 0, nil)
		return nil
	})

	shellutils.R(&CloudgroupOptions{}, "cloud-group-delete", "Delete cloudgroup", func(cli *aws.SRegion, args *CloudgroupOptions) error {
		return cli.GetClient().DeleteGroup(args.NAME)
	})

	type CloudgroupPolicyListOptions struct {
		NAME     string
		Marker   string
		MaxItems int
	}

	shellutils.R(&CloudgroupPolicyListOptions{}, "cloud-group-policy-list", "List cloudgroup policies", func(cli *aws.SRegion, args *CloudgroupPolicyListOptions) error {
		policies, err := cli.GetClient().ListGroupPolicies(args.NAME, args.Marker, args.MaxItems)
		if err != nil {
			return err
		}
		printList(policies.Policies, 0, 0, 0, nil)
		return nil
	})

	shellutils.R(&CloudgroupPolicyListOptions{}, "cloud-group-attached-policy-list", "List cloudgroup policies", func(cli *aws.SRegion, args *CloudgroupPolicyListOptions) error {
		policies, err := cli.GetClient().ListAttachedGroupPolicies(args.NAME, args.Marker, args.MaxItems)
		if err != nil {
			return err
		}
		printList(policies.AttachedPolicies, 0, 0, 0, nil)
		return nil
	})

	type GroupUserOptions struct {
		GROUP string
		USER  string
	}

	shellutils.R(&GroupUserOptions{}, "cloud-group-add-user", "Add user to cloudgroup", func(cli *aws.SRegion, args *GroupUserOptions) error {
		return cli.GetClient().AddUserToGroup(args.GROUP, args.USER)
	})

	shellutils.R(&GroupUserOptions{}, "cloud-group-remove-user", "Remove user from cloudgroup", func(cli *aws.SRegion, args *GroupUserOptions) error {
		return cli.GetClient().RemoveUserFromGroup(args.GROUP, args.USER)
	})

	type GroupPolicyOptions struct {
		GROUP  string
		POLICY string
	}

	shellutils.R(&GroupPolicyOptions{}, "cloud-group-attach-policy", "Attach policy to cloudgroup", func(cli *aws.SRegion, args *GroupPolicyOptions) error {
		return cli.GetClient().AttachGroupPolicy(args.GROUP, args.POLICY)
	})

	shellutils.R(&GroupPolicyOptions{}, "cloud-group-detach-policy", "Detach policy from cloudgroup", func(cli *aws.SRegion, args *GroupPolicyOptions) error {
		return cli.GetClient().DetachGroupPolicy(args.GROUP, args.POLICY)
	})

}
