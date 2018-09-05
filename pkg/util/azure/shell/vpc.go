package shell

import (
	"yunion.io/x/onecloud/pkg/util/azure"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type VpcListOptions struct {
		Limit  int `help:"page size"`
		Offset int `help:"page offset"`
	}
	shellutils.R(&VpcListOptions{}, "vpc-list", "List vpcs", func(cli *azure.SRegion, args *VpcListOptions) error {
		if vpcs, err := cli.GetIVpcs(); err != nil {
			return err
		} else {
			printList(vpcs, len(vpcs), args.Offset, args.Limit, []string{})
			return nil
		}
	})
}
