package shell

import (
	"yunion.io/x/onecloud/pkg/util/aliyun"
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
}
