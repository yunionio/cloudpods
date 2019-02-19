package shell

import (
	"yunion.io/x/onecloud/pkg/util/openstack"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type DiskListOptions struct {
		Category string `help:"Storage type for disk"`
	}
	shellutils.R(&DiskListOptions{}, "disk-list", "List disks", func(cli *openstack.SRegion, args *DiskListOptions) error {
		disks, err := cli.GetDisks(args.Category)
		if err != nil {
			return err
		}
		printList(disks, 0, 0, 0, []string{})
		return nil
	})

	type DiskOptions struct {
		ID string `help:"ID of disk"`
	}

	shellutils.R(&DiskOptions{}, "disk-show", "Show disk", func(cli *openstack.SRegion, args *DiskOptions) error {
		disk, err := cli.GetDisk(args.ID)
		if err != nil {
			return err
		}
		printObject(disk)
		return nil
	})

	shellutils.R(&DiskOptions{}, "disk-delete", "Delete disk", func(cli *openstack.SRegion, args *DiskOptions) error {
		return cli.DeleteDisk(args.ID)
	})

	type DiskCreateOptions struct {
		ZONE     string `help:"Zone name"`
		CATEGORY string `help:"Disk category"`
		NAME     string `help:"Disk Name"`
		SIZE     int    `help:"Disk Size GB"`
		Desc     string `help:"Description of disk"`
	}
	shellutils.R(&DiskCreateOptions{}, "disk-create", "Create disk", func(cli *openstack.SRegion, args *DiskCreateOptions) error {
		disk, err := cli.CreateDisk(args.ZONE, args.CATEGORY, args.NAME, args.SIZE, args.Desc)
		if err != nil {
			return err
		}
		printObject(disk)
		return nil
	})

	type DiskResetOptions struct {
		DISK     string `help:"ID of disk"`
		SNAPSHOT string `help:"ID of snapshot"`
	}

	shellutils.R(&DiskResetOptions{}, "disk-reset", "Reset disk", func(cli *openstack.SRegion, args *DiskResetOptions) error {
		return cli.ResetDisk(args.DISK, args.SNAPSHOT)
	})

	type DiskResizeOptions struct {
		DISK string `help:"ID of disk"`
		SIZE int64  `help:"Disk size GB"`
	}

	shellutils.R(&DiskResizeOptions{}, "disk-resize", "Resize disk", func(cli *openstack.SRegion, args *DiskResizeOptions) error {
		return cli.ResizeDisk(args.DISK, args.SIZE*1024)
	})

}
