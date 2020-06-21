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
	}
	shellutils.R(&ListRolesOptions{}, "role-list", "List ram roles", func(cli *aliyun.SRegion, args *ListRolesOptions) error {
		roles, err := cli.GetClient().ListRoles()
		if err != nil {
			return err
		}
		printList(roles, 0, 0, 0, []string{})
		return nil
	})

	type GetRoleOptions struct {
		ROLENAME string
	}
	shellutils.R(&GetRoleOptions{}, "role-show", "Show ram role", func(cli *aliyun.SRegion, args *GetRoleOptions) error {
		role, err := cli.GetClient().GetRole(args.ROLENAME)
		if err != nil {
			return err
		}
		printObject(role)
		return nil
	})

	type ListPoliciesOptions struct {
		PolicyType string
		Role       string
	}
	shellutils.R(&ListPoliciesOptions{}, "policy-list", "List ram policies", func(cli *aliyun.SRegion, args *ListPoliciesOptions) error {
		policies, err := cli.GetClient().ListPolicies(args.PolicyType, args.Role)
		if err != nil {
			return err
		}
		printList(policies, 0, 0, 0, []string{})
		return nil
	})

	type GetPolicyOptions struct {
		POLICYTYPE string
		POLICYNAME string
	}
	shellutils.R(&GetPolicyOptions{}, "policy-show", "Show ram policy", func(cli *aliyun.SRegion, args *GetPolicyOptions) error {
		policy, err := cli.GetClient().GetPolicy(args.POLICYTYPE, args.POLICYNAME)
		if err != nil {
			return err
		}
		printObject(policy)
		return nil
	})

	type DeletePolicyOptions struct {
		POLICYTYPE string
		POLICYNAME string
	}
	shellutils.R(&DeletePolicyOptions{}, "policy-delete", "Delete policy", func(cli *aliyun.SRegion, args *DeletePolicyOptions) error {
		return cli.GetClient().DeletePolicy(args.POLICYTYPE, args.POLICYNAME)
	})

	type DeleteRoleOptions struct {
		NAME string
	}
	shellutils.R(&DeleteRoleOptions{}, "role-delete", "Delete role", func(cli *aliyun.SRegion, args *DeleteRoleOptions) error {
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
