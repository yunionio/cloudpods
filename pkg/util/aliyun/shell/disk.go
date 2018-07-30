package shell

import (
	"yunion.io/yunioncloud/pkg/util/aliyun"
)

func init() {
	type DiskListOptions struct {
		Instance string `help:"Instance ID"`
		Zone     string `help:"Zone ID"`
		Category string `help:"Disk category"`
		Offset   int    `help:"List offset"`
		Limit    int    `help:"List limit"`
	}
	R(&DiskListOptions{}, "disk-list", "List disks", func(cli *aliyun.SRegion, args *DiskListOptions) error {
		disks, total, e := cli.GetDisks(args.Instance, args.Zone, args.Category, args.Offset, args.Limit)
		if e != nil {
			return e
		}
		printList(disks, total, args.Offset, args.Limit, []string{})
		return nil
	})
}
