package shell

import (
	"github.com/yunionio/onecloud/pkg/util/aliyun"
)

func init() {
	type VpcListOptions struct {
		Limit  int `help:"page size"`
		Offset int `help:"page offset"`
	}
	R(&VpcListOptions{}, "vpc-list", "List vpcs", func(cli *aliyun.SRegion, args *VpcListOptions) error {
		vpcs, total, e := cli.GetVpcs(nil, args.Offset, args.Limit)
		if e != nil {
			return e
		}
		printList(vpcs, total, args.Offset, args.Limit, []string{})
		return nil
	})
}
