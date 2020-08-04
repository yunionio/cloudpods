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
	"yunion.io/x/onecloud/pkg/multicloud/google"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {

	type RoleShowOptions struct {
	}
	shellutils.R(&RoleShowOptions{}, "iam-policy-show", "Show project policy", func(cli *google.SRegion, args *RoleShowOptions) error {
		policy, err := cli.GetClient().GetIamPolicy()
		if err != nil {
			return err
		}
		printObject(policy)
		return nil
	})

	type ClouduserListOptions struct {
	}
	shellutils.R(&ClouduserListOptions{}, "cloud-user-list", "List cloudusers", func(cli *google.SRegion, args *ClouduserListOptions) error {
		policy, err := cli.GetClient().GetIamPolicy()
		if err != nil {
			return err
		}
		users, err := policy.GetICloudusers()
		if err != nil {
			return err
		}
		printList(users, 0, 0, 0, nil)
		return nil
	})

	type RoleListOptions struct {
		ProjectId string
	}

	shellutils.R(&RoleListOptions{}, "cloud-role-list", "List roles", func(cli *google.SRegion, args *RoleListOptions) error {
		roles, err := cli.GetClient().GetRoles(args.ProjectId)
		if err != nil {
			return err
		}
		printList(roles, 0, 0, 0, nil)
		return nil
	})

	type RoleIdOption struct {
		ID string
	}

	shellutils.R(&RoleIdOption{}, "cloud-role-show", "Show role details", func(cli *google.SRegion, args *RoleIdOption) error {
		role, err := cli.GetClient().GetRole(args.ID)
		if err != nil {
			return err
		}
		printObject(role)
		return nil
	})

	type ClouduserOptions struct {
		USER  string
		ROLES []string
	}

	shellutils.R(&ClouduserOptions{}, "cloud-user-add", "Add user to project", func(cli *google.SRegion, args *ClouduserOptions) error {
		policy, err := cli.GetClient().GetIamPolicy()
		if err != nil {
			return err
		}
		return policy.AttachPolicy(args.USER, args.ROLES)
	})

	type ClouduserDetachRoleOptions struct {
		USER string
		ROLE string
	}

	shellutils.R(&ClouduserDetachRoleOptions{}, "cloud-user-detach-role", "Detach role for clouduser", func(cli *google.SRegion, args *ClouduserDetachRoleOptions) error {
		policy, err := cli.GetClient().GetIamPolicy()
		if err != nil {
			return err
		}
		return policy.DetachPolicy(args.USER, args.ROLE)
	})

	type ClouduserDeleteOptions struct {
		USER string
	}

	shellutils.R(&ClouduserDeleteOptions{}, "cloud-user-delete", "Delete clouduseruser from project", func(cli *google.SRegion, args *ClouduserDeleteOptions) error {
		policy, err := cli.GetClient().GetIamPolicy()
		if err != nil {
			return err
		}
		return policy.DeleteUser(args.USER)
	})

	type RoleDeleteOptions struct {
		NAME string
	}

	shellutils.R(&RoleDeleteOptions{}, "cloud-role-delete", "Delete role", func(cli *google.SRegion, args *RoleDeleteOptions) error {
		return cli.GetClient().DeleteRole(args.NAME)
	})

	type RoleCreateOptons struct {
		NAME        string
		Desc        string
		PERMISSIONS []string
	}

	shellutils.R(&RoleCreateOptons{}, "cloud-role-create", "Create role", func(cli *google.SRegion, args *RoleCreateOptons) error {
		role, err := cli.GetClient().CreateRole(args.PERMISSIONS, args.NAME, args.Desc)
		if err != nil {
			return err
		}
		printObject(role)
		return nil
	})

	type RoleUpdateOptions struct {
		ID          string
		PERMISSIONS []string
	}

	shellutils.R(&RoleUpdateOptions{}, "cloud-role-update", "Update role", func(cli *google.SRegion, args *RoleUpdateOptions) error {
		return cli.GetClient().UpdateRole(args.ID, args.PERMISSIONS)
	})

}
