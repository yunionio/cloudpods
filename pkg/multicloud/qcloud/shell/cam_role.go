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
	"strconv"

	"yunion.io/x/onecloud/pkg/multicloud/qcloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type RoleListOptions struct {
		Offset int
		Limit  int
	}
	shellutils.R(&RoleListOptions{}, "cloud-role-list", "List roles", func(cli *qcloud.SRegion, args *RoleListOptions) error {
		roles, _, err := cli.GetClient().DescribeRoleList(args.Offset, args.Limit)
		if err != nil {
			return err
		}
		printList(roles, 0, 0, 0, nil)
		return nil
	})

	type RoleNameOptions struct {
		ROLE string
	}

	shellutils.R(&RoleNameOptions{}, "cloud-role-show", "Show role details", func(cli *qcloud.SRegion, args *RoleNameOptions) error {
		role, err := cli.GetClient().GetRole(args.ROLE)
		if err != nil {
			return err
		}
		printObject(role)
		return nil
	})

	shellutils.R(&RoleNameOptions{}, "cloud-role-delete", "Delete role", func(cli *qcloud.SRegion, args *RoleNameOptions) error {
		return cli.GetClient().DeleteRole(args.ROLE)
	})

	type RolePolicyOptions struct {
		ROLE       string
		PolicyType string `choices:"User|QCS"`
		Offset     int
		Limit      int
	}

	shellutils.R(&RolePolicyOptions{}, "cloud-role-policy-list", "List role policies", func(cli *qcloud.SRegion, args *RolePolicyOptions) error {
		policies, _, err := cli.GetClient().ListAttachedRolePolicies(args.ROLE, args.PolicyType, args.Offset, args.Limit)
		if err != nil {
			return err
		}
		printList(policies, 0, 0, 0, nil)
		return nil
	})

	type RolePolicyActionOptions struct {
		ROLE      string
		POLICY_ID int
	}

	shellutils.R(&RolePolicyActionOptions{}, "cloud-role-attach-policy", "Attach role policy", func(cli *qcloud.SRegion, args *RolePolicyActionOptions) error {
		return cli.GetClient().AttachRolePolicy(args.ROLE, strconv.Itoa(args.POLICY_ID))
	})

	shellutils.R(&RolePolicyActionOptions{}, "cloud-role-detach-policy", "Detach role policy", func(cli *qcloud.SRegion, args *RolePolicyActionOptions) error {
		return cli.GetClient().AttachRolePolicy(args.ROLE, strconv.Itoa(args.POLICY_ID))
	})

	type RoleCreateOption struct {
		NAME     string
		DOCUMENT string
		Desc     string
	}

	shellutils.R(&RoleCreateOption{}, "cloud-role-create", "Create role", func(cli *qcloud.SRegion, args *RoleCreateOption) error {
		role, err := cli.GetClient().CreateRole(args.NAME, args.DOCUMENT, args.Desc)
		if err != nil {
			return err
		}
		printObject(role)
		return nil
	})
}
