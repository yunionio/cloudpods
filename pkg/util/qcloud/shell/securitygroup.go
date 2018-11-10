package shell

import (
	"yunion.io/x/onecloud/pkg/util/qcloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type SecurityGroupListOptions struct {
		Limit  int `help:"page size"`
		Offset int `help:"page offset"`
	}
	shellutils.R(&SecurityGroupListOptions{}, "security-group-list", "List SecurityGroup", func(cli *qcloud.SRegion, args *SecurityGroupListOptions) error {
		secgrps, total, err := cli.GetSecurityGroups("", args.Limit, args.Offset)
		if err != nil {
			return err
		}
		printList(secgrps, total, args.Offset, args.Limit, []string{})
		return nil
	})

	type SecurityGroupOptions struct {
		ID string `help:"SecurityGroup ID"`
	}
	shellutils.R(&SecurityGroupOptions{}, "security-group-show", "Show SecurityGroup", func(cli *qcloud.SRegion, args *SecurityGroupOptions) error {
		secgroup, err := cli.GetSecurityGroupDetails(args.ID)
		if err != nil {
			return err
		}
		printObject(secgroup)
		return nil
	})

	shellutils.R(&SecurityGroupOptions{}, "security-group-delete", "Delete SecurityGroup", func(cli *qcloud.SRegion, args *SecurityGroupOptions) error {
		return cli.DeleteSecurityGroup(args.ID)
	})

	type SecurityGroupCreateOptions struct {
		NAME string `help:"SecurityGroup Name"`
		Desc string `help:"SecurityGroup Description"`
	}

	shellutils.R(&SecurityGroupCreateOptions{}, "security-group-create", "Create SecurityGroup", func(cli *qcloud.SRegion, args *SecurityGroupCreateOptions) error {
		secgrp, err := cli.CreateSecurityGroup(args.NAME, args.Desc)
		if err != nil {
			return err
		}
		printObject(secgrp)
		return nil
	})

	type AddressShowOptions struct {
		Id     string `help:"IP address ID"`
		Name   string `help:"IP address name"`
		Limit  int    `help:"page size"`
		Offset int    `help:"page offset"`
	}
	shellutils.R(&AddressShowOptions{}, "address-list", "Show address", func(cli *qcloud.SRegion, args *AddressShowOptions) error {
		address, total, err := cli.AddressList(args.Id, args.Name, args.Offset, args.Limit)
		if err != nil {
			return err
		}
		printList(address, total, args.Offset, args.Limit, []string{})
		return nil
	})
}
