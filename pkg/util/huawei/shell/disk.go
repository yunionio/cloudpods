package shell

import (
	"yunion.io/x/onecloud/pkg/util/huawei"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type DiskListOptions struct {
		Zone   string `help:"Zone ID"`
		Offset int    `help:"List offset"`
		Limit  int    `help:"List limit"`
	}
	shellutils.R(&DiskListOptions{}, "disk-list", "List disks", func(cli *huawei.SRegion, args *DiskListOptions) error {
		disks, total, e := cli.GetDisks(args.Zone, args.Offset, args.Limit)
		if e != nil {
			return e
		}
		printList(disks, total, args.Offset, args.Limit, []string{})
		return nil
	})

	type DiskDeleteOptions struct {
		ID string `help:"Disk ID"`
	}
	shellutils.R(&DiskDeleteOptions{}, "disk-delete", "List disks", func(cli *huawei.SRegion, args *DiskDeleteOptions) error {
		e := cli.DeleteDisk(args.ID)
		if e != nil {
			return e
		}
		return nil
	})
}
