package shell

import (
	"yunion.io/x/onecloud/pkg/util/azure"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type DiskListOptions struct {
		Classic bool `help:"List classic disks"`
		Offset  int  `help:"List offset"`
		Limit   int  `help:"List limit"`
	}
	shellutils.R(&DiskListOptions{}, "disk-list", "List disks", func(cli *azure.SRegion, args *DiskListOptions) error {
		if args.Classic {
			disks, err := cli.GetClassicDisks()
			if err != nil {
				return err
			}
			printList(disks, len(disks), args.Offset, args.Limit, []string{})
			return nil
		}
		disks, err := cli.GetDisks()
		if err != nil {
			return err
		}
		printList(disks, len(disks), args.Offset, args.Limit, []string{})
		return nil
	})

	type DiskCreateOptions struct {
		NAME        string `help:"Disk name"`
		STORAGETYPE string `help:"Storage type" choices:"Standard_LRS|Premium_LRS|StandardSSD_LRS"`
		SizeGb      int32  `help:"Disk size"`
		Image       string `help:"Image id"`
		Desc        string `help:"description for disk"`
	}

	shellutils.R(&DiskCreateOptions{}, "disk-create", "Create disk", func(cli *azure.SRegion, args *DiskCreateOptions) error {
		disk, err := cli.CreateDisk(args.STORAGETYPE, args.NAME, args.SizeGb, args.Desc, args.Image)
		if err != nil {
			return err
		}
		printObject(disk)
		return nil
	})

	type DiskOptions struct {
		ID string `help:"Disk ID"`
	}

	shellutils.R(&DiskOptions{}, "disk-show", "Show disk", func(cli *azure.SRegion, args *DiskOptions) error {
		if disk, err := cli.GetDisk(args.ID); err != nil {
			return err
		} else {
			printObject(disk)
			return nil
		}
	})

	shellutils.R(&DiskOptions{}, "disk-delete", "Delete disks", func(cli *azure.SRegion, args *DiskOptions) error {
		return cli.DeleteDisk(args.ID)
	})

	type DiskResizeOptions struct {
		ID   string `help:"Disk ID"`
		SIZE int32  `help:"Disk SizeGb"`
	}

	shellutils.R(&DiskResizeOptions{}, "disk-resize", "Delete disks", func(cli *azure.SRegion, args *DiskResizeOptions) error {
		return cli.ResizeDisk(args.ID, args.SIZE)
	})
}
