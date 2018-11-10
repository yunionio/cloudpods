package shell

import (
	"yunion.io/x/onecloud/pkg/util/aws"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type DiskListOptions struct {
		Instance string `help:"Instance ID"`
		Zone     string `help:"Zone ID"`
		Category string `help:"Disk category"`
		Offset   int    `help:"List offset"`
		Limit    int    `help:"List limit"`
	}
	shellutils.R(&DiskListOptions{}, "disk-list", "List disks", func(cli *aws.SRegion, args *DiskListOptions) error {
		disks, total, e := cli.GetDisks(args.Instance, args.Zone, args.Category, nil, args.Offset, args.Limit)
		if e != nil {
			return e
		}
		printList(disks, total, args.Offset, args.Limit, []string{})
		return nil
	})

	type DiskDeleteOptions struct {
		Instance string `help:"Instance ID"`
	}
	shellutils.R(&DiskDeleteOptions{}, "disk-delete", "List disks", func(cli *aws.SRegion, args *DiskDeleteOptions) error {
		e := cli.DeleteDisk(args.Instance)
		if e != nil {
			return e
		}
		return nil
	})
}
