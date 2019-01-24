package shell

import (
	"yunion.io/x/onecloud/pkg/util/huawei"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type SecurityGroupListOptions struct {
		VpcId  string `help:"VPC ID"`
		Limit  int    `help:"page size"`
		Marker string `help:"page marker"`
	}
	shellutils.R(&SecurityGroupListOptions{}, "security-group-list", "List security group", func(cli *huawei.SRegion, args *SecurityGroupListOptions) error {
		secgrps, total, e := cli.GetSecurityGroups(args.VpcId, args.Limit, args.Marker)
		if e != nil {
			return e
		}
		printList(secgrps, total, 0, 0, []string{})
		return nil
	})

	type SecurityGroupShowOptions struct {
		ID string `help:"ID or name of security group"`
	}
	shellutils.R(&SecurityGroupShowOptions{}, "security-group-show", "Show details of a security group", func(cli *huawei.SRegion, args *SecurityGroupShowOptions) error {
		secgrp, err := cli.GetSecurityGroupDetails(args.ID)
		if err != nil {
			return err
		}
		printObject(secgrp)
		return nil
	})
}
