package shell

import (
	"yunion.io/x/onecloud/pkg/util/huawei"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type DiskListOptions struct {
		Zone string `help:"Zone ID"`
	}
	shellutils.R(&DiskListOptions{}, "disk-list", "List disks", func(cli *huawei.SRegion, args *DiskListOptions) error {
		disks, e := cli.GetDisks(args.Zone)
		if e != nil {
			return e
		}
		printList(disks, 0, 0, 0, nil)
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
