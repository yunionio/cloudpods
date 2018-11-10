package shell

import (
	"yunion.io/x/onecloud/pkg/util/qcloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type VpcListOptions struct {
		Limit  int `help:"page size"`
		Offset int `help:"page offset"`
	}
	shellutils.R(&VpcListOptions{}, "vpc-list", "List vpcs", func(cli *qcloud.SRegion, args *VpcListOptions) error {
		vpcs, total, err := cli.GetVpcs(nil, args.Offset, args.Limit)
		if err != nil {
			return err
		}
		printList(vpcs, total, args.Offset, args.Limit, []string{})
		return nil
	})

	type VpcCreateOptions struct {
		NAME string `help:"Name for vpc"`
		CIDR string `help:"Cidr for vpc" choices:"10.0.0.0/16|172.16.0.0/12|192.168.0.0/16"`
	}
	shellutils.R(&VpcCreateOptions{}, "vpc-create", "Create vpc", func(cli *qcloud.SRegion, args *VpcCreateOptions) error {
		vpc, err := cli.CreateIVpc(args.NAME, "", args.CIDR)
		if err != nil {
			return err
		}
		printObject(vpc)
		return nil
	})

	type VpcDeleteOptions struct {
		ID string `help:"VPC ID or Name"`
	}
	shellutils.R(&VpcDeleteOptions{}, "vpc-delete", "Delete vpc", func(cli *qcloud.SRegion, args *VpcDeleteOptions) error {
		return cli.DeleteVpc(args.ID)
	})
}
