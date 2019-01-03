package shell

import (
	"yunion.io/x/onecloud/pkg/util/huawei"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type VpcListOptions struct {
		Limit  int    `help:"page size"`
		Marker string `help:"page marker"`
	}
	shellutils.R(&VpcListOptions{}, "vpc-list", "List vpcs", func(cli *huawei.SRegion, args *VpcListOptions) error {
		vpcs, total, e := cli.GetVpcs(args.Limit, args.Marker)
		if e != nil {
			return e
		}
		printList(vpcs, total, 0, args.Limit, []string{})
		return nil
	})
}
