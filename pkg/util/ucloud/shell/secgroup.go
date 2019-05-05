package shell

import (
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/ucloud"
)

func init() {
	type SecurityGroupListOptions struct {
	}
	shellutils.R(&SecurityGroupListOptions{}, "security-group-list", "List security group", func(cli *ucloud.SRegion, args *SecurityGroupListOptions) error {
		secgrps, e := cli.GetSecurityGroups("", "")
		if e != nil {
			return e
		}
		printList(secgrps, 0, 0, 0, nil)
		return nil
	})

	type SecurityGroupShowOptions struct {
		ID string `help:"ID or name of security group"`
	}
	shellutils.R(&SecurityGroupShowOptions{}, "security-group-show", "Show details of a security group", func(cli *ucloud.SRegion, args *SecurityGroupShowOptions) error {
		secgrp, err := cli.GetSecurityGroupById(args.ID)
		if err != nil {
			return err
		}
		printObject(secgrp)
		return nil
	})
}
