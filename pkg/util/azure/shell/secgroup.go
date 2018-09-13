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

	type SecurityGroupShowOptions struct {
		ID string `help:"ID or name of security group"`
	}
	shellutils.R(&SecurityGroupShowOptions{}, "security-group-show", "Show details of a security group", func(cli *azure.SRegion, args *SecurityGroupShowOptions) error {
		if secgrp, err := cli.GetSecurityGroupDetails(args.ID); err != nil {
			return err
		} else {
			printObject(secgrp)
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
