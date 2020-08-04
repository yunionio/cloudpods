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
	type ListRolesOptions struct {
		Offset string
		Limit  int
	}
	shellutils.R(&ListRolesOptions{}, "cloud-role-list", "List ram roles", func(cli *aliyun.SRegion, args *ListRolesOptions) error {
		roles, err := cli.GetClient().ListRoles(args.Offset, args.Limit)
		if err != nil {
			return err
		}
		printList(roles.Roles.Role, 0, 0, 0, []string{})
		return nil
	})

	type GetRoleOptions struct {
		ROLENAME string
	}
	shellutils.R(&GetRoleOptions{}, "cloud-role-show", "Show ram role", func(cli *aliyun.SRegion, args *GetRoleOptions) error {
		role, err := cli.GetClient().GetRole(args.ROLENAME)
		if err != nil {
			return err
		}
		printObject(role)
		return nil
	})

	type RolePolicyOptions struct {
		ROLENAME   string
		POLICYNAME string
		POLICYTYPE string `choices:"Custom|System"`
	}

	shellutils.R(&RolePolicyOptions{}, "cloud-role-attach-policy", "Attach policy for role", func(cli *aliyun.SRegion, args *RolePolicyOptions) error {
		return cli.GetClient().AttachPolicy2Role(args.POLICYTYPE, args.POLICYNAME, args.ROLENAME)
	})

	shellutils.R(&RolePolicyOptions{}, "cloud-role-detach-policy", "Detach policy from role", func(cli *aliyun.SRegion, args *RolePolicyOptions) error {
		return cli.GetClient().DetachPolicyFromRole(args.POLICYTYPE, args.POLICYNAME, args.ROLENAME)
	})

	type RolePolicyListOptions struct {
		ROLE string
	}

	shellutils.R(&RolePolicyListOptions{}, "cloud-role-policy-list", "List cloud role policies", func(cli *aliyun.SRegion, args *RolePolicyListOptions) error {
		policies, err := cli.GetClient().ListPoliciesForRole(args.ROLE)
		if err != nil {
			return err
		}
		printList(policies, 0, 0, 0, nil)
		return nil
	})

	type DeleteRoleOptions struct {
		NAME string
	}
	shellutils.R(&DeleteRoleOptions{}, "cloud-role-delete", "Delete role", func(cli *aliyun.SRegion, args *DeleteRoleOptions) error {
		return cli.GetClient().DeleteRole(args.NAME)
	})

	shellutils.R(&ListRolesOptions{}, "enable-image-import", "Enable image import privilege", func(cli *aliyun.SRegion, args *ListRolesOptions) error {
		return cli.GetClient().EnableImageImport()
	})

	shellutils.R(&ListRolesOptions{}, "enable-image-export", "Enable image export privilege", func(cli *aliyun.SRegion, args *ListRolesOptions) error {
		return cli.GetClient().EnableImageExport()
	})

	type CallerShowOptions struct {
	}

	shellutils.R(&CallerShowOptions{}, "caller-show", "Show caller info", func(cli *aliyun.SRegion, args *CallerShowOptions) error {
		caller, err := cli.GetClient().GetCallerIdentity()
		if err != nil {
			return err
		}
		printObject(caller)
		return nil
	})

}
