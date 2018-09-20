package shell

import (
	"yunion.io/x/onecloud/pkg/util/azure"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type SecurityGroupListOptions struct {
		Limit  int `help:"page size"`
		Offset int `help:"page offset"`
	}
	shellutils.R(&SecurityGroupListOptions{}, "security-group-list", "List security group", func(cli *azure.SRegion, args *SecurityGroupListOptions) error {
		if secgrps, err := cli.GetSecurityGroups(); err != nil {
			return err
		} else {
			printList(secgrps, len(secgrps), args.Offset, args.Limit, []string{})
			return nil
		}
	})

	type SecurityGroupOptions struct {
		ID string `help:"ID or name of security group"`
	}
	shellutils.R(&SecurityGroupOptions{}, "security-group-show", "Show details of a security group", func(cli *azure.SRegion, args *SecurityGroupOptions) error {
		if secgrp, err := cli.GetSecurityGroupDetails(args.ID); err != nil {
			return err
		} else {
			printObject(secgrp)
			return nil
		}
	})

	shellutils.R(&SecurityGroupOptions{}, "security-group-rule-list", "List security group rules", func(cli *azure.SRegion, args *SecurityGroupOptions) error {
		if secgroup, err := cli.GetSecurityGroupDetails(args.ID); err != nil {
			return err
		} else if rules, err := secgroup.GetRules(); err != nil {
			return err
		} else {
			printList(rules, len(rules), 0, 30, []string{})
			return nil
		}
	})

	type SecurityGroupCreateOptions struct {
		NAME string `help:"Security Group name"`
	}

	shellutils.R(&SecurityGroupCreateOptions{}, "security-group-create", "Create security group", func(cli *azure.SRegion, args *SecurityGroupCreateOptions) error {
		if secgrp, err := cli.CreateSecurityGroup(args.NAME); err != nil {
			return err
		} else {
			printObject(secgrp)
			return nil
		}
	})
}
