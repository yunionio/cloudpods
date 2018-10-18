package shell

import (
	"yunion.io/x/onecloud/pkg/util/azure"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type VpcListOptions struct {
		Classic bool `help:"List classic vpcs"`
		Limit   int  `help:"page size"`
		Offset  int  `help:"page offset"`
	}
	shellutils.R(&VpcListOptions{}, "vpc-list", "List vpcs", func(cli *azure.SRegion, args *VpcListOptions) error {
		vpcs, err := cli.GetIVpcs()
		if err != nil {
			return err
		}
		printList(vpcs, len(vpcs), args.Offset, args.Limit, []string{})
		return nil
	})

	type VpcOptions struct {
		ID string `help:"vpc ID"`
	}

	shellutils.R(&VpcOptions{}, "vpc-show", "Show vpc details", func(cli *azure.SRegion, args *VpcOptions) error {
		vpc, err := cli.GetVpc(args.ID)
		if err != nil {
			return err
		}
		printObject(vpc)
		return nil
	})
}
