package shell

import (
	"github.com/yunionio/onecloud/pkg/util/aliyun"
)

func init() {
	type SecurityGroupListOptions struct {
		VpcId  string `help:"VPC ID"`
		Limit  int    `help:"page size"`
		Offset int    `help:"page offset"`
	}
	R(&SecurityGroupListOptions{}, "security-group-list", "List security group", func(cli *aliyun.SRegion, args *SecurityGroupListOptions) error {
		secgrps, total, e := cli.GetSecurityGroups(args.VpcId, args.Offset, args.Limit)
		if e != nil {
			return e
		}
		printList(secgrps, total, args.Offset, args.Limit, []string{})
		return nil
	})

	type SecurityGroupShowOptions struct {
		ID string `help:"ID or name of security group"`
	}
	R(&SecurityGroupShowOptions{}, "security-group-show", "Show details of a security group", func(cli *aliyun.SRegion, args *SecurityGroupShowOptions) error {
		secgrp, err := cli.GetSecurityGroupDetails(args.ID)
		if err != nil {
			return err
		}
		printObject(secgrp)
		return nil
	})
}
