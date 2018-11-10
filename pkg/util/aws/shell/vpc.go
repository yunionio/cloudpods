package shell

import (
	"yunion.io/x/onecloud/pkg/util/aws"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type VpcListOptions struct {
		Limit  int `help:"page size"`
		Offset int `help:"page offset"`
	}
	shellutils.R(&VpcListOptions{}, "vpc-list", "List vpcs", func(cli *aws.SRegion, args *VpcListOptions) error {
		vpcs, total, e := cli.GetVpcs(nil, args.Offset, args.Limit)
		if e != nil {
			return e
		}
		printList(vpcs, total, args.Offset, args.Limit, []string{})
		return nil
	})
}
