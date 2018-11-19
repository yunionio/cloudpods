package shell

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/util/aliyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type SecurityGroupListOptions struct {
		VpcId            string   `help:"VPC ID"`
		SecurityGroupIds []string `help:"SecurityGroup ids"`
		Limit            int      `help:"page size"`
		Offset           int      `help:"page offset"`
	}
	shellutils.R(&SecurityGroupListOptions{}, "security-group-list", "List security group", func(cli *aliyun.SRegion, args *SecurityGroupListOptions) error {
		secgrps, total, e := cli.GetSecurityGroups(args.VpcId, args.SecurityGroupIds, args.Offset, args.Limit)
		if e != nil {
			return e
		}
		printList(secgrps, total, args.Offset, args.Limit, []string{})
		return nil
	})

	type SecurityGroupShowOptions struct {
		ID string `help:"ID or name of security group"`
	}
	shellutils.R(&SecurityGroupShowOptions{}, "security-group-show", "Show details of a security group", func(cli *aliyun.SRegion, args *SecurityGroupShowOptions) error {
		secgrp, err := cli.GetSecurityGroupDetails(args.ID)
		if err != nil {
			return err
		}
		printObject(secgrp)
		return nil
	})

	type SecurityGroupCreateOptions struct {
		NAME  string `help:"SecurityGroup name"`
		VpcId string `help:"VPC ID"`
		Desc  string `help:"SecurityGroup description"`
	}

	shellutils.R(&SecurityGroupCreateOptions{}, "security-group-create", "Create details of a security group", func(cli *aliyun.SRegion, args *SecurityGroupCreateOptions) error {
		secgroupId, err := cli.CreateSecurityGroup(args.VpcId, args.NAME, args.Desc)
		if err != nil {
			return err
		}
		fmt.Println("secgroupId: %s", secgroupId)
		return nil
	})

}
