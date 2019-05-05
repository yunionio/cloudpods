package shell

import (
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/ucloud"
)

func init() {
	type DiskListOptions struct {
		Zone string `help:"Zone ID"`
	}
	shellutils.R(&DiskListOptions{}, "disk-list", "List disks", func(cli *ucloud.SRegion, args *DiskListOptions) error {
		disks, e := cli.GetDisks(args.Zone, "", nil)
		if e != nil {
			return e
		}
		printList(disks, 0, 0, 0, nil)
		return nil
	})

	type DiskDeleteOptions struct {
		ZONE string `help:"Zone ID"`
		ID   string `help:"Disk ID"`
	}
	shellutils.R(&DiskDeleteOptions{}, "disk-delete", "List disks", func(cli *ucloud.SRegion, args *DiskDeleteOptions) error {
		e := cli.DeleteDisk(args.ZONE, args.ID)
		if e != nil {
			return e
		}
		return nil
	})
}
